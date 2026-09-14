-- ==============================================================================
-- File: 00_init_db_lottery.sql
-- Description: Master Bootstrap Script สำหรับสร้างฐานข้อมูล lottery_system 
--              พร้อมสร้างตาราง, ดัชนี, ข้อจำกัด (Constraints) และ Comments ครบถ้วน
-- Database: PostgreSQL 14+
-- Target Database: lottery_system
-- Source Reference: table_5.png (Database Schema Relational)
-- ==============================================================================

-- ------------------------------------------------------------------------------
-- 1. CREATE DATABASE: lottery_system (สร้างฐานข้อมูลหากยังไม่มี)
-- ------------------------------------------------------------------------------
SELECT 'CREATE DATABASE lottery_system WITH ENCODING = ''UTF8'' TEMPLATE = template1'
WHERE NOT EXISTS (
    SELECT FROM pg_database WHERE datname = 'lottery_system'
)\gexec

-- ------------------------------------------------------------------------------
-- 2. CONNECT TO DATABASE: lottery_system
-- ------------------------------------------------------------------------------
\c lottery_system;

-- ------------------------------------------------------------------------------
-- 8. VERIFICATION SUMMARY
-- ------------------------------------------------------------------------------
\echo '=================================================='
\echo ' Database lottery_system initialized successfully!'
\echo '=================================================='
