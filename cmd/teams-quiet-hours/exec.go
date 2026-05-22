package main

import "os/exec"

func nilIfExecUnavailable(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}
