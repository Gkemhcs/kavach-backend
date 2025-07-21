package environment

import (
	"net/http"

	appErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// EnvironmentHandler handles HTTP requests for environments.
type EnvironmentHandler struct {
	service *EnvironmentService
	logger  *logrus.Logger
}

// NewEnvironmentHandler creates a new EnvironmentHandler.
func NewEnvironmentHandler(service *EnvironmentService, logger *logrus.Logger) *EnvironmentHandler {
	return &EnvironmentHandler{service, logger}
}

// RegisterEnvironmentRoutes registers environment routes under a secret group with JWT middleware.
func RegisterEnvironmentRoutes(handler *EnvironmentHandler,
	secretGroup *gin.RouterGroup,
	jwtMiddleware gin.HandlerFunc) {
	envGroup := secretGroup.Group("/:groupID/env")
	envGroup.Use(jwtMiddleware)
	{
		envGroup.POST("/", handler.Create)
		envGroup.GET("/", handler.List)
		envGroup.GET("/my", handler.ListMyEnvironments)
		envGroup.GET("/:envID", handler.Get)
		envGroup.PATCH("/:envID", handler.Update)
		envGroup.DELETE("/:envID", handler.Delete)
	}
}

// Create handles POST /org/:orgID/secret-group/:groupID/env
func (h *EnvironmentHandler) Create(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	groupID := c.Param("groupID")

	var req CreateEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid request")
		return
	}
	req.Organization = orgID
	req.SecretGroup = groupID
	req.UserId = userID

	env, err := h.service.CreateEnvironment(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("CreateEnvironment error: ", err)
		switch err {
		case appErrors.ErrDuplicateEnvironment:
			utils.RespondError(c, http.StatusConflict, appErrors.ErrDuplicateEnvironment.Message)
			return
		case appErrors.ErrInternalServer:
			utils.RespondError(c, http.StatusInternalServerError, appErrors.ErrInternalServer.Message)
			return
		default:
			utils.RespondError(c, http.StatusInternalServerError, "could not create environment")
			return
		}
	}
	utils.RespondSuccess(c, env)
}

// List handles GET /org/:orgID/secret-group/:groupID/env
func (h *EnvironmentHandler) List(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	groupID := c.Param("groupID")
	envs, err := h.service.ListEnvironments(c.Request.Context(), userID, orgID, groupID)
	if err != nil {
		h.logger.Error("ListEnvironments error: ", err)
		switch err {
		case appErrors.ErrInternalServer:
			utils.RespondError(c, http.StatusInternalServerError, appErrors.ErrInternalServer.Message)
			return
		default:
			utils.RespondError(c, http.StatusInternalServerError, "could not list environments")
			return
		}
	}
	utils.RespondSuccess(c, envs)
}

// ListMyEnvironments handles GET /organizations/:orgID/secret-groups/:groupID/env/my
func (h *EnvironmentHandler) ListMyEnvironments(c *gin.Context) {
	userID := c.GetString("user_id")
	envs, err := h.service.ListMyEnvironments(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("ListMyEnvironments error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, "could not list environments")
		return
	}
	utils.RespondSuccess(c, envs)
}

// Get handles GET /org/:orgID/secret-group/:groupID/env/:envID
func (h *EnvironmentHandler) Get(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	groupID := c.Param("groupID")
	envID := c.Param("envID")
	env, err := h.service.GetEnvironment(c.Request.Context(), userID, orgID, groupID, envID)
	if err != nil {
		h.logger.Error("GetEnvironment error: ", err)
		switch err {
		case appErrors.ErrNotFound:
			utils.RespondError(c, http.StatusNotFound, appErrors.ErrNotFound.Message)
			return
		case appErrors.ErrInternalServer:
			utils.RespondError(c, http.StatusInternalServerError, appErrors.ErrInternalServer.Message)
			return
		default:
			utils.RespondError(c, http.StatusInternalServerError, "could not get environment")
			return
		}
	}
	utils.RespondSuccess(c, env)
}

// Update handles PATCH /org/:orgID/secret-group/:groupID/env/:envID
func (h *EnvironmentHandler) Update(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	groupID := c.Param("groupID")
	envID := c.Param("envID")
	var req UpdateEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid request")
		return
	}
	env, err := h.service.UpdateEnvironment(c.Request.Context(), userID, orgID, groupID, envID, req)
	if err != nil {
		h.logger.Error("UpdateEnvironment error: ", err)
		switch err {
		case appErrors.ErrNotFound:
			utils.RespondError(c, http.StatusNotFound, appErrors.ErrNotFound.Message)
			return
		case appErrors.ErrInternalServer:
			utils.RespondError(c, http.StatusInternalServerError, appErrors.ErrInternalServer.Message)
			return
		default:
			utils.RespondError(c, http.StatusInternalServerError, "could not update environment")
			return
		}
	}
	utils.RespondSuccess(c, env)
}

// Delete handles DELETE /org/:orgID/secret-group/:groupID/env/:envID
func (h *EnvironmentHandler) Delete(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	groupID := c.Param("groupID")
	envID := c.Param("envID")
	err := h.service.DeleteEnvironment(c.Request.Context(), userID, orgID, groupID, envID)
	if err != nil {
		h.logger.Error("DeleteEnvironment error: ", err)
		switch err {
		case appErrors.ErrNotFound:
			utils.RespondError(c, http.StatusNotFound, appErrors.ErrNotFound.Message)
			return
		case appErrors.ErrInternalServer:
			utils.RespondError(c, http.StatusInternalServerError, appErrors.ErrInternalServer.Message)
			return
		default:
			utils.RespondError(c, http.StatusInternalServerError, "could not delete environment")
			return
		}
	}
	utils.RespondSuccess(c, gin.H{"message": "environment deleted"})
}
