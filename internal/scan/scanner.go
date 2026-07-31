package scan

// Scan tokenizes a single path-segment pattern string.
//
// This is the lexical counterpart of TypeScript AST.#parseAST: it walks the
// string character-by-character, tracks escape state and character-class
// regions, and recognizes extglob structure. It does not build an AST, does
// not expand braces, does not split on '/', and does not match paths.
//
// The segment is typically one slash-separated piece of a full pattern
// (TypeScript parses each piece after slashSplit). Passing a string that
// contains '/' is allowed; '/' is ordinary TokenText (no path splitting).
//
// Options that affect scanning:
//
//   - NoExt: never emit TokenExtglobOpen; "*(" etc. stay in TokenText
//   - MaxExtglobRecursion: depth at which nested extglobs stop being
//     recognized (default DefaultMaxExtglobRecursion), matching
//     maxExtglobRecursion ?? 2 and the adoption-aware depth rule from
//     #parseAST / #canAdoptType
//
// Other Options fields are ignored by Scan.
//
// Unfinished extglobs (opening "*(" without a closing ")") produce an open
// without a matching TokenExtglobClose. TypeScript demotes those to literal
// text when building the AST; demotion is not applied here.
// Scan tokenizes a path-segment pattern.
// noExt disables extglobs; maxDepth is maxExtglobRecursion (use 2 for default).
func Scan(segment string, noExt bool, maxDepth int) []Token {
	s := &scanner{
		src:      segment,
		noExt:    noExt,
		maxDepth: maxDepth,
	}
	s.scanRoot()
	return s.tokens
}

type scanner struct {
	src      string
	noExt    bool
	maxDepth int
	tokens   []Token
	i        int // next byte index (magic syntax is ASCII)
}

func (s *scanner) scanRoot() {
	// TypeScript: #parseAST(str, rootAST, 0, opt, 0) with root type === null.
	s.scanOutsideExtglob(0, 0)
}

// scanOutsideExtglob implements the ast.type === null branch of #parseAST.
//
// parent is 0 at true root. When non-zero, we are scanning the body of one
// extglob alternative (a nested type-null AST). In that case '|' and ')'
// end this level and are left for the caller to consume.
//
// extDepth is the TypeScript extDepth parameter at this call.
func (s *scanner) scanOutsideExtglob(parent ExtglobType, extDepth int) {
	textStart := s.i
	escaping := false
	inClass := false
	// classInnerStart: byte index of first character after '['.
	// Matches TypeScript braceStart = i after consuming '[' (i already advanced).
	classInnerStart := 0
	classNeg := false
	sawStart := false

	flushText := func() {
		if s.i > textStart {
			s.tokens = append(s.tokens, Token{
				Kind:  TokenText,
				Text:  s.src[textStart:s.i],
				Start: textStart,
				End:   s.i,
			})
		}
		textStart = s.i
	}

	for s.i < len(s.src) {
		c := s.src[s.i]

		if escaping || c == '\\' {
			escaping = !escaping
			// Any consumed char counts as class content start if in class.
			if inClass && !escaping {
				// We just finished an escape pair when escaping became false;
				// the escaped char is "content". When escaping became true we
				// only saw '\'. TS sets sawStart = true after the negate check
				// for every iteration that continues into normal processing.
				// Mirror: once we take the escape path inside a class, content began.
			}
			if inClass {
				sawStart = true
			}
			s.i++
			continue
		}

		if inClass {
			// Position of c relative to first char inside '[' (0-based).
			// TS: i is already incremented past c when comparing to braceStart+1,
			// with braceStart = index of first inner char. After reading c at
			// braceStart, i === braceStart+1 for the first inner char.
			// Here s.i is still at c; first inner char has s.i == classInnerStart.
			atFirstInner := s.i == classInnerStart
			if atFirstInner && (c == '!' || c == '^') {
				classNeg = true
				s.i++
				// TS continues without setting sawStart on this iteration.
				continue
			}
			if c == ']' && sawStart {
				// Close the class. TS also rejects closing when
				// i === braceStart+2 && braceNeg — i.e. the first content
				// after a negate marker is ']', meaning "[!]" / "[^]" with
				// empty body is not closed by that ']'.
				// After negate, classInnerStart points at '!'/'^'; next char
				// has s.i == classInnerStart+1. If that char is ']' and
				// classNeg, TS: i === braceStart+2 && braceNeg → do not close.
				if !(classNeg && s.i == classInnerStart+1) {
					inClass = false
					sawStart = false
					classNeg = false
					s.i++
					continue
				}
			}
			sawStart = true
			s.i++
			continue
		}

		if c == '[' {
			inClass = true
			s.i++
			classInnerStart = s.i
			classNeg = false
			sawStart = false
			continue
		}

		// Extglob open: type char + '(' — TypeScript doRecurse.
		if !s.noExt && IsExtglobType(c) && s.i+1 < len(s.src) && s.src[s.i+1] == '(' {
			canAdopt := parent != 0 && canAdoptType(parent, ExtglobType(c))
			// Root / non-adopting: require extDepth <= maxDepth.
			// Inside adoptable parent: also allow when extDepth > maxDepth if canAdopt
			// (TS: extDepth <= maxDepth || ast.#canAdoptType(c)).
			if extDepth <= s.maxDepth || canAdopt {
				flushText()
				typ := ExtglobType(c)
				openStart := s.i
				s.i += 2 // consume type and '('
				s.tokens = append(s.tokens, Token{
					Kind:  TokenExtglobOpen,
					Text:  string(c),
					Start: openStart,
					End:   s.i,
				})
				depthAdd := 1
				if canAdopt {
					depthAdd = 0
				}
				s.scanExtglob(typ, extDepth+depthAdd)
				textStart = s.i
				continue
			}
		}

		// Inside an alternative body (parent != 0), '|' and ')' end this text level.
		if parent != 0 && (c == '|' || c == ')') {
			flushText()
			return
		}

		s.i++
	}

	flushText()
}

// scanExtglob implements the ast.type !== null branch of #parseAST.
// pos in TypeScript is at '('; we enter with s.i already past '('.
func (s *scanner) scanExtglob(typ ExtglobType, extDepth int) {
	for {
		// Scan one alternative body (nested type-null level).
		s.scanOutsideExtglob(typ, extDepth)

		if s.i >= len(s.src) {
			// Unfinished extglob: no closing ')'.
			// TypeScript demotes the AST node; we leave tokens as emitted.
			return
		}

		c := s.src[s.i]
		if c == '|' {
			pipeStart := s.i
			s.i++
			s.tokens = append(s.tokens, Token{
				Kind:  TokenPipe,
				Text:  "|",
				Start: pipeStart,
				End:   s.i,
			})
			continue // next alternative
		}
		if c == ')' {
			closeStart := s.i
			s.i++
			s.tokens = append(s.tokens, Token{
				Kind:  TokenExtglobClose,
				Text:  ")",
				Start: closeStart,
				End:   s.i,
			})
			return
		}

		// scanOutsideExtglob returned without '|' or ')' only at EOF,
		// which is handled above. Defensive: stop.
		return
	}
}

// canAdoptType reports whether a parent extglob type may adopt a nested
// child type for the purpose of maxExtglobRecursion depth accounting.
//
// TypeScript: #canAdoptType(c, adoptionAnyMap) — used in the doRecurse
// condition and for depthAdd = 0 when adopting.
//
// This is not full adopt/usurp flattening (that is AST work); only the type
// pairs that affect whether a nested "*(" is still recognized past max depth.
func canAdoptType(parent, child ExtglobType) bool {
	// adoptionAnyMap from ast.ts
	switch parent {
	case ExtglobNegate:
		return child == ExtglobOptional || child == ExtglobOne
	case ExtglobOptional:
		return child == ExtglobOptional || child == ExtglobOne
	case ExtglobOne:
		return child == ExtglobOptional || child == ExtglobOne
	case ExtglobStar:
		return child == ExtglobStar || child == ExtglobPlus ||
			child == ExtglobOptional || child == ExtglobOne
	case ExtglobPlus:
		return child == ExtglobPlus || child == ExtglobOne ||
			child == ExtglobOptional || child == ExtglobStar
	default:
		return false
	}
}
