package minimatch

import "testing"

func TestFlattenUsurpAndAdopt(t *testing.T) {
	// Reference: AST.fromGlob(p); toRegExpSource(); toString()
	cases := []struct {
		in, want string
	}{
		{"!(!(a|b))", "@(a|b)"},
		{"?(*(a|b))", "*(a|b)"},
		{"?(+(a|b))", "*(a|b)"},
		{"@(!(a|b))", "!(a|b)"},
		{"@(?(a|b))", "@(a|b|)"},
		{"@(*(a|b))", "*(a|b)"},
		{"@(+(a|b))", "+(a|b)"},
		{"+(?(a|b))", "+(a|b|)"},
		{"+(*(a|b))", "+(a|b|)"},
		{"*(a|*(b|c)|d)", "*(a|b|c|d)"},
		{"+(a|@(b|c)|d)", "+(a|b|c|d)"},
		{"!(a|@(b|c)|d)", "!(a|b|c|d)"},
		{"!(a|?(b|c)|d)", "!(a|b|c||d)"},
		{"@(a*b)", "@(a*b)"},
		{"x@(a*b)", "x@(a*b)"},
		{"x!(!(a|b))", "x@(a|b)"},
		{"x?(*(a|b))", "x*(a|b)"},
		{"!(!(a|b))y", "@(a|b)y"},
	}
	for _, tc := range cases {
		a := ParseGlob(tc.in, Options{})
		before := a.String()
		a.Flatten()
		got := a.String()
		if got != tc.want {
			t.Errorf("%q: before=%q got=%q want=%q", tc.in, before, got, tc.want)
		}
	}
}

func TestFlattenIdempotent(t *testing.T) {
	a := ParseGlob("*(a|*(b|c)|d)", Options{})
	a.Flatten()
	once := a.String()
	a.Flatten()
	if a.String() != once {
		t.Fatalf("second flatten changed %q -> %q", once, a.String())
	}
}

func TestFlattenNoExtglobUnchanged(t *testing.T) {
	a := ParseGlob("a*b", Options{})
	a.Flatten()
	if a.String() != "a*b" {
		t.Fatal(a.String())
	}
}

func TestFlattenAdoptStar(t *testing.T) {
	// * can adopt nested *
	a := ParseGlob("*(a|*(b)|c)", Options{})
	a.Flatten()
	if a.String() != "*(a|b|c)" {
		t.Fatal(a.String())
	}
}

func TestFlattenDoesNotAdoptIllegal(t *testing.T) {
	// + cannot adopt * (would allow empty) without with-space map.
	// + *can* adopt-with-space for * → +(…|) form
	a := ParseGlob("+(*(a|b))", Options{})
	a.Flatten()
	if a.String() != "+(a|b|)" {
		t.Fatal(a.String())
	}
}

func TestFlattenChainedReturnsReceiver(t *testing.T) {
	a := ParseGlob("!(!(x))", Options{})
	if a.Flatten() != a {
		t.Fatal("want same pointer")
	}
	if a.String() != "@(x)" {
		t.Fatal(a.String())
	}
}

func TestFlattenEmptyAltInString(t *testing.T) {
	// adopt-with-space produces trailing empty alternative
	a := ParseGlob("@(?(a))", Options{})
	a.Flatten()
	if a.String() != "@(a|)" {
		t.Fatal(a.String())
	}
}
