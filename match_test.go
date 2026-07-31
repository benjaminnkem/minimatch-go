package minimatch

import "testing"

func TestMatchBasic(t *testing.T) {
	cases := []struct {
		path, pat string
		opts      Options
		want      bool
	}{
		{"bar.foo", "*.foo", Options{}, true},
		{"bar.foo", "*.bar", Options{}, false},
		{"a/b/c", "a/*/c", Options{}, true},
		{"a/b/c", "a/*/d", Options{}, false},
		{"a/x/y/b", "a/**/b", Options{}, true},
		{"a/b", "a/**/b", Options{}, true},
		{"foo", "foo", Options{}, true},
		{"foo", "bar", Options{}, false},
		{".hidden", "*", Options{}, false},
		{".hidden", "*", Options{Dot: true}, true},
	}
	for _, tc := range cases {
		ok, err := Match(tc.path, tc.pat, tc.opts)
		if err != nil {
			t.Fatal(err)
		}
		if ok != tc.want {
			t.Errorf("%q vs %q opts=%+v: got %v want %v", tc.path, tc.pat, tc.opts, ok, tc.want)
		}
	}
}

func TestMatchNegate(t *testing.T) {
	ok, err := Match("foo", "!foo", Options{})
	if err != nil || ok {
		t.Fatalf("got %v %v", ok, err)
	}
	ok, _ = Match("bar", "!foo", Options{})
	if !ok {
		t.Fatal("bar should match !foo")
	}
}

func TestMatchComment(t *testing.T) {
	ok, err := Match("#x", "#comment", Options{})
	if err != nil || ok {
		t.Fatalf("comment matches nothing: %v %v", ok, err)
	}
}

func TestMatchBrace(t *testing.T) {
	ok, _ := Match("abd", "a{b,c}d", Options{})
	if !ok {
		t.Fatal("abd")
	}
	ok, _ = Match("acd", "a{b,c}d", Options{})
	if !ok {
		t.Fatal("acd")
	}
	ok, _ = Match("axd", "a{b,c}d", Options{})
	if ok {
		t.Fatal("axd")
	}
}

func TestMatchExtglob(t *testing.T) {
	ok, _ := Match("a", "*(a|b)", Options{})
	if !ok {
		t.Fatal("*(a|b) a")
	}
	// Leading ! is pattern negation unless NoNegate; use NoNegate for !(…) extglob.
	o := Options{NoNegate: true}
	ok, _ = Match("ab", "!(a)b", o)
	if ok {
		t.Fatal("ab should not match !(a)b")
	}
	ok, _ = Match("xb", "!(a)b", o)
	if !ok {
		t.Fatal("xb should match !(a)b")
	}
}

func TestMatchGlobstarDeep(t *testing.T) {
	ok, _ := Match("a/b/c/d", "a/**/d", Options{})
	if !ok {
		t.Fatal("a/**/d")
	}
	ok, _ = Match("a/d", "a/**/d", Options{})
	if !ok {
		t.Fatal("a/**/d short")
	}
}

func TestMatchPartial(t *testing.T) {
	ok, _ := Match("/a/b", "/*/b/x/y/z", Options{Partial: true})
	if !ok {
		t.Fatal("partial")
	}
	ok, _ = Match("/x/y/z", "/a/**/z", Options{Partial: true})
	if ok {
		t.Fatal("partial false")
	}
}

func TestMatchMatchBase(t *testing.T) {
	ok, _ := Match("/xyz/123/acb", "a?b", Options{MatchBase: true})
	if !ok {
		t.Fatal("matchBase")
	}
	ok, _ = Match("/xyz/acb/123", "a?b", Options{MatchBase: true})
	if ok {
		t.Fatal("matchBase basename only")
	}
}

func TestMatchEmpty(t *testing.T) {
	ok, _ := Match("", "", Options{})
	if !ok {
		t.Fatal("empty")
	}
	ok, _ = Match("x", "", Options{})
	if ok {
		t.Fatal("empty pattern only empty path")
	}
}

func TestMatchCharacterClass(t *testing.T) {
	ok, _ := Match("a", "[a-z]", Options{})
	if !ok {
		t.Fatal("[a-z]")
	}
	ok, _ = Match("A", "[a-z]", Options{})
	if ok {
		t.Fatal("case")
	}
}

func TestMatchQmark(t *testing.T) {
	ok, _ := Match("ab", "??", Options{})
	if !ok {
		t.Fatal("??")
	}
	ok, _ = Match("a", "??", Options{})
	if ok {
		t.Fatal("too short")
	}
}
