package minimatch

import "testing"

var benchSink bool

func BenchmarkMatchSimple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ok, _ := Match("foo/bar.js", "**/*.js", Options{})
		benchSink = ok
	}
}

func BenchmarkMatchCompiled(b *testing.B) {
	m, err := NewMinimatch("**/*.js", Options{})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSink = m.Match("foo/bar/baz.js")
	}
}

func BenchmarkNewMinimatchComplex(b *testing.B) {
	pat := "a/{b,c}/**/!(tmp)/*.{js,ts}"
	for i := 0; i < b.N; i++ {
		_, _ = NewMinimatch(pat, Options{NoNegate: true})
	}
}

func BenchmarkMatchExtglob(b *testing.B) {
	m, err := NewMinimatch("*(a|b|c)", Options{})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSink = m.Match("abc")
	}
}

func BenchmarkMatchListPatternsCorpus(b *testing.B) {
	// warm a few representative patterns from the suite
	patterns := []string{"*", "**", "a/**/b", "*.js", "[a-z]*", "?(a|b)", "!(x)", "{a,b}*"}
	files := []string{"a", "b", "abc", "foo.js", "a/b/c", "x"}
	ms := make([]*Minimatch, 0, len(patterns))
	for _, p := range patterns {
		m, err := NewMinimatch(p, Options{NoNegate: true})
		if err != nil {
			b.Fatal(err)
		}
		ms = append(ms, m)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, m := range ms {
			for _, f := range files {
				benchSink = m.Match(f)
			}
		}
	}
}

func BenchmarkMakeRe(b *testing.B) {
	m, err := NewMinimatch("a/**/b/*.js", Options{})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = m.MakeRe()
		// force rebuild each time
		m.makeReBuilt = false
		m.makeReCached = nil
	}
}

func BenchmarkBraceExpand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = BraceExpand("a{1..5}{b,c,d}x", Options{})
	}
}
