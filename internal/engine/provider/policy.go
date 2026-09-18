package provider

var noDescriptionFlags = map[string]bool{
	"msg":     true,
	"message": true,
}

var skipRankingFlags = map[string]bool{
	"commit":     true,
	"commit-ish": true,
	"tree-ish":   true,
	"head":       true,
	"msg":        true,
	"message":    true,
}

func ShouldSkipRanking(flag string) bool {
	return skipRankingFlags[flag]
}

func HasNoDescription(flag string) bool {
	return noDescriptionFlags[flag]
}

// QuoteInsensitiveFlags are placeholder groups whose provider values carry
// leading/trailing quotes (e.g. messages suggested as "fix auth"). The user
// types the bare text — `git commit -m g` must match "fix git bug" — so
// prefix filtering and completion replace the token ignoring the quotes; the
// quotes are inserted by the completion itself.
var quoteInsensitiveFlags = map[string]bool{
	"msg":     true,
	"message": true,
	"name":    true,
	"url":     true,
}

func IsQuoteInsensitive(flag string) bool {
	return quoteInsensitiveFlags[flag]
}

func FlagCheck(flag string) string {
	if len(flag) >= 2 && flag[0] == '<' && flag[len(flag)-1] == '>' {
		return flag[1 : len(flag)-1]
	}
	return ""
}
