package groups

import "time"

// CreateUserGroupRequest represents the request payload for creating a new user group.
// Contains the group name, optional description, and organization context.
type CreateUserGroupRequest struct {
	GroupName      string `json:"group_name"`       // Name of the user group (required, must be unique within org)
	Description    string `json:"description"`      // Optional description of the group's purpose
	OrganizationID string `json:"org_id,omitempty"` // Organization ID where the group will be created
	UserID         string `json:"user_id,omitempty"`
}

// UserGroupResponseData represents the response payload for user group operations.
// Contains complete group information including metadata like creation and update timestamps.
type UserGroupResponseData struct {
	ID             string    `json:"id"`              // Unique identifier for the user group
	Name           string    `json:"name"`            // Name of the user group
	OrganizationID string    `json:"organization_id"` // Organization that owns this group
	Description    string    `json:"description"`     // Optional description of the group
	CreatedAt      time.Time `json:"created_at"`      // Timestamp when the group was created
	UpdatedAt      time.Time `json:"updated_at"`      // Timestamp when the group was last modified
}

// ListGroupsByOrgRow represents a user group in the context of listing groups within an organization.
// Contains minimal required fields for display purposes in group listings.
type ListGroupsByOrgRow struct {
	ID          string    `json:"id"`          // Unique identifier for the user group
	Name        string    `json:"name"`        // Name of the user group
	Description string    `json:"description"` // Optional description of the group
	CreatedAt   time.Time `json:"created_at"`  // Timestamp when the group was created
}

// AddMemberRequest represents the request payload for adding a user to a user group.
// Contains the username to add and the target group context.
type AddMemberRequest struct {
	UserGroupID string `json:"user_group_id,omitempty"` // ID of the user group (set from URL param)
	UserName    string `json:"user_name"`               // GitHub username of the user to add to the group
}

// RemoveMemberRequest represents the request payload for removing a user from a user group.
// Contains the username to remove and the target group context.
type RemoveMemberRequest struct {
	UserGroupID string `json:"user_group_id,omitempty"` // ID of the user group (set from URL param)
	UserName    string `json:"user_name"`               // GitHub username of the user to remove from the group
}

// ListGroupMembersRow represents a group member in the context of listing group members.
// Contains user details and membership information for display purposes.
type ListGroupMembersRow struct {
	ID        string    `json:"id"`         // Unique identifier for the user
	Name      string    `json:"name"`       // Display name of the user
	Email     string    `json:"email"`      // Email address of the user
	CreatedAt time.Time `json:"created_at"` // Timestamp when the user joined the group
}
