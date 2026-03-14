package swarmscript

import "strings"

// TokenizeLine breaks a line into highlighted tokens for rendering.
func TokenizeLine(line string) []SwarmToken {
	trimmed := strings.TrimSpace(line)

	// Comment line
	if strings.HasPrefix(trimmed, "#") {
		return []SwarmToken{{Text: line, Type: TokComment, Col: 0}}
	}

	var tokens []SwarmToken
	col := 0
	words := splitKeepingPositions(line)

	for _, wp := range words {
		upper := strings.ToUpper(wp.text)

		var tokType SwarmTokenType
		switch {
		case highlightKeywords[upper]:
			tokType = TokKeyword
		case highlightConditions[strings.ToLower(wp.text)]:
			tokType = TokCondition
		case highlightActions[upper]:
			tokType = TokAction
		case wp.text == ">" || wp.text == "<" || wp.text == "==" || wp.text == "=":
			tokType = TokOperator
		case isNumeric(wp.text):
			tokType = TokNumber
		default:
			tokType = TokText
		}

		tokens = append(tokens, SwarmToken{
			Text: wp.text,
			Type: tokType,
			Col:  wp.col,
		})
	}
	_ = col

	return tokens
}

// splitKeepingPositions splits a line into words while tracking their column positions.
func splitKeepingPositions(line string) []wordPos {
	var result []wordPos
	i := 0
	for i < len(line) {
		// Skip whitespace
		if line[i] == ' ' || line[i] == '\t' {
			i++
			continue
		}
		// Read word
		start := i
		for i < len(line) && line[i] != ' ' && line[i] != '\t' {
			i++
		}
		result = append(result, wordPos{text: line[start:i], col: start})
	}
	return result
}

// isNumeric checks if a string looks like a number.
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	start := 0
	if s[0] == '-' || s[0] == '+' {
		start = 1
	}
	if start >= len(s) {
		return false
	}
	for i := start; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
