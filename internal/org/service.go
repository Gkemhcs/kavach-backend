package org

import (
	"context"
	"database/sql"
	"errors"

	appErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	orgdb "github.com/Gkemhcs/kavach-backend/internal/org/gen"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// OrganizationService provides business logic for organizations.
// Encapsulates all organization-related operations and validation.
type OrganizationService struct {
	repo   orgdb.Querier
	logger *logrus.Logger
}

// NewOrganizationService creates a new OrganizationService.
// Used to inject dependencies and enable testability.
func NewOrganizationService(repo orgdb.Querier, logger *logrus.Logger) *OrganizationService {
	return &OrganizationService{repo, logger}

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
		return nil, appErrors.ErrInternalServer
	}
	addOrgMemberParams := orgdb.AddOrgMemberParams{
		OrgID:  org.ID,
		UserID: ownerID,
		Role:   orgdb.RoleTypeOwner,
	}
	err = s.repo.AddOrgMember(ctx, addOrgMemberParams)
	if err != nil {
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
func (s *OrganizationService) ListMyOrganizations(ctx context.Context, userID string) ([]orgdb.ListOrganizationsWithMemberRow, error) {
	s.logger.Infof("Listing organizations for user_id=%s", userID)
	orgs, err := s.repo.ListOrganizationsWithMember(ctx, uuid.MustParse(userID))
	if err != nil {
		return nil, err
	}
	return orgs, nil
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
func (s *OrganizationService) GetOrganizationByName(ctx context.Context, orgName string, userId string) (*orgdb.Organization, error) {

	userID := uuid.MustParse(userId)
	params := orgdb.GetOrganizationByNameParams{
		OwnerID: userID,
		Name:    orgName,
	}
	org, err := s.repo.GetOrganizationByName(ctx, params)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, appErrors.ErrOrganizationNotFound
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
func (s *OrganizationService) DeleteOrganization(ctx context.Context, orgID uuid.UUID) error {
	s.logger.Infof("Deleting organization org_id=%s ", orgID)

	err := s.repo.DeleteOrganization(ctx, orgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return appErrors.ErrOrganizationNotFound
		}
		return appErrors.ErrInternalServer
	}
	return nil
}
