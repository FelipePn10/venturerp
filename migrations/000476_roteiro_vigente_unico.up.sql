-- Uma regra só para "qual roteiro vale para este item hoje".
--
-- Antes cada rotina escolhia o roteiro do seu jeito: a OP pelo vigente da
-- máscara exata; a ordem de serviço de terceiro, a liberação da OP e o serviço
-- de terceiros por "qualquer roteiro padrão do item", sem empresa e sem
-- vigência; o rateio por centro por qualquer roteiro ativo. Num mesmo item,
-- a OP explodia um roteiro e a ordem de serviço nascia de outro.
--
-- A regra (igual a GetStandardRouteForItem em queries/routing.sql):
--   * da empresa, ativo, APROVADA e vigente na data;
--   * o roteiro da máscara pedida vence; sem ele, vale o genérico (sem máscara);
--     roteiro de OUTRA máscara nunca serve;
--   * entre os que sobram: padrão primeiro, depois a vigência mais recente,
--     depois a menor alternativa.
CREATE OR REPLACE FUNCTION erp_roteiro_vigente(
    p_enterprise_id BIGINT,
    p_item_code     BIGINT,
    p_mask          TEXT,
    p_data          DATE
) RETURNS BIGINT
LANGUAGE sql STABLE AS $$
    SELECT mr.id
    FROM manufacturing_routes mr
    WHERE mr.enterprise_id = p_enterprise_id
      AND mr.item_code = p_item_code
      AND mr.is_active
      AND mr.situation = 'APROVADA'
      AND (COALESCE(mr.mask, '') = COALESCE(p_mask, '') OR COALESCE(mr.mask, '') = '')
      AND (mr.valid_from IS NULL OR mr.valid_from <= COALESCE(p_data, CURRENT_DATE))
      AND (mr.valid_to   IS NULL OR mr.valid_to   >= COALESCE(p_data, CURRENT_DATE))
    ORDER BY (COALESCE(mr.mask, '') = COALESCE(p_mask, '')) DESC,
             mr.is_standard DESC, mr.valid_from DESC NULLS LAST, mr.alternative
    LIMIT 1
$$;

COMMENT ON FUNCTION erp_roteiro_vigente(BIGINT, BIGINT, TEXT, DATE) IS
    'Roteiro que vale para o item/máscara na data: máscara exata vence, genérico é a alternativa; padrão, vigência mais recente, menor alternativa.';

-- Dependências que sobraram de etapas já removidas: a leitura passou a
-- ignorá-las, mas elas continuavam fazendo a OP acreditar que o roteiro "tinha
-- rede" e deixar de encadear as etapas pela sequência. Antes de apagar, liga
-- quem vinha antes da etapa removida a quem vinha depois (A→B→C sem B = A→C),
-- que é o que a remoção passou a fazer.
INSERT INTO route_operation_network (predecessor_id, successor_id, overlap_pct)
SELECT DISTINCT antes.predecessor_id, depois.successor_id, 0
FROM route_operation_network antes
JOIN route_operations removida ON removida.id = antes.successor_id AND NOT removida.is_active
JOIN route_operation_network depois ON depois.predecessor_id = removida.id
JOIN route_operations a ON a.id = antes.predecessor_id AND a.is_active
JOIN route_operations c ON c.id = depois.successor_id AND c.is_active
WHERE a.id <> c.id AND a.route_id = c.route_id
ON CONFLICT (predecessor_id, successor_id) DO NOTHING;

DELETE FROM route_operation_network ron
WHERE EXISTS (SELECT 1 FROM route_operations ro WHERE ro.id = ron.predecessor_id AND NOT ro.is_active)
   OR EXISTS (SELECT 1 FROM route_operations ro WHERE ro.id = ron.successor_id AND NOT ro.is_active);

-- A sequência é única entre as etapas ATIVAS. A constraint antiga valia também
-- para as removidas (a remoção é lógica): tirar a etapa 20 e incluir outra na
-- sequência 20 devolvia erro de duplicidade, e o usuário não via etapa 20
-- nenhuma na tela.
ALTER TABLE route_operations DROP CONSTRAINT IF EXISTS route_operations_route_id_sequence_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_route_operations_sequencia_ativa
    ON route_operations (route_id, sequence) WHERE is_active;
