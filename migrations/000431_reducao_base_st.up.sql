-- Percentual de redução da base da ST (pRedBCST) da linha: a base já ia
-- reduzida, mas o XML não informava o percentual.
ALTER TABLE fiscal_exit_items ADD COLUMN IF NOT EXISTS icms_st_reducao_base_pct numeric(7,4) NOT NULL DEFAULT 0;
