-- Demanda independente enterprise (.ai/tasks/demanda-independente-enterprise.md).
--
-- A demanda independente era item, máscara, centro, quantidade e data: sem
-- motivo, sem destino, sem situação. Nunca se encerrava, e o MRP a contava
-- inteira para sempre.

-- O código era único no banco inteiro: a segunda empresa não podia usar um
-- número que a primeira já tinha. Passa a ser único por empresa.
ALTER TABLE independent_demands DROP CONSTRAINT independent_demands_code_key;

-- A tela prometia "numeração automática" e gravava o código 0. Cada código
-- que não é positivo recebe o primeiro número livre da empresa (o mesmo que a
-- numeração automática passa a usar).
DO $$
DECLARE
    r RECORD;
    livre BIGINT;
BEGIN
    FOR r IN SELECT id, enterprise_id FROM independent_demands WHERE code <= 0 ORDER BY id LOOP
        SELECT COALESCE(MIN(d.code + 1), 1) INTO livre
          FROM (SELECT code FROM independent_demands
                 WHERE enterprise_id IS NOT DISTINCT FROM r.enterprise_id AND code > 0
                UNION ALL SELECT 0) d
         WHERE NOT EXISTS (SELECT 1 FROM independent_demands x
                            WHERE x.enterprise_id IS NOT DISTINCT FROM r.enterprise_id AND x.code = d.code + 1);
        UPDATE independent_demands SET code = livre WHERE id = r.id;
    END LOOP;
END $$;

CREATE UNIQUE INDEX uq_independent_demands_empresa_codigo ON independent_demands (enterprise_id, code);

-- Motivo: a família decide como a demanda se encerra.
--   Expectativa de venda (ESTOQUE, REPOSICAO, ANTECIPACAO_VENDA): a venda do
--   item a consome e ela vence quando nenhuma venda pode mais consumi-la.
--   Finalidade própria (AMOSTRA, FEIRA, CONSUMO_INTERNO, ASSISTENCIA, PROJETO):
--   segura o estoque até a saída para a finalidade ("Atender").
-- Situação gravada: ABERTA, ENCERRADA (saldo encerrado à mão) ou CANCELADA;
-- atendida, consumida e vencida são calculadas.
ALTER TABLE independent_demands
    ADD COLUMN motivo               VARCHAR(24)  NOT NULL DEFAULT 'ESTOQUE'
        CHECK (motivo IN ('ESTOQUE', 'REPOSICAO', 'ANTECIPACAO_VENDA', 'AMOSTRA', 'FEIRA',
                          'CONSUMO_INTERNO', 'ASSISTENCIA', 'PROJETO')),
    ADD COLUMN warehouse_id         BIGINT       REFERENCES warehouse(id),
    ADD COLUMN customer_code        BIGINT,
    ADD COLUMN projeto              VARCHAR(60),
    ADD COLUMN prioridade           SMALLINT     NOT NULL DEFAULT 3 CHECK (prioridade BETWEEN 1 AND 5),
    ADD COLUMN observacao           TEXT,
    -- Entregas de uma mesma programação (100 peças em 4 semanas de 25).
    ADD COLUMN programacao          UUID,
    ADD COLUMN situacao             VARCHAR(12)  NOT NULL DEFAULT 'ABERTA'
        CHECK (situacao IN ('ABERTA', 'ENCERRADA', 'CANCELADA')),
    ADD COLUMN encerrada_em         TIMESTAMPTZ,
    ADD COLUMN encerrada_por        UUID,
    ADD COLUMN motivo_encerramento  TEXT,
    ADD COLUMN updated_by           UUID;

UPDATE independent_demands SET situacao = 'CANCELADA', encerrada_em = updated_at WHERE NOT is_active;

-- A confirmação do pedido de venda gerava uma demanda por linha (código =
-- pedido × 100000 + sequência), e o MRP também lê o pedido diretamente: o
-- pedido entrava duas vezes no cálculo. Essas demandas saem do planejamento.
UPDATE independent_demands d
   SET situacao = 'CANCELADA', is_active = FALSE, encerrada_em = NOW(),
       motivo_encerramento = 'Gerada pela confirmação do pedido de venda ' || so.order_number
                             || ': duplicava o pedido no MRP, que lê o pedido diretamente.'
  FROM sales_order_items soi
  JOIN sales_orders so ON so.code = soi.sales_order_code
  JOIN enterprise e ON e.code = so.enterprise_code
 WHERE d.situacao = 'ABERTA'
   AND e.id = d.enterprise_id
   AND d.code = so.code * 100000 + soi.sequence
   AND d.item_code = soi.item_code
   AND d.quantity = soi.requested_qty;

CREATE INDEX idx_independent_demands_situacao ON independent_demands (enterprise_id, situacao, demand_date);

-- Que ordem cobre cada demanda: pela sugestão firmada (pegging) ou por "Gerar
-- OF agora". É a ponte demanda → ordem planejada → OF da tela e do rastreio.
CREATE TABLE independent_demand_orders (
    id                   BIGSERIAL     PRIMARY KEY,
    enterprise_id        BIGINT        NOT NULL REFERENCES enterprise(id),
    demand_id            BIGINT        NOT NULL REFERENCES independent_demands(id) ON DELETE CASCADE,
    planned_order_id     BIGINT        REFERENCES planned_orders(id) ON DELETE CASCADE,
    production_order_id  BIGINT        REFERENCES production_orders(id) ON DELETE CASCADE,
    quantity             NUMERIC(15,4) NOT NULL CHECK (quantity > 0),
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_by           UUID          NOT NULL,
    CHECK (planned_order_id IS NOT NULL OR production_order_id IS NOT NULL)
);
CREATE INDEX idx_independent_demand_orders_demand ON independent_demand_orders (demand_id);
CREATE INDEX idx_independent_demand_orders_planned ON independent_demand_orders (planned_order_id);
CREATE INDEX idx_independent_demand_orders_of ON independent_demand_orders (production_order_id);

-- Janela de consumo pela venda (SAP: períodos de consumo para trás e para a
-- frente). Uma venda na data V consome a previsão e a demanda de estoque com
-- data entre V − 46 e V + 47 dias.
CREATE OR REPLACE FUNCTION semear_parametros_planejamento(p_enterprise_id BIGINT) RETURNS void
LANGUAGE sql AS $$
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
            (47, 'CONSUMO_DIAS_DEPOIS',               '0',    'Dias depois da venda em que ela consome previsao e demanda de estoque')
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
