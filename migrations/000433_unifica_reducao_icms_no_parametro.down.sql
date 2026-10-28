-- Desfaz a unificação: remove as regras copiadas da VFIS0330 (a tabela antiga
-- continua com os originais) e as colunas novas. Os percentuais voltam a 2
-- casas — valor com mais casas é arredondado.
DELETE FROM icms_ipi_tax_params WHERE origem_reducao_id IS NOT NULL;
DROP INDEX IF EXISTS uq_icms_ipi_tax_params_origem_reducao;
ALTER TABLE icms_ipi_tax_params
    DROP COLUMN IF EXISTS origem_reducao_id,
    DROP COLUMN IF EXISTS supplier_code;
SELECT instalar_conferencia_de_empresa();

DO $$
DECLARE c record;
BEGIN
    FOR c IN SELECT column_name FROM information_schema.columns
              WHERE table_schema = 'public' AND table_name = 'icms_ipi_tax_params'
                AND data_type = 'numeric' AND numeric_precision = 7 AND numeric_scale = 4
                AND column_name NOT IN ('mva_pct')
    LOOP
        EXECUTE format('ALTER TABLE icms_ipi_tax_params ALTER COLUMN %I TYPE numeric(5,2)', c.column_name);
    END LOOP;
END $$;
