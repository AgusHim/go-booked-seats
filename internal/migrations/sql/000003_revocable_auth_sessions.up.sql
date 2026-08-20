CREATE TABLE IF NOT EXISTS auth_sessions (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    user_agent VARCHAR(512) NOT NULL DEFAULT '',
    ip_address VARCHAR(64) NOT NULL DEFAULT '',
    expires_at TIMESTAMP NOT NULL,
    last_used_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP NULL,
    replaced_by_id VARCHAR(36) NULL,
    created_at TIMESTAMP NOT NULL,
    CONSTRAINT fk_auth_sessions_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_auth_sessions_replacement
        FOREIGN KEY (replaced_by_id) REFERENCES auth_sessions(id)
);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id
    ON auth_sessions(user_id);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_expires_at
    ON auth_sessions(expires_at);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_revoked_at
    ON auth_sessions(revoked_at);

CREATE TABLE IF NOT EXISTS auth_tokens (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    purpose VARCHAR(32) NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL,
    CONSTRAINT fk_auth_tokens_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_auth_tokens_user_purpose
    ON auth_tokens(user_id, purpose);

CREATE INDEX IF NOT EXISTS idx_auth_tokens_expires_at
    ON auth_tokens(expires_at);

CREATE TABLE IF NOT EXISTS user_email_verifications (
    user_id VARCHAR(36) PRIMARY KEY,
    email VARCHAR(320) NOT NULL UNIQUE,
    verified_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    CONSTRAINT fk_user_email_verifications_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
