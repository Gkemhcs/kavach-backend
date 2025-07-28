package groups

import (
	"net/http"

	apiErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	groupsdb "github.com/Gkemhcs/kavach-backend/internal/groups/gen"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// NewUserGroupHandler creates a new UserGroupHandler instance with the provided logger and service.
// This handler manages HTTP requests for user group operations including CRUD operations and member management.
func NewUserGroupHandler(logger *logrus.Logger, userGroupService *UserGroupService) *UserGroupHandler {
	return &UserGroupHandler{
		logger:           logger,
		userGroupService: userGroupService,
	}
}

// UserGroupHandler handles HTTP requests for user group operations.
// It provides endpoints for creating, deleting, listing user groups and managing group members.
type UserGroupHandler struct {
	logger           *logrus.Logger
	userGroupService *UserGroupService
}

// ToUserGroupResponseData converts a database UserGroup model to a response DTO.
// This function ensures consistent API response format and handles nullable fields properly.
func ToUserGroupResponseData(userGroup *groupsdb.UserGroup) UserGroupResponseData {
	return UserGroupResponseData{
		ID:             userGroup.ID.String(),
		Name:           userGroup.Name,
		OrganizationID: userGroup.OrganizationID.String(),
		Description:    userGroup.Description.String,
		CreatedAt:      userGroup.CreatedAt,
		UpdatedAt:      userGroup.UpdatedAt,
	}
}

// ToListGroupsByOrgRow converts a database ListGroupsByOrgRow to a response DTO.
// Used for listing user groups within an organization with minimal required fields.
func ToListGroupsByOrgRow(userGroup groupsdb.ListGroupsByOrgRow) ListGroupsByOrgRow {
	return ListGroupsByOrgRow{
		ID:          userGroup.ID.String(),
		Name:        userGroup.Name,
		Description: userGroup.Description.String,
		CreatedAt:   userGroup.CreatedAt,
	}
}

// ToListGroupMembersRow converts a database ListGroupMembersRow to a response DTO.
// Used for listing group members with user details like name, email, and membership date.
func ToListGroupMembersRow(member groupsdb.ListGroupMembersRow) ListGroupMembersRow {
	return ListGroupMembersRow{
		ID:        member.ID.String(),
		Name:      member.Name.String,
		Email:     member.Email.String,
		CreatedAt: member.CreatedAt,
	}
}

// RegisterUserGroupRoutes registers all user group related HTTP routes with the provided router group.
// All routes are protected by JWT middleware and scoped to a specific organization.
// Route structure: /:orgID/user-groups/*
func RegisterUserGroupRoutes(handler *UserGroupHandler, orgRouterGroup *gin.RouterGroup, jwtMiddleware gin.HandlerFunc) {
	userGroupRouterGroup := orgRouterGroup.Group(":orgID/user-groups")
	userGroupRouterGroup.Use(jwtMiddleware)

	// User group management endpoints
	userGroupRouterGroup.GET("/by-name", handler.GetUserGroupByName)
	userGroupRouterGroup.POST("/", handler.CreateUserGroup)
	userGroupRouterGroup.DELETE("/:userGroupID", handler.DeleteUserGroup)
	userGroupRouterGroup.GET("/", handler.ListUserGroups)

	// Group member management endpoints
	userGroupRouterGroup.POST("/:userGroupID/members", handler.AddGroupMember)
	userGroupRouterGroup.DELETE("/:userGroupID/members", handler.RemoveGroupMember)
	userGroupRouterGroup.GET("/:userGroupID/members", handler.ListGroupMembers)
}

// CreateUserGroup handles POST requests to create a new user group within an organization.
// Validates the request payload, ensures unique group names within the organization,
// and returns the created group details or appropriate error responses.
func (h *UserGroupHandler) CreateUserGroup(c *gin.Context) {
	userId := c.GetString("user_id")
	orgID := c.Param("orgID")
	var req CreateUserGroupRequest

	// Validate and parse request body
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(logrus.Fields{
			"orgID": orgID,
			"error": err.Error(),
		}).Warn("Invalid request payload for creating user group")
		utils.RespondError(c, http.StatusBadRequest, "bad_request", "invalid request payload")
		return
	}

	req.OrganizationID = orgID
	req.UserID=userId
	

	h.logger.WithFields(logrus.Fields{
		"orgID":          orgID,
		"groupName":      req.GroupName,
		"hasDescription": req.Description != "",
	}).Info("Creating user group")

	userGroup, err := h.userGroupService.CreateUserGroup(c.Request.Context(), req)
	if err != nil {
		if err == apiErrors.ErrDuplicateUserGroup {
			h.logger.WithFields(logrus.Fields{
				"orgID":     orgID,
				"groupName": req.GroupName,
			}).Warn("User group creation failed: group name already exists")
			apiErr, _ := err.(*apiErrors.APIError)
			utils.RespondError(c, http.StatusBadRequest, apiErr.Code, apiErr.Message)
			return
		}

		h.logger.WithFields(logrus.Fields{
			"orgID":     orgID,
			"groupName": req.GroupName,
			"error":     err.Error(),
		}).Error("Failed to create user group due to internal error")
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "failed to create user group")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"orgID":     orgID,
		"groupID":   userGroup.ID.String(),
		"groupName": userGroup.Name,
	}).Info("User group created successfully")

	utils.RespondSuccess(c, http.StatusCreated, ToUserGroupResponseData(userGroup))
}

// DeleteUserGroup handles DELETE requests to remove a user group from an organization.
// Validates that the group exists and belongs to the specified organization before deletion.
func (h *UserGroupHandler) DeleteUserGroup(c *gin.Context) {
	orgID := c.Param("orgID")
	userGroupID := c.Param("userGroupID")

	h.logger.WithFields(logrus.Fields{
		"orgID":       orgID,
		"userGroupID": userGroupID,
	}).Info("Deleting user group")

	err := h.userGroupService.DeleteUserGroup(c.Request.Context(), orgID, userGroupID)
	if err != nil {
		if err == apiErrors.ErrUserGroupNotFound {
			h.logger.WithFields(logrus.Fields{
				"orgID":       orgID,
				"userGroupID": userGroupID,
			}).Warn("User group deletion failed: group not found")
			apiErr, _ := err.(*apiErrors.APIError)
			utils.RespondError(c, http.StatusBadRequest, apiErr.Code, apiErr.Message)
			return
		}

		h.logger.WithFields(logrus.Fields{
			"orgID":       orgID,
			"userGroupID": userGroupID,
			"error":       err.Error(),
		}).Error("Failed to delete user group due to internal error")
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "failed to delete user group")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"orgID":       orgID,
		"userGroupID": userGroupID,
	}).Info("User group deleted successfully")

	utils.RespondSuccess(c, http.StatusOK, map[string]string{
		"message": "user group deleted successfully",
	})
}

// ListUserGroups handles GET requests to retrieve all user groups within an organization.
// Returns a paginated list of groups with basic information for display purposes.
func (h *UserGroupHandler) ListUserGroups(c *gin.Context) {
	orgID := c.Param("orgID")

	h.logger.WithFields(logrus.Fields{
		"orgID": orgID,
	}).Info("Listing user groups for organization")

	userGroups, err := h.userGroupService.ListUserGroups(c.Request.Context(), orgID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"orgID": orgID,
			"error": err.Error(),
		}).Error("Failed to list user groups due to internal error")
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "failed to list user groups")
		return
	}

	var userGroupsList []ListGroupsByOrgRow
	for _, userGroup := range userGroups {
		userGroupsList = append(userGroupsList, ToListGroupsByOrgRow(userGroup))
	}

	h.logger.WithFields(logrus.Fields{
		"orgID": orgID,
		"count": len(userGroupsList),
	}).Info("User groups listed successfully")

	utils.RespondSuccess(c, http.StatusOK, userGroupsList)
}

// GetUserGroupByName handles GET requests to retrieve a specific user group by name within an organization.
// Used for checking group existence and retrieving group details for membership operations.
func (h *UserGroupHandler) GetUserGroupByName(c *gin.Context) {
	orgID := c.Param("orgID")
	userGroupName := c.Query("name")

	if userGroupName == "" {
		h.logger.WithFields(logrus.Fields{
			"orgID": orgID,
		}).Warn("Get user group by name failed: missing group name parameter")
		utils.RespondError(c, http.StatusBadRequest, "bad_request", "group name is required")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"orgID":     orgID,
		"groupName": userGroupName,
	}).Info("Fetching user group by name")

	userGroup, err := h.userGroupService.GetUserGroupByName(c.Request.Context(), userGroupName, orgID)
	if err != nil {
		if err == apiErrors.ErrUserGroupNotFound {
			h.logger.WithFields(logrus.Fields{
				"orgID":     orgID,
				"groupName": userGroupName,
			}).Warn("User group not found by name")
			apiErr, _ := err.(*apiErrors.APIError)
			utils.RespondError(c, http.StatusBadRequest, apiErr.Code, apiErr.Message)
			return
		}

		h.logger.WithFields(logrus.Fields{
			"orgID":     orgID,
			"groupName": userGroupName,
			"error":     err.Error(),
		}).Error("Failed to fetch user group by name due to internal error")
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "failed to fetch user group")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"orgID":     orgID,
		"groupID":   userGroup.ID.String(),
		"groupName": userGroup.Name,
	}).Info("User group fetched successfully")

	utils.RespondSuccess(c, http.StatusOK, ToUserGroupResponseData(userGroup))
}

// AddGroupMember handles POST requests to add a user to a user group.
// Validates that both the user and group exist before creating the membership relationship.
func (h *UserGroupHandler) AddGroupMember(c *gin.Context) {
	userGroupID := c.Param("userGroupID")
	var req AddMemberRequest

	// Validate and parse request body
	if err := c.ShouldBindBodyWithJSON(&req); err != nil {
		h.logger.WithFields(logrus.Fields{
			"userGroupID": userGroupID,
			"error":       err.Error(),
		}).Warn("Invalid request payload for adding group member")
		utils.RespondError(c, http.StatusBadRequest, "bad_request", "invalid request payload")
		return
	}

	if req.UserName == "" {
		h.logger.WithFields(logrus.Fields{
			"userGroupID": userGroupID,
		}).Warn("Add group member failed: missing username")
		utils.RespondError(c, http.StatusBadRequest, "bad_request", "username is required")
		return
	}

	req.UserGroupID = userGroupID

	h.logger.WithFields(logrus.Fields{
		"userGroupID": userGroupID,
		"userName":    req.UserName,
	}).Info("Adding member to user group")

	err := h.userGroupService.AddGroupMember(c.Request.Context(), req)
	if err != nil {
		if err == apiErrors.ErrUserNotFound {
			h.logger.WithFields(logrus.Fields{
				"userGroupID": userGroupID,
				"userName":    req.UserName,
			}).Warn("Add group member failed: user not found")
			apiErr, _ := err.(*apiErrors.APIError)
			utils.RespondError(c, http.StatusBadRequest, apiErr.Code, apiErr.Message)
			return
		}
		if err == apiErrors.ErrDuplicateMemberOfUserGroup {
			h.logger.WithFields(logrus.Fields{
				"userGroupID": userGroupID,
				"userName":    req.UserName,
			}).Warn("Add group member failed: user already a member")
			apiErr, _ := err.(*apiErrors.APIError)
			utils.RespondError(c, http.StatusBadRequest, apiErr.Code, apiErr.Message)
			return
		}

		h.logger.WithFields(logrus.Fields{
			"userGroupID": userGroupID,
			"userName":    req.UserName,
			"error":       err.Error(),
		}).Error("Failed to add member to user group due to internal error")
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "failed to add member to group")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"userGroupID": userGroupID,
		"userName":    req.UserName,
	}).Info("Member added to user group successfully")

	utils.RespondSuccess(c, http.StatusCreated, map[string]string{
		"message": "member added successfully",
	})
}

// RemoveGroupMember handles DELETE requests to remove a user from a user group.
// Validates that the user is currently a member of the group before removal.
func (h *UserGroupHandler) RemoveGroupMember(c *gin.Context) {
	userGroupID := c.Param("userGroupID")
	var req RemoveMemberRequest

	if err := c.ShouldBindBodyWithJSON(&req); err != nil {
		h.logger.WithFields(logrus.Fields{
			"userGroupID": userGroupID,
			"error":       err.Error(),
		}).Warn("Invalid request payload for removing group member")
		utils.RespondError(c, http.StatusBadRequest, "bad_request", "invalid request payload")
		return
	}

	if req.UserName == "" {
		h.logger.WithFields(logrus.Fields{
			"userGroupID": userGroupID,
		}).Warn("Remove group member failed: missing username")
		utils.RespondError(c, http.StatusBadRequest, "bad_request", "username is required")
		return
	}

	req.UserGroupID = userGroupID

	h.logger.WithFields(logrus.Fields{
		"userGroupID": userGroupID,
		"userName":    req.UserName,
	}).Info("Removing member from user group")

	err := h.userGroupService.RemoveGroupMember(c.Request.Context(), req)
	if err != nil {
		if err == apiErrors.ErrUserNotFound {
			h.logger.WithFields(logrus.Fields{
				"userGroupID": userGroupID,
				"userName":    req.UserName,
			}).Warn("Remove group member failed: user not found")
			apiErr, _ := err.(*apiErrors.APIError)
			utils.RespondError(c, http.StatusBadRequest, apiErr.Code, apiErr.Message)
			return
		}
		if err == apiErrors.ErrUserMembershipNotFound {
			h.logger.WithFields(logrus.Fields{
				"userGroupID": userGroupID,
				"userName":    req.UserName,
			}).Warn("Remove group member failed: user not a member of group")
			apiErr, _ := err.(*apiErrors.APIError)
			utils.RespondError(c, http.StatusBadRequest, apiErr.Code, apiErr.Message)
			return
		}

		h.logger.WithFields(logrus.Fields{
			"userGroupID": userGroupID,
			"userName":    req.UserName,
			"error":       err.Error(),
		}).Error("Failed to remove member from user group due to internal error")
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "failed to remove member from group")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"userGroupID": userGroupID,
		"userName":    req.UserName,
	}).Info("Member removed from user group successfully")

	utils.RespondSuccess(c, http.StatusOK, map[string]string{
		"message": "member removed successfully",
	})
}

// ListGroupMembers handles GET requests to retrieve all members of a specific user group.
// Returns user details including name, email, and when they joined the group.
func (h *UserGroupHandler) ListGroupMembers(c *gin.Context) {
	userGroupID := c.Param("userGroupID")

	h.logger.WithFields(logrus.Fields{
		"userGroupID": userGroupID,
	}).Info("Listing members of user group")

	members, err := h.userGroupService.ListGroupMembers(c.Request.Context(), userGroupID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"userGroupID": userGroupID,
			"error":       err.Error(),
		}).Error("Failed to list group members due to internal error")
		utils.RespondError(c, http.StatusInternalServerError, "internal_error", "failed to list group members")
		return
	}

	var memberList []ListGroupMembersRow
	for _, member := range members {
		memberList = append(memberList, ToListGroupMembersRow(member))
	}

	h.logger.WithFields(logrus.Fields{
		"userGroupID": userGroupID,
		"count":       len(memberList),
	}).Info("Group members listed successfully")

	utils.RespondSuccess(c, http.StatusOK, memberList)
}
