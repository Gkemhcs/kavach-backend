package iam

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Gkemhcs/kavach-backend/internal/authz"
	apiErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	iam_db "github.com/Gkemhcs/kavach-backend/internal/iam/gen"
	"github.com/Gkemhcs/kavach-backend/internal/types"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// NewIamService creates a new IamService instance with the provided dependencies.
// This service handles business logic for IAM operations including role binding management.
func NewIamService(iam_repo iam_db.Querier, userResolver types.UserResolver, userGroupResolver types.UserGroupResolver, logger *logrus.Logger, policyEnforcer authz.Enforcer) *IamService {
	return &IamService{
		iamRepo:           iam_repo,
		userResolver:      userResolver,
		userGroupResolver: userGroupResolver,
		logger:            logger,
		policyEnforcer:    policyEnforcer,
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
	policyEnforcer    authz.Enforcer
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

func (s *IamService) DeleteRoleBinding(ctx context.Context, req DeleteRoleBindingRequest) error {

	params := iam_db.DeleteRoleBindingParams{
		ResourceType: iam_db.ResourceType(req.ResourceType),
		ResourceID:   req.ResourceID,
	}
	err := s.iamRepo.DeleteRoleBinding(ctx, params)
	if err != nil {
		if err == sql.ErrNoRows {
			return apiErrors.ErrRoleBindingNotFound
		}
		return err
	}
	return nil

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

// ListAccessibleOrganizationsEnhanced retrieves all organizations that a user has access to
// using enhanced RBAC with hierarchical inheritance and group membership support.
// Returns organizations with the user's effective role level for each accessible organization.
func (s *IamService) ListAccessibleOrganizationsEnhanced(ctx context.Context, userID string) ([]iam_db.ListAccessibleOrganizationsRow, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	bindings, err := s.iamRepo.ListAccessibleOrganizationsEnhanced(ctx, uuid.NullUUID{
		UUID:  userUUID,
		Valid: true,
	})
	if err != nil {
		return nil, err
	}
	return convertEnhancedToLegacyOrgResults(bindings), nil
}

// convertEnhancedToLegacyOrgResults converts enhanced RBAC results to legacy format
// for backward compatibility with existing code.
func convertEnhancedToLegacyOrgResults(enhanced []iam_db.ListAccessibleOrganizationsEnhancedRow) []iam_db.ListAccessibleOrganizationsRow {
	legacy := make([]iam_db.ListAccessibleOrganizationsRow, len(enhanced))
	for i, row := range enhanced {
		legacy[i] = iam_db.ListAccessibleOrganizationsRow(row)
	}
	return legacy
}

// convertEnhancedToLegacySecretGroupResults converts enhanced RBAC results to legacy format
// for backward compatibility with existing code.
func convertEnhancedToLegacySecretGroupResults(enhanced []iam_db.ListAccessibleSecretGroupsEnhancedRow) []iam_db.ListAccessibleSecretGroupsRow {
	legacy := make([]iam_db.ListAccessibleSecretGroupsRow, len(enhanced))
	for i, row := range enhanced {
		legacy[i] = iam_db.ListAccessibleSecretGroupsRow{
			ID:               uuid.NullUUID{UUID: row.ID, Valid: true},
			Name:             row.Name,
			OrganizationName: row.OrganizationName,
			Role:             row.Role,
			InheritedFrom:    row.InheritedFrom,
		}
	}
	return legacy
}

// convertEnhancedToLegacyEnvironmentResults converts enhanced RBAC results to legacy format
// for backward compatibility with existing code.
func convertEnhancedToLegacyEnvironmentResults(enhanced []iam_db.ListAccessibleEnvironmentsEnhancedRow) []iam_db.ListAccessibleEnvironmentsRow {
	legacy := make([]iam_db.ListAccessibleEnvironmentsRow, len(enhanced))
	for i, row := range enhanced {
		legacy[i] = iam_db.ListAccessibleEnvironmentsRow{
			ID:              uuid.NullUUID{UUID: row.ID, Valid: true},
			Name:            row.Name,
			SecretGroupName: row.SecretGroupName,
			Role:            row.Role,
			InheritedFrom:   row.InheritedFrom,
		}
	}
	return legacy
}

// ListAccessibleSecretGroups retrieves all secret groups within an organization that a user has access to.
// Returns secret groups with the user's role level for each accessible group.
func (s *IamService) ListAccessibleSecretGroups(ctx context.Context, userID, orgID string) ([]iam_db.ListAccessibleSecretGroupsRow, error) {
	return s.ListAccessibleSecretGroupsEnhanced(ctx, userID, orgID)
}

// ListAccessibleSecretGroupsEnhanced retrieves all secret groups within an organization that a user has access to
// using enhanced RBAC with hierarchical inheritance and group membership support.
// Returns secret groups with the user's effective role level for each accessible group.
func (s *IamService) ListAccessibleSecretGroupsEnhanced(ctx context.Context, userID, orgID string) ([]iam_db.ListAccessibleSecretGroupsRow, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, err
	}
	params := iam_db.ListAccessibleSecretGroupsEnhancedParams{
		UserID: uuid.NullUUID{
			UUID:  userUUID,
			Valid: true,
		},
		OrganizationID: orgUUID,
	}
	bindings, err := s.iamRepo.ListAccessibleSecretGroupsEnhanced(ctx, params)
	if err != nil {
		return nil, err
	}
	return convertEnhancedToLegacySecretGroupResults(bindings), nil
}

// ListAccessibleSecretGroupsEnhancedWithInheritance retrieves all secret groups within an organization that a user has access to
// using enhanced RBAC with hierarchical inheritance and group membership support.
// Returns secret groups with the user's effective role level and inheritance information for each accessible group.
func (s *IamService) ListAccessibleSecretGroupsEnhancedWithInheritance(ctx context.Context, userID, orgID string) ([]iam_db.ListAccessibleSecretGroupsEnhancedRow, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, err
	}
	params := iam_db.ListAccessibleSecretGroupsEnhancedParams{
		UserID: uuid.NullUUID{
			UUID:  userUUID,
			Valid: true,
		},
		OrganizationID: orgUUID,
	}
	return s.iamRepo.ListAccessibleSecretGroupsEnhanced(ctx, params)
}

// ListAccessibleEnvironments retrieves all environments within a secret group that a user has access to.
// Returns environments with the user's role level for each accessible environment.
func (s *IamService) ListAccessibleEnvironments(ctx context.Context, userID, orgID, groupID string) ([]iam_db.ListAccessibleEnvironmentsRow, error) {
	return s.ListAccessibleEnvironmentsEnhanced(ctx, userID, orgID, groupID)
}

// ListAccessibleEnvironmentsEnhanced retrieves all environments within a secret group that a user has access to
// using enhanced RBAC with hierarchical inheritance and group membership support.
// Returns environments with the user's effective role level for each accessible environment.
func (s *IamService) ListAccessibleEnvironmentsEnhanced(ctx context.Context, userID, orgID, groupID string) ([]iam_db.ListAccessibleEnvironmentsRow, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, err
	}
	groupUUID, err := uuid.Parse(groupID)
	if err != nil {
		return nil, err
	}
	params := iam_db.ListAccessibleEnvironmentsEnhancedParams{
		UserID: uuid.NullUUID{
			UUID:  userUUID,
			Valid: true,
		},
		OrganizationID: orgUUID,
		SecretGroupID:  groupUUID,
	}
	bindings, err := s.iamRepo.ListAccessibleEnvironmentsEnhanced(ctx, params)
	if err != nil {
		return nil, err
	}
	return convertEnhancedToLegacyEnvironmentResults(bindings), nil
}

// ListAccessibleEnvironmentsEnhancedWithInheritance retrieves all environments within a secret group that a user has access to
// using enhanced RBAC with hierarchical inheritance and group membership support.
// Returns environments with the user's effective role level and inheritance information for each accessible environment.
func (s *IamService) ListAccessibleEnvironmentsEnhancedWithInheritance(ctx context.Context, userID, orgID, groupID string) ([]iam_db.ListAccessibleEnvironmentsEnhancedRow, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, err
	}
	groupUUID, err := uuid.Parse(groupID)
	if err != nil {
		return nil, err
	}
	params := iam_db.ListAccessibleEnvironmentsEnhancedParams{
		UserID: uuid.NullUUID{
			UUID:  userUUID,
			Valid: true,
		},
		OrganizationID: orgUUID,
		SecretGroupID:  groupUUID,
	}
	return s.iamRepo.ListAccessibleEnvironmentsEnhanced(ctx, params)
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
	var sub string
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
		sub = fmt.Sprintf("group:%s", group.ID.String())
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
		sub = fmt.Sprintf("user:%s", user.ID.String())

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
	var resource string

	switch req.ResourceType {
	case "environment":
		resource = fmt.Sprintf("/organizations/%s/secret-groups/%s/environments/%s", req.OrganizationID, req.SecretGroupID.UUID.String(), req.EnvironmentID.UUID.String())
	case "secret_group":
		resource = fmt.Sprintf("/organizations/%s/secret-groups/%s", req.OrganizationID, req.SecretGroupID.UUID.String())
	default:
		resource = fmt.Sprintf("/organizations/%s", req.OrganizationID)
	}

	err = s.policyEnforcer.GrantRole(sub, req.Role, resource)
	logEntry := s.logger.WithFields(logrus.Fields{
		"sub":      sub,
		"resource": resource,
		"role":     req.Role,
	})
	if err != nil {
		logEntry.Errorf("cannot add policy to the authorization")
		return err
	}

	logEntry.Info("succesfully added policy to authorization")
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
	var sub string
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
		sub = fmt.Sprintf("group:%s", group.ID.String())
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
		sub = fmt.Sprintf("user:%s", user.ID.String())
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

	var resource string
	switch req.ResourceType {
	case "secret_group":
		resource = fmt.Sprintf("/organizations/%s/secret-groups/%s",
			req.OrganizationID.String(),
			req.SecretGroupID.UUID.String())
	case "environment":
		resource = fmt.Sprintf("/organizations/%s/secret-groups/%s/environments/%s",
			req.OrganizationID.String(),
			req.SecretGroupID.UUID.String(),
			req.EnvironmentID.UUID.String())
	default:
		resource = fmt.Sprintf("/organizations/%s", req.OrganizationID.String())
	}
	// Use cascading revocation to ensure all child resources are also revoked
	err = s.policyEnforcer.RevokeRoleCascade(sub, req.Role, resource)
	logEntry := s.logger.WithFields(logrus.Fields{
		"sub":      sub,
		"resource": resource,
		"role":     req.Role,
	})

	if err != nil {
		logEntry.Errorf("cannot remove the binding: %v", err)
		return err
	}
	logEntry.Info("successfully removed policy and all child resource policies")

	// Handle ownership transfer for organization and secret group revocations
	if req.ResourceType == "organization" || req.ResourceType == "secret_group" {
		err = s.handleOwnershipTransfer(ctx, req, params)
		if err != nil {
			logEntry.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Failed to handle ownership transfer during role binding revocation")
			// Don't return error here as the role binding was already revoked
			// Just log the ownership transfer failure
		}
	}

	return nil
}

// handleOwnershipTransfer handles the transfer of child resource ownership when revoking role bindings
func (s *IamService) handleOwnershipTransfer(ctx context.Context, req RevokeRoleBindingRequest, params iam_db.RevokeRoleBindingParams) error {
	logEntry := s.logger.WithFields(logrus.Fields{
		"operation":    "ownership_transfer",
		"resourceType": req.ResourceType,
		"resourceID":   req.ResourceID.String(),
		"userName":     req.UserName,
		"groupName":    req.GroupName,
	})

	logEntry.Info("Starting ownership transfer process")

	// Get the parent resource owner
	var parentOwnerID uuid.UUID
	var err error

	if req.ResourceType == "organization" {
		parentOwnerID, err = s.getOrganizationOwner(ctx, req.ResourceID)
	} else if req.ResourceType == "secret_group" {
		parentOwnerID, err = s.getSecretGroupOwner(ctx, req.ResourceID)
	} else {
		// No ownership transfer needed for environments
		return nil
	}

	if err != nil {
		logEntry.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Failed to get parent resource owner")
		return err
	}

	logEntry.WithFields(logrus.Fields{
		"parentOwnerID": parentOwnerID.String(),
	}).Info("Found parent resource owner")

	// Handle user-based revocation
	if req.UserName != "" {
		err = s.transferUserOwnership(ctx, req, parentOwnerID, params.UserID.UUID)
	} else {
		// Handle group-based revocation
		err = s.transferGroupOwnership(ctx, req, parentOwnerID, params.GroupID.UUID)
	}

	if err != nil {
		logEntry.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Failed to transfer ownership")
		return err
	}

	logEntry.Info("Ownership transfer completed successfully")
	return nil
}

// getOrganizationOwner gets the owner of an organization
func (s *IamService) getOrganizationOwner(ctx context.Context, orgID uuid.UUID) (uuid.UUID, error) {
	owner, err := s.iamRepo.GetOrganizationOwner(ctx, orgID)
	if err != nil {
		return uuid.Nil, err
	}
	return owner.UserID.UUID, nil
}

// getSecretGroupOwner gets the owner of a secret group
func (s *IamService) getSecretGroupOwner(ctx context.Context, groupID uuid.UUID) (uuid.UUID, error) {
	owner, err := s.iamRepo.GetSecretGroupOwner(ctx, groupID)
	if err != nil {
		return uuid.Nil, err
	}
	return owner.UserID.UUID, nil
}

// transferUserOwnership transfers ownership of child resources created by a revoked user
func (s *IamService) transferUserOwnership(ctx context.Context, req RevokeRoleBindingRequest, parentOwnerID, revokedUserID uuid.UUID) error {
	logEntry := s.logger.WithFields(logrus.Fields{
		"operation":     "transfer_user_ownership",
		"resourceType":  req.ResourceType,
		"resourceID":    req.ResourceID.String(),
		"revokedUserID": revokedUserID.String(),
		"parentOwnerID": parentOwnerID.String(),
	})

	if req.ResourceType == "organization" {
		// Transfer secret groups where the revoked user has role bindings
		secretGroups, err := s.iamRepo.GetSecretGroupsWithUserRoleBindings(ctx, iam_db.GetSecretGroupsWithUserRoleBindingsParams{
			OrganizationID: req.OrganizationID,
			UserID:         uuid.NullUUID{UUID: revokedUserID, Valid: true},
		})
		if err != nil {
			logEntry.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Failed to get secret groups with user role bindings")
			return err
		}

		if len(secretGroups) > 0 {
			logEntry.WithFields(logrus.Fields{
				"secretGroupCount": len(secretGroups),
			}).Info("Found secret groups to transfer ownership")

			// Extract secret group IDs
			secretGroupIDs := make([]uuid.UUID, len(secretGroups))
			for i, sg := range secretGroups {
				secretGroupIDs[i] = sg.ID
			}

			// Batch transfer ownership by updating role bindings
			err = s.iamRepo.BatchTransferSecretGroupRoleBindingOwnership(ctx, iam_db.BatchTransferSecretGroupRoleBindingOwnershipParams{
				Column1: secretGroupIDs,
				UserID:  uuid.NullUUID{UUID: parentOwnerID, Valid: true},
			})
			if err != nil {
				logEntry.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("Failed to batch transfer secret group role binding ownership")
				return err
			}

			logEntry.WithFields(logrus.Fields{
				"transferredSecretGroups": len(secretGroups),
			}).Info("Successfully transferred secret group ownership")

			// Also transfer ownership of environments within these secret groups
			for _, secretGroup := range secretGroups {
				environments, err := s.iamRepo.GetEnvironmentsWithUserRoleBindings(ctx, iam_db.GetEnvironmentsWithUserRoleBindingsParams{
					SecretGroupID: secretGroup.ID,
					UserID:        uuid.NullUUID{UUID: revokedUserID, Valid: true},
				})
				if err != nil {
					logEntry.WithFields(logrus.Fields{
						"error":         err.Error(),
						"secretGroupID": secretGroup.ID.String(),
					}).Error("Failed to get environments with user role bindings for secret group")
					continue // Continue with other secret groups even if one fails
				}

				if len(environments) > 0 {
					logEntry.WithFields(logrus.Fields{
						"secretGroupID":    secretGroup.ID.String(),
						"environmentCount": len(environments),
					}).Info("Found environments to transfer ownership within secret group")

					// Extract environment IDs
					environmentIDs := make([]uuid.UUID, len(environments))
					for i, env := range environments {
						environmentIDs[i] = env.ID
					}

					// Batch transfer ownership by updating role bindings
					err = s.iamRepo.BatchTransferEnvironmentRoleBindingOwnership(ctx, iam_db.BatchTransferEnvironmentRoleBindingOwnershipParams{
						Column1: environmentIDs,
						UserID:  uuid.NullUUID{UUID: parentOwnerID, Valid: true},
					})
					if err != nil {
						logEntry.WithFields(logrus.Fields{
							"error":          err.Error(),
							"secretGroupID":  secretGroup.ID.String(),
							"environmentIDs": environmentIDs,
						}).Error("Failed to batch transfer environment role binding ownership")
						continue
					}

					logEntry.WithFields(logrus.Fields{
						"secretGroupID":           secretGroup.ID.String(),
						"transferredEnvironments": len(environments),
					}).Info("Successfully transferred environment ownership within secret group")
				}
			}
		}

	} else if req.ResourceType == "secret_group" {
		// Transfer environments where the revoked user has role bindings
		environments, err := s.iamRepo.GetEnvironmentsWithUserRoleBindings(ctx, iam_db.GetEnvironmentsWithUserRoleBindingsParams{
			SecretGroupID: req.SecretGroupID.UUID,
			UserID:        uuid.NullUUID{UUID: revokedUserID, Valid: true},
		})
		if err != nil {
			logEntry.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Failed to get environments with user role bindings")
			return err
		}

		if len(environments) > 0 {
			logEntry.WithFields(logrus.Fields{
				"environmentCount": len(environments),
			}).Info("Found environments to transfer ownership")

			// Extract environment IDs
			environmentIDs := make([]uuid.UUID, len(environments))
			for i, env := range environments {
				environmentIDs[i] = env.ID
			}

			// Batch transfer ownership by updating role bindings
			err = s.iamRepo.BatchTransferEnvironmentRoleBindingOwnership(ctx, iam_db.BatchTransferEnvironmentRoleBindingOwnershipParams{
				Column1: environmentIDs,
				UserID:  uuid.NullUUID{UUID: parentOwnerID, Valid: true},
			})
			if err != nil {
				logEntry.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("Failed to batch transfer environment role binding ownership")
				return err
			}

			logEntry.WithFields(logrus.Fields{
				"transferredEnvironments": len(environments),
			}).Info("Successfully transferred environment ownership")
		}
	}

	return nil
}

// transferGroupOwnership transfers ownership of child resources created by members of a revoked group
func (s *IamService) transferGroupOwnership(ctx context.Context, req RevokeRoleBindingRequest, parentOwnerID, revokedGroupID uuid.UUID) error {
	logEntry := s.logger.WithFields(logrus.Fields{
		"operation":      "transfer_group_ownership",
		"resourceType":   req.ResourceType,
		"resourceID":     req.ResourceID.String(),
		"revokedGroupID": revokedGroupID.String(),
		"parentOwnerID":  parentOwnerID.String(),
	})

	if req.ResourceType == "organization" {
		// Transfer secret groups where members of the revoked group have role bindings
		secretGroups, err := s.iamRepo.GetSecretGroupsWithGroupRoleBindings(ctx, iam_db.GetSecretGroupsWithGroupRoleBindingsParams{
			OrganizationID: req.OrganizationID,
			UserGroupID:    revokedGroupID,
		})
		if err != nil {
			logEntry.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Failed to get secret groups with group role bindings")
			return err
		}

		if len(secretGroups) > 0 {
			logEntry.WithFields(logrus.Fields{
				"secretGroupCount": len(secretGroups),
			}).Info("Found secret groups to transfer ownership from group members")

			// Extract secret group IDs
			secretGroupIDs := make([]uuid.UUID, len(secretGroups))
			for i, sg := range secretGroups {
				secretGroupIDs[i] = sg.ID
			}

			// Batch transfer ownership by updating role bindings
			err = s.iamRepo.BatchTransferSecretGroupRoleBindingOwnership(ctx, iam_db.BatchTransferSecretGroupRoleBindingOwnershipParams{
				Column1: secretGroupIDs,
				UserID:  uuid.NullUUID{UUID: parentOwnerID, Valid: true},
			})
			if err != nil {
				logEntry.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("Failed to batch transfer secret group role binding ownership from group members")
				return err
			}

			logEntry.WithFields(logrus.Fields{
				"transferredSecretGroups": len(secretGroups),
			}).Info("Successfully transferred secret group ownership from group members")

			// Also transfer ownership of environments within these secret groups
			for _, secretGroup := range secretGroups {
				environments, err := s.iamRepo.GetEnvironmentsWithGroupRoleBindings(ctx, iam_db.GetEnvironmentsWithGroupRoleBindingsParams{
					SecretGroupID: secretGroup.ID,
					UserGroupID:   revokedGroupID,
				})
				if err != nil {
					logEntry.WithFields(logrus.Fields{
						"error":         err.Error(),
						"secretGroupID": secretGroup.ID.String(),
					}).Error("Failed to get environments with group role bindings for secret group")
					continue // Continue with other secret groups even if one fails
				}

				if len(environments) > 0 {
					logEntry.WithFields(logrus.Fields{
						"secretGroupID":    secretGroup.ID.String(),
						"environmentCount": len(environments),
					}).Info("Found environments to transfer ownership within secret group from group members")

					// Extract environment IDs
					environmentIDs := make([]uuid.UUID, len(environments))
					for i, env := range environments {
						environmentIDs[i] = env.ID
					}

					// Batch transfer ownership by updating role bindings
					err = s.iamRepo.BatchTransferEnvironmentRoleBindingOwnership(ctx, iam_db.BatchTransferEnvironmentRoleBindingOwnershipParams{
						Column1: environmentIDs,
						UserID:  uuid.NullUUID{UUID: parentOwnerID, Valid: true},
					})
					if err != nil {
						logEntry.WithFields(logrus.Fields{
							"error":          err.Error(),
							"secretGroupID":  secretGroup.ID.String(),
							"environmentIDs": environmentIDs,
						}).Error("Failed to batch transfer environment role binding ownership from group members")
						continue
					}

					logEntry.WithFields(logrus.Fields{
						"secretGroupID":           secretGroup.ID.String(),
						"transferredEnvironments": len(environments),
					}).Info("Successfully transferred environment ownership within secret group from group members")
				}
			}
		}

	} else if req.ResourceType == "secret_group" {
		// Transfer environments where members of the revoked group have role bindings
		environments, err := s.iamRepo.GetEnvironmentsWithGroupRoleBindings(ctx, iam_db.GetEnvironmentsWithGroupRoleBindingsParams{
			SecretGroupID: req.SecretGroupID.UUID,
			UserGroupID:   revokedGroupID,
		})
		if err != nil {
			logEntry.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Failed to get environments with group role bindings")
			return err
		}

		if len(environments) > 0 {
			logEntry.WithFields(logrus.Fields{
				"environmentCount": len(environments),
			}).Info("Found environments to transfer ownership from group members")

			// Extract environment IDs
			environmentIDs := make([]uuid.UUID, len(environments))
			for i, env := range environments {
				environmentIDs[i] = env.ID
			}

			// Batch transfer ownership by updating role bindings
			err = s.iamRepo.BatchTransferEnvironmentRoleBindingOwnership(ctx, iam_db.BatchTransferEnvironmentRoleBindingOwnershipParams{
				Column1: environmentIDs,
				UserID:  uuid.NullUUID{UUID: parentOwnerID, Valid: true},
			})
			if err != nil {
				logEntry.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("Failed to batch transfer environment role binding ownership from group members")
				return err
			}

			logEntry.WithFields(logrus.Fields{
				"transferredEnvironments": len(environments),
			}).Info("Successfully transferred environment ownership from group members")
		}
	}

	return nil
}

// ListOrganizationRoleBindings retrieves all role bindings for an organization with resolved names
func (s *IamService) ListOrganizationRoleBindings(ctx context.Context, orgID string) ([]iam_db.ListOrganizationRoleBindingsRow, error) {
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, err
	}

	bindings, err := s.iamRepo.ListOrganizationRoleBindings(ctx, orgUUID)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"orgID": orgID,
			"error": err.Error(),
		}).Error("Failed to list organization role bindings")
		return nil, err
	}

	s.logger.WithFields(logrus.Fields{
		"orgID":        orgID,
		"bindingCount": len(bindings),
	}).Info("Successfully retrieved organization role bindings")

	return bindings, nil
}

// ListSecretGroupRoleBindings retrieves all role bindings for a secret group with resolved names
func (s *IamService) ListSecretGroupRoleBindings(ctx context.Context, groupID uuid.UUID) ([]iam_db.ListSecretGroupRoleBindingsRow, error) {
	bindings, err := s.iamRepo.ListSecretGroupRoleBindings(ctx, uuid.NullUUID{
		UUID:  groupID,
		Valid: true,
	})
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"groupID": groupID.String(),
			"error":   err.Error(),
		}).Error("Failed to list secret group role bindings")
		return nil, err
	}

	s.logger.WithFields(logrus.Fields{
		"groupID":      groupID.String(),
		"bindingCount": len(bindings),
	}).Info("Successfully retrieved secret group role bindings")

	return bindings, nil
}

// ListEnvironmentRoleBindings retrieves all role bindings for an environment with resolved names
func (s *IamService) ListEnvironmentRoleBindings(ctx context.Context, envID uuid.UUID) ([]iam_db.ListEnvironmentRoleBindingsRow, error) {
	bindings, err := s.iamRepo.ListEnvironmentRoleBindings(ctx, uuid.NullUUID{
		UUID:  envID,
		Valid: true,
	})
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"envID": envID.String(),
			"error": err.Error(),
		}).Error("Failed to list environment role bindings")
		return nil, err
	}

	s.logger.WithFields(logrus.Fields{
		"envID":        envID.String(),
		"bindingCount": len(bindings),
	}).Info("Successfully retrieved environment role bindings")

	return bindings, nil
}
