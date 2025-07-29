package provider

import (
	"fmt"
	"net/http"

	appErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ProviderHandler handles HTTP requests for provider credentials and sync operations
type ProviderHandler struct {
	service *ProviderService
	logger  *logrus.Logger
}

// NewProviderHandler creates a new ProviderHandler
func NewProviderHandler(service *ProviderService, logger *logrus.Logger) *ProviderHandler {
	return &ProviderHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterProviderRoutes registers provider routes under an environment
func RegisterProviderRoutes(handler *ProviderHandler, envGroup *gin.RouterGroup) {
	providerGroup := envGroup.Group("/:envID/providers")
	{
		// Provider credentials CRUD operations
		providerGroup.POST("/credentials", handler.CreateProviderCredential)
		providerGroup.GET("/credentials", handler.ListProviderCredentials)
		providerGroup.GET("/credentials/:provider", handler.GetProviderCredential)
		providerGroup.PUT("/credentials/:provider", handler.UpdateProviderCredential)
		providerGroup.DELETE("/credentials/:provider", handler.DeleteProviderCredential)

	}
}

// CreateProviderCredential handles POST /orgs/:orgID/secret-groups/:groupID/environments/:envID/providers/credentials
func (h *ProviderHandler) CreateProviderCredential(c *gin.Context) {
	environmentID := c.Param("envID")
	userID := c.GetString("user_id")

	logEntry := h.logger.WithFields(logrus.Fields{
		"handler":        "CreateProviderCredential",
		"environment_id": environmentID,
		"method":         c.Request.Method,
		"path":           c.Request.URL.Path,
	})

	logEntry.Info("Processing create provider credential request")

	var req CreateProviderCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to bind request body")
		utils.RespondError(c, appErrors.ErrInvalidBody.Status, appErrors.ErrInvalidBody.Code, appErrors.ErrInvalidBody.Message)
		return
	}

	logEntry.WithFields(logrus.Fields{
		"provider": req.Provider,
	}).Info("Request validated successfully")

	result, err := h.service.CreateProviderCredential(c.Request.Context(), environmentID, userID, req)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to create provider credential")
		switch err {
		case appErrors.ErrInvalidProviderType:
			utils.RespondError(c, appErrors.ErrInvalidProviderType.Status, appErrors.ErrInvalidProviderType.Code, appErrors.ErrInvalidProviderType.Message)
			return
		case appErrors.ErrInvalidProviderData:
			utils.RespondError(c, appErrors.ErrInvalidProviderData.Status, appErrors.ErrInvalidProviderData.Code, appErrors.ErrInvalidProviderData.Message)
			return
		case appErrors.ErrProviderCredentialExists:
			utils.RespondError(c, appErrors.ErrProviderCredentialExists.Status, appErrors.ErrProviderCredentialExists.Code, appErrors.ErrProviderCredentialExists.Message)
			return
		case appErrors.ErrProviderEncryptionFailed:
			utils.RespondError(c, appErrors.ErrProviderEncryptionFailed.Status, appErrors.ErrProviderEncryptionFailed.Code, appErrors.ErrProviderEncryptionFailed.Message)
			return
		case appErrors.ErrProviderCredentialCreateFailed:
			utils.RespondError(c, appErrors.ErrProviderCredentialCreateFailed.Status, appErrors.ErrProviderCredentialCreateFailed.Code, appErrors.ErrProviderCredentialCreateFailed.Message)
			return
		case appErrors.ErrInternalServer:
			utils.RespondError(c, appErrors.ErrInternalServer.Status, appErrors.ErrInternalServer.Code, appErrors.ErrInternalServer.Message)
			return
		default:
			utils.RespondError(c, http.StatusInternalServerError, "create_provider_credential_failed", err.Error())
			return
		}
	}

	logEntry.WithFields(logrus.Fields{
		"credential_id": result.ID,
		"provider":      result.Provider,
	}).Info("Successfully created provider credential")

	utils.RespondSuccess(c, http.StatusCreated, result)
}

// GetProviderCredential handles GET /orgs/:orgID/secret-groups/:groupID/environments/:envID/providers/credentials/:provider
func (h *ProviderHandler) GetProviderCredential(c *gin.Context) {
	environmentID := c.Param("envID")
	provider := c.Param("provider")

	logEntry := h.logger.WithFields(logrus.Fields{
		"handler":        "GetProviderCredential",
		"environment_id": environmentID,
		"provider":       provider,
		"method":         c.Request.Method,
		"path":           c.Request.URL.Path,
	})

	logEntry.Info("Processing get provider credential request")

	result, err := h.service.GetProviderCredential(c.Request.Context(), environmentID, provider)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to get provider credential")
		switch err {
		case appErrors.ErrProviderCredentialNotFound:
			utils.RespondError(c, appErrors.ErrProviderCredentialNotFound.Status, appErrors.ErrProviderCredentialNotFound.Code, appErrors.ErrProviderCredentialNotFound.Message)
			return
		case appErrors.ErrProviderCredentialGetFailed:
			utils.RespondError(c, appErrors.ErrProviderCredentialGetFailed.Status, appErrors.ErrProviderCredentialGetFailed.Code, appErrors.ErrProviderCredentialGetFailed.Message)
			return
		case appErrors.ErrInternalServer:
			utils.RespondError(c, appErrors.ErrInternalServer.Status, appErrors.ErrInternalServer.Code, appErrors.ErrInternalServer.Message)
			return
		default:
			utils.RespondError(c, http.StatusInternalServerError, "get_provider_credential_failed", err.Error())
			return
		}
	}

	logEntry.Info("Successfully retrieved provider credential")

	utils.RespondSuccess(c, http.StatusOK, result)
}

// ListProviderCredentials handles GET /orgs/:orgID/secret-groups/:groupID/environments/:envID/providers/credentials
func (h *ProviderHandler) ListProviderCredentials(c *gin.Context) {
	environmentID := c.Param("envID")

	logEntry := h.logger.WithFields(logrus.Fields{
		"handler":        "ListProviderCredentials",
		"environment_id": environmentID,
		"method":         c.Request.Method,
		"path":           c.Request.URL.Path,
	})

	logEntry.Info("Processing list provider credentials request")

	result, err := h.service.ListProviderCredentials(c.Request.Context(), environmentID)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to list provider credentials")
		switch err {
		case appErrors.ErrProviderCredentialListFailed:
			utils.RespondError(c, appErrors.ErrProviderCredentialListFailed.Status, appErrors.ErrProviderCredentialListFailed.Code, appErrors.ErrProviderCredentialListFailed.Message)
			return
		case appErrors.ErrInternalServer:
			utils.RespondError(c, appErrors.ErrInternalServer.Status, appErrors.ErrInternalServer.Code, appErrors.ErrInternalServer.Message)
			return
		default:
			utils.RespondError(c, http.StatusInternalServerError, "list_provider_credentials_failed", err.Error())
			return
		}
	}

	logEntry.WithField("count", len(result)).Info("Successfully listed provider credentials")

	utils.RespondSuccess(c, http.StatusOK, result)
}

// UpdateProviderCredential handles PUT /orgs/:orgID/secret-groups/:groupID/environments/:envID/providers/credentials/:provider
func (h *ProviderHandler) UpdateProviderCredential(c *gin.Context) {
	environmentID := c.Param("envID")
	provider := c.Param("provider")

	logEntry := h.logger.WithFields(logrus.Fields{
		"handler":        "UpdateProviderCredential",
		"environment_id": environmentID,
		"provider":       provider,
		"method":         c.Request.Method,
		"path":           c.Request.URL.Path,
	})

	logEntry.Info("Processing update provider credential request")

	var req UpdateProviderCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to bind request body")
		utils.RespondError(c, appErrors.ErrInvalidBody.Status, appErrors.ErrInvalidBody.Code, appErrors.ErrInvalidBody.Message)
		return
	}

	logEntry.Info("Request validated successfully")

	result, err := h.service.UpdateProviderCredential(c.Request.Context(), environmentID, provider, req)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to update provider credential")
		switch err {
		case appErrors.ErrProviderCredentialNotFound:
			utils.RespondError(c, appErrors.ErrProviderCredentialNotFound.Status, appErrors.ErrProviderCredentialNotFound.Code, appErrors.ErrProviderCredentialNotFound.Message)
			return
		case appErrors.ErrInvalidProviderType:
			utils.RespondError(c, appErrors.ErrInvalidProviderType.Status, appErrors.ErrInvalidProviderType.Code, appErrors.ErrInvalidProviderType.Message)
			return
		case appErrors.ErrInvalidProviderData:
			utils.RespondError(c, appErrors.ErrInvalidProviderData.Status, appErrors.ErrInvalidProviderData.Code, appErrors.ErrInvalidProviderData.Message)
			return
		case appErrors.ErrProviderEncryptionFailed:
			utils.RespondError(c, appErrors.ErrProviderEncryptionFailed.Status, appErrors.ErrProviderEncryptionFailed.Code, appErrors.ErrProviderEncryptionFailed.Message)
			return
		case appErrors.ErrProviderCredentialUpdateFailed:
			utils.RespondError(c, appErrors.ErrProviderCredentialUpdateFailed.Status, appErrors.ErrProviderCredentialUpdateFailed.Code, appErrors.ErrProviderCredentialUpdateFailed.Message)
			return
		case appErrors.ErrInternalServer:
			utils.RespondError(c, appErrors.ErrInternalServer.Status, appErrors.ErrInternalServer.Code, appErrors.ErrInternalServer.Message)
			return
		default:
			utils.RespondError(c, http.StatusInternalServerError, "update_provider_credential_failed", err.Error())
			return
		}
	}

	logEntry.WithFields(logrus.Fields{
		"credential_id": result.ID,
		"provider":      result.Provider,
	}).Info("Successfully updated provider credential")

	utils.RespondSuccess(c, http.StatusOK, result)
}

// DeleteProviderCredential handles DELETE /orgs/:orgID/secret-groups/:groupID/environments/:envID/providers/credentials/:provider
func (h *ProviderHandler) DeleteProviderCredential(c *gin.Context) {
	environmentID := c.Param("envID")
	provider := c.Param("provider")

	logEntry := h.logger.WithFields(logrus.Fields{
		"handler":        "DeleteProviderCredential",
		"environment_id": environmentID,
		"provider":       provider,
		"method":         c.Request.Method,
		"path":           c.Request.URL.Path,
	})

	logEntry.Info("Processing delete provider credential request")

	err := h.service.DeleteProviderCredential(c.Request.Context(), environmentID, provider)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to delete provider credential")
		switch err {
		case appErrors.ErrProviderCredentialDeleteFailed:
			utils.RespondError(c, appErrors.ErrProviderCredentialDeleteFailed.Status, appErrors.ErrProviderCredentialDeleteFailed.Code, appErrors.ErrProviderCredentialDeleteFailed.Message)
			return
		case appErrors.ErrInternalServer:
			utils.RespondError(c, appErrors.ErrInternalServer.Status, appErrors.ErrInternalServer.Code, appErrors.ErrInternalServer.Message)
			return
		default:
			utils.RespondError(c, http.StatusInternalServerError, "delete_provider_credential_failed", err.Error())
			return
		}
	}

	logEntry.Info("Successfully deleted provider credential")

	utils.RespondSuccess(c, http.StatusOK, map[string]any{
		"message": fmt.Sprintf(" %s provider config deleted successfully", provider),
	})
}
