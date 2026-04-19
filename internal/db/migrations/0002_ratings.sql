-- 0002: add user-editable rating (0-5) to files.
ALTER TABLE files ADD COLUMN rating INTEGER NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_files_rating ON files(rating) WHERE rating > 0;
