package secretgroup

import (
	"net/http"

	"github.com/Gkemhcs/kavach-backend/internal/environment"
	environmentdb "github.com/Gkemhcs/kavach-backend/internal/environment/gen"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SecretGroupHandler handles HTTP requests for secret groups.
type SecretGroupHandler struct {
	service *SecretGroupService
	logger  *logrus.Logger
}

// NewSecretGroupHandler creates a new SecretGroupHandler.
func NewSecretGroupHandler(service *SecretGroupService, logger *logrus.Logger) *SecretGroupHandler {
	return &SecretGroupHandler{service, logger}
}

// RegisterSecretGroupRoutes registers secret group routes under an organization with JWT middleware.
func RegisterSecretGroupRoutes(handler *SecretGroupHandler,
	orgGroup *gin.RouterGroup,
	environmentRepo environmentdb.Querier,
	jwtMiddleware gin.HandlerFunc) {
	// Register under /organizations/:orgID/secret-groups
	secretGroup := orgGroup.Group(":orgID/secret-groups")
	secretGroup.Use(jwtMiddleware)
	{
		secretGroup.POST("/", handler.Create)
		secretGroup.GET("/", handler.List)
		secretGroup.GET("/my", handler.ListMySecretGroups)
		secretGroup.GET(":groupID", handler.Get)
		secretGroup.PATCH(":groupID", handler.Update)
		secretGroup.DELETE(":groupID", handler.Delete)
	}

	enviromentService := environment.NewEnvironmentService(environmentRepo, handler.logger)
	environmentHandler := environment.NewEnvironmentHandler(enviromentService, handler.logger)
	// Register environment routes under /organizations/:orgID/secret-groups/:groupID/environments
	environment.RegisterEnvironmentRoutes(environmentHandler, secretGroup, jwtMiddleware)
}

// Create handles POST /org/:orgID/secret-group
func (h *SecretGroupHandler) Create(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	var req CreateSecretGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid request")
		return
	}
	req.OrganizationID = orgID
	req.UserID = userID
	group, err := h.service.CreateSecretGroup(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("CreateSecretGroup error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, "could not create secret group")
		return
	}
	utils.RespondSuccess(c, group)
}

// List handles GET /org/:orgID/secret-group
func (h *SecretGroupHandler) List(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	groups, err := h.service.ListSecretGroups(c.Request.Context(), userID, orgID)
	if err != nil {
		h.logger.Error("ListSecretGroups error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, "could not list secret groups")
		return
	}
	utils.RespondSuccess(c, groups)
}

// ListMySecretGroups handles GET /organizations/:orgID/secret-groups/my
func (h *SecretGroupHandler) ListMySecretGroups(c *gin.Context) {
	userID := c.GetString("user_id")
	groups, err := h.service.ListMySecretGroups(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("ListMySecretGroups error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, "could not list secret groups")
		return
	}
	utils.RespondSuccess(c, groups)
}

// Get handles GET /org/:orgID/secret-group/:groupID
func (h *SecretGroupHandler) Get(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	groupID := c.Param("groupID")
	group, err := h.service.GetSecretGroup(c.Request.Context(), userID, orgID, groupID)
	if err != nil {
		h.logger.Error("GetSecretGroup error: ", err)
		utils.RespondError(c, http.StatusNotFound, "secret group not found")
		return
	}
	utils.RespondSuccess(c, group)
}

// Update handles PATCH /org/:orgID/secret-group/:groupID
func (h *SecretGroupHandler) Update(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	groupID := c.Param("groupID")
	var req UpdateSecretGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid request")
		return
	}
	group, err := h.service.UpdateSecretGroup(c.Request.Context(), userID, orgID, groupID, req)
	if err != nil {
		h.logger.Error("UpdateSecretGroup error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, "could not update secret group")
		return
	}
	utils.RespondSuccess(c, group)
}

// Delete handles DELETE /org/:orgID/secret-group/:groupID
func (h *SecretGroupHandler) Delete(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	groupID := c.Param("groupID")
	err := h.service.DeleteSecretGroup(c.Request.Context(), userID, orgID, groupID)
	if err != nil {
		h.logger.Error("DeleteSecretGroup error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, "could not delete secret group")
		return
	}
	utils.RespondSuccess(c, gin.H{"message": "secret group deleted"})
}
