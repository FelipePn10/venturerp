DROP TABLE IF EXISTS mrp_machine_allocation_slots;
DROP TABLE mrp_machine_allocations;
ALTER TABLE item_machine_times DROP COLUMN time_basis, DROP COLUMN efficiency_rate;
ALTER TABLE machines DROP COLUMN available_hours_per_day;
