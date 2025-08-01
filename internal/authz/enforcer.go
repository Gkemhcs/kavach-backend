package authz

import (
	"fmt"
	"strings"

	"github.com/casbin/casbin/v2"
	sqladapter "github.com/memwey/casbin-sqlx-adapter"
	"github.com/sirupsen/logrus"
)

// Enforcer wraps the Casbin enforcer with additional RBAC functionality
type Enforcer struct {
	enforcer *casbin.Enforcer
	logger   *logrus.Logger
}

// NewEnforcer creates a new Enforcer instance
func NewEnforcer(logger *logrus.Logger, cfg AdapterConfig) (*Enforcer, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", cfg.DB_USER, cfg.DB_PASSWORD, cfg.DB_HOST, cfg.DB_PORT, cfg.DB_NAME)
	adapter := sqladapter.NewAdapter("postgres", dsn)

	enforcer, err := casbin.NewEnforcer(cfg.MODEL_FILE_PATH, adapter)
	enforcer.EnableAutoSave(true)
	if err != nil {
		return nil, err
	}
	logger.Info("Successfully initialized the Authz Enforcer Service")

	// Load default policies
	LoadDefaultPolicies(logger, enforcer)

	return &Enforcer{
		enforcer: enforcer,
		logger:   logger,
	}, nil
}

// GetEnforcer returns the underlying Casbin enforcer
func (e *Enforcer) GetEnforcer() *casbin.Enforcer {
	return e.enforcer
}

// ============================================================================
// RESOURCE MANAGEMENT METHODS
// ============================================================================

// AddResourceHierarchy adds a resource hierarchy mapping (parent -> child)
func (e *Enforcer) AddResourceHierarchy(parentResource, childResource string) error {
	e.logger.Infof("🔗 [HIERARCHY] Adding resource hierarchy: %s -> %s", parentResource, childResource)

	ok, err := e.enforcer.AddNamedGroupingPolicy("g2", parentResource, childResource)
	if err != nil {
		e.logger.Errorf("❌ [HIERARCHY] Failed to add resource hierarchy [%s, %s]: %v", parentResource, childResource, err)
		return err
	}
	if ok {
		e.logger.Infof("✅ [HIERARCHY] Added resource hierarchy: %s -> %s", parentResource, childResource)

		// Save the policy to ensure it's persisted
		if err := e.enforcer.SavePolicy(); err != nil {
			e.logger.Errorf("❌ [HIERARCHY] Failed to save policy after adding hierarchy: %v", err)
			return err
		}
		e.logger.Infof("✅ [HIERARCHY] Policy saved successfully")

		// Verify the hierarchy was added correctly
		g2Policies, _ := e.enforcer.GetNamedGroupingPolicy("g2")
		e.logger.Infof("📋 [HIERARCHY] Current g2 policies: %v", g2Policies)
	} else {
		e.logger.Infof("ℹ️ [HIERARCHY] Resource hierarchy [%s, %s] already exists", parentResource, childResource)
	}
	return nil
}

// RemoveResourceHierarchy removes a resource hierarchy mapping
func (e *Enforcer) RemoveResourceHierarchy(parentResource, childResource string) error {
	ok, err := e.enforcer.RemoveNamedGroupingPolicy("g2", parentResource, childResource)
	if err != nil {
		e.logger.Errorf("Failed to remove resource hierarchy [%s, %s]: %v", parentResource, childResource, err)
		return err
	}
	if ok {
		e.logger.Infof("✅ Removed resource hierarchy: %s -> %s", parentResource, childResource)
	} else {
		e.logger.Infof("ℹ️ Resource hierarchy [%s, %s] doesn't exist", parentResource, childResource)
	}
	return nil
}

// AddResourceOwner adds owner permission for a resource
func (e *Enforcer) AddResourceOwner(userID, resource string) error {
	userSubject := fmt.Sprintf("user:%s", userID)
	ok, err := e.enforcer.AddPolicy(userSubject, "owner", resource)
	if err != nil {
		e.logger.Errorf("Failed to add owner permission for %s on %s: %v", userSubject, resource, err)
		return err
	}
	if ok {
		e.logger.Infof("✅ Added owner permission: %s on %s", userSubject, resource)
	} else {
		e.logger.Infof("ℹ️ Owner permission already exists: %s on %s", userSubject, resource)
	}
	return nil
}

// RemoveResource removes all policies related to a resource and its children
func (e *Enforcer) RemoveResource(resource string) error {
	// Get all policies that match this resource or its children
	policies, _ := e.enforcer.GetFilteredPolicy(2, resource)

	// Also get policies that start with this resource path
	allPolicies, _ := e.enforcer.GetPolicy()
	for _, policy := range allPolicies {
		if len(policy) >= 3 && (policy[2] == resource || e.isChildResource(policy[2], resource)) {
			policies = append(policies, policy)
		}
	}

	// Remove all matching policies
	for _, policy := range policies {
		if len(policy) >= 3 {
			ok, err := e.enforcer.RemovePolicy(policy[0], policy[1], policy[2])
			if err != nil {
				e.logger.Errorf("Failed to remove policy %v: %v", policy, err)
			} else if ok {
				e.logger.Infof("✅ Removed policy: %v", policy)
			}
		}
	}

	e.logger.Infof("✅ Deleted resource %s and all its policies", resource)
	return nil
}

// ============================================================================
// USER-GROUP MANAGEMENT METHODS
// ============================================================================

// AddUserToGroup adds a user to a user group
func (e *Enforcer) AddUserToGroup(userID, groupID string) error {
	userSubject := fmt.Sprintf("user:%s", userID)
	groupName := fmt.Sprintf("group:%s", groupID)
	ok, err := e.enforcer.AddGroupingPolicy(userSubject, groupName)
	if err != nil {
		e.logger.Errorf("Failed to add user %s to group %s: %v", userSubject, groupName, err)
		return err
	}
	if ok {
		e.logger.Infof("✅ Added user %s to group %s", userSubject, groupName)
	} else {
		e.logger.Infof("ℹ️ User %s is already in group %s", userSubject, groupName)
	}
	return nil
}

// RemoveUserFromGroup removes a user from a user group
func (e *Enforcer) RemoveUserFromGroup(userID, groupID string) error {
	userSubject := fmt.Sprintf("user:%s", userID)
	groupName := fmt.Sprintf("group:%s", groupID)
	ok, err := e.enforcer.RemoveGroupingPolicy(userSubject, groupName)
	if err != nil {
		e.logger.Errorf("Failed to remove user %s from group %s: %v", userSubject, groupName, err)
		return err
	}
	if ok {
		e.logger.Infof("✅ Removed user %s from group %s", userSubject, groupName)
	} else {
		e.logger.Infof("ℹ️ User %s is not in group %s", userSubject, groupName)
	}
	return nil
}

// DeleteUserGroup removes all users from the group and deletes all group permissions
func (e *Enforcer) DeleteUserGroup(groupID string) error {
	groupName := fmt.Sprintf("group:%s", groupID)

	// Get all users in the group
	users, err := e.enforcer.GetUsersForRole(groupName)
	if err != nil {
		e.logger.Errorf("Failed to get users for group %s: %v", groupName, err)
		return err
	}

	// Remove all users from the group
	for _, user := range users {
		ok, err := e.enforcer.RemoveGroupingPolicy(user, groupName)
		if err != nil {
			e.logger.Errorf("Failed to remove user %s from group %s: %v", user, groupName, err)
		} else if ok {
			e.logger.Infof("✅ Removed user %s from group %s", user, groupName)
		}
	}

	// Remove all policies for this group
	policies, _ := e.enforcer.GetFilteredPolicy(0, groupName)
	for _, policy := range policies {
		if len(policy) >= 3 {
			ok, err := e.enforcer.RemovePolicy(policy[0], policy[1], policy[2])
			if err != nil {
				e.logger.Errorf("Failed to remove policy %v: %v", policy, err)
			} else if ok {
				e.logger.Infof("✅ Removed policy: %v", policy)
			}
		}
	}

	e.logger.Infof("✅ Deleted user group %s and all its mappings", groupName)
	return nil
}

// ============================================================================
// ROLE MANAGEMENT METHODS
// ============================================================================

// GrantRole grants a role to a user or group on a specific resource
func (e *Enforcer) GrantRole(subject, role, resource string) error {
	ok, err := e.enforcer.AddPolicy(subject, role, resource)
	if err != nil {
		e.logger.Errorf("Failed to grant role %s to %s on %s: %v", role, subject, resource, err)
		return err
	}
	if ok {
		e.logger.Infof("✅ Granted role %s to %s on %s", role, subject, resource)
	} else {
		e.logger.Infof("ℹ️ Role %s already granted to %s on %s", role, subject, resource)
	}
	return nil
}

// GrantRoleWithPermissions grants a role to a user on a specific resource
func (e *Enforcer) GrantRoleWithPermissions(userID, role, resource string) error {
	userSubject := fmt.Sprintf("user:%s", userID)
	ok, err := e.enforcer.AddPolicy(userSubject, role, resource)
	if err != nil {
		e.logger.Errorf("Failed to grant role %s to %s on %s: %v", role, userSubject, resource, err)
		return err
	}
	if ok {
		e.logger.Infof("✅ Granted role %s to %s on %s", role, userSubject, resource)
	} else {
		e.logger.Infof("ℹ️ Role %s already granted to %s on %s", role, userSubject, resource)
	}
	return nil
}

// RevokeRole revokes a role from a user or group on a specific resource
func (e *Enforcer) RevokeRole(subject, role, resource string) error {
	// First, remove the direct policy (p) that grants the role to the user on the resource
	ok, err := e.enforcer.RemovePolicy(subject, role, resource)
	if err != nil {
		e.logger.Errorf("Failed to remove direct policy %s %s %s: %v", subject, role, resource, err)
		return err
	}
	if ok {
		e.logger.Infof("✅ Removed direct policy: %s %s %s", subject, role, resource)
	} else {
		e.logger.Infof("ℹ️ Direct policy doesn't exist: %s %s %s", subject, role, resource)
	}

	// Remove user-to-role mapping (g) if it exists
	ok, err = e.enforcer.RemoveGroupingPolicy(subject, role)
	if err != nil {
		e.logger.Errorf("Failed to remove user-role mapping %s -> %s: %v", subject, role, err)
	} else if ok {
		e.logger.Infof("✅ Removed user-role mapping: %s -> %s", subject, role)
	} else {
		e.logger.Infof("ℹ️ User-role mapping doesn't exist: %s -> %s", subject, role)
	}

	e.logger.Infof("✅ Successfully revoked %s role from %s on %s", role, subject, resource)
	return nil
}

// RevokeRoleCascade revokes a role from a user on a resource and all its child resources
// This ensures that when access is revoked at a parent level, all child resources are also revoked
func (e *Enforcer) RevokeRoleCascade(subject, role, resource string) error {
	e.logger.Infof("🔄 [CASCADE] Starting cascading role revocation for %s %s on %s", subject, role, resource)

	// First, revoke the role on the target resource
	err := e.RevokeRole(subject, role, resource)
	if err != nil {
		e.logger.Errorf("❌ [CASCADE] Failed to revoke role on target resource: %v", err)
		return err
	}

	// Get all policies for this user
	allPolicies, err := e.enforcer.GetPolicy()
	if err != nil {
		e.logger.Errorf("❌ [CASCADE] Failed to get all policies: %v", err)
		return err
	}

	// Find and remove all policies for this user on child resources
	removedCount := 0
	for _, policy := range allPolicies {
		if len(policy) >= 3 && policy[0] == subject {
			policyResource := policy[2]

			// Check if this policy resource is a child of the target resource
			if e.isChildResource(policyResource, resource) {
				e.logger.Infof("🔄 [CASCADE] Found child resource policy: %s %s %s", policy[0], policy[1], policy[2])

				ok, err := e.enforcer.RemovePolicy(policy[0], policy[1], policy[2])
				if err != nil {
					e.logger.Errorf("❌ [CASCADE] Failed to remove child policy %v: %v", policy, err)
				} else if ok {
					e.logger.Infof("✅ [CASCADE] Removed child policy: %s %s %s", policy[0], policy[1], policy[2])
					removedCount++
				}
			}
		}
	}

	e.logger.Infof("✅ [CASCADE] Successfully revoked %s role from %s on %s and %d child resources", role, subject, resource, removedCount)
	return nil
}

// ============================================================================
// PERMISSION CHECKING METHODS
// ============================================================================

// CheckPermission checks if a user has permission to perform an action on a resource
func (e *Enforcer) CheckPermission(userID, action, resource string) (bool, error) {
	userSubject := fmt.Sprintf("user:%s", userID)

	e.logger.Infof("🔍 [CAS] Checking permission - User: %s, Action: %s, Resource: %s", userSubject, action, resource)

	// First, try direct permission check
	ok, err := e.enforcer.Enforce(userSubject, action, resource)
	if err != nil {
		e.logger.Errorf("❌ [CAS] Failed to check permission [%s, %s, %s]: %v", userSubject, action, resource, err)
		return false, err
	}

	if ok {
		e.logger.Infof("✅ [CAS] Permission GRANTED - %s can %s on %s", userSubject, action, resource)
		return true, nil
	}

	e.logger.Infof("❌ [CAS] Permission DENIED - %s cannot %s on %s", userSubject, action, resource)

	// Additional debugging: show what policies exist for this user and resource
	e.logger.Infof("🔍 [CAS] Debugging permission denial...")

	// Get all policies for this user
	policies, _ := e.enforcer.GetFilteredPolicy(0, userSubject)
	e.logger.Infof("📋 [CAS] All policies for user %s: %v", userSubject, policies)

	// Get all policies for this resource
	resourcePolicies, _ := e.enforcer.GetFilteredPolicy(2, resource)
	e.logger.Infof("📋 [CAS] All policies for resource %s: %v", resource, resourcePolicies)

	// Get user's roles
	roles, _ := e.enforcer.GetRolesForUser(userSubject)
	e.logger.Infof("📋 [CAS] User %s has roles: %v", userSubject, roles)

	// Get all grouping policies
	groupings, _ := e.enforcer.GetGroupingPolicy()
	e.logger.Infof("📋 [CAS] All grouping policies: %v", groupings)

	// Get all resource hierarchy policies (g2)
	g2Policies, _ := e.enforcer.GetNamedGroupingPolicy("g2")
	e.logger.Infof("📋 [CAS] All resource hierarchy policies (g2): %v", g2Policies)

	// Check for inherited permissions from parent resources
	e.logger.Infof("🔍 [CAS] Checking for inherited permissions from parent resources...")
	
	// Try to find parent resources and check permissions there
	parentResources := e.getParentResources(resource)
	for _, parentResource := range parentResources {
		e.logger.Infof("🔍 [CAS] Checking parent resource: %s", parentResource)
		
		// Check if user has permission on parent resource
		parentOk, parentErr := e.enforcer.Enforce(userSubject, action, parentResource)
		if parentErr != nil {
			e.logger.Errorf("❌ [CAS] Failed to check parent permission [%s, %s, %s]: %v", userSubject, action, parentResource, parentErr)
			continue
		}
		
		if parentOk {
			e.logger.Infof("✅ [CAS] Inherited permission GRANTED - %s can %s on parent %s", userSubject, action, parentResource)
			return true, nil
		} else {
			e.logger.Infof("❌ [CAS] No inherited permission on parent %s", parentResource)
		}
	}

	e.logger.Infof("❌ [CAS] No direct or inherited permissions found")
	
	// Add comprehensive debugging when permission is denied
	e.logger.Infof("🔍 [CAS] === COMPREHENSIVE DEBUGGING ===")
	e.DebugUserPermissions(userID)
	e.DebugResourcePermissions(resource)
	e.logger.Infof("🔍 [CAS] === END COMPREHENSIVE DEBUGGING ===")
	
	return false, nil
}

// getParentResources returns all possible parent resources for a given resource
func (e *Enforcer) getParentResources(resource string) []string {
	var parents []string
	
	// Split the resource path
	parts := strings.Split(resource, "/")
	
	// Build parent resources by removing the last part
	for i := len(parts) - 1; i >= 2; i-- { // Start from 2 to keep at least /organizations/{orgID}
		parent := strings.Join(parts[:i], "/")
		if parent != "" {
			parents = append(parents, parent)
		}
	}
	
	return parents
}

// CheckPermissionEx checks if a user has permission to perform an action on a resource using EnforceEx
// Returns additional information including the reason for the decision
func (e *Enforcer) CheckPermissionEx(userID, action, resource string) (bool, []string, error) {
	userSubject := fmt.Sprintf("user:%s", userID)

	e.logger.Infof("🔍 [CAS] Checking permission with EnforceEx - User: %s, Action: %s, Resource: %s", userSubject, action, resource)

	ok, explanations, err := e.enforcer.EnforceEx(userSubject, action, resource)
	if err != nil {
		e.logger.Errorf("❌ [CAS] Failed to check permission with EnforceEx [%s, %s, %s]: %v", userSubject, action, resource, err)
		return false, nil, err
	}

	if ok {
		e.logger.Infof("✅ [CAS] Permission GRANTED - %s can %s on %s", userSubject, action, resource)
		if len(explanations) > 0 {
			e.logger.Infof("📋 [CAS] Grant reason: %v", explanations)
		}
	} else {
		e.logger.Infof("❌ [CAS] Permission DENIED - %s cannot %s on %s", userSubject, action, resource)
		if len(explanations) > 0 {
			e.logger.Infof("📋 [CAS] Denial reason: %v", explanations)
		}

		// Additional debugging: show what policies exist for this user and resource
		e.logger.Infof("🔍 [CAS] Debugging permission denial...")

		// Get all policies for this user
		policies, _ := e.enforcer.GetFilteredPolicy(0, userSubject)
		e.logger.Infof("📋 [CAS] All policies for user %s: %v", userSubject, policies)

		// Get all policies for this resource
		resourcePolicies, _ := e.enforcer.GetFilteredPolicy(2, resource)
		e.logger.Infof("📋 [CAS] All policies for resource %s: %v", resource, resourcePolicies)

		// Get user's roles
		roles, _ := e.enforcer.GetRolesForUser(userSubject)
		e.logger.Infof("📋 [CAS] User %s has roles: %v", userSubject, roles)

		// Get all grouping policies
		groupings, _ := e.enforcer.GetGroupingPolicy()
		e.logger.Infof("📋 [CAS] All grouping policies: %v", groupings)
	}

	return ok, explanations, nil
}

// ============================================================================
// QUERY METHODS
// ============================================================================

// GetUserRoles gets all roles for a user (direct and through groups)
func (e *Enforcer) GetUserRoles(userID string) ([]string, error) {
	roles, err := e.enforcer.GetRolesForUser(userID)
	if err != nil {
		e.logger.Errorf("Failed to get roles for user %s: %v", userID, err)
		return nil, err
	}
	e.logger.Debugf("📋 %s has roles: %v", userID, roles)
	return roles, nil
}

// GetUserGroups gets all groups for a user
func (e *Enforcer) GetUserGroups(userID string) ([]string, error) {
	groups, err := e.enforcer.GetImplicitRolesForUser(userID)
	if err != nil {
		e.logger.Errorf("Failed to get groups for user %s: %v", userID, err)
		return nil, err
	}
	e.logger.Debugf("📋 %s is in groups: %v", userID, groups)
	return groups, nil
}

// GetUsersForGroup gets all users in a group
func (e *Enforcer) GetUsersForGroup(groupID string) ([]string, error) {
	groupName := fmt.Sprintf("group:%s", groupID)
	users, err := e.enforcer.GetUsersForRole(groupName)
	if err != nil {
		e.logger.Errorf("Failed to get users for group %s: %v", groupName, err)
		return nil, err
	}
	e.logger.Debugf("📋 Group %s has users: %v", groupName, users)
	return users, nil
}

// GetResourcePermissions gets all permissions for a specific resource
func (e *Enforcer) GetResourcePermissions(resource string) ([][]string, error) {
	policies, err := e.enforcer.GetFilteredPolicy(2, resource)
	if err != nil {
		return nil, err
	}
	e.logger.Debugf("📋 Resource %s has policies: %v", resource, policies)
	return policies, nil
}

// ============================================================================
// UTILITY METHODS
// ============================================================================

// isChildResource checks if childResource is a child of parentResource
func (e *Enforcer) isChildResource(childResource, parentResource string) bool {
	// Simple string prefix check - can be enhanced for more complex hierarchies
	return len(childResource) > len(parentResource) &&
		childResource[:len(parentResource)] == parentResource &&
		(len(childResource) == len(parentResource) || childResource[len(parentResource)] == '/')
}

// SavePolicy saves the current policy to the database
func (e *Enforcer) SavePolicy() error {
	err := e.enforcer.SavePolicy()
	if err != nil {
		e.logger.Errorf("Failed to save policy: %v", err)
		return err
	}
	e.logger.Info("✅ Policy saved successfully")
	return nil
}

// LoadPolicy loads the policy from the database
func (e *Enforcer) LoadPolicy() error {
	err := e.enforcer.LoadPolicy()
	if err != nil {
		e.logger.Errorf("Failed to load policy: %v", err)
		return err
	}
	e.logger.Info("✅ Policy loaded successfully")
	return nil
}

// getActionsForRole returns the actions that a role should have
func (e *Enforcer) getActionsForRole(role string) []string {
	switch role {
	case "owner":
		return []string{"read", "create", "grant", "revoke", "delete", "update"}
	case "admin":
		return []string{"read", "create", "grant", "revoke", "update"}
	case "editor":
		return []string{"read", "create", "update"}
	case "viewer":
		return []string{"read"}
	default:
		return []string{"read"}
	}
}

// ============================================================================
// DEBUGGING METHODS
// ============================================================================

// DebugPolicies prints all current policies for debugging purposes
func (e *Enforcer) DebugPolicies() {
	e.logger.Info("🔍 [DEBUG] === CASBIN POLICY DEBUG ===")

	// Get all policies (p)
	policies, _ := e.enforcer.GetPolicy()
	e.logger.Infof("📋 [DEBUG] All policies (p): %v", policies)

	// Get all grouping policies (g)
	groupings, _ := e.enforcer.GetGroupingPolicy()
	e.logger.Infof("📋 [DEBUG] All grouping policies (g): %v", groupings)

	// Get all resource hierarchy policies (g2)
	g2Policies, _ := e.enforcer.GetNamedGroupingPolicy("g2")
	e.logger.Infof("📋 [DEBUG] All resource hierarchy policies (g2): %v", g2Policies)

	e.logger.Info("🔍 [DEBUG] === END CASBIN POLICY DEBUG ===")
}

// DebugUserPermissions prints all permissions for a specific user
func (e *Enforcer) DebugUserPermissions(userID string) {
	userSubject := fmt.Sprintf("user:%s", userID)
	e.logger.Infof("🔍 [DEBUG] === USER PERMISSIONS DEBUG for %s ===", userSubject)

	// Get all policies for this user
	policies, _ := e.enforcer.GetFilteredPolicy(0, userSubject)
	e.logger.Infof("📋 [DEBUG] Direct policies for %s: %v", userSubject, policies)

	// Get user's roles
	roles, _ := e.enforcer.GetRolesForUser(userSubject)
	e.logger.Infof("📋 [DEBUG] Roles for %s: %v", userSubject, roles)

	// Get implicit roles (through groups)
	implicitRoles, _ := e.enforcer.GetImplicitRolesForUser(userSubject)
	e.logger.Infof("📋 [DEBUG] Implicit roles for %s: %v", userSubject, implicitRoles)

	e.logger.Infof("🔍 [DEBUG] === END USER PERMISSIONS DEBUG for %s ===", userSubject)
}

// DebugResourcePermissions prints all permissions for a specific resource
func (e *Enforcer) DebugResourcePermissions(resource string) {
	e.logger.Infof("🔍 [DEBUG] === RESOURCE PERMISSIONS DEBUG for %s ===", resource)

	// Get all policies for this resource
	policies, _ := e.enforcer.GetFilteredPolicy(2, resource)
	e.logger.Infof("📋 [DEBUG] Direct policies for %s: %v", resource, policies)

	// Get all policies that might affect this resource (including parent resources)
	allPolicies, _ := e.enforcer.GetPolicy()
	var relevantPolicies [][]string
	for _, policy := range allPolicies {
		if len(policy) >= 3 {
			policyResource := policy[2]
			if policyResource == resource || e.isChildResource(resource, policyResource) {
				relevantPolicies = append(relevantPolicies, policy)
			}
		}
	}
	e.logger.Infof("📋 [DEBUG] Relevant policies for %s: %v", resource, relevantPolicies)

	e.logger.Infof("🔍 [DEBUG] === END RESOURCE PERMISSIONS DEBUG for %s ===", resource)
}
