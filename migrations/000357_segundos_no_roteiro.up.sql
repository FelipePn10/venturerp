-- O roteiro passa a aceitar tempos em segundos.
--
-- A ficha técnica da fábrica vem em segundos: "corte a laser, setup 180 s,
-- 85 s por peça". A produtividade por máquina já aceitava SEGUNDO desde a
-- migração 351, mas a operação do roteiro só aceitava MIN, HORA ou DIA — então
-- o mesmo dado precisava ser convertido à mão para entrar aqui (85 ÷ 3600 =
-- 0,0236 h) e entrava direto ali.
--
-- Converter à mão não é só trabalhoso: é onde o erro entra. 0,0236 tem quatro
-- casas e qualquer arredondamento vira minutos perdidos num lote de 500. Pior,
-- o material de treinamento precisou de uma seção inteira só para ensinar a
-- divisão — sintoma de que o sistema estava pedindo ao usuário um trabalho que
-- é dele.

ALTER TABLE operations DROP CONSTRAINT IF EXISTS chk_operations_time_unit;
ALTER TABLE operations ADD CONSTRAINT chk_operations_time_unit
    CHECK (time_unit IN ('SEGUNDO', 'MIN', 'HORA', 'DIA'));

ALTER TABLE route_operations DROP CONSTRAINT IF EXISTS chk_route_ops_time_unit;
ALTER TABLE route_operations ADD CONSTRAINT chk_route_ops_time_unit
    CHECK (time_unit IS NULL OR time_unit IN ('SEGUNDO', 'MIN', 'HORA', 'DIA'));

COMMENT ON COLUMN operations.time_unit IS
    'Unidade em que os tempos desta operação foram digitados: SEGUNDO | MIN | HORA | DIA. A ficha de fábrica costuma vir em segundos.';
