CREATE TABLE IF NOT EXISTS clips (
    room_id TEXT PRIMARY KEY,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for faster cleanup queries
CREATE INDEX IF NOT EXISTS idx_clips_updated_at ON clips(updated_at);

-- CREATE TABLE IF NOT EXISTS clips (
--     id VARCHAR(255) PRIMARY KEY,
--     content TEXT,
--     updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
--     expires_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP + INTERVAL '24 hours')
-- );

-- CREATE INDEX idx_expires ON clips(expires_at);