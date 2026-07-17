-- +goose Up
CREATE TABLE IF NOT EXISTS operation_logs (
    id BIGINT NOT NULL AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    username VARCHAR(64) NOT NULL DEFAULT '',
    action VARCHAR(128) NOT NULL,
    method VARCHAR(16) NOT NULL,
    path VARCHAR(255) NOT NULL,
    route VARCHAR(255) NOT NULL,
    http_status INT NOT NULL,
    request_id CHAR(36) NOT NULL DEFAULT '',
    client_ip VARCHAR(64) NOT NULL DEFAULT '',
    user_agent VARCHAR(255) NOT NULL DEFAULT '',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    KEY idx_operation_logs_user_id_created_at (user_id, created_at),
    KEY idx_operation_logs_action_created_at (action, created_at),
    KEY idx_operation_logs_created_at (created_at)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_general_ci;

-- +goose Down
DROP TABLE IF EXISTS operation_logs;
