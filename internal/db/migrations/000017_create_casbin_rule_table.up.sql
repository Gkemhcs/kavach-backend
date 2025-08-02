-- Create casbin_rule table for Casbin authorization
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
CREATE INDEX IF NOT EXISTS idx_casbin_rule_ptype ON casbin_rule (ptype);
CREATE INDEX IF NOT EXISTS idx_casbin_rule_v0 ON casbin_rule (v0);
CREATE INDEX IF NOT EXISTS idx_casbin_rule_v1 ON casbin_rule (v1);
CREATE INDEX IF NOT EXISTS idx_casbin_rule_v2 ON casbin_rule (v2);
CREATE INDEX IF NOT EXISTS idx_casbin_rule_v3 ON casbin_rule (v3);
CREATE INDEX IF NOT EXISTS idx_casbin_rule_v4 ON casbin_rule (v4);
CREATE INDEX IF NOT EXISTS idx_casbin_rule_v5 ON casbin_rule (v5); 