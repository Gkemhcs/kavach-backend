package authz

import (
	"github.com/casbin/casbin/v2"
	"github.com/sirupsen/logrus"
)

// LoadDefaultPolicies adds role → action groupings using Casbin's grouping policies
// This creates the foundation for what actions each role can perform
func LoadDefaultPolicies(logger *logrus.Logger, enforcer *casbin.Enforcer) {
	logger.Info("✅ Setting up role-action groupings for RBAC")

	// Define role-action mappings
	roleActions := map[string][]string{
		"owner":  {"read", "create", "grant", "revoke", "delete", "update","view_provider_config","manage_provider_config"},
		"admin":  {"read", "create", "grant", "revoke", "update","view_provider_config","manage_provider_config"},
		"editor": {"read", "create", "update","sync","view_provider_config"},
		"viewer": {"read"},
	}

	// Create grouping policies: g, role, action (g is for role-action mapping)
	for role, actions := range roleActions {
		for _, action := range actions {
			ok, err := enforcer.AddNamedGroupingPolicy("g", role, action)
			if err != nil {
				logger.Errorf("❌ Failed to add role-action grouping [%s, %s]: %v", role, action, err)
				continue
			}
			if ok {
				logger.Infof("✅ Added role-action grouping: %s can %s", role, action)
			} else {
				logger.Infof("ℹ️ Role-action grouping already exists: %s can %s", role, action)
			}
		}
	}

	logger.Info("🎯 Role-action groupings configured successfully")
}
