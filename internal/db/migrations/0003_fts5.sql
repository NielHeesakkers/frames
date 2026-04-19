-- 0003: full-text search over filename + relative_path via FTS5.
-- Backed by the `files` table (content='files', content_rowid='id'), kept in
-- sync by triggers. Query with: SELECT ... WHERE id IN (SELECT rowid FROM files_fts WHERE files_fts MATCH ?).

CREATE VIRTUAL TABLE IF NOT EXISTS files_fts USING fts5(
  filename, relative_path,
  content='files', content_rowid='id'
);

-- Seed with existing rows.
INSERT INTO files_fts(rowid, filename, relative_path)
SELECT id, filename, relative_path FROM files;

CREATE TRIGGER IF NOT EXISTS files_ai AFTER INSERT ON files BEGIN
  INSERT INTO files_fts(rowid, filename, relative_path)
  VALUES (new.id, new.filename, new.relative_path);
END;

CREATE TRIGGER IF NOT EXISTS files_ad AFTER DELETE ON files BEGIN
  INSERT INTO files_fts(files_fts, rowid, filename, relative_path)
  VALUES ('delete', old.id, old.filename, old.relative_path);
END;

CREATE TRIGGER IF NOT EXISTS files_au AFTER UPDATE ON files BEGIN
  INSERT INTO files_fts(files_fts, rowid, filename, relative_path)
  VALUES ('delete', old.id, old.filename, old.relative_path);
  INSERT INTO files_fts(rowid, filename, relative_path)
  VALUES (new.id, new.filename, new.relative_path);
END;
