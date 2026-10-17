-- ==============================================================================
-- PEGASUS LOGISTICS OS: ENTERPRISE REQUESTS & EMAILS DATABASE SCHEMA (SUPABASE)
-- ==============================================================================

-- 1. Helper trigger function for automatic updated_at timestamping
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 2. Enterprise Requests Table (Demo requests, sales inquiries, pilots)
CREATE TABLE IF NOT EXISTS enterprise_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Contact Information
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    company TEXT,
    phone TEXT,
    
    -- Enterprise Logistics Context
    role TEXT DEFAULT 'Supplier Operations',
    fleet_size TEXT,
    inquiry_type TEXT NOT NULL DEFAULT 'demo', -- 'demo', 'sales', 'pilot', 'general', 'partnership'
    subject TEXT,
    message TEXT,
    
    -- Workflow Lifecycle Management
    status TEXT NOT NULL DEFAULT 'new', -- 'new', 'in_review', 'contacted', 'qualified', 'closed'
    priority TEXT NOT NULL DEFAULT 'normal', -- 'low', 'normal', 'high', 'urgent'
    admin_notes TEXT,
    
    -- Analytics & Lead Tracking
    source_page TEXT,
    utm_source TEXT,
    utm_medium TEXT,
    utm_campaign TEXT,
    metadata JSONB DEFAULT '{}'::jsonb,
    ip_hash TEXT,

    -- Constraints
    CONSTRAINT valid_email CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT valid_status CHECK (status IN ('new', 'in_review', 'contacted', 'qualified', 'closed')),
    CONSTRAINT valid_priority CHECK (priority IN ('low', 'normal', 'high', 'urgent'))
);

-- Triggers
DROP TRIGGER IF EXISTS set_enterprise_requests_updated_at ON enterprise_requests;
CREATE TRIGGER set_enterprise_requests_updated_at
    BEFORE UPDATE ON enterprise_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Indexes for lightning-fast queries and admin filtering
CREATE INDEX IF NOT EXISTS idx_enterprise_requests_email ON enterprise_requests(email);
CREATE INDEX IF NOT EXISTS idx_enterprise_requests_created_at ON enterprise_requests(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_enterprise_requests_status ON enterprise_requests(status);
CREATE INDEX IF NOT EXISTS idx_enterprise_requests_inquiry_type ON enterprise_requests(inquiry_type);

-- 3. Newsletter & Email Subscribers Table
CREATE TABLE IF NOT EXISTS newsletter_subscribers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    email TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'subscribed', -- 'subscribed', 'unsubscribed', 'bounced'
    source TEXT DEFAULT 'website_footer',
    metadata JSONB DEFAULT '{}'::jsonb,

    CONSTRAINT valid_subscriber_email CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    CONSTRAINT valid_subscriber_status CHECK (status IN ('subscribed', 'unsubscribed', 'bounced'))
);

DROP TRIGGER IF EXISTS set_newsletter_subscribers_updated_at ON newsletter_subscribers;
CREATE TRIGGER set_newsletter_subscribers_updated_at
    BEFORE UPDATE ON newsletter_subscribers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE INDEX IF NOT EXISTS idx_newsletter_subscribers_email ON newsletter_subscribers(email);
CREATE INDEX IF NOT EXISTS idx_newsletter_subscribers_status ON newsletter_subscribers(status);

-- 4. Email Audit Logs Table (Tracks automated dispatch & confirmation receipts)
CREATE TABLE IF NOT EXISTS email_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    request_id UUID REFERENCES enterprise_requests(id) ON DELETE SET NULL,
    recipient TEXT NOT NULL,
    subject TEXT NOT NULL,
    status TEXT NOT NULL, -- 'sent', 'queued', 'failed', 'skipped_no_key'
    provider TEXT NOT NULL DEFAULT 'resend',
    error_message TEXT,
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_email_audit_logs_request_id ON email_audit_logs(request_id);
CREATE INDEX IF NOT EXISTS idx_email_audit_logs_created_at ON email_audit_logs(created_at DESC);

-- ==============================================================================
-- ROW LEVEL SECURITY (RLS) POLICIES
-- ==============================================================================
ALTER TABLE enterprise_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE newsletter_subscribers ENABLE ROW LEVEL SECURITY;
ALTER TABLE email_audit_logs ENABLE ROW LEVEL SECURITY;

-- Allow public anonymous submissions (forms)
CREATE POLICY "Allow public insert to enterprise_requests"
    ON enterprise_requests
    FOR INSERT
    TO anon, authenticated
    WITH CHECK (
        length(trim(name)) > 0 AND
        length(trim(email)) > 0
    );

CREATE POLICY "Allow public insert to newsletter_subscribers"
    ON newsletter_subscribers
    FOR INSERT
    TO anon, authenticated
    WITH CHECK (
        length(trim(email)) > 0
    );

-- Allow service role full access for backend API routes & admin dashboards
CREATE POLICY "Allow service_role full access to enterprise_requests"
    ON enterprise_requests
    FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

CREATE POLICY "Allow service_role full access to newsletter_subscribers"
    ON newsletter_subscribers
    FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);

CREATE POLICY "Allow service_role full access to email_audit_logs"
    ON email_audit_logs
    FOR ALL
    TO service_role
    USING (true)
    WITH CHECK (true);
