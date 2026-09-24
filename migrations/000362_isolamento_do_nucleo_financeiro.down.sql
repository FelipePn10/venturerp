DROP INDEX IF EXISTS uq_tax_assessments_empresa_imposto_competencia;

-- A chave global só volta se os dados couberem nela: com duas empresas
-- apurando a mesma competência, recriá-la falharia e travaria o rollback.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT imposto, competencia FROM tax_assessments GROUP BY imposto, competencia HAVING count(*) > 1
    ) THEN
        ALTER TABLE tax_assessments ADD CONSTRAINT tax_assessments_imposto_competencia_key UNIQUE (imposto, competencia);
    END IF;
END $$;

DO $$
DECLARE
    alvo TEXT;
BEGIN
    FOREACH alvo IN ARRAY ARRAY['contas_pagar','contas_receber','fluxo_caixa','tax_assessments']
    LOOP
        EXECUTE format('ALTER TABLE %I DROP CONSTRAINT IF EXISTS %I', alvo, 'fk_' || alvo || '_enterprise');
        EXECUTE format('DROP INDEX IF EXISTS %I', 'idx_' || alvo || '_enterprise');
        EXECUTE format('ALTER TABLE %I DROP COLUMN IF EXISTS enterprise_id', alvo);
    END LOOP;
END $$;
