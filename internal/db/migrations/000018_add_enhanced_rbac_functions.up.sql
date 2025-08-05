-- Migration: Add Enhanced RBAC Functions
-- Description: Adds functions for hierarchical role-based access control

-- Role hierarchy function for determining highest role
CREATE OR REPLACE FUNCTION get_highest_role(roles user_role[]) RETURNS user_role AS $$
BEGIN
    -- Role hierarchy: owner > admin > editor > viewer
    IF 'owner' = ANY(roles) THEN
        RETURN 'owner';
    ELSIF 'admin' = ANY(roles) THEN
        RETURN 'admin';
    ELSIF 'editor' = ANY(roles) THEN
        RETURN 'editor';
    ELSIF 'viewer' = ANY(roles) THEN
        RETURN 'viewer';
    ELSE
        RETURN NULL;
    END IF;
END;
$$ LANGUAGE plpgsql; 