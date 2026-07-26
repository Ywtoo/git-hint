package registry_test

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"git-hint/engine/provider"
)

// --------------------------------------------------------------------------
// Estrutura espelhando o git.json
// --------------------------------------------------------------------------

type CommandEntry struct {
	Description         string                  `json:"description"`
	CompleteDescription string                  `json:"completeDescription"`
	NUsed               int                     `json:"nUsed"`
	SubCommand          map[string]CommandEntry `json:"subCommand"`
}

// rePlaceholder detecta <flag> (coringa dinâmico).
var rePlaceholder = regexp.MustCompile(`^<(.+)>$`)

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

func loadJSON(t *testing.T) map[string]CommandEntry {
	t.Helper()

	// O teste roda de dentro de engine/registry/, então o arquivo relativo é
	// data/git.json. Caso rode de outra pasta, tenta o caminho completo.
	paths := []string{
		"data/git.json",
		"../../engine/registry/data/git.json",
		"engine/registry/data/git.json",
	}

	var data []byte
	var err error
	for _, p := range paths {
		data, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Fatalf("não conseguiu abrir git.json: %v", err)
	}

	var root map[string]CommandEntry
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	return root
}

// collectPlaceholders percorre recursivamente e coleta todos os nomes de
// placeholder encontrados (sem os <> angulares), junto com o caminho de onde
// foram encontrados (para mensagens de erro claras).
func collectPlaceholders(commands map[string]CommandEntry, path string, out map[string][]string) {
	for name, cmd := range commands {
		currentPath := path + " > " + name
		if m := rePlaceholder.FindStringSubmatch(name); len(m) == 2 {
			flag := m[1]
			out[flag] = append(out[flag], currentPath)
		}
		if cmd.SubCommand != nil {
			collectPlaceholders(cmd.SubCommand, currentPath, out)
		}
	}
}

// --------------------------------------------------------------------------
// Testes
// --------------------------------------------------------------------------

// TestJSONParseavel garante que o arquivo é JSON válido e não está vazio.
func TestJSONParseavel(t *testing.T) {
	root := loadJSON(t)
	if len(root) == 0 {
		t.Fatal("git.json está vazio ou sem comandos de nível raiz")
	}
	t.Logf("✅  git.json carregado com %d comandos de nível raiz", len(root))
}

// TestComandosTemDescription garante que nenhum comando de nível raiz está
// sem description (seria um dado incompleto).
func TestComandosTemDescription(t *testing.T) {
	root := loadJSON(t)
	for name, cmd := range root {
		if strings.TrimSpace(cmd.Description) == "" {
			t.Errorf("comando %q não tem 'description'", name)
		}
	}
}

// TestSubComandosTemDescription garante que todos os subcomandos (exceto
// placeholders) têm description.
func TestSubComandosTemDescription(t *testing.T) {
	root := loadJSON(t)

	var check func(map[string]CommandEntry, string)
	check = func(commands map[string]CommandEntry, path string) {
		for name, cmd := range commands {
			fullPath := path + " > " + name
			isPlaceholder := rePlaceholder.MatchString(name)
			if !isPlaceholder && strings.TrimSpace(cmd.Description) == "" {
				t.Errorf("subcomando sem description: %s", fullPath)
			}
			if cmd.SubCommand != nil {
				check(cmd.SubCommand, fullPath)
			}
		}
	}
	check(root, "git")
}

// TestPlaceholdersProvider varre todos os placeholders <flag> encontrados
// no JSON e cruza com provider.SupportedFlags (fonte de verdade real).
//
//   - ✅  tem provider → expande dinamicamente em runtime
// Relatório exibido:
//   - 🟢 Providers suportados e usados no JSON
//   - ⚠️  Providers órfãos (implementados em Go, mas nunca usados no JSON)
//   - 📝 Placeholders manuais (usados no JSON, mas sem provider Go)
func TestPlaceholdersProvider(t *testing.T) {
	root := loadJSON(t)

	// Coleta todos os placeholders presentes no JSON.
	found := make(map[string][]string) // flag → caminhos onde aparece
	collectPlaceholders(root, "git", found)

	var comProvider []string
	var semProvider []string

	for flag := range found {
		if provider.SupportedFlags[flag] {
			comProvider = append(comProvider, "<"+flag+">")
		} else {
			semProvider = append(semProvider, "<"+flag+">")
		}
	}

	sort.Strings(comProvider)
	sort.Strings(semProvider)

	// Procura por providers órfãos (no Go, mas não no JSON)
	var orfaos []string
	for flag := range provider.SupportedFlags {
		if _, ok := found[flag]; !ok {
			orfaos = append(orfaos, flag)
		}
	}
	sort.Strings(orfaos)

	// Montagem do Relatório Formatado
	var sb strings.Builder
	sb.WriteString("\n=======================================================\n")
	sb.WriteString("            RELATÓRIO DE PLACEHOLDERS & PROVIDERS      \n")
	sb.WriteString("=======================================================\n\n")

	sb.WriteString(fmt.Sprintf("🟢 PROVIDERS SUPORTADOS E EM USO (%d):\n", len(comProvider)))
	for _, p := range comProvider {
		sb.WriteString(fmt.Sprintf("   - %s\n", p))
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("⚠️  PROVIDERS ÓRFÃOS (No Go, mas ausentes no git.json) (%d):\n", len(orfaos)))
	if len(orfaos) == 0 {
		sb.WriteString("   (Nenhum provider órfão encontrado!)\n")
	} else {
		for _, o := range orfaos {
			sb.WriteString(fmt.Sprintf("   - <%s>  --> Provider existe em engine/provider, mas <%s> não é usado no JSON!\n", o, o))
		}
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("📝 PLACEHOLDERS MANUAIS / SEM PROVIDER (%d):\n", len(semProvider)))
	for _, flagWithBrackets := range semProvider {
		flag := strings.Trim(flagWithBrackets, "<>")
		paths := found[flag]
		sb.WriteString(fmt.Sprintf("   • %s (%d ocorrência(s))\n", flagWithBrackets, len(paths)))
		for _, p := range paths {
			sb.WriteString(fmt.Sprintf("       └── %s\n", p))
		}
	}
	sb.WriteString("=======================================================\n")

	t.Log(sb.String())

	// Falha caso haja algum provider órfão
	for _, o := range orfaos {
		t.Errorf("❌ Provider órfão detectado: %q está em provider.SupportedFlags mas nunca aparece como <%s> no JSON", o, o)
	}
}

// TestOrdemAlfabeticaRaiz verifica se os comandos de nível raiz já estão
// em ordem alfabética no JSON (útil para auditar se sort_json.go foi rodado).
func TestOrdemAlfabeticaRaiz(t *testing.T) {
	paths := []string{
		"data/git.json",
		"../../engine/registry/data/git.json",
		"engine/registry/data/git.json",
	}
	var data []byte
	var err error
	for _, p := range paths {
		data, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Fatalf("não conseguiu abrir git.json: %v", err)
	}

	// Extrai as chaves na ordem real do arquivo (encoding/json não preserva ordem).
	jsonKeys := extractTopLevelKeysOrdered(data)
	for i := 1; i < len(jsonKeys); i++ {
		if jsonKeys[i] < jsonKeys[i-1] {
			t.Errorf("chaves raiz fora de ordem: %q vem depois de %q (rode scripts/sort_json.go)",
				jsonKeys[i], jsonKeys[i-1])
		}
	}
	if !t.Failed() {
		t.Logf("✅  %d chaves raiz em ordem alfabética", len(jsonKeys))
	}
}

// extractTopLevelKeysOrdered extrai as chaves do nível raiz de um JSON object
// na ordem em que aparecem no texto, sem usar encoding/json (que não preserva ordem).
func extractTopLevelKeysOrdered(data []byte) []string {
	var keys []string
	depth := 0
	i := 0
	for i < len(data) {
		c := data[i]
		switch c {
		case '{':
			depth++
			i++
		case '}':
			depth--
			i++
		case '"':
			j := i + 1
			for j < len(data) && data[j] != '"' {
				if data[j] == '\\' {
					j++
				}
				j++
			}
			key := string(data[i+1 : j])
			i = j + 1
			for i < len(data) && (data[i] == ' ' || data[i] == '\n' || data[i] == '\r' || data[i] == '\t') {
				i++
			}
			if i < len(data) && data[i] == ':' && depth == 1 {
				keys = append(keys, key)
			}
		default:
			i++
		}
	}
	return keys
}
