DELETE FROM planning_params WHERE param_number IN (46, 47);

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
            (45, 'OBRIGAR_CONTROLE_ESTOQUE_TERCEIROS','N',    'Obrigar controle de estoque de terceiros e em terceiros')
          ) AS catalogo(param_number, param_key, value, description)
         WHERE NOT EXISTS (SELECT 1 FROM planning_params atual
                            WHERE atual.enterprise_id = p_enterprise_id
                              AND (atual.param_number = catalogo.param_number OR atual.param_key = catalogo.param_key))
        RETURNING id
    )
    INSERT INTO planning_params_semeados (planning_param_id) SELECT id FROM novos;
$$;

DROP TABLE independent_demand_orders;
DROP INDEX idx_independent_demands_situacao;

-- As demandas canceladas pelo up continuam fora do cálculo (is_active = FALSE).
ALTER TABLE independent_demands
    DROP COLUMN motivo,
    DROP COLUMN warehouse_id,
    DROP COLUMN customer_code,
    DROP COLUMN projeto,
    DROP COLUMN prioridade,
    DROP COLUMN observacao,
    DROP COLUMN programacao,
    DROP COLUMN situacao,
    DROP COLUMN encerrada_em,
    DROP COLUMN encerrada_por,
    DROP COLUMN motivo_encerramento,
    DROP COLUMN updated_by;

DROP INDEX uq_independent_demands_empresa_codigo;
-- Códigos repetidos entre empresas impedem a volta da unicidade global; ela só
-- volta quando não há repetição.
DO $$
BEGIN
    IF NOT EXISTS (SELECT code FROM independent_demands GROUP BY code HAVING COUNT(*) > 1) THEN
        ALTER TABLE independent_demands ADD CONSTRAINT independent_demands_code_key UNIQUE (code);
    END IF;
END $$;

SELECT instalar_conferencia_de_empresa();
