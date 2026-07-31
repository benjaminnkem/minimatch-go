package minimatch

import (
	"runtime"
	"testing"
)

// zeroValueBools lists every bool field that must be false on Options{}.
// If a new bool option is added to Options, add it here.
func TestZeroValueMatchesTypeScriptDefaults(t *testing.T) {
	var o Options

	bools := []struct {
		name string
		v    bool
	}{
		{"NoBrace", o.NoBrace},
		{"NoComment", o.NoComment},
		{"NoNegate", o.NoNegate},
		{"Debug", o.Debug},
		{"NoGlobStar", o.NoGlobStar},
		{"NoExt", o.NoExt},
		{"NoNull", o.NoNull},
		{"WindowsPathsNoEscape", o.WindowsPathsNoEscape},
		{"Partial", o.Partial},
		{"Dot", o.Dot},
		{"NoCase", o.NoCase},
		{"NoCaseMagicOnly", o.NoCaseMagicOnly},
		{"MagicalBraces", o.MagicalBraces},
		{"MatchBase", o.MatchBase},
		{"FlipNegate", o.FlipNegate},
		{"PreserveMultipleSlashes", o.PreserveMultipleSlashes},
	}
	for _, b := range bools {
		if b.v {
			t.Errorf("zero Options.%s = true, want false (TS default)", b.name)
		}
	}

	if o.AllowWindowsEscape != nil {
		t.Fatal("AllowWindowsEscape zero value must be nil (undefined)")
	}
	if o.OptimizationLevel != nil {
		t.Fatal("OptimizationLevel zero value must be nil")
	}
	if o.Platform != "" {
		t.Fatal("Platform zero value must be empty")
	}
	if o.WindowsNoMagicRoot != nil {
		t.Fatal("WindowsNoMagicRoot zero value must be nil")
	}
	if o.BraceExpandMax != nil {
		t.Fatal("BraceExpandMax zero value must be nil")
	}
	if o.MaxGlobstarRecursion != nil {
		t.Fatal("MaxGlobstarRecursion zero value must be nil")
	}
	if o.MaxExtglobRecursion != nil {
		t.Fatal("MaxExtglobRecursion zero value must be nil")
	}
}

func TestEffectiveDefaultsOnZeroOptions(t *testing.T) {
	var o Options

	if got := o.EffectiveOptimizationLevel(); got != DefaultOptimizationLevel {
		t.Fatalf("EffectiveOptimizationLevel: got %d want %d", got, DefaultOptimizationLevel)
	}
	if got := o.EffectiveMaxGlobstarRecursion(); got != DefaultMaxGlobstarRecursion {
		t.Fatalf("EffectiveMaxGlobstarRecursion: got %d want %d", got, DefaultMaxGlobstarRecursion)
	}
	if got := o.EffectiveMaxExtglobRecursion(); got != DefaultMaxExtglobRecursion {
		t.Fatalf("EffectiveMaxExtglobRecursion: got %d want %d", got, DefaultMaxExtglobRecursion)
	}
	if got := o.EffectiveBraceExpandMax(); got != DefaultBraceExpandMax {
		t.Fatalf("EffectiveBraceExpandMax: got %d want %d", got, DefaultBraceExpandMax)
	}
	if o.EffectiveWindowsPathsNoEscape() {
		t.Fatal("EffectiveWindowsPathsNoEscape: want false")
	}
	if o.EffectiveWindowsNoMagicRoot() {
		t.Fatal("EffectiveWindowsNoMagicRoot on zero opts: want false")
	}
	if o.EffectivePlatform() != HostPlatform() {
		t.Fatalf("EffectivePlatform: got %q want %q", o.EffectivePlatform(), HostPlatform())
	}
}

func TestEffectiveOptimizationLevelExplicitZero(t *testing.T) {
	o := Options{OptimizationLevel: Int(0)}
	if got := o.EffectiveOptimizationLevel(); got != 0 {
		t.Fatalf("explicit 0: got %d", got)
	}
	o.OptimizationLevel = Int(2)
	if got := o.EffectiveOptimizationLevel(); got != 2 {
		t.Fatalf("explicit 2: got %d", got)
	}
}

func TestEffectiveNumericOverrides(t *testing.T) {
	o := Options{
		MaxGlobstarRecursion: Int(3),
		MaxExtglobRecursion:  Int(9),
		BraceExpandMax:       Int(42),
	}
	if o.EffectiveMaxGlobstarRecursion() != 3 {
		t.Fatal("MaxGlobstarRecursion")
	}
	if o.EffectiveMaxExtglobRecursion() != 9 {
		t.Fatal("MaxExtglobRecursion")
	}
	if o.EffectiveBraceExpandMax() != 42 {
		t.Fatal("BraceExpandMax")
	}
}

func TestEffectiveWindowsPathsNoEscape(t *testing.T) {
	if (Options{}).EffectiveWindowsPathsNoEscape() {
		t.Fatal("default")
	}
	if !(Options{WindowsPathsNoEscape: true}).EffectiveWindowsPathsNoEscape() {
		t.Fatal("WindowsPathsNoEscape true")
	}
	// allowWindowsEscape === false forces the flag on
	if !(Options{AllowWindowsEscape: Bool(false)}).EffectiveWindowsPathsNoEscape() {
		t.Fatal("AllowWindowsEscape false")
	}
	// allowWindowsEscape === true does not force it on
	if (Options{AllowWindowsEscape: Bool(true)}).EffectiveWindowsPathsNoEscape() {
		t.Fatal("AllowWindowsEscape true alone")
	}
	// either path sets effective true
	o := Options{WindowsPathsNoEscape: true, AllowWindowsEscape: Bool(true)}
	if !o.EffectiveWindowsPathsNoEscape() {
		t.Fatal("WindowsPathsNoEscape wins with Allow true")
	}
}

func TestEffectiveWindowsNoMagicRoot(t *testing.T) {
	// default false on non-windows even with NoCase
	o := Options{Platform: PlatformLinux, NoCase: true}
	if o.EffectiveWindowsNoMagicRoot() {
		t.Fatal("linux+NoCase should default false")
	}
	// default true on win32+NoCase
	o = Options{Platform: PlatformWin32, NoCase: true}
	if !o.EffectiveWindowsNoMagicRoot() {
		t.Fatal("win32+NoCase should default true")
	}
	// win32 without NoCase → false
	o = Options{Platform: PlatformWin32, NoCase: false}
	if o.EffectiveWindowsNoMagicRoot() {
		t.Fatal("win32 without NoCase should default false")
	}
	// explicit false overrides default true
	o = Options{Platform: PlatformWin32, NoCase: true, WindowsNoMagicRoot: Bool(false)}
	if o.EffectiveWindowsNoMagicRoot() {
		t.Fatal("explicit false")
	}
	// explicit true overrides default false
	o = Options{Platform: PlatformLinux, WindowsNoMagicRoot: Bool(true)}
	if !o.EffectiveWindowsNoMagicRoot() {
		t.Fatal("explicit true")
	}
}

func TestEffectivePlatformOverride(t *testing.T) {
	o := Options{Platform: PlatformWin32}
	if o.EffectivePlatform() != PlatformWin32 {
		t.Fatal("override")
	}
	if !o.EffectiveIsWindows() {
		t.Fatal("EffectiveIsWindows")
	}
	o.Platform = PlatformDarwin
	if o.EffectiveIsWindows() {
		t.Fatal("darwin is not windows")
	}
}

func TestHostPlatformWindowsMapping(t *testing.T) {
	p := HostPlatform()
	if runtime.GOOS == "windows" {
		if p != PlatformWin32 {
			t.Fatalf("HostPlatform on windows: got %q want %q", p, PlatformWin32)
		}
		return
	}
	if p.IsWindows() {
		t.Fatalf("non-windows HostPlatform unexpectedly win32: %q", p)
	}
}

func TestBoolIntHelpers(t *testing.T) {
	if Bool(true) == nil || !*Bool(true) {
		t.Fatal("Bool(true)")
	}
	if Bool(false) == nil || *Bool(false) {
		t.Fatal("Bool(false)")
	}
	if Int(0) == nil || *Int(0) != 0 {
		t.Fatal("Int(0)")
	}
	o := Options{OptimizationLevel: Int(0), AllowWindowsEscape: Bool(false)}
	if o.EffectiveOptimizationLevel() != 0 {
		t.Fatal("Int helper on Options")
	}
	if !o.EffectiveWindowsPathsNoEscape() {
		t.Fatal("Bool helper on Options")
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

// Ensure every TypeScript MinimatchOptions key has a corresponding field
// by listing the TS → Go map. This test fails to compile if renamed.
func TestTypeScriptOptionFieldMap(t *testing.T) {
	// Touch every field so refactors that drop a field are noticed.
	o := Options{
		NoBrace:                 true,          // nobrace
		NoComment:               true,          // nocomment
		NoNegate:                true,          // nonegate
		Debug:                   true,          // debug
		NoGlobStar:              true,          // noglobstar
		NoExt:                   true,          // noext
		NoNull:                  true,          // nonull
		WindowsPathsNoEscape:    true,          // windowsPathsNoEscape
		AllowWindowsEscape:      Bool(false),   // allowWindowsEscape
		Partial:                 true,          // partial
		Dot:                     true,          // dot
		NoCase:                  true,          // nocase
		NoCaseMagicOnly:         true,          // nocaseMagicOnly
		MagicalBraces:           true,          // magicalBraces
		MatchBase:               true,          // matchBase
		FlipNegate:              true,          // flipNegate
		PreserveMultipleSlashes: true,          // preserveMultipleSlashes
		OptimizationLevel:       Int(1),        // optimizationLevel
		Platform:                PlatformLinux, // platform
		WindowsNoMagicRoot:      Bool(true),    // windowsNoMagicRoot
		BraceExpandMax:          Int(1),        // braceExpandMax
		MaxGlobstarRecursion:    Int(1),        // maxGlobstarRecursion
		MaxExtglobRecursion:     Int(1),        // maxExtglobRecursion
	}
	_ = o
}
