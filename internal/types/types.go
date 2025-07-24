package types

import (
	"context"

	userdb "github.com/Gkemhcs/kavach-backend/internal/auth/gen"
	groupsdb "github.com/Gkemhcs/kavach-backend/internal/groups/gen"
	orgdb "github.com/Gkemhcs/kavach-backend/internal/org/gen"
)

// OrganizationGetter defines an interface for retrieving organizations by name and user ID.
// Used for dependency injection and testability across modules.
type OrganizationGetter interface {
	// GetOrganizationByName fetches an organization by name and user ID.
	GetOrganizationByName(ctx context.Context, orgName string, userId string) (*orgdb.Organization, error)
}

type UserResolver interface {
	GetUserInfoByGithubUserName(ctx context.Context, userName string) (*userdb.User, error)
}

type UserGroupResolver interface {
	GetUserGroupByName(ctx context.Context, userGroupName, orgID string) (*groupsdb.UserGroup, error)
}
