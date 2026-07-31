package minimatch

import (
	"reflect"
	"testing"
)

func TestHasMagic(t *testing.T) {
	m, err := NewMinimatch("foo", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if m.HasMagic() {
		t.Fatal("literal")
	}
	m, _ = NewMinimatch("f*", Options{})
	if !m.HasMagic() {
		t.Fatal("star")
	}
	m, _ = NewMinimatch("a{b,c}d", Options{})
	if m.HasMagic() {
		t.Fatal("braces alone not magic by default")
	}
	m, _ = NewMinimatch("a{b,c}d", Options{MagicalBraces: true})
	if !m.HasMagic() {
		t.Fatal("magicalBraces")
	}
}

func TestFilter(t *testing.T) {
	f := Filter("*.js", Options{MatchBase: true})
	got := filterSlice([]string{"a.js", "b.txt", "/x/c.js"}, f)
	want := []string{"a.js", "/x/c.js"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}
}

func filterSlice(in []string, f func(string) bool) []string {
	var out []string
	for _, s := range in {
		if f(s) {
			out = append(out, s)
		}
	}
	return out
}

func TestMatchList(t *testing.T) {
	got, err := MatchList([]string{"a.js", "b.txt", "c.js"}, "*.js", Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a.js", "c.js"}
	if !reflect.DeepEqual(got, want) {
		t.Fatal(got)
	}
	got, _ = MatchList([]string{"a"}, "nomatch", Options{NoNull: true})
	if !reflect.DeepEqual(got, []string{"nomatch"}) {
		t.Fatal(got)
	}
	got, _ = MatchList([]string{"a"}, "nomatch", Options{})
	if len(got) != 0 {
		t.Fatal(got)
	}
}

func TestMakeReBasic(t *testing.T) {
	m, err := NewMinimatch("a/*/c", Options{})
	if err != nil {
		t.Fatal(err)
	}
	re, ok := m.MakeRe()
	if !ok || re == nil {
		t.Fatal("MakeRe")
	}
	ok2, err := re.MatchString("a/b/c")
	if err != nil || !ok2 {
		t.Fatalf("a/b/c %v %v", ok2, err)
	}
	ok2, _ = re.MatchString("a/b/d")
	if ok2 {
		t.Fatal("a/b/d")
	}
	// cache
	re2, ok := m.MakeRe()
	if !ok || re2 != re {
		t.Fatal("cache")
	}
}

func TestMakeReGlobstar(t *testing.T) {
	re, ok, err := MakeRe("a/**/b", Options{})
	if err != nil || !ok {
		t.Fatal(err, ok)
	}
	ok2, _ := re.MatchString("a/x/y/b")
	if !ok2 {
		t.Fatal("globstar re")
	}
}

func TestMakeReCommentEmpty(t *testing.T) {
	m, _ := NewMinimatch("#c", Options{})
	_, ok := m.MakeRe()
	if ok {
		t.Fatal("comment has empty set")
	}
}

func TestDefaults(t *testing.T) {
	d := NewDefaults(Options{Dot: true})
	ok, err := d.Match(".hidden", "*", Options{})
	if err != nil || !ok {
		t.Fatalf("defaults dot: %v %v", ok, err)
	}
	// call-site can still add flags
	ok, _ = d.Match("FOO", "foo", Options{NoCase: true})
	if !ok {
		t.Fatal("nocase on call")
	}
	esc := d.Escape("*", EscapeOptions{})
	if esc != `\*` {
		t.Fatal(esc)
	}
	d2 := NewDefaults(Options{WindowsPathsNoEscape: true})
	if d2.Escape("*", EscapeOptions{}) != "[*]" {
		t.Fatal(d2.Escape("*", EscapeOptions{}))
	}
}

func TestHostSep(t *testing.T) {
	s := HostSep()
	if s != SepPOSIX && s != SepWindows {
		t.Fatal(s)
	}
}
