//go:build integration

package identity

import (
	"errors"
	"github.com/google/uuid"
	"testing"
)

func TestMixedScopesAndTeamVisibility(t *testing.T) {
	h := newHarness(t)
	member, _ := h.employee(t, "scoped@example.invalid", h.agentRole)
	var teams []uuid.UUID
	for _, name := range []string{"Scoped", "Hidden"} {
		r, e := h.call("org.team.create", h.owner, Input{Name: name})
		if e != nil {
			t.Fatal(e)
		}
		teams = append(teams, r.Data.(map[string]any)["id"].(uuid.UUID))
	}
	r, e := h.call("org.role.create", h.owner, Input{Name: "Coordinator", Permissions: []string{"teams.view", "teams.manage"}})
	if e != nil {
		t.Fatal(e)
	}
	role := r.Data.(map[string]any)["id"].(uuid.UUID)
	if _, e = h.call("org.member.roles", h.owner, Input{ID: member, Revision: 1, Grants: []Grant{{h.agentRole, "ORG", nil}, {role, "TEAM", &teams[0]}}}); e != nil {
		t.Fatal(e)
	}
	session := h.login(t, "scoped@example.invalid")
	if _, e = h.call("org.get", session, Input{}); e != nil {
		t.Fatal("mixed scope lost organization view", e)
	}
	r, e = h.call("org.teams.list", session, Input{})
	if e != nil {
		t.Fatal(e)
	}
	items := r.Data.(map[string]any)["items"].([]map[string]any)
	if len(items) != 1 || items[0]["id"] != teams[0].String() {
		t.Fatal("team list leaked scope")
	}
	if _, e = h.call("org.team.update", session, Input{ID: teams[1], Revision: 1, Name: "Forbidden"}); !errors.Is(e, ErrPermission) {
		t.Fatal("team write escaped", e)
	}
	if _, e = h.call("org.team.update", session, Input{ID: teams[0], Revision: 1, Name: "Updated"}); e != nil {
		t.Fatal("scoped update denied", e)
	}
	if _, e = h.call("org.team.archive", h.owner, Input{ID: teams[0], Revision: 2}); e != nil {
		t.Fatal(e)
	}
	session = h.login(t, "scoped@example.invalid")
	if _, e = h.call("org.team.members.list", session, Input{ID: teams[0]}); !errors.Is(e, ErrPermission) {
		t.Fatal("archived direct team grant survived", e)
	}
}
