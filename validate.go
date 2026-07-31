package minimatch

import "unicode/utf16"

// MaxPatternLength is the maximum allowed pattern length, measured in
// UTF-16 code units (the same unit as JavaScript's String.length).
// Patterns longer than this are rejected.
//
// Matches the TypeScript constant MAX_PATTERN_LENGTH = 1024 * 64.
const MaxPatternLength = 1024 * 64

// ValidatePattern reports whether pattern is acceptable for minimatch.
//
// Empty patterns are valid (they match only the empty path once matching
// exists). Patterns whose UTF-16 length is greater than MaxPatternLength
// return ErrPatternTooLong.
//
// Length is measured in UTF-16 code units, not Go bytes or Unicode code
// points, so that the limit matches JavaScript's pattern.length check.
//
// This corresponds to assertValidPattern in the TypeScript implementation,
// except that type rejection (ErrInvalidPattern) is a compile-time concern
// in Go when the caller already has a string.
//
// ValidatePattern does not parse or match globs; it only enforces the
// shared size/type gate used by every public entry point in the reference.
func ValidatePattern(pattern string) error {
	if utf16Len(pattern) > MaxPatternLength {
		return ErrPatternTooLong
	}
	return nil
}

// utf16Len returns the number of UTF-16 code units in s.
// Supplementary-plane runes count as two units, matching JS string length.
func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		n += utf16.RuneLen(r)
	}
	return n
}
