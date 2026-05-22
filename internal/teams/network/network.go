package network

import "github.com/OcheOps/teams-quiet-hours/internal/teams/status"

type Manager interface {
	Block(dryRun bool) error
	Allow(dryRun bool) error
	Status() ([]status.Item, error)
}
