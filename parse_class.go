package minimatch

import "github.com/benjaminnkem/minimatch-go/internal/class"

// ParseClassResult is the outcome of ParseClass.
type ParseClassResult = class.ParseClassResult

// ParseClass parses a glob character class at position in pattern.
func ParseClass(pattern string, position int) (ParseClassResult, error) {
	return class.ParseClass(pattern, position)
}
