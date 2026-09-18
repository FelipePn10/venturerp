-- O MRP não conseguia registrar que terminou com ressalvas.
--
-- `mrp_calculation_logs.status` aceita 20 caracteres e o código grava
-- "COMPLETED_WITH_ERRORS", que tem 21. O efeito é o pior possível: o cálculo
-- roda inteiro, grava as sugestões, e falha exatamente na hora de encerrar o
-- próprio registro. Como o encerramento é o que libera a trava de concorrência,
-- o plano fica RODANDO para sempre e toda tentativa seguinte de planejar
-- responde "já existe um cálculo em andamento".
--
-- Ou seja: bastava o MRP encontrar UMA ressalva — um item sem centro de
-- trabalho, uma produtividade faltando — para o planejamento do plano travar de
-- vez, sem nenhuma mensagem que ligasse uma coisa à outra.
--
-- 40 caracteres deixam folga para estados futuros sem repetir o problema.

ALTER TABLE mrp_calculation_logs
    ALTER COLUMN status TYPE VARCHAR(40);

COMMENT ON COLUMN mrp_calculation_logs.status IS
    'RUNNING | COMPLETED | COMPLETED_WITH_ERRORS | FAILED. O encerramento deste registro é o que libera a trava de um cálculo por plano.';
