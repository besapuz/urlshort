ALTER TABLE url_mappings ADD COLUMN user_id VARCHAR(36);
CREATE INDEX idx_user_id ON url_mappings(user_id);