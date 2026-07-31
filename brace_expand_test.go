package minimatch

import (
	"reflect"
	"strings"
	"testing"
)

func TestBraceExpandExamplesFromMinimatch(t *testing.T) {
	// test/brace-expand.js
	cases := []struct {
		pattern string
		want    []string
	}{
		{
			"a{b,c{d,e},{f,g}h}x{y,z}",
			[]string{
				"abxy", "abxz",
				"acdxy", "acdxz",
				"acexy", "acexz",
				"afhxy", "afhxz",
				"aghxy", "aghxz",
			},
		},
		{"a{1..5}b", []string{"a1b", "a2b", "a3b", "a4b", "a5b"}},
		{"a{b}c", []string{"a{b}c"}},
		{"a{00..05}b", []string{"a00b", "a01b", "a02b", "a03b", "a04b", "a05b"}},
		{"z{a,b},c}d", []string{"za,c}d", "zb,c}d"}},
		{"z{a,b{,c}d", []string{"z{a,bd", "z{a,bcd"}},
		{"a{b{c{d,e}f}g}h", []string{"a{b{cdf}g}h", "a{b{cef}g}h"}},
		{
			"a{b{c{d,e}f{x,y}}g}h",
			[]string{"a{b{cdfx}g}h", "a{b{cdfy}g}h", "a{b{cefx}g}h", "a{b{cefy}g}h"},
		},
		{
			"a{b{c{d,e}f{x,y{}g}h",
			[]string{"a{b{cdfxh", "a{b{cdfy{}gh", "a{b{cefxh", "a{b{cefy{}gh"},
		},
		{"{a,b}${c}${d}", []string{"a${c}${d}", "b${c}${d}"}},
		{"${a}${b}{c,d}", []string{"${a}${b}c", "${a}${b}d"}},
	}
	for _, tc := range cases {
		got, err := BraceExpand(tc.pattern, Options{})
		if err != nil {
			t.Fatalf("%q: %v", tc.pattern, err)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%q\ngot  %#v\nwant %#v", tc.pattern, got, tc.want)
		}
	}
}

func TestBraceExpandLimit(t *testing.T) {
	// test/brace-expand.js limit brace expansion
	p := "{" + strings.Repeat("a,", 1000) + "x}"
	got, err := BraceExpand(p, Options{BraceExpandMax: Int(10)})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Split(strings.Repeat("a", 10), "")
	// strings.Split("aaaaaaaaaa","") → weird in Go; build explicitly
	want = make([]string, 10)
	for i := range want {
		want[i] = "a"
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v (len %d) want 10 a's", got, len(got))
	}
}

func TestBraceExpandNoBrace(t *testing.T) {
	got, err := BraceExpand("a{b,c}d", Options{NoBrace: true})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"a{b,c}d"}) {
		t.Fatal(got)
	}
}

func TestBraceExpandNoBracesInPattern(t *testing.T) {
	got, err := BraceExpand("abc", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"abc"}) {
		t.Fatal(got)
	}
}

func TestBraceExpandEmpty(t *testing.T) {
	// Empty has no simple brace group → [""], not expand() which returns [].
	got, err := BraceExpand("", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{""}) {
		t.Fatalf("%#v", got)
	}
}

func TestBraceExpandInvalidPattern(t *testing.T) {
	_, err := BraceExpand(strings.Repeat("x", MaxPatternLength+1), Options{})
	if err != ErrPatternTooLong {
		t.Fatalf("got %v", err)
	}
}

func TestBraceExpandReadmeExamples(t *testing.T) {
	// brace-expansion README samples
	cases := []struct {
		in   string
		want []string
	}{
		{"file-{a,b,c}.jpg", []string{"file-a.jpg", "file-b.jpg", "file-c.jpg"}},
		{"-v{,,}", []string{"-v", "-v", "-v"}},
		{"file{0..2}.jpg", []string{"file0.jpg", "file1.jpg", "file2.jpg"}},
		{"file-{a..c}.jpg", []string{"file-a.jpg", "file-b.jpg", "file-c.jpg"}},
		{"file{2..0}.jpg", []string{"file2.jpg", "file1.jpg", "file0.jpg"}},
		{"file{0..4..2}.jpg", []string{"file0.jpg", "file2.jpg", "file4.jpg"}},
		{"file-{a..e..2}.jpg", []string{"file-a.jpg", "file-c.jpg", "file-e.jpg"}},
		{"file{00..10..5}.jpg", []string{"file00.jpg", "file05.jpg", "file10.jpg"}},
		{"{{A..C},{a..c}}", []string{"A", "B", "C", "a", "b", "c"}},
		{"ppp{,config,oe{,conf}}", []string{"ppp", "pppconfig", "pppoe", "pppoeconf"}},
	}
	for _, tc := range cases {
		got, err := BraceExpand(tc.in, Options{})
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%q\ngot  %#v\nwant %#v", tc.in, got, tc.want)
		}
	}
}

func TestBraceExpandLeadingEmptyBraces(t *testing.T) {
	// expand("{},a}b") — leading {} escaped at top level
	// From algorithm comment: {},a}b will not expand to anything useful the same way
	// a{},b}c → [a}c, abc]
	got, err := BraceExpand("a{},b}c", Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a}c", "abc"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestBraceExpandEscapedBraces(t *testing.T) {
	got, err := BraceExpand(`\{a,b\}`, Options{})
	if err != nil {
		t.Fatal(err)
	}
	// Escaped braces are literal after expand
	if !reflect.DeepEqual(got, []string{`{a,b}`}) {
		t.Fatalf("%#v", got)
	}
}

func TestBraceExpandNestedComma(t *testing.T) {
	got, err := BraceExpand("{a,{b,c},d}", Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a", "b", "c", "d"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%#v", got)
	}
}

func TestBalancedBasic(t *testing.T) {
	pre, body, post, ok := balanced("{", "}", "pre{a,{b,c}}post")
	if !ok {
		t.Fatal("expected ok")
	}
	if pre != "pre" || body != "a,{b,c}" || post != "post" {
		t.Fatalf("%q %q %q", pre, body, post)
	}
}

func TestHasSimpleBraceGroup(t *testing.T) {
	if !hasSimpleBraceGroup("{a,b}") {
		t.Fatal()
	}
	if !hasSimpleBraceGroup("x{a}y") {
		t.Fatal()
	}
	if hasSimpleBraceGroup("abc") {
		t.Fatal()
	}
	if hasSimpleBraceGroup("{{") {
		t.Fatal()
	}
	// nested outer has { inside body before }, but inner {a} matches
	if !hasSimpleBraceGroup("{a{b}c}") {
		t.Fatal("inner group")
	}
}

func TestExpandSequenceNegative(t *testing.T) {
	got, err := BraceExpand("{3..-1}", Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"3", "2", "1", "0", "-1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%#v", got)
	}
}

func TestExpandPadNegative(t *testing.T) {
	got, err := BraceExpand("{-01..1}", Options{})
	if err != nil {
		t.Fatal(err)
	}
	// Reference: ["-01","000","001"] (width 3, reverse from -1 to 1)
	want := []string{"-01", "000", "001"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}
