ALTER TABLE customers DROP COLUMN IF EXISTS credit_review_attempt_at;
DELETE FROM customer_credit_reviews WHERE origin = 'AUTOMATICA';
ALTER TABLE customer_credit_reviews DROP CONSTRAINT chk_credit_review_origin;
ALTER TABLE customer_credit_reviews ADD CONSTRAINT chk_credit_review_origin
    CHECK (origin IN ('CADASTRO', 'REVISAO', 'LEGADO'));
ALTER TABLE customer_credit_policies
    DROP CONSTRAINT IF EXISTS chk_credit_policy_revisao,
    DROP COLUMN IF EXISTS auto_increase_max_pct,
    DROP COLUMN IF EXISTS auto_review_months,
    DROP COLUMN IF EXISTS declared_monthly_revenue,
    DROP COLUMN IF EXISTS customer_revenue_pct;
DROP TABLE IF EXISTS customer_financial_statements;
SELECT instalar_conferencia_de_empresa();
