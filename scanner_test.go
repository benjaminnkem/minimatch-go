package minimatch

import (
	"reflect"
	"testing"
)

func kinds(tokens []Token) []TokenKind {
	out := make([]TokenKind, len(tokens))
	for i, t := range tokens {
		out[i] = t.Kind
	}
	return out
}

func texts(tokens []Token) []string {
	out := make([]string, len(tokens))
	for i, t := range tokens {
		out[i] = t.Text
	}
	return out
}

func TestScanEmpty(t *testing.T) {
	toks := Scan("", Options{})
	if len(toks) != 0 {
		t.Fatalf("got %v", toks)
	}
}

func TestScanPlainLiteral(t *testing.T) {
	toks := Scan("hello", Options{})
	if len(toks) != 1 || toks[0].Kind != TokenText || toks[0].Text != "hello" {
		t.Fatalf("%+v", toks)
	}
	if toks[0].Start != 0 || toks[0].End != 5 {
		t.Fatalf("span %d:%d", toks[0].Start, toks[0].End)
	}
}

func TestScanGlobMagicStaysInText(t *testing.T) {
	// #parseAST does not tokenize * or ? at this stage — only *( extglob.
	toks := Scan("a*b?c", Options{})
	if len(toks) != 1 || toks[0].Kind != TokenText || toks[0].Text != "a*b?c" {
		t.Fatalf("%+v", toks)
	}
}

func TestScanSimpleExtglob(t *testing.T) {
	// foo*(a|b).js  — structural tokens like the AST shape
	toks := Scan("foo*(a|b).js", Options{})
	wantKinds := []TokenKind{
		TokenText, TokenExtglobOpen, TokenText, TokenPipe, TokenText, TokenExtglobClose, TokenText,
	}
	wantTexts := []string{"foo", "*", "a", "|", "b", ")", ".js"}
	if !reflect.DeepEqual(kinds(toks), wantKinds) {
		t.Fatalf("kinds=%v want %v tokens=%+v", kinds(toks), wantKinds, toks)
	}
	if !reflect.DeepEqual(texts(toks), wantTexts) {
		t.Fatalf("texts=%v want %v", texts(toks), wantTexts)
	}
	// ExtglobOpen spans type + '('
	open := toks[1]
	if open.End-open.Start != 2 || open.Text != "*" {
		t.Fatalf("open token %+v", open)
	}
}

func TestScanAllExtglobTypes(t *testing.T) {
	for _, typ := range []string{"!", "?", "+", "*", "@"} {
		seg := typ + "(x)"
		toks := Scan(seg, Options{})
		if len(toks) != 3 {
			t.Fatalf("%s: %+v", seg, toks)
		}
		if toks[0].Kind != TokenExtglobOpen || toks[0].Text != typ {
			t.Fatalf("%s open: %+v", seg, toks[0])
		}
		if toks[1].Kind != TokenText || toks[1].Text != "x" {
			t.Fatalf("%s body: %+v", seg, toks[1])
		}
		if toks[2].Kind != TokenExtglobClose {
			t.Fatalf("%s close: %+v", seg, toks[2])
		}
	}
}

func TestScanNoExtTreatsAsText(t *testing.T) {
	toks := Scan("*(a|b)", Options{NoExt: true})
	if len(toks) != 1 || toks[0].Kind != TokenText || toks[0].Text != "*(a|b)" {
		t.Fatalf("%+v", toks)
	}
}

func TestScanEscapedExtglobNotOpened(t *testing.T) {
	// \* ( is not an extglob start — backslash escape path
	toks := Scan(`\*(a)`, Options{})
	if len(toks) != 1 || toks[0].Kind != TokenText {
		t.Fatalf("%+v", toks)
	}
	if toks[0].Text != `\*(a)` {
		t.Fatalf("text %q", toks[0].Text)
	}
}

func TestScanCharClassOpaque(t *testing.T) {
	// "[!a-z]*(" should not open extglob inside / after class incorrectly.
	// Class [*(] keeps * ( opaque; no extglob.
	toks := Scan("[*(]", Options{})
	if len(toks) != 1 || toks[0].Text != "[*(]" {
		t.Fatalf("%+v", toks)
	}

	// After class, extglob still works: [a]*(b)
	toks = Scan("[a]*(b)", Options{})
	want := []TokenKind{TokenText, TokenExtglobOpen, TokenText, TokenExtglobClose}
	if !reflect.DeepEqual(kinds(toks), want) {
		t.Fatalf("%v %+v", kinds(toks), toks)
	}
	if toks[0].Text != "[a]" {
		t.Fatalf("class text %q", toks[0].Text)
	}
}

func TestScanEmptyClassAndNegation(t *testing.T) {
	// [] is not a closed class on first ] in #parseAST — both brackets stay text
	// and scanning continues. Actually: first ']', sawStart false, so not closed;
	// sawStart becomes true; then if end, still inClass. Entire "[]" is text.
	toks := Scan("[]", Options{})
	if len(toks) != 1 || toks[0].Text != "[]" {
		t.Fatalf("%+v", toks)
	}

	// []] — first ] after sawStart from... first ] doesn't close; sawStart true;
	// second ] closes. Still all text one token.
	toks = Scan("[]]", Options{})
	if len(toks) != 1 || toks[0].Text != "[]]" {
		t.Fatalf("%+v", toks)
	}
}

func TestScanNestedExtglob(t *testing.T) {
	// *(a|@(b|c))
	toks := Scan("*(a|@(b|c))", Options{})
	wantKinds := []TokenKind{
		TokenExtglobOpen, // *
		TokenText,        // a
		TokenPipe,
		TokenExtglobOpen, // @
		TokenText,        // b
		TokenPipe,
		TokenText, // c
		TokenExtglobClose,
		TokenExtglobClose,
	}
	if !reflect.DeepEqual(kinds(toks), wantKinds) {
		t.Fatalf("kinds=%v want %v\n%+v", kinds(toks), wantKinds, toks)
	}
	wantTexts := []string{"*", "a", "|", "@", "b", "|", "c", ")", ")"}
	if !reflect.DeepEqual(texts(toks), wantTexts) {
		t.Fatalf("texts=%v want %v", texts(toks), wantTexts)
	}
}

func TestScanPipeAndCloseAtRootAreText(t *testing.T) {
	toks := Scan("a|b)c", Options{})
	if len(toks) != 1 || toks[0].Text != "a|b)c" {
		t.Fatalf("%+v", toks)
	}
}

func TestScanUnfinishedExtglob(t *testing.T) {
	// *(a|b  — no close
	toks := Scan("*(a|b", Options{})
	wantKinds := []TokenKind{TokenExtglobOpen, TokenText, TokenPipe, TokenText}
	if !reflect.DeepEqual(kinds(toks), wantKinds) {
		t.Fatalf("kinds=%v want %v %+v", kinds(toks), wantKinds, toks)
	}
}

func TestScanMaxExtglobRecursion(t *testing.T) {
	// Default max depth 2. Nested opens that exceed depth become text.
	// Pattern: *(@(+(! (x)))) — need to count depth carefully.
	// Root depth 0: *(  → enter depth 1
	//   @( → enter depth 2
	//     +( → enter depth 3 — if max is 2, +( may not open unless adopt
	//
	// * can adopt +, so depthAdd=0 when parent is * and child is +...
	// Here parent of + is @, @ can adopt ? and @ only, not +.
	// So +( at depth 2: extDepth is 2, max 2, 2<=2 so still opens at depth 3.
	// At depth 3: !( : 3<=2 false, parent + can adopt ? and @ and * not ! — no open.
	//
	// Use maxDepth 0: only extDepth 0 can open; first *( has extDepth 0 <= 0, opens with depth 1.
	// Inside, next @( at extDepth 1: 1<=0 false, @ not adoptable from *? * can adopt @!
	// canAdopt(*, @) true → still opens with depthAdd 0, still depth 1.
	// So adoption keeps recognizing.
	//
	// maxDepth 0, pattern "*(x)" only: opens fine at depth 0.
	toks := Scan("*(x)", Options{MaxExtglobRecursion: Int(0)})
	if kinds(toks)[0] != TokenExtglobOpen {
		t.Fatalf("depth 0 should still open first extglob: %+v", toks)
	}

	// With NoExt-equivalent via impossible depth: parent null cannot adopt,
	// extDepth 0, maxDepth -1? Use MaxExtglobRecursion -1: 0 <= -1 false, no adopt at root.
	toks = Scan("*(x)", Options{MaxExtglobRecursion: Int(-1)})
	if len(toks) != 1 || toks[0].Kind != TokenText {
		t.Fatalf("maxDepth -1 should not open: %+v", toks)
	}
}

func TestScanMultipleExtglobs(t *testing.T) {
	toks := Scan("*(a)@(b)", Options{})
	want := []TokenKind{
		TokenExtglobOpen, TokenText, TokenExtglobClose,
		TokenExtglobOpen, TokenText, TokenExtglobClose,
	}
	if !reflect.DeepEqual(kinds(toks), want) {
		t.Fatalf("%v %+v", kinds(toks), toks)
	}
}

func TestScanEmptyExtglobBody(t *testing.T) {
	// *() — empty body; TS sets #emptyExt. We emit Open, Close with no text between.
	toks := Scan("*()", Options{})
	want := []TokenKind{TokenExtglobOpen, TokenExtglobClose}
	if !reflect.DeepEqual(kinds(toks), want) {
		t.Fatalf("%v %+v", kinds(toks), toks)
	}
}

func TestScanEmptyAlternative(t *testing.T) {
	// *(a||b) — empty middle alternative
	toks := Scan("*(a||b)", Options{})
	want := []TokenKind{
		TokenExtglobOpen, TokenText, TokenPipe, TokenPipe, TokenText, TokenExtglobClose,
	}
	if !reflect.DeepEqual(kinds(toks), want) {
		t.Fatalf("%v %+v", kinds(toks), toks)
	}
}

func TestTokenKindString(t *testing.T) {
	if TokenText.String() != "Text" {
		t.Fatal(TokenText.String())
	}
	if TokenExtglobOpen.String() != "ExtglobOpen" {
		t.Fatal()
	}
}

func TestIsExtglobType(t *testing.T) {
	for _, c := range []byte{'!', '?', '+', '*', '@'} {
		if !IsExtglobType(c) {
			t.Fatalf("%c", c)
		}
	}
	if IsExtglobType('x') || IsExtglobType('(') {
		t.Fatal("false positives")
	}
}

func TestScanSpansCoverSegment(t *testing.T) {
	seg := "ab*(c|d)e"
	toks := Scan(seg, Options{})
	// Reconstruct by concatenating raw slices for structural tokens.
	var rebuilt string
	for _, tok := range toks {
		rebuilt += seg[tok.Start:tok.End]
	}
	if rebuilt != seg {
		t.Fatalf("rebuilt %q want %q tokens=%+v", rebuilt, seg, toks)
	}
}

func TestCanAdoptTypeMatchesMap(t *testing.T) {
	// Spot-check adoptionAnyMap
	if !canAdoptType(ExtglobStar, ExtglobOne) {
		t.Fatal("* adopts @")
	}
	if canAdoptType(ExtglobNegate, ExtglobStar) {
		t.Fatal("! must not adopt *")
	}
	if !canAdoptType(ExtglobPlus, ExtglobStar) {
		t.Fatal("+ adopts * (adoptionAnyMap)")
	}
}
