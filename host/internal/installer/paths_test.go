package installer

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallBinaryPathReturnsAbsPath(t *testing.T) {
	p, err := installBinaryPath("/home/u")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(p) {
		t.Fatalf("installBinaryPath: %q is not absolute", p)
	}
}

func TestInstallBinaryPathIsOSAppropriate(t *testing.T) {
	p, err := installBinaryPath("/home/u")
	if err != nil {
		t.Fatal(err)
	}
	switch runtime.GOOS {
	case "windows":
		if !strings.HasSuffix(strings.ToLower(p), "hackintosh.exe") {
			t.Fatalf("windows install path %q should end with hackintosh.exe", p)
		}
	case "darwin":
		if !strings.Contains(p, ".app/Contents/MacOS/Hackintosh") {
			t.Fatalf("darwin install path %q should live inside Hackintosh.app/Contents/MacOS", p)
		}
	case "linux":
		if !strings.HasSuffix(p, "/.local/bin/hackintosh") {
			t.Fatalf("linux install path %q should end with /.local/bin/hackintosh", p)
		}
	}
}
