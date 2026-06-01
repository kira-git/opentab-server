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
	ID          string
	Account     string
	DisplayName string
	Password    string
	Token       string
	Permissions []string
}

type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type MeResponse struct {
	UserID      string   `json:"userId"`
	DisplayName string   `json:"displayName"`
	Permissions []string `json:"permissions"`
	Team        Team     `json:"team"`
}

type SuccessResponse struct {
	Success bool `json:"success"`
}
