package minimatch

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"
)

// charset for structured fuzz inputs (glob-relevant + a few literals).
const fuzzAlphabet = "abc*?[]!.-/_{},+@()\\"

func fuzzSanitize(s string, max int) string {
	if len(s) > max {
		s = s[:max]
	}
	// Keep valid UTF-8; drop invalid bytes.
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "")
	}
	return s
}

// FuzzMatchNoPanic ensures Match/NewMinimatch do not panic on random input.
func FuzzMatchNoPanic(f *testing.F) {
	for _, s := range []string{"*", "**", "a*", "?.js", "!(a)", "a/**/b", "[a-z]", "[[:alpha:]]", ""} {
		f.Add(s, s)
	}
	f.Fuzz(func(t *testing.T, pattern, path string) {
		pattern = fuzzSanitize(pattern, 256)
		path = fuzzSanitize(path, 256)
		if len(pattern) > MaxPatternLength {
			return
		}
		_, _ = Match(path, pattern, Options{})
		_, _ = Match(path, pattern, Options{Dot: true, NoCase: true, Partial: true})
		_, _ = Match(path, pattern, Options{NoNegate: true, NoGlobStar: true})
		m, err := NewMinimatch(pattern, Options{})
		if err != nil {
			return
		}
		_ = m.Match(path)
		_, _ = m.MakeRe()
		_ = m.HasMagic()
	})
}

// TestFuzzDifferentialBatch generates random cases and compares to Node
// when node and the reference package are available.
func TestFuzzDifferentialBatch(t *testing.T) {
	oracle := nodeOraclePath(t)
	if oracle == "" {
		t.Skip("node oracle unavailable")
	}

	rng := newFuzzRNG(1)
	const n = 80
	type cas struct {
		Files   []string       `json:"files"`
		Pattern string         `json:"pattern"`
		Options map[string]any `json:"options"`
	}
	cases := make([]cas, 0, n)
	goOpts := make([]Options, 0, n)

	for i := 0; i < n; i++ {
		pat := randomGlob(rng, 1+rng.Intn(24))
		files := make([]string, 1+rng.Intn(6))
		for j := range files {
			files[j] = randomPath(rng, 1+rng.Intn(20))
		}
		// Force POSIX for parity with Node default / fixtures (see differential_test).
		o := Options{Platform: PlatformLinux}
		om := map[string]any{"platform": "linux"}
		if rng.Intn(2) == 0 {
			o.Dot = true
			om["dot"] = true
		}
		if rng.Intn(3) == 0 {
			o.NoCase = true
			om["nocase"] = true
		}
		if rng.Intn(4) == 0 {
			o.NoNegate = true
			om["nonegate"] = true
		}
		if rng.Intn(4) == 0 {
			o.NoGlobStar = true
			om["noglobstar"] = true
		}
		if rng.Intn(5) == 0 {
			o.MatchBase = true
			om["matchBase"] = true
		}
		cases = append(cases, cas{Files: files, Pattern: pat, Options: om})
		goOpts = append(goOpts, o)
	}

	nodeOut, err := runNodeOracle(oracle, cases)
	if err != nil {
		t.Fatalf("oracle: %v", err)
	}
	if len(nodeOut) != len(cases) {
		t.Fatalf("oracle len %d want %d", len(nodeOut), len(cases))
	}

	var failed int
	for i, c := range cases {
		if nodeOut[i].Error != "" {
			// Node threw — Go should error or not match; skip strict compare
			continue
		}
		got, err := MatchList(c.Files, c.Pattern, goOpts[i])
		if err != nil {
			// Node succeeded; Go error is a divergence
			failed++
			if failed <= 15 {
				t.Errorf("case %d pattern %q: go err %v node %q", i, c.Pattern, err, nodeOut[i].Result)
			}
			continue
		}
		gotS := sortedCopy(got)
		wantS := sortedCopy(nodeOut[i].Result)
		if !reflect.DeepEqual(gotS, wantS) {
			failed++
			if failed <= 15 {
				t.Errorf("case %d pattern %q files=%q\n  got  %q\n  want %q\n  opts %v",
					i, c.Pattern, c.Files, gotS, wantS, c.Options)
			}
		}
	}
	if failed > 0 {
		t.Fatalf("%d / %d random cases differed from Node", failed, n)
	}
	t.Logf("ok: %d random cases match Node", n)
}

type oracleResult struct {
	Result []string `json:"result"`
	Error  string   `json:"error"`
}

func nodeOraclePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	oracle := filepath.Join(filepath.Dir(file), "testdata", "oracle.mjs")
	if _, err := os.Stat(oracle); err != nil {
		return ""
	}
	// reference package dist
	ref := filepath.Join(filepath.Dir(file), "..", "minimatch", "dist", "esm", "index.js")
	if _, err := os.Stat(ref); err != nil {
		return ""
	}
	if _, err := exec.LookPath("node"); err != nil {
		return ""
	}
	return oracle
}

func runNodeOracle(oracle string, cases any) ([]oracleResult, error) {
	body, err := json.Marshal(cases)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("node", oracle)
	cmd.Stdin = bytes.NewReader(body)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	var out []oracleResult
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// tiny deterministic PRNG (xorshift) so tests are reproducible without math/rand/v2 issues.
type fuzzRNG struct{ s uint64 }

func newFuzzRNG(seed uint64) *fuzzRNG {
	if seed == 0 {
		seed = 1
	}
	return &fuzzRNG{s: seed}
}

func (r *fuzzRNG) next() uint64 {
	x := r.s
	x ^= x << 13
	x ^= x >> 7
	x ^= x << 17
	r.s = x
	return x
}

func (r *fuzzRNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	return int(r.next() % uint64(n))
}

func randomGlob(r *fuzzRNG, n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		// bias toward interesting tokens
		switch r.Intn(12) {
		case 0:
			b.WriteString("**")
		case 1:
			b.WriteByte('*')
		case 2:
			b.WriteByte('?')
		case 3:
			b.WriteString("*/")
		case 4:
			b.WriteString("[a-z]")
		case 5:
			b.WriteString("!(a|b)")
		case 6:
			b.WriteString("+(x)")
		case 7:
			b.WriteString("{a,b}")
		default:
			b.WriteByte(fuzzAlphabet[r.Intn(len(fuzzAlphabet))])
		}
	}
	s := b.String()
	if len(s) > 64 {
		s = s[:64]
	}
	return s
}

func randomPath(r *fuzzRNG, n int) string {
	var parts []string
	remain := n
	for remain > 0 {
		L := 1 + r.Intn(min(6, remain))
		var b strings.Builder
		for i := 0; i < L; i++ {
			b.WriteByte(fuzzAlphabet[r.Intn(len(fuzzAlphabet))])
		}
		parts = append(parts, b.String())
		remain -= L
		if remain > 0 && r.Intn(2) == 0 {
			// slash separate
		}
	}
	if len(parts) == 0 {
		return "a"
	}
	// join some with /
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		if r.Intn(3) == 0 {
			out += parts[i]
		} else {
			out += "/" + parts[i]
		}
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
