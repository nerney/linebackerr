package matcher

import (
	"regexp"
	"strings"
)

var (
	nonAlphanumericRunRegex = regexp.MustCompile(`[^[:alnum:]]+`)
	dateRegex               = regexp.MustCompile(`((?:19|20)\d{2})\s*([0-1]\d)\s*([0-3]\d)`)
)

// NormalizeForMatch lowercases the input, collapses runs of non-alphanumeric
// characters into a single space, and trims leading/trailing separators.
func NormalizeForMatch(input string) string {
	normalized := strings.ToLower(input)
	normalized = nonAlphanumericRunRegex.ReplaceAllString(normalized, " ")
	return strings.TrimSpace(normalized)
}

// TokenizeForMatch splits the normalized input into case-insensitive tokens.
func TokenizeForMatch(input string) []string {
	normalized := NormalizeForMatch(input)
	if normalized == "" {
		return nil
	}

	return strings.Fields(normalized)
}

func normalizeForMatching(input string) string {
	return NormalizeForMatch(input)
}

func tokenizeForMatching(input string) []string {
	return TokenizeForMatch(input)
}

func hasNormalizedToken(tokens []string, want string) bool {
	want = NormalizeForMatch(want)
	if want == "" {
		return false
	}

	for _, token := range tokens {
		if token == want {
			return true
		}
	}

	collapsed := strings.Join(tokens, "")
	return collapsed == want || strings.Contains(collapsed, want)
}

func canonicalPostseasonLabel(raw string) (string, bool) {
	switch NormalizeForMatch(raw) {
	case "super bowl":
		return "Super.Bowl", true
	case "divisional":
		return "Divisional", true
	case "wildcard", "wild card":
		return "Wildcard", true
	case "championship":
		return "Championship", true
	default:
		return "", false
	}
}

func extractPostseasonMatch(normalized string) (string, bool) {
	tokens := TokenizeForMatch(normalized)
	for i, token := range tokens {
		if !isSeasonYearToken(token) {
			continue
		}

		if label, ok := postseasonLabelFromTokens(tokens[i+1:]); ok {
			return token + "." + label, true
		}
		if i > 0 {
			if label, ok := postseasonLabelFromTokens(tokens[:i]); ok {
				return token + "." + label, true
			}
		}
	}

	return "", false
}

func postseasonLabelFromTokens(tokens []string) (string, bool) {
	for i := 0; i < len(tokens); i++ {
		if tokens[i] == "super" && i+1 < len(tokens) && tokens[i+1] == "bowl" {
			return "Super.Bowl", true
		}
		if label, ok := canonicalPostseasonLabel(tokens[i]); ok {
			return label, true
		}
		if tokens[i] == "wild" && i+1 < len(tokens) && tokens[i+1] == "card" {
			return "Wildcard", true
		}
	}

	return "", false
}

func isSeasonYearToken(token string) bool {
	return len(token) == 4 && strings.HasPrefix(token, "19") || len(token) == 4 && strings.HasPrefix(token, "20")
}

func extractDateMatch(normalized string) (string, bool) {
	compact := strings.ReplaceAll(normalized, " ", "")
	match := dateRegex.FindStringSubmatch(compact)
	if match == nil {
		return "", false
	}

	return match[1] + "-" + match[2] + "-" + match[3], true
}
