-- Crédito sem depender só de dado público e revisado sozinho a cada trimestre.
--
-- 1. Faturamento declarado do cliente (DRE, balanço, declaração, IRPF): mede a
--    capacidade de quem não tem capital social na Receita (CPF, ou CNPJ sem o
--    capital informado). Fica numa trilha — cada declaração com data e fonte;
--    vale a mais recente com até 18 meses.
CREATE TABLE customer_financial_statements (
    id             BIGSERIAL PRIMARY KEY,
    enterprise_id  BIGINT NOT NULL,
    customer_code  BIGINT NOT NULL,
    annual_revenue NUMERIC(15,2) NOT NULL,
    reference_date DATE NOT NULL,
    source         VARCHAR(20) NOT NULL,
    notes          TEXT,
    created_by     UUID NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_financial_statements_customer FOREIGN KEY (enterprise_id, customer_code)
        REFERENCES customers (enterprise_id, code) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT chk_financial_statements_revenue CHECK (annual_revenue > 0),
    CONSTRAINT chk_financial_statements_source CHECK (source IN ('DRE', 'BALANCO', 'DECLARACAO', 'IRPF', 'OUTRO'))
);
CREATE INDEX ix_financial_statements_customer
    ON customer_financial_statements (enterprise_id, customer_code, reference_date DESC);

-- 2. Política: % do faturamento mensal declarado do cliente que vira
--    capacidade; o faturamento mensal da própria empresa enquanto o sistema
--    tem menos de 3 meses de notas (implantação); e a revisão automática —
--    a cada quantos meses (0 = desligada) e quanto ela pode aumentar de uma vez.
ALTER TABLE customer_credit_policies
    ADD COLUMN customer_revenue_pct     NUMERIC(7,4)  NOT NULL DEFAULT 10,
    ADD COLUMN declared_monthly_revenue NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN auto_review_months       INTEGER       NOT NULL DEFAULT 3,
    ADD COLUMN auto_increase_max_pct    NUMERIC(7,4)  NOT NULL DEFAULT 50,
    ADD CONSTRAINT chk_credit_policy_revisao CHECK (
        customer_revenue_pct BETWEEN 0 AND 100 AND declared_monthly_revenue >= 0
        AND auto_review_months BETWEEN 0 AND 12 AND auto_increase_max_pct BETWEEN 0 AND 1000);

-- 3. A revisão automática entra na trilha com origem própria; a tentativa
--    fica marcada para a fila andar quando um cliente falha.
ALTER TABLE customer_credit_reviews DROP CONSTRAINT chk_credit_review_origin;
ALTER TABLE customer_credit_reviews ADD CONSTRAINT chk_credit_review_origin
    CHECK (origin IN ('CADASTRO', 'REVISAO', 'LEGADO', 'AUTOMATICA'));
ALTER TABLE customers ADD COLUMN credit_review_attempt_at TIMESTAMPTZ;

SELECT instalar_conferencia_de_empresa();
