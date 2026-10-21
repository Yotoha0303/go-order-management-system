-- +goose Up
CREATE TABLE IF NOT EXISTS order_timeout_dead_letters (
    id BIGINT NOT NULL AUTO_INCREMENT,
    message_id VARCHAR(64) NOT NULL DEFAULT '',
    event_id VARCHAR(64) NOT NULL DEFAULT '',
    order_id BIGINT NOT NULL DEFAULT 0,
    user_id BIGINT NOT NULL DEFAULT 0,
    retry_count INT NOT NULL DEFAULT 0,
    reason VARCHAR(64) NOT NULL DEFAULT '',
    payload TEXT NOT NULL,
    error_detail TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_ot_dead_letters_order_id (order_id),
    KEY idx_ot_dead_letters_event_id (event_id),
    KEY idx_ot_dead_letters_created_at (created_at),
    KEY idx_ot_dead_letters_status (status)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_general_ci;

-- +goose Down
DROP TABLE IF EXISTS order_timeout_dead_letters;
