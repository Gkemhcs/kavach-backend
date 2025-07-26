package authz

import (
	"github.com/gin-gonic/gin"
)

// Context keys for authorization
const (
	SubjectKey = "authz_subject"
	OrgIDKey   = "authz_org_id"
)

// Subject represents the authenticated entity (user or group)
type Subject struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // "user" or "group"
	Username string `json:"username,omitempty"`
}

// SetSubject sets the authorization subject in the Gin context
func SetSubject(c *gin.Context, subject Subject) {
	c.Set(SubjectKey, subject)
}

// GetSubject retrieves the authorization subject from the Gin context
func GetSubject(c *gin.Context) (Subject, bool) {
	if subject, exists := c.Get(SubjectKey); exists {
		if s, ok := subject.(Subject); ok {
			return s, true
		}
	}
	return Subject{}, false
}

// SetOrgID sets the organization ID in the Gin context
func SetOrgID(c *gin.Context, orgID string) {
	c.Set(OrgIDKey, orgID)
}

// GetOrgID retrieves the organization ID from the Gin context
func GetOrgID(c *gin.Context) (string, bool) {
	if orgID, exists := c.Get(OrgIDKey); exists {
		if o, ok := orgID.(string); ok {
			return o, true
		}
	}
	return "", false
}

// GetUserID retrieves the user ID from the Gin context (for backward compatibility)
func GetUserID(c *gin.Context) string {
	return c.GetString("user_id")
}

// GetUserIDFromContext retrieves the user ID from the Gin context
func GetUserIDFromContext(c *gin.Context) (string, bool) {
	userID := c.GetString("user_id")
	return userID, userID != ""
}
