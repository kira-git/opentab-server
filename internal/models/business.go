package models

type ApprovalSummary struct {
	PendingCount  int            `json:"pendingCount"`
	ApprovedToday int            `json:"approvedToday"`
	Items         []ApprovalItem `json:"items"`
}

type ApprovalItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Applicant string `json:"applicant"`
	Amount    int    `json:"amount,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	Comment   string `json:"comment,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type CalendarSummary struct {
	TodayCount int             `json:"todayCount"`
	Events     []CalendarEvent `json:"events"`
}

type CalendarEvent struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description,omitempty"`
	StartTime    string   `json:"startTime"`
	EndTime      string   `json:"endTime"`
	Location     string   `json:"location"`
	Participants []string `json:"participants,omitempty"`
}

type ApprovalActionRequest struct {
	Comment string `json:"comment"`
}

type ApprovalActionResponse struct {
	Success bool   `json:"success"`
	ItemID  string `json:"itemId"`
	Status  string `json:"status"`
}

type CreateCalendarEventRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	StartTime   string `json:"startTime"`
	EndTime     string `json:"endTime"`
	Location    string `json:"location"`
}

type CreateCalendarEventResponse struct {
	Success bool   `json:"success"`
	EventID string `json:"eventId"`
}
