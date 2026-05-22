package process

import "github.com/OcheOps/teams-quiet-hours/internal/teams/status"

type Manager interface {
	Stop(dryRun bool) error
	Start(dryRun bool) error
	Status() ([]status.Item, error)
}
