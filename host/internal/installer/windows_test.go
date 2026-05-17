//go:build windows

package installer

import (
	"os"
	"path/filepath"
	"testing"
)

func setEnvForTest(t *testing.T, key, val string) {
	t.Helper()
	orig := os.Getenv(key)
	os.Setenv(key, val)
	t.Cleanup(func() { os.Setenv(key, orig) })
}

func setHomeForTest(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	orig := userHome
	userHome = func() (string, error) { return home, nil }
	t.Cleanup(func() { userHome = orig })

	setEnvForTest(t, "LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))
	setEnvForTest(t, "APPDATA", filepath.Join(home, "AppData", "Roaming"))
	return home
}

func TestWindowsInstallCopiesBinary(t *testing.T) {
	t.Skip("modifies HKCU registry; verify manually")

	home := setHomeForTest(t)

	res, err := doInstall()
	if err != nil {
		t.Fatalf("doInstall: %v", err)
	}

	binPath := filepath.Join(home, "AppData", "Local", "Hackintosh", "hackintosh.exe")
	if _, err := os.Stat(binPath); err != nil {
		t.Errorf("binary not at %s: %v", binPath, err)
	}

	if len(res.Files) == 0 {
		t.Error("Result.Files should not be empty")
	}
}

func TestWindowsUninstallIsIdempotent(t *testing.T) {
	setHomeForTest(t)
	if _, err := doUninstall(); err != nil {
		t.Fatalf("uninstall on fresh home: %v", err)
	}
}
