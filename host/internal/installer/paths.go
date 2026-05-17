package installer

import (
	"os"
	"path/filepath"
	"runtime"
)

// installBinaryPath returns the absolute path where the installed binary
// should live for this OS. The home arg is the user's home dir (passed
// explicitly so it's testable on hosts without that home).
func installBinaryPath(home string) (string, error) {
	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(base, "Hackintosh", "hackintosh.exe"), nil
	case "darwin":
		return filepath.Join(home, "Applications", "Hackintosh.app", "Contents", "MacOS", "Hackintosh"), nil
	case "linux":
		return filepath.Join(home, ".local", "bin", "hackintosh"), nil
	}
	return "", errUnsupportedOS
}

// userHome wraps os.UserHomeDir so callers can override it in tests.
var userHome = os.UserHomeDir

var errUnsupportedOS = &unsupportedOSError{}

type unsupportedOSError struct{}

func (e *unsupportedOSError) Error() string { return "installer: unsupported OS: " + runtime.GOOS }
