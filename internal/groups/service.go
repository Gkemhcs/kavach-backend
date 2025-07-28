package groups

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Gkemhcs/kavach-backend/internal/auth"
	"github.com/Gkemhcs/kavach-backend/internal/authz"
	apiErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	groupsdb "github.com/Gkemhcs/kavach-backend/internal/groups/gen"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// NewUserGroupService creates a new UserGroupService instance with the provided dependencies.
// This service handles business logic for user group operations including CRUD operations and member management.
func NewUserGroupService(logger *logrus.Logger, usergroupRepo groupsdb.Querier, userService auth.UserInfoGetter, policyEnforcer *authz.Enforcer) *UserGroupService {
	return &UserGroupService{
		logger:         logger,
		userGroupRepo:  usergroupRepo,
		userService:    userService,
		policyEnforcer: policyEnforcer,
	}
}

// UserGroupService provides business logic for user group operations.
// It coordinates between the repository layer and external services like user management.
type UserGroupService struct {
	logger         *logrus.Logger
	userGroupRepo  groupsdb.Querier
	userService    auth.UserInfoGetter
	policyEnforcer *authz.Enforcer
}

// CreateUserGroup creates a new user group within an organization.
// Validates input parameters, ensures unique group names within the organization,
// and handles database constraints like unique violations.
func (s *UserGroupService) CreateUserGroup(ctx context.Context, req CreateUserGroupRequest) (*groupsdb.UserGroup, error) {
	orgID := req.OrganizationID

	s.logger.WithFields(logrus.Fields{
		"orgID":          orgID,
		"groupName":      req.GroupName,
		"hasDescription": req.Description != "",
	}).Info("Creating user group in database")

	// Prepare database parameters with proper null handling for description
	params := groupsdb.CreateGroupParams{
		OrganizationID: uuid.MustParse(req.OrganizationID),
		Name:           req.GroupName,
		Description: sql.NullString{
			String: req.Description,
			Valid:  req.Description != "",
		},
	}

	group, err := s.userGroupRepo.CreateGroup(ctx, params)
	if err != nil {
		if apiErrors.IsUniqueViolation(err) {
			s.logger.WithFields(logrus.Fields{
				"orgID":     orgID,
				"groupName": req.GroupName,
			}).Warn("User group creation failed: unique constraint violation")
			return nil, apiErrors.ErrDuplicateUserGroup
		}

		s.logger.WithFields(logrus.Fields{
			"orgID":     orgID,
			"groupName": req.GroupName,
			"error":     err.Error(),
		}).Error("Failed to create user group in database")
		return nil, err
	}

	s.logger.WithFields(logrus.Fields{
		"orgID":     orgID,
		"groupID":   group.ID.String(),
		"groupName": group.Name,
	}).Info("User group created successfully in database")

	parentResourcePath := fmt.Sprintf("/organizations/%s", orgID)
	childResourcePath := fmt.Sprintf("/organizations/%s/user-groups/%s", orgID, group.ID)
	err = s.policyEnforcer.AddResourceOwner(req.UserID, childResourcePath)
	if err != nil {
		s.logger.Errorf("failed to add secure permissions for usergroup %s", req.GroupName)
		return nil, err
	}
	err = s.policyEnforcer.AddResourceHierarchy(parentResourcePath, childResourcePath)
	if err != nil {
		s.logger.Errorf("unable to add %s to resoure heirarchy of %s ", childResourcePath, parentResourcePath)
		return nil, err
	}
	return &group, nil
}

// GetUserGroupByName retrieves a user group by its name within a specific organization.
// Used for validation and lookup operations when group name is known but ID is not.
func (s *UserGroupService) GetUserGroupByName(ctx context.Context, userGroupName, orgID string) (*groupsdb.UserGroup, error) {
	s.logger.WithFields(logrus.Fields{
		"orgID":     orgID,
		"groupName": userGroupName,
	}).Info("Fetching user group by name from database")

	params := groupsdb.GetGroupByNameParams{
		Name:           userGroupName,
		OrganizationID: uuid.MustParse(orgID),
	}

	userGroup, err := s.userGroupRepo.GetGroupByName(ctx, params)
	if err != nil {
		if err == sql.ErrNoRows {
			s.logger.WithFields(logrus.Fields{
				"orgID":     orgID,
				"groupName": userGroupName,
			}).Warn("User group not found by name in database")
			return nil, apiErrors.ErrUserGroupNotFound
		}

		s.logger.WithFields(logrus.Fields{
			"orgID":     orgID,
			"groupName": userGroupName,
			"error":     err.Error(),
		}).Error("Failed to fetch user group by name from database")
		return nil, err
	}

	s.logger.WithFields(logrus.Fields{
		"orgID":     orgID,
		"groupID":   userGroup.ID.String(),
		"groupName": userGroup.Name,
	}).Info("User group found by name in database")

	return &userGroup, nil
}

// DeleteUserGroup removes a user group from an organization.
// Validates that the group exists and belongs to the specified organization before deletion.
func (s *UserGroupService) DeleteUserGroup(ctx context.Context, orgID, userGroupID string) error {
	s.logger.WithFields(logrus.Fields{
		"orgID":       orgID,
		"userGroupID": userGroupID,
	}).Info("Deleting user group from database")

	params := groupsdb.DeleteGroupParams{
		OrganizationID: uuid.MustParse(orgID),
		ID:             uuid.MustParse(userGroupID),
	}

	err := s.userGroupRepo.DeleteGroup(ctx, params)
	if err != nil {
		if err == sql.ErrNoRows {
			s.logger.WithFields(logrus.Fields{
				"orgID":       orgID,
				"userGroupID": userGroupID,
			}).Warn("User group deletion failed: group not found in database")
			return apiErrors.ErrUserGroupNotFound
		}

		s.logger.WithFields(logrus.Fields{
			"orgID":       orgID,
			"userGroupID": userGroupID,
			"error":       err.Error(),
		}).Error("Failed to delete user group from database")
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"orgID":       orgID,
		"userGroupID": userGroupID,
	}).Info("User group deleted successfully from database")

	// Clean up all group-related policies and memberships
	err = s.policyEnforcer.DeleteUserGroup(userGroupID)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"orgID":       orgID,
			"userGroupID": userGroupID,
			"error":       err.Error(),
		}).Error("Failed to delete group policies and memberships")
		return err
	}

	// Remove resource hierarchy
	parentResourcePath := fmt.Sprintf("/organizations/%s", orgID)
	childResourcePath := fmt.Sprintf("/organizations/%s/user-groups/%s", orgID, userGroupID)
	err = s.policyEnforcer.RemoveResourceHierarchy(parentResourcePath, childResourcePath)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"orgID":       orgID,
			"userGroupID": userGroupID,
			"error":       err.Error(),
		}).Error("Failed to remove resource hierarchy")
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"orgID":       orgID,
		"userGroupID": userGroupID,
	}).Info("User group and all associated policies deleted successfully")
	return nil
}

// ListUserGroups retrieves all user groups within an organization.
// Returns groups ordered by creation date for consistent pagination and display.
func (s *UserGroupService) ListUserGroups(ctx context.Context, orgID string) ([]groupsdb.ListGroupsByOrgRow, error) {
	s.logger.WithFields(logrus.Fields{
		"orgID": orgID,
	}).Info("Fetching user groups from database")

	groups, err := s.userGroupRepo.ListGroupsByOrg(ctx, uuid.MustParse(orgID))
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"orgID": orgID,
			"error": err.Error(),
		}).Error("Failed to fetch user groups from database")
		return nil, err
	}

	s.logger.WithFields(logrus.Fields{
		"orgID": orgID,
		"count": len(groups),
	}).Info("User groups fetched successfully from database")

	return groups, nil
}

// AddGroupMember adds a user to a user group.
// Validates that both the user and group exist, then creates the membership relationship.
// Handles duplicate membership attempts gracefully.
func (s *UserGroupService) AddGroupMember(ctx context.Context, req AddMemberRequest) error {
	s.logger.WithFields(logrus.Fields{
		"userGroupID": req.UserGroupID,
		"userName":    req.UserName,
	}).Info("Adding member to user group")

	// First, validate that the user exists by fetching user information
	user, err := s.userService.GetUserInfoByGithubUserName(ctx, req.UserName)
	if err != nil {
		if err == sql.ErrNoRows {
			s.logger.WithFields(logrus.Fields{
				"userGroupID": req.UserGroupID,
				"userName":    req.UserName,
			}).Warn("Add group member failed: user not found in system")
			return apiErrors.ErrUserNotFound
		}

		s.logger.WithFields(logrus.Fields{
			"userGroupID": req.UserGroupID,
			"userName":    req.UserName,
			"error":       err.Error(),
		}).Error("Failed to fetch user information for group membership")
		return err
	}

	// Create the group membership in database
	params := groupsdb.AddGroupMemberParams{
		UserGroupID: uuid.MustParse(req.UserGroupID),
		UserID:      user.ID,
	}

	err = s.userGroupRepo.AddGroupMember(ctx, params)
	if apiErrors.IsUniqueViolation(err) {
		s.logger.WithFields(logrus.Fields{
			"userGroupID": req.UserGroupID,
			"userName":    req.UserName,
			"userID":      user.ID.String(),
		}).Warn("Add group member failed: membership already exists")
		return apiErrors.ErrDuplicateMemberOfUserGroup
	}

	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"userGroupID": req.UserGroupID,
			"userName":    req.UserName,
			"userID":      user.ID.String(),
			"error":       err.Error(),
		}).Error("Failed to add member to user group in database")
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"userGroupID": req.UserGroupID,
		"userName":    req.UserName,
		"userID":      user.ID.String(),
	}).Info("Member added to user group successfully")
	err = s.policyEnforcer.AddUserToGroup(user.ID.String(), req.UserGroupID)
	if err != nil {
		s.logger.Errorf("unable to add bind user %s with user group %s", user.Name.String, req.UserGroupID)
		return err
	}
	return nil
}

// RemoveGroupMember removes a user from a user group.
// Validates that the user exists and is currently a member of the group before removal.
func (s *UserGroupService) RemoveGroupMember(ctx context.Context, req RemoveMemberRequest) error {
	s.logger.WithFields(logrus.Fields{
		"userGroupID": req.UserGroupID,
		"userName":    req.UserName,
	}).Info("Removing member from user group")

	// First, validate that the user exists by fetching user information
	user, err := s.userService.GetUserInfoByGithubUserName(ctx, req.UserName)
	if err != nil {
		if err == sql.ErrNoRows {
			s.logger.WithFields(logrus.Fields{
				"userGroupID": req.UserGroupID,
				"userName":    req.UserName,
			}).Warn("Remove group member failed: user not found in system")
			return apiErrors.ErrUserNotFound
		}

		s.logger.WithFields(logrus.Fields{
			"userGroupID": req.UserGroupID,
			"userName":    req.UserName,
			"error":       err.Error(),
		}).Error("Failed to fetch user information for group membership removal")
		return err
	}

	// Remove the group membership from database
	params := groupsdb.RemoveGroupMemberParams{
		UserGroupID: uuid.MustParse(req.UserGroupID),
		UserID:      user.ID,
	}

	res, err := s.userGroupRepo.RemoveGroupMember(ctx, params)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"userGroupID": req.UserGroupID,
			"userName":    req.UserName,
			"userID":      user.ID.String(),
			"error":       err.Error(),
		}).Error("Failed to remove member from user group in database")
		return err
	}

	// Check if any rows were affected to determine if membership existed
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"userGroupID": req.UserGroupID,
			"userName":    req.UserName,
			"userID":      user.ID.String(),
			"error":       err.Error(),
		}).Error("Failed to get rows affected count for member removal")
		return err
	}

	if rowsAffected == 0 {
		s.logger.WithFields(logrus.Fields{
			"userGroupID": req.UserGroupID,
			"userName":    req.UserName,
			"userID":      user.ID.String(),
		}).Warn("Remove group member failed: membership not found in database")
		return apiErrors.ErrUserMembershipNotFound
	}

	s.logger.WithFields(logrus.Fields{
		"userGroupID":  req.UserGroupID,
		"userName":     req.UserName,
		"userID":       user.ID.String(),
		"rowsAffected": rowsAffected,
	}).Info("Member removed from user group successfully")

	err = s.policyEnforcer.RemoveUserFromGroup(user.ID.String(), req.UserGroupID)
	if err != nil {
		s.logger.Errorf("unable to remove the user %s from group %s", req.UserName, req.UserGroupID)
		return err
	}
	return nil
}

// ListGroupMembers retrieves all members of a specific user group.
// Returns user details including name, email, and membership creation date.
func (s *UserGroupService) ListGroupMembers(ctx context.Context, userGroupId string) ([]groupsdb.ListGroupMembersRow, error) {
	s.logger.WithFields(logrus.Fields{
		"userGroupID": userGroupId,
	}).Info("Fetching group members from database")

	members, err := s.userGroupRepo.ListGroupMembers(ctx, uuid.MustParse(userGroupId))
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"userGroupID": userGroupId,
			"error":       err.Error(),
		}).Error("Failed to fetch group members from database")
		return nil, err
	}

	s.logger.WithFields(logrus.Fields{
		"userGroupID": userGroupId,
		"count":       len(members),
	}).Info("Group members fetched successfully from database")

	return members, nil
}
