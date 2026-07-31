package minimatch

import "runtime"

// Platform identifies the operating system personality that controls
// Windows-specific path behaviour (UNC paths, backslash handling, drive
// letters, windowsNoMagicRoot defaults).
//
// Values match Node.js process.platform strings from the TypeScript API,
// not necessarily Go's runtime.GOOS (in particular Windows is "win32").
type Platform string

// Platform constants corresponding to the TypeScript Platform union.
const (
	PlatformAIX     Platform = "aix"
	PlatformAndroid Platform = "android"
	PlatformDarwin  Platform = "darwin"
	PlatformFreeBSD Platform = "freebsd"
	PlatformHaiku   Platform = "haiku"
	PlatformLinux   Platform = "linux"
	PlatformOpenBSD Platform = "openbsd"
	PlatformSunOS   Platform = "sunos"
	PlatformWin32   Platform = "win32"
	PlatformCygwin  Platform = "cygwin"
	PlatformNetBSD  Platform = "netbsd"
)

// Sep is a path separator character used when reporting the active
// separator for the default platform (TypeScript minimatch.sep).
type Sep string

// Path separator constants.
const (
	// SepPOSIX is the forward slash used on non-Windows platforms.
	SepPOSIX Sep = "/"
	// SepWindows is the backslash used when the platform is win32.
	SepWindows Sep = `\`
)

// HostPlatform returns the Platform value for the running operating system.
//
// Go's runtime.GOOS uses "windows"; the TypeScript API uses "win32".
// This function maps that difference so Windows-specific behaviour aligns
// with the reference implementation. Other GOOS values are returned as
// Platform(GOOS) when they match a Node platform string; unknown systems
// are returned as their GOOS string without special handling (only win32
// changes matching semantics in the reference).
func HostPlatform() Platform {
	if runtime.GOOS == "windows" {
		return PlatformWin32
	}
	return Platform(runtime.GOOS)
}

// IsWindows reports whether p is the Windows platform (Node "win32").
func (p Platform) IsWindows() bool {
	return p == PlatformWin32
}

// PathSep returns the path separator associated with p.
// Non-Windows platforms use SepPOSIX.
func (p Platform) PathSep() Sep {
	if p.IsWindows() {
		return SepWindows
	}
	return SepPOSIX
}
