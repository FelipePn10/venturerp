-- Cadastro de cliente (VCLI0500) revisto e limite de crédito calculado.
--
-- Saem do cliente os campos que não pertencem a ele: a condição de pagamento e
-- o tipo de nota são decididos no pedido, a tabela de venda é do item, e a
-- visibilidade de condição e o grupo de portadores só existiam por causa deles.
-- O que estava gravado vai para customers_campos_retirados (o down devolve).

CREATE TABLE customers_campos_retirados (
    customer_id             BIGINT PRIMARY KEY,
    enterprise_id           BIGINT NOT NULL,
    code                    BIGINT NOT NULL,
    payment_condition_id    BIGINT,
    sales_table_id          BIGINT,
    invoice_type_id         BIGINT,
    carrier_group_id        BIGINT,
    payment_cond_visibility payment_condition_visibility_enum,
    retirado_em             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO customers_campos_retirados (customer_id, enterprise_id, code, payment_condition_id,
       sales_table_id, invoice_type_id, carrier_group_id, payment_cond_visibility)
SELECT id, enterprise_id, code, payment_condition_id, sales_table_id, invoice_type_id,
       carrier_group_id, payment_cond_visibility
  FROM customers;

ALTER TABLE customers
    DROP COLUMN payment_condition_id,
    DROP COLUMN sales_table_id,
    DROP COLUMN invoice_type_id,
    DROP COLUMN carrier_group_id,
    DROP COLUMN payment_cond_visibility;

ALTER TABLE customers
    -- Fiscal
    ADD COLUMN ie_indicator           VARCHAR(20),
    ADD COLUMN tax_regime             VARCHAR(20),
    ADD COLUMN main_cnae              VARCHAR(10),
    ADD COLUMN nfe_email              VARCHAR(255),
    ADD COLUMN billing_email          VARCHAR(255),
    -- Comercial e logística
    -- Sem FK: o representante vale pela empresa a que está VINCULADO
    -- (representative_enterprises.enterprise_code), não pela que o cadastrou;
    -- a aplicação confere o vínculo (representativevalidation).
    ADD COLUMN representative_code    BIGINT,
    ADD COLUMN shipping_carrier_id    BIGINT,
    ADD COLUMN freight_type           VARCHAR(10),
    ADD COLUMN requires_customer_po   BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN requires_quality_cert  BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN allows_partial_billing BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN receiving_notes        TEXT,
    ADD COLUMN commercial_notes       TEXT,
    ADD COLUMN financial_notes        TEXT,
    -- Situação na Receita (consulta do CNPJ, refeita periodicamente)
    ADD COLUMN registration_status    VARCHAR(40),
    ADD COLUMN registration_checked_at TIMESTAMPTZ,
    -- Última TENTATIVA de consulta: a fila do agendador anda por ela, senão um
    -- CNPJ que a Receita nunca responde ficaria para sempre na frente.
    ADD COLUMN registration_attempt_at TIMESTAMPTZ,
    ADD COLUMN opening_date           DATE,
    ADD COLUMN share_capital          NUMERIC(17,2),
    ADD COLUMN company_size           VARCHAR(60),
    ADD COLUMN legal_nature           VARCHAR(160),
    -- Crédito: o limite vigente é credit_limit; o resto vem da última análise
    ADD COLUMN credit_score           INTEGER,
    ADD COLUMN credit_risk_class      CHAR(1),
    ADD COLUMN credit_limit_suggested NUMERIC(15,2),
    ADD COLUMN credit_limit_expires_at DATE,
    ADD COLUMN credit_reviewed_at     TIMESTAMPTZ,
    ADD CONSTRAINT chk_customers_ie_indicator
        CHECK (ie_indicator IN ('CONTRIBUINTE', 'ISENTO', 'NAO_CONTRIBUINTE')),
    ADD CONSTRAINT chk_customers_tax_regime
        CHECK (tax_regime IN ('SIMPLES', 'MEI', 'NORMAL')),
    ADD CONSTRAINT chk_customers_freight_type
        CHECK (freight_type IN ('CIF', 'FOB', 'TERCEIROS', 'SEM_FRETE')),
    ADD CONSTRAINT chk_customers_credit_risk_class
        CHECK (credit_risk_class IN ('A', 'B', 'C', 'D', 'E')),
    ADD CONSTRAINT customers_shipping_carrier_id_fkey
        FOREIGN KEY (shipping_carrier_id) REFERENCES shipping_carriers(id);

-- A conferência de empresa do gatilho de customers ainda cita as colunas que
-- saíram (carrier_group_id...): sem reinstalar aqui, o UPDATE dos limites
-- legados lá embaixo falha em qualquer base que tenha cliente com limite.
SELECT instalar_conferencia_de_empresa();

-- Grupo econômico: a raiz do CNPJ (8 primeiros dígitos) junta matriz e filiais.
CREATE INDEX ix_customers_cnpj_root
    ON customers (enterprise_id, (LEFT(regexp_replace(document_number, '\D', '', 'g'), 8)))
 WHERE document_type = 'CNPJ';

-- Política de crédito da empresa (uma linha por empresa; sem linha, valem os
-- padrões abaixo).
CREATE TABLE customer_credit_policies (
    enterprise_id            BIGINT PRIMARY KEY REFERENCES enterprise(id),
    max_limit                NUMERIC(15,2) NOT NULL DEFAULT 0,
    new_customer_capital_pct NUMERIC(7,4)  NOT NULL DEFAULT 10,
    new_customer_cap         NUMERIC(15,2) NOT NULL DEFAULT 30000,
    override_tolerance_pct   NUMERIC(7,4)  NOT NULL DEFAULT 10,
    overdue_block_days       INTEGER       NOT NULL DEFAULT 30,
    history_months           INTEGER       NOT NULL DEFAULT 24,
    validity_months_a        INTEGER       NOT NULL DEFAULT 12,
    validity_months_b        INTEGER       NOT NULL DEFAULT 6,
    validity_months_c        INTEGER       NOT NULL DEFAULT 3,
    validity_months_d        INTEGER       NOT NULL DEFAULT 1,
    validity_months_e        INTEGER       NOT NULL DEFAULT 1,
    registration_check_days  INTEGER       NOT NULL DEFAULT 30,
    updated_at               TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_by               UUID,
    CONSTRAINT chk_credit_policy_valores CHECK (
        max_limit >= 0 AND new_customer_capital_pct BETWEEN 0 AND 100
        AND new_customer_cap >= 0 AND override_tolerance_pct BETWEEN 0 AND 100
        AND overdue_block_days > 0 AND history_months BETWEEN 6 AND 60
        AND LEAST(validity_months_a, validity_months_b, validity_months_c,
                  validity_months_d, validity_months_e) > 0
        AND registration_check_days > 0)
);

-- Trilha de cada limite gravado: o que o sistema sugeriu, o que foi aprovado,
-- quem aprovou e por quê. Não tem FK para o cliente de propósito: a trilha
-- sobrevive à exclusão do cadastro.
CREATE TABLE customer_credit_reviews (
    id              BIGSERIAL PRIMARY KEY,
    enterprise_id   BIGINT NOT NULL REFERENCES enterprise(id),
    customer_code   BIGINT NOT NULL,
    origin          VARCHAR(20) NOT NULL,
    suggested_limit NUMERIC(15,2) NOT NULL,
    approved_limit  NUMERIC(15,2) NOT NULL,
    previous_limit  NUMERIC(15,2),
    score           INTEGER NOT NULL,
    risk_class      CHAR(1) NOT NULL,
    expires_at      DATE NOT NULL,
    deviates        BOOLEAN NOT NULL,
    reason          TEXT,
    factors         JSONB NOT NULL DEFAULT '[]',
    created_by      UUID NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_credit_review_origin CHECK (origin IN ('CADASTRO', 'REVISAO', 'LEGADO')),
    CONSTRAINT chk_credit_review_class CHECK (risk_class IN ('A', 'B', 'C', 'D', 'E')),
    CONSTRAINT chk_credit_review_reason CHECK (NOT deviates OR length(btrim(reason)) >= 15)
);
CREATE INDEX ix_customer_credit_reviews_customer
    ON customer_credit_reviews (enterprise_id, customer_code, created_at DESC);

-- Limites digitados antes da política continuam valendo, mas vencem em 30
-- dias: a próxima revisão passa pelo cálculo.
INSERT INTO customer_credit_reviews (enterprise_id, customer_code, origin, suggested_limit,
       approved_limit, previous_limit, score, risk_class, expires_at, deviates, reason, created_by)
SELECT enterprise_id, code, 'LEGADO', credit_limit, credit_limit, credit_limit, 0, 'C',
       CURRENT_DATE + 30, FALSE, 'Limite informado manualmente antes da política de crédito', created_by
  FROM customers
 WHERE credit_limit > 0;

UPDATE customers
   SET credit_limit_expires_at = CURRENT_DATE + 30,
       credit_risk_class = 'C'
 WHERE credit_limit > 0;

SELECT instalar_conferencia_de_empresa();
