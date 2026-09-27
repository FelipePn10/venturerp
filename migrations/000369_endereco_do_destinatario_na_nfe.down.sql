DROP INDEX IF EXISTS idx_fiscal_exits_customer;

ALTER TABLE public.fiscal_exits
    DROP COLUMN IF EXISTS dest_logradouro,
    DROP COLUMN IF EXISTS dest_numero,
    DROP COLUMN IF EXISTS dest_complemento,
    DROP COLUMN IF EXISTS dest_bairro,
    DROP COLUMN IF EXISTS dest_municipio,
    DROP COLUMN IF EXISTS dest_codigo_municipio,
    DROP COLUMN IF EXISTS dest_cep,
    DROP COLUMN IF EXISTS dest_email,
    DROP COLUMN IF EXISTS dest_telefone,
    DROP COLUMN IF EXISTS customer_code;
