-- 1. Create ENUMs

-- Enum for user roles
CREATE TYPE user_role AS ENUM (
    'owner',
    'admin',
    'editor',
    'viewer'
);

-- Enum for resource types
CREATE TYPE resource_type AS ENUM (
    'organization',
    'secret_group',
    'environment'
);

-- 2. Create role_bindings table

CREATE TABLE role_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_id UUID NOT NULL,
    role user_role NOT NULL,

    resource_type resource_type NOT NULL,
    resource_id UUID NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),


    CONSTRAINT fk_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

