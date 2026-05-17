package installer

import (
	"os"
	"strings"
	"testing"
)

func TestBuildInfoPlistMatchesGolden(t *testing.T) {
	got := buildInfoPlist()
	want, err := os.ReadFile("testdata/darwin.info.plist.golden")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimRight(got, "\n") != strings.TrimRight(string(want), "\n") {
		t.Fatalf("Info.plist mismatch.\nGOT:\n%s\nWANT:\n%s", got, string(want))
	}
}

func TestBuildLaunchAgentPlistMatchesGolden(t *testing.T) {
	got := buildLaunchAgentPlist("/Users/u/Applications/Hackintosh.app/Contents/MacOS/Hackintosh")
	want, err := os.ReadFile("testdata/darwin.launchagent.plist.golden")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimRight(got, "\n") != strings.TrimRight(string(want), "\n") {
		t.Fatalf("LaunchAgent plist mismatch.\nGOT:\n%s\nWANT:\n%s", got, string(want))
	}
}
