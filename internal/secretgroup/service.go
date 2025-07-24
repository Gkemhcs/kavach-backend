package secretgroup

import (
	"context"
	"database/sql"
	"errors"

	appErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	"github.com/Gkemhcs/kavach-backend/internal/iam"
	iam_db "github.com/Gkemhcs/kavach-backend/internal/iam/gen"
	secretgroupdb "github.com/Gkemhcs/kavach-backend/internal/secretgroup/gen"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// SecretGroupService provides business logic for secret groups.
// Encapsulates all secret group-related operations and validation.
type SecretGroupService struct {
	repo       secretgroupdb.Querier
	logger     *logrus.Logger
	iamService iam.IamService
}

// NewSecretGroupService creates a new SecretGroupService.
// Used to inject dependencies and enable testability.
func NewSecretGroupService(repo secretgroupdb.Querier, logger *logrus.Logger, iamService iam.IamService) *SecretGroupService {
	return &SecretGroupService{repo, logger, iamService}
}

// CreateSecretGroup creates a new secret group under an organization.
// Adds the creator as the owner member of the secret group.
func (s *SecretGroupService) CreateSecretGroup(ctx context.Context, req CreateSecretGroupRequest) (*secretgroupdb.SecretGroup, error) {
	s.logger.Infof("Creating secret group for org_id=%s user_id=%s", req.OrganizationID, req.UserID)
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}
	orgUUID, err := uuid.Parse(req.OrganizationID)
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}
	params := secretgroupdb.CreateSecretGroupParams{
		Name:           req.Name,
		OrganizationID: orgUUID,
		Description:    utils.DerefString(req.Description),
	}
	group, err := s.repo.CreateSecretGroup(ctx, params)
	if appErrors.IsUniqueViolation(err) {
		return nil, appErrors.ErrDuplicateSecretGroup
	}
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}

	createBindingRequest := iam.CreateRoleBindingRequest{
		UserID:         userID,
		Role:           "owner",
		ResourceType:   "secret_group",
		ResourceID:     group.ID,
		OrganizationID: orgUUID,
		SecretGroupID: uuid.NullUUID{
			UUID:  group.ID,
			Valid: true,
		},
		EnvironmentID: uuid.NullUUID{Valid: false},
	}
	_, err = s.iamService.CreateRoleBinding(ctx, createBindingRequest)

	if err != nil {
		return nil, err
	}
	return &group, nil
}

// ListSecretGroups lists all secret groups under an organization.
func (s *SecretGroupService) ListSecretGroups(ctx context.Context, userID, orgID string) ([]secretgroupdb.SecretGroup, error) {
	s.logger.Infof("Listing secret groups for org_id=%s user_id=%s", orgID, userID)
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}
	groups, err := s.repo.ListSecretGroupsByOrg(ctx, orgUUID)
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}
	return groups, nil
}

// ListMySecretGroups lists all secret groups where the user is a member.
func (s *SecretGroupService) ListMySecretGroups(ctx context.Context, orgId, userID string) ([]iam_db.ListAccessibleSecretGroupsRow, error) {
	s.logger.Infof("Listing secret groups for user_id=%s", userID)

	groups, err := s.iamService.ListAccessibleSecretGroups(ctx, userID, orgId)
	if err != nil {
		return nil, err
	}
	return groups, nil

}

// GetSecretGroup gets a specific secret group by ID under an organization.
func (s *SecretGroupService) GetSecretGroup(ctx context.Context, userID, orgID, groupID string) (*secretgroupdb.SecretGroup, error) {
	s.logger.Infof("Getting secret group group_id=%s for org_id=%s user_id=%s", groupID, orgID, userID)
	groupUUID, err := uuid.Parse(groupID)
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}
	group, err := s.repo.GetSecretGroupByID(ctx, groupUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, appErrors.ErrNotFound
		}
		return nil, appErrors.ErrInternalServer
	}
	return &group, nil
}

// GetSecretGroupByName gets a specific secret group by name under an organization.
func (s *SecretGroupService) GetSecretGroupByName(ctx context.Context, orgId, groupName string) (*secretgroupdb.SecretGroup, error) {
	params := secretgroupdb.GetSecretGroupByNameParams{
		Name:           groupName,
		OrganizationID: uuid.MustParse(orgId),
	}
	group, err := s.repo.GetSecretGroupByName(ctx, params)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, appErrors.ErrSecretGroupNotFound
	}
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// UpdateSecretGroup updates a secret group by ID under an organization.
func (s *SecretGroupService) UpdateSecretGroup(ctx context.Context, userID, orgID, groupID string, req UpdateSecretGroupRequest) (*secretgroupdb.SecretGroup, error) {
	s.logger.Infof("Updating secret group group_id=%s for org_id=%s user_id=%s", groupID, orgID, userID)
	groupUUID, err := uuid.Parse(groupID)
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}
	params := secretgroupdb.UpdateSecretGroupParams{
		ID:   groupUUID,
		Name: req.Name,
	}
	group, err := s.repo.UpdateSecretGroup(ctx, params)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, appErrors.ErrNotFound
		}
		return nil, appErrors.ErrInternalServer
	}
	return &group, nil
}

// DeleteSecretGroup deletes a secret group by ID under an organization.
func (s *SecretGroupService) DeleteSecretGroup(ctx context.Context, userID, orgID, groupID string) error {
	s.logger.Infof("Deleting secret group group_id=%s for org_id=%s user_id=%s", groupID, orgID, userID)
	groupUUID, err := uuid.Parse(groupID)
	if err != nil {
		return appErrors.ErrInternalServer
	}
	err = s.repo.DeleteSecretGroup(ctx, groupUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			return appErrors.ErrNotFound
		}
		return appErrors.ErrInternalServer
	}
	return nil
}
