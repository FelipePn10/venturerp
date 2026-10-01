-- Só as travas voltam atrás. A máscara original não é recuperável: era
-- apresentação, e o dígito é a mesma informação — não há o que restaurar.
BEGIN;

ALTER TABLE public.fiscal_classifications
    DROP CONSTRAINT IF EXISTS ck_fiscal_classifications_ncm_sem_mascara;
ALTER TABLE public.ncm_tax_table
    DROP CONSTRAINT IF EXISTS ck_ncm_tax_table_ncm_sem_mascara;

COMMIT;
