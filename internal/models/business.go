package models

type ApprovalSummary struct {
	PendingCount  int            `json:"pendingCount"`
	ApprovedToday int            `json:"approvedToday"`
	Items         []ApprovalItem `json:"items"`
}

type ApprovalItem struct {
	ID          string         `json:"id"`
	TeamID      string         `json:"teamId,omitempty"`
	TeamName    string         `json:"teamName,omitempty"`
	Type        string         `json:"type,omitempty"`
	Title       string         `json:"title"`
	ApplicantID string         `json:"applicantId,omitempty"`
	Applicant   string         `json:"applicant"`
	ApproverID  string         `json:"approverId,omitempty"`
	Approver    string         `json:"approver,omitempty"`
	Amount      int            `json:"amount,omitempty"`
	Reason      string         `json:"reason,omitempty"`
	Summary     string         `json:"summary,omitempty"`
	Form        map[string]any `json:"form,omitempty"`
	Status      string         `json:"status"`
	CreatedAt   string         `json:"createdAt"`
	Comment     string         `json:"comment,omitempty"`
	UpdatedAt   string         `json:"updatedAt,omitempty"`
}

type CalendarSummary struct {
	TodayCount int             `json:"todayCount"`
	Events     []CalendarEvent `json:"events"`
}

type CalendarEvent struct {
	ID             string   `json:"id"`
	TeamID         string   `json:"teamId,omitempty"`
	TeamName       string   `json:"teamName,omitempty"`
	Visibility     string   `json:"visibility,omitempty"`
	CreatorID      string   `json:"creatorId,omitempty"`
	CreatorName    string   `json:"creatorName,omitempty"`
	Title          string   `json:"title"`
	Description    string   `json:"description,omitempty"`
	StartTime      string   `json:"startTime"`
	EndTime        string   `json:"endTime"`
	Location       string   `json:"location"`
	Participants   []string `json:"participants,omitempty"`
	ParticipantIDs []string `json:"participantIds,omitempty"`
	UpdatedAt      string   `json:"updatedAt,omitempty"`
}

type ApprovalActionRequest struct {
	Comment string `json:"comment"`
}

type ApprovalActionResponse struct {
	Success bool   `json:"success"`
	ItemID  string `json:"itemId"`
	Status  string `json:"status"`
}

type CreateApprovalItemRequest struct {
	TeamID string         `json:"teamId"`
	Type   string         `json:"type"`
	Title  string         `json:"title"`
	Reason string         `json:"reason"`
	Form   map[string]any `json:"form"`
}

type CreateCalendarEventRequest struct {
	TeamID         string   `json:"teamId"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	StartTime      string   `json:"startTime"`
	EndTime        string   `json:"endTime"`
	Location       string   `json:"location"`
	Visibility     string   `json:"visibility"`
	ParticipantIDs []string `json:"participantIds"`
}

type CreateCalendarEventResponse struct {
	Success bool   `json:"success"`
	EventID string `json:"eventId"`
}

type Announcement struct {
	ID            string `json:"id"`
	TeamID        string `json:"teamId,omitempty"`
	TeamName      string `json:"teamName,omitempty"`
	Scope         string `json:"scope"`
	Title         string `json:"title"`
	Content       string `json:"content"`
	PublisherID   string `json:"publisherId"`
	PublisherName string `json:"publisherName"`
	Pinned        bool   `json:"pinned"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

type AnnouncementRequest struct {
	TeamID  string `json:"teamId"`
	Scope   string `json:"scope"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Pinned  bool   `json:"pinned"`
}

type TeamAdminItem struct {
	TeamID       string `json:"teamId"`
	TeamName     string `json:"teamName"`
	Description  string `json:"description"`
	MemberCount  int    `json:"memberCount"`
	ManagerCount int    `json:"managerCount"`
	Enabled      bool   `json:"enabled"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type TeamRequest struct {
	TeamName    string `json:"teamName"`
	Description string `json:"description"`
}

type TeamMemberItem struct {
	UserID      string `json:"userId"`
	Account     string `json:"account"`
	DisplayName string `json:"displayName"`
	TeamID      string `json:"teamId"`
	TeamName    string `json:"teamName"`
	TeamRole    string `json:"teamRole"`
	JoinedAt    string `json:"joinedAt"`
	Enabled     bool   `json:"enabled"`
}

type AdminUserItem struct {
	UserID      string           `json:"userId"`
	Account     string           `json:"account"`
	DisplayName string           `json:"displayName"`
	GlobalRole  *string          `json:"globalRole"`
	Memberships []TeamMembership `json:"memberships"`
	Enabled     bool             `json:"enabled"`
}

type TeamMemberMutationRequest struct {
	UserID   string `json:"userId"`
	TeamRole string `json:"teamRole"`
}

type TeamMemberMutationResponse struct {
	Success  bool   `json:"success"`
	TeamID   string `json:"teamId"`
	UserID   string `json:"userId"`
	TeamRole string `json:"teamRole"`
}

type GlobalRoleRequest struct {
	GlobalRole *string `json:"globalRole"`
}
