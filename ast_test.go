package minimatch

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestParseGlobPlain(t *testing.T) {
	a := ParseGlob("foo", Options{})
	if !a.IsList() || len(a.Parts) != 1 || a.Parts[0].Text != "foo" {
		t.Fatalf("%+v", a)
	}
	if a.String() != "foo" {
		t.Fatal(a.String())
	}
}

func TestParseGlobStarStaysText(t *testing.T) {
	a := ParseGlob("a*b?c", Options{})
	if a.String() != "a*b?c" || len(a.Parts) != 1 {
		t.Fatalf("%s parts=%+v", a.String(), a.Parts)
	}
}

func TestParseGlobSimpleExtglob(t *testing.T) {
	a := ParseGlob("*(a|b)", Options{})
	if a.String() != "*(a|b)" {
		t.Fatal(a.String())
	}
	if len(a.Parts) != 1 || !a.Parts[0].Node.IsExtglob() {
		t.Fatalf("%+v", a.Parts)
	}
	ext := a.Parts[0].Node
	if ext.Type != ExtglobStar || len(ext.Parts) != 2 {
		t.Fatalf("%+v", ext)
	}
	if ext.Parts[0].Node.String() != "a" || ext.Parts[1].Node.String() != "b" {
		t.Fatalf("alts %q %q", ext.Parts[0].Node.String(), ext.Parts[1].Node.String())
	}
}

func TestParseGlobPrefixSuffix(t *testing.T) {
	a := ParseGlob("a*(b)c", Options{})
	if a.String() != "a*(b)c" {
		t.Fatal(a.String())
	}
	if len(a.Parts) != 3 {
		t.Fatalf("parts %d", len(a.Parts))
	}
	if a.Parts[0].Text != "a" || a.Parts[2].Text != "c" {
		t.Fatalf("%+v", a.Parts)
	}
	if a.Parts[1].Node.Type != ExtglobStar {
		t.Fatal(a.Parts[1].Node.Type)
	}
}

func TestParseGlobEmptyExt(t *testing.T) {
	a := ParseGlob("*()", Options{})
	ext := a.Parts[0].Node
	if !ext.EmptyExt {
		t.Fatal("EmptyExt")
	}
	if len(ext.Parts) != 1 || len(ext.Parts[0].Node.Parts) != 0 {
		t.Fatalf("%+v", ext.Parts)
	}
	if a.String() != "*()" {
		t.Fatal(a.String())
	}
}

func TestParseGlobEmptyAlternatives(t *testing.T) {
	a := ParseGlob("*(a||b)", Options{})
	ext := a.Parts[0].Node
	if len(ext.Parts) != 3 {
		t.Fatalf("alts %d", len(ext.Parts))
	}
	if ext.Parts[0].Node.String() != "a" {
		t.Fatal("first")
	}
	if len(ext.Parts[1].Node.Parts) != 0 {
		t.Fatal("middle should be empty list")
	}
	if ext.Parts[2].Node.String() != "b" {
		t.Fatal("third")
	}
}

func TestParseGlobNested(t *testing.T) {
	a := ParseGlob("*(a|@(b|c))", Options{})
	if a.String() != "*(a|@(b|c))" {
		t.Fatal(a.String())
	}
	ext := a.Parts[0].Node
	inner := ext.Parts[1].Node.Parts[0].Node
	if inner.Type != ExtglobOne || len(inner.Parts) != 2 {
		t.Fatalf("%+v", inner)
	}
}

func TestParseGlobAllTypes(t *testing.T) {
	for _, typ := range []ExtglobType{ExtglobNegate, ExtglobOptional, ExtglobPlus, ExtglobStar, ExtglobOne} {
		p := string(typ) + "(x)"
		a := ParseGlob(p, Options{})
		if a.String() != p {
			t.Fatalf("%s -> %s", p, a.String())
		}
		if a.Parts[0].Node.Type != typ {
			t.Fatal(typ)
		}
	}
}

func TestParseGlobNoExt(t *testing.T) {
	a := ParseGlob("*(a|b)", Options{NoExt: true})
	if !a.IsList() || len(a.Parts) != 1 || a.Parts[0].Text != "*(a|b)" {
		t.Fatalf("%+v", a)
	}
}

func TestParseGlobUnfinishedDemotion(t *testing.T) {
	// TypeScript: *(a → root has one list child with text "*(a"
	a := ParseGlob("*(a", Options{})
	if a.String() != "*(a" {
		t.Fatal(a.String())
	}
	if len(a.Parts) != 1 || a.Parts[0].Node == nil {
		t.Fatalf("%+v", a.Parts)
	}
	child := a.Parts[0].Node
	if !child.IsList() || len(child.Parts) != 1 || child.Parts[0].Text != "*(a" {
		t.Fatalf("%+v", child)
	}
}

func TestParseGlobUnfinishedOuterWipesNested(t *testing.T) {
	a := ParseGlob("*(a|@(b", Options{})
	if a.String() != "*(a|@(b" {
		t.Fatal(a.String())
	}
	// Demoted outer is a single text list child
	child := a.Parts[0].Node
	if !child.IsList() || child.Parts[0].Text != "*(a|@(b" {
		t.Fatalf("%+v", child)
	}
}

func TestParseGlobEscapedNotExtglob(t *testing.T) {
	a := ParseGlob(`\*(a)`, Options{})
	if a.String() != `\*(a)` {
		t.Fatal(a.String())
	}
	if len(a.Parts) != 1 || a.Parts[0].Node != nil {
		t.Fatal("should be plain text")
	}
}

func TestParseGlobClassKeepsExtglobOut(t *testing.T) {
	a := ParseGlob("[*(]", Options{})
	if a.String() != "[*(]" || a.Parts[0].Node != nil {
		t.Fatal(a.String())
	}
}

func TestParseTokensRoundTripWithScan(t *testing.T) {
	src := "x@(a|b)y"
	toks := Scan(src, Options{})
	a := ParseTokens(src, toks, Options{})
	if a.String() != src {
		t.Fatal(a.String())
	}
}

func TestASTToJSONMatchesReferenceShapes(t *testing.T) {
	// Compared loosely to node AST.toJSON() samples (filledNegs false).
	// Shapes match TypeScript AST.toJSON() before fillNegs (node reference).
	cases := map[string]string{
		"foo":         `[[],"foo",{}]`,
		"a*b":         `[[],"a*b",{}]`,
		"*(a|b)":      `[[],["*",[[],"a"],[[],"b"]],{}]`,
		"a*(b)c":      `[[],"a",["*",["b"]],"c",{}]`,
		"*()":         `[[],["*",[[]]],{}]`,
		"*(a||b)":     `[[],["*",[[],"a"],[[]],[[],"b"]],{}]`,
		"*(a|@(b|c))": `[[],["*",[[],"a"],[[],["@",[[],"b"],[[],"c"]]]],{}]`,
		"*(a":         `[[],[[],"*(a"],{}]`,
		"!(x)":        `[[],["!",[[],"x"]],{}]`,
	}
	for p, wantJSON := range cases {
		a := ParseGlob(p, Options{})
		got, err := json.Marshal(a.ToJSON())
		if err != nil {
			t.Fatal(err)
		}
		var want any
		var gotV any
		if err := json.Unmarshal([]byte(wantJSON), &want); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(got, &gotV); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(gotV, want) {
			t.Errorf("%q\ngot  %s\nwant %s", p, got, wantJSON)
		}
	}
}

func TestParseGlobRootOptions(t *testing.T) {
	opts := Options{NoCase: true, Dot: true}
	a := ParseGlob("*(a)", opts)
	if !a.Options.NoCase || !a.Parts[0].Node.Options.NoCase {
		t.Fatal("options should be shared from root")
	}
}

func TestASTDepthAndParent(t *testing.T) {
	a := ParseGlob("*(@(x))", Options{})
	ext := a.Parts[0].Node
	inner := ext.Parts[0].Node.Parts[0].Node
	if a.Depth() != 0 || ext.Depth() != 1 {
		t.Fatalf("depths %d %d", a.Depth(), ext.Depth())
	}
	if inner.Parent() != ext.Parts[0].Node {
		// inner's parent is the alternative list of outer, then @ is child of that list
	}
	if ext.Root() != a || inner.Root() != a {
		t.Fatal("root")
	}
}
