// Internationalization support
const i18n = {
    zh: {
        // Navigation
        'dashboard': '仪表板',
        'keys': '密钥管理',
        'scanner': '扫描器',
        'queries': '查询规则',
        'settings': '设置',
        'logs': '日志',
        'connected': '已连接',
        'disconnected': '已断开',

        // Dashboard
        'valid_keys': '有效密钥',
        'rate_limited': '限流密钥',
        'files_scanned': '已扫描文件',
        'scan_status': '扫描状态',
        'idle': '空闲',
        'scanning': '扫描中',
        'key_discovery_over_time': '密钥发现趋势',
        'scanner_progress': '扫描进度',
        'queries': '查询',
        'current_query': '当前查询',
        'processed_files': '已处理文件',
        'error_count': '错误数量',
        'recently_discovered_keys': '最近发现的密钥',
        'repository': '仓库',
        'file': '文件',
        'status': '状态',
        'discovered_at': '发现时间',
        'no_keys_found_yet': '暂未发现密钥',

        // Keys Management
        'api_keys_management': 'API密钥管理',
        'valid_keys_tab': '有效密钥',
        'rate_limited_keys_tab': '限流密钥',
        'search_repository': '搜索仓库...',
        'all_sources': '全部来源',
        'search': '搜索',
        'key': '密钥',
        'provider': '提供商',
        'tier': '层级',
        'actions': '操作',
        'validated_at': '验证时间',
        'reason': '原因',
        'created_at': '创建时间',
        'no_valid_keys_found': '未找到有效密钥',
        'no_rate_limited_keys_found': '未找到限流密钥',
        'loading': '加载中...',
        'unknown': '未知',
        'free': '免费',
        'paid': '付费',

        // Scanner Control
        'scanner_control': '扫描器控制',
        'start_scan': '开始扫描',
        'stop_scan': '停止扫描',
        'refresh': '刷新',
        'ready': '就绪',
        'configuration': '配置',
        'worker_count': '工作线程数',
        'scan_interval_seconds': '扫描间隔（秒）',
        'date_range_days': '日期范围（天）',
        'save_config': '保存配置',

        // Query Rules
        'query_rules_editor': '查询规则编辑器',
        'load_current': '加载当前',
        'save_rules': '保存规则',
        'validate': '验证',
        'enhanced_4_phase_query_strategy': '增强的四阶段查询策略',
        'queries_loaded': '已加载查询',
        'query_phases': '查询阶段',
        'phase_1': '阶段一',
        'core_api_detection': '核心API检测',
        'phase_2': '阶段二', 
        'contextual_targeting': '上下文目标',
        'phase_3': '阶段三',
        'behavioral_forensics': '行为取证',
        'phase_4': '阶段四',
        'global_syntax_matrix': '全局语法矩阵',
        'quick_actions': '快速操作',
        'add_custom_query': '添加自定义查询',
        'reset_to_default': '重置为默认',
        'export_rules': '导出规则',
        'query_rules_guide': '查询规则指南',
        'query_rules_guide_text': '使用GitHub搜索语法。阶段按顺序执行（1-4）。更高优先级阶段优先运行以更好地管理API配额。',

        // Settings
        'validator_settings': '验证器设置',
        'validation_model': '验证模型',
        'tier_detection_model': '层级检测模型', 
        'validator_worker_count': '验证器工作线程数',
        'validation_timeout_seconds': '验证超时（秒）',
        'enable_tier_detection': '启用层级检测',
        'enable_tier_detection_desc': '自动检测Gemini密钥是免费还是付费账户',

        'scanner_settings': '扫描器设置',
        'scanner_worker_count': '扫描器工作线程数',
        'batch_size': '批处理大小',
        'auto_start_scanning': '自动启动扫描',

        'security_notifications': '安全通知',
        'enable_security_notifications': '启用安全通知',
        'auto_create_github_issues': '自动创建GitHub问题',
        'notification_severity_threshold': '通知严重级别阈值',
        'all_severities': '所有级别',
        'high_critical_only': '仅高级和关键',
        'critical_only': '仅关键',
        'dry_run_mode': '干运行模式',
        'dry_run_mode_desc': '测试模式，不创建实际问题',
        'security_notice': '安全提醒',
        'security_notice_text': '启用后，此工具将在发现泄露的API密钥的仓库中自动创建GitHub问题。请确保您的GitHub令牌具有必要的权限。',

        'rate_limiting': '速率限制',
        'enable_rate_limiting': '启用速率限制',
        'requests_per_minute': '每分钟请求数',
        'burst_size': '突发大小',
        'enable_adaptive_rate_limiting': '启用自适应速率限制',

        'key_filtering': '密钥过滤',
        'prioritize_paid_keys': '优先使用付费密钥',
        'prioritize_paid_keys_desc': '当有多个可用密钥时，优先选择付费密钥而不是免费密钥',
        'provider_filter': '提供商过滤',
        'all_providers': '所有提供商',
        'gemini_only': '仅Gemini',
        'openai_only': '仅OpenAI',
        'claude_only': '仅Claude',
        'tier_filter': '层级过滤',
        'all_tiers': '所有层级',
        'paid_only': '仅付费',
        'free_only': '仅免费',
        'unknown_only': '仅未知',

        'save_all_settings': '保存所有设置',
        'load_current_settings': '加载当前设置',
        'reset_to_defaults': '重置为默认值',

        // Security Review
        'security_review': '安全审核',
        'pending_review': '待审核',
        'human_review_notice': '人工审核说明',
        'human_review_desc': '为遵守GitHub反机器人政策，所有issue创建需人工审核确认。请仔细检查每个发现的真实性。',
        'pending': '待审核',
        'approved': '已批准', 
        'rejected': '已拒绝',
        'created': '已创建',
        'all_status': '全部状态',
        'critical': '关键',
        'high': '高级',
        'medium': '中级',
        'file_path': '文件路径',
        'loading_issues': '加载审核项目中...',
        'review_security_issue': '审核安全问题',
        'approve': '批准',
        'reject': '拒绝',
        'review_note': '审核备注',
        'reviewer': '审核人员',
        'submit_review': '提交审核',

        // Logs
        'system_logs': '系统日志',
        'clear': '清空',
        'all_levels': '所有级别',
        'debug': '调试',
        'info': '信息',
        'warn': '警告',
        'error': '错误',
        'loading_logs': '加载日志中...',

        // Common
        'previous': '上一页',
        'next': '下一页',
        'delete': '删除',
        'confirm_delete': '确定要删除这个密钥吗？',
        'success': '成功',
        'failed': '失败',
        'cancel': '取消',
        'ok': '确定',
        'close': '关闭',

        // Messages
        'scan_started_successfully': '扫描启动成功',
        'scan_stopped_successfully': '扫描停止成功',
        'key_deleted_successfully': '密钥删除成功',
        'settings_saved_successfully': '设置保存成功',
        'failed_to_start_scan': '启动扫描失败',
        'failed_to_stop_scan': '停止扫描失败', 
        'failed_to_delete_key': '删除密钥失败',
        'failed_to_save_settings': '保存设置失败',
        'failed_to_load_settings': '加载设置失败',
        'query_rules_loaded_successfully': '查询规则加载成功',
        'query_rules_saved_successfully': '查询规则保存成功',
        'failed_to_load_query_rules': '加载查询规则失败',
        'failed_to_save_query_rules': '保存查询规则失败',
        'query_rules_exported_successfully': '查询规则导出成功',
        'reset_to_default_query_rules': '重置为默认查询规则',
        'failed_to_reset_query_rules': '重置查询规则失败',
        'settings_reset_to_defaults': '设置已重置为默认值',
        'config_refresh_not_implemented': '配置刷新功能尚未实现',
        'config_save_not_implemented': '配置保存功能尚未实现',
        'log_clear_not_implemented': '日志清理功能尚未实现'
    },
    en: {
        // Keep existing English texts as fallback
        'dashboard': 'Dashboard',
        'keys': 'Keys',
        'scanner': 'Scanner',
        'queries': 'Query Rules',
        'settings': 'Settings',
        'logs': 'Logs',
        'connected': 'Connected',
        'disconnected': 'Disconnected',
        // ... (keep all existing English texts)
    }
};

// Current language
let currentLanguage = localStorage.getItem('language') || 'zh';

// Translation function
function t(key) {
    return i18n[currentLanguage] && i18n[currentLanguage][key] || i18n.en[key] || key;
}

// Switch language function
function switchLanguage(lang) {
    currentLanguage = lang;
    localStorage.setItem('language', lang);
    updatePageTexts();
}

// Update all page texts
function updatePageTexts() {
    // Update navigation
    document.querySelectorAll('[data-i18n]').forEach(element => {
        const key = element.getAttribute('data-i18n');
        const translated = t(key);
        
        if (element.tagName === 'INPUT' && element.type === 'text') {
            element.placeholder = translated;
        } else if (element.tagName === 'OPTION') {
            element.textContent = translated;
        } else {
            // Handle HTML content with icons
            if (element.innerHTML.includes('<i class=')) {
                const icon = element.querySelector('i');
                element.innerHTML = '';
                if (icon) element.appendChild(icon);
                element.innerHTML += ' ' + translated;
            } else {
                element.textContent = translated;
            }
        }
    });

    // Update placeholders
    document.querySelectorAll('[data-i18n-placeholder]').forEach(element => {
        const key = element.getAttribute('data-i18n-placeholder');
        element.placeholder = t(key);
    });

    // Update titles
    document.querySelectorAll('[data-i18n-title]').forEach(element => {
        const key = element.getAttribute('data-i18n-title');
        element.title = t(key);
    });
}