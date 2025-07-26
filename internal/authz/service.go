package authz

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Service provides high-level authorization operations
type Service struct {
	enforcer *Enforcer
	db       *sql.DB
	logger   *logrus.Logger
}

// NewService creates a new authorization service
func NewService(enforcer *Enforcer, db *sql.DB, logger *logrus.Logger) *Service {
	return &Service{
		enforcer: enforcer,
		db:       db,
		logger:   logger,
	}
}

// RoleBinding represents a role binding from the database
type RoleBinding struct {
	ID             uuid.UUID  `json:"id"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	GroupID        *uuid.UUID `json:"group_id,omitempty"`
	Role           string     `json:"role"`
	ResourceType   string     `json:"resource_type"`
	ResourceID     uuid.UUID  `json:"resource_id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	SecretGroupID  *uuid.UUID `json:"secret_group_id,omitempty"`
	EnvironmentID  *uuid.UUID `json:"environment_id,omitempty"`
}

// GrantRoleBinding grants a role to a user or group
func (s *Service) GrantRoleBinding(ctx context.Context, binding RoleBinding) error {
	// Insert into database
	err := s.insertRoleBinding(ctx, binding)
	if err != nil {
		return fmt.Errorf("failed to insert role binding: %w", err)
	}

	// Add to Casbin enforcer
	err = s.addRoleBindingToEnforcer(binding)
	if err != nil {
		// Try to rollback database insertion
		s.logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Failed to add role binding to enforcer, attempting rollback")

		// Note: In production, you might want to implement proper rollback
		return fmt.Errorf("failed to add role binding to enforcer: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"userID":       binding.UserID,
		"groupID":      binding.GroupID,
		"role":         binding.Role,
		"resourceType": binding.ResourceType,
		"resourceID":   binding.ResourceID,
	}).Info("Role binding granted successfully")

	return nil
}

// RevokeRoleBinding revokes a role from a user or group
func (s *Service) RevokeRoleBinding(ctx context.Context, binding RoleBinding) error {
	// Remove from database
	err := s.deleteRoleBinding(ctx, binding)
	if err != nil {
		return fmt.Errorf("failed to delete role binding: %w", err)
	}

	// Remove from Casbin enforcer
	err = s.removeRoleBindingFromEnforcer(binding)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Failed to remove role binding from enforcer")

		return fmt.Errorf("failed to remove role binding from enforcer: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"userID":       binding.UserID,
		"groupID":      binding.GroupID,
		"role":         binding.Role,
		"resourceType": binding.ResourceType,
		"resourceID":   binding.ResourceID,
	}).Info("Role binding revoked successfully")

	return nil
}

// SyncRoleBindings syncs all role bindings from database to Casbin enforcer
func (s *Service) SyncRoleBindings(ctx context.Context) error {
	s.logger.Info("Starting role bindings sync")

	// Get all role bindings from database
	bindings, err := s.getAllRoleBindings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get role bindings: %w", err)
	}

	// Clear existing policies
	err = s.enforcer.LoadPolicy()
	if err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	// Add each role binding to enforcer
	for _, binding := range bindings {
		err = s.addRoleBindingToEnforcer(binding)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"binding": binding,
				"error":   err.Error(),
			}).Error("Failed to add role binding to enforcer during sync")
			// Continue with other bindings
		}
	}

	s.logger.WithFields(logrus.Fields{
		"totalBindings": len(bindings),
	}).Info("Role bindings sync completed")

	return nil
}

// insertRoleBinding inserts a role binding into the database
func (s *Service) insertRoleBinding(ctx context.Context, binding RoleBinding) error {
	query := `
		INSERT INTO role_bindings (
			user_id, group_id, role, resource_type, resource_id, 
			organization_id, secret_group_id, environment_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := s.db.ExecContext(ctx, query,
		binding.UserID,
		binding.GroupID,
		binding.Role,
		binding.ResourceType,
		binding.ResourceID,
		binding.OrganizationID,
		binding.SecretGroupID,
		binding.EnvironmentID,
	)

	return err
}

// deleteRoleBinding deletes a role binding from the database
func (s *Service) deleteRoleBinding(ctx context.Context, binding RoleBinding) error {
	var query string
	var args []interface{}

	if binding.UserID != nil {
		query = `
			DELETE FROM role_bindings 
			WHERE user_id = $1 AND role = $2 AND resource_type = $3 AND resource_id = $4
		`
		args = []interface{}{binding.UserID, binding.Role, binding.ResourceType, binding.ResourceID}
	} else if binding.GroupID != nil {
		query = `
			DELETE FROM role_bindings 
			WHERE group_id = $1 AND role = $2 AND resource_type = $3 AND resource_id = $4
		`
		args = []interface{}{binding.GroupID, binding.Role, binding.ResourceType, binding.ResourceID}
	} else {
		return fmt.Errorf("either user_id or group_id must be provided")
	}

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// getAllRoleBindings retrieves all role bindings from the database
func (s *Service) getAllRoleBindings(ctx context.Context) ([]RoleBinding, error) {
	query := `
		SELECT id, user_id, group_id, role, resource_type, resource_id, 
		       organization_id, secret_group_id, environment_id
		FROM role_bindings
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bindings []RoleBinding
	for rows.Next() {
		var binding RoleBinding
		err := rows.Scan(
			&binding.ID,
			&binding.UserID,
			&binding.GroupID,
			&binding.Role,
			&binding.ResourceType,
			&binding.ResourceID,
			&binding.OrganizationID,
			&binding.SecretGroupID,
			&binding.EnvironmentID,
		)
		if err != nil {
			return nil, err
		}
		bindings = append(bindings, binding)
	}

	return bindings, nil
}

// addRoleBindingToEnforcer adds a role binding to the Casbin enforcer
func (s *Service) addRoleBindingToEnforcer(binding RoleBinding) error {
	// Determine subject (user or group)
	var subject string
	if binding.UserID != nil {
		subject = fmt.Sprintf("user:%s", binding.UserID.String())
	} else if binding.GroupID != nil {
		subject = fmt.Sprintf("group:%s", binding.GroupID.String())
	} else {
		return fmt.Errorf("either user_id or group_id must be provided")
	}

	// Create role string
	role := s.createRoleString(binding)

	// Add role assignment (g rule)
	err := s.enforcer.AddRoleForUser(subject, role)
	if err != nil {
		return fmt.Errorf("failed to add role assignment: %w", err)
	}

	// Add policy rules (p rules) based on resource type and role
	err = s.addPolicyRules(binding, role)
	if err != nil {
		return fmt.Errorf("failed to add policy rules: %w", err)
	}

	return nil
}

// removeRoleBindingFromEnforcer removes a role binding from the Casbin enforcer
func (s *Service) removeRoleBindingFromEnforcer(binding RoleBinding) error {
	// Determine subject (user or group)
	var subject string
	if binding.UserID != nil {
		subject = fmt.Sprintf("user:%s", binding.UserID.String())
	} else if binding.GroupID != nil {
		subject = fmt.Sprintf("group:%s", binding.GroupID.String())
	} else {
		return fmt.Errorf("either user_id or group_id must be provided")
	}

	// Create role string
	role := s.createRoleString(binding)

	// Remove role assignment (g rule)
	err := s.enforcer.RemoveRoleForUser(subject, role)
	if err != nil {
		return fmt.Errorf("failed to remove role assignment: %w", err)
	}

	return nil
}

// createRoleString creates a role string in Casbin format
func (s *Service) createRoleString(binding RoleBinding) string {
	switch binding.ResourceType {
	case "organization":
		return fmt.Sprintf("org:%s:%s", binding.ResourceID.String(), binding.Role)
	case "secret_group":
		return fmt.Sprintf("secretgroup:%s:%s", binding.ResourceID.String(), binding.Role)
	case "environment":
		return fmt.Sprintf("environment:%s:%s", binding.ResourceID.String(), binding.Role)
	default:
		return fmt.Sprintf("%s:%s:%s", binding.ResourceType, binding.ResourceID.String(), binding.Role)
	}
}

// addPolicyRules adds policy rules for a role binding
func (s *Service) addPolicyRules(binding RoleBinding, role string) error {
	// Define permissions based on role
	permissions := s.getPermissionsForRole(binding.Role)

	// Add policies for each permission
	for _, permission := range permissions {
		object := s.createObjectString(binding)
		err := s.enforcer.AddPolicy(role, object, Action(permission))
		if err != nil {
			return fmt.Errorf("failed to add policy %s %s %s: %w", role, object, permission, err)
		}
	}

	return nil
}

// getPermissionsForRole returns the permissions for a given role
func (s *Service) getPermissionsForRole(role string) []string {
	switch strings.ToLower(role) {
	case "admin":
		return []string{ActionRead.String(), ActionWrite.String(), ActionDelete.String(), ActionGrant.String()}
	case "editor":
		return []string{ActionRead.String(), ActionWrite.String()}
	case "viewer":
		return []string{ActionRead.String()}
	default:
		return []string{ActionRead.String()}
	}
}

// createObjectString creates an object string in Casbin format
func (s *Service) createObjectString(binding RoleBinding) string {
	switch binding.ResourceType {
	case "organization":
		return fmt.Sprintf("/organizations/%s/*", binding.ResourceID.String())
	case "secret_group":
		return fmt.Sprintf("/organizations/%s/secret-groups/%s/*", binding.OrganizationID.String(), binding.ResourceID.String())
	case "environment":
		// For environments, we need to include the secret group path
		if binding.SecretGroupID != nil {
			return fmt.Sprintf("/organizations/%s/secret-groups/%s/environments/%s/*",
				binding.OrganizationID.String(), binding.SecretGroupID.String(), binding.ResourceID.String())
		}
		return fmt.Sprintf("/organizations/%s/secret-groups/*/environments/%s/*",
			binding.OrganizationID.String(), binding.ResourceID.String())
	default:
		return fmt.Sprintf("/%s/%s/*", binding.ResourceType, binding.ResourceID.String())
	}
}
