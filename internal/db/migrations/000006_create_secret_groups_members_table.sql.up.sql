CREATE TABLE secret_group_members (
    secret_group_id UUID REFERENCES secret_groups(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role role_type NOT NULL,
    PRIMARY KEY (secret_group_id, user_id)
);
