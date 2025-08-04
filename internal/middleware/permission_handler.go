package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/Gkemhcs/kavach-backend/internal/authz"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// PermissionHandler handles authorization for permission operations
type PermissionHandler struct {
	enforcer authz.Enforcer
	logger   *logrus.Logger
}

// NewPermissionHandler creates a new permission handler
func NewPermissionHandler(enforcer authz.Enforcer, logger *logrus.Logger) *PermissionHandler {
	return &PermissionHandler{
		enforcer: enforcer,
		logger:   logger,
	}
}

// HandleGrantPermission handles authorization for granting permissions
func (ph *PermissionHandler) HandleGrantPermission(c *gin.Context, userID string) error {
	logEntry := ph.logger.WithFields(logrus.Fields{
		"operation": "grant_permission",
		"user_id":   userID,
	})

	// Read the request body without consuming it
	body, err := c.GetRawData()
	if err != nil {
		logEntry.WithField("error", "failed_to_read_body").Error("Failed to read request body")
		return fmt.Errorf("failed to read request body: %v", err)
	}

	// Re-set the body so it can be read by subsequent handlers
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// Parse only the fields we need for authorization
	var req struct {
		ResourceType   string        `json:"resource_type"`
		ResourceID     uuid.UUID     `json:"resource_id"`
		OrganizationID uuid.UUID     `json:"organization_id"`
		SecretGroupID  uuid.NullUUID `json:"secret_group_id"`
		EnvironmentID  uuid.NullUUID `json:"environment_id"`
	}

	if err := json.Unmarshal(body, &req); err != nil {
		logEntry.WithFields(logrus.Fields{
			"error": "failed_to_decode_body",
			"body":  string(body),
		}).Error("Failed to decode request body")
		return fmt.Errorf("failed to decode request body: %v", err)
	}

	// Construct resource object based on resource type
	resource, err := ph.constructResourceFromRequest(GrantRoleBindingRequest{
		ResourceType:   req.ResourceType,
		ResourceID:     req.ResourceID,
		OrganizationID: req.OrganizationID,
		SecretGroupID:  req.SecretGroupID,
		EnvironmentID:  req.EnvironmentID,
	})
	if err != nil {
		logEntry.WithField("error", "failed_to_construct_resource").Error("Failed to construct resource")
		return fmt.Errorf("failed to construct resource: %v", err)
	}

	hasPermission, explanations, err := ph.enforcer.CheckPermissionEx(userID, "grant", resource)
	if err != nil {
		logEntry.WithFields(logrus.Fields{
			"error":      "permission_check_failed",
			"permission": "grant",
			"resource":   resource,
		}).Error("Failed to check permission")
		return fmt.Errorf("failed to check permission: %v", err)
	}

	if !hasPermission {
		logEntry.WithFields(logrus.Fields{
			"permission": "grant",
			"resource":   resource,
			"result":     "denied",
			"reason":     explanations,
		}).Warn("User does not have grant permission")
		return fmt.Errorf("user %s does not have grant permission on %s", userID, resource)
	}

	return nil
}

// HandleRevokePermission handles authorization for revoking permissions
func (ph *PermissionHandler) HandleRevokePermission(c *gin.Context, userID string) error {
	logEntry := ph.logger.WithFields(logrus.Fields{
		"operation": "revoke_permission",
		"user_id":   userID,
	})

	// Read the request body without consuming it
	body, err := c.GetRawData()
	if err != nil {
		logEntry.WithField("error", "failed_to_read_body").Error("Failed to read request body")
		return fmt.Errorf("failed to read request body: %v", err)
	}

	// Re-set the body so it can be read by subsequent handlers
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// Parse only the fields we need for authorization
	var req struct {
		ResourceType   string        `json:"resource_type"`
		ResourceID     uuid.UUID     `json:"resource_id"`
		OrganizationID uuid.UUID     `json:"organization_id"`
		SecretGroupID  uuid.NullUUID `json:"secret_group_id"`
		EnvironmentID  uuid.NullUUID `json:"environment_id"`
	}

	if err := json.Unmarshal(body, &req); err != nil {
		logEntry.WithField("error", "failed_to_decode_body").Error("Failed to decode request body")
		return fmt.Errorf("failed to decode request body: %v", err)
	}

	// Construct resource object based on resource type
	resource, err := ph.constructResourceFromRequest(GrantRoleBindingRequest{
		ResourceType:   req.ResourceType,
		ResourceID:     req.ResourceID,
		OrganizationID: req.OrganizationID,
		SecretGroupID:  req.SecretGroupID,
		EnvironmentID:  req.EnvironmentID,
	})
	if err != nil {
		logEntry.WithField("error", "failed_to_construct_resource").Error("Failed to construct resource")
		return fmt.Errorf("failed to construct resource: %v", err)
	}

	hasPermission, explanations, err := ph.enforcer.CheckPermissionEx(userID, "revoke", resource)
	if err != nil {
		logEntry.WithFields(logrus.Fields{
			"error":      "permission_check_failed",
			"permission": "revoke",
			"resource":   resource,
		}).Error("Failed to check permission")
		return fmt.Errorf("failed to check permission: %v", err)
	}

	if !hasPermission {
		logEntry.WithFields(logrus.Fields{
			"permission": "revoke",
			"resource":   resource,
			"result":     "denied",
			"reason":     explanations,
		}).Warn("User does not have revoke permission")
		return fmt.Errorf("user %s does not have revoke permission on %s", userID, resource)
	}

	return nil
}

// constructResourceFromRequest constructs resource path from request body
func (ph *PermissionHandler) constructResourceFromRequest(req GrantRoleBindingRequest) (string, error) {
	switch req.ResourceType {
	case "organization":
		if req.OrganizationID == uuid.Nil {
			return "", fmt.Errorf("organization_id is required for organization resource type")
		}
		return fmt.Sprintf("/organizations/%s", req.OrganizationID), nil

	case "secret_group":
		if req.OrganizationID == uuid.Nil || req.ResourceID == uuid.Nil {
			return "", fmt.Errorf("organization_id and resource_id are required for secret_group resource type")
		}
		return fmt.Sprintf("/organizations/%s/secret-groups/%s", req.OrganizationID, req.ResourceID), nil

	case "environment":
		if req.OrganizationID == uuid.Nil || !req.SecretGroupID.Valid || !req.EnvironmentID.Valid {
			return "", fmt.Errorf("organization_id, secret_group_id, and environment_id are required for environment resource type")
		}
		return fmt.Sprintf("/organizations/%s/secret-groups/%s/environments/%s",
			req.OrganizationID, req.SecretGroupID.UUID, req.EnvironmentID.UUID), nil

	default:
		return "", fmt.Errorf("unsupported resource type: %s", req.ResourceType)
	}
}
