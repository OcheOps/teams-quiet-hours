//go:build darwin

package main

import (
	"github.com/OcheOps/teams-quiet-hours/internal/platform"
	"github.com/OcheOps/teams-quiet-hours/internal/platform/darwin"
)

func currentRuntime() platform.Runtime {
	return darwin.New()
}
