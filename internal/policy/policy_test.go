package policy

import "testing"

func TestAddAndRemoveTeamsURLs(t *testing.T) {
	existing := []string{"https://example.com/*", TeamsURLs[0]}
	withTeams := AddTeamsURLs(existing)
	if !ContainsAllTeamsURLs(withTeams) {
		t.Fatalf("expected all Teams URLs in %v", withTeams)
	}

	withoutTeams := RemoveTeamsURLs(withTeams)
	if ContainsAllTeamsURLs(withoutTeams) {
		t.Fatalf("expected Teams URLs removed from %v", withoutTeams)
	}
	if len(withoutTeams) != 1 || withoutTeams[0] != "https://example.com/*" {
		t.Fatalf("expected unrelated URL to be preserved, got %v", withoutTeams)
	}
}
