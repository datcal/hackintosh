//go:build linux

package installer

import (
	"io"
	"os"
	"path/filepath"
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
	iconPath := filepath.Join(home, ".local", "share", "hackintosh", "icon.png")
	autostart := filepath.Join(home, ".config", "autostart", "hackintosh.desktop")
	launcher := filepath.Join(home, ".local", "share", "applications", "hackintosh.desktop")

	// 1. Copy the running binary to ~/.local/bin/hackintosh.
	if err := copyExecutableTo(binPath); err != nil {
		return r, formatPath(binPath, err)
	}
	r.record(binPath, "installed binary at "+binPath)

	// 2. Drop an icon next to it.
	if err := writeIcon(iconPath); err != nil {
		return r, formatPath(iconPath, err)
	}
	r.record(iconPath, "wrote icon at "+iconPath)

	// 3. Autostart .desktop.
	entry := buildDesktopEntry(binPath, iconPath)
	if err := writeFile(autostart, []byte(entry), 0644); err != nil {
		return r, formatPath(autostart, err)
	}
	r.record(autostart, "wrote autostart entry at "+autostart)

	// 4. App menu .desktop (identical content).
	if err := writeFile(launcher, []byte(entry), 0644); err != nil {
		return r, formatPath(launcher, err)
	}
	r.record(launcher, "wrote launcher entry at "+launcher)

	return r, nil
}

func doUninstall() (Result, error) {
	var r Result
	home, err := userHome()
	if err != nil {
		return r, err
	}

	for _, p := range []string{
		filepath.Join(home, ".config", "autostart", "hackintosh.desktop"),
		filepath.Join(home, ".local", "share", "applications", "hackintosh.desktop"),
		filepath.Join(home, ".local", "share", "hackintosh", "icon.png"),
		filepath.Join(home, ".local", "bin", "hackintosh"),
	} {
		removed, err := removeIfExists(p)
		if err != nil {
			return r, formatPath(p, err)
		}
		if removed {
			r.record(p, "removed "+p)
		}
	}
	return r, nil
}

// --- small filesystem helpers ---

func writeFile(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, mode)
}

func writeIcon(path string) error {
	return writeFile(path, trayIconBytes(), 0644)
}

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
