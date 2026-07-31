package minimatch

import "strings"

// EscapeOptions is the subset of minimatch options that affect Escape.
//
// The zero value matches TypeScript escape() defaults:
// windowsPathsNoEscape false, magicalBraces false.
type EscapeOptions struct {
	// WindowsPathsNoEscape escapes magic characters by wrapping them in
	// character classes ([*]) instead of backslashes, and does not escape \.
	// TypeScript: windowsPathsNoEscape
	WindowsPathsNoEscape bool

	// MagicalBraces also escapes { and }.
	// TypeScript: magicalBraces (default false for escape)
	MagicalBraces bool
}

// EscapeOptionsFrom extracts EscapeOptions from a full Options value.
func EscapeOptionsFrom(o Options) EscapeOptions {
	return EscapeOptions{
		WindowsPathsNoEscape: o.EffectiveWindowsPathsNoEscape(),
		MagicalBraces:        o.MagicalBraces,
	}
}

// Escape escapes all magic characters in a glob pattern so the result
// matches only the literal string.
//
// Characters escaped by default: ? * ( ) [ ] and \ (unless
// WindowsPathsNoEscape). With MagicalBraces, { and } are also escaped.
//
// + @ ! are not escaped on their own; escaping parentheses is enough to
// prevent extglob interpretation. Escaping ! as [!] is intentionally
// avoided because [!]] is a valid class meaning "not ]".
//
// In WindowsPathsNoEscape mode, magic characters are wrapped in [] because
// a character class containing only that character matches it literally,
// and \ is left alone as a path separator.
//
// Slashes are never escaped.
//
// Corresponds to TypeScript minimatch.escape / escape().
func Escape(s string, opts EscapeOptions) string {
	if opts.WindowsPathsNoEscape {
		return escapeWindows(s, opts.MagicalBraces)
	}
	return escapePosix(s, opts.MagicalBraces)
}

func escapePosix(s string, magicalBraces bool) string {
	var b strings.Builder
	b.Grow(len(s) + len(s)/4)
	for _, r := range s {
		if isEscapeMagic(r, false, magicalBraces) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func escapeWindows(s string, magicalBraces bool) string {
	var b strings.Builder
	b.Grow(len(s) + len(s)/2)
	for _, r := range s {
		if isEscapeMagic(r, true, magicalBraces) {
			b.WriteByte('[')
			b.WriteRune(r)
			b.WriteByte(']')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// isEscapeMagic reports whether r must be escaped by Escape.
// windows: \ is not magic; braces only if magicalBraces.
func isEscapeMagic(r rune, windows, magicalBraces bool) bool {
	switch r {
	case '?', '*', '(', ')', '[', ']':
		return true
	case '\\':
		return !windows
	case '{', '}':
		return magicalBraces
	default:
		return false
	}
}
