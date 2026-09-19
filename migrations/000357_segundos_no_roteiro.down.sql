-- Operações cadastradas em segundos passam para minutos, convertendo os tempos
-- para não mudarem de significado ao perder a unidade.
UPDATE operations SET
    setup_time = setup_time / 60, run_time = run_time / 60, labor_time = labor_time / 60,
    queue_time = queue_time / 60, wait_time = wait_time / 60, move_time = move_time / 60,
    standard_time = standard_time / 60, time_unit = 'MIN'
WHERE time_unit = 'SEGUNDO';

UPDATE route_operations SET
    setup_time = setup_time / 60, run_time = run_time / 60, labor_time = labor_time / 60,
    queue_time = queue_time / 60, wait_time = wait_time / 60, move_time = move_time / 60,
    standard_time = standard_time / 60, time_unit = 'MIN'
WHERE time_unit = 'SEGUNDO';

ALTER TABLE operations DROP CONSTRAINT IF EXISTS chk_operations_time_unit;
ALTER TABLE operations ADD CONSTRAINT chk_operations_time_unit
    CHECK (time_unit IN ('MIN', 'HORA', 'DIA'));

ALTER TABLE route_operations DROP CONSTRAINT IF EXISTS chk_route_ops_time_unit;
ALTER TABLE route_operations ADD CONSTRAINT chk_route_ops_time_unit
    CHECK (time_unit IS NULL OR time_unit IN ('MIN', 'HORA', 'DIA'));
