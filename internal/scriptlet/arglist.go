package scriptlet

import (
	"fmt"
	"io"
	"strings"
)

// argList represents the argument list of a scriptlet, excluding the function call expression.
type argList string

func (al argList) ConvertUboToCanonical() argList {
	args := strings.Split(string(al), ",")
	for i := range args {
		// uBo scriptlets may use both quoted and unquoted strings.
		if !isQuoted(args[i]) {
			args[i] = fmt.Sprintf(`"%s"`, strings.TrimSpace(args[i]))
		}
	}
	return argList(strings.Join(args, ","))
}

func (al argList) Normalize() (argList, error) {
	args := strings.Split(string(al), ",")
	var normalized string
	for i, arg := range args {
		arg = strings.TrimSpace(arg)
		if !isValidJSString(arg) {
			return "", fmt.Errorf("argument %q is not a valid JS string", arg)
		}
		normalized += arg
		if i < len(args)-1 {
			normalized += ","
		}
	}
	return argList(normalized), nil
}

func (al argList) GenerateInjection(w io.Writer) error {
	_, err := fmt.Fprintf(w, `try{scriptlet(%s)}catch(ex){console.error(ex);}`, al)
	return err
}

func isQuoted(s string) bool {
	if len(s) < 2 {
		return false
	}
	if s[0] != '"' || s[len(s)-1] != '"' {
		return false
	}
	return s[0] == s[len(s)-1]
}

func isValidJSString(s string) bool {
	// Must be at least 2 characters: opening & closing quotes.
	if len(s) < 2 {
		return false
	}

	openingQuote := s[0]
	if openingQuote != '"' && openingQuote != '\'' {
		return false
	}

	if s[len(s)-1] != openingQuote {
		return false
	}

	var escaped bool // Tracks whether the current character is escaped.
	for i := 1; i < len(s)-1; i++ {
		c := s[i]

		if escaped {
			// Current character is escaped by the preceding backslash.
			// We accept anything here, then reset `escaped`.
			escaped = false
		} else {
			switch c {
			case '\\':
				// If it's a backslash and not escaped yet, mark next char as escaped.
				escaped = true
			case openingQuote:
				// We found an unescaped quote matching the outer quote.
				return false
			}
		}
	}

	// Return false if the closing quote is escaped; otherwise, return true.
	return !escaped
}
