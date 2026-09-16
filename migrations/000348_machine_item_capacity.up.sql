ALTER TABLE machines ADD COLUMN available_hours_per_day NUMERIC(6,3);
ALTER TABLE machines ADD CONSTRAINT machines_available_hours_check
 CHECK (available_hours_per_day IS NULL OR available_hours_per_day > 0 AND available_hours_per_day <= 24);
ALTER TABLE item_machine_times ADD COLUMN efficiency_rate NUMERIC(8,5);
ALTER TABLE item_machine_times ADD COLUMN time_basis TEXT NOT NULL DEFAULT 'CYCLE';
ALTER TABLE item_machine_times ADD CONSTRAINT item_machine_efficiency_check
 CHECK (efficiency_rate IS NULL OR efficiency_rate > 0 AND efficiency_rate <= 1);
ALTER TABLE item_machine_times ADD CONSTRAINT item_machine_time_basis_check CHECK (time_basis IN ('CYCLE','PROPORTIONAL'));
COMMENT ON COLUMN machines.available_hours_per_day IS 'Horas disponíveis por dia sem calendário; nulo herda o centro de trabalho.';
COMMENT ON COLUMN item_machine_times.efficiency_rate IS 'Eficiência específica; nulo herda a máquina. Taxa real medida usa 1.';
COMMENT ON COLUMN item_machine_times.time_basis IS 'CYCLE arredonda ciclos; PROPORTIONAL representa taxa por quantidade (ex.: peças/h).';

CREATE TABLE mrp_machine_allocations (
 suggestion_code BIGINT PRIMARY KEY REFERENCES mrp_planned_suggestions(code) ON DELETE CASCADE,
 enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
 machine_id BIGINT NOT NULL REFERENCES machines(id),
 production_minutes NUMERIC(18,6) NOT NULL CHECK(production_minutes > 0),
 schedule_date DATE NOT NULL
);
CREATE INDEX mrp_machine_allocations_tenant ON mrp_machine_allocations(enterprise_id,machine_id,schedule_date);

ALTER TABLE mrp_machine_allocations ADD COLUMN scheduled_start TIMESTAMP, ADD COLUMN scheduled_end TIMESTAMP;
CREATE TABLE mrp_machine_allocation_slots (
 machine_id BIGINT NOT NULL REFERENCES machines(id),
 route_operation_id BIGINT REFERENCES route_operations(id),
 suggestion_code BIGINT NOT NULL REFERENCES mrp_machine_allocations(suggestion_code) ON DELETE CASCADE,
 starts_at TIMESTAMP NOT NULL,
 ends_at TIMESTAMP NOT NULL CHECK(ends_at>starts_at),
 PRIMARY KEY(suggestion_code,starts_at)
);
-- A consulta de ocupação filtra por máquina e faixa de tempo. Só com a chave
-- primária (suggestion_code,starts_at) o planejador não tinha como podar por
-- máquina e varria a tabela inteira a cada sugestão × etapa do roteiro. Com
-- 50 mil slots a janela de um ano caiu de 141 ms para 60 ms mesmo com todos os
-- slots na mesma máquina; com várias máquinas a poda é muito maior.
CREATE INDEX mrp_machine_allocation_slots_machine ON mrp_machine_allocation_slots(machine_id,starts_at,ends_at);
