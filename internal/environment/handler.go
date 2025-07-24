package environment

import (
	"database/sql"
	"net/http"

	environmentdb "github.com/Gkemhcs/kavach-backend/internal/environment/gen"
	appErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	iam_db "github.com/Gkemhcs/kavach-backend/internal/iam/gen"

	"github.com/Gkemhcs/kavach-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// EnvironmentHandler handles HTTP requests for environments.
// It acts as the controller for environment-related API endpoints.
type EnvironmentHandler struct {
	service *EnvironmentService
	logger  *logrus.Logger
}

// NewEnvironmentHandler creates a new EnvironmentHandler.
// Used to inject dependencies and enable testability.
func NewEnvironmentHandler(service *EnvironmentService, logger *logrus.Logger) *EnvironmentHandler {
	return &EnvironmentHandler{service, logger}
}

// RegisterEnvironmentRoutes registers environment routes under a secret group with JWT middleware.
// Centralizes route registration for maintainability and security.
func RegisterEnvironmentRoutes(handler *EnvironmentHandler,
	secretGroup *gin.RouterGroup,
	jwtMiddleware gin.HandlerFunc) {
	envGroup := secretGroup.Group("/:groupID/environments")
	envGroup.Use(jwtMiddleware)
	{
		envGroup.GET("/by-name/:envName", handler.GetEnvironmentByName)
		envGroup.POST("/", handler.Create)
		envGroup.GET("/", handler.List)
		envGroup.GET("/my", handler.ListMyEnvironments)
		envGroup.GET("/:envID", handler.Get)
		envGroup.PATCH("/:envID", handler.Update)
		envGroup.DELETE("/:envID", handler.Delete)
	}
}

// ToEnvironmentResponse converts an environment DB model to API response data.
func ToEnvironmentResponse(environment *environmentdb.GetEnvironmentByNameRow) EnvironmentResponseData {
	return EnvironmentResponseData{
		Name:          environment.Name,
		Description:   toNullableString(environment.Description),
		SecretGroupID: environment.SecretGroupID.String(),
		CreatedAt:     environment.CreatedAt,
		UpdatedAt:     environment.UpdatedAt,
		ID:            environment.ID.String(),
		OrganizationID: environment.OrganizationID.String(),
	}
}


func ToListAccessibleEnvironmentsRow(env iam_db.ListAccessibleEnvironmentsRow)ListAccessibleEnvironmentsRow{
	return ListAccessibleEnvironmentsRow{
		ID: env.ID.UUID.String(),
		Name: env.Name,
		SecretGroupName: env.SecretGroupName,
		Role: string(env.Role),
	}
}
// toNullableString safely converts sql.NullString to *string for JSON marshalling.
func toNullableString(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

// Create handles POST /org/:orgID/secret-group/:groupID/env
// Creates a new environment under a secret group.
func (h *EnvironmentHandler) Create(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	groupID := c.Param("groupID")

	var req CreateEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "bad_request", "invalid request")
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
			utils.RespondError(c, http.StatusConflict, appErrors.ErrDuplicateEnvironment.Code, appErrors.ErrDuplicateEnvironment.Message)
			return
		case appErrors.ErrInternalServer:
			utils.RespondError(c, http.StatusInternalServerError, appErrors.ErrInternalServer.Code, appErrors.ErrInternalServer.Message)
			return
		case appErrors.ErrEnvironmenNameNotAllowed:
			utils.RespondError(c, http.StatusInternalServerError, appErrors.ErrEnvironmenNameNotAllowed.Code, appErrors.ErrEnvironmenNameNotAllowed.Message)
			return
		default:
			utils.RespondError(c, http.StatusInternalServerError, "internal_error", "could not create environment")
			return
		}
	}
	utils.RespondSuccess(c, http.StatusCreated, env)
}

// List handles GET /org/:orgID/secret-group/:groupID/env
// Lists all environments under a secret group.
func (h *EnvironmentHandler) List(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	groupID := c.Param("groupID")
	envs, err := h.service.ListEnvironments(c.Request.Context(), userID, orgID, groupID)
	if err != nil {
		h.logger.Error("ListEnvironments error: ", err)
		switch err {
		case appErrors.ErrInternalServer:
			utils.RespondError(c, http.StatusInternalServerError, appErrors.ErrInternalServer.Code, appErrors.ErrInternalServer.Message)
			return
		default:
			utils.RespondError(c, http.StatusInternalServerError, "internal_error", "could not list environments")
			return
		}
	}
	utils.RespondSuccess(c, http.StatusOK, envs)
}

// ListMyEnvironments handles GET /organizations/:orgID/secret-groups/:groupID/env/my
// Lists all environments where the user is a member.
func (h *EnvironmentHandler) ListMyEnvironments(c *gin.Context) {
	userID := c.GetString("user_id")
	groupId := c.Param("groupID")
	orgID := c.Param("orgID")
	envs, err := h.service.ListMyEnvironments(c.Request.Context(), userID, groupId, orgID)
	if err != nil {
		h.logger.Error("ListMyEnvironments error: ", err)
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "could not list environments")
		return
	}
	var listEnvRows []ListAccessibleEnvironmentsRow

	for _,env := range envs{
		listEnvRows=append(listEnvRows,ToListAccessibleEnvironmentsRow(env))
	}

	utils.RespondSuccess(c, http.StatusOK, listEnvRows)
}

// Get handles GET /org/:orgID/secret-group/:groupID/env/:envID
// Gets a specific environment by ID under a secret group.
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
			utils.RespondError(c, http.StatusNotFound, appErrors.ErrNotFound.Code, appErrors.ErrNotFound.Message)
			return
		case appErrors.ErrInternalServer:
			utils.RespondError(c, http.StatusInternalServerError, appErrors.ErrInternalServer.Code, appErrors.ErrInternalServer.Message)
			return
		default:
			utils.RespondError(c, http.StatusInternalServerError, "internal_error", "could not get environment")
			return
		}
	}
	utils.RespondSuccess(c, http.StatusOK, env)
}

// GetEnvironmentByName handles GET /org/:orgID/secret-group/:groupID/env/by-name/:envName
// Gets a specific environment by name under a secret group.
func (h *EnvironmentHandler) GetEnvironmentByName(c *gin.Context) {
	groupID := c.Param("groupID")
	envName := c.Param("envName")
	environment, err := h.service.GetEnvironmentByName(c.Request.Context(), envName, groupID)

	if err == appErrors.ErrEnvironmentNotFound {
		apiErr, _ := err.(*appErrors.APIError)
		h.logger.Errorf("secretgroup  is not found with name %s", envName)
		utils.RespondError(c, http.StatusBadRequest, apiErr.Code, apiErr.Message)
		return
	}
	if err != nil {
		h.logger.Errorf("%v", err)
		utils.RespondError(c, http.StatusBadRequest, "internal_error", "internal server error")
		return
	}

	h.logger.Info("Request succeeded successfully")
	utils.RespondSuccess(c, http.StatusOK, environment)

}

// Update handles PATCH /org/:orgID/secret-group/:groupID/env/:envID
// Updates an environment by ID under a secret group.
func (h *EnvironmentHandler) Update(c *gin.Context) {
	userID := c.GetString("user_id")
	orgID := c.Param("orgID")
	groupID := c.Param("groupID")
	envID := c.Param("envID")
	var req UpdateEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "bad_request", "invalid request")
		return
	}
	env, err := h.service.UpdateEnvironment(c.Request.Context(), userID, orgID, groupID, envID, req)
	if err != nil {
		h.logger.Error("UpdateEnvironment error: ", err)
		switch err {
		case appErrors.ErrNotFound:
			utils.RespondError(c, http.StatusNotFound, appErrors.ErrNotFound.Code, appErrors.ErrNotFound.Message)
			return
		case appErrors.ErrInternalServer:
			utils.RespondError(c, http.StatusInternalServerError, appErrors.ErrInternalServer.Code, appErrors.ErrInternalServer.Message)
			return
		default:
			utils.RespondError(c, http.StatusInternalServerError, "internal_error", "could not update environment")
			return
		}
	}
	utils.RespondSuccess(c, http.StatusOK, env)
}

// Delete handles DELETE /org/:orgID/secret-group/:groupID/env/:envID
// Deletes an environment by ID under a secret group.
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
			utils.RespondError(c, http.StatusNotFound, appErrors.ErrNotFound.Code, appErrors.ErrNotFound.Message)
			return
		case appErrors.ErrInternalServer:
			utils.RespondError(c, http.StatusInternalServerError, appErrors.ErrInternalServer.Code, appErrors.ErrInternalServer.Message)
			return
		default:
			utils.RespondError(c, http.StatusInternalServerError, "internal_error", "could not delete environment")
			return
		}
	}
	utils.RespondSuccess(c, http.StatusOK, gin.H{"message": "environment deleted"})
}
