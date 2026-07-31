package brace

import (
	"reflect"
	"strings"
	"testing"
)

func be(p string) []string {
	if !HasSimpleGroup(p) {
		return []string{p}
	}
	return Expand(p, ExpansionMax, ExpansionMaxLength)
}

func TestExpandExamples(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"a{b,c}d", []string{"abd", "acd"}},
		{"a{1..3}b", []string{"a1b", "a2b", "a3b"}},
		{"a{b}c", []string{"a{b}c"}},
		{"file-{a,b}.txt", []string{"file-a.txt", "file-b.txt"}},
		{"{a,b}${c}${d}", []string{"a${c}${d}", "b${c}${d}"}},
		{"a{},b}c", []string{"a}c", "abc"}},
	}
	for _, tc := range cases {
		if got := be(tc.in); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%q: got %#v want %#v", tc.in, got, tc.want)
		}
	}
}

func TestExpandLimit(t *testing.T) {
	p := "{" + strings.Repeat("a,", 1000) + "x}"
	got := Expand(p, 10, ExpansionMaxLength)
	if len(got) != 10 {
		t.Fatal(len(got))
	}
}

func TestBalanced(t *testing.T) {
	pre, body, post, ok := balanced("{", "}", "pre{a,{b,c}}post")
	if !ok || pre != "pre" || body != "a,{b,c}" || post != "post" {
		t.Fatalf("%q %q %q %v", pre, body, post, ok)
	}
}

func TestHasSimpleGroup(t *testing.T) {
	if !HasSimpleGroup("{a,b}") || HasSimpleGroup("abc") || HasSimpleGroup("{{") {
		t.Fatal()
	}
}
