//go:build windows

package main

import (
	"github.com/OcheOps/teams-quiet-hours/internal/platform"
	"github.com/OcheOps/teams-quiet-hours/internal/platform/windows"
)

func currentRuntime() platform.Runtime {
	return windows.New()
}
