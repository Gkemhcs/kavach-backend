# 🔐 Authorization System (Kavach)

A production-grade, centralized RBAC authorization system using Casbin with PostgreSQL adapter for the Kavach backend.

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP Request  │───▶│   Gin Middleware│───▶│  Casbin Enforcer│
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │   Resolver      │    │ PostgreSQL DB   │
                       │ (Extract params)│    │ (Policy Storage)│
                       └─────────────────┘    └─────────────────┘
```

## 📁 File Structure

```
authz/
├── model.conf          # Casbin RBAC model configuration
├── actions.go          # Authorization action constants
├── context.go          # Gin context helpers
├── resolver.go         # Request parameter extraction
├── enforcer.go         # Casbin enforcer wrapper
├── middleware.go       # Gin authorization middleware
├── service.go          # High-level authorization operations
├── integration.go      # System initialization and setup
└── README.md          # This documentation
```

## 🚀 Quick Start

### 1. Initialize the Authorization System

```go
package main

import (
    "database/sql"
    "github.com/Gkemhcs/kavach-backend/internal/authz"
    "github.com/sirupsen/logrus"
)

func main() {
    // Initialize database connection
    db, err := sql.Open("postgres", "your-database-url")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Initialize logger
    logger := logrus.New()

    // Create authorization system
    authSystem, err := authz.NewSystem(db, nil, logger)
    if err != nil {
        log.Fatal(err)
    }

    // Setup Gin router
    router := gin.Default()
    
    // Apply authorization middleware
    authSystem.SetupRoutes(router)

    // Start server
    router.Run(":8080")
}
```

### 2. Grant Roles

```go
// Grant admin role to user on organization
err := authSystem.GrantRole(
    "user-uuid",           // userID
    "",                    // groupID (empty for user)
    "admin",               // role
    "organization",        // resourceType
    "org-uuid",           // resourceID
    "org-uuid",           // organizationID
)

// Grant viewer role to group on secret group
err := authSystem.GrantRole(
    "",                    // userID (empty for group)
    "group-uuid",          // groupID
    "viewer",              // role
    "secret_group",        // resourceType
    "secret-group-uuid",   // resourceID
    "org-uuid",           // organizationID
)
```

### 3. Check Permissions

```go
// Check if user can read organization
allowed, err := authSystem.CheckPermission(
    "user-uuid",
    "/organizations/org-uuid",
    "read",
)
```

## 🎯 Features

### ✅ **Production-Grade**
- **PostgreSQL adapter** for atomic operations
- **Concurrent-safe** with proper locking
- **Structured logging** with detailed audit trails
- **Error handling** with graceful degradation

### ✅ **RBAC with Inheritance**
- **Role-based access control** with viewer/editor/admin roles
- **Hierarchical inheritance** (org → secret groups → environments)
- **User and group support** with automatic role inheritance

### ✅ **Centralized Enforcement**
- **Single middleware** for all API routes
- **No scattered logic** in individual handlers
- **Consistent authorization** across all endpoints

### ✅ **Special Route Support**
- **By-name routes** accessible with viewer+ access on any parent/child
- **Organization creation** allowed for all authenticated users
- **Configurable exclusions** for public endpoints

## 🔧 Configuration

### Casbin Model (`model.conf`)

```ini
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && keyMatch2(r.obj, p.obj) && r.act == p.act
```

### Database Schema

The system uses your existing `role_bindings` table:

```sql
CREATE TABLE role_bindings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users(id) ON DELETE CASCADE,
    group_id        UUID REFERENCES user_groups(id) ON DELETE CASCADE,
    role            user_role NOT NULL,
    resource_type   resource_type NOT NULL,
    resource_id     UUID NOT NULL,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    secret_group_id UUID REFERENCES secret_groups(id) ON DELETE CASCADE,
    environment_id  UUID REFERENCES environments(id) ON DELETE CASCADE,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    
    CONSTRAINT chk_subject_exclusive CHECK (
        (user_id IS NOT NULL AND group_id IS NULL) OR 
        (user_id IS NULL AND group_id IS NOT NULL)
    ),
    
    CONSTRAINT unique_user_role_per_resource UNIQUE (user_id, resource_type, resource_id) 
        WHERE user_id IS NOT NULL,
    CONSTRAINT unique_group_role_per_resource UNIQUE (group_id, resource_type, resource_id) 
        WHERE group_id IS NOT NULL
);
```

## 🎭 Roles and Permissions

| Role   | Read | Write | Delete | Grant |
|--------|------|-------|--------|-------|
| Viewer | ✅   | ❌    | ❌     | ❌    |
| Editor | ✅   | ✅    | ❌     | ❌    |
| Admin  | ✅   | ✅    | ✅     | ✅    |

## 🛣️ Resource Hierarchy

```
Organization
├── Secret Groups
│   └── Environments
└── User Groups
```

**Inheritance Rules:**
- Organization roles apply to all child resources
- Secret group roles apply to all environments
- Child roles can override parent permissions

## 🔄 Integration with Existing Code

### 1. Update IAM Handler

```go
// In your IAM handler, replace direct database operations with authz service
func (h *IamHandler) GrantRoleBinding(c *gin.Context) {
    // ... existing validation ...
    
    // Use authz system instead of direct DB operations
    err := h.authSystem.GrantRole(
        req.UserName,
        req.GroupName,
        req.Role,
        req.ResourceType,
        req.ResourceID.String(),
        req.OrganizationID.String(),
    )
    
    // ... rest of handler ...
}
```

### 2. Remove Existing Authorization Logic

Remove any authorization checks from your existing handlers - the middleware handles everything centrally.

### 3. Update Main Router

```go
// In your main.go or server setup
func setupRoutes(router *gin.Engine, authSystem *authz.System) {
    // Apply authorization middleware to API routes
    authSystem.SetupRoutes(router)
    
    // Register your existing handlers (no changes needed)
    // The middleware will handle all authorization
}
```

## 🧪 Testing

### Unit Tests

```go
func TestAuthorization(t *testing.T) {
    // Setup test database
    db := setupTestDB()
    defer db.Close()
    
    // Create auth system
    authSystem, err := authz.NewSystem(db, nil, logrus.New())
    require.NoError(t, err)
    
    // Test role granting
    err = authSystem.GrantRole("user1", "", "admin", "organization", "org1", "org1")
    require.NoError(t, err)
    
    // Test permission checking
    allowed, err := authSystem.CheckPermission("user1", "/organizations/org1", "read")
    require.NoError(t, err)
    assert.True(t, allowed)
}
```

### Integration Tests

```go
func TestAuthorizationMiddleware(t *testing.T) {
    // Setup test server with auth middleware
    router := gin.New()
    authSystem := setupTestAuthSystem()
    authSystem.SetupRoutes(router)
    
    // Test authorized request
    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/api/v1/organizations/org1", nil)
    req.Header.Set("Authorization", "Bearer valid-token")
    router.ServeHTTP(w, req)
    
    assert.Equal(t, http.StatusOK, w.Code)
}
```

## 📊 Monitoring and Observability

### Logging

The system provides structured logging for all authorization decisions:

```json
{
  "level": "info",
  "msg": "Access granted",
  "subject": "user:123",
  "object": "/organizations/456/secret-groups/789",
  "action": "read",
  "method": "GET",
  "path": "/api/v1/organizations/456/secret-groups/789"
}
```

### Metrics

Consider adding metrics for:
- Authorization decision counts (allow/deny)
- Policy evaluation latency
- Role binding operations
- Cache hit rates

## 🔒 Security Considerations

1. **Principle of Least Privilege**: Start with viewer roles, grant additional permissions as needed
2. **Regular Audits**: Review role bindings periodically
3. **Role Minimization**: Avoid granting admin roles unless necessary
4. **Monitoring**: Monitor authorization failures for potential security issues

## 🚨 Troubleshooting

### Common Issues

1. **Permission Denied**: Check if user has the required role on the resource or any parent
2. **Policy Sync Failures**: Verify database connectivity and role_bindings table structure
3. **Performance Issues**: Consider adding Redis caching for frequently accessed policies

### Debug Mode

Enable debug logging to see detailed authorization decisions:

```go
logger.SetLevel(logrus.DebugLevel)
```

## 📚 API Reference

### System Methods

- `NewSystem(db, config, logger)` - Initialize authorization system
- `SetupRoutes(router)` - Apply middleware to Gin router
- `GrantRole(userID, groupID, role, resourceType, resourceID, orgID)` - Grant role
- `RevokeRole(userID, groupID, role, resourceType, resourceID)` - Revoke role
- `CheckPermission(userID, resourcePath, action)` - Check permission
- `SyncPolicies()` - Sync policies from database

### Middleware Methods

- `Authorize()` - Main authorization middleware
- `AuthorizeByRole(role)` - Role-specific middleware
- `AuthorizeOrganizationCreation()` - Special middleware for org creation

## 🤝 Contributing

When modifying the authorization system:

1. **Test thoroughly** - Authorization is critical for security
2. **Update documentation** - Keep this README current
3. **Follow logging standards** - Use structured logging with relevant fields
4. **Consider performance** - Authorization runs on every request

---

**🎯 This authorization system provides production-grade, centralized RBAC with hierarchical inheritance, supporting both users and groups, with atomic database operations and comprehensive logging.** 