ALTER TABLE fiscal_exit_items
    DROP COLUMN IF EXISTS ipi_codigo_enquadramento,
    DROP COLUMN IF EXISTS icms_motivo_desoneracao,
    DROP COLUMN IF EXISTS icms_valor_desonerado;
ALTER TABLE icms_ipi_tax_params DROP COLUMN IF EXISTS ipi_codigo_enquadramento;
SELECT instalar_conferencia_de_empresa();
