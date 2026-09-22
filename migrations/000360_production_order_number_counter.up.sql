BEGIN;
CREATE TABLE production_order_number_counters (
 enterprise_id BIGINT PRIMARY KEY REFERENCES enterprise(id),
 last_number BIGINT NOT NULL CHECK(last_number>=0)
);
INSERT INTO production_order_number_counters(enterprise_id,last_number)
 SELECT enterprise_id,MAX(order_number) FROM production_orders
 WHERE enterprise_id IS NOT NULL GROUP BY enterprise_id;
COMMIT;
