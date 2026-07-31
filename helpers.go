package minimatch

func isOnlyStars(glob string) bool {
	if glob == "" {
		return false
	}
	for i := 0; i < len(glob); i++ {
		if glob[i] != '*' {
			return false
		}
	}
	return true
}

func regexpEscape(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '-', '[', ']', '{', '}', '(', ')', '*', '+', '?', '.',
			',', '\\', '^', '$', '|', '#', ' ', '\t', '\n', '\r', '\f', '\v':
			b = append(b, '\\', c)
		default:
			b = append(b, c)
		}
	}
	return string(b)
}
