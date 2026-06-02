package models

type LoginRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Account     string `json:"account"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

type LoginResponse struct {
	Token       string   `json:"token"`
	UserID      string   `json:"userId,omitempty"`
	DisplayName string   `json:"displayName"`
	Permissions []string `json:"permissions"`
}

type User struct {
	ID            string
	Account       string
	DisplayName   string
	Password      string
	Token         string
	GlobalRole    string
	CurrentTeamID string
	Memberships   []TeamMembership
	Permissions   []string
}

type Team struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type TeamMembership struct {
	TeamID   string `json:"teamId"`
	TeamName string `json:"teamName"`
	TeamRole string `json:"teamRole"`
}

type MeResponse struct {
	UserID        string           `json:"userId"`
	DisplayName   string           `json:"displayName"`
	GlobalRole    *string          `json:"globalRole"`
	CurrentTeamID *string          `json:"currentTeamId"`
	Memberships   []TeamMembership `json:"memberships"`
	Permissions   []string         `json:"permissions"`
	Team          *Team            `json:"team"`
}

type SuccessResponse struct {
	Success bool `json:"success"`
}
