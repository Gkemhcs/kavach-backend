package org

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Gkemhcs/kavach-backend/internal/authz"
	appErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	"github.com/Gkemhcs/kavach-backend/internal/iam"
	iam_db "github.com/Gkemhcs/kavach-backend/internal/iam/gen"

	orgdb "github.com/Gkemhcs/kavach-backend/internal/org/gen"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// OrganizationService provides business logic for organizations.
// Encapsulates all organization-related operations and validation.
type OrganizationService struct {
	repo           orgdb.Querier
	logger         *logrus.Logger
	iamService     iam.IamService
	policyEnforcer authz.Enforcer
}

// NewOrganizationService creates a new OrganizationService.
// Used to inject dependencies and enable testability.
func NewOrganizationService(repo orgdb.Querier, logger *logrus.Logger, iamService iam.IamService, policEnforcer authz.Enforcer) *OrganizationService {
	return &OrganizationService{repo, logger, iamService, policEnforcer}

}

// CreateOrganization creates a new organization for the user.
// Adds the creator as the owner member of the organization.
func (s *OrganizationService) CreateOrganization(ctx context.Context, req CreateOrganizationRequest) (*orgdb.Organization, error) {
	s.logger.Infof("Creating organization for user_id=%s", req.UserID)
	ownerID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}
	params := orgdb.CreateOrganizationParams{
		Name:        req.Name,
		Description: utils.DerefString(req.Description),
		OwnerID:     ownerID,
	}
	org, err := s.repo.CreateOrganization(ctx, params)
	if appErrors.IsUniqueViolation(err) {
		return nil, appErrors.ErrDuplicateOrganization
	}
	if err != nil {
		s.logger.Error(err)
		return nil, appErrors.ErrInternalServer
	}
	createBindingParams := iam.CreateRoleBindingRequest{
		UserID:         ownerID,
		Role:           "owner",
		ResourceType:   "organization",
		ResourceID:     org.ID,
		OrganizationID: org.ID,
		SecretGroupID:  uuid.NullUUID{Valid: false},

		EnvironmentID: uuid.NullUUID{Valid: false},
	}
	_, err = s.iamService.CreateRoleBinding(ctx, createBindingParams)
	if err != nil {
		return nil, err
	}

	// Grant secure resource-specific permissions instead of wildcard access
	resourcePath := fmt.Sprintf("/organizations/%s", org.ID)
	err = s.policyEnforcer.GrantRoleWithPermissions(req.UserID, "owner", resourcePath)
	if err != nil {
		s.logger.Errorf("Failed to grant secure permissions for organization %s: %v", org.ID, err)
		return nil, err
	}

	return &org, nil
}

// ListOrganizations lists all organizations for the user.
func (s *OrganizationService) ListOrganizations(ctx context.Context, userID string) ([]orgdb.Organization, error) {
	s.logger.Infof("Listing organizations for user_id=%s", userID)
	ownerID, err := uuid.Parse(userID)
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}
	orgs, err := s.repo.ListOrganizationsByOwner(ctx, ownerID)
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}
	return orgs, nil
}

// ListMyOrganizations lists all organizations where the user is a member.
func (s *OrganizationService) ListMyOrganizations(ctx context.Context, userID string) ([]iam_db.ListAccessibleOrganizationsRow, error) {
	s.logger.Infof("Listing organizations for user_id=%s", userID)
	orgs, err := s.iamService.ListAccessibleOrganizations(ctx, userID)
	if err != nil {
		return nil, err
	}
	return orgs, nil
}

// ListMyOrganizationsEnhanced lists all organizations where the user is a member
// using enhanced RBAC with hierarchical inheritance and group membership support.
func (s *OrganizationService) ListMyOrganizationsEnhanced(ctx context.Context, userID string) ([]iam_db.ListAccessibleOrganizationsRow, error) {
	s.logger.Infof("Listing organizations with enhanced RBAC for user_id=%s", userID)

	enhancedOrgs, err := s.iamService.ListAccessibleOrganizationsEnhanced(ctx, userID)
	if err != nil {
		return nil, err
	}
	return enhancedOrgs, nil
}

// GetOrganization gets a specific organization by ID for the user.
func (s *OrganizationService) GetOrganization(ctx context.Context, userID, orgID string) (*orgdb.Organization, error) {
	s.logger.Infof("Getting organization org_id=%s for user_id=%s", orgID, userID)
	id, err := uuid.Parse(orgID)
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}
	org, err := s.repo.GetOrganizationByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, appErrors.ErrNotFound
		}
		return nil, appErrors.ErrInternalServer
	}
	return &org, nil
}

// GetOrganizationByName gets a specific organization by name for the user.
func (s *OrganizationService) GetOrganizationByName(ctx context.Context, orgName string) (*orgdb.Organization, error) {

	org, err := s.repo.GetOrganizationByName(ctx, orgName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appErrors.ErrOrganizationNotFound
		}
		return nil, appErrors.ErrInternalServer
	}
	return &org, nil

}

// UpdateOrganization updates an organization by ID for the user.
func (s *OrganizationService) UpdateOrganization(ctx context.Context, userID, orgID string, req UpdateOrganizationRequest) (*orgdb.Organization, error) {
	s.logger.Infof("Updating organization org_id=%s for user_id=%s", orgID, userID)
	id, err := uuid.Parse(orgID)
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}
	params := orgdb.UpdateOrganizationParams{
		ID:   id,
		Name: req.Name,
	}
	org, err := s.repo.UpdateOrganization(ctx, params)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, appErrors.ErrNotFound
		}
		return nil, appErrors.ErrInternalServer
	}
	return &org, nil
}

// DeleteOrganization deletes an organization by ID for the user.
func (s *OrganizationService) DeleteOrganization(ctx context.Context, orgID string) error {
	s.logger.Infof("Deleting organization org_id=%s ", orgID)
	orgId, err := uuid.Parse(orgID)
	if err != nil {
		return err
	}

	err = s.repo.DeleteOrganization(ctx, orgId)
	if err != nil {
		if err == sql.ErrNoRows {
			return appErrors.ErrOrganizationNotFound
		}
		if appErrors.IsViolatingForeignKeyConstraints(err) {
			return appErrors.ErrForeignKeyViolation
		}

		return err
	}
	params := iam.DeleteRoleBindingRequest{
		ResourceID:   orgId,
		ResourceType: "organization",
	}
	err = s.iamService.DeleteRoleBinding(ctx, params)
	if err != nil {
		return err
	}
	resource := fmt.Sprintf("/organizations/%s", orgID)
	err = s.policyEnforcer.RemoveResource(resource)
	if err != nil {
		s.logger.Errorf("Failed to remove secure permissions assigned to organization %s: %v", orgID, err)
		return err
	}
	return nil
}

// ListOrganizationRoleBindings retrieves all role bindings for an organization with resolved names.
func (s *OrganizationService) ListOrganizationRoleBindings(ctx context.Context, orgID string) ([]iam_db.ListOrganizationRoleBindingsRow, error) {
	s.logger.WithFields(logrus.Fields{
		"orgID": orgID,
	}).Info("Listing organization role bindings")

	bindings, err := s.iamService.ListOrganizationRoleBindings(ctx, orgID)
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
