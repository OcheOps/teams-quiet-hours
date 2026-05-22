package policy

const (
	PolicyName = "NotificationsBlockedForUrls"
	StateFile  = "policy-state.json"
)

var TeamsURLs = []string{
	"https://teams.microsoft.com/*",
	"https://*.teams.microsoft.com/*",
}

type Manager interface {
	Block(browser string, dryRun bool) error
	Allow(browser string, dryRun bool) error
	Status(browser string) ([]Status, error)
}

type Status struct {
	Browser string
	Target  string
	Blocked bool
	Detail  string
}

func ContainsAllTeamsURLs(values []string) bool {
	found := map[string]bool{}
	for _, value := range values {
		found[value] = true
	}
	for _, value := range TeamsURLs {
		if !found[value] {
			return false
		}
	}
	return true
}

func AddTeamsURLs(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values)+len(TeamsURLs))
	for _, value := range values {
		if !seen[value] {
			out = append(out, value)
			seen[value] = true
		}
	}
	for _, value := range TeamsURLs {
		if !seen[value] {
			out = append(out, value)
			seen[value] = true
		}
	}
	return out
}

func RemoveTeamsURLs(values []string) []string {
	remove := map[string]bool{}
	for _, value := range TeamsURLs {
		remove[value] = true
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if !remove[value] {
			out = append(out, value)
		}
	}
	return out
}
