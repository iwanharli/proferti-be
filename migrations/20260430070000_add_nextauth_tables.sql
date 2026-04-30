-- +goose Up
ALTER TABLE t_users ADD COLUMN email_verified TIMESTAMPTZ;
ALTER TABLE t_users ADD COLUMN image TEXT;
ALTER TABLE t_users ALTER COLUMN password DROP NOT NULL; -- NextAuth OAuth tidak butuh password

-- Membuat tabel-tabel tambahan untuk NextAuth dengan prefix na_
CREATE TABLE na_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES t_users (id) ON DELETE CASCADE,
    type VARCHAR(255) NOT NULL,
    provider VARCHAR(255) NOT NULL,
    provider_account_id VARCHAR(255) NOT NULL,
    refresh_token TEXT,
    access_token TEXT,
    expires_at BIGINT,
    token_type VARCHAR(255),
    scope VARCHAR(255),
    id_token TEXT,
    session_state TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_account_id)
);

CREATE TABLE na_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_token VARCHAR(255) NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES t_users (id) ON DELETE CASCADE,
    expires TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE na_verification_tokens (
    identifier VARCHAR(255) NOT NULL,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (identifier, token)
);

-- +goose Down
DROP TABLE IF EXISTS na_verification_tokens;
DROP TABLE IF EXISTS na_sessions;
DROP TABLE IF EXISTS na_accounts;
ALTER TABLE t_users DROP COLUMN IF EXISTS email_verified;
ALTER TABLE t_users DROP COLUMN IF EXISTS image;
ALTER TABLE t_users ALTER COLUMN password SET NOT NULL;



