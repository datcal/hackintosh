package installer

import (
	"os"
	"strings"
	"testing"
)

func TestBuildDesktopEntryMatchesGolden(t *testing.T) {
	got := buildDesktopEntry("/home/u/.local/bin/hackintosh", "/home/u/.local/share/hackintosh/icon.png")
	want, err := os.ReadFile("testdata/linux.desktop.golden")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimRight(got, "\n") != strings.TrimRight(string(want), "\n") {
		t.Fatalf("desktop entry mismatch.\nGOT:\n%s\nWANT:\n%s", got, string(want))
	}
}
