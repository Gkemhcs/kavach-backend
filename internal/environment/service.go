package environment

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Gkemhcs/kavach-backend/internal/authz"
	environmentdb "github.com/Gkemhcs/kavach-backend/internal/environment/gen"
	"github.com/Gkemhcs/kavach-backend/internal/iam"
	iam_db "github.com/Gkemhcs/kavach-backend/internal/iam/gen"

	appErrors "github.com/Gkemhcs/kavach-backend/internal/errors"
	"github.com/Gkemhcs/kavach-backend/internal/utils"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// EnvironmentService provides business logic for environments.
// Encapsulates all environment-related operations and validation.
type EnvironmentService struct {
	repo           environmentdb.Querier
	logger         *logrus.Logger
	iamService     iam.IamService
	policyEnforcer authz.Enforcer
}

// NewEnvironmentService creates a new EnvironmentService.
// Used to inject dependencies and enable testability.
func NewEnvironmentService(repo environmentdb.Querier, logger *logrus.Logger, iamService iam.IamService, policyEnforcer authz.Enforcer) *EnvironmentService {
	return &EnvironmentService{repo, logger, iamService, policyEnforcer}
}

// CreateEnvironment creates a new environment under a secret group.
// Adds the creator as the owner member of the environment.
func (s *EnvironmentService) CreateEnvironment(ctx context.Context, req CreateEnvironmentRequest) (*environmentdb.Environment, error) {
	s.logger.Infof("Creating environment for group_id=%s org_id=%s user_id=%s", req.SecretGroup, req.Organization, req.UserId)
	s.logger.Info(req.Name)

	// Validate UUIDs
	groupUUID, err := uuid.Parse(req.SecretGroup)
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}

	userUUID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}

	orgUUID, err := uuid.Parse(req.Organization)
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
		return nil, appErrors.ErrEnvironmentNameNotAllowed
	}
	if err != nil {
		s.logger.Print(err)
		return nil, appErrors.ErrInternalServer
	}

	createBindingParams := iam.CreateRoleBindingRequest{
		UserID:         userUUID,
		Role:           "owner",
		ResourceType:   "environment",
		ResourceID:     env.ID,
		OrganizationID: orgUUID,
		SecretGroupID: uuid.NullUUID{
			UUID:  groupUUID,
			Valid: true,
		},
		EnvironmentID: uuid.NullUUID{
			UUID:  env.ID,
			Valid: true,
		},
	}
	_, err = s.iamService.CreateRoleBinding(ctx, createBindingParams)

	if err != nil {

		return nil, err

	}
	parentResourcePath := fmt.Sprintf("/organizations/%s/secret-groups/%s", req.Organization, req.SecretGroup)
	childResourcePath := fmt.Sprintf("/organizations/%s/secret-groups/%s/environments/%s", req.Organization, req.SecretGroup, env.ID)
	err = s.policyEnforcer.AddResourceOwner(req.UserId, childResourcePath)
	if err != nil {
		s.logger.Errorf("Failed to grant secure permissions for envrionment %s: %v", env.ID, err)
		return nil, err
	}
	err = s.policyEnforcer.AddResourceHierarchy(parentResourcePath, childResourcePath)
	if err != nil {
		s.logger.Errorf("unable to add environment to resource hierarchy")
		return nil, err
	}
	return &env, nil
}

// ListEnvironments lists all environments under a secret group.
func (s *EnvironmentService) ListEnvironments(ctx context.Context, userID, orgID, groupID string) ([]environmentdb.Environment, error) {
	s.logger.Infof("Listing environments for group_id=%s org_id=%s user_id=%s", groupID, orgID, userID)

	// Validate UUIDs
	groupUUID, err := uuid.Parse(groupID)
	if err != nil {
		return nil, appErrors.ErrInternalServer
	}

	_, err = uuid.Parse(orgID)
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
func (s *EnvironmentService) ListMyEnvironments(ctx context.Context, userID, groupID, orgID string) ([]iam_db.ListAccessibleEnvironmentsRow, error) {
	s.logger.Infof("Listing environments for user_id=%s", userID)

	envs, err := s.iamService.ListAccessibleEnvironments(ctx, userID, orgID, groupID)

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
func (s *EnvironmentService) GetEnvironmentByName(ctx context.Context, environmentName, groupID string) (*environmentdb.GetEnvironmentByNameRow, error) {
	groupId, err := uuid.Parse(groupID)
	if err != nil {
		return nil, err
	}
	s.logger.Info(groupID)
	params := environmentdb.GetEnvironmentByNameParams{
		SecretGroupID: groupId,
		Name:          environmentName,
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
			return nil, appErrors.ErrEnvironmentNotFound
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
			return appErrors.ErrEnvironmentNotFound
		}
		if appErrors.IsViolatingForeignKeyConstraints(err) {
			return appErrors.ErrForeignKeyViolation
		}
		return appErrors.ErrInternalServer
	}
	params := iam.DeleteRoleBindingRequest{
		ResourceType: "environment",
		ResourceID:   envUUID,
	}
	err = s.iamService.DeleteRoleBinding(ctx, params)
	if err != nil {

		return err
	}
	parentResourcePath := fmt.Sprintf("/organizations/%s/secret-groups/%s", orgID, groupID)
	childResourcePath := fmt.Sprintf("/organizations/%s/secret-groups/%s/environments/%s", orgID, groupID, envID)
	err = s.policyEnforcer.RemoveResource(childResourcePath)
	if err != nil {
		s.logger.Errorf("Failed to remove secure permissions assigned to environment %s: %v", envID, err)
	}
	err = s.policyEnforcer.RemoveResourceHierarchy(parentResourcePath, childResourcePath)
	if err != nil {
		s.logger.Errorf("failed to remove from resource heirarchy")
		return err
	}

	return nil
}

// ListEnvironmentRoleBindings retrieves all role bindings for an environment with resolved names.
func (s *EnvironmentService) ListEnvironmentRoleBindings(ctx context.Context, orgID, groupID, envID string) ([]iam_db.ListEnvironmentRoleBindingsRow, error) {
	s.logger.WithFields(logrus.Fields{
		"orgID":   orgID,
		"groupID": groupID,
		"envID":   envID,
	}).Info("Listing environment role bindings")

	// Parse the environment ID
	envUUID, err := uuid.Parse(envID)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"envID": envID,
			"error": err.Error(),
		}).Error("Failed to parse environment ID")
		return nil, err
	}

	bindings, err := s.iamService.ListEnvironmentRoleBindings(ctx, envUUID)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"orgID":   orgID,
			"groupID": groupID,
			"envID":   envID,
			"error":   err.Error(),
		}).Error("Failed to list environment role bindings")
		return nil, err
	}

	s.logger.WithFields(logrus.Fields{
		"orgID":        orgID,
		"groupID":      groupID,
		"envID":        envID,
		"bindingCount": len(bindings),
	}).Info("Successfully retrieved environment role bindings")

	return bindings, nil
}
