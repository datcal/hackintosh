package tray

import (
	"os"
	"testing"
)

// TestMain sets up package-level state before running tests.
// We replace systrayQuit with a no-op so tests don't panic when the systray
// subsystem is not initialised (which it never is in unit tests).
func TestMain(m *testing.M) {
	systrayQuit = func() {}
	os.Exit(m.Run())
}
