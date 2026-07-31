package minimatch

// Default values applied when the corresponding Options field is unset.
//
// These match the TypeScript Minimatch implementation: booleans default to
// false via the zero value; numeric and tri-state fields use the constants
// and pointer-nil rules documented on Options.
const (
	// DefaultOptimizationLevel is used when Options.OptimizationLevel is nil.
	// TypeScript: optimizationLevel ?? 1 (note: explicit 0 is valid and distinct).
	DefaultOptimizationLevel = 1

	// DefaultMaxGlobstarRecursion is used when Options.MaxGlobstarRecursion is nil.
	// TypeScript: maxGlobstarRecursion ?? 200
	DefaultMaxGlobstarRecursion = 200

	// DefaultMaxExtglobRecursion is used when Options.MaxExtglobRecursion is nil.
	// TypeScript: maxExtglobRecursion ?? 2
	DefaultMaxExtglobRecursion = 2

	// DefaultBraceExpandMax is used when Options.BraceExpandMax is nil.
	// TypeScript / brace-expansion: default max expansion count 100_000.
	DefaultBraceExpandMax = 100_000
)

// Options controls glob compilation and matching behaviour.
//
// Field names are idiomatic Go; comments give the TypeScript MinimatchOptions
// key. Zero-value Options matches TypeScript defaults for every boolean flag
// (false / unset).
//
// Pointer fields distinguish "unset → use default" from an explicit zero
// or false, which the TypeScript API expresses with undefined vs a provided
// value:
//
//   - OptimizationLevel nil → DefaultOptimizationLevel (1); non-nil 0 is level 0
//   - MaxGlobstarRecursion, MaxExtglobRecursion, BraceExpandMax nil → defaults
//   - WindowsNoMagicRoot nil → true when platform is win32 and NoCase is true
//
// Platform empty means "use HostPlatform()" at construction time (TypeScript
// process.platform).
//
// No matching or parsing is performed by this type alone; it is shared
// configuration for later subsystems.
type Options struct {
	// NoBrace disables {a,b} and {1..3} brace expansion.
	// TypeScript: nobrace
	NoBrace bool

	// NoComment disables treating a leading # as a comment pattern.
	// TypeScript: nocomment
	NoComment bool

	// NoNegate disables treating leading ! as pattern negation.
	// TypeScript: nonegate
	NoNegate bool

	// Debug enables verbose diagnostic output (stderr in TypeScript).
	// TypeScript: debug
	Debug bool

	// NoGlobStar treats ** the same as *.
	// TypeScript: noglobstar
	NoGlobStar bool

	// NoExt disables extglob patterns such as +(a|b).
	// TypeScript: noext
	NoExt bool

	// NoNull, when used with list Match, returns the pattern itself if
	// nothing matched (TypeScript nonull / nullglob-like behaviour).
	// TypeScript: nonull
	NoNull bool

	// WindowsPathsNoEscape treats \ as a path separator in patterns, never
	// as an escape character. When set, backslashes in the pattern are
	// rewritten to forward slashes.
	// TypeScript: windowsPathsNoEscape
	WindowsPathsNoEscape bool

	// AllowWindowsEscape is the deprecated inverse of WindowsPathsNoEscape.
	// In TypeScript, allowWindowsEscape === false forces windowsPathsNoEscape.
	// Prefer WindowsPathsNoEscape in new code.
	// TypeScript: allowWindowsEscape
	AllowWindowsEscape *bool

	// Partial allows a path to match if it is a valid prefix of a full
	// match (for filesystem walks).
	// TypeScript: partial
	Partial bool

	// Dot allows patterns to match path segments that start with '.' even
	// when the pattern does not contain an explicit leading dot there.
	// TypeScript: dot
	Dot bool

	// NoCase enables case-insensitive matching.
	// TypeScript: nocase
	NoCase bool

	// NoCaseMagicOnly, with NoCase, applies case-insensitivity only to
	// magic pattern parts, leaving literal string parts case-sensitive.
	// TypeScript: nocaseMagicOnly
	NoCaseMagicOnly bool

	// MagicalBraces makes brace expansions count as magic for HasMagic,
	// and affects escape/unescape of '{' and '}'.
	// TypeScript: magicalBraces
	MagicalBraces bool

	// MatchBase matches patterns that contain no slashes against the
	// basename of the path only.
	// TypeScript: matchBase
	MatchBase bool

	// FlipNegate returns true on a hit even for negated patterns (inverts
	// the usual handling of negate for the boolean result).
	// TypeScript: flipNegate
	FlipNegate bool

	// PreserveMultipleSlashes disables collapsing consecutive / characters
	// (except for Windows UNC handling rules applied elsewhere).
	// TypeScript: preserveMultipleSlashes
	PreserveMultipleSlashes bool

	// OptimizationLevel selects preprocess aggressiveness (0, 1, or ≥2).
	// Nil means DefaultOptimizationLevel. A pointer to 0 requests level 0.
	// TypeScript: optimizationLevel
	OptimizationLevel *int

	// Platform overrides host OS personality. Empty means HostPlatform().
	// TypeScript: platform
	Platform Platform

	// WindowsNoMagicRoot controls whether UNC/drive roots stay literal
	// strings under NoCase. Nil means default: true when win32 and NoCase.
	// TypeScript: windowsNoMagicRoot
	WindowsNoMagicRoot *bool

	// BraceExpandMax caps brace expansion cardinality. Nil means
	// DefaultBraceExpandMax.
	// TypeScript: braceExpandMax
	BraceExpandMax *int

	// MaxGlobstarRecursion caps recursive ** body walks. Nil means
	// DefaultMaxGlobstarRecursion. Exceeding the limit is a non-match.
	// TypeScript: maxGlobstarRecursion
	MaxGlobstarRecursion *int

	// MaxExtglobRecursion caps nested extglob parse depth. Nil means
	// DefaultMaxExtglobRecursion.
	// TypeScript: maxExtglobRecursion
	MaxExtglobRecursion *int
}

// EffectivePlatform returns o.Platform, or HostPlatform() if unset.
func (o Options) EffectivePlatform() Platform {
	if o.Platform == "" {
		return HostPlatform()
	}
	return o.Platform
}

// EffectiveOptimizationLevel returns the optimization level, applying
// DefaultOptimizationLevel when OptimizationLevel is nil.
func (o Options) EffectiveOptimizationLevel() int {
	if o.OptimizationLevel == nil {
		return DefaultOptimizationLevel
	}
	return *o.OptimizationLevel
}

// EffectiveMaxGlobstarRecursion returns the globstar recursion limit.
func (o Options) EffectiveMaxGlobstarRecursion() int {
	if o.MaxGlobstarRecursion == nil {
		return DefaultMaxGlobstarRecursion
	}
	return *o.MaxGlobstarRecursion
}

// EffectiveMaxExtglobRecursion returns the extglob nesting limit.
func (o Options) EffectiveMaxExtglobRecursion() int {
	if o.MaxExtglobRecursion == nil {
		return DefaultMaxExtglobRecursion
	}
	return *o.MaxExtglobRecursion
}

// EffectiveBraceExpandMax returns the brace expansion cardinality cap.
func (o Options) EffectiveBraceExpandMax() int {
	if o.BraceExpandMax == nil {
		return DefaultBraceExpandMax
	}
	return *o.BraceExpandMax
}

// EffectiveWindowsPathsNoEscape reports whether backslashes in patterns
// are path separators. True when WindowsPathsNoEscape is set, or when
// AllowWindowsEscape is explicitly false (TypeScript legacy behaviour).
func (o Options) EffectiveWindowsPathsNoEscape() bool {
	if o.WindowsPathsNoEscape {
		return true
	}
	if o.AllowWindowsEscape != nil && !*o.AllowWindowsEscape {
		return true
	}
	return false
}

// EffectiveWindowsNoMagicRoot reports whether UNC/drive roots should stay
// non-magic under case-insensitive matching. When WindowsNoMagicRoot is
// nil, the default is true only for win32 with NoCase (TypeScript).
func (o Options) EffectiveWindowsNoMagicRoot() bool {
	if o.WindowsNoMagicRoot != nil {
		return *o.WindowsNoMagicRoot
	}
	return o.EffectivePlatform().IsWindows() && o.NoCase
}
