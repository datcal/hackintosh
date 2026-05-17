//go:build linux

package installer

import (
	"os"
	"path/filepath"
	"testing"
)

func setHomeForTest(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	orig := userHome
	userHome = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHome = orig })
	return home
}

func TestLinuxInstallWritesAllExpectedFiles(t *testing.T) {
	home := setHomeForTest(t)

	res, err := doInstall()
	if err != nil {
		t.Fatalf("doInstall: %v", err)
	}

	expectedFiles := []string{
		filepath.Join(home, ".local", "bin", "hackintosh"),
		filepath.Join(home, ".local", "share", "hackintosh", "icon.png"),
		filepath.Join(home, ".config", "autostart", "hackintosh.desktop"),
		filepath.Join(home, ".local", "share", "applications", "hackintosh.desktop"),
	}
	for _, p := range expectedFiles {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected file missing after install: %s (%v)", p, err)
		}
	}
	if len(res.Files) != len(expectedFiles) {
		t.Errorf("Result.Files: got %d, want %d (%v)", len(res.Files), len(expectedFiles), res.Files)
	}
}

func TestLinuxUninstallRemovesAllFiles(t *testing.T) {
	home := setHomeForTest(t)
	if _, err := doInstall(); err != nil {
		t.Fatalf("doInstall: %v", err)
	}

	if _, err := doUninstall(); err != nil {
		t.Fatalf("doUninstall: %v", err)
	}

	for _, rel := range []string{
		".config/autostart/hackintosh.desktop",
		".local/share/applications/hackintosh.desktop",
		".local/bin/hackintosh",
	} {
		p := filepath.Join(home, rel)
		if _, err := os.Stat(p); err == nil {
			t.Errorf("expected %s removed, but it still exists", p)
		}
	}
}

func TestLinuxInstallIsIdempotent(t *testing.T) {
	setHomeForTest(t)
	if _, err := doInstall(); err != nil {
		t.Fatalf("first doInstall: %v", err)
	}
	if _, err := doInstall(); err != nil {
		t.Fatalf("second doInstall (should be idempotent): %v", err)
	}
}

func TestLinuxUninstallIsIdempotent(t *testing.T) {
	setHomeForTest(t)
	if _, err := doUninstall(); err != nil {
		t.Fatalf("doUninstall on fresh home: %v", err)
	}
}
