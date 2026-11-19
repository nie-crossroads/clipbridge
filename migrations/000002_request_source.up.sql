ALTER TABLE cb_request_logs
    ADD COLUMN source_ip_redacted VARCHAR(48) CHARACTER SET ascii COLLATE ascii_bin NULL
    COMMENT 'Masked client network prefix; null for unavailable or older records',
    ALGORITHM=INSTANT;
