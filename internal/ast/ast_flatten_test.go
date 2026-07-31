package ast

import "testing"

func TestFlattenUsurpAndAdopt(t *testing.T) {
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
		a := ParseGlob(tc.in, Config{})
		a.Flatten()
		if got := a.String(); got != tc.want {
			t.Errorf("%q: got %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestFlattenIdempotent(t *testing.T) {
	a := ParseGlob("*(a|*(b|c)|d)", Config{})
	a.Flatten()
	once := a.String()
	a.Flatten()
	if a.String() != once {
		t.Fatal(a.String())
	}
}
