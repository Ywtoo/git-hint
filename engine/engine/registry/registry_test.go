package registry

import (
	"testing"
)

func TestResolveCommandPath(t *testing.T) {
	input := "git"

	cmd, err := ResolveCommandPath(input)

	if err != nil {
		t.Fatalf("ResolveCommandPath falhou inesperadamente: %v", err)
	}

	if cmd == "" {
		t.Fatal("ResolveCommandPath retornou vazio, mas deveria ter retornado um comando")
	}
}
