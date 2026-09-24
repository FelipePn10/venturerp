DROP INDEX IF EXISTS uq_plano_contas_empresa_codigo;
DROP INDEX IF EXISTS uq_centros_custo_empresa_codigo;

-- A unicidade global só volta se os dados couberem nela; com duas empresas
-- usando o mesmo código, recriar a restrição falharia e travaria o rollback.
DO $$
BEGIN
    IF NOT EXISTS (SELECT codigo FROM plano_contas GROUP BY codigo HAVING count(*) > 1) THEN
        ALTER TABLE plano_contas ADD CONSTRAINT plano_contas_codigo_key UNIQUE (codigo);
    END IF;
    IF NOT EXISTS (SELECT codigo FROM centros_custo GROUP BY codigo HAVING count(*) > 1) THEN
        ALTER TABLE centros_custo ADD CONSTRAINT centros_custo_codigo_key UNIQUE (codigo);
    END IF;
END $$;

DO $$
DECLARE
    alvo TEXT;
BEGIN
    FOREACH alvo IN ARRAY ARRAY['contas_bancarias','condicoes_pagamento','payment_conditions',
                                'plano_contas','centros_custo','restrictions','allocation_bases']
    LOOP
        EXECUTE format('ALTER TABLE %I DROP CONSTRAINT IF EXISTS %I', alvo, 'fk_' || alvo || '_enterprise');
        EXECUTE format('DROP INDEX IF EXISTS %I', 'idx_' || alvo || '_enterprise');
        EXECUTE format('ALTER TABLE %I DROP COLUMN IF EXISTS enterprise_id', alvo);
    END LOOP;
END $$;
