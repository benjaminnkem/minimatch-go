package class

import (
	"strings"
	"unicode/utf8"
)

// ParseClassResult is the outcome of ParseClass.
//
// Corresponds to TypeScript ParseClassResult:
//
//	[src, uFlag, consumed, hasMagic]
type ParseClassResult struct {
	// Src is regular-expression source for the class (or a literal character,
	// or "$.", or empty if not a valid class).
	Src string
	// UFlag is true when the result needs the Unicode regexp flag (/u in JS).
	UFlag bool
	// Consumed is how many code units (JS string length / UTF-16 units for
	// BMP-heavy globs; we count bytes in Go for ASCII/magic which matches)
	// of the pattern were consumed. 0 means "not a class".
	//
	// For behavioural parity with TypeScript, Consumed is the number of
	// UTF-16 code units from position through the closing ']', matching
	// pattern.length arithmetic in JS. For typical ASCII glob patterns this
	// equals the byte length in Go.
	Consumed int
	// HasMagic is false for single-literal classes like [_] (escape form).
	HasMagic bool
}

// posixClass maps a POSIX class spelling to (RE fragment, needs u flag, negated).
// TypeScript posixClasses in brace-expressions.ts.
type posixClass struct {
	re      string
	uFlag   bool
	negated bool // graph/print-style: stored in negs with inverted sense
}

// Order matters only for startsWith checks; longer names are unique prefixes.
var posixClasses = []struct {
	name string
	posixClass
}{
	{"[:alnum:]", posixClass{`\p{L}\p{Nl}\p{Nd}`, true, false}},
	{"[:alpha:]", posixClass{`\p{L}\p{Nl}`, true, false}},
	{"[:ascii:]", posixClass{`\x00-\x7f`, false, false}},
	{"[:blank:]", posixClass{`\p{Zs}\t`, true, false}},
	{"[:cntrl:]", posixClass{`\p{Cc}`, true, false}},
	{"[:digit:]", posixClass{`\p{Nd}`, true, false}},
	// graph: negated — everything that is NOT Z or C
	{"[:graph:]", posixClass{`\p{Z}\p{C}`, true, true}},
	{"[:lower:]", posixClass{`\p{Ll}`, true, false}},
	// print: TypeScript lists as ['\\p{C}', true] without n?:boolean true in the
	// third slot — wait, looking at source:
	// '[:print:]': ['\\p{C}', true],  — only 2 elements, not negated in the map!
	// But graph has true as third. print is just \p{C} with u flag in ranges.
	// Actually for print: ['\\p{C}', true] means re=\p{C}, u=true, n=undefined/false
	// So print is positive \p{C}? That seems wrong for POSIX print...
	// Trust the TypeScript map exactly:
	{"[:print:]", posixClass{`\p{C}`, true, false}},
	{"[:punct:]", posixClass{`\p{P}`, true, false}},
	{"[:space:]", posixClass{`\p{Z}\t\r\n\v\f`, true, false}},
	{"[:upper:]", posixClass{`\p{Lu}`, true, false}},
	{"[:word:]", posixClass{`\p{L}\p{Nl}\p{Nd}\p{Pc}`, true, false}},
	{"[:xdigit:]", posixClass{`A-Fa-f0-9`, false, false}},
}

// ParseClass parses a glob character class starting at position in pattern.
//
// Corresponds to TypeScript parseClass(glob, position) in brace-expressions.ts.
// position must point at '['. The return value describes an equivalent regexp
// fragment, whether the Unicode flag is needed, how much of pattern was
// consumed, and whether the class is "magic".
//
// Edge behaviours preserved from the reference:
//   - out-of-order ranges are dropped
//   - empty/impossible classes become "$.' (never matches) and consume the rest
//   - unclosed classes return Consumed 0 (not a class)
//   - single-character classes like [_] are non-magic literals
//   - POSIX classes map to Unicode properties
//
// Does not match paths; pure parsing of a class substring.
func ParseClass(pattern string, position int) (ParseClassResult, error) {
	if position < 0 || position >= len(pattern) || pattern[position] != '[' {
		return ParseClassResult{}, ErrNotInClass
	}
	return parseClass(pattern, position), nil
}

// ErrNotInClass matches the defensive TypeScript throw when not at '['.
var ErrNotInClass = errParseClass("not in a brace expression")

type errParseClass string

func (e errParseClass) Error() string { return string(e) }

func parseClass(glob string, pos int) ParseClassResult {
	var ranges, negs []string

	i := pos + 1
	sawStart := false
	uflag := false
	escaping := false
	negate := false
	endPos := pos
	rangeStart := ""

	for i < len(glob) {
		// Use UTF-8 runes for character comparison; magic and ranges in tests
		// are BMP/ASCII. For multi-byte runes, JS uses UTF-16 units — rare in globs.
		c, cSize := decodeAt(glob, i)

		if (c == "!" || c == "^") && i == pos+1 {
			negate = true
			i += cSize
			continue
		}

		if c == "]" && sawStart && !escaping {
			endPos = i + cSize
			break
		}

		sawStart = true

		if c == `\` {
			if !escaping {
				escaping = true
				i += cSize
				continue
			}
			// escaped \ — fall through as normal char
		}

		if c == "[" && !escaping {
			matchedPosix := false
			for _, pc := range posixClasses {
				if strings.HasPrefix(glob[i:], pc.name) {
					// invalid: rangeStart then posix — poison
					if rangeStart != "" {
						return ParseClassResult{
							Src:      "$.",
							UFlag:    false,
							Consumed: utf16Len(glob[pos:]),
							HasMagic: true,
						}
					}
					i += len(pc.name)
					if pc.negated {
						negs = append(negs, pc.re)
					} else {
						ranges = append(ranges, pc.re)
					}
					uflag = uflag || pc.uFlag
					matchedPosix = true
					break
				}
			}
			if matchedPosix {
				escaping = false
				continue
			}
		}

		// normal character
		escaping = false
		if rangeStart != "" {
			// JS: c > rangeStart string compare (UTF-16 code unit order for BMP)
			if c > rangeStart {
				ranges = append(ranges, braceEscape(rangeStart)+"-"+braceEscape(c))
			} else if c == rangeStart {
				ranges = append(ranges, braceEscape(c))
			}
			// else drop out-of-order range
			rangeStart = ""
			i += cSize
			continue
		}

		// range start? look ahead for -] or -
		// TypeScript: startsWith('-]', i+1) then i += 2 from current i (the
		// character c), leaving ']' for the next loop iteration to close.
		// startsWith('-', i+1) then i += 2 (c and '-'), next char is range end.
		rest := glob[i+cSize:]
		if strings.HasPrefix(rest, "-]") {
			ranges = append(ranges, braceEscape(c+"-"))
			// Advance past c and '-'; only (not ']').
			i += cSize + 1
			continue
		}
		if strings.HasPrefix(rest, "-") {
			rangeStart = c
			i += cSize + 1 // past c and '-'
			continue
		}

		ranges = append(ranges, braceEscape(c))
		i += cSize
	}

	if endPos < i {
		// unclosed class
		return ParseClassResult{Src: "", UFlag: false, Consumed: 0, HasMagic: false}
	}

	if len(ranges) == 0 && len(negs) == 0 {
		return ParseClassResult{
			Src:      "$.",
			UFlag:    false,
			Consumed: utf16Len(glob[pos:]),
			HasMagic: true,
		}
	}

	// single literal character class → non-magic
	if len(negs) == 0 && len(ranges) == 1 && isSingleEscapedOrChar(ranges[0]) && !negate {
		r := ranges[0]
		if len(r) == 2 && r[0] == '\\' {
			r = r[1:]
		}
		return ParseClassResult{
			Src:      regexpEscape(r),
			UFlag:    false,
			Consumed: utf16Len(glob[pos:endPos]),
			HasMagic: false,
		}
	}

	sranges := "["
	if negate {
		sranges += "^"
	}
	sranges += strings.Join(ranges, "") + "]"

	snegs := "["
	if !negate {
		snegs += "^"
	}
	snegs += strings.Join(negs, "") + "]"

	var comb string
	switch {
	case len(ranges) > 0 && len(negs) > 0:
		comb = "(" + sranges + "|" + snegs + ")"
	case len(ranges) > 0:
		comb = sranges
	default:
		comb = snegs
	}

	return ParseClassResult{
		Src:      comb,
		UFlag:    uflag,
		Consumed: utf16Len(glob[pos:endPos]),
		HasMagic: true,
	}
}

func decodeAt(s string, i int) (char string, size int) {
	if i >= len(s) {
		return "", 0
	}
	r, size := utf8.DecodeRuneInString(s[i:])
	if r == utf8.RuneError && size == 1 {
		return s[i : i+1], 1
	}
	return string(r), size
}

// braceEscape escapes [ \ ] - inside character classes.
// TypeScript: s.replace(/[[\]\\-]/g, '\\$&')
func braceEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '[', ']', '\\', '-':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// regexpEscape escapes regexp metacharacters.
// TypeScript: s.replace(/[-[\]{}()*+?.,\\^$|#\s]/g, '\\$&')
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

// isSingleEscapedOrChar matches /^\\?.$/ — one char or backslash+char.
func isSingleEscapedOrChar(s string) bool {
	if s == "" {
		return false
	}
	if s[0] == '\\' {
		// \ + one more character (rune)
		_, size := utf8.DecodeRuneInString(s[1:])
		return size > 0 && 1+size == len(s)
	}
	_, size := utf8.DecodeRuneInString(s)
	return size == len(s)
}

func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
	}
	return n
}
