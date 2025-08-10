package secretgroup

import (
	"database/sql"
	"net/http"

	"github.com/Gkemhcs/kavach-backend/internal/environment"
	iam_db "github.com/Gkemhcs/kavach-backend/internal/iam/gen"
	"github.com/Gkemhcs/kavach-backend/internal/provider"
	"github.com/Gkemhcs/kavach-backend/internal/secret"

	apiErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	secretgroupdb "github.com/Gkemhcs/kavach-backend/internal/secretgroup/gen"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SecretGroupHandler handles HTTP requests for secret groups.
// Acts as the controller for secret group-related API endpoints.
type SecretGroupHandler struct {
	service *SecretGroupService

	logger *logrus.Logger
}

// NewSecretGroupHandler creates a new SecretGroupHandler.
// Used to inject dependencies and enable testability.
func NewSecretGroupHandler(service *SecretGroupService, logger *logrus.Logger) *SecretGroupHandler {
	return &SecretGroupHandler{
		service,
		logger}
}

// RegisterSecretGroupRoutes registers secret group routes under an organization with JWT middleware.
// Centralizes route registration for maintainability and security.
func RegisterSecretGroupRoutes(handler *SecretGroupHandler,
	orgGroup *gin.RouterGroup,
	environmentHandler *environment.EnvironmentHandler,
	secretHandler *secret.SecretHandler,
	providerHandler *provider.ProviderHandler,
	jwtMiddleware gin.HandlerFunc) {
	// Register under /organizations/:orgID/secret-groups
	secretGroup := orgGroup.Group(":orgID/secret-groups")
	secretGroup.Use(jwtMiddleware)
	{
		secretGroup.GET("/by-name/:groupName", handler.GetSecretGroupByName)
		secretGroup.POST("/", handler.Create)
		secretGroup.GET("/", handler.List)
		secretGroup.GET("/my", handler.ListMySecretGroups)
		secretGroup.GET(":groupID", handler.Get)
		secretGroup.PATCH(":groupID", handler.Update)
		secretGroup.DELETE(":groupID", handler.Delete)

		// Role bindings routes
		secretGroup.GET("/:groupID/role-bindings", handler.ListSecretGroupRoleBindings)
	}

	environment.RegisterEnvironmentRoutes(environmentHandler, secretHandler, providerHandler, secretGroup, jwtMiddleware)
}

// ToSecretGroupResponse converts a secret group DB model to API response data.
func ToSecretGroupResponse(secretgroup *secretgroupdb.SecretGroup) SecretGroupResponseData {
	return SecretGroupResponseData{
		ID:             secretgroup.ID.String(),
		Name:           secretgroup.Name,
		Description:    toNullableString(secretgroup.Description),
		OrganizationID: secretgroup.OrganizationID.String(),
		CreatedAt:      secretgroup.CreatedAt,
		UpdatedAt:      secretgroup.UpdatedAt,
	}
}

func ToSecretGroupRowResponse(secretgroup iam_db.ListAccessibleSecretGroupsRow) ListAccessibleSecretGroupsRow {
	return ListAccessibleSecretGroupsRow{
		ID:               secretgroup.ID.UUID.String(),
		SecretGroupName:  secretgroup.Name,
		OrganizationName: secretgroup.OrganizationName,
		Role:             string(secretgroup.Role),
		InheritedFrom:    secretgroup.InheritedFrom,
	}
}

// toNullableString safely converts sql.NullString to *string for JSON marshalling.
func toNullableString(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// Create handles POST /org/:orgID/secret-group
// Creates a new secret group under an organization.
func (h *SecretGroupHandler) Create(c *gin.Context) {
	userID := c.GetString("user_id")
	orgId := c.Param("orgID")

	var req CreateSecretGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "bad_request", "invalid request")
		return
	}
	req.UserID = userID
	req.OrganizationID = orgId

	group, err := h.service.CreateSecretGroup(c.Request.Context(), req)
	if err != nil && err == apiErrors.ErrDuplicateSecretGroup {
		h.logger.Error("Secret  Group already exists")
		apiErr, _ := err.(*apiErrors.APIError)

		utils.RespondError(c, http.StatusBadRequest, apiErr.Code, apiErr.Message)
		return
	}
	if err != nil {
		h.logger.Error("CreateSecretGroup error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "could not create secret group")
		return
	}
	utils.RespondSuccess(c, http.StatusCreated, ToSecretGroupResponse(group))
}

// List handles GET /org/:orgID/secret-group
// Lists all secret groups under an organization.
func (h *SecretGroupHandler) List(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	groups, err := h.service.ListSecretGroups(c.Request.Context(), userID, orgID)
	if err != nil {
		h.logger.Error("ListSecretGroups error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "could not list secret groups")
		return
	}
	utils.RespondSuccess(c, http.StatusOK, groups)
}

// ListMySecretGroups handles GET /organizations/:orgID/secret-groups/my
// Lists all secret groups where the user is a member.
func (h *SecretGroupHandler) ListMySecretGroups(c *gin.Context) {
	userID := c.GetString("user_id")
	orgId := c.Param("orgID")
	h.logger.Info(orgId, userID)
	groups, err := h.service.ListMySecretGroups(c.Request.Context(), orgId, userID)
	if err != nil {
		h.logger.Error("ListMySecretGroups error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "could not list secret groups")
		return
	}
	var listMyGroups []ListAccessibleSecretGroupsRow
	for _, group := range groups {
		listMyGroups = append(listMyGroups, ToSecretGroupRowResponse(group))
	}

	utils.RespondSuccess(c, http.StatusOK, listMyGroups)
}

// Get handles GET /org/:orgID/secret-group/:groupID
// Gets a specific secret group by ID under an organization.
func (h *SecretGroupHandler) Get(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	groupID := c.Param("groupID")
	group, err := h.service.GetSecretGroup(c.Request.Context(), userID, orgID, groupID)
	if err != nil {
		h.logger.Error("GetSecretGroup error: ", err)
		utils.RespondError(c, http.StatusNotFound, "not_found", "secret group not found")
		return
	}
	utils.RespondSuccess(c, http.StatusOK, group)
}

// GetSecretGroupByName handles GET /org/:orgID/secret-group/by-name/:groupName
// Gets a specific secret group by name under an organization.
func (h *SecretGroupHandler) GetSecretGroupByName(c *gin.Context) {
	orgId := c.Param("orgID")
	groupName := c.Param("groupName")
	group, err := h.service.GetSecretGroupByName(c.Request.Context(), orgId, groupName)
	if err == apiErrors.ErrSecretGroupNotFound {
		apiErr, _ := err.(*apiErrors.APIError)
		h.logger.Errorf("secretgroup  is not found with name %s", groupName)
		utils.RespondError(c, http.StatusBadRequest, apiErr.Code, apiErr.Message)
		return
	}
	if err != nil {
		h.logger.Errorf("%v", err)
		utils.RespondError(c, http.StatusBadRequest, "internal_error", "internal server error")
		return
	}
	h.logger.Info("Request succeeded successfully")
	utils.RespondSuccess(c, http.StatusOK, ToSecretGroupResponse(group))
}

// Update handles PATCH /org/:orgID/secret-group/:groupID
// Updates a secret group by ID under an organization.
func (h *SecretGroupHandler) Update(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	groupID := c.Param("groupID")
	var req UpdateSecretGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "bad_request", "invalid request")
		return
	}
	group, err := h.service.UpdateSecretGroup(c.Request.Context(), userID, orgID, groupID, req)
	if err != nil {
		h.logger.Error("UpdateSecretGroup error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "could not update secret group")
		return
	}
	utils.RespondSuccess(c, http.StatusOK, group)
}

// Delete handles DELETE /org/:orgID/secret-group/:groupID
// Deletes a secret group by ID under an organization.
func (h *SecretGroupHandler) Delete(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	groupID := c.Param("groupID")
	err := h.service.DeleteSecretGroup(c.Request.Context(), userID, orgID, groupID)
	if err != nil {
		if err == apiErrors.ErrForeignKeyViolation {
			apiErr := err.(*apiErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		}

		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "could not delete secret group")
		return
	}
	utils.RespondSuccess(c, http.StatusOK, gin.H{"message": "secret group deleted"})
}

// ListSecretGroupRoleBindings handles GET /organizations/:orgID/secret-groups/:groupID/role-bindings
// Lists all role bindings for a secret group with resolved user and group names.
func (h *SecretGroupHandler) ListSecretGroupRoleBindings(c *gin.Context) {
	orgID := c.Param("orgID")
	groupID := c.Param("groupID")
	if orgID == "" || groupID == "" {
		h.logger.Warn("List secret group role bindings failed: missing organization ID or group ID")
		utils.RespondError(c, http.StatusBadRequest, "missing_parameters", "organization ID and group ID are required")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"orgID":   orgID,
		"groupID": groupID,
	}).Info("Listing secret group role bindings")

	bindings, err := h.service.ListSecretGroupRoleBindings(c.Request.Context(), orgID, groupID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"orgID":   orgID,
			"groupID": groupID,
			"error":   err.Error(),
		}).Error("Failed to list secret group role bindings")

		// Handle specific error types
		if err == apiErrors.ErrOrganizationNotFound {
			apiErr := err.(*apiErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		}

		if err == apiErrors.ErrSecretGroupNotFound {
			apiErr := err.(*apiErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		}

		if err == apiErrors.ErrPermissionDeniedForRoleBindings {
			apiErr := err.(*apiErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		}

		if err == apiErrors.ErrNoRoleBindingsFound {
			apiErr := err.(*apiErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		}

		if err == apiErrors.ErrInvalidResourceID {
			apiErr := err.(*apiErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		}

		// Default error
		utils.RespondError(c, http.StatusInternalServerError, "internal_server_error", "failed to list secret group role bindings")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"orgID":        orgID,
		"groupID":      groupID,
		"bindingCount": len(bindings),
	}).Info("Successfully listed secret group role bindings")

	utils.RespondSuccess(c, http.StatusOK, gin.H{
		"organization_id": orgID,
		"secret_group_id": groupID,
		"bindings":        bindings,
		"count":           len(bindings),
	})
}
