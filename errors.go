package minimatch

import "errors"

// Sentinel errors shared across the package.
//
// Message strings match the TypeScript TypeError messages where the
// reference implementation throws, so behavioural tests can compare text.
var (
	// ErrInvalidPattern is returned when a pattern is not a valid string.
	// In TypeScript this covers non-string values (null, numbers, objects).
	// The typed Go API accepts string, so call sites with a string typically
	// never see this; it remains for API parity and untyped adapters.
	// TypeScript message: "invalid pattern"
	ErrInvalidPattern = errors.New("invalid pattern")

	// ErrPatternTooLong is returned when a pattern exceeds MaxPatternLength
	// UTF-16 code units.
	// TypeScript message: "pattern is too long"
	ErrPatternTooLong = errors.New("pattern is too long")
)
