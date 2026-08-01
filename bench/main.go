// Comparative bench — Go minimatch-go side.
// stdout: JSON with scenarios + meta.
//
//	go run ./bench
//	go run ./bench -startup-only   # one Match then exit (for cold-start timing)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"time"

	"github.com/benjaminnkem/minimatch-go"
)

func main() {
	startupOnly := flag.Bool("startup-only", false, "run one Match and exit (cold start helper)")
	flag.Parse()
	if *startupOnly {
		_, _ = minimatch.Match("foo/bar.js", "**/*.js", minimatch.Options{Platform: minimatch.PlatformLinux})
		return
	}

	opts := minimatch.Options{Platform: minimatch.PlatformLinux}
	patterns := []string{"*", "**", "a/**/b", "*.js", "[a-z]*", "?(a|b)", "!(x)", "{a,b}*"}
	files := []string{"a", "b", "abc", "foo.js", "a/b/c", "x"}

	scenarios := map[string]any{}

	scenarios["match_oneshot"] = timeLoop(3000, func() {
		_, _ = minimatch.Match("foo/bar.js", "**/*.js", opts)
	})

	m, err := minimatch.NewMinimatch("**/*.js", opts)
	if err != nil {
		fail(err)
	}
	scenarios["match_compiled"] = timeLoop(20000, func() {
		_ = m.Match("foo/bar/baz.js")
	})

	mExt, err := minimatch.NewMinimatch("*(a|b|c)", opts)
	if err != nil {
		fail(err)
	}
	scenarios["match_extglob"] = timeLoop(20000, func() {
		_ = mExt.Match("abc")
	})

	scenarios["match_complex_compile"] = timeLoop(500, func() {
		_, _ = minimatch.NewMinimatch("a/{b,c}/**/!(tmp)/*.{js,ts}", minimatch.Options{
			Platform: minimatch.PlatformLinux,
			NoNegate: true,
		})
	})

	scenarios["brace_expand"] = timeLoop(5000, func() {
		_, _ = minimatch.BraceExpand("a{1..5}{b,c,d}x", opts)
	})

	ms := make([]*minimatch.Minimatch, 0, len(patterns))
	for _, p := range patterns {
		mm, err := minimatch.NewMinimatch(p, minimatch.Options{Platform: minimatch.PlatformLinux, NoNegate: true})
		if err != nil {
			fail(err)
		}
		ms = append(ms, mm)
	}
	scenarios["corpus"] = timeLoop(2000, func() {
		for _, mm := range ms {
			for _, f := range files {
				_ = mm.Match(f)
			}
		}
	})

	var msMem runtime.MemStats
	runtime.ReadMemStats(&msMem)

	out := map[string]any{
		"impl":    "go-minimatch-go",
		"version": "0.1.0",
		"meta": map[string]any{
			"go":       runtime.Version(),
			"platform": runtime.GOOS,
			"arch":     runtime.GOARCH,
			"cpus":     runtime.NumCPU(),
		},
		"memory": map[string]any{
			"heap_bytes": msMem.HeapAlloc,
			// RSS is OS-specific; heap is the honest portable signal here.
			"rss_bytes": nil,
		},
		"scenarios": scenarios,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fail(err)
	}
}

func timeLoop(iters int, fn func()) map[string]any {
	warm := iters
	if warm > 200 {
		warm = 200
	}
	for i := 0; i < warm; i++ {
		fn()
	}
	samples := make([]float64, 0, iters)
	t0 := time.Now()
	for i := 0; i < iters; i++ {
		a := time.Now()
		fn()
		samples = append(samples, float64(time.Since(a).Nanoseconds()))
	}
	total := time.Since(t0)
	q := quantiles(samples)
	return map[string]any{
		"ops":             iters,
		"ns_per_op_mean":  float64(total.Nanoseconds()) / float64(iters),
		"ns_per_op_p50":   q[0],
		"ns_per_op_p95":   q[1],
		"ns_per_op_p99":   q[2],
		"samples":         len(samples),
	}
}

func quantiles(samples []float64) [3]float64 {
	if len(samples) == 0 {
		return [3]float64{}
	}
	s := append([]float64(nil), samples...)
	sort.Float64s(s)
	at := func(p float64) float64 {
		i := int(p * float64(len(s)-1))
		if i < 0 {
			i = 0
		}
		if i >= len(s) {
			i = len(s) - 1
		}
		return s[i]
	}
	return [3]float64{at(0.5), at(0.95), at(0.99)}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
