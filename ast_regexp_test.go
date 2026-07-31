package minimatch

import (
	"testing"
)

func TestToRegExpSourceLiterals(t *testing.T) {
	a := ParseGlob("foo", Options{})
	src := a.ToRegExpSource(nil)
	if src.Re != "foo" || src.Body != "foo" || src.HasMagic {
		t.Fatalf("%+v", src)
	}
	mm, err := a.ToMMPattern()
	if err != nil {
		t.Fatal(err)
	}
	if mm.IsRE || mm.Literal != "foo" {
		t.Fatalf("%+v", mm)
	}
	ok, _ := mm.Match("foo")
	if !ok {
		t.Fatal("match foo")
	}
	ok, _ = mm.Match("bar")
	if ok {
		t.Fatal("no match bar")
	}
}

func TestToRegExpSourceStar(t *testing.T) {
	// Reference: ["(?!\\.)[^/]+?","[^/]+?",true,false]
	a := ParseGlob("*", Options{})
	src := a.ToRegExpSource(nil)
	if src.Re != `(?!\.)[^/]+?` {
		t.Fatalf("Re=%q", src.Re)
	}
	if !src.HasMagic {
		t.Fatal("magic")
	}
	mm, err := a.ToMMPattern()
	if err != nil {
		t.Fatal(err)
	}
	if !mm.IsRE {
		t.Fatal("expected RE")
	}
	ok, err := mm.Match("foo")
	if err != nil || !ok {
		t.Fatalf("match foo: %v %v", ok, err)
	}
	ok, _ = mm.Match(".hidden")
	if ok {
		t.Fatal("dotfile should not match without dot:true")
	}
}

func TestToRegExpSourceStarDot(t *testing.T) {
	// Reference with dot: ["(?!(?:^|/)\\.\\.?(?:$|/))[^/]+?", ...]
	a := ParseGlob("*", Options{Dot: true})
	src := a.ToRegExpSource(nil)
	if src.Re != `(?!(?:^|/)\.\.?(?:$|/))[^/]+?` {
		t.Fatalf("Re=%q", src.Re)
	}
	mm, err := a.ToMMPattern()
	if err != nil {
		t.Fatal(err)
	}
	ok, _ := mm.Match(".hidden")
	if !ok {
		t.Fatal("dotfile should match with Dot")
	}
	ok, _ = mm.Match(".")
	if ok {
		t.Fatal(". alone should not match *")
	}
}

func TestToRegExpSourceStarExt(t *testing.T) {
	a := ParseGlob("*.js", Options{})
	src := a.ToRegExpSource(nil)
	if src.Re != `(?!\.)[^/]*?\.js` {
		t.Fatalf("Re=%q", src.Re)
	}
	mm, _ := a.ToMMPattern()
	ok, _ := mm.Match("a.js")
	if !ok {
		t.Fatal("a.js")
	}
	ok, _ = mm.Match(".js")
	if ok {
		t.Fatal(".js should not match without Dot")
	}
}

func TestToRegExpSourceAStarB(t *testing.T) {
	a := ParseGlob("a*b", Options{})
	src := a.ToRegExpSource(nil)
	if src.Re != `a[^/]*?b` {
		t.Fatalf("Re=%q", src.Re)
	}
}

func TestToRegExpSourceQmark(t *testing.T) {
	a := ParseGlob("?", Options{})
	src := a.ToRegExpSource(nil)
	if src.Re != `(?!\.)[^/]` {
		t.Fatalf("Re=%q", src.Re)
	}
}

func TestToRegExpSourceClass(t *testing.T) {
	a := ParseGlob("[a-z]", Options{})
	src := a.ToRegExpSource(nil)
	if src.Re != `(?!\.)[a-z]` {
		t.Fatalf("Re=%q", src.Re)
	}
}

func TestToRegExpSourcePosix(t *testing.T) {
	a := ParseGlob("[[:alpha:]]", Options{})
	src := a.ToRegExpSource(nil)
	if src.Re != `(?!\.)[\p{L}\p{Nl}]` {
		t.Fatalf("Re=%q", src.Re)
	}
	if !src.UFlag {
		t.Fatal("uflag")
	}
	mm, err := a.ToMMPattern()
	if err != nil {
		t.Fatal(err)
	}
	ok, err := mm.Match("é")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("unicode alpha")
	}
}

func TestToRegExpSourceExtglobs(t *testing.T) {
	cases := []struct {
		pat, re string
	}{
		{"*(a|b)", `(?:a|b)*`},
		{"+(x)", `(?:x)+`},
		{"?(a|b)", `(?:a|b)?`},
		{"@(a)", `(?:a)`},
	}
	for _, tc := range cases {
		a := ParseGlob(tc.pat, Options{})
		src := a.ToRegExpSource(nil)
		if src.Re != tc.re {
			t.Errorf("%s: Re=%q want %q", tc.pat, src.Re, tc.re)
		}
	}
}

func TestToRegExpSourceNegExtglob(t *testing.T) {
	// Reference: (?:(?!(?:ab(?:$|\/)))(?!\.)[^/]*?)b
	a := ParseGlob("!(a)b", Options{})
	src := a.ToRegExpSource(nil)
	want := `(?:(?!(?:ab(?:$|\/)))(?!\.)[^/]*?)b`
	if src.Re != want {
		t.Fatalf("Re=%q\nwant %q", src.Re, want)
	}
	mm, err := a.ToMMPattern()
	if err != nil {
		t.Fatal(err)
	}
	ok, _ := mm.Match("xb")
	if !ok {
		t.Fatal("xb should match !(a)b")
	}
	ok, _ = mm.Match("ab")
	if ok {
		t.Fatal("ab should not match !(a)b")
	}
}

func TestToRegExpSourceDots(t *testing.T) {
	for _, p := range []string{".", ".."} {
		a := ParseGlob(p, Options{})
		mm, err := a.ToMMPattern()
		if err != nil {
			t.Fatal(err)
		}
		if mm.IsRE {
			t.Fatalf("%s should be literal", p)
		}
		if mm.Literal != p {
			// body is unescaped; "." stays "."
			if mm.Literal != p {
				t.Fatalf("%s literal %q", p, mm.Literal)
			}
		}
	}
}

func TestToMMPatternNonMagicClass(t *testing.T) {
	a := ParseGlob("[_]", Options{})
	mm, err := a.ToMMPattern()
	if err != nil {
		t.Fatal(err)
	}
	if mm.IsRE || mm.Literal != "_" {
		t.Fatalf("%+v", mm)
	}
}

func TestToRegExpSourceEscapedStar(t *testing.T) {
	a := ParseGlob(`\*`, Options{})
	src := a.ToRegExpSource(nil)
	// escaped * is literal *
	if src.HasMagic {
		t.Fatalf("should not be magic: %+v", src)
	}
	mm, _ := a.ToMMPattern()
	ok, _ := mm.Match("*")
	if !ok {
		t.Fatal("literal *")
	}
}

func TestToMMPatternNocase(t *testing.T) {
	a := ParseGlob("*.JS", Options{NoCase: true})
	mm, err := a.ToMMPattern()
	if err != nil {
		t.Fatal(err)
	}
	ok, _ := mm.Match("foo.js")
	if !ok {
		t.Fatal("nocase")
	}
}

func TestParseGlobOnlyStarsNoEmpty(t *testing.T) {
	// whole pattern * uses +?
	a := ParseGlob("**", Options{}) // as single segment text ** not globstar yet
	// Wait - ** in a segment is just two stars in text, coalesced to one star
	src := a.ToRegExpSource(nil)
	// noEmpty true, only stars → +?
	if src.Re != `(?!\.)[^/]+?` {
		t.Fatalf("Re=%q", src.Re)
	}
}
