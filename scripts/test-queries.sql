-- Test Queries for Teller Notification Service
-- Run these queries to validate system functionality

-- ============================================
-- 1. BASIC HEALTH CHECKS
-- ============================================

-- Check if notifications table exists and has data
SELECT
    'Notifications Table' as check_name,
    CASE
        WHEN COUNT(*) > 0 THEN 'PASS: ' || COUNT(*) || ' records found'
        ELSE 'FAIL: No notifications found'
    END as result
FROM notifications;

-- Check table structure
SELECT
    column_name,
    data_type,
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_name = 'notifications'
ORDER BY ordinal_position;

-- ============================================
-- 2. NOTIFICATION STATUS ANALYSIS
-- ============================================

-- Status distribution
SELECT
    status,
    COUNT(*) as count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER(), 2) as percentage
FROM notifications
GROUP BY status
ORDER BY count DESC;

-- Processing time analysis
SELECT
    status,
    COUNT(*) as count,
    AVG(EXTRACT(EPOCH FROM (updated_at - created_at))) as avg_processing_seconds,
    MIN(EXTRACT(EPOCH FROM (updated_at - created_at))) as min_processing_seconds,
    MAX(EXTRACT(EPOCH FROM (updated_at - created_at))) as max_processing_seconds
FROM notifications
WHERE updated_at IS NOT NULL
GROUP BY status;

-- ============================================
-- 3. WORKER PERFORMANCE
-- ============================================

-- Worker distribution
SELECT
    COALESCE(processing_worker_id, 'unassigned') as worker_id,
    COUNT(*) as total_processed,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
    COUNT(CASE WHEN status = 'processing' THEN 1 END) as currently_processing
FROM notifications
GROUP BY processing_worker_id
ORDER BY total_processed DESC;

-- Average processing attempts per worker
SELECT
    processing_worker_id,
    AVG(processing_attempts) as avg_attempts,
    MAX(processing_attempts) as max_attempts
FROM notifications
WHERE processing_worker_id IS NOT NULL
GROUP BY processing_worker_id;

-- ============================================
-- 4. RACE CONDITION DETECTION
-- ============================================

-- Check for notifications claimed by multiple workers (should be 0)
SELECT
    'Race Condition Check' as test,
    CASE
        WHEN COUNT(*) = 0 THEN 'PASS: No race conditions detected'
        ELSE 'FAIL: ' || COUNT(*) || ' notifications processed by multiple workers'
    END as result
FROM (
    SELECT notification_id
    FROM notifications
    WHERE processing_worker_id IS NOT NULL
    GROUP BY notification_id
    HAVING COUNT(DISTINCT processing_worker_id) > 1
) as duplicates;

-- ============================================
-- 5. STUCK NOTIFICATIONS
-- ============================================

-- Find stuck notifications (processing for more than 5 minutes)
SELECT
    notification_id,
    title,
    status,
    processing_worker_id,
    processing_started_at,
    EXTRACT(EPOCH FROM (NOW() - processing_started_at)) / 60 as minutes_processing
FROM notifications
WHERE status = 'processing'
    AND processing_started_at < NOW() - INTERVAL '5 minutes'
ORDER BY processing_started_at;

-- Count of stuck notifications by worker
SELECT
    processing_worker_id,
    COUNT(*) as stuck_count
FROM notifications
WHERE status = 'processing'
    AND processing_started_at < NOW() - INTERVAL '5 minutes'
GROUP BY processing_worker_id;

-- ============================================
-- 6. CHANNEL ANALYSIS
-- ============================================

-- Extract and analyze channels (using JSON operations)
WITH channel_data AS (
    SELECT
        notification_id,
        jsonb_array_elements(channels::jsonb) as channel_obj
    FROM notifications
    WHERE channels IS NOT NULL AND channels != '[]'
)
SELECT
    channel_obj->>'channel' as channel_type,
    COUNT(*) as count,
    COUNT(DISTINCT notification_id) as unique_notifications
FROM channel_data
GROUP BY channel_obj->>'channel'
ORDER BY count DESC;

-- Notifications with multiple channels
SELECT
    notification_id,
    title,
    jsonb_array_length(channels::jsonb) as channel_count,
    channels
FROM notifications
WHERE channels IS NOT NULL
    AND jsonb_array_length(channels::jsonb) > 1
LIMIT 10;

-- ============================================
-- 7. RETRY ANALYSIS
-- ============================================

-- Notifications requiring retries
SELECT
    processing_attempts,
    COUNT(*) as count
FROM notifications
WHERE processing_attempts > 0
GROUP BY processing_attempts
ORDER BY processing_attempts;

-- Failed notifications after max retries
SELECT
    notification_id,
    title,
    status,
    processing_attempts,
    error_message
FROM notifications
WHERE processing_attempts >= 5
    AND status = 'failed';

-- ============================================
-- 8. THROUGHPUT ANALYSIS
-- ============================================

-- Notifications per minute (last hour)
SELECT
    DATE_TRUNC('minute', created_at) as minute,
    COUNT(*) as notifications_created,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) as notifications_completed
FROM notifications
WHERE created_at > NOW() - INTERVAL '1 hour'
GROUP BY DATE_TRUNC('minute', created_at)
ORDER BY minute DESC
LIMIT 60;

-- Hourly throughput (last 24 hours)
SELECT
    DATE_TRUNC('hour', created_at) as hour,
    COUNT(*) as total,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
FROM notifications
WHERE created_at > NOW() - INTERVAL '24 hours'
GROUP BY DATE_TRUNC('hour', created_at)
ORDER BY hour DESC;

-- ============================================
-- 9. DATA INTEGRITY CHECKS
-- ============================================

-- Check for required fields
SELECT
    'Required Fields Check' as test,
    COUNT(*) as missing_required_fields
FROM notifications
WHERE notification_id IS NULL
    OR title IS NULL
    OR status IS NULL;

-- Check for orphaned processing records
SELECT
    'Orphaned Processing Check' as test,
    COUNT(*) as orphaned_count
FROM notifications
WHERE processing_worker_id IS NOT NULL
    AND status NOT IN ('processing', 'completed', 'failed');

-- Check for invalid status values
SELECT
    'Invalid Status Check' as test,
    COUNT(*) as invalid_status_count
FROM notifications
WHERE status NOT IN ('pending', 'processing', 'completed', 'failed');

-- ============================================
-- 10. RECENT ACTIVITY
-- ============================================

-- Last 10 notifications
SELECT
    notification_id,
    title,
    status,
    processing_worker_id,
    processing_attempts,
    created_at,
    updated_at
FROM notifications
ORDER BY created_at DESC
LIMIT 10;

-- Recent failures
SELECT
    notification_id,
    title,
    error_message,
    processing_attempts,
    processing_worker_id,
    updated_at
FROM notifications
WHERE status = 'failed'
    AND updated_at > NOW() - INTERVAL '1 hour'
ORDER BY updated_at DESC
LIMIT 10;

-- ============================================
-- 11. SUMMARY REPORT
-- ============================================

-- Overall system health summary
WITH stats AS (
    SELECT
        COUNT(*) as total_notifications,
        COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
        COUNT(CASE WHEN status = 'processing' THEN 1 END) as processing,
        COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed,
        COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
        COUNT(DISTINCT processing_worker_id) as active_workers,
        AVG(processing_attempts) as avg_attempts
    FROM notifications
)
SELECT
    'Total Notifications' as metric, total_notifications::text as value FROM stats
UNION ALL
SELECT 'Pending', pending::text FROM stats
UNION ALL
SELECT 'Processing', processing::text FROM stats
UNION ALL
SELECT 'Completed', completed::text FROM stats
UNION ALL
SELECT 'Failed', failed::text FROM stats
UNION ALL
SELECT 'Active Workers', active_workers::text FROM stats
UNION ALL
SELECT 'Average Attempts', ROUND(avg_attempts, 2)::text FROM stats
UNION ALL
SELECT 'Success Rate (%)',
    CASE
        WHEN total_notifications > 0
        THEN ROUND(completed * 100.0 / total_notifications, 2)::text
        ELSE 'N/A'
    END
FROM stats;