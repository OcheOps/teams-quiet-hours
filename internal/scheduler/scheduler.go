package scheduler

import "github.com/OcheOps/teams-quiet-hours/internal/config"

type Manager interface {
	Install(cfg config.Config, executable string, dryRun bool) error
	Uninstall(dryRun bool) error
	Status() ([]Status, error)
}

type Status struct {
	Name      string
	Target    string
	Installed bool
	Detail    string
}
