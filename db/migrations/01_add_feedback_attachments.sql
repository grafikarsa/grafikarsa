-- ============================================================================
-- MIGRATION: Add Feedback Attachments Support
-- ============================================================================

-- Add attachment_url column to feedback table
ALTER TABLE feedback 
ADD COLUMN attachment_url TEXT;

COMMENT ON COLUMN feedback.attachment_url IS 'URL file attachment (screenshot, dokumen, dll) untuk feedback';

-- Create index for faster queries on feedback with attachments
CREATE INDEX idx_feedback_has_attachment ON feedback(attachment_url) WHERE attachment_url IS NOT NULL;
