package org

import (
	"net/http"

	environmentdb "github.com/Gkemhcs/kavach-backend/internal/environment/gen"
	"github.com/Gkemhcs/kavach-backend/internal/secretgroup"
	secretgroupdb "github.com/Gkemhcs/kavach-backend/internal/secretgroup/gen"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// OrganizationHandler handles HTTP requests for organizations.
type OrganizationHandler struct {
	service *OrganizationService
	logger  *logrus.Logger
}

// NewOrganizationHandler creates a new OrganizationHandler.
func NewOrganizationHandler(service *OrganizationService,
	logger *logrus.Logger,
) *OrganizationHandler {
	return &OrganizationHandler{service, logger}
}

// RegisterOrganizationRoutes registers organization routes with JWT middleware.
func RegisterOrganizationRoutes(handler *OrganizationHandler,
	routerGroup *gin.RouterGroup,
	secretGroupRepo secretgroupdb.Querier,
	environmentRepo environmentdb.Querier,
	jwtMiddleware gin.HandlerFunc) {
	orgGroup := routerGroup.Group("/organizations")
	orgGroup.Use(jwtMiddleware)

	// Register secret group routes FIRST to avoid Gin wildcard conflicts
	secretGroupService := secretgroup.NewSecretGroupService(secretGroupRepo, handler.logger)
	secretGroupHandler := secretgroup.NewSecretGroupHandler(secretGroupService, handler.logger)
	secretgroup.RegisterSecretGroupRoutes(secretGroupHandler, orgGroup, environmentRepo, jwtMiddleware)

	// Now register organization routes
	orgGroup.POST("/", handler.CreateOrganization)
	orgGroup.GET("/", handler.ListOrganizations)
	orgGroup.GET(":orgID", handler.GetOrganization)
	orgGroup.PUT(":orgID", handler.UpdateOrganization)
	orgGroup.GET("/my", handler.ListMyOrganizations)
	orgGroup.DELETE(":orgID", handler.DeleteOrganization)
}

// CreateOrganization handles POST /organizations
func (h *OrganizationHandler) CreateOrganization(c *gin.Context) {
	userID := c.GetString("user_id")
	var req CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid request")
		return
	}
	req.UserID = userID
	org, err := h.service.CreateOrganization(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("CreateOrganization error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondSuccess(c, org)
}

// ListOrganizations handles GET /organizations
func (h *OrganizationHandler) ListOrganizations(c *gin.Context) {
	userID := c.GetString("user_id")
	orgs, err := h.service.ListOrganizations(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("ListOrganizations error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, "could not list organizations")
		return
	}
	utils.RespondSuccess(c, orgs)
}

func(h *OrganizationHandler)ListMyOrganizations(c *gin.Context){
	userId:=c.GetString("user_id")
	orgs,err:=h.service.ListMyOrganizations(c.Request.Context(),userId)
	if err!=nil{
		h.logger.Error("ListOrganizations error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, "could not list organizations")
		return
	}
	utils.RespondSuccess(c, orgs)
}

// GetOrganization handles GET /organizations/:id
func (h *OrganizationHandler) GetOrganization(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	org, err := h.service.GetOrganization(c.Request.Context(), userID, orgID)
	if err != nil {
		h.logger.Error("GetOrganization error: ", err)
		utils.RespondError(c, http.StatusNotFound, "organization not found")
		return
	}
	utils.RespondSuccess(c, org)
}

// UpdateOrganization handles PUT /organizations/:id
func (h *OrganizationHandler) UpdateOrganization(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	var req UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "invalid request")
		return
	}
	org, err := h.service.UpdateOrganization(c.Request.Context(), userID, orgID, req)
	if err != nil {
		h.logger.Error("UpdateOrganization error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, "could not update organization")
		return
	}
	utils.RespondSuccess(c, org)
}

// DeleteOrganization handles DELETE /organizations/:id
func (h *OrganizationHandler) DeleteOrganization(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	err := h.service.DeleteOrganization(c.Request.Context(), userID, orgID)
	if err != nil {
		h.logger.Error("DeleteOrganization error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, "could not delete organization")
		return
	}
	utils.RespondSuccess(c, gin.H{"message": "organization deleted"})
}
