package minimatch

import (
	"runtime"
	"testing"
)

func TestHostPlatformWindowsMapping(t *testing.T) {
	p := HostPlatform()
	if runtime.GOOS == "windows" {
		if p != PlatformWin32 {
			t.Fatalf("HostPlatform on windows: got %q want %q", p, PlatformWin32)
		}
		if !p.IsWindows() {
			t.Fatal("expected IsWindows")
		}
		if p.PathSep() != SepWindows {
			t.Fatalf("PathSep: got %q", p.PathSep())
		}
		return
	}
	if p.IsWindows() {
		t.Fatalf("non-windows HostPlatform unexpectedly win32: %q", p)
	}
	if p.PathSep() != SepPOSIX {
		t.Fatalf("PathSep: got %q want %q", p.PathSep(), SepPOSIX)
	}
}

func TestEffectiveOptimizationLevel(t *testing.T) {
	var o Options
	if got := o.EffectiveOptimizationLevel(); got != DefaultOptimizationLevel {
		t.Fatalf("nil OptimizationLevel: got %d want %d", got, DefaultOptimizationLevel)
	}
	zero := 0
	o.OptimizationLevel = &zero
	if got := o.EffectiveOptimizationLevel(); got != 0 {
		t.Fatalf("explicit 0: got %d want 0", got)
	}
	two := 2
	o.OptimizationLevel = &two
	if got := o.EffectiveOptimizationLevel(); got != 2 {
		t.Fatalf("explicit 2: got %d want 2", got)
	}
}

func TestEffectiveNumericDefaults(t *testing.T) {
	var o Options
	if o.EffectiveMaxGlobstarRecursion() != DefaultMaxGlobstarRecursion {
		t.Fatal("MaxGlobstarRecursion default")
	}
	if o.EffectiveMaxExtglobRecursion() != DefaultMaxExtglobRecursion {
		t.Fatal("MaxExtglobRecursion default")
	}
	if o.EffectiveBraceExpandMax() != DefaultBraceExpandMax {
		t.Fatal("BraceExpandMax default")
	}
	n := 3
	o.MaxExtglobRecursion = &n
	if o.EffectiveMaxExtglobRecursion() != 3 {
		t.Fatal("explicit MaxExtglobRecursion")
	}
}

func TestEffectiveWindowsPathsNoEscape(t *testing.T) {
	var o Options
	if o.EffectiveWindowsPathsNoEscape() {
		t.Fatal("default should be false")
	}
	o.WindowsPathsNoEscape = true
	if !o.EffectiveWindowsPathsNoEscape() {
		t.Fatal("WindowsPathsNoEscape true")
	}
	o = Options{}
	f := false
	o.AllowWindowsEscape = &f
	if !o.EffectiveWindowsPathsNoEscape() {
		t.Fatal("AllowWindowsEscape false forces windows paths no escape")
	}
	tr := true
	o.AllowWindowsEscape = &tr
	if o.EffectiveWindowsPathsNoEscape() {
		t.Fatal("AllowWindowsEscape true should not force it alone")
	}
}

func TestEffectiveWindowsNoMagicRoot(t *testing.T) {
	o := Options{Platform: PlatformLinux, NoCase: true}
	if o.EffectiveWindowsNoMagicRoot() {
		t.Fatal("non-win32 should default false")
	}
	o.Platform = PlatformWin32
	if !o.EffectiveWindowsNoMagicRoot() {
		t.Fatal("win32+NoCase should default true")
	}
	f := false
	o.WindowsNoMagicRoot = &f
	if o.EffectiveWindowsNoMagicRoot() {
		t.Fatal("explicit false")
	}
}

func TestDefaultConstants(t *testing.T) {
	if DefaultOptimizationLevel != 1 {
		t.Fatal(DefaultOptimizationLevel)
	}
	if DefaultMaxGlobstarRecursion != 200 {
		t.Fatal(DefaultMaxGlobstarRecursion)
	}
	if DefaultMaxExtglobRecursion != 2 {
		t.Fatal(DefaultMaxExtglobRecursion)
	}
	if DefaultBraceExpandMax != 100_000 {
		t.Fatal(DefaultBraceExpandMax)
	}
}
