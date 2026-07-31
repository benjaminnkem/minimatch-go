package ast

import (
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/tochison/minimatch/internal/class"
	"github.com/tochison/minimatch/internal/scan"
)

// Regexp fragments used when compiling path-segment patterns.
// TypeScript ast.ts: startNoDot, startNoTraversal, qmark, star, starNoEmpty.
const (
	reStartNoDot       = `(?!\.)`
	reStartNoTraversal = `(?!(?:^|/)\.\.?(?:$|/))`
	reQmark            = `[^/]`
	reStar             = reQmark + `*?`
	reStarNoEmpty      = reQmark + `+?`
)

// reSpecials are characters that need escaping when they appear after a
// glob backslash escape (TypeScript reSpecials Set).
func isReSpecial(c byte) bool {
	switch c {
	case '(', ')', '.', '*', '{', '}', '+', '?', '[', ']', '^', '$', '\\', '!':
		return true
	default:
		return false
	}
}

func isAddPatternStart(c byte) bool {
	return c == '[' || c == '.'
}

func isJustDots(s string) bool {
	return s == "." || s == ".."
}

// RegExpSource is the result of ToRegExpSource (TypeScript tuple return).
type RegExpSource struct {
	// Re is the regular-expression source for this node (may include
	// lookarounds for dots / negations).
	Re string
	// Body is the unescaped pattern body used for non-magic comparison.
	Body string
	// HasMagic is true when a RegExp is required to match.
	HasMagic bool
	// UFlag is true when Unicode property escapes are present (JS /u).
	UFlag bool
}

// MMPattern is a compiled path-segment pattern (TypeScript toMMPattern).
//
// If IsRE is false, match with exact equality to Literal (unescaped body).
// If IsRE is true, use RE (anchored ^…$ as in TypeScript).
type MMPattern struct {
	// Literal is the non-magic match string (TypeScript string return).
	Literal string
	// RE is the compiled regular expression (nil if literal).
	// Uses regexp2 so lookarounds from minimatch sources work.
	RE *regexp2.Regexp
	// Src is the unanchored source (_src in TypeScript).
	Src string
	// Glob is the reconstructed glob string (_glob in TypeScript).
	Glob string
	// IsRE is true when RE should be used instead of Literal.
	IsRE bool
}

// Match reports whether s matches this pattern (full segment).
func (p MMPattern) Match(s string) (bool, error) {
	if !p.IsRE {
		return s == p.Literal, nil
	}
	if p.RE == nil {
		return false, nil
	}
	return p.RE.MatchString(s)
}

// ToMMPattern compiles this AST (root) to a literal or regular expression.
//
// Corresponds to TypeScript AST.toMMPattern(). Calls Flatten and FillNegs
// via ToRegExpSource. Does not perform multi-segment path matching.
func (n *AST) ToMMPattern() (MMPattern, error) {
	if n == nil {
		return MMPattern{Literal: "", IsRE: false}, nil
	}
	if n.root != nil && n != n.root {
		return n.root.ToMMPattern()
	}
	glob := n.String()
	src := n.ToRegExpSource(nil)
	anyMagic := src.HasMagic || n.hasMagic ||
		(n.Options.NoCase && !n.Options.NoCaseMagicOnly &&
			strings.ToUpper(glob) != strings.ToLower(glob))
	if !anyMagic {
		return MMPattern{
			Literal: src.Body,
			Src:     src.Re,
			Glob:    glob,
			IsRE:    false,
		}, nil
	}
	flags := regexp2.None
	if n.Options.NoCase {
		flags |= regexp2.IgnoreCase
	}
	// regexp2 handles \p{} with Unicode option
	if src.UFlag {
		flags |= regexp2.Unicode
	}
	re, err := regexp2.Compile("^"+src.Re+"$", flags)
	if err != nil {
		return MMPattern{}, err
	}
	return MMPattern{
		RE:   re,
		Src:  src.Re,
		Glob: glob,
		IsRE: true,
	}, nil
}

// ToRegExpSource returns regexp source for this node.
//
// Corresponds to TypeScript AST.toRegExpSource(allowDot?).
// On the root, runs Flatten and FillNegs first.
//
// allowDot nil means use Options.Dot; non-nil overrides (extglob dual-body).
func (n *AST) ToRegExpSource(allowDot *bool) RegExpSource {
	if n == nil {
		return RegExpSource{}
	}
	dot := n.Options.Dot
	if allowDot != nil {
		dot = *allowDot
	}

	if n.root == n {
		// TypeScript toRegExpSource on root always flattens then fillNegs.
		n.Flatten()
		n.FillNegs()
	}

	if n.IsList() {
		return n.listToRegExpSource(allowDot, dot)
	}
	return n.extglobToRegExpSource(allowDot, dot)
}

func (n *AST) listToRegExpSource(allowDot *bool, dot bool) RegExpSource {
	noEmpty := n.isStart() && n.isEnd() && !n.hasNonStringPart()
	var b strings.Builder
	for _, p := range n.Parts {
		var part RegExpSource
		if p.Node != nil {
			part = p.Node.ToRegExpSource(allowDot)
		} else {
			part = parseGlob(p.Text, n.hasMagic, noEmpty)
		}
		n.hasMagic = n.hasMagic || part.HasMagic
		n.uFlag = n.uFlag || part.UFlag
		b.WriteString(part.Re)
	}
	src := b.String()

	start := ""
	if n.isStart() {
		if len(n.Parts) > 0 && n.Parts[0].Node == nil {
			dotTravAllowed := len(n.Parts) == 1 && isJustDots(n.Parts[0].Text)
			if !dotTravAllowed {
				needNoTrav := false
				needNoDot := false
				if len(src) > 0 {
					c0 := src[0]
					needNoTrav = (dot && isAddPatternStart(c0)) ||
						(strings.HasPrefix(src, `\.`) && len(src) > 2 && isAddPatternStart(src[2])) ||
						(strings.HasPrefix(src, `\.\.`) && len(src) > 4 && isAddPatternStart(src[4]))
					allowDotVal := false
					if allowDot != nil {
						allowDotVal = *allowDot
					}
					needNoDot = !dot && !allowDotVal && isAddPatternStart(c0)
				}
				if needNoTrav {
					start = reStartNoTraversal
				} else if needNoDot {
					start = reStartNoDot
				}
			}
		}
	}

	end := ""
	if n.isEnd() && n.root != nil && n.root.filledNegs &&
		n.parent != nil && n.parent.Type == scan.ExtglobNegate {
		end = `(?:$|\/)`
	}

	final := start + src + end
	return RegExpSource{
		Re:       final,
		Body:     unescape(src),
		HasMagic: n.hasMagic,
		UFlag:    n.uFlag,
	}
}

func (n *AST) hasNonStringPart() bool {
	for _, p := range n.Parts {
		if p.Node != nil {
			return true
		}
	}
	return false
}

func (n *AST) extglobToRegExpSource(allowDot *bool, dot bool) RegExpSource {
	repeated := n.Type == scan.ExtglobStar || n.Type == scan.ExtglobPlus
	start := `(?:`
	if n.Type == scan.ExtglobNegate {
		start = `(?:(?!(?:`
	}

	body := n.partsToRegExp(dot)

	if n.isStart() && n.isEnd() && body == "" && n.Type != scan.ExtglobNegate {
		// invalid empty extglob as whole path portion
		s := n.String()
		n.Parts = []ASTPart{{Text: s}}
		n.Type = 0
		n.hasMagic = false
		return RegExpSource{
			Re:       s,
			Body:     unescape(n.String()),
			HasMagic: false,
			UFlag:    false,
		}
	}

	bodyDotAllowed := ""
	allowDotVal := false
	if allowDot != nil {
		allowDotVal = *allowDot
	}
	// TypeScript: bodyDotAllowed when repeated && !allowDot && !dot.
	// The dual body enables patterns like *(?) to match "a.b". It also
	// creates deeply nested lookaround REs that hang in regexp2 (e.g.
	// *(*.json|!(*.js))). We still emit the dual form for fidelity when
	// the non-dot body is simple (no nested "!" lookarounds).
	if repeated && !allowDotVal && !dot {
		candidate := n.partsToRegExp(true)
		// Dual body is required for *(?) matching "a.b", but combined with
		// nested negative extglobs ("(?!(?:…)") it can hang regexp2 (ReDoS-like).
		// Skip dual only when a negative-extglob lookaround is present.
		if candidate != body && !strings.Contains(body, "(?!(?:") {
			bodyDotAllowed = candidate
		}
	}
	if bodyDotAllowed != "" {
		body = `(?:` + body + `)(?:` + bodyDotAllowed + `)*?`
	}

	var final string
	if n.Type == scan.ExtglobNegate && n.EmptyExt {
		prefix := ""
		if n.isStart() && !dot {
			prefix = reStartNoDot
		}
		final = prefix + reStarNoEmpty
	} else {
		var close string
		switch n.Type {
		case scan.ExtglobNegate:
			close = `))`
			if n.isStart() && !dot && !allowDotVal {
				close += reStartNoDot
			}
			close += reStar + `)`
		case scan.ExtglobOne:
			close = `)`
		case scan.ExtglobOptional:
			close = `)?`
		case scan.ExtglobPlus:
			if bodyDotAllowed != "" {
				close = `)`
			} else {
				close = `)+`
			}
		case scan.ExtglobStar:
			if bodyDotAllowed != "" {
				close = `)?`
			} else {
				close = `)*`
			}
		default:
			close = `)`
		}
		final = start + body + close
	}

	n.hasMagic = true
	return RegExpSource{
		Re:       final,
		Body:     unescape(body),
		HasMagic: true,
		UFlag:    n.uFlag,
	}
}

// partsToRegExp joins extglob alternatives (TypeScript #partsToRegExp).
func (n *AST) partsToRegExp(dot bool) string {
	ad := dot
	allow := &ad
	var parts []string
	for _, p := range n.Parts {
		if p.Node == nil {
			continue
		}
		src := p.Node.ToRegExpSource(allow)
		n.uFlag = n.uFlag || src.UFlag
		parts = append(parts, src.Re)
	}
	// filter: if isStart && isEnd, drop empty strings
	if n.isStart() && n.isEnd() {
		filtered := parts[:0]
		for _, p := range parts {
			if p != "" {
				filtered = append(filtered, p)
			}
		}
		parts = filtered
	}
	return strings.Join(parts, "|")
}

// parseGlob compiles a leaf glob text run (TypeScript AST.#parseGlob).
// Iteration is by UTF-8 rune so multi-byte characters (e.g. "å") stay one unit.
func parseGlob(glob string, hasMagic bool, noEmpty bool) RegExpSource {
	escaping := false
	var re strings.Builder
	uflag := false
	inStar := false
	runes := []rune(glob)

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if escaping {
			escaping = false
			if r < 128 && isReSpecial(byte(r)) {
				re.WriteByte('\\')
			}
			re.WriteRune(r)
			continue
		}
		if r == '*' {
			if inStar {
				continue
			}
			inStar = true
			if noEmpty && isOnlyStars(glob) {
				re.WriteString(reStarNoEmpty)
			} else {
				re.WriteString(reStar)
			}
			hasMagic = true
			continue
		}
		inStar = false

		if r == '\\' {
			if i == len(runes)-1 {
				re.WriteString(`\\`)
			} else {
				escaping = true
			}
			continue
		}
		if r == '[' {
			// parseClass uses byte indices into original string
			byteIdx := len(string(runes[:i]))
			pc := parseClassAt(glob, byteIdx)
			if pc.Consumed > 0 {
				re.WriteString(pc.Src)
				uflag = uflag || pc.UFlag
				// Advance rune index by consumed UTF-16 units ≈ runes for BMP
				consumed := string(runes[i:]) // from current
				// Consume pc.Consumed UTF-16 code units from glob[byteIdx:]
				adv := advanceRunesByUTF16(runes[i:], pc.Consumed)
				i += adv - 1
				hasMagic = hasMagic || pc.HasMagic
				_ = consumed
				continue
			}
		}
		if r == '?' {
			re.WriteString(reQmark)
			hasMagic = true
			continue
		}
		re.WriteString(regexpEscape(string(r)))
	}

	return RegExpSource{
		Re:       re.String(),
		Body:     unescape(glob),
		HasMagic: hasMagic,
		UFlag:    uflag,
	}
}

// advanceRunesByUTF16 returns how many runes cover n UTF-16 code units.
func advanceRunesByUTF16(runes []rune, utf16Units int) int {
	n := 0
	units := 0
	for n < len(runes) && units < utf16Units {
		r := runes[n]
		if r >= 0x10000 {
			units += 2
		} else {
			units++
		}
		n++
	}
	return n
}

func isOnlyStars(glob string) bool {
	if glob == "" {
		return false
	}
	for i := 0; i < len(glob); i++ {
		if glob[i] != '*' {
			return false
		}
	}
	return true
}

func parseClassAt(glob string, pos int) class.ParseClassResult {
	// Use unexported path via ParseClass
	r, err := class.ParseClass(glob, pos)
	if err != nil {
		return class.ParseClassResult{}
	}
	return r
}

func regexpEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '-', '[', ']', '{', '}', '(', ')', '*', '+', '?', '.',
			',', '\\', '^', '$', '|', '#', ' ', '\t', '\n', '\r', '\f', '\v':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
