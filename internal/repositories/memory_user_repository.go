package repositories

import (
	"time"

	"opentab-server/internal/mockdata"
	"opentab-server/internal/models"
)

type MemoryUserRepository struct{}

type memoryAuthSession struct {
	UserID    string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

var memoryAuthSessions = map[string]memoryAuthSession{}
var memoryAuthSessionsSeeded bool

func NewMemoryUserRepository() *MemoryUserRepository {
	return &MemoryUserRepository{}
}

func (r *MemoryUserRepository) FindByAccount(account string) (*models.User, error) {
	user := mockdata.FindUser(account)
	if user == nil {
		return nil, ErrNotFound
	}
	applyMemoryMemberships(user)
	return user, nil
}

func (r *MemoryUserRepository) FindByToken(token string) (*models.User, error) {
	seedMemoryAuthSessions()
	session, ok := memoryAuthSessions[token]
	if !ok || session.RevokedAt != nil || time.Now().After(session.ExpiresAt) {
		return nil, ErrNotFound
	}
	user := findMemoryUserByID(session.UserID)
	if user == nil {
		return nil, ErrNotFound
	}
	user.Token = token
	applyMemoryMemberships(user)
	return user, nil
}

func (r *MemoryUserRepository) Create(user models.User, enabledTabIDs []string) error {
	if mockdata.FindUser(user.Account) != nil {
		return ErrConflict
	}

	mockdata.Users = append(mockdata.Users, user)
	mockdata.Users[len(mockdata.Users)-1].CurrentTeamID = "team-product"
	if mockdata.UserTabs == nil {
		mockdata.UserTabs = map[string]map[string]bool{}
	}
	mockdata.UserTabs[user.ID] = map[string]bool{}
	for _, tabID := range enabledTabIDs {
		mockdata.UserTabs[user.ID][tabID] = true
	}
	return nil
}

func (r *MemoryUserRepository) CreateSession(userID string, token string, expiresAt time.Time) error {
	seedMemoryAuthSessions()
	if _, exists := memoryAuthSessions[token]; exists {
		return ErrConflict
	}
	memoryAuthSessions[token] = memoryAuthSession{
		UserID:    userID,
		ExpiresAt: expiresAt,
	}
	return nil
}

func (r *MemoryUserRepository) RevokeToken(token string) error {
	seedMemoryAuthSessions()
	session, exists := memoryAuthSessions[token]
	if !exists {
		return ErrNotFound
	}
	now := time.Now()
	session.RevokedAt = &now
	memoryAuthSessions[token] = session
	return nil
}

func seedMemoryAuthSessions() {
	if memoryAuthSessionsSeeded {
		return
	}
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	for _, user := range mockdata.Users {
		if user.Token != "" {
			memoryAuthSessions[user.Token] = memoryAuthSession{
				UserID:    user.ID,
				ExpiresAt: expiresAt,
			}
		}
	}
	memoryAuthSessionsSeeded = true
}

func findMemoryUserByID(userID string) *models.User {
	for i := range mockdata.Users {
		if mockdata.Users[i].ID == userID {
			return &mockdata.Users[i]
		}
	}
	return nil
}

func applyMemoryMemberships(user *models.User) {
	switch user.ID {
	case "user-product-manager":
		user.CurrentTeamID = "team-product"
		user.Memberships = []models.TeamMembership{{TeamID: "team-product", TeamName: "产品研发部", TeamRole: "manager"}}
	case "user-product-employee", "user-demo":
		user.CurrentTeamID = "team-product"
		user.Memberships = []models.TeamMembership{{TeamID: "team-product", TeamName: "产品研发部", TeamRole: "employee"}}
	case "user-operation-manager":
		user.CurrentTeamID = "team-operation"
		user.Memberships = []models.TeamMembership{{TeamID: "team-operation", TeamName: "运营支持部", TeamRole: "manager"}}
	case "user-operation-employee":
		user.CurrentTeamID = "team-operation"
		user.Memberships = []models.TeamMembership{{TeamID: "team-operation", TeamName: "运营支持部", TeamRole: "employee"}}
	default:
		user.Memberships = nil
		user.CurrentTeamID = ""
	}
}
