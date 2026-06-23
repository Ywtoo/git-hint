package keymap

var Selected int

func KeyHandler(key string) (string, int) {
	switch key {
	case "arrowUP":
		if Selected <= 0 {
			// Se já está no topo ou não há seleção, sobe o histórico do shell
			return "up-line-or-history", -1
		}
		Selected--
		return "", Selected

	case "arrowDOWN":
		if Selected == -1 {
			// Primeira vez que desce: seleciona a primeira opção
			Selected = 0
		} else {
			Selected++
		}
		return "", Selected

	case "TAB":
		// Placeholder para aceitar a sugestão
		return "accept-input", Selected

	default:
		return "", Selected
	}
}
