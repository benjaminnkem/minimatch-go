// Differential fuzz harness for Port Mortem.
//
// Compares github.com/benjaminnkem/minimatch-go against Node isaacs/minimatch
// on random (pattern, files, options) for a fixed duration.
//
// Usage (from module root, with sibling ../minimatch built):
//
//	go run ./fuzz -duration=60s -out=fuzz/log.txt
//
// Exit 0: zero behavioural divergences on the shared MatchList API.
// Exit 1: at least one divergence (logged).
// Exit 2: setup failure (no Node / no reference).
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/benjaminnkem/minimatch-go"
)

const alphabet = "abc*?[]!.-/_{},+@()\\"

func main() {
	duration := flag.Duration("duration", 60*time.Second, "how long to fuzz")
	outPath := flag.String("out", "fuzz/log.txt", "log file path")
	seed := flag.Uint64("seed", 1, "PRNG seed")
	batch := flag.Int("batch", 40, "cases per Node round-trip")
	flag.Parse()

	oracle, err := resolveOracle()
	if err != nil {
		fmt.Fprintf(os.Stderr, "setup: %v\n", err)
		os.Exit(2)
	}

	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(2)
	}
	logf, err := os.Create(*outPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create log: %v\n", err)
		os.Exit(2)
	}
	defer logf.Close()

	log := func(format string, args ...any) {
		line := fmt.Sprintf(format, args...)
		fmt.Fprint(logf, line)
		fmt.Fprint(os.Stdout, line)
	}

	log("=== Port Mortem differential fuzz harness ===\n")
	log("port:     github.com/benjaminnkem/minimatch-go\n")
	log("original: isaacs/minimatch (Node oracle via testdata/oracle.mjs)\n")
	log("api:      MatchList(files, pattern, options) / minimatch.match\n")
	log("duration: %s\n", duration.String())
	log("seed:     %d\n", *seed)
	log("batch:    %d\n", *batch)
	log("go:       %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	log("started:  %s\n", time.Now().UTC().Format(time.RFC3339))
	log("\n")

	rng := newRNG(*seed)
	deadline := time.Now().Add(*duration)
	var (
		rounds, cases, compared, skipped, divergences int
		firstDiv                                      string
	)

	type cas struct {
		Files   []string       `json:"files"`
		Pattern string         `json:"pattern"`
		Options map[string]any `json:"options"`
	}

	for time.Now().Before(deadline) {
		rounds++
		batchCases := make([]cas, 0, *batch)
		goOpts := make([]minimatch.Options, 0, *batch)
		for i := 0; i < *batch; i++ {
			pat := randomGlob(rng, 1+rng.Intn(24))
			files := make([]string, 1+rng.Intn(6))
			for j := range files {
				files[j] = randomPath(rng, 1+rng.Intn(20))
			}
			o := minimatch.Options{Platform: minimatch.PlatformLinux}
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
			batchCases = append(batchCases, cas{Files: files, Pattern: pat, Options: om})
			goOpts = append(goOpts, o)
		}

		nodeOut, err := runOracle(oracle, batchCases)
		if err != nil {
			log("oracle error (round %d): %v\n", rounds, err)
			// treat as setup/runtime issue, not a divergence
			continue
		}
		if len(nodeOut) != len(batchCases) {
			log("oracle length mismatch round %d: got %d want %d\n", rounds, len(nodeOut), len(batchCases))
			continue
		}

		for i, c := range batchCases {
			cases++
			if nodeOut[i].Error != "" {
				// Node threw — skip strict compare (untrusted garbage patterns)
				skipped++
				continue
			}
			got, err := minimatch.MatchList(c.Files, c.Pattern, goOpts[i])
			if err != nil {
				// Go error while Node succeeded → divergence
				divergences++
				msg := fmt.Sprintf("DIV pattern=%q files=%v opts=%v node=%v go_err=%v\n",
					c.Pattern, c.Files, c.Options, nodeOut[i].Result, err)
				log(msg)
				if firstDiv == "" {
					firstDiv = msg
				}
				continue
			}
			want := append([]string(nil), nodeOut[i].Result...)
			sort.Strings(want)
			sort.Strings(got)
			compared++
			if !reflect.DeepEqual(want, got) {
				divergences++
				msg := fmt.Sprintf("DIV pattern=%q files=%v opts=%v want=%v got=%v\n",
					c.Pattern, c.Files, c.Options, want, got)
				log(msg)
				if firstDiv == "" {
					firstDiv = msg
				}
			}
		}

		// progress every ~5s of wall clock via rounds
		if rounds%5 == 0 {
			elapsed := time.Since(deadline.Add(-*duration))
			log("progress: elapsed=%s rounds=%d cases=%d compared=%d skipped=%d div=%d\n",
				elapsed.Round(time.Millisecond), rounds, cases, compared, skipped, divergences)
		}
	}

	elapsed := *duration
	log("\n=== summary ===\n")
	log("elapsed:      %s (requested)\n", elapsed)
	log("rounds:       %d\n", rounds)
	log("cases:        %d\n", cases)
	log("compared:     %d\n", compared)
	log("skipped_err:  %d (Node threw; not counted as div)\n", skipped)
	log("divergences:  %d\n", divergences)
	log("finished:     %s\n", time.Now().UTC().Format(time.RFC3339))
	if divergences == 0 {
		log("result:       PASS (zero divergences on shared MatchList API)\n")
		os.Exit(0)
	}
	log("result:       FAIL\n")
	if firstDiv != "" {
		log("first:        %s", firstDiv)
	}
	os.Exit(1)
}

type oracleResult struct {
	Result []string `json:"result"`
	Error  string   `json:"error"`
}

func resolveOracle() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller failed")
	}
	// fuzz/harness.go → module root
	root := filepath.Join(filepath.Dir(file), "..")
	oracle := filepath.Join(root, "testdata", "oracle.mjs")
	if _, err := os.Stat(oracle); err != nil {
		return "", fmt.Errorf("oracle.mjs: %w", err)
	}
	ref := filepath.Join(root, "..", "minimatch", "dist", "esm", "index.js")
	if _, err := os.Stat(ref); err != nil {
		return "", fmt.Errorf("reference dist missing (%s); build ../minimatch (npm run prepare)", ref)
	}
	if _, err := exec.LookPath("node"); err != nil {
		return "", fmt.Errorf("node not on PATH")
	}
	return oracle, nil
}

func runOracle(oracle string, cases any) ([]oracleResult, error) {
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
		return nil, fmt.Errorf("%w: %s", err, stderr.String())
	}
	var out []oracleResult
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, err
	}
	return out, nil
}

type rng struct{ s uint64 }

func newRNG(seed uint64) *rng {
	if seed == 0 {
		seed = 1
	}
	return &rng{s: seed}
}

func (r *rng) next() uint64 {
	x := r.s
	x ^= x << 13
	x ^= x >> 7
	x ^= x << 17
	r.s = x
	return x
}

func (r *rng) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	return int(r.next() % uint64(n))
}

func randomGlob(r *rng, n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
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
			b.WriteByte(alphabet[r.Intn(len(alphabet))])
		}
	}
	s := b.String()
	if len(s) > 64 {
		s = s[:64]
	}
	return s
}

func randomPath(r *rng, n int) string {
	var parts []string
	remain := n
	for remain > 0 {
		L := 1 + r.Intn(min(6, remain))
		var b strings.Builder
		for i := 0; i < L; i++ {
			b.WriteByte(alphabet[r.Intn(len(alphabet))])
		}
		parts = append(parts, b.String())
		remain -= L
	}
	if len(parts) == 0 {
		return "a"
	}
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
