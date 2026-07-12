CREATE TABLE production_delivery_lines (
    id BIGSERIAL PRIMARY KEY,
    production_delivery_id BIGINT NOT NULL REFERENCES production_deliveries(id) ON DELETE CASCADE,
    movement_class VARCHAR(3) NOT NULL CHECK (movement_class IN ('EP','EPP','EPE')),
    quantity NUMERIC(18,6) NOT NULL CHECK (quantity > 0),
    UNIQUE (production_delivery_id, movement_class)
);

INSERT INTO production_delivery_lines (production_delivery_id, movement_class, quantity)
SELECT id, movement_class, quantity
FROM production_deliveries
WHERE quantity > 0;
