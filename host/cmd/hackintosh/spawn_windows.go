//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

func spawnDetached(bin string) error {
	cmd := exec.Command(bin)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}
