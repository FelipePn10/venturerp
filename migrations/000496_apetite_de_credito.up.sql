-- Limite de cliente novo (sem histórico): o menor entre a capacidade do cliente
-- (% do capital social) e o apetite da empresa, que é este % do faturamento
-- médio mensal dela — quanto ela aceita expor num cliente ainda não testado.
-- new_customer_cap passa a ser o piso desse apetite (empresa com pouco
-- faturamento no sistema).
ALTER TABLE customer_credit_policies
    ADD COLUMN new_customer_revenue_pct NUMERIC(7,4) NOT NULL DEFAULT 20,
    ADD CONSTRAINT chk_credit_policy_apetite CHECK (new_customer_revenue_pct BETWEEN 0 AND 100);
