-- ==============================================================================
-- Table: users
-- Description: เก็บข้อมูลผู้ใช้งานที่เข้ามาค้นหาและรับการจัดสรรสลาก
-- Source: table_5.png (Database Schema Relational)
-- ==============================================================================

CREATE TABLE IF NOT EXISTS users (
    user_id     BIGSERIAL PRIMARY KEY,
    name        VARCHAR(100) NOT NULL,
    email       VARCHAR(100) UNIQUE NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Comments
COMMENT ON TABLE users IS 'Stores registered users participating in lottery ticket search & allocation';
COMMENT ON COLUMN users.user_id IS 'Primary key, unique identifier for each user';
COMMENT ON COLUMN users.name IS 'Full name of the user';
COMMENT ON COLUMN users.email IS 'Unique email address of the user';
COMMENT ON COLUMN users.created_at IS 'Timestamp when the user record was created';
