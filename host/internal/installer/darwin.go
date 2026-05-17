//go:build darwin

package installer

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// runLaunchctl is overridable so tests don't actually invoke launchctl.
var runLaunchctl = func(args ...string) error {
	cmd := exec.Command("launchctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

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
	bundleRoot := filepath.Join(home, "Applications", "Hackintosh.app")
	infoPlist := filepath.Join(bundleRoot, "Contents", "Info.plist")
	launchAgent := filepath.Join(home, "Library", "LaunchAgents", darwinBundleID+".plist")

	// 1. Build the .app bundle directory structure.
	for _, d := range []string{
		filepath.Join(bundleRoot, "Contents", "MacOS"),
		filepath.Join(bundleRoot, "Contents", "Resources"),
	} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return r, formatPath(d, err)
		}
	}

	// 2. Info.plist.
	if err := os.WriteFile(infoPlist, []byte(buildInfoPlist()), 0644); err != nil {
		return r, formatPath(infoPlist, err)
	}
	r.record(infoPlist, "wrote Info.plist at "+infoPlist)

	// 3. Copy the running binary into MacOS/.
	if err := copyExecutableTo(binPath); err != nil {
		return r, formatPath(binPath, err)
	}
	if err := os.Chmod(binPath, 0755); err != nil {
		return r, formatPath(binPath, err)
	}
	r.record(binPath, "installed binary at "+binPath)

	// 4. LaunchAgent plist.
	if err := os.MkdirAll(filepath.Dir(launchAgent), 0755); err != nil {
		return r, formatPath(filepath.Dir(launchAgent), err)
	}
	if err := os.WriteFile(launchAgent, []byte(buildLaunchAgentPlist(binPath)), 0644); err != nil {
		return r, formatPath(launchAgent, err)
	}
	r.record(launchAgent, "wrote LaunchAgent at "+launchAgent)

	// 5. Bootstrap launchctl. If it fails, continue -- file is in place.
	uid := os.Getuid()
	domain := fmt.Sprintf("gui/%d", uid)
	if err := runLaunchctl("bootstrap", domain, launchAgent); err != nil {
		r.noteOnly(fmt.Sprintf("warning: launchctl bootstrap failed (autostart will activate on next login): %v", err))
	} else {
		r.noteOnly("launchctl bootstrap succeeded for " + domain)
	}

	return r, nil
}

func doUninstall() (Result, error) {
	var r Result
	home, err := userHome()
	if err != nil {
		return r, err
	}

	uid := os.Getuid()
	domain := fmt.Sprintf("gui/%d", uid)
	target := domain + "/" + darwinBundleID
	if err := runLaunchctl("bootout", target); err != nil {
		r.noteOnly(fmt.Sprintf("launchctl bootout %s (ignored if not loaded): %v", target, err))
	}

	for _, p := range []string{
		filepath.Join(home, "Library", "LaunchAgents", darwinBundleID+".plist"),
	} {
		removed, err := removeIfExists(p)
		if err != nil {
			return r, formatPath(p, err)
		}
		if removed {
			r.record(p, "removed "+p)
		}
	}

	// Remove the .app bundle directory.
	bundleRoot := filepath.Join(home, "Applications", "Hackintosh.app")
	if _, err := os.Stat(bundleRoot); err == nil {
		if err := os.RemoveAll(bundleRoot); err != nil {
			return r, formatPath(bundleRoot, err)
		}
		r.record(bundleRoot, "removed "+bundleRoot)
	}

	return r, nil
}

// --- shared helpers (same shape as the other platforms) ---

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
