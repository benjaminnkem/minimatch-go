package minimatch

// Package-level default constants for Options fields whose TypeScript
// default is not the Go zero value (or where “unset” must differ from 0).
//
// Boolean MinimatchOptions in TypeScript default to false when omitted.
// That maps directly to the Go zero value for bool fields on Options.
//
// Numeric and tri-state fields use nil pointers for “omitted / undefined”
// so that an explicit 0 or false remains representable.
const (
	// DefaultOptimizationLevel is applied when Options.OptimizationLevel is nil.
	//
	// TypeScript: const { optimizationLevel = 1 } = this.options
	// Explicit 0 is a valid, distinct level (no .. collapsing beyond adjacent **).
	DefaultOptimizationLevel = 1

	// DefaultMaxGlobstarRecursion is applied when Options.MaxGlobstarRecursion is nil.
	//
	// TypeScript: options.maxGlobstarRecursion ?? 200
	DefaultMaxGlobstarRecursion = 200

	// DefaultMaxExtglobRecursion is applied when Options.MaxExtglobRecursion is nil.
	//
	// TypeScript: options.maxExtglobRecursion ?? 2
	DefaultMaxExtglobRecursion = 2

	// DefaultBraceExpandMax is applied when Options.BraceExpandMax is nil.
	//
	// TypeScript passes options.braceExpandMax into brace-expansion; when
	// undefined, brace-expansion uses 100_000.
	DefaultBraceExpandMax = 100_000
)

// Options controls glob compilation and matching behaviour.
//
// It is the Go equivalent of TypeScript MinimatchOptions. Field names are
// idiomatic Go; each field documents its TypeScript key.
//
// Options is pure configuration: reading or constructing it does not parse
// patterns, expand braces, or match paths. Later subsystems consume Options
// (and the Effective* helpers) when those behaviours are implemented.
//
// # Default behaviour (zero value)
//
//	var o Options  // and Options{}
//
// matches TypeScript `{}` / omitted options for all flags:
//
//	NoBrace, NoComment, NoNegate, Debug, NoGlobStar, NoExt, NoNull,
//	WindowsPathsNoEscape, Partial, Dot, NoCase, NoCaseMagicOnly,
//	MagicalBraces, MatchBase, FlipNegate, PreserveMultipleSlashes
//	  → false
//
//	AllowWindowsEscape     → nil (undefined; does not force WindowsPathsNoEscape)
//	OptimizationLevel      → nil → EffectiveOptimizationLevel() == 1
//	Platform               → ""  → EffectivePlatform() == HostPlatform()
//	WindowsNoMagicRoot     → nil → true iff win32 && NoCase
//	BraceExpandMax         → nil → 100_000
//	MaxGlobstarRecursion   → nil → 200
//	MaxExtglobRecursion    → nil → 2
//
// Use Bool and Int helpers to set pointer fields without awkward locals:
//
//	opts := Options{OptimizationLevel: Int(0), WindowsNoMagicRoot: Bool(false)}
//
// # Relation to Escape / Unescape
//
// Escape and Unescape use small option structs (EscapeOptions,
// UnescapeOptions) because TypeScript applies different defaults for
// magicalBraces on those free functions (false vs true). EscapeOptionsFrom
// and UnescapeOptionsFrom project a full Options value into those structs.
type Options struct {
	// NoBrace disables brace expansion of {a,b} and {1..3} style sets.
	//
	// When false (default), brace expansion runs before other interpretation,
	// so patterns that look invalid before expansion can become valid after.
	// When true, the pattern is left unchanged by the brace-expand step.
	//
	// TypeScript: nobrace
	// Default: false
	NoBrace bool

	// NoComment disables treating a pattern that starts with '#' as a comment.
	//
	// When false (default), a leading '#' means the pattern matches nothing
	// (comment). When true, '#' is ordinary pattern text.
	//
	// TypeScript: nocomment
	// Default: false
	NoComment bool

	// NoNegate disables treating leading '!' characters as pattern negation.
	//
	// When false (default), each leading '!' toggles negation (so "!!" cancels).
	// When true, leading '!' is ordinary pattern text (useful when the pattern
	// should start with a negative extglob like "!(a|b)").
	//
	// TypeScript: nonegate
	// Default: false
	NoNegate bool

	// Debug enables verbose diagnostic logging during compile/match.
	//
	// In TypeScript this prints to stderr via console.error. The Go port will
	// honour this flag when matching is implemented; the flag itself carries
	// no behaviour in the Options model alone.
	//
	// TypeScript: debug
	// Default: false
	Debug bool

	// NoGlobStar disables multi-directory ** semantics.
	//
	// When false (default), a path segment that is exactly "**" is a globstar.
	// When true, "**" is treated like "*". Adjacent "**" collapsing and
	// globstar matching do not apply beyond that rewrite.
	//
	// TypeScript: noglobstar
	// Default: false
	NoGlobStar bool

	// NoExt disables extglob patterns such as +(a|b), *(a|b), ?(a|b),
	// @(a|b), and !(a|b).
	//
	// When true, those forms are not parsed as extglobs (they are ordinary
	// characters / other magic, depending on the rest of the pattern).
	//
	// TypeScript: noext
	// Default: false
	NoExt bool

	// NoNull changes list-filter behaviour when nothing matches.
	//
	// When used with the list Match API (TypeScript minimatch.match): if no
	// path matches and NoNull is true, the result is a one-element list
	// containing the pattern string itself; if false (default), the result
	// is an empty list. This is akin to bash nullglob being off when NoNull
	// is true (return the pattern), but escaped characters are not resolved.
	//
	// TypeScript: nonull
	// Default: false
	NoNull bool

	// WindowsPathsNoEscape treats '\' in patterns as a path separator only,
	// never as an escape character.
	//
	// When true, all '\' in the pattern are rewritten to '/' before further
	// processing. That makes it impossible to escape magic characters with
	// backslashes, but allows patterns built with Windows path.join-style
	// strings. Prefer forward slashes in patterns when possible.
	//
	// Also becomes effective when AllowWindowsEscape is explicitly false
	// (legacy TypeScript behaviour). See EffectiveWindowsPathsNoEscape.
	//
	// TypeScript: windowsPathsNoEscape
	// Default: false
	WindowsPathsNoEscape bool

	// AllowWindowsEscape is the deprecated inverse of WindowsPathsNoEscape.
	//
	// TypeScript only treats the exact value false as meaningful:
	// allowWindowsEscape === false forces windowsPathsNoEscape on.
	// true or undefined leave WindowsPathsNoEscape unchanged.
	//
	// Nil means undefined (default). Prefer WindowsPathsNoEscape in new code.
	//
	// TypeScript: allowWindowsEscape (deprecated)
	// Default: nil (undefined)
	AllowWindowsEscape *bool

	// Partial enables prefix matching for incomplete paths.
	//
	// When true, a path matches if the path segments present do not
	// contradict the pattern — useful while walking a tree before the full
	// path exists. Example (TypeScript semantics):
	//
	//	partial /a/b against /a/*/c/d → true (might become /a/b/c/d)
	//	partial /x/y/z against /a/**/z → false (x !== a)
	//
	// TypeScript: partial
	// Default: false
	Partial bool

	// Dot allows matching path segments that start with '.' even when the
	// pattern does not place a literal dot (or other explicit dot-matching
	// form) in that position.
	//
	// When false (default), patterns like "*" and "a/**/b" do not match
	// ".hidden" or "a/.d/b". When true, those matches are allowed subject
	// to the rest of the pattern. "." and ".." still have special cases in
	// the matcher (documented with matching, not here).
	//
	// TypeScript: dot
	// Default: false
	Dot bool

	// NoCase enables case-insensitive matching.
	//
	// When true, magic portions typically become case-insensitive (e.g.
	// regular expressions with the 'i' flag in TypeScript), and some
	// comparisons fold case. Interacts with NoCaseMagicOnly and
	// WindowsNoMagicRoot.
	//
	// TypeScript: nocase
	// Default: false
	NoCase bool

	// NoCaseMagicOnly, together with NoCase, limits case-insensitivity to
	// magic pattern parts only.
	//
	// When NoCase is true and NoCaseMagicOnly is true, literal string
	// segments stay case-sensitive while wildcards/classes use case-insensitive
	// rules. Has no effect when NoCase is false.
	//
	// TypeScript: nocaseMagicOnly
	// Default: false
	NoCaseMagicOnly bool

	// MagicalBraces controls whether brace expansion counts as “magic” for
	// HasMagic, and whether Escape/Unescape treat '{' and '}' as magic.
	//
	// When false (default), a pattern like "a{b,c}d" has HasMagic false if
	// the expanded alternatives have no other magic. When true, multiple
	// brace alternatives are treated as magic.
	//
	// Note: the free functions Escape and Unescape use their own option
	// structs; Unescape defaults magicalBraces to true even though this
	// field defaults to false on Options (TypeScript free-function defaults).
	//
	// TypeScript: magicalBraces
	// Default: false
	MagicalBraces bool

	// MatchBase matches a pattern that contains no '/' against the basenames
	// of paths that do contain slashes.
	//
	// Example: pattern "a?b" with MatchBase matches path "/xyz/123/acb" but
	// not "/xyz/acb/123".
	//
	// TypeScript: matchBase
	// Default: false
	MatchBase bool

	// FlipNegate changes the boolean result of negated patterns.
	//
	// Normally a negated pattern returns false on a hit (path is excluded).
	// With FlipNegate true, a hit returns true and a miss returns false —
	// as if the pattern were not negated for the purpose of the return value.
	//
	// TypeScript: flipNegate
	// Default: false
	FlipNegate bool

	// PreserveMultipleSlashes disables collapsing consecutive '/' characters
	// in patterns and paths.
	//
	// When false (default), "a///b" is treated like "a/b", except that a
	// leading "//" on Windows UNC forms is preserved specially. When true,
	// empty path segments from repeated slashes are kept.
	//
	// TypeScript: preserveMultipleSlashes
	// Default: false
	PreserveMultipleSlashes bool

	// OptimizationLevel selects how aggressively patterns are rewritten
	// before matching (TypeScript preprocess).
	//
	// Nil means DefaultOptimizationLevel (1). A non-nil pointer to 0
	// requests level 0 (explicit zero is not the same as unset).
	//
	//	0  — only collapse adjacent ** (when not noglobstar); keep . and ..
	//	1  — default; also cancel p/.. when p is not **, ., .., or empty
	//	≥2 — aggressive rewrites for filesystem walks (may diverge from
	//	     makeRe unless the path is optimized similarly)
	//
	// noglobstar always rewrites ** → * regardless of level. Adjacent **
	// collapsing always applies.
	//
	// TypeScript: optimizationLevel
	// Default: nil → 1
	OptimizationLevel *int

	// Platform selects OS personality for path rules (UNC, '\', drive letters).
	//
	// Empty means HostPlatform() (TypeScript process.platform). Only
	// PlatformWin32 ("win32") enables Windows-specific matching behaviour;
	// other values behave like POSIX for matching purposes.
	//
	// TypeScript: platform
	// Default: "" → HostPlatform()
	Platform Platform

	// WindowsNoMagicRoot keeps UNC/drive root segments as literal strings
	// under case-insensitive mode instead of case-insensitive magic.
	//
	// When nil, defaults to true if EffectivePlatform is win32 and NoCase
	// is true; otherwise false. When non-nil, that value is used exactly.
	//
	// TypeScript: windowsNoMagicRoot
	// Default: nil → (win32 && NoCase)
	WindowsNoMagicRoot *bool

	// BraceExpandMax caps how many strings brace expansion may produce.
	//
	// Nil means DefaultBraceExpandMax (100_000). Passed through to the
	// brace-expansion step when that subsystem exists.
	//
	// TypeScript: braceExpandMax
	// Default: nil → 100_000
	BraceExpandMax *int

	// MaxGlobstarRecursion caps how many non-adjacent ** body sections may
	// be walked recursively during matching.
	//
	// Nil means DefaultMaxGlobstarRecursion (200). If the limit is exceeded,
	// the reference treats the path as non-matching (intentional false
	// negative for security/performance).
	//
	// TypeScript: maxGlobstarRecursion
	// Default: nil → 200
	MaxGlobstarRecursion *int

	// MaxExtglobRecursion caps nested extglob parse depth (e.g. *(a|*(b|c))).
	//
	// Nil means DefaultMaxExtglobRecursion (2). When the limit is hit, nested
	// extglob syntax is not parsed further (effectively noext for that nest);
	// adoption/flattening of nestable forms can avoid hitting the limit.
	//
	// TypeScript: maxExtglobRecursion
	// Default: nil → 2
	MaxExtglobRecursion *int
}

// Bool returns a *bool suitable for optional Options fields such as
// AllowWindowsEscape and WindowsNoMagicRoot.
func Bool(v bool) *bool { return &v }

// Int returns a *int suitable for optional Options fields such as
// OptimizationLevel, BraceExpandMax, MaxGlobstarRecursion, and
// MaxExtglobRecursion.
func Int(v int) *int { return &v }

// EffectivePlatform returns o.Platform, or HostPlatform() when Platform is
// empty (TypeScript: options.platform || process.platform).
func (o Options) EffectivePlatform() Platform {
	if o.Platform == "" {
		return HostPlatform()
	}
	return o.Platform
}

// EffectiveOptimizationLevel returns the optimization level, applying
// DefaultOptimizationLevel when OptimizationLevel is nil.
//
// TypeScript: const { optimizationLevel = 1 } = this.options
func (o Options) EffectiveOptimizationLevel() int {
	if o.OptimizationLevel == nil {
		return DefaultOptimizationLevel
	}
	return *o.OptimizationLevel
}

// EffectiveMaxGlobstarRecursion returns the ** recursion limit.
//
// TypeScript: options.maxGlobstarRecursion ?? 200
func (o Options) EffectiveMaxGlobstarRecursion() int {
	if o.MaxGlobstarRecursion == nil {
		return DefaultMaxGlobstarRecursion
	}
	return *o.MaxGlobstarRecursion
}

// EffectiveMaxExtglobRecursion returns the nested extglob depth limit.
//
// TypeScript: options.maxExtglobRecursion ?? 2
func (o Options) EffectiveMaxExtglobRecursion() int {
	if o.MaxExtglobRecursion == nil {
		return DefaultMaxExtglobRecursion
	}
	return *o.MaxExtglobRecursion
}

// EffectiveBraceExpandMax returns the brace expansion cardinality cap.
//
// TypeScript / brace-expansion: options.braceExpandMax ?? 100_000
func (o Options) EffectiveBraceExpandMax() int {
	if o.BraceExpandMax == nil {
		return DefaultBraceExpandMax
	}
	return *o.BraceExpandMax
}

// EffectiveWindowsPathsNoEscape reports whether '\' in patterns is a path
// separator (and not an escape).
//
// True when WindowsPathsNoEscape is true, or when AllowWindowsEscape is
// explicitly false (TypeScript: !!windowsPathsNoEscape || allowWindowsEscape === false).
func (o Options) EffectiveWindowsPathsNoEscape() bool {
	if o.WindowsPathsNoEscape {
		return true
	}
	if o.AllowWindowsEscape != nil && !*o.AllowWindowsEscape {
		return true
	}
	return false
}

// EffectiveWindowsNoMagicRoot reports whether UNC/drive roots should remain
// non-magic under NoCase.
//
// TypeScript:
//
//	windowsNoMagicRoot !== undefined
//	  ? windowsNoMagicRoot
//	  : !!(isWindows && nocase)
func (o Options) EffectiveWindowsNoMagicRoot() bool {
	if o.WindowsNoMagicRoot != nil {
		return *o.WindowsNoMagicRoot
	}
	return o.EffectivePlatform().IsWindows() && o.NoCase
}

// EffectiveIsWindows reports whether EffectivePlatform is win32.
func (o Options) EffectiveIsWindows() bool {
	return o.EffectivePlatform().IsWindows()
}
