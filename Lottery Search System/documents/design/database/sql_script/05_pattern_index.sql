-- ==============================================================================
-- Table: pattern_index
-- Description: Inverted Index จัดเก็บความสัมพันธ์ระหว่าง Pattern, ตำแหน่งตัวเลข, และสลาก
-- Source: table_5.png (Database Schema Relational)
-- ==============================================================================

CREATE TABLE IF NOT EXISTS pattern_index (
    index_id      BIGSERIAL PRIMARY KEY,
    pattern_id    BIGINT NOT NULL,
    position      INT NOT NULL,
    digit         CHAR(1) NULL,
    ticket_id     BIGINT NOT NULL,

    -- Foreign Keys
    CONSTRAINT fk_pattern_index_pattern 
        FOREIGN KEY (pattern_id) REFERENCES patterns(pattern_id) 
        ON DELETE CASCADE,

    CONSTRAINT fk_pattern_index_ticket 
        FOREIGN KEY (ticket_id) REFERENCES tickets(ticket_id) 
        ON DELETE CASCADE,

    -- Constraints
    -- pattern_index UNIQUE(pattern_id, position, digit, ticket_id)
    CONSTRAINT uq_pattern_index UNIQUE (pattern_id, position, digit, ticket_id)
);

-- Recommended Indexes
-- (Optional) composite index: (pattern_id, position, digit)
CREATE INDEX IF NOT EXISTS idx_pattern_index_pattern_pos_digit 
ON pattern_index (pattern_id, position, digit);

-- Comments
COMMENT ON TABLE pattern_index IS 'Inverted index mapping search patterns and digit positions to matching tickets';
COMMENT ON COLUMN pattern_index.index_id IS 'Primary key for each pattern index entry';
COMMENT ON COLUMN pattern_index.pattern_id IS 'Foreign key referencing patterns table';
COMMENT ON COLUMN pattern_index.position IS 'Position of the character/digit in the 6-digit number (1 to 6)';
COMMENT ON COLUMN pattern_index.digit IS 'Character value at the position (0-9, or NULL for wildcard)';
COMMENT ON COLUMN pattern_index.ticket_id IS 'Foreign key referencing tickets table';
