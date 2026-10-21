-- +goose Up
ALTER TABLE product_inventories
    ADD COLUMN locked_quantity BIGINT NOT NULL DEFAULT 0 AFTER stock_quantity,
    ADD CONSTRAINT chk_product_inventories_locked_quantity_non_negative CHECK (locked_quantity >= 0);

-- +goose Down
ALTER TABLE product_inventories
    DROP CHECK chk_product_inventories_locked_quantity_non_negative,
    DROP COLUMN locked_quantity;
