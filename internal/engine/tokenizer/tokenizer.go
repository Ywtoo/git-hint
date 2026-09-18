package tokenizer

import "strings"

func TokenizeBuffer(buffer string) []string {
	var tokens []string
	var current strings.Builder
	inQuotes := false

	for _, r := range buffer {
		switch {
		case r == '"':
			current.WriteRune(r)
			inQuotes = !inQuotes
		case r == ' ' && !inQuotes:
			tokens = append(tokens, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	tokens = append(tokens, current.String())

	return tokens
}
