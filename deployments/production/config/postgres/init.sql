-- 初始化数据库结构
-- 创建扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- 创建用户表 (用于Web界面认证)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    email VARCHAR(100),
    role VARCHAR(20) DEFAULT 'user' CHECK (role IN ('admin', 'user')),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建有效密钥表
CREATE TABLE IF NOT EXISTS valid_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key_value VARCHAR(255) NOT NULL,
    source VARCHAR(50) NOT NULL DEFAULT 'github',
    repo_name VARCHAR(255),
    file_path TEXT,
    file_url TEXT,
    sha VARCHAR(40),
    validated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP WITH TIME ZONE,
    usage_count INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建频率限制密钥表
CREATE TABLE IF NOT EXISTS rate_limited_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key_value VARCHAR(255) NOT NULL,
    source VARCHAR(50) NOT NULL DEFAULT 'github',
    repo_name VARCHAR(255),
    file_path TEXT,
    file_url TEXT,
    sha VARCHAR(40),
    reason VARCHAR(100) DEFAULT 'rate_limited',
    retry_after TIMESTAMP WITH TIME ZONE,
    retry_count INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建已扫描文件表
CREATE TABLE IF NOT EXISTS scanned_shas (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sha VARCHAR(40) UNIQUE NOT NULL,
    repo_name VARCHAR(255),
    file_path TEXT,
    scanned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    keys_found INTEGER DEFAULT 0,
    valid_keys INTEGER DEFAULT 0,
    file_size INTEGER,
    metadata JSONB DEFAULT '{}'
);

-- 创建处理过的查询表
CREATE TABLE IF NOT EXISTS processed_queries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    query_text TEXT NOT NULL,
    query_hash VARCHAR(64) UNIQUE NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    results_count INTEGER DEFAULT 0,
    processing_time_ms INTEGER,
    success BOOLEAN DEFAULT true,
    error_message TEXT,
    metadata JSONB DEFAULT '{}'
);

-- 创建扫描进度表
CREATE TABLE IF NOT EXISTS scan_progress (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    is_scanning BOOLEAN DEFAULT false,
    current_query TEXT,
    queries_processed INTEGER DEFAULT 0,
    total_queries INTEGER DEFAULT 0,
    files_processed INTEGER DEFAULT 0,
    valid_keys_found INTEGER DEFAULT 0,
    rate_limited_keys INTEGER DEFAULT 0,
    start_time TIMESTAMP WITH TIME ZONE,
    last_scan_time TIMESTAMP WITH TIME ZONE,
    estimated_completion TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建同步记录表
CREATE TABLE IF NOT EXISTS sync_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    service_name VARCHAR(50) NOT NULL,
    service_url VARCHAR(255),
    sync_type VARCHAR(20) CHECK (sync_type IN ('push', 'pull')),
    keys_count INTEGER DEFAULT 0,
    success BOOLEAN DEFAULT false,
    error_message TEXT,
    response_data JSONB,
    sync_duration_ms INTEGER,
    synced_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建系统配置表
CREATE TABLE IF NOT EXISTS system_config (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    config_key VARCHAR(100) UNIQUE NOT NULL,
    config_value TEXT,
    config_type VARCHAR(20) DEFAULT 'string' CHECK (config_type IN ('string', 'number', 'boolean', 'json')),
    description TEXT,
    is_secret BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建审计日志表
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50),
    resource_id VARCHAR(255),
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
-- valid_keys表索引
CREATE INDEX IF NOT EXISTS idx_valid_keys_key_value ON valid_keys(key_value);
CREATE INDEX IF NOT EXISTS idx_valid_keys_source ON valid_keys(source);
CREATE INDEX IF NOT EXISTS idx_valid_keys_repo_name ON valid_keys(repo_name);
CREATE INDEX IF NOT EXISTS idx_valid_keys_validated_at ON valid_keys(validated_at);
CREATE INDEX IF NOT EXISTS idx_valid_keys_is_active ON valid_keys(is_active);
CREATE INDEX IF NOT EXISTS idx_valid_keys_created_at ON valid_keys(created_at);

-- rate_limited_keys表索引
CREATE INDEX IF NOT EXISTS idx_rate_limited_keys_key_value ON rate_limited_keys(key_value);
CREATE INDEX IF NOT EXISTS idx_rate_limited_keys_source ON rate_limited_keys(source);
CREATE INDEX IF NOT EXISTS idx_rate_limited_keys_retry_after ON rate_limited_keys(retry_after);
CREATE INDEX IF NOT EXISTS idx_rate_limited_keys_created_at ON rate_limited_keys(created_at);

-- scanned_shas表索引
CREATE INDEX IF NOT EXISTS idx_scanned_shas_sha ON scanned_shas(sha);
CREATE INDEX IF NOT EXISTS idx_scanned_shas_repo_name ON scanned_shas(repo_name);
CREATE INDEX IF NOT EXISTS idx_scanned_shas_scanned_at ON scanned_shas(scanned_at);

-- processed_queries表索引
CREATE INDEX IF NOT EXISTS idx_processed_queries_hash ON processed_queries(query_hash);
CREATE INDEX IF NOT EXISTS idx_processed_queries_processed_at ON processed_queries(processed_at);

-- sync_records表索引
CREATE INDEX IF NOT EXISTS idx_sync_records_service_name ON sync_records(service_name);
CREATE INDEX IF NOT EXISTS idx_sync_records_synced_at ON sync_records(synced_at);
CREATE INDEX IF NOT EXISTS idx_sync_records_success ON sync_records(success);

-- audit_logs表索引
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);

-- 创建更新时间触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为相关表创建更新时间触发器
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_valid_keys_updated_at BEFORE UPDATE ON valid_keys FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_rate_limited_keys_updated_at BEFORE UPDATE ON rate_limited_keys FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_scan_progress_updated_at BEFORE UPDATE ON scan_progress FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_system_config_updated_at BEFORE UPDATE ON system_config FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 插入默认系统配置
INSERT INTO system_config (config_key, config_value, config_type, description, is_secret) VALUES
('scan_interval', '30m', 'string', 'Default scan interval', false),
('max_worker_count', '20', 'number', 'Maximum worker count', false),
('rate_limit_per_minute', '120', 'number', 'Rate limit per minute', false),
('enable_auto_sync', 'true', 'boolean', 'Enable automatic synchronization', false),
('maintenance_mode', 'false', 'boolean', 'System maintenance mode', false),
('version', '1.0.0', 'string', 'Application version', false)
ON CONFLICT (config_key) DO NOTHING;

-- 创建默认管理员用户 (密码: admin123)
-- 生产环境中应该修改默认密码
INSERT INTO users (username, password_hash, email, role, is_active) VALUES
('admin', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'admin@example.com', 'admin', true)
ON CONFLICT (username) DO NOTHING;

-- 创建统计视图
CREATE OR REPLACE VIEW stats_summary AS
SELECT 
    (SELECT COUNT(*) FROM valid_keys WHERE is_active = true) as active_valid_keys,
    (SELECT COUNT(*) FROM rate_limited_keys WHERE retry_after > CURRENT_TIMESTAMP) as pending_retry_keys,
    (SELECT COUNT(*) FROM scanned_shas) as total_scanned_files,
    (SELECT COUNT(*) FROM processed_queries WHERE success = true) as successful_queries,
    (SELECT COUNT(*) FROM processed_queries WHERE success = false) as failed_queries,
    (SELECT COALESCE(AVG(processing_time_ms), 0) FROM processed_queries WHERE success = true) as avg_query_time_ms,
    (SELECT COUNT(*) FROM sync_records WHERE synced_at >= CURRENT_TIMESTAMP - INTERVAL '24 hours' AND success = true) as successful_syncs_24h;

-- 创建性能优化
-- 分区表 (如果数据量大)
-- 这里以审计日志为例，可以按月分区
CREATE TABLE IF NOT EXISTS audit_logs_template (LIKE audit_logs INCLUDING ALL);

-- 创建清理函数 (清理旧数据)
CREATE OR REPLACE FUNCTION cleanup_old_data()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER := 0;
BEGIN
    -- 清理30天前的审计日志
    DELETE FROM audit_logs WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '30 days';
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    -- 清理90天前的同步记录
    DELETE FROM sync_records WHERE synced_at < CURRENT_TIMESTAMP - INTERVAL '90 days';
    GET DIAGNOSTICS deleted_count = deleted_count + ROW_COUNT;
    
    -- 清理无效的频率限制记录 (retry_after已过期且retry_count > 5)
    DELETE FROM rate_limited_keys 
    WHERE retry_after < CURRENT_TIMESTAMP - INTERVAL '7 days' 
    AND retry_count > 5;
    GET DIAGNOSTICS deleted_count = deleted_count + ROW_COUNT;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- 设置数据库参数 (用于性能优化)
-- 这些设置需要在postgresql.conf中配置，这里只是示例
COMMENT ON DATABASE kooix_hajimi IS 'Kooix Hajimi production database with optimized settings';

-- 完成消息
DO $$
BEGIN
    RAISE NOTICE 'Database initialization completed successfully!';
    RAISE NOTICE 'Default admin user created: username=admin, password=admin123';
    RAISE NOTICE 'Please change the default password in production!';
END $$;