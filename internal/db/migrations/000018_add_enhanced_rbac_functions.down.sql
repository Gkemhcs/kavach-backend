-- Migration: Remove Enhanced RBAC Functions
-- Description: Removes functions for hierarchical role-based access control

-- Drop the role hierarchy function
DROP FUNCTION IF EXISTS get_highest_role(user_role[]); 