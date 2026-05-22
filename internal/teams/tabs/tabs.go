package tabs

type Manager interface {
	CloseTeamsTabs(dryRun bool) error
}
