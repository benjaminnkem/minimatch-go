package ast

import (
	"encoding/json"
	"github.com/tochison/minimatch/internal/scan"
	"reflect"
	"testing"
)

func prepareAST(pattern string) *AST {
	a := ParseGlob(pattern, Config{})
	a.Flatten()
	a.FillNegs()
	return a
}

func TestFillNegsCopiesTrailingText(t *testing.T) {
	// !(a)b → each alt of ! gains "b"
	a := prepareAST("!(a)b")
	neg := a.Parts[0].Node
	if neg.Type != scan.ExtglobNegate {
		t.Fatal(neg.Type)
	}
	alt := neg.Parts[0].Node
	if len(alt.Parts) != 2 || alt.Parts[0].Text != "a" || alt.Parts[1].Text != "b" {
		t.Fatalf("alt parts %+v", alt.Parts)
	}
	// Root still has trailing "b" as well
	if a.Parts[1].Text != "b" {
		t.Fatal(a.Parts[1])
	}
}

func TestFillNegsMultipleAlts(t *testing.T) {
	a := prepareAST("!(a|b)c")
	neg := a.Parts[0].Node
	if len(neg.Parts) != 2 {
		t.Fatal(len(neg.Parts))
	}
	for i, want := range []string{"a", "b"} {
		alt := neg.Parts[i].Node
		if len(alt.Parts) != 2 || alt.Parts[0].Text != want || alt.Parts[1].Text != "c" {
			t.Fatalf("alt %d %+v", i, alt.Parts)
		}
	}
}

func TestFillNegsPrefixAndSuffix(t *testing.T) {
	a := prepareAST("x!(a)y")
	// parts: "x", !, "y"
	neg := a.Parts[1].Node
	alt := neg.Parts[0].Node
	// "a" may be without isStart marker; parts text a, y
	if alt.String() != "ay" {
		t.Fatalf("alt %q parts %+v", alt.String(), alt.Parts)
	}
}

func TestFillNegsNestedInAt(t *testing.T) {
	// @(!(a)b) → ! gains "b" from following sibling in list alt
	a := prepareAST("@(!(a)b)")
	at := a.Parts[0].Node
	list := at.Parts[0].Node
	var neg *AST
	for _, p := range list.Parts {
		if p.Node != nil && p.Node.Type == scan.ExtglobNegate {
			neg = p.Node
			break
		}
	}
	if neg == nil {
		t.Fatal("missing !")
	}
	alt := neg.Parts[0].Node
	if alt.String() != "ab" {
		t.Fatalf("got %q %+v", alt.String(), alt.Parts)
	}
}

func TestFillNegsIdempotent(t *testing.T) {
	a := ParseGlob("!(a)b", Config{})
	a.Flatten()
	a.FillNegs()
	j1, _ := json.Marshal(a.ToJSON())
	a.FillNegs()
	j2, _ := json.Marshal(a.ToJSON())
	if !reflect.DeepEqual(j1, j2) {
		t.Fatalf("%s vs %s", j1, j2)
	}
}

func TestFillNegsJSONMatchesReference(t *testing.T) {
	// Captured from TypeScript AST after toRegExpSource (flatten+fillNegs).
	cases := map[string]string{
		"!(a)":     `[[],["!",[[],"a",{}]],{}]`,
		"!(a)b":    `[[],["!",[[],"a","b",{}]],"b",{}]`,
		"x!(a)y":   `[[],"x",["!",["a","y",{}]],"y",{}]`,
		"!(a|b)c":  `[[],["!",[[],"a","c",{}],[[],"b","c",{}]],"c",{}]`,
		"a!(b|c)d": `[[],"a",["!",["b","d",{}],["c","d",{}]],"d",{}]`,
		"!(a*)":    `[[],["!",[[],"a*",{}]],{}]`,
		"@(!(a)b)": `[[],["@",[[],["!",[[],"a","b",{}]],"b"]],{}]`,
		// flatten turns !(!(a)) into @(a); fillNegs has little left to do
		"!(!(a))": `[[],["@",[[],"a"]],{}]`,
	}
	for p, wantJSON := range cases {
		a := prepareAST(p)
		got, err := json.Marshal(a.ToJSON())
		if err != nil {
			t.Fatal(err)
		}
		var want, gotV any
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

func TestFillNegsDoubleNegationChain(t *testing.T) {
	// !(a)!(b) — first ! receives a clone of second ! as tail
	a := prepareAST("!(a)!(b)")
	first := a.Parts[0].Node
	alt := first.Parts[0].Node
	// should contain text "a" and a nested !
	if len(alt.Parts) < 2 {
		t.Fatalf("parts %+v", alt.Parts)
	}
	if alt.Parts[0].Text != "a" {
		t.Fatal(alt.Parts[0])
	}
	if alt.Parts[1].Node == nil || alt.Parts[1].Node.Type != scan.ExtglobNegate {
		t.Fatalf("expected nested ! %+v", alt.Parts)
	}
}

func TestFillNegsChainedReturnsReceiver(t *testing.T) {
	a := ParseGlob("!(x)y", Config{})
	if a.Flatten().FillNegs() != a {
		t.Fatal("want same pointer")
	}
	if !a.FilledNegs() {
		t.Fatal("FilledNegs")
	}
}

func TestFillNegsNoNegationsNoop(t *testing.T) {
	a := prepareAST("*(a|b)")
	if a.String() != "*(a|b)" {
		// String is not cached; after fill still same structure
		t.Fatal(a.String())
	}
	if !a.FilledNegs() {
		t.Fatal("should still mark filled")
	}
}
