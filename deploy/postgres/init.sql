-- Create database if not exists
SELECT 'CREATE DATABASE telltheworld'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'telltheworld')\gexec

-- Connect to the database
\c telltheworld;

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Create notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,

    notification_id VARCHAR(255) NOT NULL,
    title VARCHAR(255) NOT NULL,
    body TEXT,
    ref_id VARCHAR(255),

    schedule_at TIMESTAMP WITH TIME ZONE,
    sent_at TIMESTAMP WITH TIME ZONE,

    channels JSONB,
    results JSONB,

    status VARCHAR(50) DEFAULT 'pending',
    processing_started_at TIMESTAMP WITH TIME ZONE,
    processing_worker_id VARCHAR(255),
    processing_attempts INTEGER DEFAULT 0,

    CONSTRAINT unique_notification_id UNIQUE (notification_id)
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
CREATE INDEX IF NOT EXISTS idx_notifications_ref_id ON notifications(ref_id);
CREATE INDEX IF NOT EXISTS idx_notifications_processing_started_at ON notifications(processing_started_at);
CREATE INDEX IF NOT EXISTS idx_notifications_deleted_at ON notifications(deleted_at);
CREATE INDEX IF NOT EXISTS idx_notifications_schedule_at ON notifications(schedule_at);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger to auto-update updated_at
CREATE TRIGGER update_notifications_updated_at BEFORE UPDATE
    ON notifications FOR EACH ROW EXECUTE PROCEDURE
    update_updated_at_column();

-- Create composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_notifications_pending_status
    ON notifications(notification_id, status)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_notifications_processing
    ON notifications(processing_worker_id, status)
    WHERE status = 'processing';

-- Grant permissions (adjust as needed)
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO postgres;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO postgres;

-- Add comments for documentation
COMMENT ON TABLE notifications IS 'Stores all notification records';
COMMENT ON COLUMN notifications.notification_id IS 'UUID for the notification';
COMMENT ON COLUMN notifications.status IS 'Current status: pending, processing, sent, failed';
COMMENT ON COLUMN notifications.processing_worker_id IS 'ID of the worker currently processing this notification';
COMMENT ON COLUMN notifications.processing_attempts IS 'Number of times this notification has been attempted';
COMMENT ON COLUMN notifications.channels IS 'JSON array of notification channels (email, push, etc.)';
COMMENT ON COLUMN notifications.results IS 'JSON array of results per channel';