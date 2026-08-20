CREATE TABLE IF NOT EXISTS community_follows (
    id VARCHAR(36) PRIMARY KEY,
    community_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    CONSTRAINT fk_community_follows_community
        FOREIGN KEY (community_id) REFERENCES communities(id) ON DELETE CASCADE,
    CONSTRAINT fk_community_follows_user
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT uq_community_follows_community_user
        UNIQUE (community_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_community_follows_user_id
    ON community_follows(user_id);
