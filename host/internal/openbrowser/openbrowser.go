// Package openbrowser opens a URL in the user's default browser.
// Used by the tray menu's "Open Simulator" item.
package openbrowser

import "os/exec"

// Open launches the OS-default browser pointed at url.
// It does not wait for the browser to exit.
func Open(url string) error {
	name, args := cmd(url)
	return exec.Command(name, args...).Start()
}
