BEGIN;

ALTER TABLE public.fiscal_configs
    DROP COLUMN IF EXISTS logradouro,
    DROP COLUMN IF EXISTS numero,
    DROP COLUMN IF EXISTS complemento,
    DROP COLUMN IF EXISTS bairro,
    DROP COLUMN IF EXISTS municipio,
    DROP COLUMN IF EXISTS codigo_municipio,
    DROP COLUMN IF EXISTS cep,
    DROP COLUMN IF EXISTS telefone;

COMMIT;
