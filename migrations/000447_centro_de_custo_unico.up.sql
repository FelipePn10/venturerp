-- Centro de custo único (2026-10-27). Havia dois cadastros: cost_centers
-- (VCTB0102 — máquinas, OPs, estruturas, MRP, rateios) e centros_custo
-- (VFIN0130 — itens da NF de entrada, contas a pagar e receber, frete). Agora é
-- um só, cost_centers; centros_custo vira legado (só para o rollback).
--
-- O código que a empresa usa (alfanumérico: CC-FAB-CORTE) fica na gestão do
-- centro; o code numérico de cost_centers continua a chave que máquinas,
-- estruturas e pedidos guardam.

-- 1. Gestão do centro: código da empresa, responsável, aprovador, conta padrão.
CREATE TABLE cost_center_gestao (
    cost_center_id      BIGINT       PRIMARY KEY REFERENCES cost_centers(id) ON DELETE CASCADE,
    enterprise_id       BIGINT       NOT NULL REFERENCES enterprise(id),
    codigo              VARCHAR(50)  NOT NULL,
    responsavel_id      UUID         REFERENCES users(id),
    aprovador_id        UUID         REFERENCES users(id),
    -- Requisição de compra com item deste centro só gera pedido depois que o
    -- aprovador do centro aprovar.
    aprova_requisicoes  BOOLEAN      NOT NULL DEFAULT FALSE,
    conta_padrao_id     BIGINT       REFERENCES plano_contas(id),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by          UUID,
    CONSTRAINT ck_cc_gestao_codigo CHECK (btrim(codigo) <> ''),
    CONSTRAINT ck_cc_gestao_aprovador CHECK (NOT aprova_requisicoes OR aprovador_id IS NOT NULL)
);
CREATE UNIQUE INDEX uq_cc_gestao_codigo ON cost_center_gestao (enterprise_id, upper(codigo));

INSERT INTO cost_center_gestao (cost_center_id, enterprise_id, codigo)
SELECT id, enterprise_id, code::text FROM cost_centers;

-- Todo centro novo, por qualquer caminho, nasce com a gestão (código da
-- empresa = o code, até alguém trocar).
CREATE OR REPLACE FUNCTION cost_center_gestao_inicial() RETURNS trigger AS $$
BEGIN
    INSERT INTO cost_center_gestao (cost_center_id, enterprise_id, codigo, updated_by)
    VALUES (NEW.id, NEW.enterprise_id, NEW.code::text, NEW.created_by)
    ON CONFLICT (cost_center_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 2. Os centros do cadastro financeiro entram no cadastro único, empresa a
-- empresa. Código repetido (já existe na gestão) reaproveita o centro.
CREATE TABLE centro_custo_migracao (
    antigo_id  BIGINT PRIMARY KEY,
    novo_id    BIGINT NOT NULL REFERENCES cost_centers(id),
    criado     BOOLEAN NOT NULL
);

INSERT INTO centro_custo_migracao (antigo_id, novo_id, criado)
SELECT cc.id, g.cost_center_id, FALSE
  FROM centros_custo cc
  JOIN cost_center_gestao g ON g.enterprise_id = cc.enterprise_id AND upper(g.codigo) = upper(btrim(cc.codigo));

WITH novos AS (
    SELECT cc.*, row_number() OVER (ORDER BY cc.enterprise_id, cc.id) AS n
      FROM centros_custo cc
     WHERE NOT EXISTS (SELECT 1 FROM centro_custo_migracao m WHERE m.antigo_id = cc.id)
), base AS (
    SELECT COALESCE(MAX(code), 0) AS ultimo FROM cost_centers
), inseridos AS (
    INSERT INTO cost_centers (code, description, type, is_ratio, start_date, is_active, created_by, enterprise_id)
    SELECT (base.ultimo + novos.n)::int,
           btrim(novos.descricao),
           CASE upper(btrim(novos.tipo))
                WHEN 'PRODUTIVO' THEN 'PRODUCTIVE'
                WHEN 'COMERCIAL' THEN 'COMMERCIAL'
                WHEN 'AUXILIAR'  THEN 'AUXILIARY'
                ELSE 'ADMINISTRATIVE' END::type_cc_enum,
           FALSE,
           novos.created_at::date,
           novos.is_active,
           COALESCE((SELECT ue.user_id FROM user_enterprises ue WHERE ue.enterprise_id = novos.enterprise_id ORDER BY (ue.role = 'ADMIN') DESC, ue.user_id LIMIT 1),
                    (SELECT u.id FROM users u ORDER BY u.created_at LIMIT 1)),
           novos.enterprise_id
      FROM novos, base
    RETURNING id, code, enterprise_id
)
INSERT INTO centro_custo_migracao (antigo_id, novo_id, criado)
SELECT novos.id, inseridos.id, TRUE
  FROM novos JOIN base ON TRUE
  JOIN inseridos ON inseridos.code = (base.ultimo + novos.n)::int;

INSERT INTO cost_center_gestao (cost_center_id, enterprise_id, codigo)
SELECT m.novo_id, cc.enterprise_id, btrim(cc.codigo)
  FROM centro_custo_migracao m JOIN centros_custo cc ON cc.id = m.antigo_id
 WHERE m.criado;

CREATE TRIGGER trg_cost_center_gestao_inicial AFTER INSERT ON cost_centers
    FOR EACH ROW EXECUTE FUNCTION cost_center_gestao_inicial();

-- Hierarquia pelo prefixo do código (CC-FAB-CORTE fica abaixo de CC-FAB).
UPDATE cost_centers filho
   SET parent_code = pai.code
  FROM centro_custo_migracao m
  JOIN cost_center_gestao gf ON gf.cost_center_id = m.novo_id,
       LATERAL (
           SELECT c.code
             FROM cost_center_gestao gp JOIN cost_centers c ON c.id = gp.cost_center_id
            WHERE gp.enterprise_id = gf.enterprise_id
              AND upper(gf.codigo) LIKE upper(gp.codigo) || '-%'
            ORDER BY length(gp.codigo) DESC
            LIMIT 1
       ) pai
 WHERE m.criado AND filho.id = m.novo_id AND filho.parent_code IS NULL;

-- 3. As referências financeiras passam para o cadastro único.
ALTER TABLE fiscal_entry_items DROP CONSTRAINT IF EXISTS fiscal_entry_items_centro_custo_id_fkey;
ALTER TABLE fiscal_entry_installment_allocations DROP CONSTRAINT IF EXISTS fiscal_entry_installment_allocations_centro_custo_id_fkey;
ALTER TABLE contas_pagar_rateios DROP CONSTRAINT IF EXISTS contas_pagar_rateios_centro_custo_id_fkey;

UPDATE fiscal_entry_items t SET centro_custo_id = m.novo_id FROM centro_custo_migracao m WHERE t.centro_custo_id = m.antigo_id;
UPDATE fiscal_entry_installment_allocations t SET centro_custo_id = m.novo_id FROM centro_custo_migracao m WHERE t.centro_custo_id = m.antigo_id;
UPDATE contas_pagar_rateios t SET centro_custo_id = m.novo_id FROM centro_custo_migracao m WHERE t.centro_custo_id = m.antigo_id;
UPDATE contas_pagar t SET centro_custo_id = m.novo_id FROM centro_custo_migracao m WHERE t.centro_custo_id = m.antigo_id;
UPDATE contas_receber t SET centro_custo_id = m.novo_id FROM centro_custo_migracao m WHERE t.centro_custo_id = m.antigo_id;
UPDATE fiscal_freight_allocations t SET centro_custo_id = m.novo_id FROM centro_custo_migracao m WHERE t.centro_custo_id = m.antigo_id;
UPDATE cost_overhead_rules t SET centro_custo_id = m.novo_id FROM centro_custo_migracao m WHERE t.centro_custo_id = m.antigo_id;
UPDATE accounting_account_refs t SET cost_center_id = m.novo_id FROM centro_custo_migracao m WHERE t.cost_center_id = m.antigo_id;

ALTER TABLE fiscal_entry_items ADD CONSTRAINT fiscal_entry_items_centro_custo_id_fkey
    FOREIGN KEY (centro_custo_id) REFERENCES cost_centers(id);
ALTER TABLE fiscal_entry_installment_allocations ADD CONSTRAINT fiscal_entry_installment_allocations_centro_custo_id_fkey
    FOREIGN KEY (centro_custo_id) REFERENCES cost_centers(id);
ALTER TABLE contas_pagar_rateios ADD CONSTRAINT contas_pagar_rateios_centro_custo_id_fkey
    FOREIGN KEY (centro_custo_id) REFERENCES cost_centers(id);

ALTER TABLE centros_custo RENAME TO centros_custo_legado;

-- 4. Orçamento mensal por centro (o realizado vem do contas a pagar).
CREATE TABLE cost_center_orcamentos (
    id              BIGSERIAL     PRIMARY KEY,
    enterprise_id   BIGINT        NOT NULL REFERENCES enterprise(id),
    cost_center_id  BIGINT        NOT NULL REFERENCES cost_centers(id) ON DELETE CASCADE,
    ano             SMALLINT      NOT NULL CHECK (ano BETWEEN 2000 AND 2100),
    mes             SMALLINT      NOT NULL CHECK (mes BETWEEN 1 AND 12),
    valor           NUMERIC(15,2) NOT NULL CHECK (valor >= 0),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_by      UUID,
    UNIQUE (cost_center_id, ano, mes)
);
CREATE INDEX idx_cc_orcamentos_empresa ON cost_center_orcamentos (enterprise_id, ano, mes);

-- 5. Regra de obrigatoriedade: a conta que exige centro de custo no lançamento.
ALTER TABLE plano_contas ADD COLUMN exige_centro_custo BOOLEAN NOT NULL DEFAULT FALSE;

-- 6. Aprovação da requisição de compra pelo aprovador do centro.
CREATE TABLE requisicao_aprovacoes_centro (
    id                 BIGSERIAL    PRIMARY KEY,
    enterprise_id      BIGINT       NOT NULL REFERENCES enterprise(id),
    requisition_code   BIGINT       NOT NULL,
    cost_center_id     BIGINT       NOT NULL REFERENCES cost_centers(id),
    status             VARCHAR(10)  NOT NULL DEFAULT 'PENDENTE' CHECK (status IN ('PENDENTE', 'APROVADA', 'REPROVADA')),
    aprovador_id       UUID         NOT NULL REFERENCES users(id),
    decidido_por       UUID         REFERENCES users(id),
    decidido_em        TIMESTAMPTZ,
    justificativa      TEXT,
    criado_em          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (requisition_code, cost_center_id)
);
CREATE INDEX idx_req_aprov_centro_pendentes ON requisicao_aprovacoes_centro (enterprise_id, aprovador_id) WHERE status = 'PENDENTE';

-- 7. Alertas: orçamento do centro estourado no mês e requisição esperando o
-- aprovador do centro.
INSERT INTO notification_event_catalog (event_key, version, name_pt_br, description_pt_br, module, event_kind, severity, allowed_cadences,
    template_key, suggested_recipient_roles, producer_status, producer_description)
VALUES
    ('FINANCEIRO_CENTRO_ORCAMENTO_ESTOURADO', 1, 'Orçamento do centro de custo estourado', 'O realizado do mês no centro de custo passou do orçado.',
     'FINANCEIRO', 'PENDENCIA', 'ATENCAO', ARRAY['IMEDIATO', 'RESUMO_DIARIO'], 'alerta_padrao', ARRAY['FINANCEIRO'],
     'ATIVO', 'Produtor conectado e monitorado pelo backend.'),
    ('COMPRAS_REQUISICAO_AGUARDANDO_CENTRO', 1, 'Requisição aguardando o aprovador do centro de custo', 'Requisição de compra com item de centro de custo que aprova requisições, ainda sem decisão.',
     'COMPRAS', 'PENDENCIA', 'ATENCAO', ARRAY['IMEDIATO', 'RESUMO_DIARIO'], 'alerta_padrao', ARRAY['COMPRAS'],
     'ATIVO', 'Produtor conectado e monitorado pelo backend.');

SELECT instalar_conferencia_de_empresa();
