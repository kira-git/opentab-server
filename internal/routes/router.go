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
	status   RuntimeStatus
	sseDelay time.Duration
}

type RuntimeStatus struct {
	AppMode          string
	DatabaseEnabled  bool
	DatabaseType     string
	AIServiceBaseURL string
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
	return &Handler{
		auth:     services.NewAuthService(repos.Users),
		tabs:     services.NewTabService(repos.Tabs),
		business: services.NewBusinessService(repos.Business),
		oncall:   services.NewOnCallService(repos.OnCall, status.AIServiceBaseURL),
		debug:    services.NewDebugService(repos.Debug),
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

func registerWithHandler(router *gin.Engine, handler *Handler) {
	router.GET("/health", handler.health)
	router.POST("/api/chat/stream", handler.streamAIChat)

	auth := router.Group("/auth")
	{
		auth.POST("/login", handler.login)
		auth.POST("/register", handler.register)
		auth.POST("/logout", middleware.Auth(handler.auth), handler.logout)
	}

	authorized := router.Group("/")
	authorized.Use(middleware.Auth(handler.auth))

	authorized.GET("/me", handler.me)

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
		business.GET("/approval/items/:itemId", handler.getApprovalItem)
		business.POST("/approval/items/:itemId/approve", handler.approveItem)
		business.POST("/approval/items/:itemId/reject", handler.rejectItem)
		business.GET("/calendar/summary", handler.calendarSummary)
		business.GET("/calendar/events", handler.listCalendarEvents)
		business.GET("/calendar/events/:eventId", handler.getCalendarEvent)
		business.POST("/calendar/events", handler.createCalendarEvent)
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
