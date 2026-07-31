package minimatch

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestValidatePatternEmpty(t *testing.T) {
	if err := ValidatePattern(""); err != nil {
		t.Fatalf("empty pattern must be valid, got %v", err)
	}
}

func TestValidatePatternAcceptsMaxLength(t *testing.T) {
	// JS: pattern.length > MAX → throw; equality is allowed.
	p := strings.Repeat("x", MaxPatternLength)
	if err := ValidatePattern(p); err != nil {
		t.Fatalf("pattern of exactly MaxPatternLength must be valid, got %v", err)
	}
}

func TestValidatePatternRejectsTooLong(t *testing.T) {
	// Matches test/basic.js: 'x'.repeat(64 * 1024) + 'y'
	p := strings.Repeat("x", MaxPatternLength) + "y"
	err := ValidatePattern(p)
	if !errors.Is(err, ErrPatternTooLong) {
		t.Fatalf("expected ErrPatternTooLong, got %v", err)
	}
	if err.Error() != "pattern is too long" {
		t.Fatalf("error message must match TypeScript, got %q", err.Error())
	}
}

func TestValidatePatternRedosTooLong(t *testing.T) {
	// test/redos.js builds '!(' + genstr(1024*64, '\\') + 'A)'
	// genstr(len) loops i from 0..len inclusive → len+1 chars of '\\'.
	exploit := "!(" + strings.Repeat("\\", 1024*64+1) + "A)"
	err := ValidatePattern(exploit)
	if !errors.Is(err, ErrPatternTooLong) {
		t.Fatalf("expected ErrPatternTooLong for redos-length exploit, got %v", err)
	}
}

func TestValidatePatternAcceptsLongButUnderLimit(t *testing.T) {
	// test/redos.js patterns of size ~15KiB must still validate.
	exploit := "!(" + strings.Repeat("\\", 1024*15+1) + "A)"
	if err := ValidatePattern(exploit); err != nil {
		t.Fatalf("under-limit pattern must be valid, got %v", err)
	}
}

func TestValidatePatternUTF16LengthNotBytes(t *testing.T) {
	// A single emoji is 1 code point, 2 UTF-16 code units, 4 UTF-8 bytes.
	// The limit must use UTF-16 units (JS String.length), not Go len().
	emoji := "😀" // U+1F600
	if utf16.RuneLen([]rune(emoji)[0]) != 2 {
		t.Fatal("test setup: emoji must be supplementary plane")
	}
	// Floor(MaxPatternLength/2) emojis → exactly MaxPatternLength UTF-16 units
	// if MaxPatternLength is even (it is).
	n := MaxPatternLength / 2
	p := strings.Repeat(emoji, n)
	if got := utf16Len(p); got != MaxPatternLength {
		t.Fatalf("utf16Len setup: got %d want %d", got, MaxPatternLength)
	}
	if err := ValidatePattern(p); err != nil {
		t.Fatalf("UTF-16-exact max length must be valid, got %v", err)
	}
	if err := ValidatePattern(p + "y"); !errors.Is(err, ErrPatternTooLong) {
		t.Fatalf("one unit over UTF-16 max must be too long, got %v", err)
	}
}

func TestErrInvalidPatternMessage(t *testing.T) {
	// Parity with TypeScript TypeError('invalid pattern') message text.
	if ErrInvalidPattern.Error() != "invalid pattern" {
		t.Fatalf("ErrInvalidPattern message: got %q", ErrInvalidPattern.Error())
	}
}

func TestMaxPatternLengthValue(t *testing.T) {
	if MaxPatternLength != 1024*64 {
		t.Fatalf("MaxPatternLength: got %d want %d", MaxPatternLength, 1024*64)
	}
}

func TestUTF16LenASCII(t *testing.T) {
	if got := utf16Len("abc"); got != 3 {
		t.Fatalf("utf16Len(abc)=%d", got)
	}
}

func TestUTF16LenEmpty(t *testing.T) {
	if got := utf16Len(""); got != 0 {
		t.Fatalf("utf16Len(empty)=%d", got)
	}
}
