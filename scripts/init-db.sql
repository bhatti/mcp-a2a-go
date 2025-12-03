-- Initialize database with pgvector extension and multi-tenant schema

-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create tenants table
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT TRUE,
    settings JSONB DEFAULT '{}'::jsonb
);

-- Create documents table with tenant isolation
CREATE TABLE IF NOT EXISTS documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    metadata JSONB DEFAULT '{}'::jsonb,
    embedding vector(1536),  -- OpenAI ada-002 dimension
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

-- Create index on embeddings for fast similarity search
CREATE INDEX IF NOT EXISTS idx_documents_embedding ON documents USING ivfflat (embedding vector_cosine_ops)
WITH (lists = 100);

-- Create index on tenant_id for efficient filtering
CREATE INDEX IF NOT EXISTS idx_documents_tenant_id ON documents(tenant_id);

-- Create index on metadata for JSON queries
CREATE INDEX IF NOT EXISTS idx_documents_metadata ON documents USING gin(metadata);

-- Create full-text search index for BM25-like ranking
CREATE INDEX IF NOT EXISTS idx_documents_fulltext ON documents USING gin(to_tsvector('english', title || ' ' || content));

-- Enable Row-Level Security
ALTER TABLE documents ENABLE ROW LEVEL SECURITY;

-- Create RLS policy for tenant isolation
CREATE POLICY tenant_isolation_policy ON documents
    FOR ALL
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- Create usage tracking table for cost control
CREATE TABLE IF NOT EXISTS usage_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id VARCHAR(255),
    operation VARCHAR(50) NOT NULL,
    model VARCHAR(100),
    prompt_tokens INTEGER DEFAULT 0,
    completion_tokens INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    cost_usd DECIMAL(10, 6) DEFAULT 0,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index for usage queries
CREATE INDEX IF NOT EXISTS idx_usage_logs_tenant_user ON usage_logs(tenant_id, user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_usage_logs_created_at ON usage_logs(created_at DESC);

-- Insert demo tenants
INSERT INTO tenants (id, name, settings) VALUES
    ('11111111-1111-1111-1111-111111111111', 'acme-corp', '{"monthly_budget_usd": 1000, "rate_limit_per_minute": 100}'::jsonb),
    ('22222222-2222-2222-2222-222222222222', 'beta-inc', '{"monthly_budget_usd": 500, "rate_limit_per_minute": 50}'::jsonb),
    ('33333333-3333-3333-3333-333333333333', 'gamma-ltd', '{"monthly_budget_usd": 2000, "rate_limit_per_minute": 200}'::jsonb)
ON CONFLICT (name) DO NOTHING;

-- Insert sample documents for demo tenant (acme-corp)
INSERT INTO documents (tenant_id, title, content, metadata) VALUES
    (
        '11111111-1111-1111-1111-111111111111',
        'Q4 Security Policy',
        'All employees must use MFA for accessing production systems. Password rotation is required every 90 days. VPN access is mandatory for remote work. Security training is required annually. Report security incidents immediately to the security team.',
        '{"category": "security", "department": "engineering", "version": "2024.Q4", "tags": ["security", "policy", "mfa", "vpn"]}'::jsonb
    ),
    (
        '11111111-1111-1111-1111-111111111111',
        'Remote Work Guidelines',
        'Remote work is permitted with manager approval. Employees must maintain regular working hours and be available during core hours (10am-3pm local time). Use company-approved collaboration tools for meetings. Ensure secure home network setup.',
        '{"category": "hr", "department": "all", "version": "2024.1", "tags": ["remote", "work", "policy"]}'::jsonb
    ),
    (
        '11111111-1111-1111-1111-111111111111',
        'API Design Standards',
        'All APIs must follow RESTful principles. Use JSON for request/response payloads. Implement proper error handling with standard HTTP status codes. Version APIs using URL paths (e.g., /api/v1/). Document all endpoints using OpenAPI/Swagger.',
        '{"category": "engineering", "department": "engineering", "version": "2024.2", "tags": ["api", "rest", "standards"]}'::jsonb
    ),
    (
        '11111111-1111-1111-1111-111111111111',
        'Machine Learning Model Development',
        'Machine learning models must be versioned and tracked. Use experiment tracking tools like MLflow or Weights & Biases. Document model architecture, training data, and hyperparameters. Implement model monitoring in production.',
        '{"category": "ml", "department": "data-science", "version": "2024.1", "tags": ["machine-learning", "ml", "models", "ai"]}'::jsonb
    ),
    (
        '11111111-1111-1111-1111-111111111111',
        'Data Privacy and GDPR Compliance',
        'Handle customer data according to GDPR requirements. Implement data retention policies. Obtain explicit consent for data collection. Provide mechanisms for data export and deletion. Encrypt sensitive data at rest and in transit.',
        '{"category": "compliance", "department": "legal", "version": "2024.3", "tags": ["privacy", "gdpr", "compliance", "data"]}'::jsonb
    ),
    (
        '11111111-1111-1111-1111-111111111111',
        'Microservices Architecture Guidelines',
        'Design services around business capabilities. Each microservice should have its own database. Use API gateways for routing. Implement circuit breakers and retry logic. Use container orchestration like Kubernetes for deployment.',
        '{"category": "architecture", "department": "engineering", "version": "2024.2", "tags": ["microservices", "architecture", "kubernetes"]}'::jsonb
    ),
    (
        '11111111-1111-1111-1111-111111111111',
        'Code Review Best Practices',
        'All code must be reviewed before merging. Focus on logic errors, security issues, and code style. Keep pull requests small and focused. Respond to review comments within 24 hours. Use automated linting and testing.',
        '{"category": "process", "department": "engineering", "version": "2024.1", "tags": ["code-review", "process", "quality"]}'::jsonb
    ),
    (
        '11111111-1111-1111-1111-111111111111',
        'Database Performance Optimization',
        'Index frequently queried columns. Use connection pooling to manage database connections. Implement caching for read-heavy workloads. Optimize slow queries using EXPLAIN plans. Consider read replicas for scaling reads.',
        '{"category": "database", "department": "engineering", "version": "2024.2", "tags": ["database", "performance", "optimization"]}'::jsonb
    ),
    (
        '11111111-1111-1111-1111-111111111111',
        'Incident Response Procedures',
        'Immediately notify on-call engineer for production incidents. Create incident ticket with severity level. Communicate status updates every 30 minutes. Conduct post-incident review within 48 hours. Document lessons learned.',
        '{"category": "operations", "department": "sre", "version": "2024.3", "tags": ["incident", "operations", "sre"]}'::jsonb
    ),
    (
        '11111111-1111-1111-1111-111111111111',
        'AI Ethics and Responsible AI',
        'Ensure AI systems are fair, transparent, and accountable. Test models for bias across different demographic groups. Provide explanations for AI decisions when possible. Implement human oversight for high-stakes decisions. Regular audits of AI systems.',
        '{"category": "ai-ethics", "department": "ai", "version": "2024.1", "tags": ["ai", "ethics", "responsible-ai", "fairness"]}'::jsonb
    )
ON CONFLICT DO NOTHING;

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger to auto-update updated_at
CREATE TRIGGER update_documents_updated_at BEFORE UPDATE ON documents
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create application user with proper RLS enforcement
-- Note: mcp_user is created by Docker with POSTGRES_USER and has superuser privileges
-- We create a separate app_user for the application to use, which enforces RLS
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'app_user') THEN
        CREATE ROLE app_user WITH LOGIN PASSWORD 'mcp_password' NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
    END IF;
END
$$;

-- Grant permissions to app_user
GRANT CONNECT ON DATABASE mcp_db TO app_user;
GRANT USAGE ON SCHEMA public TO app_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO app_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO app_user;

-- Also grant to mcp_user for backward compatibility
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO mcp_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO mcp_user;

-- Set default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO app_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO app_user;
