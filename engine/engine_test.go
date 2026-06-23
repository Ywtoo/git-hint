package engine

import (
	"fmt"
	"testing"
)

func TestIntegration(t *testing.T) {
	input := "git c"
	fmt.Printf("🚀 User Input Test: \"%s\"\n", input)

	sortedMatches, err := Suggestions(input)
	if err != nil {
		t.Fatalf("Erro: %s", err)
	}

	if len(sortedMatches) == 0 {
		fmt.Println("❌ Error RankSuggestions returned empty ")
	} else {
		for _, match := range sortedMatches {
			fmt.Printf("   👉 %s (Uso: %d): %s\n", match.Name, match.NUsed, match.Description)
		}
	}
}
