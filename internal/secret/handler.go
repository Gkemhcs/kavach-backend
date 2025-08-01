package secret

import (
	"net/http"

	appErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SecretHandler handles HTTP requests for secret management
type SecretHandler struct {
	service *SecretService
	logger  *logrus.Logger
}

// NewSecretHandler creates a new SecretHandler
func NewSecretHandler(service *SecretService, logger *logrus.Logger) *SecretHandler {
	return &SecretHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterSecretRoutes registers secret routes under an environment
func RegisterSecretRoutes(handler *SecretHandler, envGroup *gin.RouterGroup) {
	secretsGroup := envGroup.Group("/:envID/secrets")
	{
		secretsGroup.POST("/", handler.CreateVersion)
		secretsGroup.GET("/versions", handler.ListVersions)
		secretsGroup.GET("/versions/:versionID", handler.GetVersionDetails)
		secretsGroup.POST("/rollback", handler.RollbackToVersion)
		secretsGroup.GET("/diff", handler.GetVersionDiff)
		secretsGroup.POST("/sync", handler.SyncSecrets)
	}
}

// CreateVersion handles POST /orgs/:orgID/secret-groups/:groupID/environments/:envID/secrets
func (h *SecretHandler) CreateVersion(c *gin.Context) {
	environmentID := c.Param("envID")

	logEntry := h.logger.WithFields(logrus.Fields{
		"handler":        "CreateVersion",
		"environment_id": environmentID,
		"method":         c.Request.Method,
		"path":           c.Request.URL.Path,
	})

	logEntry.Info("Processing create secret version request")

	var req CreateSecretVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to bind request body")
		utils.RespondError(c, appErrors.ErrInvalidBody.Status, appErrors.ErrInvalidBody.Code, appErrors.ErrInvalidBody.Message)
		return
	}

	logEntry.WithFields(logrus.Fields{
		"secret_count":   len(req.Secrets),
		"commit_message": req.CommitMessage,
	}).Info("Request validated successfully")

	result, err := h.service.CreateVersion(c.Request.Context(), environmentID, req)
	if err != nil {

		switch err {
		case appErrors.ErrEmptySecrets:
			apiErr := err.(*appErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		case appErrors.ErrTooManySecrets:
			apiErr := err.(*appErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		case appErrors.ErrInvalidSecretName:
			apiErr := err.(*appErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		case appErrors.ErrSecretValueTooLong:
			apiErr := err.(*appErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		case appErrors.ErrEncryptionFailed:
			apiErr := err.(*appErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		default:
			logEntry.WithField("error", err.Error()).Error("Failed to create secret version")
			utils.RespondError(c, http.StatusInternalServerError, "create_version_failed", err.Error())
			return

		}

	}

	logEntry.WithFields(logrus.Fields{
		"version_id":   result.ID,
		"secret_count": result.SecretCount,
	}).Info("Successfully created secret version")

	utils.RespondSuccess(c, http.StatusCreated, result)
}

// ListVersions handles GET /orgs/:orgID/secret-groups/:groupID/environments/:envID/secrets/versions
func (h *SecretHandler) ListVersions(c *gin.Context) {
	environmentID := c.Param("envID")

	logEntry := h.logger.WithFields(logrus.Fields{
		"handler":        "ListVersions",
		"environment_id": environmentID,
		"method":         c.Request.Method,
		"path":           c.Request.URL.Path,
	})

	logEntry.Info("Processing list secret versions request")

	versions, err := h.service.ListVersions(c.Request.Context(), environmentID)
	if err != nil {
		switch err {
		case appErrors.ErrEnvironmentNotFound:
			apiErr := err.(*appErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
		default:
			logEntry.WithField("error", err.Error()).Error("Failed to list secret versions")
			utils.RespondError(c, http.StatusInternalServerError, "list_versions_failed", err.Error())
			return
		}
	}

	logEntry.WithField("version_count", len(versions)).Info("Successfully listed secret versions")

	utils.RespondSuccess(c, http.StatusOK, versions)
}

// GetVersionDetails handles GET /orgs/:orgID/secret-groups/:groupID/environments/:envID/secrets/versions/:versionID
func (h *SecretHandler) GetVersionDetails(c *gin.Context) {
	versionID := c.Param("versionID")

	logEntry := h.logger.WithFields(logrus.Fields{
		"handler":    "GetVersionDetails",
		"version_id": versionID,
		"method":     c.Request.Method,
		"path":       c.Request.URL.Path,
	})

	logEntry.Info("Processing get version details request")

	version, err := h.service.GetVersionDetails(c.Request.Context(), versionID)
	if err != nil {
		switch err {
		case appErrors.ErrSecretVersionNotFound:
			apiErr := err.(*appErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		case appErrors.ErrSecretNotFound:
			apiErr := err.(*appErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		case appErrors.ErrDecryptionFailed:
			apiErr := err.(*appErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		case appErrors.ErrInternalServer:
			apiErr := err.(*appErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		default:
			logEntry.WithField("error", err.Error()).Error("Failed to get version details")
			utils.RespondError(c, http.StatusInternalServerError, "get_version_details_failed", err.Error())
			return
		}
	}

	logEntry.WithFields(logrus.Fields{
		"version_id":   versionID,
		"secret_count": len(version.Secrets),
	}).Info("Successfully retrieved version details")

	utils.RespondSuccess(c, http.StatusOK, version)
}

// RollbackToVersion handles POST /orgs/:orgID/secret-groups/:groupID/environments/:envID/secrets/rollback
func (h *SecretHandler) RollbackToVersion(c *gin.Context) {
	environmentID := c.Param("envID")

	logEntry := h.logger.WithFields(logrus.Fields{
		"handler":        "RollbackToVersion",
		"environment_id": environmentID,
		"method":         c.Request.Method,
		"path":           c.Request.URL.Path,
	})

	logEntry.Info("Processing rollback request")

	var req RollbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to bind rollback request body")
		utils.RespondError(c, appErrors.ErrInvalidBody.Status, appErrors.ErrInvalidBody.Code, appErrors.ErrInvalidBody.Message)
		return
	}

	logEntry.WithFields(logrus.Fields{
		"target_version": req.VersionID,
		"commit_message": req.CommitMessage,
	}).Info("Rollback request validated successfully")

	result, err := h.service.RollbackToVersion(c.Request.Context(), environmentID, req)
	if err != nil {
		switch err {
		case appErrors.ErrEnvironmentNotFound:
			apiErr := err.(*appErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		case appErrors.ErrTargetSecretVersionNotFound:
			apiErr := err.(*appErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		case appErrors.ErrEnvironmentsMisMatch:
			apiErr := err.(*appErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		case appErrors.ErrRollbackFailed:
			apiErr := err.(*appErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		case appErrors.ErrCopySecretsFailed:
			apiErr := err.(*appErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		case appErrors.ErrEncryptionFailed:
			apiErr := err.(*appErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		case appErrors.ErrInternalServer:
			apiErr := err.(*appErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		default:
			logEntry.WithField("error", err.Error()).Error("Failed to rollback to version")
			utils.RespondError(c, http.StatusInternalServerError, "rollback_failed", err.Error())
			return
		}
	}

	logEntry.WithFields(logrus.Fields{
		"new_version_id": result.ID,
		"target_version": req.VersionID,
		"secret_count":   result.SecretCount,
	}).Info("Successfully rolled back to version")

	utils.RespondSuccess(c, http.StatusCreated, result)
}

// GetVersionDiff handles GET /orgs/:orgID/secret-groups/:groupID/environments/:envID/secrets/diff?from=x&to=y
func (h *SecretHandler) GetVersionDiff(c *gin.Context) {
	fromVersion := c.Query("from")
	toVersion := c.Query("to")

	logEntry := h.logger.WithFields(logrus.Fields{
		"handler":      "GetVersionDiff",
		"from_version": fromVersion,
		"to_version":   toVersion,
		"method":       c.Request.Method,
		"path":         c.Request.URL.Path,
	})

	logEntry.Info("Processing version diff request")

	// Validate query parameters
	if fromVersion == "" || toVersion == "" {
		logEntry.Error("Missing required query parameters")
		utils.RespondError(c, http.StatusBadRequest, "missing_parameters", "Both 'from' and 'to' query parameters are required")
		return
	}

	diff, err := h.service.GetVersionDiff(c.Request.Context(), fromVersion, toVersion)
	if err != nil {
		switch err {
		case appErrors.ErrSecretVersionNotFound:
			apiErr := err.(*appErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		case appErrors.ErrDecryptionFailed:
			apiErr := err.(*appErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		case appErrors.ErrInternalServer:
			apiErr := err.(*appErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		default:
			logEntry.WithField("error", err.Error()).Error("Failed to get version diff")
			utils.RespondError(c, http.StatusInternalServerError, "get_diff_failed", err.Error())
			return
		}
	}

	logEntry.WithField("change_count", len(diff.Changes)).Info("Successfully generated version diff")

	utils.RespondSuccess(c, http.StatusOK, diff)
}

// SyncSecrets handles POST /orgs/:orgID/secret-groups/:groupID/environments/:envID/secrets/sync
func (h *SecretHandler) SyncSecrets(c *gin.Context) {
	environmentID := c.Param("envID")

	logEntry := h.logger.WithFields(logrus.Fields{
		"handler":        "SyncSecrets",
		"environment_id": environmentID,
		"method":         c.Request.Method,
		"path":           c.Request.URL.Path,
	})

	logEntry.Info("Processing sync secrets request")

	var req SyncSecretsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to bind request body")
		utils.RespondError(c, appErrors.ErrInvalidBody.Status, appErrors.ErrInvalidBody.Code, appErrors.ErrInvalidBody.Message)
		return
	}

	logEntry.WithFields(logrus.Fields{
		"provider":   req.Provider,
		"version_id": req.VersionID,
	}).Info("Request validated successfully")

	result, err := h.service.SyncSecrets(c.Request.Context(), environmentID, req)
	if err != nil {
		logEntry.WithField("error", err.Error()).Error("Failed to sync secrets")
		switch err {
		case appErrors.ErrNoSecretsToSync:
			utils.RespondError(c, appErrors.ErrNoSecretsToSync.Status, appErrors.ErrNoSecretsToSync.Code, appErrors.ErrNoSecretsToSync.Message)
			return
		case appErrors.ErrProviderSyncFailed:
			utils.RespondError(c, appErrors.ErrProviderSyncFailed.Status, appErrors.ErrProviderSyncFailed.Code, appErrors.ErrProviderSyncFailed.Message)
			return
		case appErrors.ErrProviderCredentialNotFound:
			utils.RespondError(c, appErrors.ErrProviderCredentialNotFound.Status, appErrors.ErrProviderCredentialNotFound.Code, appErrors.ErrProviderCredentialNotFound.Message)
			return
		case appErrors.ErrProviderCredentialValidationFailed:
			utils.RespondError(c, appErrors.ErrProviderCredentialValidationFailed.Status, appErrors.ErrProviderCredentialValidationFailed.Code, appErrors.ErrProviderCredentialValidationFailed.Message)
			return
		case appErrors.ErrGitHubEnvironmentNotFound:
			utils.RespondError(c, appErrors.ErrGitHubEnvironmentNotFound.Status, appErrors.ErrGitHubEnvironmentNotFound.Code, appErrors.ErrGitHubEnvironmentNotFound.Message)
			return
		case appErrors.ErrGitHubEncryptionFailed:
			utils.RespondError(c, appErrors.ErrGitHubEncryptionFailed.Status, appErrors.ErrGitHubEncryptionFailed.Code, appErrors.ErrGitHubEncryptionFailed.Message)
			return
		case appErrors.ErrInvalidProviderType:
			utils.RespondError(c, appErrors.ErrInvalidProviderType.Status, appErrors.ErrInvalidProviderType.Code, appErrors.ErrInvalidProviderType.Message)
			return
		case appErrors.ErrInternalServer:
			utils.RespondError(c, appErrors.ErrInternalServer.Status, appErrors.ErrInternalServer.Code, appErrors.ErrInternalServer.Message)
			return
		default:
			utils.RespondError(c, http.StatusInternalServerError, "sync_secrets_failed", err.Error())
			return
		}
	}

	logEntry.WithFields(logrus.Fields{
		"provider":     result.Provider,
		"status":       result.Status,
		"synced_count": result.SyncedCount,
		"failed_count": result.FailedCount,
		"total_count":  result.TotalCount,
	}).Info("Successfully synced secrets to provider")

	utils.RespondSuccess(c, http.StatusOK, result)
}
