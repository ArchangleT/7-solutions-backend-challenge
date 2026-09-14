-- ==============================================================================
-- File: 00_init_schema.sql
-- Description: Master Script รวบรวม DDL ทั้งหมดตามลำดับ Dependency
-- Source: table_5.png (Database Schema Relational)
-- ==============================================================================

\echo 'Creating table: users...'
\i 01_users.sql

\echo 'Creating table: tickets...'
\i 02_tickets.sql

\echo 'Creating table: patterns...'
\i 03_patterns.sql

\echo 'Creating table: allocations...'
\i 04_allocations.sql

\echo 'Creating table: pattern_index...'
\i 05_pattern_index.sql

\echo 'Schema initialization complete.'
