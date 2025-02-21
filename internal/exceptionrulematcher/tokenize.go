package exceptionrulematcher

import "regexp"

var (
	reTokenSep = regexp.MustCompile(`(^https|^http|\.|-|_|:\/\/|\/|\?|=|&|:|\^)`)
)

func tokenize(s string) []string {
	tokenRanges := reTokenSep.FindAllStringIndex(s, -1)
	// assume that each separator is followed by a token
	// over-allocating is fine, since the token arrays will be short-lived
	tokens := make([]string, 0, len(tokenRanges)+1)

	// check if the first range doesn't start at the beginning of the string
	// if it doesn't, then the first token is the substring from the beginning
	// of the string to the start of the first range
	if len(tokenRanges) > 0 && tokenRanges[0][0] > 0 {
		tokens = append(tokens, s[:tokenRanges[0][0]])
	}

	var nextStartIndex int
	for i, tokenRange := range tokenRanges {
		tokens = append(tokens, s[tokenRange[0]:tokenRange[1]])

		nextStartIndex = tokenRange[1]
		if i < len(tokenRanges)-1 {
			nextEndIndex := tokenRanges[i+1][0]
			if nextStartIndex < nextEndIndex {
				tokens = append(tokens, s[nextStartIndex:nextEndIndex])
			}
		}
	}

	if nextStartIndex < len(s) {
		tokens = append(tokens, s[nextStartIndex:])
	}

	return tokens
}
