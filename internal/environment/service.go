package environment

import (
	"context"
	"database/sql"

	environmentdb "github.com/Gkemhcs/kavach-backend/internal/environment/gen"

	appErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// EnvironmentService provides business logic for environments.
// Encapsulates all environment-related operations and validation.
type EnvironmentService struct {
	repo   environmentdb.Querier
	logger *logrus.Logger
}

// NewEnvironmentService creates a new EnvironmentService.
// Used to inject dependencies and enable testability.
func NewEnvironmentService(repo environmentdb.Querier, logger *logrus.Logger) *EnvironmentService {
	return &EnvironmentService{repo, logger}
}

// CreateEnvironment creates a new environment under a secret group.
// Adds the creator as the owner member of the environment.
func (s *EnvironmentService) CreateEnvironment(ctx context.Context, req CreateEnvironmentRequest) (*environmentdb.Environment, error) {
	s.logger.Infof("Creating environment for group_id=%s org_id=%s user_id=%s", req.SecretGroup, req.Organization, req.UserId)
	s.logger.Info(req.Name)
	groupUUID, err := uuid.Parse(req.SecretGroup)
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}

	params := environmentdb.CreateEnvironmentParams{
		Name:          req.Name,
		SecretGroupID: groupUUID,
		Description:   utils.DerefString(req.Description),
	}
	env, err := s.repo.CreateEnvironment(ctx, params)
	if appErrors.IsUniqueViolation(err) {
		return nil, appErrors.ErrDuplicateEnvironment
	}
	if appErrors.IsCheckConstraintViolation(err) {
		return nil, appErrors.ErrEnvironmenNameNotAllowed
	}
	if err != nil {
		s.logger.Print(err)
		return nil, appErrors.ErrInternalServer
	}

	addEnvironmentMemberParams := environmentdb.AddEnvironmentMemberParams{
		EnvironmentID: env.ID,
		UserID:        uuid.MustParse(req.UserId),
		Role:          environmentdb.RoleTypeOwner,
	}
	err = s.repo.AddEnvironmentMember(ctx, addEnvironmentMemberParams)
	if err != nil {

		return nil, err
	}
	return &env, nil
}

// ListEnvironments lists all environments under a secret group.
func (s *EnvironmentService) ListEnvironments(ctx context.Context, userID, orgID, groupID string) ([]environmentdb.Environment, error) {
	s.logger.Infof("Listing environments for group_id=%s org_id=%s user_id=%s", groupID, orgID, userID)
	groupUUID, err := uuid.Parse(groupID)
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}
	envs, err := s.repo.ListEnvironmentsBySecretGroup(ctx, groupUUID)
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}
	return envs, nil
}

// ListMyEnvironments lists all environments where the user is a member.
func (s *EnvironmentService) ListMyEnvironments(ctx context.Context, userID, groupID, orgID string) ([]environmentdb.ListEnvironmentsWithMemberRow, error) {
	s.logger.Infof("Listing environments for user_id=%s", userID)
	params := environmentdb.ListEnvironmentsWithMemberParams{
		UserID:         uuid.MustParse(userID),
		SecretGroupID:  uuid.MustParse(groupID),
		OrganizationID: uuid.MustParse(orgID),
	}
	envs, err := s.repo.ListEnvironmentsWithMember(ctx, params)
	if err != nil {
		return nil, err
	}
	return envs, nil
}

// GetEnvironment gets a specific environment by ID under a secret group.
func (s *EnvironmentService) GetEnvironment(ctx context.Context, userID, orgID, groupID, envID string) (*environmentdb.Environment, error) {
	s.logger.Infof("Getting environment env_id=%s for group_id=%s org_id=%s user_id=%s", envID, groupID, orgID, userID)
	envUUID, err := uuid.Parse(envID)
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}
	env, err := s.repo.GetEnvironmentByID(ctx, envUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, appErrors.ErrEnvironmentNotFound
		}
		return nil, appErrors.ErrInternalServer
	}
	return &env, nil
}

// GetEnvironmentByName gets a specific environment by environment Name under a secret group.
func (s *EnvironmentService) GetEnvironmentByName(ctx context.Context, environmentName, groupID string) (*environmentdb.Environment, error) {
	params := environmentdb.GetEnvironmentByNameParams{
		Name:          environmentName,
		SecretGroupID: uuid.MustParse(groupID),
	}
	environment, err := s.repo.GetEnvironmentByName(ctx, params)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, appErrors.ErrEnvironmentNotFound
		}
		return nil, appErrors.ErrInternalServer
	}
	return &environment, nil
}

// UpdateEnvironment updates an environment by ID under a secret group.
func (s *EnvironmentService) UpdateEnvironment(ctx context.Context, userID, orgID, groupID, envID string, req UpdateEnvironmentRequest) (*environmentdb.Environment, error) {
	s.logger.Infof("Updating environment env_id=%s for group_id=%s org_id=%s user_id=%s", envID, groupID, orgID, userID)
	envUUID, err := uuid.Parse(envID)
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}
	params := environmentdb.UpdateEnvironmentParams{
		ID:   envUUID,
		Name: req.Name,
	}
	env, err := s.repo.UpdateEnvironment(ctx, params)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, appErrors.ErrNotFound
		}
		return nil, appErrors.ErrInternalServer
	}
	return &env, nil
}

// DeleteEnvironment deletes an environment by ID under a secret group.
func (s *EnvironmentService) DeleteEnvironment(ctx context.Context, userID, orgID, groupID, envID string) error {
	s.logger.Infof("Deleting environment env_id=%s for group_id=%s org_id=%s user_id=%s", envID, groupID, orgID, userID)
	envUUID, err := uuid.Parse(envID)
	if err != nil {
		return appErrors.ErrInternalServer
	}
	err = s.repo.DeleteEnvironment(ctx, envUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			return appErrors.ErrNotFound
		}
		return appErrors.ErrInternalServer
	}
	return nil
}
