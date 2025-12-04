-- Drop indexes
DROP INDEX IF EXISTS idx_created_by_ip;
DROP INDEX IF EXISTS idx_created_at;
DROP INDEX IF EXISTS idx_short_code;

-- Drop table
DROP TABLE IF EXISTS url_mappings;
