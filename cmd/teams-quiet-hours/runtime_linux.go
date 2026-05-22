//go:build linux

package main

import (
	"github.com/OcheOps/teams-quiet-hours/internal/platform"
	"github.com/OcheOps/teams-quiet-hours/internal/platform/linux"
)

func currentRuntime() platform.Runtime {
	return linux.New()
}
