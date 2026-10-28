-- Código de benefício fiscal (cBenef) do diferimento da configuração fiscal.
-- O diferimento parcial vem de fiscal_configs.icms_diferimento_percentual, mas
-- o cBenef que a SEFAZ exige no CST 51 (no PR, PR830001 — art. 28, I, do Anexo
-- VIII do RICMS/2017) só podia vir de uma regra de ICMS/IPI: sem regra, toda
-- venda interna a contribuinte ficava impedida. Vazio = sem código padrão.
ALTER TABLE fiscal_configs ADD COLUMN IF NOT EXISTS cbenef_diferimento varchar(10);
