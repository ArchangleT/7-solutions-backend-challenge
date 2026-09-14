-- ==============================================================================
-- Table: tickets
-- Description: เก็บข้อมูลสลากกินแบ่งรัฐบาลทั้งหมด (10,000,000 ใบ)
-- Source: table_5.png (Database Schema Relational)
-- ==============================================================================

CREATE TABLE IF NOT EXISTS tickets (
    ticket_id       BIGINT PRIMARY KEY,
    ticket_number   CHAR(6) NOT NULL,
    allocated       BOOLEAN NOT NULL DEFAULT FALSE,
    allocated_at    TIMESTAMP NULL,

    -- Constraints
    CONSTRAINT uq_tickets_ticket_number UNIQUE (ticket_number)
);

-- Recommended Indexes
-- tickets(allocated, ticket_number)
CREATE INDEX IF NOT EXISTS idx_tickets_allocated_number 
ON tickets (allocated, ticket_number);

-- Comments
COMMENT ON TABLE tickets IS 'Stores lottery tickets inventory (10,000,000 records)';
COMMENT ON COLUMN tickets.ticket_id IS 'Primary key, unique identifier for each ticket (1 to 10,000,000)';
COMMENT ON COLUMN tickets.ticket_number IS '6-digit lottery number (000000 to 999999), unique per ticket in this schema';
COMMENT ON COLUMN tickets.allocated IS 'Flag indicating whether the ticket has been allocated/reserved';
COMMENT ON COLUMN tickets.allocated_at IS 'Timestamp when the ticket was allocated (NULL if available)';
