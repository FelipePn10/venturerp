DROP TRIGGER IF EXISTS atendido_da_op_pelo_consumo ON production_consumptions;
DROP FUNCTION IF EXISTS erp_atendido_da_op_pelo_consumo();
DROP FUNCTION IF EXISTS erp_recalcular_atendido_da_op(bigint, bigint);
