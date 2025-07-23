package types

import (
	"context"

	orgdb "github.com/Gkemhcs/kavach-backend/internal/org/gen"
)

// OrganizationGetter defines an interface for retrieving organizations by name and user ID.
// Used for dependency injection and testability across modules.
type OrganizationGetter interface{
	// GetOrganizationByName fetches an organization by name and user ID.
	GetOrganizationByName(ctx context.Context,orgName string,userId string)(*orgdb.Organization, error)
}