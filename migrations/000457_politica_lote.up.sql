-- Política de lote completa (VPLA0310): período de agrupamento e lote máximo.
-- Ver .ai/tasks/paridade-mrp-lote-reserva-excecao.md.
--
-- lote_periodo: as necessidades do item no mesmo período viram uma ordem, na
-- data da primeira (SAP: lote por dia/semana/mês). Herda item → classificação
-- → parâmetro 50 da empresa.
-- lote_maximo: a necessidade acima dele vira várias ordens (só na política do
-- item: é quantidade do item).
ALTER TABLE politicas_planejamento
    ADD COLUMN lote_periodo VARCHAR(8) CHECK (lote_periodo IN ('NENHUM', 'DIA', 'SEMANA', 'MES')),
    ADD COLUMN lote_maximo  NUMERIC(18,6) CHECK (lote_maximo > 0),
    ADD CONSTRAINT politicas_planejamento_lote_maximo_do_item CHECK (lote_maximo IS NULL OR item_code IS NOT NULL);

ALTER TABLE politicas_planejamento DROP CONSTRAINT politicas_planejamento_check1;
ALTER TABLE politicas_planejamento ADD CONSTRAINT politicas_planejamento_algum_campo
    CHECK (estrategia IS NOT NULL OR consumo_dias_antes IS NOT NULL OR consumo_dias_depois IS NOT NULL
           OR horizonte_firme_dias IS NOT NULL OR tempo_seguranca_dias IS NOT NULL
           OR lote_periodo IS NOT NULL OR lote_maximo IS NOT NULL);

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
            (47, 'CONSUMO_DIAS_DEPOIS',               '0',    'Dias depois da venda em que ela consome previsao e demanda de estoque'),
            (48, 'HORIZONTE_FIRME_DIAS',              '0',    'Dias a partir de hoje em que o MRP nao cria ordem nova (0 = desligado)'),
            (49, 'TEMPO_SEGURANCA_DIAS',              '0',    'Dias uteis que a ordem planejada chega antes da necessidade'),
            (50, 'LOTE_PERIODO',                      'NENHUM', 'Agrupa as necessidades do item no periodo: NENHUM, DIA, SEMANA ou MES')
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
