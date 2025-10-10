-- HowlerOps Development Database Initialization Script
-- This script sets up the development database with sample data

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create development user
INSERT INTO users (email, password_hash, name, role, is_active, email_verified)
VALUES (
    'dev@sqlstudio.local',
    '$2b$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', -- password
    'Developer',
    'admin',
    true,
    true
) ON CONFLICT (email) DO NOTHING;

-- Insert sample database connections
INSERT INTO database_connections (user_id, name, type, host, port, database_name, username, is_active)
SELECT
    u.id,
    'Local PostgreSQL',
    'postgres',
    'localhost',
    5432,
    'postgres',
    'postgres',
    true
FROM users u WHERE u.email = 'dev@sqlstudio.local'
ON CONFLICT DO NOTHING;

INSERT INTO database_connections (user_id, name, type, host, port, database_name, username, is_active)
SELECT
    u.id,
    'Test MySQL',
    'mysql',
    'localhost',
    3306,
    'test',
    'root',
    true
FROM users u WHERE u.email = 'dev@sqlstudio.local'
ON CONFLICT DO NOTHING;

-- Insert sample saved queries
INSERT INTO saved_queries (user_id, name, description, query_text, tags, is_public)
SELECT
    u.id,
    'Show All Tables',
    'Display all tables in the current database',
    'SELECT table_name FROM information_schema.tables WHERE table_schema = ''public'';',
    ARRAY['utility', 'tables'],
    true
FROM users u WHERE u.email = 'dev@sqlstudio.local'
ON CONFLICT DO NOTHING;

INSERT INTO saved_queries (user_id, name, description, query_text, tags, is_public)
SELECT
    u.id,
    'Database Size',
    'Check the size of the current database',
    'SELECT pg_size_pretty(pg_database_size(current_database())) as database_size;',
    ARRAY['monitoring', 'size'],
    true
FROM users u WHERE u.email = 'dev@sqlstudio.local'
ON CONFLICT DO NOTHING;

INSERT INTO saved_queries (user_id, name, description, query_text, tags, is_public)
SELECT
    u.id,
    'Active Connections',
    'Show current active database connections',
    'SELECT datname, usename, application_name, client_addr, state, query_start FROM pg_stat_activity WHERE state = ''active'';',
    ARRAY['monitoring', 'connections'],
    true
FROM users u WHERE u.email = 'dev@sqlstudio.local'
ON CONFLICT DO NOTHING;

-- Insert sample query history
INSERT INTO query_history (user_id, connection_id, query_text, execution_time_ms, rows_affected, status)
SELECT
    u.id,
    dc.id,
    'SELECT version();',
    15,
    1,
    'success'
FROM users u
CROSS JOIN database_connections dc
WHERE u.email = 'dev@sqlstudio.local' AND dc.name = 'Local PostgreSQL';

INSERT INTO query_history (user_id, connection_id, query_text, execution_time_ms, rows_affected, status)
SELECT
    u.id,
    dc.id,
    'SELECT COUNT(*) FROM information_schema.tables;',
    8,
    1,
    'success'
FROM users u
CROSS JOIN database_connections dc
WHERE u.email = 'dev@sqlstudio.local' AND dc.name = 'Local PostgreSQL';