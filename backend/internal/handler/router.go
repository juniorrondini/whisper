package handler

import (
	"net/http"
	"strings"

	"whisper/backend/internal/config"
	"whisper/backend/internal/middleware"
	"whisper/backend/internal/platform/httputil"
	"whisper/backend/internal/repository"
	"whisper/backend/internal/service"
	realtime "whisper/backend/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type API struct {
	cfg config.Config
	app *service.App
	hub *realtime.Hub
}

func NewRouter(cfg config.Config, app *service.App, hub *realtime.Hub) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	api := &API{cfg: cfg, app: app, hub: hub}
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), middleware.CORS(cfg.CORSOrigins))

	router.GET("/health", api.health)
	router.GET("/ws", realtime.Serve(app, hub, cfg.JWTSecret))

	auth := router.Group("/api/auth")
	auth.POST("/register-company", api.registerCompany)
	auth.POST("/login", middleware.LoginRateLimit(app.Redis()), api.login)
	auth.POST("/refresh", api.refresh)
	auth.POST("/logout", api.logout)
	auth.POST("/forgot-password", api.forgotPassword)

	protected := router.Group("/api", middleware.Auth(cfg.JWTSecret))
	protected.GET("/me", api.me)
	protected.GET("/dashboard", api.dashboard)
	protected.GET("/users", middleware.RequireRoles("ADMIN", "SUPERVISOR"), api.listUsers)
	protected.POST("/users", middleware.RequireRoles("ADMIN"), api.createUser)
	protected.GET("/customers", api.listCustomers)
	protected.POST("/customers", api.createCustomer)
	protected.GET("/customers/:id", api.getCustomer)
	protected.GET("/departments", api.listDepartments)
	protected.POST("/departments", middleware.RequireRoles("ADMIN"), api.createDepartment)
	protected.GET("/conversations", api.listConversations)
	protected.POST("/conversations", api.createConversation)
	protected.GET("/conversations/:id", api.getConversation)
	protected.GET("/conversations/:id/messages", api.listMessages)
	protected.POST("/conversations/:id/messages", api.createMessage)
	protected.POST("/conversations/:id/assign", api.assignConversation)
	protected.POST("/conversations/:id/status", api.updateConversationStatus)
	protected.GET("/quick-replies", api.listQuickReplies)
	protected.POST("/quick-replies", middleware.RequireRoles("ADMIN", "SUPERVISOR"), api.createQuickReply)

	return router
}

func (a *API) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "app": a.cfg.AppName})
}

type registerRequest struct {
	CompanyName string `json:"company_name"`
	CNPJ        string `json:"cnpj"`
	AdminName   string `json:"admin_name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

func (a *API) registerCompany(c *gin.Context) {
	var req registerRequest
	if !bind(c, &req) {
		return
	}
	company, tokens, err := a.app.RegisterCompany(c.Request.Context(), req.CompanyName, req.CNPJ, req.AdminName, req.Email, req.Password)
	if err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"company": company, "tokens": tokens})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (a *API) login(c *gin.Context) {
	var req loginRequest
	if !bind(c, &req) {
		return
	}
	tokens, err := a.app.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		httputil.Error(c, http.StatusUnauthorized, "invalid credentials")
		return
	}
	c.JSON(http.StatusOK, tokens)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (a *API) refresh(c *gin.Context) {
	var req refreshRequest
	if !bind(c, &req) {
		return
	}
	tokens, err := a.app.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		httputil.Unauthorized(c)
		return
	}
	c.JSON(http.StatusOK, tokens)
}

func (a *API) logout(c *gin.Context) {
	var req refreshRequest
	_ = c.ShouldBindJSON(&req)
	_ = a.app.Logout(c.Request.Context(), req.RefreshToken)
	c.Status(http.StatusNoContent)
}

func (a *API) forgotPassword(c *gin.Context) {
	c.JSON(http.StatusAccepted, gin.H{"message": "password recovery flow is prepared for future email provider integration"})
}

func (a *API) me(c *gin.Context) {
	user, err := a.app.Store().GetUser(c.Request.Context(), middleware.CompanyID(c), middleware.UserID(c))
	if err != nil {
		httputil.Unauthorized(c)
		return
	}
	c.JSON(http.StatusOK, user)
}

func (a *API) dashboard(c *gin.Context) {
	out, err := a.app.Store().Dashboard(c.Request.Context(), middleware.CompanyID(c))
	if err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, out)
}

func (a *API) listUsers(c *gin.Context) {
	items, err := a.app.Store().ListUsers(c.Request.Context(), middleware.CompanyID(c))
	respondList(c, items, err)
}

type createUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Password string `json:"password"`
}

func (a *API) createUser(c *gin.Context) {
	var req createUserRequest
	if !bind(c, &req) {
		return
	}
	user, err := a.app.CreateUser(c.Request.Context(), middleware.CompanyID(c), req.Name, req.Email, strings.ToUpper(req.Role), req.Password)
	respondCreated(c, user, err)
}

type createCustomerRequest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email"`
	Notes string `json:"notes"`
}

func (a *API) listCustomers(c *gin.Context) {
	items, err := a.app.Store().ListCustomers(c.Request.Context(), middleware.CompanyID(c), c.Query("q"))
	respondList(c, items, err)
}

func (a *API) createCustomer(c *gin.Context) {
	var req createCustomerRequest
	if !bind(c, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Phone) == "" {
		httputil.BadRequest(c, "name and phone are required")
		return
	}
	item, err := a.app.Store().CreateCustomer(c.Request.Context(), middleware.CompanyID(c), req.Name, req.Phone, req.Email, req.Notes)
	respondCreated(c, item, err)
}

func (a *API) getCustomer(c *gin.Context) {
	item, err := a.app.Store().GetCustomer(c.Request.Context(), middleware.CompanyID(c), c.Param("id"))
	respondOne(c, item, err)
}

func (a *API) listDepartments(c *gin.Context) {
	items, err := a.app.Store().ListDepartments(c.Request.Context(), middleware.CompanyID(c))
	respondList(c, items, err)
}

type createDepartmentRequest struct {
	Name string `json:"name"`
}

func (a *API) createDepartment(c *gin.Context) {
	var req createDepartmentRequest
	if !bind(c, &req) {
		return
	}
	item, err := a.app.Store().CreateDepartment(c.Request.Context(), middleware.CompanyID(c), req.Name)
	respondCreated(c, item, err)
}

type createConversationRequest struct {
	CustomerID   string `json:"customer_id"`
	DepartmentID string `json:"department_id"`
	Subject      string `json:"subject"`
	Priority     string `json:"priority"`
	Origin       string `json:"origin"`
	InitialText  string `json:"initial_text"`
}

func (a *API) listConversations(c *gin.Context) {
	items, err := a.app.Store().ListConversations(c.Request.Context(), middleware.CompanyID(c), c.Query("status"))
	respondList(c, items, err)
}

func (a *API) createConversation(c *gin.Context) {
	var req createConversationRequest
	if !bind(c, &req) {
		return
	}
	priority := defaultString(req.Priority, "normal")
	origin := defaultString(req.Origin, "panel")
	item, err := a.app.Store().CreateConversation(c.Request.Context(), middleware.CompanyID(c), req.CustomerID, req.DepartmentID, req.Subject, priority, origin)
	if err != nil {
		respondCreated(c, item, err)
		return
	}
	if strings.TrimSpace(req.InitialText) != "" {
		message, err := a.app.CreateMessage(c.Request.Context(), middleware.CompanyID(c), item.ID, "customer", nil, req.InitialText)
		if err == nil {
			a.hub.Broadcast(realtime.Event{CompanyID: middleware.CompanyID(c).String(), ConversationID: item.ID, Type: "message.created", Payload: message})
		}
	}
	c.JSON(http.StatusCreated, item)
}

func (a *API) getConversation(c *gin.Context) {
	item, err := a.app.Store().GetConversation(c.Request.Context(), middleware.CompanyID(c), c.Param("id"))
	respondOne(c, item, err)
}

func (a *API) listMessages(c *gin.Context) {
	items, err := a.app.Store().ListMessages(c.Request.Context(), middleware.CompanyID(c), c.Param("id"))
	respondList(c, items, err)
}

type createMessageRequest struct {
	Content string `json:"content"`
}

func (a *API) createMessage(c *gin.Context) {
	var req createMessageRequest
	if !bind(c, &req) {
		return
	}
	userID := middleware.UserID(c)
	message, err := a.app.CreateMessage(c.Request.Context(), middleware.CompanyID(c), c.Param("id"), "agent", &userID, req.Content)
	if err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}
	a.hub.Broadcast(realtime.Event{CompanyID: middleware.CompanyID(c).String(), ConversationID: c.Param("id"), Type: "message.created", Payload: message})
	c.JSON(http.StatusCreated, message)
}

type assignRequest struct {
	UserID string `json:"user_id"`
}

func (a *API) assignConversation(c *gin.Context) {
	var req assignRequest
	if !bind(c, &req) {
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		httputil.BadRequest(c, "invalid user_id")
		return
	}
	err = a.app.Store().AssignConversation(c.Request.Context(), middleware.CompanyID(c), c.Param("id"), userID)
	if err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

type statusRequest struct {
	Status string `json:"status"`
}

func (a *API) updateConversationStatus(c *gin.Context) {
	var req statusRequest
	if !bind(c, &req) {
		return
	}
	if req.Status != "open" && req.Status != "pending" && req.Status != "resolved" && req.Status != "closed" {
		httputil.BadRequest(c, "invalid status")
		return
	}
	err := a.app.Store().UpdateConversationStatus(c.Request.Context(), middleware.CompanyID(c), c.Param("id"), req.Status)
	if err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *API) listQuickReplies(c *gin.Context) {
	items, err := a.app.Store().ListQuickReplies(c.Request.Context(), middleware.CompanyID(c))
	respondList(c, items, err)
}

type quickReplyRequest struct {
	DepartmentID string `json:"department_id"`
	Title        string `json:"title"`
	Shortcut     string `json:"shortcut"`
	Content      string `json:"content"`
}

func (a *API) createQuickReply(c *gin.Context) {
	var req quickReplyRequest
	if !bind(c, &req) {
		return
	}
	item, err := a.app.Store().CreateQuickReply(c.Request.Context(), middleware.CompanyID(c), req.DepartmentID, req.Title, req.Shortcut, req.Content)
	respondCreated(c, item, err)
}

func bind(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		httputil.BadRequest(c, "invalid json body")
		return false
	}
	return true
}

func respondList[T any](c *gin.Context, items []T, err error) {
	if err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}
	if items == nil {
		items = []T{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func respondCreated(c *gin.Context, item any, err error) {
	if err != nil {
		httputil.BadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusCreated, item)
}

func respondOne(c *gin.Context, item any, err error) {
	if err != nil {
		if repository.IsNotFound(err) {
			httputil.Error(c, http.StatusNotFound, "not found")
			return
		}
		httputil.BadRequest(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, item)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
