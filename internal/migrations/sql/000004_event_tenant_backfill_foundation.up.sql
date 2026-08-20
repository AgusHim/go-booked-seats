CREATE TABLE IF NOT EXISTS event_community_assignments (
    event_id VARCHAR(36) PRIMARY KEY,
    community_id VARCHAR(36) NOT NULL,
    source VARCHAR(32) NOT NULL,
    assigned_at TIMESTAMP NOT NULL,
    reviewed_at TIMESTAMP NULL,
    CONSTRAINT fk_event_community_assignments_event
        FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE,
    CONSTRAINT fk_event_community_assignments_community
        FOREIGN KEY (community_id) REFERENCES communities(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_event_community_assignments_community_id
    ON event_community_assignments(community_id);
