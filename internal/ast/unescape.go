package ast

// unescape mirrors public Unescape defaults (magicalBraces true, not windows).
func unescape(s string) string {
	return unescapePosix(s, true)
}

func unescapePosix(s string, magicalBraces bool) string {
	runes := []rune(s)
	runes = unescapePosixClasses(runes, magicalBraces)
	return unescapePosixBackslashes(runes, magicalBraces)
}

func unescapePosixClasses(runes []rune, magicalBraces bool) []rune {
	if len(runes) == 0 {
		return runes
	}
	out := make([]rune, 0, len(runes))
	i := 0
	for i < len(runes) {
		if i == 0 && len(runes) >= 3 && runes[0] == '[' && runes[2] == ']' {
			c := runes[1]
			if isUnescapeClassInner(c, magicalBraces) {
				out = append(out, c)
				i = 3
				continue
			}
		}
		if i+3 < len(runes) && runes[i] != '\\' && runes[i+1] == '[' && runes[i+3] == ']' {
			c := runes[i+2]
			if isUnescapeClassInner(c, magicalBraces) {
				out = append(out, runes[i], c)
				i += 4
				continue
			}
		}
		out = append(out, runes[i])
		i++
	}
	return out
}

func unescapePosixBackslashes(runes []rune, magicalBraces bool) string {
	var b []rune
	for i := 0; i < len(runes); i++ {
		if runes[i] == '\\' && i+1 < len(runes) {
			c := runes[i+1]
			if isUnescapeBackslashInner(c, magicalBraces) {
				b = append(b, c)
				i++
				continue
			}
		}
		b = append(b, runes[i])
	}
	return string(b)
}

func isUnescapeClassInner(c rune, magicalBraces bool) bool {
	if c == '/' || c == '\\' {
		return false
	}
	if !magicalBraces && (c == '{' || c == '}') {
		return false
	}
	return true
}

func isUnescapeBackslashInner(c rune, magicalBraces bool) bool {
	if c == '/' {
		return false
	}
	if !magicalBraces && (c == '{' || c == '}') {
		return false
	}
	return true
}
