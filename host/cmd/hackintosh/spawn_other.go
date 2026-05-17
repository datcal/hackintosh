//go:build !windows

package main

import "os/exec"

func spawnDetached(bin string) error {
	return exec.Command(bin).Start()
}
