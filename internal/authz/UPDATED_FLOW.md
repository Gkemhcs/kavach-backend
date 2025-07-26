# 🔄 Updated Authorization Flow

## 🎯 **Key Changes Based on Your Clarification**

### **JWT Middleware Behavior**
- **Only injects `user_id`** into context (no group information)
- **No group authentication** at JWT level
- **Group membership** is checked during authorization enforcement

### **Authorization Flow for Alice's Case**

```
1. Alice (user_id: alice-123) makes request to create secret group in org "wiz"
2. JWT middleware injects user_id: "alice-123" into context
3. Authorization middleware extracts:
   - Subject: user:alice-123
   - Object: /organizations/wiz-org-id/secret-groups/*
   - Action: write (from POST method)
4. Enforcer checks:
   a) Direct user permissions: Does alice-123 have write access to org wiz?
   b) If no direct access: Check group memberships
   c) Query: Find all groups alice-123 is a member of
   d) Check each group for write permissions on org wiz
   e) Found: alice-123 is in "developers" group which has editor access to org wiz
   f) Result: ALLOW (access granted through group membership)
```

## 🛣️ **Route Handling**

### **1. IAM Routes (Special Handling)**
- **Paths**: `/permissions/grant`, `/permissions/revoke`
- **Method**: POST/DELETE
- **Resource Info**: Extracted from request body (not URL path)
- **Example Request Body**:
```json
{
  "user_name": "alice",
  "role": "editor",
  "resource_type": "organization",
  "resource_id": "wiz-org-id",
  "organization_id": "wiz-org-id"
}
```

### **2. Auth Routes (Excluded)**
- **Paths**: `/auth/login`, `/auth/register`, `/auth/refresh`, `/auth/logout`, `/auth/verify`
- **Authorization**: Skipped entirely
- **Reason**: Authentication endpoints don't need authorization

### **3. Regular Routes (URL Path Extraction)**
- **Paths**: `/organizations/{orgID}/secret-groups/{groupID}/environments/{envID}`
- **Resource IDs**: Extracted from URL path parameters
- **Organization ID**: Automatically set in context for inheritance

### **4. By-Name Routes (Special Access)**
- **Paths**: `/organizations/by-name/{orgName}`, `/secret-groups/by-name/{groupName}`
- **Access Rule**: User needs at least viewer access on any parent or child resource
- **Example**: Can access `/organizations/by-name/wiz` if has viewer+ on org wiz or any of its secret groups

## 🔧 **Implementation Details**

### **Group Membership Checking**

```go
// EnforceWithGroupCheck checks if the user has permission, including group membership
func (e *Enforcer) EnforceWithGroupCheck(userID, object string, action Action, db *sql.DB) (bool, error) {
    // 1. Check direct user permissions first
    subject := fmt.Sprintf("user:%s", userID)
    allowed, err := e.Enforce(subject, object, action)
    if err != nil {
        return false, err
    }
    if allowed {
        return true, nil // Direct access granted
    }

    // 2. If no direct access, check group memberships
    allowed, err = e.checkGroupMembershipPermissions(userID, object, action, db)
    if err != nil {
        return false, err
    }

    return allowed, nil
}
```

### **Group Membership Query**

```sql
SELECT DISTINCT ug.id, ug.name 
FROM user_groups ug
JOIN user_group_members ugm ON ug.id = ugm.user_group_id
JOIN users u ON ugm.user_id = u.id
WHERE u.id = $1
```

### **IAM Route Resource Extraction**

```go
// resolveIAMObject handles IAM routes by extracting resource info from request body
func (r *Resolver) resolveIAMObject(c *gin.Context, path string) (string, error) {
    // Parse request body to get resource information
    var requestBody map[string]interface{}
    if err := c.ShouldBindJSON(&requestBody); err != nil {
        return "", fmt.Errorf("failed to parse request body for IAM route: %w", err)
    }

    // Extract resource details from body
    resourceType := requestBody["resource_type"].(string)
    resourceID := requestBody["resource_id"].(string)
    orgID := requestBody["organization_id"].(string)

    // Build object path based on resource type
    switch resourceType {
    case "organization":
        return fmt.Sprintf("/organizations/%s/*", resourceID), nil
    case "secret_group":
        return fmt.Sprintf("/organizations/%s/secret-groups/%s/*", orgID, resourceID), nil
    // ... other resource types
    }
}
```

## 📊 **Authorization Decision Matrix**

| User Access | Group Access | Result | Log Message |
|-------------|--------------|--------|-------------|
| ✅ Direct | N/A | **ALLOW** | "Direct user access granted" |
| ❌ Direct | ✅ Group | **ALLOW** | "User has access through group membership" |
| ❌ Direct | ❌ Group | **DENY** | "Access denied - no direct or group permissions" |

## 🔍 **Logging Examples**

### **Direct Access Granted**
```json
{
  "level": "debug",
  "msg": "Access granted",
  "userID": "alice-123",
  "object": "/organizations/wiz-org-id/secret-groups/*",
  "action": "write",
  "method": "POST",
  "path": "/api/v1/organizations/wiz-org-id/secret-groups"
}
```

### **Group Access Granted**
```json
{
  "level": "debug",
  "msg": "User has access through group membership",
  "userID": "alice-123",
  "groupID": "developers-group-id",
  "groupName": "developers",
  "object": "/organizations/wiz-org-id/secret-groups/*",
  "action": "write"
}
```

### **Access Denied**
```json
{
  "level": "warn",
  "msg": "Access denied",
  "userID": "alice-123",
  "object": "/organizations/wiz-org-id/secret-groups/*",
  "action": "write",
  "method": "POST",
  "path": "/api/v1/organizations/wiz-org-id/secret-groups"
}
```

## 🎯 **Benefits of This Approach**

### ✅ **Flexible Group Management**
- Users can be added/removed from groups without changing authorization policies
- Group permissions automatically apply to all members
- No need to update individual user permissions

### ✅ **Performance Optimized**
- Direct user permissions checked first (fastest path)
- Group membership only queried when needed
- Database queries are optimized with proper indexes

### ✅ **Audit Trail**
- Clear logging of how access was granted (direct vs group)
- Group membership information included in logs
- Easy to trace authorization decisions

### ✅ **Scalable**
- Supports unlimited group memberships per user
- Hierarchical inheritance works with groups
- No performance degradation with large numbers of groups

## 🚀 **Integration Steps**

1. **Update your main application** to pass database connection to middleware
2. **Ensure your group membership tables** are properly indexed
3. **Test with your existing IAM endpoints** to verify body parsing works
4. **Verify group membership queries** work with your database schema
5. **Monitor logs** to ensure authorization decisions are being made correctly

This updated flow perfectly handles your use case where Alice gets access through group membership, while maintaining the flexibility to handle both direct user permissions and group-based permissions efficiently! 🎉 