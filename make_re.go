package minimatch

import (
	"strings"

	"github.com/dlclark/regexp2"
)

// Full-path ** fragments (TypeScript index.ts twoStarDot / twoStarNoDot / star).
const (
	makeReTwoStarDot   = `(?:(?!(?:\/|^)(?:\.{1,2})($|\/)).)*?`
	makeReTwoStarNoDot = `(?:(?!(?:\/|^)\.).)*?`
	makeReStar         = `[^/]*?`
)

// MakeRe builds a single regular expression for the entire path pattern.
//
// Corresponds to TypeScript Minimatch.makeRe() / minimatch.makeRe().
// Returns (nil, false) when the pattern cannot form a useful regexp
// (empty set after compile), matching TypeScript's false return.
//
// Prefer Match for correctness with optimizationLevel ≥ 2; MakeRe is a
// convenience for fnmatch-style full-string tests.
func (m *Minimatch) MakeRe() (*regexp2.Regexp, bool) {
	if m.makeReBuilt {
		if !m.makeReOK {
			return nil, false
		}
		if m.makeReCached != nil {
			return m.makeReCached, true
		}
		re, err := regexp2.Compile(m.makeReSrc, m.makeReFlags())
		if err != nil {
			return nil, false
		}
		m.makeReCached = re
		return re, true
	}
	m.makeReBuilt = true

	if len(m.Set) == 0 {
		m.makeReOK = false
		return nil, false
	}

	twoStar := makeReTwoStarNoDot
	if m.Options.NoGlobStar {
		twoStar = makeReStar
	} else if m.Options.Dot {
		twoStar = makeReTwoStarDot
	}

	var alts []string
	needU := false
	for _, pattern := range m.Set {
		// Build pp as string or globstar markers
		type item struct {
			gs  bool
			src string
		}
		pp := make([]item, len(pattern))
		for i, p := range pattern {
			if p.IsGlobStar {
				pp[i] = item{gs: true}
				continue
			}
			if p.UFlag {
				needU = true
			}
			pp[i] = item{src: p.reSrc()}
		}

		// Expand GLOBSTAR positions (TypeScript forEach over pp)
		for i := 0; i < len(pp); i++ {
			if !pp[i].gs {
				continue
			}
			var prevGS bool
			if i > 0 {
				prevGS = pp[i-1].gs
			}
			if prevGS {
				continue
			}
			var next *item
			if i+1 < len(pp) {
				next = &pp[i+1]
			}
			var prev *item
			if i > 0 {
				prev = &pp[i-1]
			}

			if prev == nil {
				if next != nil && !next.gs {
					next.src = `(?:\/|` + twoStar + `\/)?` + next.src
				} else {
					pp[i] = item{src: twoStar}
					pp[i].gs = false
				}
			} else if next == nil {
				prev.src = prev.src + `(?:\/|\/` + twoStar + `)?`
			} else if !next.gs {
				prev.src = prev.src + `(?:\/|\/` + twoStar + `\/)` + next.src
				// mark next as consumed GS placeholder
				pp[i+1] = item{gs: true} // will filter; TS sets GLOBSTAR
				// Actually TS: pp[i+1] = GLOBSTAR so filtered out, and next content already in prev
				pp[i+1] = item{gs: true, src: ""}
			}
		}

		var filtered []string
		for _, it := range pp {
			if it.gs {
				continue
			}
			filtered = append(filtered, it.src)
		}

		if m.Partial && len(filtered) >= 1 {
			var prefixes []string
			for i := 1; i <= len(filtered); i++ {
				prefixes = append(prefixes, strings.Join(filtered[:i], "/"))
			}
			alts = append(alts, `(?:`+strings.Join(prefixes, "|")+`)`)
		} else {
			alts = append(alts, strings.Join(filtered, "/"))
		}
	}

	reBody := strings.Join(alts, "|")
	open, close := "", ""
	if len(m.Set) > 1 {
		open, close = `(?:`, `)`
	}
	re := `^` + open + reBody + close + `$`

	if m.Partial {
		// '^(?:\\/|' + open + re.slice(1, -1) + close + ')$'
		inner := re
		if len(inner) >= 2 {
			inner = inner[1 : len(inner)-1]
		}
		re = `^(?:\/|` + open + inner + close + `)$`
	}

	if m.Negate {
		re = `^(?!` + re + `).+$`
	}

	m.makeReSrc = re
	m.makeReUFlag = needU
	flags := m.makeReFlags()
	compiled, err := regexp2.Compile(re, flags)
	if err != nil {
		m.makeReOK = false
		return nil, false
	}
	m.makeReOK = true
	m.makeReCached = compiled
	return compiled, true
}

// fields for MakeRe cache (extended)
func (m *Minimatch) makeReFlags() regexp2.RegexOptions {
	flags := regexp2.None
	if m.Options.NoCase {
		flags |= regexp2.IgnoreCase
	}
	if m.makeReUFlag {
		flags |= regexp2.Unicode
	}
	return flags
}

// MakeRe compiles pattern to a full-path regular expression.
// Corresponds to minimatch.makeRe(pattern, options).
func MakeRe(pattern string, opts Options) (*regexp2.Regexp, bool, error) {
	m, err := NewMinimatch(pattern, opts)
	if err != nil {
		return nil, false, err
	}
	re, ok := m.MakeRe()
	return re, ok, nil
}
