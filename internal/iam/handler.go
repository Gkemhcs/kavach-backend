package iam

import (
	"net/http"

	apiErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// NewIamHandler creates a new IamHandler instance with the provided service and logger.
// This handler manages HTTP requests for IAM operations including role granting and revocation.
func NewIamHandler(iamService IamService, logger *logrus.Logger) *IamHandler {
	return &IamHandler{
		iamService,
		logger,
	}
}

// IamHandler handles HTTP requests for IAM (Identity and Access Management) operations.
// It provides endpoints for granting and revoking role bindings for users and user groups.
type IamHandler struct {
	iamService IamService
	logger     *logrus.Logger
}

// RegisterIamRoutes registers all IAM related HTTP routes with the provided router group.
// All routes handle permission management operations for role-based access control.
func RegisterIamRoutes(handler *IamHandler, routerGroup *gin.RouterGroup) {
	iamGroup := routerGroup.Group("/permissions")
	iamGroup.POST("/grant", handler.GrantRoleBinding)
	iamGroup.DELETE("/revoke", handler.RevokeRoleBinding)
}

// GrantRoleBinding handles POST requests to grant a role to a user or user group.
// Validates that only one of username or group name is provided, then delegates to the service layer.
// Supports granting roles on organizations, secret groups, and environments.
func (h *IamHandler) GrantRoleBinding(c *gin.Context) {
	var req GrantRoleBindingRequest

	// Validate and parse request body
	if err := c.ShouldBindBodyWithJSON(&req); err != nil {
		h.logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Warn("Invalid request payload for granting role binding")
		utils.RespondError(c, http.StatusUnprocessableEntity, "invalid_request_body", "unable to parse request body")
		return
	}

	// Validate that only one of username or group name is provided
	if req.UserName != "" && req.GroupName != "" {
		h.logger.WithFields(logrus.Fields{
			"userName":  req.UserName,
			"groupName": req.GroupName,
		}).Warn("Grant role binding failed: both username and group name provided")
		utils.RespondError(c, http.StatusConflict, "only_one_of_users_and_user_groups", "only one of username and usergroup are allowed")
		return
	}

	// Validate that at least one of username or group name is provided
	if req.GroupName == "" && req.UserName == "" {
		h.logger.WithFields(logrus.Fields{
			"resourceType": req.ResourceType,
			"resourceID":   req.ResourceID.String(),
			"role":         req.Role,
		}).Warn("Grant role binding failed: neither username nor group name provided")
		utils.RespondError(c, http.StatusBadRequest, "none_of_user_and_usr_group_passed", "none of user and user groups are passed")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"userName":     req.UserName,
		"groupName":    req.GroupName,
		"role":         req.Role,
		"resourceType": req.ResourceType,
		"resourceID":   req.ResourceID.String(),
		"orgID":        req.OrganizationID.String(),
	}).Info("Granting role binding")

	err := h.iamService.GrantRoleBinding(c.Request.Context(), req)
	if err != nil {
		if err == apiErrors.ErrUserNotFound || err == apiErrors.ErrUserGroupNotFound || err == apiErrors.ErrDuplicateRoleBinding {
			h.logger.WithFields(logrus.Fields{
				"userName":     req.UserName,
				"groupName":    req.GroupName,
				"role":         req.Role,
				"resourceType": req.ResourceType,
				"resourceID":   req.ResourceID.String(),
				"error":        err.Error(),
			}).Warn("Grant role binding failed: validation error")
			apiErr := err.(*apiErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		}

		h.logger.WithFields(logrus.Fields{
			"userName":     req.UserName,
			"groupName":    req.GroupName,
			"role":         req.Role,
			"resourceType": req.ResourceType,
			"resourceID":   req.ResourceID.String(),
			"error":        err.Error(),
		}).Error("Failed to grant role binding due to internal error")
		utils.RespondError(c, http.StatusBadRequest, "internal_server_error", "sorry internal server happened try out after sometime")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"userName":     req.UserName,
		"groupName":    req.GroupName,
		"role":         req.Role,
		"resourceType": req.ResourceType,
		"resourceID":   req.ResourceID.String(),
	}).Info("Role binding granted successfully")

	utils.RespondSuccess(c, http.StatusOK, map[string]string{
		"message": "role granted successfully",
	})
}

// RevokeRoleBinding handles DELETE requests to revoke a role from a user or user group.
// Validates that only one of username or group name is provided, then delegates to the service layer.
// Supports revoking roles on organizations, secret groups, and environments.
func (h *IamHandler) RevokeRoleBinding(c *gin.Context) {
	var req RevokeRoleBindingRequest

	// Validate and parse request body
	if err := c.ShouldBindBodyWithJSON(&req); err != nil {
		h.logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Warn("Invalid request payload for revoking role binding")
		utils.RespondError(c, http.StatusUnprocessableEntity, "invalid_request_body", "unable to parse request body")
		return
	}

	// Validate that only one of username or group name is provided
	if req.UserName != "" && req.GroupName != "" {
		h.logger.WithFields(logrus.Fields{
			"userName":  req.UserName,
			"groupName": req.GroupName,
		}).Warn("Revoke role binding failed: both username and group name provided")
		utils.RespondError(c, http.StatusConflict, "only_one_of_users_and_user_groups", "only one of username and usergroup are allowed")
		return
	}

	// Validate that at least one of username or group name is provided
	if req.GroupName == "" && req.UserName == "" {
		h.logger.WithFields(logrus.Fields{
			"resourceType": req.ResourceType,
			"resourceID":   req.ResourceID.String(),
			"role":         req.Role,
		}).Warn("Revoke role binding failed: neither username nor group name provided")
		utils.RespondError(c, http.StatusBadRequest, "none_of_user_and_usr_group_passed", "none of user and user groups are passed")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"userName":     req.UserName,
		"groupName":    req.GroupName,
		"role":         req.Role,
		"resourceType": req.ResourceType,
		"resourceID":   req.ResourceID.String(),
		"orgID":        req.OrganizationID.String(),
	}).Info("Revoking role binding")

	err := h.iamService.RevokeRoleBinding(c.Request.Context(), req)
	if err != nil {
		if err == apiErrors.ErrUserNotFound || err == apiErrors.ErrUserGroupNotFound || err == apiErrors.ErrRoleBindingNotFound {
			h.logger.WithFields(logrus.Fields{
				"userName":     req.UserName,
				"groupName":    req.GroupName,
				"role":         req.Role,
				"resourceType": req.ResourceType,
				"resourceID":   req.ResourceID.String(),
				"error":        err.Error(),
			}).Warn("Revoke role binding failed: validation error")
			apiErr := err.(*apiErrors.APIError)
			utils.RespondError(c, apiErr.Status, apiErr.Code, apiErr.Message)
			return
		}

		h.logger.WithFields(logrus.Fields{
			"userName":     req.UserName,
			"groupName":    req.GroupName,
			"role":         req.Role,
			"resourceType": req.ResourceType,
			"resourceID":   req.ResourceID.String(),
			"error":        err.Error(),
		}).Error("Failed to revoke role binding due to internal error")
		utils.RespondError(c, http.StatusBadRequest, "internal_server_error", "sorry internal server happened try out after sometime")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"userName":     req.UserName,
		"groupName":    req.GroupName,
		"role":         req.Role,
		"resourceType": req.ResourceType,
		"resourceID":   req.ResourceID.String(),
	}).Info("Role binding revoked successfully")

	utils.RespondSuccess(c, http.StatusOK, map[string]string{
		"message": "role revoked successfully",
	})
}
