package minimatch

import "strings"

// GlobStar marks a ** path segment (TypeScript GLOBSTAR Symbol).
type globStar struct{}

// GlobStar is the singleton ** marker used in compiled pattern sets.
var GlobStar = globStar{}

// PatternPart is one compiled path segment (TypeScript ParseReturnFiltered).
//
// Exactly one of:
//   - IsGlobStar
//   - literal Str (IsRE false and Test nil) for exact string match
//   - MM + optional Test for magic match
type PatternPart struct {
	// IsGlobStar is true for **.
	IsGlobStar bool
	// Str is the literal string when matching exactly (also used for
	// windowsNoMagicRoot roots kept as strings).
	Str string
	// MM is the compiled segment pattern when magic.
	MM MMPattern
	// HasMM is true when MM is valid for matching (magic or forced RE).
	HasMM bool
	// Test is an optional fast-path predicate replacing MM.Match.
	Test func(string) bool
}

// matchSegment reports whether fileSeg matches this pattern part.
func (p PatternPart) matchSegment(fileSeg string) bool {
	if p.IsGlobStar {
		return false
	}
	if p.Test != nil {
		return p.Test(fileSeg)
	}
	if p.HasMM {
		ok, _ := p.MM.Match(fileSeg)
		return ok
	}
	return fileSeg == p.Str
}

// isStringPart reports a non-magic string segment.
func (p PatternPart) isStringPart() bool {
	return !p.IsGlobStar && !p.HasMM && p.Test == nil
}

// globMagic detects magic in a raw segment for windowsNoMagicRoot.
// TypeScript: /[?*]|[+@!]\(.*?\)|\[|\]/
func globMagic(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '?' || c == '*' || c == '[' || c == ']' {
			return true
		}
		if (c == '+' || c == '@' || c == '!') && i+1 < len(s) && s[i+1] == '(' {
			return true
		}
	}
	return false
}

// fastPathTest returns a fast matcher for common patterns, or nil.
func fastPathTest(pattern string, opts Options) func(string) bool {
	// /^\*+$/
	if isOnlyStars(pattern) {
		if opts.Dot {
			return func(f string) bool {
				return len(f) != 0 && f != "." && f != ".."
			}
		}
		return func(f string) bool {
			return len(f) != 0 && !strings.HasPrefix(f, ".")
		}
	}

	// /^\*+([^+@!?*[(]*)$/  — * then extension without magic chars
	if strings.HasPrefix(pattern, "*") {
		i := 0
		for i < len(pattern) && pattern[i] == '*' {
			i++
		}
		ext := pattern[i:]
		if extOKForStarDotExt(ext) {
			if opts.NoCase {
				extL := strings.ToLower(ext)
				if opts.Dot {
					return func(f string) bool {
						return strings.HasSuffix(strings.ToLower(f), extL)
					}
				}
				return func(f string) bool {
					return !strings.HasPrefix(f, ".") && strings.HasSuffix(strings.ToLower(f), extL)
				}
			}
			if opts.Dot {
				return func(f string) bool { return strings.HasSuffix(f, ext) }
			}
			return func(f string) bool {
				return !strings.HasPrefix(f, ".") && strings.HasSuffix(f, ext)
			}
		}
	}

	// /^\*+\.\*+$/
	if starDotStar(pattern) {
		if opts.Dot {
			return func(f string) bool {
				return f != "." && f != ".." && strings.Contains(f, ".")
			}
		}
		return func(f string) bool {
			return !strings.HasPrefix(f, ".") && strings.Contains(f, ".")
		}
	}

	// /^\.\*+$/
	if len(pattern) >= 2 && pattern[0] == '.' && isOnlyStars(pattern[1:]) {
		return func(f string) bool {
			return f != "." && f != ".." && strings.HasPrefix(f, ".")
		}
	}

	// /^\?+([^+@!?*[(]*)?$/
	if qmarks, ext, ok := parseQmarks(pattern); ok {
		return qmarksTest(qmarks, ext, opts)
	}

	return nil
}

func extOKForStarDotExt(ext string) bool {
	for i := 0; i < len(ext); i++ {
		switch ext[i] {
		case '+', '@', '!', '?', '*', '[', '(':
			return false
		}
	}
	return true
}

func starDotStar(pattern string) bool {
	i := 0
	for i < len(pattern) && pattern[i] == '*' {
		i++
	}
	if i == 0 || i >= len(pattern) || pattern[i] != '.' {
		return false
	}
	i++
	if i >= len(pattern) {
		return false
	}
	for i < len(pattern) && pattern[i] == '*' {
		i++
	}
	return i == len(pattern)
}

func parseQmarks(pattern string) (n int, ext string, ok bool) {
	i := 0
	for i < len(pattern) && pattern[i] == '?' {
		i++
	}
	if i == 0 {
		return 0, "", false
	}
	ext = pattern[i:]
	if !extOKForStarDotExt(ext) {
		return 0, "", false
	}
	return i, ext, true
}

func qmarksTest(n int, ext string, opts Options) func(string) bool {
	base := func(f string) bool {
		if opts.Dot {
			return len(f) == n && f != "." && f != ".."
		}
		return len(f) == n && !strings.HasPrefix(f, ".")
	}
	if ext == "" {
		return base
	}
	if opts.NoCase {
		extL := strings.ToLower(ext)
		return func(f string) bool {
			return base(f) && strings.HasSuffix(strings.ToLower(f), extL)
		}
	}
	return func(f string) bool {
		return base(f) && strings.HasSuffix(f, ext)
	}
}
