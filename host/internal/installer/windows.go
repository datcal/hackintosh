//go:build windows

package installer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const (
	runKeyPath   = `Software\Microsoft\Windows\CurrentVersion\Run`
	runKeyName   = "Hackintosh"
	shortcutName = "Hackintosh.lnk"
)

func doInstall() (Result, error) {
	var r Result
	home, err := userHome()
	if err != nil {
		return r, err
	}

	binPath, err := installBinaryPath(home)
	if err != nil {
		return r, err
	}

	// 1. Copy the running binary into %LOCALAPPDATA%\Hackintosh\.
	if err := copyExecutableTo(binPath); err != nil {
		return r, formatPath(binPath, err)
	}
	r.record(binPath, "installed binary at "+binPath)

	// 2. Write the registry Run value (per-user autostart).
	if err := writeRunKey(binPath); err != nil {
		return r, fmt.Errorf("registry write: %w", err)
	}
	r.noteOnly(fmt.Sprintf(`set HKCU\%s\%s = %s`, runKeyPath, runKeyName, binPath))

	// 3. Create the Start Menu shortcut via PowerShell COM.
	startMenu := os.Getenv("APPDATA")
	if startMenu == "" {
		return r, fmt.Errorf("APPDATA env var is empty")
	}
	lnkPath := filepath.Join(startMenu, "Microsoft", "Windows", "Start Menu", "Programs", shortcutName)
	if err := createShortcut(lnkPath, binPath); err != nil {
		return r, fmt.Errorf("shortcut: %w", err)
	}
	r.record(lnkPath, "created Start Menu shortcut at "+lnkPath)

	return r, nil
}

func doUninstall() (Result, error) {
	var r Result
	home, err := userHome()
	if err != nil {
		return r, err
	}

	// 1. Remove the registry value (ignore not-found).
	if err := deleteRunKey(); err != nil {
		return r, fmt.Errorf("registry delete: %w", err)
	}
	r.noteOnly(fmt.Sprintf(`removed HKCU\%s\%s`, runKeyPath, runKeyName))

	// 2. Remove the shortcut.
	startMenu := os.Getenv("APPDATA")
	if startMenu != "" {
		lnkPath := filepath.Join(startMenu, "Microsoft", "Windows", "Start Menu", "Programs", shortcutName)
		removed, err := removeIfExists(lnkPath)
		if err != nil {
			return r, formatPath(lnkPath, err)
		}
		if removed {
			r.record(lnkPath, "removed "+lnkPath)
		}
	}

	// 3. Remove the binary.
	binPath, err := installBinaryPath(home)
	if err == nil {
		removed, err := removeIfExists(binPath)
		if err != nil {
			return r, formatPath(binPath, err)
		}
		if removed {
			r.record(binPath, "removed "+binPath)
		}
	}

	return r, nil
}

func writeRunKey(value string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(runKeyName, value)
}

func deleteRunKey() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}
	defer k.Close()
	if err := k.DeleteValue(runKeyName); err != nil {
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}
	return nil
}

func createShortcut(lnkPath, target string) error {
	if err := os.MkdirAll(filepath.Dir(lnkPath), 0755); err != nil {
		return err
	}
	script := buildShortcutScript(lnkPath, target)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", "-")
	cmd.Stdin = bytes.NewBufferString(script)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}
	return nil
}

// --- shared helpers (same shape as linux.go's copies; lives here for the
// Windows build because Go won't let two build-tagged files declare the same
// function) ---

func copyExecutableTo(dst string) error {
	src, err := os.Executable()
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func removeIfExists(path string) (bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if err := os.Remove(path); err != nil {
		return false, err
	}
	return true, nil
}
