ALTER TABLE fiscal_exit_items
    DROP COLUMN IF EXISTS fcp_uf_dest_valor,
    DROP COLUMN IF EXISTS fcp_uf_dest_pct,
    DROP COLUMN IF EXISTS icms_uf_dest_valor,
    DROP COLUMN IF EXISTS icms_aliq_interestadual,
    DROP COLUMN IF EXISTS icms_uf_dest_aliq_interna,
    DROP COLUMN IF EXISTS icms_uf_dest_base,
    DROP COLUMN IF EXISTS valor_credito_simples,
    DROP COLUMN IF EXISTS aliq_credito_simples;

ALTER TABLE icms_ipi_tax_params
    DROP CONSTRAINT IF EXISTS chk_icms_ipi_tax_params_mva,
    DROP CONSTRAINT IF EXISTS chk_icms_ipi_tax_params_finalidade_compra,
    DROP COLUMN IF EXISTS mva_ajustar_interestadual,
    DROP COLUMN IF EXISTS mva_pct,
    DROP COLUMN IF EXISTS finalidade_compra;

DROP INDEX IF EXISTS idx_fiscal_exits_invoice_type;
ALTER TABLE fiscal_exits
    DROP CONSTRAINT IF EXISTS fiscal_exits_invoice_type_id_empresa_fkey,
    DROP CONSTRAINT IF EXISTS chk_fiscal_exits_finalidade_compra,
    DROP COLUMN IF EXISTS invoice_type_id,
    DROP COLUMN IF EXISTS finalidade_compra;

ALTER TABLE sales_orders
    DROP CONSTRAINT IF EXISTS chk_sales_orders_finalidade_compra,
    DROP COLUMN IF EXISTS finalidade_compra;

ALTER TABLE customers
    DROP CONSTRAINT IF EXISTS chk_customers_finalidade_compra,
    DROP COLUMN IF EXISTS finalidade_compra;

ALTER TABLE fiscal_configs
    DROP CONSTRAINT IF EXISTS chk_fiscal_configs_credito_simples,
    DROP COLUMN IF EXISTS aliquota_credito_simples,
    DROP COLUMN IF EXISTS exigir_regra_icms_ipi;
