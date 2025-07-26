-- Create Casbin rules table for authorization policies
CREATE TABLE IF NOT EXISTS casbin_rule (
    id SERIAL PRIMARY KEY,
    ptype VARCHAR(10) NOT NULL,
    v0 VARCHAR(256),
    v1 VARCHAR(256),
    v2 VARCHAR(256),
    v3 VARCHAR(256),
    v4 VARCHAR(256),
    v5 VARCHAR(256)
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_casbin_rule_ptype ON casbin_rule(ptype);
CREATE INDEX IF NOT EXISTS idx_casbin_rule_v0 ON casbin_rule(v0);
CREATE INDEX IF NOT EXISTS idx_casbin_rule_v1 ON casbin_rule(v1);
CREATE INDEX IF NOT EXISTS idx_casbin_rule_v2 ON casbin_rule(v2);

-- Add comments for documentation
COMMENT ON TABLE casbin_rule IS 'Stores Casbin authorization policies for RBAC enforcement';
COMMENT ON COLUMN casbin_rule.ptype IS 'Policy type: p (policy) or g (role assignment)';
COMMENT ON COLUMN casbin_rule.v0 IS 'Subject (user or group) for g rules, or role for p rules';
COMMENT ON COLUMN casbin_rule.v1 IS 'Role for g rules, or resource path for p rules';
COMMENT ON COLUMN casbin_rule.v2 IS 'Action for p rules (read, write, delete, grant)';
COMMENT ON COLUMN casbin_rule.v3 IS 'Additional parameters (unused in current implementation)';
COMMENT ON COLUMN casbin_rule.v4 IS 'Additional parameters (unused in current implementation)';
COMMENT ON COLUMN casbin_rule.v5 IS 'Additional parameters (unused in current implementation)'; 