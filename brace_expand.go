package minimatch

import "github.com/benjaminnkem/minimatch-go/internal/brace"

// ExpansionMax is the default brace expansion cardinality cap.
const ExpansionMax = brace.ExpansionMax

// ExpansionMaxLength caps total expansion character volume.
const ExpansionMaxLength = brace.ExpansionMaxLength

// BraceExpand performs bash-style brace expansion on pattern.
//
// Corresponds to TypeScript minimatch.braceExpand.
func BraceExpand(pattern string, opts Options) ([]string, error) {
	if err := ValidatePattern(pattern); err != nil {
		return nil, err
	}
	if opts.NoBrace || !brace.HasSimpleGroup(pattern) {
		return []string{pattern}, nil
	}
	return brace.Expand(pattern, opts.EffectiveBraceExpandMax(), ExpansionMaxLength), nil
}
