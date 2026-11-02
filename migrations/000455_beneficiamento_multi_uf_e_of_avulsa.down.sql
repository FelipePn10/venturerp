ALTER TABLE production_orders DROP COLUMN IF EXISTS sales_order_code;
ALTER TABLE fiscal_configs DROP COLUMN IF EXISTS cbenef_suspensao;
ALTER TABLE fiscal_configs DROP COLUMN IF EXISTS obs_retorno_industrializacao;

SELECT instalar_conferencia_de_empresa();
