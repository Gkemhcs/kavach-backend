package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AuthHandler handles HTTP requests related to authentication.
type AuthHandler struct {
	service *AuthService
	logger  *logrus.Logger
}

// NewAuthHandler creates a new AuthHandler with the given service and logger.
func NewAuthHandler(service *AuthService, logger *logrus.Logger) *AuthHandler {
	return &AuthHandler{
		service: service,
		logger:  logger,
	}
}

func RegisterAuthRoutes(handler *AuthHandler, routerGroup *gin.RouterGroup) {
	authGroup := routerGroup.Group("/auth")
	{
		authGroup.GET("/github/login", handler.login)
		authGroup.GET("/github/callback", handler.loginCallback)
		authGroup.POST("/device/code", handler.DeviceCode)
		authGroup.POST("/device/token", handler.DeviceToken)
	}
}



func (h *AuthHandler) DeviceCode(c *gin.Context) {
	deviceResp, err := h.service.StartDeviceFlow(c.Request.Context())
	if err != nil {
		h.logger.Errorf("Device flow start error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "device flow start failed"})
		return
	}
	c.JSON(http.StatusOK, deviceResp)
}

func (h *AuthHandler) DeviceToken(c *gin.Context) {
	var req struct {
		DeviceCode string `json:"device_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	tokenResp, err := h.service.PollDeviceToken(c.Request.Context(), req.DeviceCode)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tokenResp)
}

// login handles the /auth/github/login endpoint. Redirects user to the OAuth provider's login page.
func (h *AuthHandler) login(c *gin.Context) {
	// In production, use a securely generated random state to prevent CSRF.
	state := "random-secure-state"
	url := h.service.GetLoginURL(state)
	c.Redirect(http.StatusFound, url)
}

// loginCallback handles the /auth/github/callback endpoint. Processes the OAuth callback and returns user info and JWT.
func (h *AuthHandler) loginCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}

	userInfo, token, err := h.service.HandleCallback(c.Request.Context(), code)
	if err != nil {
		h.logger.Errorf("OAuth callback error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth failed"})
		return
	}
	// Log user info and token for debugging (avoid logging sensitive data in production)
	h.logger.Infof("GitHub User Info: %+v", userInfo)
	h.logger.Info(token)
	c.JSON(http.StatusOK, gin.H{
		"message": "OAuth successful",
		"user":    userInfo,
		"token":   token,
	})
}
