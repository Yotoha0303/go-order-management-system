-- +goose Up
CREATE TABLE IF NOT EXISTS order_timeout_outbox (
    id BIGINT NOT NULL AUTO_INCREMENT,
    event_id CHAR(36) NOT NULL,
    order_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    timeout_at DATETIME(3) NOT NULL,
    published_at DATETIME(3) NULL,
    attempts INT NOT NULL DEFAULT 0,
    next_attempt_at DATETIME(3) NOT NULL,
    last_error VARCHAR(255) NOT NULL DEFAULT '',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_order_timeout_outbox_event_id (event_id),
    UNIQUE KEY uk_order_timeout_outbox_order_id (order_id),
    KEY idx_order_timeout_outbox_pending (published_at, next_attempt_at),
    CONSTRAINT fk_order_timeout_outbox_order FOREIGN KEY (order_id) REFERENCES orders(id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_general_ci;

-- +goose Down
DROP TABLE IF EXISTS order_timeout_outbox;
