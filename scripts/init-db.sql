-- HowlerOps Database Initialization Script
-- This script sets up the initial database schema for production

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- Create application user (if not exists)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'sqlstudio_app') THEN
        CREATE ROLE sqlstudio_app WITH LOGIN PASSWORD 'change-this-app-password';
    END IF;
END$$;

-- Grant necessary permissions
GRANT CONNECT ON DATABASE sqlstudio TO sqlstudio_app;
GRANT USAGE ON SCHEMA public TO sqlstudio_app;
GRANT CREATE ON SCHEMA public TO sqlstudio_app;

-- Create initial tables (these should match your backend migrations)
-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'user',
    is_active BOOLEAN DEFAULT true,
    email_verified BOOLEAN DEFAULT false,
    last_login TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Database connections table
CREATE TABLE IF NOT EXISTS database_connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- postgres, mysql, sqlite, etc.
    host VARCHAR(255),
    port INTEGER,
    database_name VARCHAR(255),
    username VARCHAR(255),
    password_encrypted TEXT, -- encrypted password
    ssl_enabled BOOLEAN DEFAULT false,
    ssl_cert TEXT,
    connection_params JSONB,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Query history table
CREATE TABLE IF NOT EXISTS query_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    connection_id UUID REFERENCES database_connections(id) ON DELETE CASCADE,
    query_text TEXT NOT NULL,
    execution_time_ms INTEGER,
    rows_affected INTEGER,
    status VARCHAR(50), -- success, error, cancelled
    error_message TEXT,
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Saved queries table
CREATE TABLE IF NOT EXISTS saved_queries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    query_text TEXT NOT NULL,
    tags TEXT[],
    is_public BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User sessions table
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_accessed TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_database_connections_user_id ON database_connections(user_id);
CREATE INDEX IF NOT EXISTS idx_query_history_user_id ON query_history(user_id);
CREATE INDEX IF NOT EXISTS idx_query_history_executed_at ON query_history(executed_at);
CREATE INDEX IF NOT EXISTS idx_saved_queries_user_id ON saved_queries(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_token_hash ON user_sessions(token_hash);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);

-- Grant permissions to app user
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO sqlstudio_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO sqlstudio_app;

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_database_connections_updated_at BEFORE UPDATE ON database_connections
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_saved_queries_updated_at BEFORE UPDATE ON saved_queries
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert default admin user (password: admin123 - CHANGE THIS!)
INSERT INTO users (email, password_hash, name, role, is_active, email_verified)
VALUES (
    'admin@sqlstudio.local',
    '$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewbqKOg9vrf0ey.C', -- admin123
    'Administrator',
    'admin',
    true,
    true
) ON CONFLICT (email) DO NOTHING;

-- Log initialization
INSERT INTO query_history (user_id, connection_id, query_text, execution_time_ms, rows_affected, status)
SELECT
    (SELECT id FROM users WHERE email = 'admin@sqlstudio.local'),
    NULL,
    'Database initialization completed',
    0,
    0,
    'success';