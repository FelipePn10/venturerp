-- As dependências órfãs apagadas na subida não voltam: apontavam para etapas
-- removidas e não tinham efeito válido. A constraint antiga só volta se não
-- houver sequência repetida entre etapas ativas e removidas.
DROP INDEX IF EXISTS uq_route_operations_sequencia_ativa;
ALTER TABLE route_operations ADD CONSTRAINT route_operations_route_id_sequence_key UNIQUE (route_id, sequence);
DROP FUNCTION IF EXISTS erp_roteiro_vigente(BIGINT, BIGINT, TEXT, DATE);
