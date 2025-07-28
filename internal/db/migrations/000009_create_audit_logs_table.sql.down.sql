-- Drop audit logs table and related objects

-- Drop indexes
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_user_id;
DROP INDEX IF EXISTS idx_audit_logs_organization_id;
DROP INDEX IF EXISTS idx_audit_logs_action;
DROP INDEX IF EXISTS idx_audit_logs_resource_type;
DROP INDEX IF EXISTS idx_audit_logs_resource_id;
DROP INDEX IF EXISTS idx_audit_logs_severity;
DROP INDEX IF EXISTS idx_audit_logs_request_id;
DROP INDEX IF EXISTS idx_audit_logs_session_id;
DROP INDEX IF EXISTS idx_audit_logs_created_at_date;
DROP INDEX IF EXISTS idx_audit_logs_user_org_date;
DROP INDEX IF EXISTS idx_audit_logs_resource_date;
DROP INDEX IF EXISTS idx_audit_logs_severity_date;
DROP INDEX IF EXISTS idx_audit_logs_search;

-- Drop table
DROP TABLE IF EXISTS audit_logs;

-- Drop types
DROP TYPE IF EXISTS audit_severity;
DROP TYPE IF EXISTS audit_action; 