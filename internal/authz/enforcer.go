package authz

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"github.com/casbin/casbin/v2/util"
	"github.com/sirupsen/logrus"
)

// Enforcer manages the Casbin authorization enforcer
type Enforcer struct {
	enforcer *casbin.Enforcer
	logger   *logrus.Logger
	mu       sync.RWMutex
}

// NewEnforcer creates a new authorization enforcer with PostgreSQL adapter
func NewEnforcer(adapter persist.Adapter, logger *logrus.Logger) (*Enforcer, error) {
	// Load the RBAC model
	m, err := model.NewModelFromFile("internal/authz/model.conf")
	if err != nil {
		return nil, fmt.Errorf("failed to load Casbin model: %w", err)
	}

	// Create the enforcer
	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create Casbin enforcer: %w", err)
	}

	// Enable auto-save to database
	enforcer.EnableAutoSave(true)

	// Enable logging
	enforcer.EnableLog(true)

	// Set up function for keyMatch2
	enforcer.AddFunction("keyMatch2", util.KeyMatch2Func)

	logger.Info("Authorization enforcer initialized successfully")

	return &Enforcer{
		enforcer: enforcer,
		logger:   logger,
	}, nil
}

// Enforce checks if the subject has permission to perform the action on the object
func (e *Enforcer) Enforce(subject, object string, action Action) (bool, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	allowed, err := e.enforcer.Enforce(subject, object, action.String())
	if err != nil {
		e.logger.WithFields(logrus.Fields{
			"subject": subject,
			"object":  object,
			"action":  action,
			"error":   err.Error(),
		}).Error("Authorization enforcement failed")
		return false, fmt.Errorf("authorization enforcement failed: %w", err)
	}

	e.logger.WithFields(logrus.Fields{
		"subject": subject,
		"object":  object,
		"action":  action,
		"allowed": allowed,
	}).Debug("Authorization decision made")

	return allowed, nil
}

// EnforceWithGroupCheck checks if the user has permission, including group membership
func (e *Enforcer) EnforceWithGroupCheck(userID, object string, action Action, db *sql.DB) (bool, error) {
	// First check direct user permissions
	subject := fmt.Sprintf("user:%s", userID)
	allowed, err := e.Enforce(subject, object, action)
	if err != nil {
		return false, err
	}
	if allowed {
		return true, nil
	}

	// If no direct permission, check group memberships
	allowed, err = e.checkGroupMembershipPermissions(userID, object, action, db)
	if err != nil {
		return false, err
	}

	return allowed, nil
}

// checkGroupMembershipPermissions checks if user has access through group membership
func (e *Enforcer) checkGroupMembershipPermissions(userID, object string, action Action, db *sql.DB) (bool, error) {
	// Query to find all groups the user is a member of
	query := `
		SELECT DISTINCT ug.id, ug.name 
		FROM user_groups ug
		JOIN user_group_members ugm ON ug.id = ugm.user_group_id
		JOIN users u ON ugm.user_id = u.id
		WHERE u.id = $1
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return false, fmt.Errorf("failed to query user group memberships: %w", err)
	}
	defer rows.Close()

	// Check each group for permissions
	for rows.Next() {
		var groupID, groupName string
		if err := rows.Scan(&groupID, &groupName); err != nil {
			continue
		}

		// Check if this group has the required permission
		groupSubject := fmt.Sprintf("group:%s", groupID)
		allowed, err := e.Enforce(groupSubject, object, action)
		if err != nil {
			e.logger.WithFields(logrus.Fields{
				"userID":    userID,
				"groupID":   groupID,
				"groupName": groupName,
				"object":    object,
				"action":    action,
				"error":     err.Error(),
			}).Error("Failed to check group permissions")
			continue
		}

		if allowed {
			e.logger.WithFields(logrus.Fields{
				"userID":    userID,
				"groupID":   groupID,
				"groupName": groupName,
				"object":    object,
				"action":    action,
			}).Debug("User has access through group membership")
			return true, nil
		}
	}

	return false, nil
}

// AddPolicy adds a policy rule to the enforcer
func (e *Enforcer) AddPolicy(subject, object string, action Action) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	added, err := e.enforcer.AddPolicy(subject, object, action.String())
	if err != nil {
		e.logger.WithFields(logrus.Fields{
			"subject": subject,
			"object":  object,
			"action":  action,
			"error":   err.Error(),
		}).Error("Failed to add policy")
		return fmt.Errorf("failed to add policy: %w", err)
	}

	if added {
		e.logger.WithFields(logrus.Fields{
			"subject": subject,
			"object":  object,
			"action":  action,
		}).Info("Policy added successfully")
	} else {
		e.logger.WithFields(logrus.Fields{
			"subject": subject,
			"object":  object,
			"action":  action,
		}).Debug("Policy already exists")
	}

	return nil
}

// RemovePolicy removes a policy rule from the enforcer
func (e *Enforcer) RemovePolicy(subject, object string, action Action) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	removed, err := e.enforcer.RemovePolicy(subject, object, action.String())
	if err != nil {
		e.logger.WithFields(logrus.Fields{
			"subject": subject,
			"object":  object,
			"action":  action,
			"error":   err.Error(),
		}).Error("Failed to remove policy")
		return fmt.Errorf("failed to remove policy: %w", err)
	}

	if removed {
		e.logger.WithFields(logrus.Fields{
			"subject": subject,
			"object":  object,
			"action":  action,
		}).Info("Policy removed successfully")
	} else {
		e.logger.WithFields(logrus.Fields{
			"subject": subject,
			"object":  object,
			"action":  action,
		}).Debug("Policy not found for removal")
	}

	return nil
}

// AddRoleForUser adds a role for a user
func (e *Enforcer) AddRoleForUser(user, role string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	added, err := e.enforcer.AddRoleForUser(user, role)
	if err != nil {
		e.logger.WithFields(logrus.Fields{
			"user":  user,
			"role":  role,
			"error": err.Error(),
		}).Error("Failed to add role for user")
		return fmt.Errorf("failed to add role for user: %w", err)
	}

	if added {
		e.logger.WithFields(logrus.Fields{
			"user": user,
			"role": role,
		}).Info("Role added for user successfully")
	} else {
		e.logger.WithFields(logrus.Fields{
			"user": user,
			"role": role,
		}).Debug("Role already exists for user")
	}

	return nil
}

// RemoveRoleForUser removes a role from a user
func (e *Enforcer) RemoveRoleForUser(user, role string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Casbin doesn't have a direct RemoveRoleForUser method, so we use RemoveFilteredPolicy
	// to remove the role assignment (g rule)
	removed, err := e.enforcer.RemoveFilteredPolicy(0, user, role)
	if err != nil {
		e.logger.WithFields(logrus.Fields{
			"user":  user,
			"role":  role,
			"error": err.Error(),
		}).Error("Failed to remove role from user")
		return fmt.Errorf("failed to remove role from user: %w", err)
	}

	if removed {
		e.logger.WithFields(logrus.Fields{
			"user": user,
			"role": role,
		}).Info("Role removed from user successfully")
	} else {
		e.logger.WithFields(logrus.Fields{
			"user": user,
			"role": role,
		}).Debug("Role not found for user")
	}

	return nil
}

// GetRolesForUser returns all roles for a user
func (e *Enforcer) GetRolesForUser(user string) ([]string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	roles, err := e.enforcer.GetRolesForUser(user)
	if err != nil {
		e.logger.WithFields(logrus.Fields{
			"user":  user,
			"error": err.Error(),
		}).Error("Failed to get roles for user")
		return nil, fmt.Errorf("failed to get roles for user: %w", err)
	}

	return roles, nil
}

// GetUsersForRole returns all users for a role
func (e *Enforcer) GetUsersForRole(role string) ([]string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	users, err := e.enforcer.GetUsersForRole(role)
	if err != nil {
		e.logger.WithFields(logrus.Fields{
			"role":  role,
			"error": err.Error(),
		}).Error("Failed to get users for role")
		return nil, fmt.Errorf("failed to get users for role: %w", err)
	}

	return users, nil
}

// LoadPolicy reloads the policy from the database
func (e *Enforcer) LoadPolicy() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	err := e.enforcer.LoadPolicy()
	if err != nil {
		e.logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Failed to load policy")
		return fmt.Errorf("failed to load policy: %w", err)
	}

	e.logger.Info("Policy loaded successfully")
	return nil
}

// SavePolicy saves the policy to the database
func (e *Enforcer) SavePolicy() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	err := e.enforcer.SavePolicy()
	if err != nil {
		e.logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Failed to save policy")
		return fmt.Errorf("failed to save policy: %w", err)
	}

	e.logger.Info("Policy saved successfully")
	return nil
}

// GetPolicy returns all policy rules
func (e *Enforcer) GetPolicy() [][]string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	policy, _ := e.enforcer.GetPolicy()
	return policy
}

// GetFilteredPolicy returns filtered policy rules
func (e *Enforcer) GetFilteredPolicy(fieldIndex int, fieldValues ...string) [][]string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	policy, _ := e.enforcer.GetFilteredPolicy(fieldIndex, fieldValues...)
	return policy
}
