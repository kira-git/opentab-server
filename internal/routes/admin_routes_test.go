package routes

import (
	"net/http"
	"testing"

	"opentab-server/internal/models"
)

func TestAdminTeamAndUserManagementEndpoints(t *testing.T) {
	router := setupTestRouter()

	createTeam := performRequest(router, http.MethodPost, "/admin/teams", "mock-admin-token", `{"teamName":"测试团队","description":"用于接口测试"}`)
	if createTeam.Code != http.StatusOK {
		t.Fatalf("create team expected status 200, got %d: %s", createTeam.Code, createTeam.Body.String())
	}
	team := decodeJSON[models.TeamAdminItem](t, createTeam.Body)
	if team.TeamID == "" {
		t.Fatalf("expected created team id")
	}

	updateTeam := performRequest(router, http.MethodPut, "/admin/teams/"+team.TeamID, "mock-admin-token", `{"teamName":"测试团队新版","description":"已更新"}`)
	if updateTeam.Code != http.StatusOK {
		t.Fatalf("update team expected status 200, got %d: %s", updateTeam.Code, updateTeam.Body.String())
	}

	addMember := performRequest(router, http.MethodPost, "/admin/teams/"+team.TeamID+"/members", "mock-admin-token", `{"userId":"user-product-employee","teamRole":"employee"}`)
	if addMember.Code != http.StatusOK {
		t.Fatalf("add member expected status 200, got %d: %s", addMember.Code, addMember.Body.String())
	}

	updateMember := performRequest(router, http.MethodPut, "/admin/teams/"+team.TeamID+"/members/user-product-employee", "mock-admin-token", `{"teamRole":"manager"}`)
	if updateMember.Code != http.StatusOK {
		t.Fatalf("update member expected status 200, got %d: %s", updateMember.Code, updateMember.Body.String())
	}

	deleteMember := performRequest(router, http.MethodDelete, "/admin/teams/"+team.TeamID+"/members/user-product-employee", "mock-admin-token", "")
	if deleteMember.Code != http.StatusOK {
		t.Fatalf("delete member expected status 200, got %d: %s", deleteMember.Code, deleteMember.Body.String())
	}

	setRole := performRequest(router, http.MethodPut, "/admin/users/user-product-employee/global-role", "mock-admin-token", `{"globalRole":"admin"}`)
	if setRole.Code != http.StatusOK {
		t.Fatalf("set global role expected status 200, got %d: %s", setRole.Code, setRole.Body.String())
	}
	user := decodeJSON[models.AdminUserItem](t, setRole.Body)
	if user.GlobalRole == nil || *user.GlobalRole != "admin" {
		t.Fatalf("expected user global role admin, got %+v", user.GlobalRole)
	}

	clearRole := performRequest(router, http.MethodPut, "/admin/users/user-product-employee/global-role", "mock-admin-token", `{"globalRole":null}`)
	if clearRole.Code != http.StatusOK {
		t.Fatalf("clear global role expected status 200, got %d: %s", clearRole.Code, clearRole.Body.String())
	}

	deleteTeam := performRequest(router, http.MethodDelete, "/admin/teams/"+team.TeamID, "mock-admin-token", "")
	if deleteTeam.Code != http.StatusOK {
		t.Fatalf("delete team expected status 200, got %d: %s", deleteTeam.Code, deleteTeam.Body.String())
	}
}
