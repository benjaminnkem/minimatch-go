package minimatch

import (
	"reflect"
	"testing"
)

func TestNewMinimatchNegate(t *testing.T) {
	m, err := NewMinimatch("!foo", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !m.Negate || m.Pattern != "foo" || m.Comment {
		t.Fatalf("%+v", m)
	}
	if !reflect.DeepEqual(m.GlobSet, []string{"foo"}) {
		t.Fatal(m.GlobSet)
	}
	if !reflect.DeepEqual(m.GlobParts, [][]string{{"foo"}}) {
		t.Fatal(m.GlobParts)
	}

	m, _ = NewMinimatch("!!foo", Options{})
	if m.Negate {
		t.Fatal("double negate")
	}
	if m.Pattern != "foo" {
		t.Fatal(m.Pattern)
	}

	m, _ = NewMinimatch("!foo", Options{NoNegate: true})
	if m.Negate || m.Pattern != "!foo" {
		t.Fatalf("%+v pattern=%q", m, m.Pattern)
	}
}

func TestNewMinimatchComment(t *testing.T) {
	m, err := NewMinimatch("#comment", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !m.Comment || len(m.GlobSet) != 0 {
		t.Fatalf("%+v", m)
	}

	m, _ = NewMinimatch("#comment", Options{NoComment: true})
	if m.Comment {
		t.Fatal("nocomment")
	}
	if !reflect.DeepEqual(m.GlobParts, [][]string{{"#comment"}}) {
		t.Fatal(m.GlobParts)
	}
}

func TestNewMinimatchEmpty(t *testing.T) {
	m, err := NewMinimatch("", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !m.Empty {
		t.Fatal("empty")
	}
}

func TestSlashSplitCoalesce(t *testing.T) {
	m, _ := NewMinimatch("x", Options{})
	got := m.SlashSplit("a//b///c")
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%v want %v", got, want)
	}
}

func TestSlashSplitPreserve(t *testing.T) {
	m, _ := NewMinimatch("x", Options{PreserveMultipleSlashes: true})
	got := m.SlashSplit("a//b")
	want := []string{"a", "", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}
}

func TestSlashSplitUNC(t *testing.T) {
	m, _ := NewMinimatch("x", Options{Platform: PlatformWin32})
	got := m.SlashSplit("//host/share")
	want := []string{"", "", "host", "share"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%v want %v", got, want)
	}
}

func TestPreprocessAdjacentGlobstar(t *testing.T) {
	m, _ := NewMinimatch("a/**/**/b", Options{})
	if !reflect.DeepEqual(m.GlobParts, [][]string{{"a", "**", "b"}}) {
		t.Fatal(m.GlobParts)
	}
	m, _ = NewMinimatch("a/**/**/b", Options{OptimizationLevel: Int(0)})
	if !reflect.DeepEqual(m.GlobParts, [][]string{{"a", "**", "b"}}) {
		t.Fatal(m.GlobParts)
	}
}

func TestPreprocessLevelOneDotDot(t *testing.T) {
	m, _ := NewMinimatch("a/b/../c", Options{})
	if !reflect.DeepEqual(m.GlobParts, [][]string{{"a", "c"}}) {
		t.Fatal(m.GlobParts)
	}
	m, _ = NewMinimatch("a/b/../c", Options{OptimizationLevel: Int(0)})
	if !reflect.DeepEqual(m.GlobParts, [][]string{{"a", "b", "..", "c"}}) {
		t.Fatal(m.GlobParts)
	}
}

func TestPreprocessLevelTwo(t *testing.T) {
	m, _ := NewMinimatch("./a/./b", Options{OptimizationLevel: Int(2)})
	// Reference: [[".","a","b"]]
	if !reflect.DeepEqual(m.GlobParts, [][]string{{".", "a", "b"}}) {
		t.Fatal(m.GlobParts)
	}

	m, _ = NewMinimatch("a/**/../b/c", Options{OptimizationLevel: Int(2)})
	// Reference: [["b","c"],["a","**","b","c"]]
	want := [][]string{{"b", "c"}, {"a", "**", "b", "c"}}
	if !reflect.DeepEqual(m.GlobParts, want) {
		t.Fatalf("got %v want %v", m.GlobParts, want)
	}
}

func TestPreprocessNoGlobStar(t *testing.T) {
	m, _ := NewMinimatch("a/**/b", Options{NoGlobStar: true})
	if !reflect.DeepEqual(m.GlobParts, [][]string{{"a", "*", "b"}}) {
		t.Fatal(m.GlobParts)
	}
}

func TestBraceExpandInPipeline(t *testing.T) {
	m, err := NewMinimatch("a{b,c}d", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.GlobSet, []string{"abd", "acd"}) {
		t.Fatal(m.GlobSet)
	}
	if !reflect.DeepEqual(m.GlobParts, [][]string{{"abd"}, {"acd"}}) {
		t.Fatal(m.GlobParts)
	}
}

func TestWindowsPathsNoEscapeRewrite(t *testing.T) {
	m, _ := NewMinimatch(`a\b\c`, Options{WindowsPathsNoEscape: true})
	if m.Pattern != "a/b/c" {
		t.Fatal(m.Pattern)
	}
}

func TestInvalidPattern(t *testing.T) {
	_, err := NewMinimatch(string(make([]byte, MaxPatternLength+1)), Options{})
	if err != ErrPatternTooLong {
		t.Fatal(err)
	}
}

func TestLevelTwoFileOptimize(t *testing.T) {
	m, _ := NewMinimatch("*", Options{OptimizationLevel: Int(2)})
	got := m.LevelTwoFileOptimize([]string{"a", "b", "..", "c"})
	if !reflect.DeepEqual(got, []string{"a", "c"}) {
		t.Fatal(got)
	}
}

func TestPartsMatchStarDedupe(t *testing.T) {
	m, _ := NewMinimatch("*", Options{OptimizationLevel: Int(2)})
	// a/*/b vs a/x/b → a/*/b
	got := m.partsMatch([]string{"a", "*", "b"}, []string{"a", "x", "b"}, true)
	if !reflect.DeepEqual(got, []string{"a", "*", "b"}) {
		t.Fatal(got)
	}
}
