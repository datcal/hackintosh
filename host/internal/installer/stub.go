//go:build !windows && !darwin && !linux

package installer

func doInstall() (Result, error)   { return Result{}, errUnsupportedOS }
func doUninstall() (Result, error) { return Result{}, errUnsupportedOS }
