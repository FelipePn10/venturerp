ALTER TABLE invoice_types
    DROP COLUMN IF EXISTS csll_pct_presumption,
    DROP COLUMN IF EXISTS ir_pct_presumption,
    DROP COLUMN IF EXISTS exige_suframa,
    DROP COLUMN IF EXISTS gera_titulo_dev,
    DROP COLUMN IF EXISTS baixa_pedido,
    DROP COLUMN IF EXISTS cst_cofins,
    DROP COLUMN IF EXISTS cst_pis,
    DROP COLUMN IF EXISTS cst_ipi,
    DROP COLUMN IF EXISTS csosn_icms,
    DROP COLUMN IF EXISTS cst_icms,
    DROP COLUMN IF EXISTS model_nf;

ALTER TABLE sales_tables
    DROP COLUMN IF EXISTS observation,
    DROP COLUMN IF EXISTS icms_interestadual_por_dentro,
    DROP COLUMN IF EXISTS allow_items_below_cent,
    DROP COLUMN IF EXISTS base_date,
    DROP COLUMN IF EXISTS table_type,
    DROP COLUMN IF EXISTS composition;

DROP TYPE IF EXISTS base_date_enum;
DROP TYPE IF EXISTS table_type_enum;
DROP TYPE IF EXISTS table_composition_enum;
