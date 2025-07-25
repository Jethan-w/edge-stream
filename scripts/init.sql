-- EdgeStream Database Initialization Script
-- This script initializes the PostgreSQL database for EdgeStream

-- Create database if not exists (this is handled by docker-compose)
-- CREATE DATABASE IF NOT EXISTS edgestream;

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- Create schemas
CREATE SCHEMA IF NOT EXISTS edgestream;
CREATE SCHEMA IF NOT EXISTS metrics;
CREATE SCHEMA IF NOT EXISTS audit;

-- Set default schema
SET search_path TO edgestream, public;

-- Create tables for EdgeStream

-- FlowFile metadata table
CREATE TABLE IF NOT EXISTS flowfiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    filename VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL DEFAULT 0,
    attributes JSONB,
    content_type VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    status VARCHAR(50) DEFAULT 'active',
    checksum VARCHAR(64),
    source_system VARCHAR(100),
    destination_system VARCHAR(100)
);

-- Create indexes for flowfiles
CREATE INDEX IF NOT EXISTS idx_flowfiles_created_at ON flowfiles(created_at);
CREATE INDEX IF NOT EXISTS idx_flowfiles_status ON flowfiles(status);
CREATE INDEX IF NOT EXISTS idx_flowfiles_source ON flowfiles(source_system);
CREATE INDEX IF NOT EXISTS idx_flowfiles_destination ON flowfiles(destination_system);
CREATE INDEX IF NOT EXISTS idx_flowfiles_attributes ON flowfiles USING GIN(attributes);

-- Processor execution history
CREATE TABLE IF NOT EXISTS processor_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    flowfile_id UUID REFERENCES flowfiles(id) ON DELETE CASCADE,
    processor_name VARCHAR(100) NOT NULL,
    processor_type VARCHAR(50) NOT NULL,
    execution_time_ms INTEGER,
    status VARCHAR(50) DEFAULT 'success',
    error_message TEXT,
    input_attributes JSONB,
    output_attributes JSONB,
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for processor_executions
CREATE INDEX IF NOT EXISTS idx_processor_executions_flowfile ON processor_executions(flowfile_id);
CREATE INDEX IF NOT EXISTS idx_processor_executions_processor ON processor_executions(processor_name);
CREATE INDEX IF NOT EXISTS idx_processor_executions_executed_at ON processor_executions(executed_at);
CREATE INDEX IF NOT EXISTS idx_processor_executions_status ON processor_executions(status);

-- Stream processing state
CREATE TABLE IF NOT EXISTS stream_state (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    stream_name VARCHAR(100) NOT NULL UNIQUE,
    state_data JSONB,
    checkpoint_data JSONB,
    last_checkpoint TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for stream_state
CREATE INDEX IF NOT EXISTS idx_stream_state_name ON stream_state(stream_name);
CREATE INDEX IF NOT EXISTS idx_stream_state_updated_at ON stream_state(updated_at);

-- Metrics tables in metrics schema
SET search_path TO metrics, public;

-- System metrics
CREATE TABLE IF NOT EXISTS system_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    metric_name VARCHAR(100) NOT NULL,
    metric_value DOUBLE PRECISION NOT NULL,
    metric_type VARCHAR(50) NOT NULL, -- counter, gauge, histogram, summary
    labels JSONB,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for system_metrics
CREATE INDEX IF NOT EXISTS idx_system_metrics_name ON system_metrics(metric_name);
CREATE INDEX IF NOT EXISTS idx_system_metrics_timestamp ON system_metrics(timestamp);
CREATE INDEX IF NOT EXISTS idx_system_metrics_labels ON system_metrics USING GIN(labels);

-- Performance metrics
CREATE TABLE IF NOT EXISTS performance_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    component_name VARCHAR(100) NOT NULL,
    component_type VARCHAR(50) NOT NULL, -- processor, source, sink, etc.
    throughput_per_second DOUBLE PRECISION,
    latency_ms DOUBLE PRECISION,
    error_rate DOUBLE PRECISION,
    memory_usage_mb DOUBLE PRECISION,
    cpu_usage_percent DOUBLE PRECISION,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance_metrics
CREATE INDEX IF NOT EXISTS idx_performance_metrics_component ON performance_metrics(component_name);
CREATE INDEX IF NOT EXISTS idx_performance_metrics_type ON performance_metrics(component_type);
CREATE INDEX IF NOT EXISTS idx_performance_metrics_timestamp ON performance_metrics(timestamp);

-- Audit tables in audit schema
SET search_path TO audit, public;

-- Configuration changes audit
CREATE TABLE IF NOT EXISTS config_changes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    config_key VARCHAR(200) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    changed_by VARCHAR(100),
    change_reason TEXT,
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for config_changes
CREATE INDEX IF NOT EXISTS idx_config_changes_key ON config_changes(config_key);
CREATE INDEX IF NOT EXISTS idx_config_changes_changed_at ON config_changes(changed_at);
CREATE INDEX IF NOT EXISTS idx_config_changes_changed_by ON config_changes(changed_by);

-- System events audit
CREATE TABLE IF NOT EXISTS system_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB,
    severity VARCHAR(20) DEFAULT 'info', -- debug, info, warn, error, fatal
    source_component VARCHAR(100),
    user_id VARCHAR(100),
    session_id VARCHAR(100),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for system_events
CREATE INDEX IF NOT EXISTS idx_system_events_type ON system_events(event_type);
CREATE INDEX IF NOT EXISTS idx_system_events_severity ON system_events(severity);
CREATE INDEX IF NOT EXISTS idx_system_events_timestamp ON system_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_system_events_component ON system_events(source_component);
CREATE INDEX IF NOT EXISTS idx_system_events_data ON system_events USING GIN(event_data);

-- Create functions for automatic timestamp updates
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for automatic timestamp updates
CREATE TRIGGER update_flowfiles_updated_at BEFORE UPDATE ON edgestream.flowfiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_stream_state_updated_at BEFORE UPDATE ON edgestream.stream_state
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create views for common queries
SET search_path TO edgestream, public;

-- Active flowfiles view
CREATE OR REPLACE VIEW active_flowfiles AS
SELECT 
    id,
    filename,
    size,
    attributes,
    content_type,
    created_at,
    updated_at,
    source_system,
    destination_system
FROM flowfiles
WHERE status = 'active';

-- Processor performance view
CREATE OR REPLACE VIEW processor_performance AS
SELECT 
    processor_name,
    processor_type,
    COUNT(*) as execution_count,
    AVG(execution_time_ms) as avg_execution_time_ms,
    MIN(execution_time_ms) as min_execution_time_ms,
    MAX(execution_time_ms) as max_execution_time_ms,
    COUNT(CASE WHEN status = 'success' THEN 1 END) as success_count,
    COUNT(CASE WHEN status = 'error' THEN 1 END) as error_count,
    ROUND(COUNT(CASE WHEN status = 'success' THEN 1 END) * 100.0 / COUNT(*), 2) as success_rate
FROM processor_executions
WHERE executed_at >= NOW() - INTERVAL '24 hours'
GROUP BY processor_name, processor_type;

-- Grant permissions
GRANT USAGE ON SCHEMA edgestream TO PUBLIC;
GRANT USAGE ON SCHEMA metrics TO PUBLIC;
GRANT USAGE ON SCHEMA audit TO PUBLIC;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA edgestream TO PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA metrics TO PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA audit TO PUBLIC;

GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA edgestream TO PUBLIC;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA metrics TO PUBLIC;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA audit TO PUBLIC;

-- Insert initial data
INSERT INTO edgestream.stream_state (stream_name, state_data, checkpoint_data) 
VALUES 
    ('default_stream', '{}', '{}')
ON CONFLICT (stream_name) DO NOTHING;

-- Log initialization
INSERT INTO audit.system_events (event_type, event_data, severity, source_component)
VALUES 
    ('database_initialized', '{"version": "1.0.0", "timestamp": "' || NOW() || '"}', 'info', 'database');

COMMIT;