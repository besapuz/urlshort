ALTER TABLE url_mappings ADD COLUMN deleted BOOLEAN DEFAULT FALSE;
CREATE INDEX idx_deleted ON url_mappings(deleted);