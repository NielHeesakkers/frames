-- 0004: allow shares to scope to a specific set of files (by ID) instead of
-- the whole folder subtree. JSON array stored as TEXT. NULL/empty means the
-- share still covers the whole folder as before.
ALTER TABLE shares ADD COLUMN file_ids TEXT;
