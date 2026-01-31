-- Alert deduplication support
ALTER TABLE alerts ADD COLUMN IF NOT EXISTS occurrence_count INT NOT NULL DEFAULT 1;
ALTER TABLE alerts ADD COLUMN IF NOT EXISTS last_occurrence TIMESTAMPTZ;

-- Index for efficient deduplication lookup
CREATE INDEX IF NOT EXISTS idx_alerts_dedup ON alerts(server_id, title) WHERE status = 'open';
