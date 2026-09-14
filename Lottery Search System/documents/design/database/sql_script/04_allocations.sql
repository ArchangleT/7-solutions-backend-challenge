-- ==============================================================================
-- Table: allocations
-- Description: บันทึกข้อมูลการจัดสรร/จองสลากระหว่างผู้ใช้กับสลากตาม Pattern
-- Source: table_5.png (Database Schema Relational)
-- ==============================================================================

CREATE TABLE IF NOT EXISTS allocations (
    allocation_id   BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL,
    ticket_id       BIGINT NOT NULL,
    pattern         VARCHAR(6) NOT NULL,
    allocated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at      TIMESTAMP NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',

    -- Foreign Keys
    CONSTRAINT fk_allocations_user 
        FOREIGN KEY (user_id) REFERENCES users(user_id) 
        ON DELETE CASCADE,

    CONSTRAINT fk_allocations_ticket 
        FOREIGN KEY (ticket_id) REFERENCES tickets(ticket_id) 
        ON DELETE RESTRICT
);

-- Recommended Indexes
-- allocations(user_id, allocated_at)
CREATE INDEX IF NOT EXISTS idx_allocations_user_allocated_at 
ON allocations (user_id, allocated_at);

-- Partial Index สำหรับตรวจสอบสลากที่หมดอายุการจอง (Expiry Reaper Worker)
CREATE INDEX IF NOT EXISTS idx_allocations_active_expires_at 
ON allocations (expires_at) 
WHERE status = 'ACTIVE';

-- Comments
COMMENT ON TABLE allocations IS 'Stores ticket allocations/reservations per user and search pattern';
COMMENT ON COLUMN allocations.allocation_id IS 'Primary key, unique identifier for each allocation';
COMMENT ON COLUMN allocations.user_id IS 'Foreign key referencing users table';
COMMENT ON COLUMN allocations.ticket_id IS 'Foreign key referencing tickets table';
COMMENT ON COLUMN allocations.pattern IS 'Search pattern used during allocation (e.g. 1****5)';
COMMENT ON COLUMN allocations.allocated_at IS 'Timestamp when the allocation was created';
COMMENT ON COLUMN allocations.expires_at IS 'Timestamp when the temporary reservation expires';
COMMENT ON COLUMN allocations.status IS 'Current status of the allocation (ACTIVE, COMPLETED, EXPIRED, CANCELLED)';
