CREATE TABLE IF NOT EXISTS url_mappings (
    uuid VARCHAR(36) PRIMARY KEY,
    short_url VARCHAR(50) UNIQUE NOT NULL,
    original_url TEXT UNIQUE NOT NULL,
    user_id VARCHAR(36),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_short_url ON url_mappings(short_url);
CREATE UNIQUE INDEX IF NOT EXISTS idx_original_url ON url_mappings(original_url);
CREATE INDEX IF NOT EXISTS idx_user_id ON url_mappings(user_id);