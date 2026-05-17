// Package installer adds and removes per-OS autostart entries and launcher
// shortcuts for the Hackintosh host binary. Per-user only -- never requires
// admin/root.
package installer

import "fmt"

// Result describes what an Install or Uninstall call did. Used for logging
// back to the user via the install/uninstall subcommands.
type Result struct {
	// Files lists absolute paths created (Install) or removed (Uninstall).
	Files []string
	// Notes are short human-readable lines describing each step.
	Notes []string
}

func (r *Result) record(path string, note string) {
	r.Files = append(r.Files, path)
	r.Notes = append(r.Notes, note)
}

func (r *Result) noteOnly(note string) {
	r.Notes = append(r.Notes, note)
}

// String formats the Result for stdout display.
func (r Result) String() string {
	if len(r.Notes) == 0 {
		return "(no changes)"
	}
	out := ""
	for _, n := range r.Notes {
		out += "  " + n + "\n"
	}
	return out
}

// Install lays down autostart + launcher shortcut for the current OS.
// The caller is responsible for ensuring the running binary is the one we
// want to install (typically: the build output the user just produced).
func Install() (Result, error) {
	return doInstall()
}

// Uninstall reverses Install. Idempotent -- missing files are not errors.
func Uninstall() (Result, error) {
	return doUninstall()
}

// formatPath is a tiny helper for consistent error messages.
func formatPath(path string, err error) error {
	return fmt.Errorf("%s: %w", path, err)
}
