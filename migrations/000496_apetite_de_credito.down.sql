ALTER TABLE customer_credit_policies
    DROP CONSTRAINT IF EXISTS chk_credit_policy_apetite,
    DROP COLUMN IF EXISTS new_customer_revenue_pct;
SELECT instalar_conferencia_de_empresa();
