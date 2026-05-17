//go:build darwin

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

	origRun := runLaunchctl
	runLaunchctl = func(args ...string) error { return nil }
	t.Cleanup(func() { runLaunchctl = origRun })
	return home
}

func TestDarwinInstallCreatesAppBundleAndPlist(t *testing.T) {
	home := setHomeForTest(t)
	if _, err := doInstall(); err != nil {
		t.Fatalf("doInstall: %v", err)
	}

	expectedFiles := []string{
		filepath.Join(home, "Applications", "Hackintosh.app", "Contents", "Info.plist"),
		filepath.Join(home, "Applications", "Hackintosh.app", "Contents", "MacOS", "Hackintosh"),
		filepath.Join(home, "Library", "LaunchAgents", "com.datcal.hackintosh.plist"),
	}
	for _, p := range expectedFiles {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected file missing after install: %s (%v)", p, err)
		}
	}
}

func TestDarwinUninstallRemovesBundleAndPlist(t *testing.T) {
	home := setHomeForTest(t)
	if _, err := doInstall(); err != nil {
		t.Fatalf("doInstall: %v", err)
	}
	if _, err := doUninstall(); err != nil {
		t.Fatalf("doUninstall: %v", err)
	}

	for _, rel := range []string{
		"Applications/Hackintosh.app",
		"Library/LaunchAgents/com.datcal.hackintosh.plist",
	} {
		p := filepath.Join(home, rel)
		if _, err := os.Stat(p); err == nil {
			t.Errorf("expected %s removed, still exists", p)
		}
	}
}

func TestDarwinInstallIsIdempotent(t *testing.T) {
	setHomeForTest(t)
	if _, err := doInstall(); err != nil {
		t.Fatalf("first install: %v", err)
	}
	if _, err := doInstall(); err != nil {
		t.Fatalf("second install: %v", err)
	}
}

func TestDarwinUninstallIsIdempotent(t *testing.T) {
	setHomeForTest(t)
	if _, err := doUninstall(); err != nil {
		t.Fatalf("uninstall on fresh home: %v", err)
	}
}
