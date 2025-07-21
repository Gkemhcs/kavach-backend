package org

import (
	"context"
	"database/sql"

	appErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	orgdb "github.com/Gkemhcs/kavach-backend/internal/org/gen"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// OrganizationService provides business logic for organizations.
type OrganizationService struct {
	repo   orgdb.Querier
	logger *logrus.Logger
}

// NewOrganizationService creates a new OrganizationService.
func NewOrganizationService(repo orgdb.Querier, logger *logrus.Logger) *OrganizationService {
	return &OrganizationService{repo, logger}

}

func derefString(s string) sql.NullString {
	if s != "" {
		return sql.NullString{
			String: s,
			Valid:  true,
		}
	} else {
		return sql.NullString{
			Valid: false,
		}
	}

}

// CreateOrganization creates a new organization for the user.
func (s *OrganizationService) CreateOrganization(ctx context.Context, req CreateOrganizationRequest) (*orgdb.Organization, error) {
	s.logger.Infof("Creating organization for user_id=%s", req.UserID)
	ownerID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}
	params := orgdb.CreateOrganizationParams{
		Name:        req.Name,
		Description: derefString(req.Description),
		OwnerID:     ownerID,
	}
	org, err := s.repo.CreateOrganization(ctx, params)
	if appErrors.IsUniqueViolation(err) {
		return nil, appErrors.ErrDuplicateOrganization
	}
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}
	addOrgMemberParams:=orgdb.AddOrgMemberParams{
			OrgID: org.ID,
			UserID: ownerID,
			Role: orgdb.RoleTypeOwner,
	}
	err=s.repo.AddOrgMember(ctx,addOrgMemberParams)
	if err!=nil{
		return nil,err
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


func (s *OrganizationService) ListMyOrganizations(ctx context.Context ,userID string)([]orgdb.ListOrganizationsWithMemberRow,error){
	s.logger.Infof("Listing organizations for user_id=%s", userID)
	orgs,err:=s.repo.ListOrganizationsWithMember(ctx,uuid.MustParse(userID))
	if err!=nil{
		return nil,err
	}
	return orgs,nil 
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
func (s *OrganizationService) DeleteOrganization(ctx context.Context, userID, orgID string) error {
	s.logger.Infof("Deleting organization org_id=%s for user_id=%s", orgID, userID)
	id, err := uuid.Parse(orgID)
	if err != nil {
		return appErrors.ErrInternalServer
	}
	err = s.repo.DeleteOrganization(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return appErrors.ErrNotFound
		}
		return appErrors.ErrInternalServer
	}
	return nil
}
