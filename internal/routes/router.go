package routes

import (
	"net/http"
	"time"

	"opentab-server/internal/middleware"
	"opentab-server/internal/repositories"
	"opentab-server/internal/services"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	auth     *services.AuthService
	tabs     *services.TabService
	business *services.BusinessService
	oncall   *services.OnCallService
	debug    *services.DebugService
	audit    repositories.AuditRepository
	status   RuntimeStatus
	sseDelay time.Duration
}

type RuntimeStatus struct {
	AppMode          string
	DatabaseEnabled  bool
	DatabaseType     string
	AIServiceBaseURL string
}

type HandlerOptions struct {
	OnCall services.OnCallOptions
}

func NewHandler() *Handler {
	return NewHandlerWithRepositories(repositories.NewMemoryRepositorySet())
}

func NewHandlerWithRepositories(repos repositories.RepositorySet) *Handler {
	return NewHandlerWithStatus(repos, RuntimeStatus{
		AppMode:         "mock",
		DatabaseEnabled: false,
		DatabaseType:    "memory",
	})
}

func NewHandlerWithStatus(repos repositories.RepositorySet, status RuntimeStatus) *Handler {
	return NewHandlerWithStatusAndOptions(repos, status, HandlerOptions{})
}

func NewHandlerWithStatusAndOptions(repos repositories.RepositorySet, status RuntimeStatus, opts HandlerOptions) *Handler {
	return &Handler{
		auth:     services.NewAuthService(repos.Users),
		tabs:     services.NewTabService(repos.Tabs),
		business: services.NewBusinessService(repos.Business),
		oncall:   services.NewOnCallServiceWithOptions(repos.OnCall, status.AIServiceBaseURL, opts.OnCall),
		debug:    services.NewDebugService(repos.Debug),
		audit:    repos.Audit,
		status:   status,
		sseDelay: 300 * time.Millisecond,
	}
}

func Register(router *gin.Engine) {
	handler := NewHandler()
	registerWithHandler(router, handler)
}

func RegisterWithRepositories(router *gin.Engine, repos repositories.RepositorySet) {
	handler := NewHandlerWithRepositories(repos)
	registerWithHandler(router, handler)
}

func RegisterWithStatus(router *gin.Engine, repos repositories.RepositorySet, status RuntimeStatus) {
	handler := NewHandlerWithStatus(repos, status)
	registerWithHandler(router, handler)
}

func RegisterWithStatusAndOptions(router *gin.Engine, repos repositories.RepositorySet, status RuntimeStatus, opts HandlerOptions) {
	handler := NewHandlerWithStatusAndOptions(repos, status, opts)
	registerWithHandler(router, handler)
}

func registerWithHandler(router *gin.Engine, handler *Handler) {
	router.Use(middleware.RequestID())
	router.Use(middleware.Audit(handler.audit))

	router.GET("/health", handler.health)
	if handler.status.AppMode != "postgres" {
		router.POST("/api/chat/stream", handler.streamAIChat)
	}

	auth := router.Group("/auth")
	{
		auth.POST("/login", handler.login)
		auth.POST("/register", handler.register)
		auth.POST("/logout", middleware.Auth(handler.auth), handler.logout)
	}

	authorized := router.Group("/")
	authorized.Use(middleware.Auth(handler.auth))

	authorized.GET("/me", handler.me)
	if handler.status.AppMode == "postgres" {
		authorized.POST("/api/chat/stream", handler.streamAIChat)
	}

	authorized.GET("/tabs", handler.listTabs)
	authorized.POST("/tabs", handler.createCustomTab)
	authorized.GET("/tabs/catalog", handler.listTabCatalog)
	authorized.POST("/tabs/validate", handler.validateTab)
	authorized.GET("/tabs/:tabId", handler.getTab)
	authorized.PUT("/tabs/:tabId", handler.updateCustomTab)
	authorized.DELETE("/tabs/:tabId", handler.deleteCustomTab)
	authorized.POST("/tabs/:tabId/actions/:actionId", handler.reportTabAction)

	currentUser := authorized.Group("/me")
	{
		currentUser.POST("/tabs", handler.enableMyTab)
		currentUser.DELETE("/tabs/:tabId", handler.disableMyTab)
		currentUser.PUT("/tabs/order", handler.reorderMyTabs)
	}

	business := authorized.Group("/business")
	{
		business.GET("/approval/summary", handler.approvalSummary)
		business.GET("/approval/items", handler.listApprovalItems)
		business.POST("/approval/items", handler.createApprovalItem)
		business.GET("/approval/items/:itemId", handler.getApprovalItem)
		business.POST("/approval/items/:itemId/approve", handler.approveItem)
		business.POST("/approval/items/:itemId/reject", handler.rejectItem)
		business.POST("/approval/items/:itemId/cancel", handler.cancelApprovalItem)
		business.GET("/calendar/summary", handler.calendarSummary)
		business.GET("/calendar/events", handler.listCalendarEvents)
		business.GET("/calendar/events/:eventId", handler.getCalendarEvent)
		business.POST("/calendar/events", handler.createCalendarEvent)
		business.PUT("/calendar/events/:eventId", handler.updateCalendarEvent)
		business.DELETE("/calendar/events/:eventId", handler.deleteCalendarEvent)
		business.GET("/announcements", handler.listAnnouncements)
		business.POST("/announcements", handler.createAnnouncement)
		business.GET("/announcements/:announcementId", handler.getAnnouncement)
		business.PUT("/announcements/:announcementId", handler.updateAnnouncement)
		business.DELETE("/announcements/:announcementId", handler.deleteAnnouncement)
	}

	admin := authorized.Group("/admin")
	{
		admin.GET("/teams", handler.listAdminTeams)
		admin.POST("/teams", handler.createAdminTeam)
		admin.PUT("/teams/:teamId", handler.updateAdminTeam)
		admin.DELETE("/teams/:teamId", handler.deleteAdminTeam)
		admin.GET("/teams/:teamId/members", handler.listAdminTeamMembers)
		admin.POST("/teams/:teamId/members", handler.addAdminTeamMember)
		admin.PUT("/teams/:teamId/members/:userId", handler.updateAdminTeamMember)
		admin.DELETE("/teams/:teamId/members/:userId", handler.deleteAdminTeamMember)
		admin.GET("/users", handler.listAdminUsers)
		admin.GET("/users/:userId", handler.getAdminUser)
		admin.PUT("/users/:userId/global-role", handler.updateAdminUserGlobalRole)
	}

	debug := authorized.Group("/debug")
	{
		debug.GET("/status", handler.debugStatus)
		debug.GET("/permissions", handler.debugPermissions)
		debug.GET("/sample-tabs", handler.debugSampleTabs)
	}

	authorized.GET("/oncall/stream", handler.streamOnCall)
	authorized.POST("/oncall/sessions", handler.createOnCallSession)
	authorized.GET("/oncall/sessions", handler.listOnCallSessions)
	authorized.POST("/oncall/sessions/:sessionId/messages", handler.addOnCallMessage)
	authorized.GET("/oncall/sessions/:sessionId/messages", handler.listOnCallMessages)
	authorized.GET("/oncall/sessions/:sessionId/stream", handler.streamOnCallSession)
	authorized.POST("/oncall/sessions/:sessionId/cancel", handler.cancelOnCallGeneration)
	authorized.DELETE("/oncall/sessions/:sessionId", handler.deleteOnCallSession)
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"service":    "tab-container-server",
		"mode":       h.status.AppMode,
		"serverTime": time.Now().Format(time.RFC3339),
	})
}
