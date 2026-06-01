package models

type OnCallSession struct {
	SessionID    string `json:"sessionId"`
	Title        string `json:"title"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt,omitempty"`
	MessageCount int    `json:"messageCount,omitempty"`
}

type CreateOnCallSessionRequest struct {
	Title string `json:"title"`
}

type OnCallMessageRequest struct {
	Content     string `json:"content"`
	ContentType string `json:"contentType"`
}

type OnCallMessage struct {
	MessageID   string `json:"messageId"`
	SessionID   string `json:"sessionId"`
	Role        string `json:"role"`
	Content     string `json:"content"`
	ContentType string `json:"contentType"`
	CreatedAt   string `json:"createdAt"`
}

type DeleteOnCallSessionResponse struct {
	Success   bool   `json:"success"`
	SessionID string `json:"sessionId"`
}
