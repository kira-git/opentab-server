package repositories

import (
	"testing"

	"opentab-server/internal/models"
)

func TestMemoryAddTeamMemberMovesUserToSingleActiveTeam(t *testing.T) {
	repo := NewMemoryBusinessRepository()

	if _, err := repo.AddTeamMember("team-operation", models.TeamMemberMutationRequest{
		UserID:   "user-product-employee",
		TeamRole: "employee",
	}); err != nil {
		t.Fatalf("add team member failed: %v", err)
	}

	productMembers, err := repo.ListTeamMembers("team-product")
	if err != nil {
		t.Fatalf("list product members failed: %v", err)
	}
	for _, member := range productMembers {
		if member.UserID == "user-product-employee" {
			t.Fatalf("expected moved user to leave old team")
		}
	}

	operationMembers, err := repo.ListTeamMembers("team-operation")
	if err != nil {
		t.Fatalf("list operation members failed: %v", err)
	}
	found := false
	for _, member := range operationMembers {
		if member.UserID == "user-product-employee" && member.TeamRole == "employee" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected moved user to appear in new team")
	}
}
