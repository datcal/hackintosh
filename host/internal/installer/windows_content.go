package installer

import "fmt"

// buildShortcutScript returns a PowerShell one-liner that creates a Windows
// .lnk shortcut at lnkPath pointing at targetPath. Used by the Windows
// installer; piped to `powershell -NoProfile -Command -`.
func buildShortcutScript(lnkPath, targetPath string) string {
	return fmt.Sprintf(
		`$ws = New-Object -ComObject WScript.Shell; `+
			`$s = $ws.CreateShortcut("%s"); `+
			`$s.TargetPath = "%s"; `+
			`$s.Save()`,
		psEscape(lnkPath), psEscape(targetPath),
	)
}

// psEscape escapes a string for inclusion inside a PowerShell double-quoted
// literal. Only double quotes and backticks need escaping there.
func psEscape(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch r {
		case '"':
			out = append(out, '`', '"')
		case '`':
			out = append(out, '`', '`')
		default:
			out = append(out, r)
		}
	}
	return string(out)
}
