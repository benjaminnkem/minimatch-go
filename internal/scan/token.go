package scan

// TokenKind is the kind of a lexical token produced by Scan.
//
// These kinds correspond to what TypeScript AST.#parseAST recognizes while
// walking a path-segment string. They are structural / lexical only:
// leaf glob magic such as standalone "*" or "?" stays inside TokenText and is
// not re-tokenized here (that happens later, in the equivalent of #parseGlob).
type TokenKind int

const (
	// TokenText is a run of characters that are not extglob structure.
	//
	// In TypeScript #parseAST this is the "acc" string: escapes, character
	// classes, "*", "?", ordinary letters, and even "|" or ")" when not
	// inside an extglob are all accumulated as text.
	//
	// Text holds the raw substring, including backslashes and "["..."]"
	// class spans kept opaque (the class is not parsed into ranges here).
	TokenText TokenKind = iota

	// TokenExtglobOpen is an extglob introducer: one of ! ? + * @ followed
	// immediately by '('.
	//
	// TypeScript: isExtglobType(c) && str.charAt(i) === '(' with noext off
	// and depth limits allowing recursion.
	//
	// Text is the single type character ("!", "?", "+", "*", or "@").
	// The token span covers both the type character and the '('.
	TokenExtglobOpen

	// TokenPipe is "|" separating extglob alternatives.
	//
	// TypeScript: only recognized while parsing inside an extglob
	// (ast.type !== null). At root level, "|" is TokenText.
	TokenPipe

	// TokenExtglobClose is ")" closing the current extglob.
	//
	// TypeScript: only recognized inside an extglob. At root level, ")" is
	// TokenText.
	TokenExtglobClose
)

// String returns a stable name for diagnostics and tests.
func (k TokenKind) String() string {
	switch k {
	case TokenText:
		return "Text"
	case TokenExtglobOpen:
		return "ExtglobOpen"
	case TokenPipe:
		return "Pipe"
	case TokenExtglobClose:
		return "ExtglobClose"
	default:
		return "TokenKind(" + itoa(int(k)) + ")"
	}
}

// Token is one lexical unit of a path-segment pattern.
//
// Start and End are byte offsets into the original segment string such that
// segment[Start:End] is the raw lexeme (except that TokenExtglobOpen.Text is
// only the type character, while Start:End still spans type + '(').
type Token struct {
	Kind  TokenKind
	Text  string
	Start int
	End   int
}

// ExtglobType is a TypeScript ExtglobType: '!' | '?' | '+' | '*' | '@'.
type ExtglobType byte

// Extglob type characters recognized by isExtglobType in ast.ts.
const (
	ExtglobNegate   ExtglobType = '!' // !(...)
	ExtglobOptional ExtglobType = '?' // ?(...)
	ExtglobPlus     ExtglobType = '+' // +(...)
	ExtglobStar     ExtglobType = '*' // *(...)
	ExtglobOne      ExtglobType = '@' // @(...)
)

// IsExtglobType reports whether c is a valid extglob type character.
//
// TypeScript: types = new Set(['!', '?', '+', '*', '@']); isExtglobType(c)
func IsExtglobType(c byte) bool {
	switch ExtglobType(c) {
	case ExtglobNegate, ExtglobOptional, ExtglobPlus, ExtglobStar, ExtglobOne:
		return true
	default:
		return false
	}
}

// itoa avoids strconv for a tiny helper used only in String().
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
