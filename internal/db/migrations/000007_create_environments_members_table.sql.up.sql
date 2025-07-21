CREATE TABLE environment_members (
    environment_id UUID REFERENCES environments(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role role_type NOT NULL,
    PRIMARY KEY (environment_id, user_id)
);
