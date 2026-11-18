-- ClipBridge 多平台媒体解析服务：初始业务表，设计稿 2026-11-18。
-- 目标：MySQL 8.0+。当前本地实例为 8.0.17，项目数据库为 clipbridge。
-- 先创建/选定 clipbridge 并核对同名表，再执行本脚本；本文件不负责创建数据库。
-- 不创建数据库、不插入默认账户、不删除已有数据。
-- DDL 可能隐式提交；部分成功后须核对，不能假定整份文件可事务回滚。

SET NAMES utf8mb4;
SET time_zone = '+00:00';

CREATE TABLE cb_admin_users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    username VARCHAR(64) NOT NULL,
    display_name VARCHAR(128) NOT NULL DEFAULT '',
    password_hash VARCHAR(255) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    status VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT 'active',
    last_login_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_cb_admin_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE cb_admin_sessions (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    admin_id BIGINT UNSIGNED NOT NULL,
    token_hash BINARY(32) NOT NULL,
    csrf_hash BINARY(32) NOT NULL,
    expires_at DATETIME(3) NOT NULL,
    last_seen_at DATETIME(3) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_cb_session_token (token_hash),
    KEY idx_cb_session_admin (admin_id),
    KEY idx_cb_session_expiry (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE cb_api_keys (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(128) NOT NULL,
    channel VARCHAR(8) CHARACTER SET ascii COLLATE ascii_bin NOT NULL COMMENT 'rest or mcp; validated by application',
    key_hash BINARY(32) NOT NULL,
    key_hint VARCHAR(48) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    scopes JSON NOT NULL,
    platform_allowlist JSON NOT NULL,
    rate_limit_per_minute INT UNSIGNED NOT NULL DEFAULT 60 COMMENT '0: no key-specific rate limit',
    daily_parse_limit BIGINT UNSIGNED NOT NULL DEFAULT 1000 COMMENT '0: no key-specific daily limit',
    max_concurrency INT UNSIGNED NOT NULL DEFAULT 2,
    status VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT 'active',
    version INT UNSIGNED NOT NULL DEFAULT 1,
    expires_at DATETIME(3) NULL,
    revoked_at DATETIME(3) NULL,
    last_used_at DATETIME(3) NULL,
    created_by BIGINT UNSIGNED NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_cb_key_hash (key_hash),
    KEY idx_cb_key_channel_status (channel, status),
    KEY idx_cb_key_creator (created_by),
    KEY idx_cb_key_expiry (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE cb_platform_configs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    platform VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    enabled TINYINT UNSIGNED NOT NULL DEFAULT 0,
    config_json JSON NULL,
    credentials_ciphertext MEDIUMBLOB NULL,
    secret_key_id VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NULL,
    config_version INT UNSIGNED NOT NULL DEFAULT 1,
    last_test_status VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NULL,
    last_test_error_code VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
    last_tested_at DATETIME(3) NULL,
    updated_by BIGINT UNSIGNED NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_cb_platform_code (platform)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE cb_system_settings (
    setting_key VARCHAR(96) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    setting_value JSON NOT NULL,
    version INT UNSIGNED NOT NULL DEFAULT 1,
    updated_by BIGINT UNSIGNED NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (setting_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE cb_quota_daily (
    api_key_id BIGINT UNSIGNED NOT NULL,
    quota_date DATE NOT NULL COMMENT 'Asia/Shanghai business date, computed by application',
    used_units BIGINT UNSIGNED NOT NULL DEFAULT 0,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (api_key_id, quota_date),
    KEY idx_cb_quota_date (quota_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE cb_request_logs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    request_id CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    channel VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NOT NULL COMMENT 'rest, mcp, web, admin, proxy',
    api_key_id BIGINT UNSIGNED NULL,
    admin_id BIGINT UNSIGNED NULL,
    operation VARCHAR(48) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    accepted_units INT UNSIGNED NOT NULL DEFAULT 0,
    success_items INT UNSIGNED NOT NULL DEFAULT 0,
    failed_items INT UNSIGNED NOT NULL DEFAULT 0,
    http_status SMALLINT UNSIGNED NULL,
    error_code VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
    duration_ms BIGINT UNSIGNED NOT NULL DEFAULT 0,
    proxy_bytes_sent BIGINT UNSIGNED NOT NULL DEFAULT 0,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_cb_request_id (request_id),
    KEY idx_cb_request_key_time (api_key_id, created_at),
    KEY idx_cb_request_channel_time (channel, created_at),
    KEY idx_cb_request_error_time (error_code, created_at),
    KEY idx_cb_request_time (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE cb_parse_records (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    request_id CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    item_index INT UNSIGNED NOT NULL DEFAULT 0,
    platform VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NULL,
    item_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NULL,
    input_hash BINARY(32) NOT NULL,
    source_url_redacted TEXT NULL,
    media_type VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NULL,
    status VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    cache_hit TINYINT UNSIGNED NOT NULL DEFAULT 0,
    result_summary JSON NULL COMMENT 'Allowlisted metadata only; no signed media URLs or credentials',
    error_code VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NULL,
    error_message VARCHAR(255) NULL,
    duration_ms BIGINT UNSIGNED NOT NULL DEFAULT 0,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_cb_parse_request_item (request_id, item_index),
    KEY idx_cb_parse_platform_time (platform, created_at),
    KEY idx_cb_parse_item (platform, item_id),
    KEY idx_cb_parse_time (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE cb_audit_logs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    actor_admin_id BIGINT UNSIGNED NULL COMMENT 'NULL for explicitly identified bootstrap/system actions',
    action VARCHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    target_type VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    target_id VARCHAR(128) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    request_id CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NULL,
    details_json JSON NULL COMMENT 'Redacted change summary; never secret values',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_cb_audit_actor_time (actor_admin_id, created_at),
    KEY idx_cb_audit_target_time (target_type, target_id, created_at),
    KEY idx_cb_audit_time (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
