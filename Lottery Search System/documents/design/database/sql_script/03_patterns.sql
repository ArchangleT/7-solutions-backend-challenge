-- ==============================================================================
-- Table: patterns
-- Description: เก็บข้อมูล Search Patterns (เช่น 1****5, ****23) ที่ถูกค้นหา
-- Source: table_5.png (Database Schema Relational)
-- ==============================================================================

CREATE TABLE IF NOT EXISTS patterns (
    pattern_id    BIGSERIAL PRIMARY KEY,
    pattern_text  VARCHAR(6) NOT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT uq_patterns_pattern_text UNIQUE (pattern_text)
);

-- Recommended Indexes
-- patterns(pattern_text)
CREATE INDEX IF NOT EXISTS idx_patterns_pattern_text 
ON patterns (pattern_text);

-- Comments
COMMENT ON TABLE patterns IS 'Stores unique search patterns requested by users';
COMMENT ON COLUMN patterns.pattern_id IS 'Primary key, unique identifier for each pattern';
COMMENT ON COLUMN patterns.pattern_text IS '6-character search pattern with digits and wildcards (e.g. 1****5, ****23)';
COMMENT ON COLUMN patterns.created_at IS 'Timestamp when the pattern was first recorded';
