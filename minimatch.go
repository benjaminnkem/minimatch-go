package minimatch

import (
	"regexp"
	"strings"
)

// Minimatch is a compiled glob pattern for path matching.
//
// Corresponds to the TypeScript Minimatch class through the make() pipeline
// stages that produce GlobSet and GlobParts (validate, comment/empty,
// negate, brace expand, slashSplit, preprocess). Segment compilation into
// matchers (set) and path matching are later steps.
type Minimatch struct {
	// Options is the options bag used to build this pattern.
	Options Options
	// Pattern is the working pattern string (may have leading ! stripped
	// and \\ rewritten to / under WindowsPathsNoEscape).
	Pattern string

	// Negate is true when an odd number of leading ! were stripped.
	Negate bool
	// Comment is true when the pattern is a # comment (matches nothing).
	Comment bool
	// Empty is true when the pattern is the empty string (matches only "").
	Empty bool

	// Resolved flags / platform (mirrors TS instance fields).
	Nonegate                bool
	Partial                 bool
	NoCase                  bool
	PreserveMultipleSlashes bool
	WindowsPathsNoEscape    bool
	WindowsNoMagicRoot      bool
	IsWindows               bool
	Platform                Platform
	MaxGlobstarRecursion    int

	// GlobSet is brace-expanded unique alternatives (order preserved).
	GlobSet []string
	// GlobParts is each GlobSet entry slash-split and preprocessed.
	GlobParts [][]string
}

// driveLetterRE matches a drive root segment like "C:".
var driveLetterRE = regexp.MustCompile(`(?i)^[a-z]:$`)

// uncLeadRE matches //host… for Windows UNC preservation in slashSplit.
var uncLeadRE = regexp.MustCompile(`^//[^/]+`)

// NewMinimatch validates pattern and runs the compile pipeline through
// preprocess (GlobParts). It does not yet build per-segment matchers.
//
// Corresponds to `new Minimatch(pattern, options)` up to and including
// this.globParts = this.preprocess(...).
func NewMinimatch(pattern string, opts Options) (*Minimatch, error) {
	if err := ValidatePattern(pattern); err != nil {
		return nil, err
	}

	m := &Minimatch{
		Options:                 opts,
		Pattern:                 pattern,
		MaxGlobstarRecursion:    opts.EffectiveMaxGlobstarRecursion(),
		Platform:                opts.EffectivePlatform(),
		Nonegate:                opts.NoNegate,
		Partial:                 opts.Partial,
		NoCase:                  opts.NoCase,
		PreserveMultipleSlashes: opts.PreserveMultipleSlashes,
		WindowsPathsNoEscape:    opts.EffectiveWindowsPathsNoEscape(),
	}
	m.IsWindows = m.Platform.IsWindows()
	m.WindowsNoMagicRoot = opts.EffectiveWindowsNoMagicRoot()

	if m.WindowsPathsNoEscape {
		m.Pattern = strings.ReplaceAll(m.Pattern, `\`, `/`)
	}

	if err := m.makeThroughPreprocess(); err != nil {
		return nil, err
	}
	return m, nil
}

// makeThroughPreprocess runs comment/empty/negate/brace/split/preprocess.
func (m *Minimatch) makeThroughPreprocess() error {
	pattern := m.Pattern
	opts := m.Options

	// empty patterns and comments match nothing.
	if !opts.NoComment && len(pattern) > 0 && pattern[0] == '#' {
		m.Comment = true
		return nil
	}
	if pattern == "" {
		m.Empty = true
		return nil
	}

	m.parseNegate()

	expanded, err := BraceExpand(m.Pattern, opts)
	if err != nil {
		return err
	}
	m.GlobSet = uniquePreserveOrder(expanded)

	raw := make([][]string, len(m.GlobSet))
	for i, s := range m.GlobSet {
		raw[i] = m.SlashSplit(s)
	}
	m.GlobParts = m.Preprocess(raw)
	return nil
}

// parseNegate strips leading ! characters and sets Negate.
// TypeScript Minimatch.parseNegate.
func (m *Minimatch) parseNegate() {
	if m.Nonegate {
		return
	}
	pattern := m.Pattern
	negate := false
	offset := 0
	for offset < len(pattern) && pattern[offset] == '!' {
		negate = !negate
		offset++
	}
	if offset > 0 {
		m.Pattern = pattern[offset:]
	}
	m.Negate = negate
}

// SlashSplit splits a path or pattern on / according to platform rules.
//
// TypeScript Minimatch.slashSplit:
//   - preserveMultipleSlashes: split on each /
//   - win32 UNC //host…: preserve leading empty segments
//   - else: coalesce runs of / via split on /+
func (m *Minimatch) SlashSplit(p string) []string {
	if m.PreserveMultipleSlashes {
		return strings.Split(p, "/")
	}
	if m.IsWindows && uncLeadRE.MatchString(p) {
		// add an extra '' for the one we lose
		parts := splitCoalesceSlashes(p)
		return append([]string{""}, parts...)
	}
	return splitCoalesceSlashes(p)
}

// splitCoalesceSlashes is p.split(/\/+/) in JavaScript.
func splitCoalesceSlashes(p string) []string {
	if p == "" {
		return []string{""}
	}
	// strings.Split on regexp-like /+
	parts := slashPlusRE.Split(p, -1)
	return parts
}

var slashPlusRE = regexp.MustCompile(`/+`)

// Preprocess applies noglobstar rewrite and optimization-level transforms.
// TypeScript Minimatch.preprocess.
func (m *Minimatch) Preprocess(globParts [][]string) [][]string {
	// defensive copy of outer slice; inner slices may be mutated in place
	// after we clone each row.
	out := make([][]string, len(globParts))
	for i, row := range globParts {
		out[i] = append([]string(nil), row...)
	}

	if m.Options.NoGlobStar {
		for _, partset := range out {
			for j, p := range partset {
				if p == "**" {
					partset[j] = "*"
				}
			}
		}
	}

	level := m.Options.EffectiveOptimizationLevel()
	if level >= 2 {
		out = m.firstPhasePreProcess(out)
		out = m.secondPhasePreProcess(out)
	} else if level >= 1 {
		out = m.levelOneOptimize(out)
	} else {
		out = m.adjacentGlobstarOptimize(out)
	}
	return out
}

// adjacentGlobstarOptimize collapses consecutive ** (TS adjascentGlobstarOptimize).
func (m *Minimatch) adjacentGlobstarOptimize(globParts [][]string) [][]string {
	for i, parts := range globParts {
		gs := indexOf(parts, "**", 0)
		for gs != -1 {
			j := gs
			for j+1 < len(parts) && parts[j+1] == "**" {
				j++
			}
			if j != gs {
				// splice(gs, j-gs) removes extras, keeps one **
				parts = append(parts[:gs], parts[j:]...)
			}
			gs = indexOf(parts, "**", gs+1)
		}
		if len(parts) == 0 {
			parts = []string{""}
		}
		globParts[i] = parts
	}
	return globParts
}

// levelOneOptimize collapses ** and cancels p/.. (TS levelOneOptimize).
func (m *Minimatch) levelOneOptimize(globParts [][]string) [][]string {
	for i, parts := range globParts {
		var set []string
		for _, part := range parts {
			prev := ""
			if len(set) > 0 {
				prev = set[len(set)-1]
			}
			if part == "**" && prev == "**" {
				continue
			}
			if part == ".." {
				if prev != "" && prev != ".." && prev != "." && prev != "**" {
					set = set[:len(set)-1]
					continue
				}
			}
			set = append(set, part)
		}
		if len(set) == 0 {
			set = []string{""}
		}
		globParts[i] = set
	}
	return globParts
}

// LevelTwoFileOptimize optimizes a file path the way match() does under
// optimizationLevel >= 2 (TS levelTwoFileOptimize).
func (m *Minimatch) LevelTwoFileOptimize(parts []string) []string {
	parts = append([]string(nil), parts...)
	didSomething := true
	for didSomething {
		didSomething = false
		if !m.PreserveMultipleSlashes {
			for i := 1; i < len(parts)-1; i++ {
				p := parts[i]
				if i == 1 && p == "" && parts[0] == "" {
					continue
				}
				if p == "." || p == "" {
					didSomething = true
					parts = append(parts[:i], parts[i+1:]...)
					i--
				}
			}
			if len(parts) == 2 && parts[0] == "." && (parts[1] == "." || parts[1] == "") {
				didSomething = true
				parts = parts[:1]
			}
		}
		dd := indexOf(parts, "..", 1)
		for dd != -1 {
			var p string
			if dd > 0 {
				p = parts[dd-1]
			}
			if p != "" && p != "." && p != ".." && p != "**" &&
				!(m.IsWindows && driveLetterRE.MatchString(p)) {
				didSomething = true
				parts = append(parts[:dd-1], parts[dd+1:]...)
				dd -= 2
				if dd < 0 {
					dd = 0
				}
			}
			dd = indexOf(parts, "..", dd+1)
		}
	}
	if len(parts) == 0 {
		return []string{""}
	}
	return parts
}

// firstPhasePreProcess is optimization level 2 phase 1 (TS firstPhasePreProcess).
func (m *Minimatch) firstPhasePreProcess(globParts [][]string) [][]string {
	didSomething := true
	for didSomething {
		didSomething = false
		// iterate by index because we may append
		for pi := 0; pi < len(globParts); pi++ {
			parts := globParts[pi]
			gs := indexOf(parts, "**", 0)
			for gs != -1 {
				gss := gs
				for gss+1 < len(parts) && parts[gss+1] == "**" {
					gss++
				}
				if gss > gs {
					// splice(gs+1, gss-gs) keep one **
					parts = append(parts[:gs+1], parts[gss+1:]...)
					globParts[pi] = parts
				}

				var next, p, p2 string
				if gs+1 < len(parts) {
					next = parts[gs+1]
				}
				if gs+2 < len(parts) {
					p = parts[gs+2]
				}
				if gs+3 < len(parts) {
					p2 = parts[gs+3]
				}
				if next != ".." {
					gs = indexOf(parts, "**", gs+1)
					continue
				}
				if p == "" || p == "." || p == ".." || p2 == "" || p2 == "." || p2 == ".." {
					gs = indexOf(parts, "**", gs+1)
					continue
				}
				didSomething = true
				// remove ** at gs
				parts = append(parts[:gs], parts[gs+1:]...)
				other := append([]string(nil), parts...)
				// other[gs] = '**' — after splice, index gs is former next ("..")
				if gs < len(other) {
					other[gs] = "**"
				}
				globParts[pi] = parts
				globParts = append(globParts, other)
				gs--
				if gs < -1 {
					gs = -1
				}
				gs = indexOf(parts, "**", gs+1)
			}

			// squeeze . and ''
			if !m.PreserveMultipleSlashes {
				for i := 1; i < len(parts)-1; i++ {
					p := parts[i]
					if i == 1 && p == "" && parts[0] == "" {
						continue
					}
					if p == "." || p == "" {
						didSomething = true
						parts = append(parts[:i], parts[i+1:]...)
						globParts[pi] = parts
						i--
					}
				}
				if len(parts) == 2 && parts[0] == "." && (parts[1] == "." || parts[1] == "") {
					didSomething = true
					parts = parts[:1]
					globParts[pi] = parts
				}
			}

			// cancel p/..
			dd := indexOf(parts, "..", 1)
			for dd != -1 {
				var p string
				if dd > 0 {
					p = parts[dd-1]
				}
				if p != "" && p != "." && p != ".." && p != "**" {
					didSomething = true
					needDot := dd == 1 && dd+1 < len(parts) && parts[dd+1] == "**"
					var splin []string
					if needDot {
						splin = []string{"."}
					}
					parts = append(parts[:dd-1], append(splin, parts[dd+1:]...)...)
					if len(parts) == 0 {
						parts = []string{""}
					}
					globParts[pi] = parts
					dd -= 2
					if dd < 0 {
						dd = 0
					}
				}
				dd = indexOf(parts, "..", dd+1)
			}
		}
	}
	return globParts
}

// secondPhasePreProcess dedupes equivalent patterns (TS secondPhasePreProcess).
func (m *Minimatch) secondPhasePreProcess(globParts [][]string) [][]string {
	for i := 0; i < len(globParts)-1; i++ {
		for j := i + 1; j < len(globParts); j++ {
			matched := m.partsMatch(globParts[i], globParts[j], !m.PreserveMultipleSlashes)
			if matched != nil {
				globParts[i] = []string{}
				globParts[j] = matched
				break
			}
		}
	}
	out := make([][]string, 0, len(globParts))
	for _, gs := range globParts {
		if len(gs) > 0 {
			out = append(out, gs)
		}
	}
	return out
}

// partsMatch merges two pattern rows if they are compatible (TS partsMatch).
// Returns nil if no match (TS false).
func (m *Minimatch) partsMatch(a, b []string, emptyGSMatch bool) []string {
	ai, bi := 0, 0
	var result []string
	which := ""
	for ai < len(a) && bi < len(b) {
		if a[ai] == b[bi] {
			if which == "b" {
				result = append(result, b[bi])
			} else {
				result = append(result, a[ai])
			}
			ai++
			bi++
		} else if emptyGSMatch && a[ai] == "**" && ai+1 < len(a) && b[bi] == a[ai+1] {
			result = append(result, a[ai])
			ai++
		} else if emptyGSMatch && b[bi] == "**" && bi+1 < len(b) && a[ai] == b[bi+1] {
			result = append(result, b[bi])
			bi++
		} else if a[ai] == "*" && b[bi] != "" &&
			(m.Options.Dot || !strings.HasPrefix(b[bi], ".")) &&
			b[bi] != "**" {
			if which == "b" {
				return nil
			}
			which = "a"
			result = append(result, a[ai])
			ai++
			bi++
		} else if b[bi] == "*" && a[ai] != "" &&
			(m.Options.Dot || !strings.HasPrefix(a[ai], ".")) &&
			a[ai] != "**" {
			if which == "a" {
				return nil
			}
			which = "b"
			result = append(result, b[bi])
			ai++
			bi++
		} else {
			return nil
		}
	}
	if len(a) == len(b) {
		return result
	}
	return nil
}

func indexOf(parts []string, s string, from int) int {
	if from < 0 {
		from = 0
	}
	for i := from; i < len(parts); i++ {
		if parts[i] == s {
			return i
		}
	}
	return -1
}

func uniquePreserveOrder(ss []string) []string {
	seen := make(map[string]struct{}, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
