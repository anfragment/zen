package cosmetic

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// SanitizeCSSSelector validates and sanitizes a CSS selector.
func SanitizeCSSSelector(selectorInput string) (string, error) {
	// Step 1: Decode Unicode escapes.
	selector := decodeUnicodeEscapes(selectorInput)

	// Step 2: Check for balanced quotes and brackets.
	if !hasBalancedQuotesAndBrackets(selector) {
		return "", fmt.Errorf("selector has unbalanced quotes or brackets")
	}

	// Step 3: Sanitize dangerous characters outside of strings.
	sanitizedSelector, err := sanitizeSelector(selector)
	if err != nil {
		return "", fmt.Errorf("sanitize selector: %w", err)
	}

	return sanitizedSelector, nil
}

// decodeUnicodeEscapes replaces CSS Unicode escapes with their actual characters.
func decodeUnicodeEscapes(s string) string {
	re := regexp.MustCompile(`\\([0-9A-Fa-f]{1,6})(\s)?`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		hexDigits := submatches[1]
		r, err := strconv.ParseInt(hexDigits, 16, 32)
		if err != nil {
			return match
		}
		return string(rune(r))
	})
}

// hasBalancedQuotesAndBrackets checks for balanced quotes and brackets in the selector.
func hasBalancedQuotesAndBrackets(s string) bool {
	var stack []rune
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for _, c := range s {
		if escaped {
			escaped = false
			continue
		}

		if c == '\\' {
			escaped = true
			continue
		}

		if inSingleQuote {
			if c == '\'' {
				inSingleQuote = false
			}
			continue
		}

		if inDoubleQuote {
			if c == '"' {
				inDoubleQuote = false
			}
			continue
		}

		if c == '\'' {
			inSingleQuote = true
			continue
		}

		if c == '"' {
			inDoubleQuote = true
			continue
		}

		if c == '(' || c == '[' || c == '{' {
			stack = append(stack, c)
		} else if c == ')' || c == ']' || c == '}' {
			if len(stack) == 0 {
				return false
			}
			last := stack[len(stack)-1]
			if (c == ')' && last != '(') ||
				(c == ']' && last != '[') ||
				(c == '}' && last != '{') {
				return false
			}
			stack = stack[:len(stack)-1]
		}
	}

	return !inSingleQuote && !inDoubleQuote && len(stack) == 0 && !escaped
}

// sanitizeSelector removes dangerous characters outside of strings.
func sanitizeSelector(s string) (string, error) {
	var builder strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false
	runes := []rune(s)

	for i := 0; i < len(runes); i++ {
		c := runes[i]

		if escaped {
			escaped = false
			builder.WriteRune(c)
			continue
		}

		if c == '\\' {
			escaped = true
			builder.WriteRune(c)
			continue
		}

		if inSingleQuote {
			if c == '\'' {
				inSingleQuote = false
			}
			builder.WriteRune(c)
			continue
		}

		if inDoubleQuote {
			if c == '"' {
				inDoubleQuote = false
			}
			builder.WriteRune(c)
			continue
		}

		if c == '\'' {
			inSingleQuote = true
			builder.WriteRune(c)
			continue
		}

		if c == '"' {
			inDoubleQuote = true
			builder.WriteRune(c)
			continue
		}

		if !inSingleQuote && !inDoubleQuote {
			// Check for dangerous sequences.
			if c == '/' && i+1 < len(runes) && runes[i+1] == '*' {
				return "", fmt.Errorf("found '/*' outside of quotes")
			}

			if c == '*' && i+1 < len(runes) && runes[i+1] == '/' {
				return "", fmt.Errorf("found '*/' outside of quotes")
			}

			if c == '{' || c == '}' || c == ';' || c == '@' {
				return "", fmt.Errorf("found dangerous character '%c' outside of quotes", c)
			}
		}

		builder.WriteRune(c)
	}

	return builder.String(), nil
}
