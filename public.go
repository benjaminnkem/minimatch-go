package minimatch

import "github.com/dlclark/regexp2"

// HostSep is the path separator for the host platform (TypeScript minimatch.sep).
func HostSep() Sep {
	return HostPlatform().PathSep()
}

// HasMagic reports whether the compiled pattern contains magic segments.
//
// Corresponds to TypeScript Minimatch.hasMagic().
// With MagicalBraces, multiple brace alternatives count as magic even if
// each alternative is a pure literal.
func (m *Minimatch) HasMagic() bool {
	if m.Options.MagicalBraces && len(m.Set) > 1 {
		return true
	}
	for _, pattern := range m.Set {
		for _, part := range pattern {
			if !part.isStringPart() {
				return true
			}
		}
	}
	return false
}

// Filter returns a predicate suitable for filtering path lists.
//
// Corresponds to TypeScript minimatch.filter(pattern, options).
// Invalid patterns yield a predicate that always returns false.
func Filter(pattern string, opts Options) func(string) bool {
	return func(p string) bool {
		ok, err := Match(p, pattern, opts)
		return err == nil && ok
	}
}

// MatchList filters list to paths matching pattern.
//
// Corresponds to TypeScript minimatch.match(list, pattern, options).
// If nothing matches and Options.NoNull is set, returns []string{pattern}.
func MatchList(list []string, pattern string, opts Options) ([]string, error) {
	m, err := NewMinimatch(pattern, opts)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, f := range list {
		if m.Match(f) {
			out = append(out, f)
		}
	}
	if m.Options.NoNull && len(out) == 0 {
		return []string{pattern}, nil
	}
	return out, nil
}

// Defaults holds a base Options bag applied under per-call options.
//
// Corresponds to TypeScript minimatch.defaults(def). Boolean flags use
// OR-merge (true in either bag wins); pointer fields and Platform use
// non-nil/non-empty override wins. Call-site true flags always apply.
type Defaults struct {
	base Options
}

// NewDefaults returns a Defaults wrapper for def.
// Empty def behaves like the zero Options defaults.
func NewDefaults(def Options) Defaults {
	return Defaults{base: def}
}

// apply merges base defaults under opts (call-site wins for set pointers /
// true bools / non-empty platform).
func (d Defaults) apply(opts Options) Options {
	return applyDefaults(d.base, opts)
}

// applyDefaults merges base under over roughly like Object.assign for our
// Options layout. True booleans in over set true; pointer non-nil and
// non-empty Platform in over win. Base values remain when over leaves
// zero/nil (cannot force false over a true default without a pointer field).
func applyDefaults(base, over Options) Options {
	out := base
	if over.NoBrace {
		out.NoBrace = true
	}
	if over.NoComment {
		out.NoComment = true
	}
	if over.NoNegate {
		out.NoNegate = true
	}
	if over.Debug {
		out.Debug = true
	}
	if over.NoGlobStar {
		out.NoGlobStar = true
	}
	if over.NoExt {
		out.NoExt = true
	}
	if over.NoNull {
		out.NoNull = true
	}
	if over.WindowsPathsNoEscape {
		out.WindowsPathsNoEscape = true
	}
	if over.AllowWindowsEscape != nil {
		out.AllowWindowsEscape = over.AllowWindowsEscape
	}
	if over.Partial {
		out.Partial = true
	}
	if over.Dot {
		out.Dot = true
	}
	if over.NoCase {
		out.NoCase = true
	}
	if over.NoCaseMagicOnly {
		out.NoCaseMagicOnly = true
	}
	if over.MagicalBraces {
		out.MagicalBraces = true
	}
	if over.MatchBase {
		out.MatchBase = true
	}
	if over.FlipNegate {
		out.FlipNegate = true
	}
	if over.PreserveMultipleSlashes {
		out.PreserveMultipleSlashes = true
	}
	if over.OptimizationLevel != nil {
		out.OptimizationLevel = over.OptimizationLevel
	}
	if over.Platform != "" {
		out.Platform = over.Platform
	}
	if over.WindowsNoMagicRoot != nil {
		out.WindowsNoMagicRoot = over.WindowsNoMagicRoot
	}
	if over.BraceExpandMax != nil {
		out.BraceExpandMax = over.BraceExpandMax
	}
	if over.MaxGlobstarRecursion != nil {
		out.MaxGlobstarRecursion = over.MaxGlobstarRecursion
	}
	if over.MaxExtglobRecursion != nil {
		out.MaxExtglobRecursion = over.MaxExtglobRecursion
	}
	return out
}

// Match applies defaults then Match.
func (d Defaults) Match(p, pattern string, opts Options) (bool, error) {
	return Match(p, pattern, d.apply(opts))
}

// Filter applies defaults then Filter.
func (d Defaults) Filter(pattern string, opts Options) func(string) bool {
	return Filter(pattern, d.apply(opts))
}

// MatchList applies defaults then MatchList.
func (d Defaults) MatchList(list []string, pattern string, opts Options) ([]string, error) {
	return MatchList(list, pattern, d.apply(opts))
}

// NewMinimatch applies defaults then NewMinimatch.
func (d Defaults) NewMinimatch(pattern string, opts Options) (*Minimatch, error) {
	return NewMinimatch(pattern, d.apply(opts))
}

// MakeRe applies defaults then MakeRe.
func (d Defaults) MakeRe(pattern string, opts Options) (*regexp2.Regexp, bool, error) {
	return MakeRe(pattern, d.apply(opts))
}

// BraceExpand applies defaults then BraceExpand.
func (d Defaults) BraceExpand(pattern string, opts Options) ([]string, error) {
	return BraceExpand(pattern, d.apply(opts))
}

// Escape applies defaults then Escape (windowsPathsNoEscape / magicalBraces).
func (d Defaults) Escape(s string, opts EscapeOptions) string {
	merged := d.apply(Options{
		WindowsPathsNoEscape: opts.WindowsPathsNoEscape,
		MagicalBraces:        opts.MagicalBraces,
	})
	return Escape(s, EscapeOptionsFrom(merged))
}

// Unescape applies defaults then Unescape.
func (d Defaults) Unescape(s string, opts UnescapeOptions) string {
	merged := d.apply(Options{WindowsPathsNoEscape: opts.WindowsPathsNoEscape})
	uopts := UnescapeOptions{
		WindowsPathsNoEscape: merged.EffectiveWindowsPathsNoEscape(),
		MagicalBraces:        opts.MagicalBraces, // nil keeps TS default true
	}
	if opts.MagicalBraces == nil && merged.MagicalBraces {
		// base defaults asked for magical braces true
		t := true
		uopts.MagicalBraces = &t
	}
	return Unescape(s, uopts)
}

// ParseGlob applies defaults then ParseGlob.
func (d Defaults) ParseGlob(pattern string, opts Options) *AST {
	return ParseGlob(pattern, d.apply(opts))
}
