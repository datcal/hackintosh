package installer

import "github.com/datcal/hackintosh/host/internal/tray"

// trayIconBytes returns the PNG bytes of the application tray icon.
// The bytes are embedded once (in the tray package) and shared here so
// the installer can write them to disk without duplicating the asset.
func trayIconBytes() []byte { return tray.IconBytes() }
