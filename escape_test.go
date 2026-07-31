package minimatch

import (
	"testing"
)

func TestEscapePosixBasic(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"*", `\*`},
		{"?", `\?`},
		{"foo*", `foo\*`},
		{"a(b)", `a\(b\)`},
		{"a[b]", `a\[b\]`},
		{`a\b`, `a\\b`},
		{"{a,b}", "{a,b}"}, // magicalBraces false: braces untouched
		{"/", "/"},
		{"", ""},
		{"abc", "abc"},
	}
	for _, tc := range cases {
		got := Escape(tc.in, EscapeOptions{})
		if got != tc.want {
			t.Errorf("Escape(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestEscapeWindowsBasic(t *testing.T) {
	opts := EscapeOptions{WindowsPathsNoEscape: true}
	cases := []struct {
		in, want string
	}{
		{"*", "[*]"},
		{"?", "[?]"},
		{"foo*", "foo[*]"},
		{"a(b)", "a[(]b[)]"},
		{"a[b]", "a[[]b[]]"},
		{`a\b`, `a\b`}, // backslash not escaped
		{"{a,b}", "{a,b}"},
		{"/", "/"},
	}
	for _, tc := range cases {
		got := Escape(tc.in, opts)
		if got != tc.want {
			t.Errorf("Escape(%q, win) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestEscapeMagicalBraces(t *testing.T) {
	// From test/escape-has-magic.js bracedEscapeTests
	tests := []struct {
		pattern string
		escaped string
		opts    EscapeOptions
	}{
		{"{a,b}", `\{a,b\}`, EscapeOptions{MagicalBraces: true}},
		{"{a,b}", "[{]a,b[}]", EscapeOptions{MagicalBraces: true, WindowsPathsNoEscape: true}},
		{"[{]", `\[\{\]`, EscapeOptions{MagicalBraces: true}},
		{"[{]", "[[][{][]]", EscapeOptions{MagicalBraces: true, WindowsPathsNoEscape: true}},
		{`[\{]`, `\[\\\{\]`, EscapeOptions{MagicalBraces: true}},
		{`[\{]`, `[[]\[{][]]`, EscapeOptions{MagicalBraces: true, WindowsPathsNoEscape: true}},
		{"{a,b}", "{a,b}", EscapeOptions{}},
		{"{a,b}", "{a,b}", EscapeOptions{WindowsPathsNoEscape: true}},
		{"[{]", `\[{\]`, EscapeOptions{}},
		{"[{]", "[[]{[]]", EscapeOptions{WindowsPathsNoEscape: true}},
		{`[\{]`, `\[\\{\]`, EscapeOptions{}},
		{`[\{]`, `[[]\{[]]`, EscapeOptions{WindowsPathsNoEscape: true}},
	}
	for _, tc := range tests {
		got := Escape(tc.pattern, tc.opts)
		if got != tc.escaped {
			t.Errorf("Escape(%q, %+v) = %q, want %q", tc.pattern, tc.opts, got, tc.escaped)
		}
	}
}

func TestEscapeUnescapeRoundTripBraced(t *testing.T) {
	// Same table: unescape(escape(pattern)) == pattern with matching opts.
	// Unescape magicalBraces default differs; pass explicit pointer matching Escape.
	tests := []struct {
		pattern string
		opts    EscapeOptions
	}{
		{"{a,b}", EscapeOptions{MagicalBraces: true}},
		{"{a,b}", EscapeOptions{MagicalBraces: true, WindowsPathsNoEscape: true}},
		{"[{]", EscapeOptions{MagicalBraces: true}},
		{"[{]", EscapeOptions{MagicalBraces: true, WindowsPathsNoEscape: true}},
		{`[\{]`, EscapeOptions{MagicalBraces: true}},
		{`[\{]`, EscapeOptions{MagicalBraces: true, WindowsPathsNoEscape: true}},
		{"{a,b}", EscapeOptions{}},
		{"{a,b}", EscapeOptions{WindowsPathsNoEscape: true}},
		{"[{]", EscapeOptions{}},
		{"[{]", EscapeOptions{WindowsPathsNoEscape: true}},
		{`[\{]`, EscapeOptions{}},
		{`[\{]`, EscapeOptions{WindowsPathsNoEscape: true}},
		{"*", EscapeOptions{}},
		{"*", EscapeOptions{WindowsPathsNoEscape: true}},
		{"foo?(bar)", EscapeOptions{}},
		{"a[b]c*", EscapeOptions{WindowsPathsNoEscape: true}},
	}
	for _, tc := range tests {
		escaped := Escape(tc.pattern, tc.opts)
		uopts := UnescapeOptions{WindowsPathsNoEscape: tc.opts.WindowsPathsNoEscape}
		mb := tc.opts.MagicalBraces
		uopts.MagicalBraces = &mb
		got := Unescape(escaped, uopts)
		if got != tc.pattern {
			t.Errorf("round-trip %q opts=%+v: escaped=%q unescaped=%q",
				tc.pattern, tc.opts, escaped, got)
		}
	}
}

func TestUnescapeEdgeBracketBackslash(t *testing.T) {
	// test/escape-has-magic.js: unescape('[\\]') === '[]'
	if got := Unescape(`[\]`, UnescapeOptions{}); got != "[]" {
		t.Fatalf(`Unescape("[\\]") = %q, want "[]"`, got)
	}
}

func TestUnescapeDefaultMagicalBracesTrue(t *testing.T) {
	// Zero-value UnescapeOptions must unescape braces (TS default true).
	if got := Unescape(`\{a,b\}`, UnescapeOptions{}); got != "{a,b}" {
		t.Fatalf("default Unescape braces: got %q", got)
	}
	// Explicit false must preserve brace escapes.
	f := false
	if got := Unescape(`\{a,b\}`, UnescapeOptions{MagicalBraces: &f}); got != `\{a,b\}` {
		t.Fatalf("MagicalBraces false: got %q", got)
	}
}

func TestEscapeOptionsFrom(t *testing.T) {
	f := false
	o := Options{
		WindowsPathsNoEscape: false,
		AllowWindowsEscape:   &f, // forces WindowsPathsNoEscape effective true
		MagicalBraces:        true,
	}
	eo := EscapeOptionsFrom(o)
	if !eo.WindowsPathsNoEscape {
		t.Fatal("expected EffectiveWindowsPathsNoEscape")
	}
	if !eo.MagicalBraces {
		t.Fatal("expected MagicalBraces")
	}
}

func TestUnescapeOptionsFrom(t *testing.T) {
	o := Options{MagicalBraces: false, WindowsPathsNoEscape: true}
	uo := UnescapeOptionsFrom(o)
	if !uo.WindowsPathsNoEscape {
		t.Fatal("windows")
	}
	if uo.MagicalBraces == nil || *uo.MagicalBraces {
		t.Fatal("explicit MagicalBraces false from Options")
	}
}

func TestEscapeDoesNotEscapeSlash(t *testing.T) {
	if got := Escape("a/b*", EscapeOptions{}); got != `a/b\*` {
		t.Fatalf("got %q", got)
	}
	if got := Escape(`a\b*`, EscapeOptions{WindowsPathsNoEscape: true}); got != `a\b[*]` {
		t.Fatalf("got %q", got)
	}
}

func TestDefaultsStyleStar(t *testing.T) {
	// test/defaults.js style checks without defaults() helper
	if got := Escape("*", EscapeOptions{}); got != `\*` {
		t.Fatal(got)
	}
	if got := Unescape(Escape("*", EscapeOptions{}), UnescapeOptions{}); got != "*" {
		t.Fatal(got)
	}
	w := EscapeOptions{WindowsPathsNoEscape: true}
	if got := Escape("*", w); got != "[*]" {
		t.Fatal(got)
	}
	if got := Unescape(Escape("*", w), UnescapeOptions{WindowsPathsNoEscape: true}); got != "*" {
		t.Fatal(got)
	}
}
