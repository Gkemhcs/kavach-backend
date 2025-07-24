package iam

import (
	"context"

	apiErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	iam_db "github.com/Gkemhcs/kavach-backend/internal/iam/gen"
	"github.com/Gkemhcs/kavach-backend/internal/types"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// NewIamService creates a new IamService instance with the provided dependencies.
// This service handles business logic for IAM operations including role binding management.
func NewIamService(iam_repo iam_db.Querier, userResolver types.UserResolver, userGroupResolver types.UserGroupResolver, logger *logrus.Logger) *IamService {
	return &IamService{
		iamRepo:           iam_repo,
		userResolver:      userResolver,
		userGroupResolver: userGroupResolver,
		logger:            logger,
	}
}

// mapRole converts a string role to the corresponding database enum value.
// Supports owner, admin, editor, and viewer roles with viewer as the default.
func mapRole(role string) iam_db.UserRole {
	switch role {
	case "owner":
		return iam_db.UserRoleOwner
	case "admin":
		return iam_db.UserRoleAdmin
	case "editor":
		return iam_db.UserRoleEditor
	default:
		return iam_db.UserRoleViewer
	}
}

// mapResourceType converts a string resource type to the corresponding database enum value.
// Supports organization, secret_group, and environment resource types with environment as the default.
func mapResourceType(resourceType string) iam_db.ResourceType {
	switch resourceType {
	case "organization":
		return iam_db.ResourceTypeOrganization
	case "secret_group":
		return iam_db.ResourceTypeSecretGroup
	default:
		return iam_db.ResourceTypeEnvironment
	}
}

// IamService provides business logic for IAM (Identity and Access Management) operations.
// It coordinates between the repository layer and external services for user and group resolution.
type IamService struct {
	iamRepo           iam_db.Querier
	logger            *logrus.Logger
	userResolver      types.UserResolver
	userGroupResolver types.UserGroupResolver
}

// CreateRoleBinding creates a new role binding for a user on a specific resource.
// Used internally for creating role bindings with explicit user IDs.
func (s *IamService) CreateRoleBinding(ctx context.Context, req CreateRoleBindingRequest) (*iam_db.RoleBinding, error) {
	params := iam_db.CreateRoleBindingParams{
		UserID: uuid.NullUUID{
			UUID:  req.UserID,
			Valid: true,
		},
		Role:           mapRole(req.Role),
		ResourceType:   mapResourceType(req.ResourceType),
		ResourceID:     req.ResourceID,
		OrganizationID: req.OrganizationID,
		SecretGroupID:  req.SecretGroupID,
		EnvironmentID:  req.EnvironmentID,
	}
	binding, err := s.iamRepo.CreateRoleBinding(ctx, params)
	if err != nil {
		return nil, err
	}
	return &binding, nil
}

// ListAccessibleOrganizations retrieves all organizations that a user has access to.
// Returns organizations with the user's role level for each accessible organization.
func (s *IamService) ListAccessibleOrganizations(ctx context.Context, userID string) ([]iam_db.ListAccessibleOrganizationsRow, error) {
	bindings, err := s.iamRepo.ListAccessibleOrganizations(ctx, uuid.NullUUID{
		UUID:  uuid.MustParse(userID),
		Valid: true,
	})
	if err != nil {
		return nil, err
	}
	return bindings, nil
}

// ListAccessibleSecretGroups retrieves all secret groups within an organization that a user has access to.
// Returns secret groups with the user's role level for each accessible group.
func (s *IamService) ListAccessibleSecretGroups(ctx context.Context, userID, orgID string) ([]iam_db.ListAccessibleSecretGroupsRow, error) {
	params := iam_db.ListAccessibleSecretGroupsParams{
		UserID: uuid.NullUUID{
			UUID:  uuid.MustParse(userID),
			Valid: true,
		},
		OrganizationID: uuid.MustParse(orgID),
	}
	bindings, err := s.iamRepo.ListAccessibleSecretGroups(ctx, params)
	if err != nil {
		return nil, err
	}
	return bindings, nil
}

// ListAccessibleEnvironments retrieves all environments within a secret group that a user has access to.
// Returns environments with the user's role level for each accessible environment.
func (s *IamService) ListAccessibleEnvironments(ctx context.Context, userID, orgID, groupID string) ([]iam_db.ListAccessibleEnvironmentsRow, error) {
	params := iam_db.ListAccessibleEnvironmentsParams{
		UserID: uuid.NullUUID{
			UUID:  uuid.MustParse(userID),
			Valid: true,
		},
		OrganizationID: uuid.MustParse(orgID),
		SecretGroupID: uuid.NullUUID{
			UUID:  uuid.MustParse(groupID),
			Valid: true,
		},
	}
	bindings, err := s.iamRepo.ListAccessibleEnvironments(ctx, params)
	if err != nil {
		return nil, err
	}
	return bindings, nil
}

// GrantRoleBinding grants a role to either a user or a user group on a specific resource.
// Validates that the user/group exists, then creates or updates the role binding.
// Handles duplicate role binding attempts gracefully using UPSERT semantics.
func (s *IamService) GrantRoleBinding(ctx context.Context, req GrantRoleBindingRequest) error {
	s.logger.WithFields(logrus.Fields{
		"userName":     req.UserName,
		"groupName":    req.GroupName,
		"role":         req.Role,
		"resourceType": req.ResourceType,
		"resourceID":   req.ResourceID.String(),
		"orgID":        req.OrganizationID.String(),
	}).Info("Granting role binding in database")

	params := iam_db.GrantRoleBindingParams{
		ResourceType:   mapResourceType(req.ResourceType),
		Role:           mapRole(req.Role),
		ResourceID:     req.ResourceID,
		OrganizationID: req.OrganizationID,
		SecretGroupID:  req.SecretGroupID,
		EnvironmentID:  req.EnvironmentID,
	}

	// Handle user-based role binding
	if req.UserName == "" {
		params.UserID = uuid.NullUUID{Valid: false}

		// Resolve user group by name
		group, err := s.userGroupResolver.GetUserGroupByName(ctx, req.GroupName, req.OrganizationID.String())
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"groupName": req.GroupName,
				"orgID":     req.OrganizationID.String(),
				"error":     err.Error(),
			}).Error("Failed to resolve user group for role binding")
			return err
		}

		params.GroupID = uuid.NullUUID{
			UUID:  group.ID,
			Valid: true,
		}

		s.logger.WithFields(logrus.Fields{
			"groupID":      group.ID.String(),
			"groupName":    req.GroupName,
			"role":         req.Role,
			"resourceType": req.ResourceType,
			"resourceID":   req.ResourceID.String(),
		}).Info("Resolved user group for role binding")
	} else {
		// Handle group-based role binding
		params.GroupID = uuid.NullUUID{Valid: false}

		// Resolve user by GitHub username
		user, err := s.userResolver.GetUserInfoByGithubUserName(ctx, req.UserName)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"userName": req.UserName,
				"error":    err.Error(),
			}).Error("Failed to resolve user for role binding")
			return err
		}

		params.UserID = uuid.NullUUID{
			UUID:  user.ID,
			Valid: true,
		}

		s.logger.WithFields(logrus.Fields{
			"userID":       user.ID.String(),
			"userName":     req.UserName,
			"role":         req.Role,
			"resourceType": req.ResourceType,
			"resourceID":   req.ResourceID.String(),
		}).Info("Resolved user for role binding")
	}

	// Create or update the role binding in database
	err := s.iamRepo.GrantRoleBinding(ctx, params)
	if err != nil {
		if apiErrors.IsUniqueViolation(err) || apiErrors.IsCheckConstraintViolation(err) {
			s.logger.WithFields(logrus.Fields{
				"userName":     req.UserName,
				"groupName":    req.GroupName,
				"role":         req.Role,
				"resourceType": req.ResourceType,
				"resourceID":   req.ResourceID.String(),
			}).Warn("Role binding grant failed: duplicate binding or constraint violation")
			return apiErrors.ErrDuplicateRoleBinding
		}

		s.logger.WithFields(logrus.Fields{
			"userName":     req.UserName,
			"groupName":    req.GroupName,
			"role":         req.Role,
			"resourceType": req.ResourceType,
			"resourceID":   req.ResourceID.String(),
			"error":        err.Error(),
		}).Error("Failed to grant role binding in database")
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"userName":     req.UserName,
		"groupName":    req.GroupName,
		"role":         req.Role,
		"resourceType": req.ResourceType,
		"resourceID":   req.ResourceID.String(),
	}).Info("Role binding granted successfully in database")

	return nil
}

// RevokeRoleBinding revokes a role from either a user or a user group on a specific resource.
// Validates that the user/group exists, then removes the role binding.
// Returns an error if the role binding doesn't exist.
func (s *IamService) RevokeRoleBinding(ctx context.Context, req RevokeRoleBindingRequest) error {
	s.logger.WithFields(logrus.Fields{
		"userName":     req.UserName,
		"groupName":    req.GroupName,
		"role":         req.Role,
		"resourceType": req.ResourceType,
		"resourceID":   req.ResourceID.String(),
		"orgID":        req.OrganizationID.String(),
	}).Info("Revoking role binding from database")

	params := iam_db.RevokeRoleBindingParams{
		ResourceType: mapResourceType(req.ResourceType),
		Role:         mapRole(req.Role),
		ResourceID:   req.ResourceID,
	}

	// Handle user-based role binding revocation
	if req.UserName == "" {
		params.UserID = uuid.NullUUID{Valid: false}

		// Resolve user group by name
		group, err := s.userGroupResolver.GetUserGroupByName(ctx, req.GroupName, req.OrganizationID.String())
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"groupName": req.GroupName,
				"orgID":     req.OrganizationID.String(),
				"error":     err.Error(),
			}).Error("Failed to resolve user group for role binding revocation")
			return err
		}

		params.GroupID = uuid.NullUUID{
			UUID:  group.ID,
			Valid: true,
		}

		s.logger.WithFields(logrus.Fields{
			"groupID":      group.ID.String(),
			"groupName":    req.GroupName,
			"role":         req.Role,
			"resourceType": req.ResourceType,
			"resourceID":   req.ResourceID.String(),
		}).Info("Resolved user group for role binding revocation")
	} else {
		// Handle group-based role binding revocation
		params.GroupID = uuid.NullUUID{Valid: false}

		// Resolve user by GitHub username
		user, err := s.userResolver.GetUserInfoByGithubUserName(ctx, req.UserName)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"userName": req.UserName,
				"error":    err.Error(),
			}).Error("Failed to resolve user for role binding revocation")
			return err
		}

		params.UserID = uuid.NullUUID{
			UUID:  user.ID,
			Valid: true,
		}

		s.logger.WithFields(logrus.Fields{
			"userID":       user.ID.String(),
			"userName":     req.UserName,
			"role":         req.Role,
			"resourceType": req.ResourceType,
			"resourceID":   req.ResourceID.String(),
		}).Info("Resolved user for role binding revocation")
	}

	// Remove the role binding from database
	result, err := s.iamRepo.RevokeRoleBinding(ctx, params)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"userName":     req.UserName,
			"groupName":    req.GroupName,
			"role":         req.Role,
			"resourceType": req.ResourceType,
			"resourceID":   req.ResourceID.String(),
			"error":        err.Error(),
		}).Error("Failed to revoke role binding from database")
		return err
	}

	// Check if any rows were affected to determine if role binding existed
	rows, err := result.RowsAffected()
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"userName":     req.UserName,
			"groupName":    req.GroupName,
			"role":         req.Role,
			"resourceType": req.ResourceType,
			"resourceID":   req.ResourceID.String(),
			"error":        err.Error(),
		}).Error("Failed to get rows affected count for role binding revocation")
		return err
	}

	if rows == 0 {
		s.logger.WithFields(logrus.Fields{
			"userName":     req.UserName,
			"groupName":    req.GroupName,
			"role":         req.Role,
			"resourceType": req.ResourceType,
			"resourceID":   req.ResourceID.String(),
		}).Warn("Role binding revocation failed: binding not found in database")
		return apiErrors.ErrRoleBindingNotFound
	}

	s.logger.WithFields(logrus.Fields{
		"userName":     req.UserName,
		"groupName":    req.GroupName,
		"role":         req.Role,
		"resourceType": req.ResourceType,
		"resourceID":   req.ResourceID.String(),
		"rowsAffected": rows,
	}).Info("Role binding revoked successfully from database")

	return nil
}
