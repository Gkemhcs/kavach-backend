CREATE TYPE role_type AS ENUM ('owner', 'admin', 'member', 'viewer');




CREATE TABLE org_members (
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
   role role_type NOT NULL,
    PRIMARY KEY (org_id, user_id)
);
