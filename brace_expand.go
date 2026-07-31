package minimatch

import (
	"strconv"
	"strings"
)

// Expansion limits matching brace-expansion@5 (used by minimatch).
const (
	// ExpansionMax is the default cap on the number of expansion results.
	// TypeScript brace-expansion: EXPANSION_MAX = 100_000
	// Also Options DefaultBraceExpandMax.
	ExpansionMax = 100_000

	// ExpansionMaxLength caps total characters held across all results in
	// the accumulator (CVE-2026-14257 mitigation).
	// TypeScript: EXPANSION_MAX_LENGTH = 4_000_000
	ExpansionMaxLength = 4_000_000
)

// Sentinels for escaped brace metacharacters during expand.
// Fixed tokens (brace-expansion uses Math.random()); null bytes keep them
// out of normal glob text.
const (
	escSlash  = "\x00SLASH\x00"
	escOpen   = "\x00OPEN\x00"
	escClose  = "\x00CLOSE\x00"
	escComma  = "\x00COMMA\x00"
	escPeriod = "\x00PERIOD\x00"
)

// BraceExpand performs bash-style brace expansion on pattern.
//
// Corresponds to TypeScript minimatch.braceExpand:
//
//  1. ValidatePattern(pattern)
//  2. If NoBrace or no simple `{…}` group exists, return []string{pattern}
//  3. Otherwise expand via the brace-expansion algorithm with
//     max = EffectiveBraceExpandMax()
//
// Does not match paths, parse globs, or build an AST.
//
// Examples (same as the reference):
//
//	BraceExpand("a{b,c}d", Options{}) → ["abd", "acd"]
//	BraceExpand("a{1..3}b", Options{}) → ["a1b", "a2b", "a3b"]
//	BraceExpand("a{b}c", Options{}) → ["a{b}c"]  // not a valid set
func BraceExpand(pattern string, opts Options) ([]string, error) {
	if err := ValidatePattern(pattern); err != nil {
		return nil, err
	}
	// TypeScript: options.nobrace || !/\{(?:(?!\{).)*\}/.test(pattern)
	if opts.NoBrace || !hasSimpleBraceGroup(pattern) {
		return []string{pattern}, nil
	}
	max := opts.EffectiveBraceExpandMax()
	return expand(pattern, max, ExpansionMaxLength), nil
}

// hasSimpleBraceGroup reports whether pattern contains a `{…}` group whose
// body does not contain `{`, matching the ReDoS-safe pre-check regex
// /\{(?:(?!\{).)*\}/ used by minimatch.braceExpand.
func hasSimpleBraceGroup(pattern string) bool {
	for i := 0; i < len(pattern); i++ {
		if pattern[i] != '{' {
			continue
		}
		j := i + 1
		for j < len(pattern) && pattern[j] != '{' && pattern[j] != '}' {
			j++
		}
		if j < len(pattern) && pattern[j] == '}' {
			return true
		}
	}
	return false
}

// expand is brace-expansion's expand().
func expand(str string, max, maxLength int) []string {
	if str == "" {
		return []string{}
	}
	// Bash: leading {} at top level is preserved as literal braces.
	if strings.HasPrefix(str, "{}") {
		str = `\{\}` + str[2:]
	}
	out := expand_(escapeBraces(str), max, maxLength, true)
	for i := range out {
		out[i] = unescapeBraces(out[i])
	}
	return out
}

func escapeBraces(str string) string {
	// Sequential replaces matching brace-expansion's chained .replace calls:
	// \\ then \{ \} \, \.
	str = strings.ReplaceAll(str, `\\`, escSlash)
	str = strings.ReplaceAll(str, `\{`, escOpen)
	str = strings.ReplaceAll(str, `\}`, escClose)
	str = strings.ReplaceAll(str, `\,`, escComma)
	str = strings.ReplaceAll(str, `\.`, escPeriod)
	return str
}

func unescapeBraces(str string) string {
	str = strings.ReplaceAll(str, escSlash, `\`)
	str = strings.ReplaceAll(str, escOpen, `{`)
	str = strings.ReplaceAll(str, escClose, `}`)
	str = strings.ReplaceAll(str, escComma, `,`)
	str = strings.ReplaceAll(str, escPeriod, `.`)
	return str
}

// parseCommaParts splits on commas but keeps nested brace groups intact.
// TypeScript parseCommaParts.
func parseCommaParts(str string) []string {
	if str == "" {
		return []string{""}
	}
	pre, body, post, ok := balanced("{", "}", str)
	if !ok {
		return strings.Split(str, ",")
	}
	p := strings.Split(pre, ",")
	p[len(p)-1] += "{" + body + "}"
	postParts := parseCommaParts(post)
	if len(post) > 0 {
		p[len(p)-1] += postParts[0]
		p = append(p, postParts[1:]...)
	}
	return p
}

func embrace(str string) string {
	return "{" + str + "}"
}

func isPadded(el string) bool {
	// /^-?0\d/
	if strings.HasPrefix(el, "-") {
		return len(el) >= 3 && el[1] == '0' && el[2] >= '0' && el[2] <= '9'
	}
	return len(el) >= 2 && el[0] == '0' && el[1] >= '0' && el[1] <= '9'
}

// numeric mirrors brace-expansion numeric(): parseInt if numeric, else char code.
func numeric(str string) int {
	// JS: !isNaN(str) ? parseInt(str, 10) : str.charCodeAt(0)
	// For sequence bodies the regex guarantees integers or single letters.
	if n, err := strconv.Atoi(str); err == nil {
		return n
	}
	// Number(str) path: empty string is 0 in JS isNaN('') === false, parseInt('',10) is NaN.
	// Fall back to first byte code unit (alpha sequences are single ASCII letters).
	if str == "" {
		return 0
	}
	return int(str[0])
}

func expandSequence(body string, isAlphaSequence bool, max int) []string {
	n := strings.Split(body, "..")
	if len(n) < 2 {
		return nil
	}
	x := numeric(n[0])
	y := numeric(n[1])
	width := len(n[0])
	if len(n[1]) > width {
		width = len(n[1])
	}
	incr := 1
	if len(n) == 3 {
		step := numeric(n[2])
		if step < 0 {
			step = -step
		}
		if step < 1 {
			step = 1
		}
		incr = step
	}
	test := func(i, y int) bool { return i <= y }
	reverse := y < x
	if reverse {
		incr = -incr
		test = func(i, y int) bool { return i >= y }
	}
	pad := false
	for _, el := range n {
		if isPadded(el) {
			pad = true
			break
		}
	}
	var N []string
	for i := x; test(i, y) && len(N) < max; i += incr {
		var c string
		if isAlphaSequence {
			c = string(rune(i))
			if c == `\` {
				c = ""
			}
		} else {
			c = strconv.Itoa(i)
			if pad {
				need := width - len(c)
				if need > 0 {
					z := strings.Repeat("0", need)
					if i < 0 {
						c = "-" + z + c[1:]
					} else {
						c = z + c
					}
				}
			}
		}
		N = append(N, c)
	}
	return N
}

// combine builds acc[a]+pre+values[v] for every combination, respecting max
// and maxLength. TypeScript combine().
func combine(acc []string, pre string, values []string, max, maxLength int, dropEmpties bool) []string {
	out := make([]string, 0, len(acc)*len(values))
	length := 0
	for a := 0; a < len(acc); a++ {
		for v := 0; v < len(values); v++ {
			if len(out) >= max {
				return out
			}
			expansion := acc[a] + pre + values[v]
			if dropEmpties && expansion == "" {
				continue
			}
			if length+len(expansion) > maxLength {
				return out
			}
			out = append(out, expansion)
			length += len(expansion)
		}
	}
	return out
}

// expand_ is the core iterative brace-expansion algorithm.
func expand_(str string, max, maxLength int, isTop bool) []string {
	acc := []string{""}
	dropEmpties := false
	firstGroup := true

	for {
		pre, body, post, ok := balanced("{", "}", str)
		if !ok {
			return combine(acc, str, []string{""}, max, maxLength, dropEmpties)
		}

		// ${...} — dollar before brace: skip expansion of this group.
		if strings.HasSuffix(pre, "$") {
			acc = combine(acc, pre+"{"+body+"}", []string{""}, max, maxLength, dropEmpties && len(post) == 0)
			firstGroup = false
			if len(post) == 0 {
				break
			}
			str = post
			continue
		}

		isNumericSequence := isNumericSequenceBody(body)
		isAlphaSequence := isAlphaSequenceBody(body)
		isSequence := isNumericSequence || isAlphaSequence
		isOptions := strings.Contains(body, ",")

		if !isSequence && !isOptions {
			// {a},b} — rewrite and retry (TS: m.post.match(/,(?!,).*\}/))
			if postHasCommaOption(post) {
				str = pre + "{" + body + escClose + post
				isTop = true
				continue
			}
			// Nothing expands: rest is literal.
			return combine(acc, pre+"{"+body+"}"+post, []string{""}, max, maxLength, dropEmpties)
		}

		if firstGroup {
			dropEmpties = isTop && !isSequence
			firstGroup = false
		}

		var values []string
		if isSequence {
			values = expandSequence(body, isAlphaSequence, max)
		} else {
			n := parseCommaParts(body)
			if len(n) == 1 {
				// x{{a,b}}y ⇒ x{a}y x{b}y
				inner := expand_(n[0], max, maxLength, false)
				for i := range inner {
					inner[i] = embrace(inner[i])
				}
				n = inner
				if len(n) == 1 {
					acc = combine(acc, pre+n[0], []string{""}, max, maxLength, dropEmpties && len(post) == 0)
					if len(post) == 0 {
						break
					}
					str = post
					continue
				}
			}
			values = nil
			for j := 0; j < len(n); j++ {
				values = append(values, expand_(n[j], max, maxLength, false)...)
			}
		}

		acc = combine(acc, pre, values, max, maxLength, dropEmpties && len(post) == 0)
		if len(post) == 0 {
			break
		}
		str = post
	}
	return acc
}

// isNumericSequenceBody: /^-?\d+\.\.-?\d+(?:\.\.-?\d+)?$/
func isNumericSequenceBody(body string) bool {
	// Manual parse to avoid complex regex and match JS exactly.
	i := 0
	if i < len(body) && body[i] == '-' {
		i++
	}
	start := i
	for i < len(body) && body[i] >= '0' && body[i] <= '9' {
		i++
	}
	if i == start || i+2 > len(body) || body[i] != '.' || body[i+1] != '.' {
		return false
	}
	i += 2
	if i < len(body) && body[i] == '-' {
		i++
	}
	start = i
	for i < len(body) && body[i] >= '0' && body[i] <= '9' {
		i++
	}
	if i == start {
		return false
	}
	if i == len(body) {
		return true
	}
	if i+2 > len(body) || body[i] != '.' || body[i+1] != '.' {
		return false
	}
	i += 2
	if i < len(body) && body[i] == '-' {
		i++
	}
	start = i
	for i < len(body) && body[i] >= '0' && body[i] <= '9' {
		i++
	}
	return i > start && i == len(body)
}

// isAlphaSequenceBody: /^[a-zA-Z]\.\.[a-zA-Z](?:\.\.-?\d+)?$/
func isAlphaSequenceBody(body string) bool {
	if len(body) < 4 {
		return false
	}
	if !isASCIILetter(body[0]) || body[1] != '.' || body[2] != '.' || !isASCIILetter(body[3]) {
		return false
	}
	if len(body) == 4 {
		return true
	}
	// \.\.-?\d+
	if len(body) < 7 || body[4] != '.' || body[5] != '.' {
		return false
	}
	i := 6
	if i < len(body) && body[i] == '-' {
		i++
	}
	start := i
	for i < len(body) && body[i] >= '0' && body[i] <= '9' {
		i++
	}
	return i > start && i == len(body)
}

func isASCIILetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// postHasCommaOption matches /,(?!,).*\}/ on post — a comma that is not
// part of `,,` and is eventually followed by `}`.
//
// JS: m.post.match(/,(?!,).*\}/)
// Go RE2 lacks (?!,); implement equivalently: find `,` not followed by `,`,
// then later a `}`.
func postHasCommaOption(post string) bool {
	for i := 0; i < len(post); i++ {
		if post[i] != ',' {
			continue
		}
		if i+1 < len(post) && post[i+1] == ',' {
			continue // (?!,) fails
		}
		if strings.Contains(post[i+1:], "}") {
			return true
		}
	}
	return false
}
