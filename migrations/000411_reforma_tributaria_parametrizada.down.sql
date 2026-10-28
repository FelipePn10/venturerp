ALTER TABLE fiscal_exits
    DROP COLUMN IF EXISTS valor_cbs,
    DROP COLUMN IF EXISTS valor_ibs;
ALTER TABLE fiscal_exit_items
    DROP COLUMN IF EXISTS cbs_valor,
    DROP COLUMN IF EXISTS cbs_aliquota_efetiva,
    DROP COLUMN IF EXISTS ibs_mun_valor,
    DROP COLUMN IF EXISTS ibs_mun_aliquota_efetiva,
    DROP COLUMN IF EXISTS ibs_uf_valor,
    DROP COLUMN IF EXISTS ibs_uf_aliquota_efetiva,
    DROP COLUMN IF EXISTS ibscbs_base,
    DROP COLUMN IF EXISTS ibscbs_origem,
    DROP COLUMN IF EXISTS ibscbs_cclass_enviado,
    DROP COLUMN IF EXISTS ibscbs_cst_enviado,
    DROP COLUMN IF EXISTS ibscbs_cclass_trib;
DROP TABLE IF EXISTS ibscbs_cfop_classificacoes;
DROP TABLE IF EXISTS ibscbs_aliquotas;
DROP TABLE IF EXISTS ibscbs_classificacoes;
