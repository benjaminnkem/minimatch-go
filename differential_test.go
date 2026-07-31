package minimatch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"
)

// patternCase is one row from testdata/patterns_node.json (Node oracle).
type patternCase struct {
	Pattern string         `json:"pattern"`
	Expect  []string       `json:"expect"`
	Options map[string]any `json:"options"`
	Files   []string       `json:"files"`
}

func loadPatternCases(t *testing.T) []patternCase {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	path := filepath.Join(filepath.Dir(file), "testdata", "patterns_node.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (run testdata/generate_patterns.mjs)", path, err)
	}
	var cases []patternCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	return cases
}

func optionsFromJSON(m map[string]any) Options {
	var o Options
	if m == nil {
		return o
	}
	boolField := func(key string) bool {
		v, ok := m[key]
		if !ok || v == nil {
			return false
		}
		b, _ := v.(bool)
		return b
	}
	o.NoBrace = boolField("nobrace")
	o.NoComment = boolField("nocomment")
	o.NoNegate = boolField("nonegate")
	o.Debug = boolField("debug")
	o.NoGlobStar = boolField("noglobstar")
	o.NoExt = boolField("noext")
	// patterns.js `null: true` is historical and ignored by modern minimatch.
	o.NoNull = boolField("nonull")
	o.WindowsPathsNoEscape = boolField("windowsPathsNoEscape")
	o.Partial = boolField("partial")
	o.Dot = boolField("dot")
	o.NoCase = boolField("nocase")
	o.NoCaseMagicOnly = boolField("nocaseMagicOnly")
	o.MagicalBraces = boolField("magicalBraces")
	o.MatchBase = boolField("matchBase")
	o.FlipNegate = boolField("flipNegate")
	o.PreserveMultipleSlashes = boolField("preserveMultipleSlashes")
	if v, ok := m["optimizationLevel"]; ok {
		switch n := v.(type) {
		case float64:
			i := int(n)
			o.OptimizationLevel = &i
		}
	}
	if v, ok := m["platform"].(string); ok {
		o.Platform = Platform(v)
	}
	if v, ok := m["windowsNoMagicRoot"]; ok {
		if b, ok := v.(bool); ok {
			o.WindowsNoMagicRoot = &b
		}
	}
	if v, ok := m["braceExpandMax"]; ok {
		if n, ok := v.(float64); ok {
			i := int(n)
			o.BraceExpandMax = &i
		}
	}
	if v, ok := m["maxGlobstarRecursion"]; ok {
		if n, ok := v.(float64); ok {
			i := int(n)
			o.MaxGlobstarRecursion = &i
		}
	}
	if v, ok := m["maxExtglobRecursion"]; ok {
		if n, ok := v.(float64); ok {
			i := int(n)
			o.MaxExtglobRecursion = &i
		}
	}
	if v, ok := m["allowWindowsEscape"]; ok {
		if b, ok := v.(bool); ok {
			o.AllowWindowsEscape = &b
		}
	}
	return o
}

func sortedCopy(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

// TestDifferentialPatterns compares Go MatchList to Node minimatch.match
// results captured in testdata/patterns_node.json.
func TestDifferentialPatterns(t *testing.T) {
	cases := loadPatternCases(t)
	var failed int
	const maxReport = 40
	for i, c := range cases {
		opts := optionsFromJSON(c.Options)
		// Node nonull
		if v, ok := c.Options["nonull"].(bool); ok && v {
			opts.NoNull = true
		}
		got, err := MatchList(c.Files, c.Pattern, opts)
		if err != nil {
			failed++
			if failed <= maxReport {
				t.Errorf("case %d pattern %q: error %v", i, c.Pattern, err)
			}
			continue
		}
		gotS := sortedCopy(got)
		wantS := sortedCopy(c.Expect)
		if !reflect.DeepEqual(gotS, wantS) {
			failed++
			if failed <= maxReport {
				t.Errorf("case %d pattern %q opts=%v\n  files=%q\n  got  %q\n  want %q",
					i, c.Pattern, c.Options, c.Files, gotS, wantS)
			}
		}
	}
	if failed > 0 {
		t.Fatalf("%d / %d pattern cases differed from Node oracle", failed, len(cases))
	}
	t.Logf("ok: %d cases match Node minimatch.match", len(cases))
}

// TestDifferentialTrickyNegations covers test/tricky-negations.js style cases.
func TestDifferentialTrickyNegations(t *testing.T) {
	// Subset of tricky-negations.js (filename → pattern → expect)
	// Using NoNegate where patterns start with !( for extglob.
	type row struct {
		file, pat string
		opts      Options
		want      bool
	}
	cases := []row{
		{"bar.min.js", "*.!(js|css)", Options{NoNegate: true}, true},
		{"bar.min.js", "*.+(js|css)", Options{}, true},
		{"a-integration-test.js", "*-!(integration-)test.js", Options{NoNegate: true}, true},
		{"a-integration-test.js", "!(*-integration-test.js)", Options{NoNegate: true}, false},
		{"foo.jszzz.js", "*.!(js).js", Options{NoNegate: true}, true},
	}
	for _, c := range cases {
		ok, err := Match(c.file, c.pat, c.opts)
		if err != nil {
			t.Fatal(err)
		}
		if ok != c.want {
			t.Errorf("%q %q: got %v want %v", c.file, c.pat, ok, c.want)
		}
	}
}

// TestDifferentialPartial covers test/partial.ts style cases.
func TestDifferentialPartial(t *testing.T) {
	cases := []struct {
		path, pat string
		want      bool
	}{
		{"/a/b", "/*/b/x/y/z", true},
		{"/a/b/c", "/*/b/x/y/z", false},
		{"/", "x", true},
		{"/b/c/d/a", "/**/a/b/c", true},
		{"a", "a/**", true},
		{"b/a", "a/**", false},
	}
	for _, c := range cases {
		ok, err := Match(c.path, c.pat, Options{Partial: true})
		if err != nil {
			t.Fatal(err)
		}
		if ok != c.want {
			t.Errorf("%q %q: got %v want %v", c.path, c.pat, ok, c.want)
		}
	}
}
