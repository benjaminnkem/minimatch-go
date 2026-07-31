package minimatch

import "strings"

// UnescapeOptions is the subset of minimatch options that affect Unescape.
//
// The zero value matches TypeScript unescape() defaults:
// windowsPathsNoEscape false, magicalBraces true (note: true, unlike Escape).
type UnescapeOptions struct {
	// WindowsPathsNoEscape removes only []-style escapes, not backslash
	// escapes, because \ is a path separator in that mode.
	// TypeScript: windowsPathsNoEscape
	WindowsPathsNoEscape bool

	// MagicalBraces controls whether brace escapes ({ }) are unescaped.
	// When nil, braces are unescaped (TypeScript default true for unescape).
	// When non-nil, the pointed-to value is used.
	// TypeScript: magicalBraces (default true for unescape)
	MagicalBraces *bool
}

// UnescapeOptionsFrom extracts UnescapeOptions from a full Options value.
//
// MagicalBraces is taken as an explicit bool from o (including false),
// matching Object.assign when the property is present on the options object.
// For TypeScript-style free-function defaults (magicalBraces undefined → true),
// use the zero value UnescapeOptions{} instead.
func UnescapeOptionsFrom(o Options) UnescapeOptions {
	mb := o.MagicalBraces
	return UnescapeOptions{
		WindowsPathsNoEscape: o.EffectiveWindowsPathsNoEscape(),
		MagicalBraces:        &mb,
	}
}

// Unescape reverses escaping produced by Escape.
//
// In WindowsPathsNoEscape mode, only character-class escapes ([x]) are
// removed; backslash sequences are left intact.
//
// Otherwise both [x] class escapes and \x backslash escapes are removed,
// with the restrictions below.
//
// Slashes are never unescaped. In WindowsPathsNoEscape mode, backslashes
// are not unescaped either.
//
// When MagicalBraces is false (explicit), escapes of { and } are not
// removed. When MagicalBraces is nil (zero-value options), braces are
// unescaped — matching TypeScript unescape()'s default of true.
//
// Corresponds to TypeScript minimatch.unescape / unescape().
func Unescape(s string, opts UnescapeOptions) string {
	magicalBraces := true
	if opts.MagicalBraces != nil {
		magicalBraces = *opts.MagicalBraces
	}
	if opts.WindowsPathsNoEscape {
		return unescapeWindows(s, magicalBraces)
	}
	return unescapePosix(s, magicalBraces)
}

// unescapeWindows implements /\[([^/\\])\]/g or /\[([^/\\{}])\]/g.
func unescapeWindows(s string, magicalBraces bool) string {
	runes := []rune(s)
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(runes); i++ {
		if i+2 < len(runes) && runes[i] == '[' && runes[i+2] == ']' {
			c := runes[i+1]
			if isUnescapeClassInner(c, magicalBraces) {
				b.WriteRune(c)
				i += 2
				continue
			}
		}
		b.WriteRune(runes[i])
	}
	return b.String()
}

// unescapePosix mirrors the two TypeScript replacements:
//  1. /((?!\\).|^)\[([^/\\])\]/g → $1$2  (or with {} excluded)
//  2. /\\([^/])/g → $1                  (or with {} excluded)
//
// Go's regexp engine lacks lookaheads, so this is implemented by hand.
func unescapePosix(s string, magicalBraces bool) string {
	runes := []rune(s)
	runes = unescapePosixClasses(runes, magicalBraces)
	return unescapePosixBackslashes(runes, magicalBraces)
}

// unescapePosixClasses applies the first TS replace (class unwrap with
// "not preceded by backslash" via ((?!\\).|^)).
func unescapePosixClasses(runes []rune, magicalBraces bool) []rune {
	if len(runes) == 0 {
		return runes
	}
	out := make([]rune, 0, len(runes))
	i := 0
	for i < len(runes) {
		// ^\[c\] at the current start of the remaining string (i==0 of original
		// is handled when i is the scan index; for global replace, ^ only at 0).
		// In JS, ^ matches only at the beginning of the string, not at each
		// resume point. Global search still only allows ^ at index 0.
		if i == 0 && len(runes) >= 3 && runes[0] == '[' && runes[2] == ']' {
			c := runes[1]
			if isUnescapeClassInner(c, magicalBraces) {
				out = append(out, c)
				i = 3
				continue
			}
		}

		// ((?!\\).)\[c\] — one non-\ character, then [c]
		if i+3 < len(runes) &&
			runes[i] != '\\' &&
			runes[i+1] == '[' &&
			runes[i+3] == ']' {
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

// unescapePosixBackslashes applies /\\([^/])/g or /\\([^/{}])/g.
func unescapePosixBackslashes(runes []rune, magicalBraces bool) string {
	var b strings.Builder
	b.Grow(len(runes))
	for i := 0; i < len(runes); i++ {
		if runes[i] == '\\' && i+1 < len(runes) {
			c := runes[i+1]
			if isUnescapeBackslashInner(c, magicalBraces) {
				b.WriteRune(c)
				i++
				continue
			}
		}
		b.WriteRune(runes[i])
	}
	return b.String()
}

// isUnescapeClassInner is the character class for TS [^/\\] or [^/\\{}].
func isUnescapeClassInner(c rune, magicalBraces bool) bool {
	if c == '/' || c == '\\' {
		return false
	}
	if !magicalBraces && (c == '{' || c == '}') {
		return false
	}
	return true
}

// isUnescapeBackslashInner is the character class for TS [^/] or [^/{}].
func isUnescapeBackslashInner(c rune, magicalBraces bool) bool {
	if c == '/' {
		return false
	}
	if !magicalBraces && (c == '{' || c == '}') {
		return false
	}
	return true
}
