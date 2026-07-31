package minimatch_test

import (
	"fmt"

	"github.com/benjaminnkem/minimatch-go"
)

func ExampleMatch() {
	ok, err := minimatch.Match("src/app/index.ts", "**/*.{js,ts}", minimatch.Options{})
	if err != nil {
		panic(err)
	}
	fmt.Println(ok)
	// Output: true
}

func ExampleNewMinimatch() {
	m, err := minimatch.NewMinimatch("*.md", minimatch.Options{MatchBase: true})
	if err != nil {
		panic(err)
	}
	fmt.Println(m.Match("docs/README.md"))
	fmt.Println(m.HasMagic())
	// Output:
	// true
	// true
}

func ExampleMatchList() {
	files, err := minimatch.MatchList(
		[]string{"a.js", "b.txt", "c.js"},
		"*.js",
		minimatch.Options{},
	)
	if err != nil {
		panic(err)
	}
	fmt.Println(files)
	// Output: [a.js c.js]
}

func ExampleBraceExpand() {
	out, err := minimatch.BraceExpand("file-{a,b}.txt", minimatch.Options{})
	if err != nil {
		panic(err)
	}
	fmt.Println(out)
	// Output: [file-a.txt file-b.txt]
}
