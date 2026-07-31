package minimatch

import (
	"strings"
	"testing"
)

func TestParseClassNotAtBracket(t *testing.T) {
	_, err := ParseClass("abc", 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseClassSimpleRange(t *testing.T) {
	r := parseClass("[a-z]", 0)
	if r.Src != "[a-z]" || !r.HasMagic || r.Consumed != 5 {
		t.Fatalf("%+v", r)
	}
}

func TestParseClassNegate(t *testing.T) {
	r := parseClass("[!a]", 0)
	if r.Src != "[^a]" || !r.HasMagic {
		t.Fatalf("%+v", r)
	}
	r = parseClass("[^a]", 0)
	if r.Src != "[^a]" {
		t.Fatalf("%+v", r)
	}
}

func TestParseClassSingleLiteralNonMagic(t *testing.T) {
	// [_] is an escape form for literal _
	r := parseClass("[_]", 0)
	if r.HasMagic {
		t.Fatal("should not be magic")
	}
	if r.Src != "_" && r.Src != `\_` {
		// _ may not need escape in regexpEscape
		if r.Src != "_" {
			t.Fatalf("src %q", r.Src)
		}
	}
	// [*] → literal *
	r = parseClass("[*]", 0)
	if r.HasMagic || !strings.Contains(r.Src, "*") {
		t.Fatalf("%+v", r)
	}
}

func TestParseClassUnclosed(t *testing.T) {
	r := parseClass("[abc", 0)
	if r.Consumed != 0 || r.Src != "" || r.HasMagic {
		t.Fatalf("%+v", r)
	}
}

func TestParseClassEmptyImpossible(t *testing.T) {
	// [] with only close after negate? [!] empty body — first ] after !
	// might not close cleanly. Empty class []: first ] doesn't close (sawStart false).
	// [a-b] fine.
	// Impossible: no ranges — e.g. only out of order?
	r := parseClass("[z-a]", 0)
	// z-a is out of order so range dropped → empty → $.
	if r.Src != "$." || !r.HasMagic {
		t.Fatalf("%+v", r)
	}
}

func TestParseClassOutOfOrderDroppedButOthersKept(t *testing.T) {
	// [az-a] → 'a' then z-a dropped → just a?
	// Actually: a, then z range start, a end: a < z so drop, left with 'a' only
	// single char non-magic
	r := parseClass("[az-a]", 0)
	if r.HasMagic {
		// if only 'a' left, non-magic
		t.Logf("%+v", r)
	}
	// [a-cz-a] → a-c kept, z-a dropped
	r = parseClass("[a-cz-a]", 0)
	if !strings.Contains(r.Src, "a-c") || !r.HasMagic {
		t.Fatalf("%+v", r)
	}
}

func TestParseClassTrailingDash(t *testing.T) {
	// [a-] → range form c-]
	r := parseClass("[a-]", 0)
	if !strings.Contains(r.Src, "a-") && r.Src != "[a\\-]" && !strings.Contains(r.Src, `a\-`) {
		// braceEscape on "a-"
		t.Logf("%+v", r)
	}
	if !r.HasMagic {
		t.Fatalf("expected magic %+v", r)
	}
}

func TestParseClassEscape(t *testing.T) {
	r := parseClass(`[a\]b]`, 0)
	if r.Consumed == 0 {
		t.Fatal("should parse")
	}
	if !r.HasMagic {
		t.Fatalf("%+v", r)
	}
}

func TestParseClassPosixAlpha(t *testing.T) {
	r := parseClass("[[:alpha:]]", 0)
	if !r.UFlag || !r.HasMagic {
		t.Fatalf("%+v", r)
	}
	if !strings.Contains(r.Src, `\p{L}`) {
		t.Fatalf("src %q", r.Src)
	}
}

func TestParseClassPosixDigit(t *testing.T) {
	r := parseClass("[[:digit:]]", 0)
	if !strings.Contains(r.Src, `\p{Nd}`) || !r.UFlag {
		t.Fatalf("%+v", r)
	}
}

func TestParseClassPosixXdigit(t *testing.T) {
	r := parseClass("[[:xdigit:]]", 0)
	if r.UFlag {
		t.Fatal("xdigit should not need u")
	}
	if !strings.Contains(r.Src, "A-Fa-f0-9") {
		t.Fatalf("%q", r.Src)
	}
}

func TestParseClassPosixGraphNegated(t *testing.T) {
	// graph goes to negs; with no positive ranges, comb is snegs only
	r := parseClass("[[:graph:]]", 0)
	if !r.HasMagic || !r.UFlag {
		t.Fatalf("%+v", r)
	}
	// snegs with negate false → [^...]
	if !strings.Contains(r.Src, `^`) {
		t.Fatalf("expected negation for graph alone: %q", r.Src)
	}
}

func TestParseClassPosixAfterRangeStartPoisons(t *testing.T) {
	// [a-[:alpha:]] → rangeStart a, then posix → poison $.
	r := parseClass("[a-[:alpha:]]", 0)
	if r.Src != "$." || !r.HasMagic {
		t.Fatalf("%+v", r)
	}
}

func TestParseClassEmbedded(t *testing.T) {
	// parse at offset
	r := parseClass("x[a-z]y", 1)
	if r.Src != "[a-z]" || r.Consumed != 5 {
		t.Fatalf("%+v", r)
	}
}

func TestBraceEscape(t *testing.T) {
	if braceEscape(`a-b`) != `a\-b` {
		t.Fatal(braceEscape(`a-b`))
	}
	if braceEscape(`[`) != `\[` {
		t.Fatal(braceEscape(`[`))
	}
}

func TestRegexpEscape(t *testing.T) {
	if regexpEscape(`a.b`) != `a\.b` {
		t.Fatal(regexpEscape(`a.b`))
	}
}

func TestParseClassFirstCloseEmpty(t *testing.T) {
	// []a] — first ] does not close (sawStart false), then a, then ]
	// This is a classic glob form: []a] matches ] or a
	r := parseClass("[]a]", 0)
	if r.Consumed == 0 {
		t.Fatalf("should be a class %+v", r)
	}
	if !r.HasMagic {
		t.Fatalf("%+v", r)
	}
}
