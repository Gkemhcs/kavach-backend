-- Create audit logs table for comprehensive audit trail
-- This table stores all security-relevant events for compliance and monitoring

CREATE TYPE audit_severity AS ENUM ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL');
CREATE TYPE audit_action AS ENUM (
    'CREATE', 'READ', 'UPDATE', 'DELETE', 
    'GRANT', 'REVOKE', 'LOGIN', 'LOGOUT', 
    'FAILED_LOGIN', 'PASSWORD_CHANGE', 'PERMISSION_CHANGE',
    'SECRET_ACCESS', 'SECRET_MODIFY', 'ORGANIZATION_JOIN', 'ORGANIZATION_LEAVE'
);

CREATE TABLE audit_logs (
    -- Core identification
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL DEFAULT gen_random_uuid(),
    
    -- Timestamp and correlation
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    request_id TEXT,
    session_id TEXT,
    
    -- User and authentication context
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    user_email TEXT, -- Denormalized for easier querying
    authentication_method TEXT, -- 'oauth_github', 'jwt', etc.
    ip_address INET,
    user_agent TEXT,
    
    -- Action and resource context
    action audit_action NOT NULL,
    resource_type TEXT NOT NULL, -- 'organization', 'secret_group', 'environment', etc.
    resource_id UUID,
    resource_path TEXT NOT NULL, -- Full resource path like '/organizations/123/secret-groups/456'
    organization_id UUID REFERENCES organizations(id) ON DELETE SET NULL,
    
    -- State changes (for updates)
    previous_state JSONB, -- What the resource looked like before
    new_state JSONB,     -- What the resource looks like after
    changed_fields TEXT[], -- Array of field names that changed
    
    -- Security and risk assessment
    severity audit_severity NOT NULL DEFAULT 'LOW',
    risk_score INTEGER DEFAULT 0, -- 0-100 scale
    anomaly_flags TEXT[], -- Array of detected anomalies
    
    -- Additional context
    metadata JSONB, -- Flexible field for additional context
    description TEXT, -- Human-readable description of the event
    tags TEXT[], -- Array of tags for categorization
    
    -- Performance optimization
    created_at_date DATE GENERATED ALWAYS AS (created_at::date) STORED
);

-- Indexes for efficient querying
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_organization_id ON audit_logs(organization_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource_type ON audit_logs(resource_type);
CREATE INDEX idx_audit_logs_resource_id ON audit_logs(resource_id);
CREATE INDEX idx_audit_logs_severity ON audit_logs(severity);
CREATE INDEX idx_audit_logs_request_id ON audit_logs(request_id);
CREATE INDEX idx_audit_logs_session_id ON audit_logs(session_id);
CREATE INDEX idx_audit_logs_created_at_date ON audit_logs(created_at_date);

-- Composite indexes for common query patterns
CREATE INDEX idx_audit_logs_user_org_date ON audit_logs(user_id, organization_id, created_at DESC);
CREATE INDEX idx_audit_logs_resource_date ON audit_logs(resource_type, resource_id, created_at DESC);
CREATE INDEX idx_audit_logs_severity_date ON audit_logs(severity, created_at DESC);

-- Full-text search index for description and metadata
CREATE INDEX idx_audit_logs_search ON audit_logs USING gin(to_tsvector('english', description || ' ' || COALESCE(metadata::text, '')));

-- Partitioning by date for large-scale deployments (optional)
-- CREATE TABLE audit_logs_2024_01 PARTITION OF audit_logs FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
-- CREATE TABLE audit_logs_2024_02 PARTITION OF audit_logs FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');

-- Retention policy (optional - for automatic cleanup)
-- CREATE POLICY audit_logs_retention ON audit_logs FOR DELETE USING (created_at < now() - interval '2 years'); 