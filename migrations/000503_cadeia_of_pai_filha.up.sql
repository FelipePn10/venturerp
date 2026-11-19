-- Cadeia de fabricação: a OF do componente fabricado (bucha, tubo…) sabe para
-- qual OF pai ela produz e, por ela, de que pedido de venda a peça é.
--
-- Até aqui só a ordem do produto final ficava ligada ao pedido
-- (sales_order_planned_orders, migração 454): o pegging do componente aponta a
-- sugestão do pai, e o pegging é refeito a cada cálculo. Firmada a sugestão do
-- componente, a OF dele nascia solta — sem pedido, sem pai — e a peça pronta ia
-- para o estoque comum, onde a OF de outro pedido podia levá-la.
--
-- Ver .ai/tasks/cadeia-of-pai-filha.md.

-- Vínculo filho → pai, gravado ao firmar (de cima para baixo ou de baixo para
-- cima). O pai é a ordem planejada (a OF vem por production_orders.planned_order_id)
-- ou, quando o componente veio do empenho de uma OF sem ordem planejada, a OF.
CREATE TABLE production_chain_links (
    id                         BIGSERIAL     PRIMARY KEY,
    enterprise_id              BIGINT        NOT NULL REFERENCES enterprise(id),
    child_planned_order_id     BIGINT        NOT NULL REFERENCES planned_orders(id) ON DELETE CASCADE,
    parent_planned_order_id    BIGINT        REFERENCES planned_orders(id) ON DELETE CASCADE,
    parent_production_order_id BIGINT        REFERENCES production_orders(id) ON DELETE CASCADE,
    quantity                   NUMERIC(18,6) NOT NULL CHECK (quantity > 0),
    created_at                 TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_by                 UUID          NOT NULL,
    CONSTRAINT ck_production_chain_parent CHECK (parent_planned_order_id IS NOT NULL OR parent_production_order_id IS NOT NULL),
    CONSTRAINT ck_production_chain_not_self CHECK (parent_planned_order_id IS DISTINCT FROM child_planned_order_id)
);
CREATE UNIQUE INDEX uq_production_chain_links ON production_chain_links
    (enterprise_id, child_planned_order_id, COALESCE(parent_planned_order_id, 0), COALESCE(parent_production_order_id, 0));
CREATE INDEX idx_production_chain_parent_planned ON production_chain_links (parent_planned_order_id) WHERE parent_planned_order_id IS NOT NULL;
CREATE INDEX idx_production_chain_parent_of ON production_chain_links (parent_production_order_id) WHERE parent_production_order_id IS NOT NULL;

COMMENT ON TABLE production_chain_links IS
    'OF filha (componente fabricado) → ordem pai que ela abastece; gravado ao firmar a sugestão do MRP.';

-- A reserva do componente para a OF pai (reference_type OF_COMPONENTE) é lida
-- por OF a cada saída de material.
CREATE INDEX IF NOT EXISTS idx_stock_reservations_ativas_ref ON stock_reservations (enterprise_id, reference_type, reference_code)
    WHERE status = 'ACTIVE';

-- A OF pai concluída, encerrada ou cancelada libera o que sobrou reservado para
-- ela: por qualquer caminho (entrega final, coletor, encerramento, cancelamento).
CREATE FUNCTION liberar_reservas_de_componente_da_of() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    r record;
BEGIN
    IF NEW.status IN ('COMPLETED', 'CLOSED', 'CANCELLED') AND OLD.status IS DISTINCT FROM NEW.status THEN
        FOR r IN SELECT id, item_code, mask, warehouse_id, COALESCE(address, '') AS address, COALESCE(lot, '') AS lot, quantity
                   FROM stock_reservations
                  WHERE enterprise_id = NEW.enterprise_id AND reference_type = 'OF_COMPONENTE'
                    AND reference_code = NEW.id AND status = 'ACTIVE'
                  ORDER BY id FOR UPDATE
        LOOP
            UPDATE stock_reservations SET status = 'CANCELLED', updated_at = NOW(),
                   notes = concat_ws(E'\n', notes, 'Liberada: a OF ' || NEW.order_number || ' foi encerrada')
             WHERE id = r.id;
            UPDATE stock_balances SET reserved_qty = GREATEST(reserved_qty - r.quantity, 0), updated_at = NOW()
             WHERE enterprise_id = NEW.enterprise_id AND item_code = r.item_code AND mask = r.mask
               AND warehouse_id = r.warehouse_id AND address = r.address;
            IF r.lot <> '' THEN
                UPDATE stock_lot_balances SET reserved_qty = GREATEST(reserved_qty - r.quantity, 0), updated_at = NOW()
                 WHERE enterprise_id = NEW.enterprise_id AND item_code = r.item_code AND mask = r.mask
                   AND warehouse_id = r.warehouse_id AND lot = r.lot AND address = r.address;
            END IF;
        END LOOP;
    END IF;
    RETURN NEW;
END $$;

CREATE TRIGGER trg_liberar_reservas_de_componente AFTER UPDATE OF status ON production_orders
    FOR EACH ROW EXECUTE FUNCTION liberar_reservas_de_componente_da_of();

-- Parâmetro 52: o que fazer ao iniciar a OF com componente faltando ou OF filha
-- não concluída. NAO (não confere), AVISAR (inicia e avisa) ou BLOQUEAR.
CREATE OR REPLACE FUNCTION public.semear_parametros_planejamento(p_enterprise_id bigint)
 RETURNS void
 LANGUAGE sql
AS $$
    WITH novos AS (
        INSERT INTO planning_params (param_number, param_key, value, description, updated_by, enterprise_id)
        SELECT catalogo.param_number, catalogo.param_key, catalogo.value, catalogo.description,
               '00000000-0000-0000-0000-000000000000', p_enterprise_id
          FROM (VALUES
            (1,  'AGRUPA_DEMANDA_ESTOQUE',            'S',    'Agrupa demanda de estoque com primeira demanda do item'),
            (2,  'COD_FORNECEDOR_INTERFABRICA',       '',     'Codigo fornecedor para pedido compra inter-fabrica'),
            (3,  'COD_CLIENTE_INTERFABRICA',          '',     'Codigo cliente para pedido venda inter-fabrica'),
            (4,  'GERAR_DEMANDA_SEGURANCA_TODOS',     'S',    'Gerar demandas estoque seguranca para todos itens'),
            (5,  'OBRIGATORIEDADE_REFUGO',            'N',    'Obrigatoriedade de refugo na finalizacao de apontamentos'),
            (6,  'DATA_NECESSIDADE_ESTOQUE',          'S',    'Data necessidade das demandas de estoque'),
            (7,  'GERAR_PRIORIDADES_ORDENS',          'S',    'Gerar prioridades nas ordens'),
            (8,  'DIAS_PRIORIDADES',                  '5',    'Gerar prioridades ate quantos dias'),
            (10, 'ITENS_FANTASMAS_GRAVAR',            'N',    'Itens fantasmas devem ser gravados na ordem de producao'),
            (11, 'DESCONSIDERA_SEMANAS_PASSADAS',     'S',    'Desconsidera semanas ja passadas do mes na previsao'),
            (12, 'CONSIDERA_DATAS_TANQUES',           'N',    'Considerar as datas dos tanques no calculo do MRP'),
            (13, 'VERIFICA_SITUACAO_PEDIDO_PROJETO',  'N',    'Verifica a situacao do pedido de projeto'),
            (14, 'UTILIZA_CALCULO_MPS',               'S',    'Utiliza calculo do MPS'),
            (15, 'PROPORCAO_ENTREGA',                 'N',    'Verifica se efetua calculo de proporcionalidade das demandas'),
            (16, 'VALIDA_RESTRICOES_ESTRUTURA',       'S',    'Valida restricoes do item na estrutura'),
            (17, 'TRATA_ASSISTENCIA_TECNICA',         'N',    'Indica se calculo do MRP deve tratar assistencia tecnica'),
            (18, 'PORCENTAGEM_PROPORCAO_VALORIZACAO', '0',    '% da proporcao da entrega para relatorio de erros da valorizacao'),
            (19, 'DEFAULT_POSICAO',                   'A',    'Default para o campo posicao em alguns programas'),
            (20, 'FORMULA_PERDAS_ESTRUTURA',          '2',    'Formula para calculo das perdas na quantidade da estrutura'),
            (24, 'NUMERACAO_ORDENS',                  'AUTO', 'Numeracao das ordens geradas'),
            (45, 'OBRIGAR_CONTROLE_ESTOQUE_TERCEIROS','N',    'Obrigar controle de estoque de terceiros e em terceiros'),
            (46, 'CONSUMO_DIAS_ANTES',                '30',   'Dias antes da venda em que ela consome previsao e demanda de estoque'),
            (47, 'CONSUMO_DIAS_DEPOIS',               '0',    'Dias depois da venda em que ela consome previsao e demanda de estoque'),
            (48, 'HORIZONTE_FIRME_DIAS',              '0',    'Dias a partir de hoje em que o MRP nao cria ordem nova (0 = desligado)'),
            (49, 'TEMPO_SEGURANCA_DIAS',              '0',    'Dias uteis que a ordem planejada chega antes da necessidade'),
            (50, 'LOTE_PERIODO',                      'NENHUM', 'Agrupa as necessidades do item no periodo: NENHUM, DIA, SEMANA ou MES'),
            (51, 'HORIZONTE_REPROGRAMACAO_DIAS',      '30',    'Dias a partir de hoje em que a ordem firme atrasada cobre a necessidade em vez de gerar sugestao nova (0 = desligado)'),
            (52, 'CONFERE_COMPONENTES_INICIO_OF',     'AVISAR', 'Ao iniciar a OF com componente faltando ou OF filha aberta: NAO, AVISAR ou BLOQUEAR')
          ) AS catalogo(param_number, param_key, value, description)
         WHERE NOT EXISTS (SELECT 1 FROM planning_params atual
                            WHERE atual.enterprise_id = p_enterprise_id
                              AND (atual.param_number = catalogo.param_number OR atual.param_key = catalogo.param_key))
        RETURNING id
    )
    INSERT INTO planning_params_semeados (planning_param_id) SELECT id FROM novos;
$$;

SELECT semear_parametros_planejamento(id) FROM enterprise;

SELECT instalar_conferencia_de_empresa();
