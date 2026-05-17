package openbrowser

import (
	"runtime"
	"testing"
)

func TestCmdConstructsExpectedCommandForCurrentOS(t *testing.T) {
	name, args := cmd("http://localhost:8080")
	switch runtime.GOOS {
	case "windows":
		if name != "rundll32" {
			t.Fatalf("windows: want rundll32, got %q", name)
		}
		if len(args) != 2 || args[0] != "url.dll,FileProtocolHandler" || args[1] != "http://localhost:8080" {
			t.Fatalf("windows: bad args: %v", args)
		}
	case "darwin":
		if name != "open" {
			t.Fatalf("darwin: want open, got %q", name)
		}
		if len(args) != 1 || args[0] != "http://localhost:8080" {
			t.Fatalf("darwin: bad args: %v", args)
		}
	case "linux":
		if name != "xdg-open" {
			t.Fatalf("linux: want xdg-open, got %q", name)
		}
		if len(args) != 1 || args[0] != "http://localhost:8080" {
			t.Fatalf("linux: bad args: %v", args)
		}
	default:
		t.Skip("unsupported OS for this test")
	}
}
