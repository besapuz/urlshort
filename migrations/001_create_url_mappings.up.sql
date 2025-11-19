CREATE TABLE IF NOT EXISTS url_mappings (
    id SERIAL PRIMARY KEY,
    uuid TEXT UNIQUE NOT NULL,
    short_url TEXT UNIQUE NOT NULL,
    original_url TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_short_url ON url_mappings(short_url);
CREATE INDEX idx_uuid ON url_mappings(uuid);