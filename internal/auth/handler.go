package auth

import (
	"net/http"

	apperrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AuthHandler handles HTTP requests related to authentication.
// It acts as the controller layer, delegating logic to AuthService and formatting HTTP responses.
type AuthHandler struct {
	service *AuthService
	logger  *logrus.Logger
}

// NewAuthHandler creates a new AuthHandler with the given service and logger.
// This is used to inject dependencies and enable testability.
func NewAuthHandler(service *AuthService, logger *logrus.Logger) *AuthHandler {
	return &AuthHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterAuthRoutes registers all authentication-related routes under /auth.
// This function centralizes route registration for maintainability.
func RegisterAuthRoutes(handler *AuthHandler, routerGroup *gin.RouterGroup) {
	authGroup := routerGroup.Group("/auth")
	{
		authGroup.GET("/github/login", handler.login)
		authGroup.GET("/github/callback", handler.loginCallback)
		authGroup.POST("/device/code", handler.DeviceCode)
		authGroup.POST("/device/token", handler.DeviceToken)
		authGroup.POST("/refresh", handler.RefreshToken)
	}
}

// DeviceCode starts the OAuth device flow and returns device/user codes for CLI login.
// This enables device-based authentication for non-browser clients.
func (h *AuthHandler) DeviceCode(c *gin.Context) {
	deviceResp, err := h.service.StartDeviceFlow(c.Request.Context())
	if err != nil {
		h.logger.Errorf("Device flow start error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "device flow start failed"})
		return
	}
	c.JSON(http.StatusOK, deviceResp)
}

// DeviceToken polls for a device flow token using the device code.
// This endpoint is used by CLI clients to complete device login.
func (h *AuthHandler) DeviceToken(c *gin.Context) {
	var req struct {
		DeviceCode string `json:"device_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "bad_request", "invalid request")
		return
	}
	tokenResp, err := h.service.PollDeviceToken(c.Request.Context(), req.DeviceCode)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tokenResp)
}

// login handles the /auth/github/login endpoint.
// Redirects user to the OAuth provider's login page. Used for browser-based login.
func (h *AuthHandler) login(c *gin.Context) {
	// In production, use a securely generated random state to prevent CSRF.
	state := "random-secure-state"
	url := h.service.GetLoginURL(state)
	c.Redirect(http.StatusFound, url)
}

// loginCallback handles the /auth/github/callback endpoint.
// Processes the OAuth callback, upserts user, and returns user info and JWT.
func (h *AuthHandler) loginCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}

	userInfo, token, refreshToken, err := h.service.HandleCallback(c.Request.Context(), code)
	if err != nil {
		h.logger.Errorf("OAuth callback error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OAuth failed"})
		return
	}
	// Log user info and token for debugging (avoid logging sensitive data in production)
	h.logger.Infof("GitHub User Info: %+v", userInfo)
	h.logger.Info(token)
	c.JSON(http.StatusOK, gin.H{
		"message":       "OAuth successful",
		"user":          userInfo,
		"token":         token,
		"refresh_token": refreshToken,
	})
}

// RefreshToken handles the /auth/refresh endpoint.
// Accepts a refresh token and issues a new access and refresh token. Used for session renewal.
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		utils.RespondError(c, http.StatusBadRequest, "bad_request", "invalid request")
		return
	}
	token, refreshToken, err := h.service.RefreshTokens(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if err == apperrors.ErrExpiredToken {
			h.logger.Warnf("Expired refresh token: %v", err)
			utils.RespondError(c, http.StatusUnauthorized, apperrors.ErrExpiredToken.Code, "expired refresh token")
			return
		}
		if err == apperrors.ErrInvalidToken {
			h.logger.Warnf("Invalid refresh token: %v", err)
			utils.RespondError(c, http.StatusUnauthorized, apperrors.ErrInvalidToken.Code, "invalid refresh token")
			return
		}
		h.logger.Errorf("Refresh token error: %v", err)
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "could not refresh token")
		return
	}
	utils.RespondSuccess(c, http.StatusOK, gin.H{
		"token":         token,
		"refresh_token": refreshToken,
	})
}
