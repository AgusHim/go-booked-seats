CREATE TABLE IF NOT EXISTS communities (
    id VARCHAR(36) PRIMARY KEY,
    slug VARCHAR(160) NOT NULL UNIQUE,
    name VARCHAR(160) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    type VARCHAR(32) NOT NULL DEFAULT 'general',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    logo_url TEXT NOT NULL DEFAULT '',
    cover_url TEXT NOT NULL DEFAULT '',
    location VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS community_members (
    id VARCHAR(36) PRIMARY KEY,
    community_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    role VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    CONSTRAINT fk_community_members_community
        FOREIGN KEY (community_id) REFERENCES communities(id) ON DELETE CASCADE,
    CONSTRAINT fk_community_members_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT uq_community_members_community_user
        UNIQUE (community_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_community_members_user_id
    ON community_members(user_id);

CREATE TABLE IF NOT EXISTS community_invitations (
    id VARCHAR(36) PRIMARY KEY,
    community_id VARCHAR(36) NOT NULL,
    email VARCHAR(320) NOT NULL,
    role VARCHAR(32) NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    invited_by VARCHAR(36) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    accepted_at TIMESTAMP NULL,
    accepted_by VARCHAR(36) NULL,
    created_at TIMESTAMP NOT NULL,
    CONSTRAINT fk_community_invitations_community
        FOREIGN KEY (community_id) REFERENCES communities(id) ON DELETE CASCADE,
    CONSTRAINT fk_community_invitations_inviter
        FOREIGN KEY (invited_by) REFERENCES users(id),
    CONSTRAINT fk_community_invitations_accepter
        FOREIGN KEY (accepted_by) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_community_invitations_community_email
    ON community_invitations(community_id, email);

CREATE INDEX IF NOT EXISTS idx_community_invitations_expires_at
    ON community_invitations(expires_at);
