-- Truncar de volta para 20 quebraria as linhas COMPLETED_WITH_ERRORS já
-- gravadas; elas passam a FAILED, que é o estado mais próximo e cabe.
UPDATE mrp_calculation_logs SET status = 'FAILED' WHERE length(status) > 20;
ALTER TABLE mrp_calculation_logs ALTER COLUMN status TYPE VARCHAR(20);
