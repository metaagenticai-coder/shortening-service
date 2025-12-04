-- Create url_mappings table
CREATE TABLE IF NOT EXISTS url_mappings (
    id BIGSERIAL PRIMARY KEY,
    short_code VARCHAR(8) UNIQUE NOT NULL,
    long_url TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    created_by_ip INET,

    CONSTRAINT short_code_length CHECK (char_length(short_code) BETWEEN 6 AND 8),
    CONSTRAINT long_url_length CHECK (char_length(long_url) BETWEEN 10 AND 2048),
    CONSTRAINT long_url_scheme CHECK (long_url LIKE 'http://%' OR long_url LIKE 'https://%')
);

-- Create indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_short_code ON url_mappings(short_code);
CREATE INDEX IF NOT EXISTS idx_created_at ON url_mappings(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_created_by_ip ON url_mappings(created_by_ip) WHERE created_by_ip IS NOT NULL;

-- Add comment to table
COMMENT ON TABLE url_mappings IS 'Stores mappings between short codes and long URLs';
COMMENT ON COLUMN url_mappings.id IS 'Auto-incrementing unique identifier';
COMMENT ON COLUMN url_mappings.short_code IS 'Base62-encoded short code (6-8 characters)';
COMMENT ON COLUMN url_mappings.long_url IS 'Original long URL (10-2048 characters)';
COMMENT ON COLUMN url_mappings.created_at IS 'Timestamp when the URL was shortened (UTC)';
COMMENT ON COLUMN url_mappings.created_by_ip IS 'Client IP address for abuse tracking';
