BEGIN;

UPDATE machines m SET enterprise_id=ue.enterprise_id FROM user_enterprises ue
WHERE m.enterprise_id IS NULL AND ue.user_id=m.created_by
  AND (SELECT COUNT(*) FROM user_enterprises x WHERE x.user_id=m.created_by)=1;
UPDATE machine_types mt SET enterprise_id=ue.enterprise_id FROM user_enterprises ue
WHERE mt.enterprise_id IS NULL AND ue.user_id=mt.created_by
  AND (SELECT COUNT(*) FROM user_enterprises x WHERE x.user_id=mt.created_by)=1;

ALTER TABLE employees ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
UPDATE employees e SET enterprise_id=ue.enterprise_id FROM user_enterprises ue
WHERE e.enterprise_id IS NULL AND ue.user_id=e.created_by
  AND (SELECT COUNT(*) FROM user_enterprises x WHERE x.user_id=e.created_by)=1;
CREATE UNIQUE INDEX IF NOT EXISTS uq_employees_tenant_code ON employees(enterprise_id,code) WHERE enterprise_id IS NOT NULL;

CREATE TABLE employee_contacts (
 id BIGSERIAL PRIMARY KEY, enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
 employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
 contact_type VARCHAR(10) NOT NULL CHECK(contact_type IN ('PHONE','EMAIL')),
 value VARCHAR(254) NOT NULL, is_primary BOOLEAN NOT NULL DEFAULT FALSE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE(enterprise_id,employee_id,contact_type,value)
);
CREATE TABLE employee_functions (
 id BIGSERIAL PRIMARY KEY, enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
 employee_id BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
 function_name VARCHAR(100) NOT NULL, cost_center_id BIGINT REFERENCES cost_centers(id),
 is_supervisor BOOLEAN NOT NULL DEFAULT FALSE, is_manager BOOLEAN NOT NULL DEFAULT FALSE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE(enterprise_id,employee_id,function_name,cost_center_id)
);
CREATE UNIQUE INDEX uq_employee_cost_center_supervisor ON employee_functions(enterprise_id,cost_center_id) WHERE is_supervisor;
CREATE UNIQUE INDEX uq_employee_cost_center_manager ON employee_functions(enterprise_id,cost_center_id) WHERE is_manager;
CREATE TABLE employee_credit_limits (
 enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
 employee_id BIGINT PRIMARY KEY REFERENCES employees(id) ON DELETE CASCADE,
 credit_limit NUMERIC(18,4) NOT NULL DEFAULT 0 CHECK(credit_limit>=0), valid_until DATE, updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE machines
 ADD COLUMN IF NOT EXISTS usage_description TEXT,
 ADD COLUMN IF NOT EXISTS acquired_on DATE,
 ADD COLUMN IF NOT EXISTS preparation_time NUMERIC(12,4) NOT NULL DEFAULT 0 CHECK(preparation_time>=0),
 ADD COLUMN IF NOT EXISTS preparation_time_unit VARCHAR(10) NOT NULL DEFAULT 'MINUTE' CHECK(preparation_time_unit IN ('MINUTE','HOUR')),
 ADD COLUMN IF NOT EXISTS supplier_code BIGINT,
 ADD COLUMN IF NOT EXISTS brand VARCHAR(100),
 ADD COLUMN IF NOT EXISTS is_preferred BOOLEAN NOT NULL DEFAULT FALSE,
 ADD COLUMN IF NOT EXISTS maintenance_responsible_employee_id BIGINT REFERENCES employees(id);

CREATE TABLE preventive_services (
 id BIGSERIAL PRIMARY KEY, enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
 code VARCHAR(30) NOT NULL, description VARCHAR(200) NOT NULL,
 service_type VARCHAR(20) NOT NULL CHECK(service_type IN ('ELECTRICAL','MECHANICAL','BOTH')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE(enterprise_id,code)
);
CREATE TABLE machine_preventive_services (
 id BIGSERIAL PRIMARY KEY, enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
 machine_id BIGINT NOT NULL REFERENCES machines(id) ON DELETE CASCADE,
 service_id BIGINT NOT NULL REFERENCES preventive_services(id), frequency_value INT NOT NULL CHECK(frequency_value>0),
 frequency_unit VARCHAR(10) NOT NULL CHECK(frequency_unit IN ('DAY','WEEK','MONTH','YEAR','UNIT')),
 max_tolerance INT NOT NULL DEFAULT 0 CHECK(max_tolerance>=0), supplier_code BIGINT,
 implemented_on DATE NOT NULL, last_executed_on DATE, started_at TIMESTAMPTZ, finished_at TIMESTAMPTZ, notes TEXT,
 UNIQUE(enterprise_id,machine_id,service_id)
);
CREATE TABLE machine_service_items (
 id BIGSERIAL PRIMARY KEY, enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
 machine_service_id BIGINT NOT NULL REFERENCES machine_preventive_services(id) ON DELETE CASCADE,
 item_code BIGINT NOT NULL, quantity NUMERIC(18,6) NOT NULL CHECK(quantity>0), notes TEXT
);
CREATE TABLE machine_service_responsibles (
 machine_service_id BIGINT NOT NULL REFERENCES machine_preventive_services(id) ON DELETE CASCADE,
 employee_id BIGINT NOT NULL REFERENCES employees(id), enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
 PRIMARY KEY(machine_service_id,employee_id)
);
CREATE TABLE machine_special_fields (
 id BIGSERIAL PRIMARY KEY, enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
 name VARCHAR(100) NOT NULL, value_type VARCHAR(10) NOT NULL CHECK(value_type IN ('TEXT','NUMBER')),
 max_length INT CHECK(max_length IS NULL OR max_length>0), UNIQUE(enterprise_id,name)
);
CREATE TABLE machine_special_values (
 machine_id BIGINT NOT NULL REFERENCES machines(id) ON DELETE CASCADE,
 field_id BIGINT NOT NULL REFERENCES machine_special_fields(id) ON DELETE CASCADE,
 enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
 text_value TEXT, numeric_value NUMERIC(18,6), PRIMARY KEY(machine_id,field_id),
 CHECK((text_value IS NULL)<>(numeric_value IS NULL))
);

CREATE TABLE machine_downtimes (
 id BIGSERIAL PRIMARY KEY, enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
 machine_id BIGINT NOT NULL REFERENCES machines(id) ON DELETE CASCADE,
 starts_at TIMESTAMPTZ NOT NULL, ends_at TIMESTAMPTZ NOT NULL,
 downtime_type VARCHAR(20) NOT NULL CHECK(downtime_type IN ('PLANNED','UNPLANNED','MAINTENANCE')),
 reason TEXT NOT NULL, maintenance_order_id BIGINT REFERENCES maintenance_orders(id),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), CHECK(ends_at>starts_at)
);
CREATE INDEX idx_machine_downtimes_tenant_range ON machine_downtimes(enterprise_id,machine_id,starts_at,ends_at);
COMMIT;
