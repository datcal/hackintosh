package installer

import (
	"os"
	"strings"
	"testing"
)

func TestBuildShortcutScriptMatchesGolden(t *testing.T) {
	got := buildShortcutScript(
		`C:\Users\u\AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Hackintosh.lnk`,
		`C:\Users\u\AppData\Local\Hackintosh\hackintosh.exe`,
	)
	want, err := os.ReadFile("testdata/windows.shortcut.ps1.golden")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimRight(got, "\r\n") != strings.TrimRight(string(want), "\r\n") {
		t.Fatalf("shortcut script mismatch.\nGOT:\n%s\nWANT:\n%s", got, string(want))
	}
}
