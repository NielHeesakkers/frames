-- 0005: One-off backfill for photos stored with raw-sensor width/height and
-- an EXIF orientation flag that rotates the display 90°. After this migration
-- width/height always reflect the *displayed* image; the justified-rows grid
-- and the lightbox info panel can trust them.
UPDATE files
SET width = height, height = width
WHERE orientation IN (5, 6, 7, 8)
  AND width > height;
