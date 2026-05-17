package installer

import "fmt"

const darwinBundleID = "com.datcal.hackintosh"

// buildInfoPlist returns the contents of Hackintosh.app/Contents/Info.plist.
// LSUIElement=1 keeps the app out of the Dock -- important for a tray-only
// utility.
func buildInfoPlist() string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleIdentifier</key>
    <string>%s</string>
    <key>CFBundleName</key>
    <string>Hackintosh</string>
    <key>CFBundleDisplayName</key>
    <string>Hackintosh</string>
    <key>CFBundleExecutable</key>
    <string>Hackintosh</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0</string>
    <key>LSUIElement</key>
    <true/>
    <key>LSMinimumSystemVersion</key>
    <string>10.13</string>
</dict>
</plist>
`, darwinBundleID)
}

// buildLaunchAgentPlist returns the contents of
// ~/Library/LaunchAgents/com.datcal.hackintosh.plist. The plist runs the
// inner app binary at user login.
func buildLaunchAgentPlist(execPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
    <key>ProcessType</key>
    <string>Interactive</string>
</dict>
</plist>
`, darwinBundleID, execPath)
}
