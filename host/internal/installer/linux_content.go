package installer

import "fmt"

// buildDesktopEntry returns the contents of a freedesktop .desktop file
// pointing at the given binary. The same content is used for both the
// autostart entry (~/.config/autostart/) and the application menu entry
// (~/.local/share/applications/).
func buildDesktopEntry(execPath, iconPath string) string {
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Hackintosh
Comment=Companion daemon for the OLED Mac-style display
Exec=%s
Icon=%s
StartupNotify=false
X-GNOME-Autostart-enabled=true
Categories=Utility;
`, execPath, iconPath)
}
