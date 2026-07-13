-- +goose Up
CREATE TABLE IF NOT EXISTS feedback_tickets (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT NULL,
    source VARCHAR(32) NOT NULL DEFAULT 'app',
    category VARCHAR(32) NOT NULL DEFAULT 'feedback',
    content TEXT NOT NULL,
    contact VARCHAR(128) NOT NULL DEFAULT '',
    status TINYINT NOT NULL DEFAULT 1,
    handler_id BIGINT NULL,
    process_result TEXT NOT NULL,
    processed_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_feedback_user_id (user_id),
    KEY idx_feedback_status_created_at (status, created_at),
    KEY idx_feedback_category (category),
    KEY idx_feedback_source (source)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS feedback_tickets;
