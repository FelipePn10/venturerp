-- Parâmetros de planejamento por empresa.
--
-- A 000221 copiou o catálogo (linhas sem empresa) só para as empresas que já
-- existiam. Empresa criada depois — a Usimac, a Tecnofer e toda instalação nova
-- — ficou sem parâmetros: a tela vem vazia e nada pode ser alterado, e o MRP
-- roda nos padrões do código. Os valores abaixo são os do catálogo da 000058,
-- que coincidem com esses padrões (DefaultTypedPlanningParams): semear não muda
-- o cálculo, só torna os parâmetros visíveis e editáveis.

-- Os parâmetros são configuração da empresa: saem com ela.
ALTER TABLE planning_params DROP CONSTRAINT planning_params_enterprise_id_fkey;
ALTER TABLE planning_params ADD CONSTRAINT planning_params_enterprise_id_fkey
    FOREIGN KEY (enterprise_id) REFERENCES enterprise(id) ON DELETE CASCADE;

-- Linhas criadas por esta migration (pela semeadura ou pelo gatilho), para o
-- down remover só o que ela criou e ninguém alterou.
CREATE TABLE planning_params_semeados (
    planning_param_id BIGINT PRIMARY KEY REFERENCES planning_params(id) ON DELETE CASCADE
);

CREATE FUNCTION semear_parametros_planejamento(p_enterprise_id BIGINT) RETURNS void
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

CREATE FUNCTION semear_parametros_planejamento_nova_empresa() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    PERFORM semear_parametros_planejamento(NEW.id);
    RETURN NEW;
END $$;

CREATE TRIGGER trg_enterprise_planning_params AFTER INSERT ON enterprise
    FOR EACH ROW EXECUTE FUNCTION semear_parametros_planejamento_nova_empresa();

SELECT semear_parametros_planejamento(id) FROM enterprise ORDER BY id;
