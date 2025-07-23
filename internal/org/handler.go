package org

import (
	"database/sql"
	"net/http"
	"time"

	environmentdb "github.com/Gkemhcs/kavach-backend/internal/environment/gen"
	apiErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	orgdb "github.com/Gkemhcs/kavach-backend/internal/org/gen"
	"github.com/Gkemhcs/kavach-backend/internal/secretgroup"
	secretgroupdb "github.com/Gkemhcs/kavach-backend/internal/secretgroup/gen"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// OrganizationHandler handles HTTP requests for organizations.
// Acts as the controller for organization-related API endpoints.
type OrganizationHandler struct {
	service *OrganizationService
	logger  *logrus.Logger
}

// NewOrganizationHandler creates a new OrganizationHandler.
// Used to inject dependencies and enable testability.
func NewOrganizationHandler(service *OrganizationService,
	logger *logrus.Logger,
) *OrganizationHandler {
	return &OrganizationHandler{service, logger}
}

// RegisterOrganizationRoutes registers organization routes with JWT middleware.
// Centralizes route registration for maintainability and security.
func RegisterOrganizationRoutes(handler *OrganizationHandler,
	routerGroup *gin.RouterGroup,
	secretGroupRepo secretgroupdb.Querier,
	environmentRepo environmentdb.Querier,
	jwtMiddleware gin.HandlerFunc) {
	orgGroup := routerGroup.Group("/organizations")
	orgGroup.Use(jwtMiddleware)

	// Register secret group routes FIRST to avoid Gin wildcard conflicts
	secretGroupService := secretgroup.NewSecretGroupService(secretGroupRepo, handler.logger)
	secretGroupHandler := secretgroup.NewSecretGroupHandler(secretGroupService, handler.logger, handler.service)
	secretgroup.RegisterSecretGroupRoutes(secretGroupHandler, orgGroup, environmentRepo, jwtMiddleware)

	// Now register organization routes
	orgGroup.GET("/by-name/:orgName", handler.GetOrganizationByName)
	orgGroup.DELETE("/by-name/:orgName", handler.DeleteOrganization)
	orgGroup.GET("/my", handler.ListMyOrganizations)
	orgGroup.POST("/", handler.CreateOrganization)
	orgGroup.GET("/", handler.ListOrganizations)
	orgGroup.GET(":orgID", handler.GetOrganization)
	orgGroup.PUT(":orgID", handler.UpdateOrganization)

}

// ToOrganizationResponse converts an organization DB model to API response data.
func ToOrganizationResponse(org *orgdb.Organization) OrganizationResponseData {
	return OrganizationResponseData{
		ID:          org.ID.String(),
		Name:        org.Name,
		CreatedAt:   org.CreatedAt.Format(time.RFC3339),
		Description: toNullableString(org.Description),
	}
}

// ToOrganisationMemberResponse converts a DB row to API response for organization membership.
func ToOrganisationMemberResponse(org orgdb.ListOrganizationsWithMemberRow) ListOrganizationsWithMemberRow {
	return ListOrganizationsWithMemberRow{
		OrgID: org.OrgID,
		Name:  org.Name,
		Role:  string(org.Role),
	}
}

// toNullableString safely converts sql.NullString to *string for JSON marshalling.
func toNullableString(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// CreateOrganization handles POST /organizations
// Creates a new organization for the user.
func (h *OrganizationHandler) CreateOrganization(c *gin.Context) {
	userID := c.GetString("user_id")
	var req CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "bad_request", "invalid request")
		return
	}
	req.UserID = userID
	org, err := h.service.CreateOrganization(c.Request.Context(), req)
	if err != nil && err == apiErrors.ErrDuplicateOrganization {
		h.logger.Error("Organisation already exists")
		apiErr, _ := err.(*apiErrors.APIError)

		utils.RespondError(c, http.StatusBadRequest, apiErr.Code, apiErr.Message)
	}
	if err != nil {
		h.logger.Error("CreateOrganization error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respData := ToOrganizationResponse(org)
	utils.RespondSuccess(c, http.StatusCreated, respData)
}

// ListOrganizations handles GET /organizations
// Lists all organizations for the user.
func (h *OrganizationHandler) ListOrganizations(c *gin.Context) {
	userID := c.GetString("user_id")
	orgs, err := h.service.ListOrganizations(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("ListOrganizations error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "could not list organizations")
		return
	}
	utils.RespondSuccess(c, http.StatusOK, orgs)
}

// ListMyOrganizations handles GET /organizations/my
// Lists all organizations where the user is a member.
func (h *OrganizationHandler) ListMyOrganizations(c *gin.Context) {
	userId := c.GetString("user_id")
	orgs, err := h.service.ListMyOrganizations(c.Request.Context(), userId)

	if err != nil {
		h.logger.Error("ListOrganizations error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "could not list organizations")
		return
	}
	var respData []ListOrganizationsWithMemberRow
	for _, org := range orgs {
		respData = append(respData, ToOrganisationMemberResponse(org))
	}
	utils.RespondSuccess(c, http.StatusOK, respData)
}

// GetOrganizationByName handles GET /organizations/by-name/:orgName
// Gets a specific organization by name for the user.
func (h *OrganizationHandler) GetOrganizationByName(c *gin.Context) {
	orgName := c.Param("orgName")
	userID := c.GetString("user_id")
	org, err := h.service.GetOrganizationByName(c.Request.Context(), orgName, userID)
	if err == apiErrors.ErrOrganizationNotFound {
		apiErr, _ := err.(*apiErrors.APIError)
		h.logger.Errorf("organisation is not found with name %s", orgName)
		utils.RespondError(c, http.StatusBadRequest, apiErr.Code, apiErr.Message)
		return
	}
	if err != nil {
		h.logger.Errorf("%v", err)
		utils.RespondError(c, http.StatusBadRequest, "internal_error", "internal server error")
		return
	}
	h.logger.Info("Request succeeded successfully")
	h.logger.Info(org.ID)
	utils.RespondSuccess(c, http.StatusOK, ToOrganizationResponse(org))

}

// GetOrganization handles GET /organizations/:id
// Gets a specific organization by ID for the user.
func (h *OrganizationHandler) GetOrganization(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	org, err := h.service.GetOrganization(c.Request.Context(), userID, orgID)
	if err != nil {
		h.logger.Error("GetOrganization error: ", err)
		utils.RespondError(c, http.StatusNotFound, "not_found", "organization not found")
		return
	}
	utils.RespondSuccess(c, http.StatusOK, org)
}

// UpdateOrganization handles PUT /organizations/:id
// Updates an organization by ID for the user.
func (h *OrganizationHandler) UpdateOrganization(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	var req UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "bad_request", "invalid request")
		return
	}
	org, err := h.service.UpdateOrganization(c.Request.Context(), userID, orgID, req)
	if err != nil {
		h.logger.Error("UpdateOrganization error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "could not update organization")
		return
	}
	utils.RespondSuccess(c, http.StatusOK, org)
}

// DeleteOrganization handles DELETE /organizations/:id
// Deletes an organization by name for the user.
func (h *OrganizationHandler) DeleteOrganization(c *gin.Context) {
	userID := c.GetString("user_id")
	orgName := c.Param("orgName")

	org, err := h.service.GetOrganizationByName(c.Request.Context(), orgName, userID)
	if err == apiErrors.ErrOrganizationNotFound {
		apiErr, _ := err.(*apiErrors.APIError)
		h.logger.Errorf("organisation is not found with name")
		utils.RespondError(c, http.StatusBadRequest, apiErr.Code, apiErr.Message)
		return
	}

	err = h.service.DeleteOrganization(c.Request.Context(), org.ID)

	if err != nil {
		h.logger.Error("DeleteOrganization error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "could not delete organization")
		return
	}
	utils.RespondSuccess(c, http.StatusOK, gin.H{"message": "organization deleted"})
}
