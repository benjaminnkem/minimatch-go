package minimatch

import "testing"

func TestPlatformConstantsMatchNode(t *testing.T) {
	// Values must match the TypeScript Platform string union exactly.
	cases := []struct {
		p    Platform
		want string
	}{
		{PlatformAIX, "aix"},
		{PlatformAndroid, "android"},
		{PlatformDarwin, "darwin"},
		{PlatformFreeBSD, "freebsd"},
		{PlatformHaiku, "haiku"},
		{PlatformLinux, "linux"},
		{PlatformOpenBSD, "openbsd"},
		{PlatformSunOS, "sunos"},
		{PlatformWin32, "win32"},
		{PlatformCygwin, "cygwin"},
		{PlatformNetBSD, "netbsd"},
	}
	for _, tc := range cases {
		if string(tc.p) != tc.want {
			t.Fatalf("%v: got %q want %q", tc.p, tc.p, tc.want)
		}
	}
}

func TestSepConstants(t *testing.T) {
	if SepPOSIX != "/" {
		t.Fatalf("SepPOSIX=%q", SepPOSIX)
	}
	if SepWindows != `\` {
		t.Fatalf("SepWindows=%q", SepWindows)
	}
}
