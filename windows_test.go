package minimatch

import "testing"

// Windows personality tests use Options.Platform = win32 so they run on any host.

func TestWindowsBackslashInPath(t *testing.T) {
	// Path uses \ as separator; pattern uses /
	ok, err := Match(`a\b\c`, "a/b/c", Options{Platform: PlatformWin32})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal(`path a\b\c should match a/b/c on win32`)
	}
}

func TestWindowsPathsNoEscapeInPattern(t *testing.T) {
	ok, err := Match("a/b/c", `a\b\c`, Options{
		Platform:             PlatformWin32,
		WindowsPathsNoEscape: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal(`pattern a\b\c with windowsPathsNoEscape should match a/b/c`)
	}
}

func TestWindowsDriveCaseInsensitive(t *testing.T) {
	o := Options{Platform: PlatformWin32, NoCase: true}
	ok, err := Match("c:/foo/bar", "C:/foo/*", o)
	if err != nil || !ok {
		t.Fatalf("drive case: %v %v", ok, err)
	}
	ok, _ = Match("C:/foo/bar", "C:/foo/*", o)
	if !ok {
		t.Fatal("same case drive")
	}
}

func TestWindowsUNCShare(t *testing.T) {
	o := Options{Platform: PlatformWin32, NoCase: true}
	ok, err := Match("//host/share/a", "//host/share/*", o)
	if err != nil || !ok {
		t.Fatalf("UNC: %v %v", ok, err)
	}
}

func TestWindowsUNCDriveLongPath(t *testing.T) {
	// //?/C:/foo ↔ C:/foo
	o := Options{Platform: PlatformWin32, NoCase: true}
	ok, err := Match("//?/C:/foo", "C:/foo", o)
	if err != nil || !ok {
		t.Fatalf("//?/C:/foo vs C:/foo: %v %v", ok, err)
	}
	ok, err = Match("C:/foo", "//?/C:/foo", o)
	if err != nil || !ok {
		t.Fatalf("C:/foo vs //?/C:/foo: %v %v", ok, err)
	}
}

func TestWindowsSlashSplitUNC(t *testing.T) {
	m, err := NewMinimatch("x", Options{Platform: PlatformWin32})
	if err != nil {
		t.Fatal(err)
	}
	parts := m.SlashSplit("//host/share/x")
	// ["", "", "host", "share", "x"]
	if len(parts) < 4 || parts[0] != "" || parts[1] != "" || parts[2] != "host" {
		t.Fatalf("%q", parts)
	}
}

func TestWindowsNoMagicRootLeavesDriveString(t *testing.T) {
	// win32 + nocase defaults windowsNoMagicRoot
	m, err := NewMinimatch("C:/foo/*", Options{Platform: PlatformWin32, NoCase: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Set) == 0 || len(m.Set[0]) == 0 {
		t.Fatal("empty set")
	}
	// First segment should remain a string "C:" (or similar), not a case-folded RE only
	if !m.Set[0][0].isStringPart() {
		t.Fatalf("drive root should be string part, got %+v", m.Set[0][0])
	}
}

func TestWindowsGlobPartsLongUNC(t *testing.T) {
	m, err := NewMinimatch("//?/C:/**/*.txt", Options{Platform: PlatformWin32, NoCase: true})
	if err != nil {
		t.Fatal(err)
	}
	// Expect UNC prefix preserved: "", "", "?", "C:", ...
	if len(m.GlobParts) == 0 || len(m.GlobParts[0]) < 4 {
		t.Fatalf("globParts %v", m.GlobParts)
	}
	gp := m.GlobParts[0]
	if gp[0] != "" || gp[1] != "" || gp[2] != "?" {
		t.Fatalf("UNC prefix %v", gp[:3])
	}
}
