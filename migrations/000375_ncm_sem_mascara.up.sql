-- NCM gravado sempre sem máscara, nos dois cadastros que o motor fiscal casa.
--
-- O problema: a tabela tributária (ncm_tax_table) e a classificação fiscal do item
-- (fiscal_classifications) são cadastros independentes e nada obrigava os dois a
-- usarem a mesma máscara. O motor casa um com o outro por string exata e, quando
-- não acha, NÃO reclama — cai na alíquota padrão. "8466.20.90" no item com
-- "84662090" na tabela produzia nota com IPI/PIS/COFINS zerados e nenhum erro.
-- Encontrado na carga da Usimac: 94 linhas com máscara convivendo com 1 sem.
--
-- A forma canônica é só dígitos, que é também a que a SEFAZ exige no campo <NCM>
-- do XML (máscara é recusada na autorização).
--
-- Fora do escopo de propósito: ibpt_rates.ncm, icms_ipi_tax_params.ncm_code e
-- icms_reduction_substitutions.ncm_code podem guardar PREFIXO (regra por capítulo),
-- então normalizar o comprimento ali mudaria o significado da regra. Os itens de
-- documento já emitido (fiscal_exit_items, fiscal_entry_items) também ficam como
-- estão: são registro histórico, e o motor passou a normalizar na leitura.

BEGIN;

-- 1. Desduplicar o que colapsa ao perder a máscara (ncm é UNIQUE).
--    Mantém a linha mais recente — é a correção mais nova do usuário — e avisa no
--    log quais linhas foram descartadas, para a conferência pós-release.
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT regexp_replace(ncm, '[^0-9]', '', 'g') AS limpo,
               array_agg(ncm ORDER BY created_at DESC, id DESC)  AS mascaras,
               array_agg(id  ORDER BY created_at DESC, id DESC)  AS ids
        FROM public.ncm_tax_table
        GROUP BY 1
        HAVING count(*) > 1
    LOOP
        RAISE NOTICE 'NCM % estava cadastrado % vezes (%); mantida a linha id=%',
            r.limpo, array_length(r.ids, 1), r.mascaras, r.ids[1];
        DELETE FROM public.ncm_tax_table
        WHERE id = ANY(r.ids[2:array_length(r.ids, 1)]);
    END LOOP;
END $$;

-- 2. Normalizar.
UPDATE public.ncm_tax_table
SET ncm = regexp_replace(ncm, '[^0-9]', '', 'g')
WHERE ncm <> regexp_replace(ncm, '[^0-9]', '', 'g');

UPDATE public.fiscal_classifications
SET ncm = regexp_replace(ncm, '[^0-9]', '', 'g')
WHERE ncm IS NOT NULL
  AND ncm <> regexp_replace(ncm, '[^0-9]', '', 'g');

-- 3. Impedir que a máscara volte.
--    NOT VALID de propósito: passa a valer para toda gravação nova, sem recusar a
--    migração por causa de alguma linha legada fora do padrão de 8 dígitos (que
--    existe em base antiga e não é problema desta mudança).
ALTER TABLE public.ncm_tax_table
    DROP CONSTRAINT IF EXISTS ck_ncm_tax_table_ncm_sem_mascara;
ALTER TABLE public.ncm_tax_table
    ADD CONSTRAINT ck_ncm_tax_table_ncm_sem_mascara
    CHECK (ncm ~ '^[0-9]{8}$') NOT VALID;

ALTER TABLE public.fiscal_classifications
    DROP CONSTRAINT IF EXISTS ck_fiscal_classifications_ncm_sem_mascara;
ALTER TABLE public.fiscal_classifications
    ADD CONSTRAINT ck_fiscal_classifications_ncm_sem_mascara
    CHECK (ncm IS NULL OR ncm ~ '^[0-9]{8}$') NOT VALID;

COMMIT;
