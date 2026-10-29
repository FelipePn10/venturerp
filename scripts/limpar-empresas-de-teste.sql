-- ============================================================================
-- Faxina do banco da suíte de integração: apaga as empresas que os testes
-- criaram e não conseguiram apagar (a linha de parâmetros, notificação ou ordem
-- gerada pelo teste segurava a empresa). Fica só a empresa semeada (código 1,
-- scripts/seed-test.sql).
--
-- SÓ RODA em banco cujo nome contém "test": em qualquer outro, aborta.
--
-- Uso: psql "$TEST_DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/limpar-empresas-de-teste.sql
-- A função é a mesma de internal/infrastructure/testutil/apagar_empresas.go.
-- ============================================================================

\set ON_ERROR_STOP on

DO $$
BEGIN
    IF current_database() NOT ILIKE '%test%' THEN
        RAISE EXCEPTION 'limpar-empresas-de-teste: o banco % não é de teste; nada foi apagado', current_database();
    END IF;
END $$;
CREATE OR REPLACE FUNCTION pg_temp.apagar_empresas(p_ids bigint[]) RETURNS integer
LANGUAGE plpgsql AS $f$
DECLARE
    r record;
    codigos bigint[];
    passada int;
    restantes int;
    empresa bigint;
BEGIN
    SELECT array_agg(code) INTO codigos FROM enterprise WHERE id = ANY(p_ids);
    IF codigos IS NULL THEN
        RETURN 0;
    END IF;
    FOR passada IN 1..8 LOOP
        FOR r IN
            SELECT DISTINCT n.nspname AS esquema, cl.relname AS tabela, a.attname AS coluna, ra.attname AS alvo
              FROM pg_constraint c
              JOIN pg_class cl ON cl.oid = c.conrelid
              JOIN pg_namespace n ON n.oid = cl.relnamespace
              JOIN pg_class t ON t.oid = c.confrelid
              JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
              JOIN pg_attribute ra ON ra.attrelid = c.confrelid AND ra.attnum = c.confkey[1]
             WHERE c.contype = 'f' AND t.relname = 'enterprise' AND cl.relname <> 'enterprise'
               AND array_length(c.conkey, 1) = 1
        LOOP
            BEGIN
                EXECUTE format('DELETE FROM %I.%I WHERE %I = ANY($1)', r.esquema, r.tabela, r.coluna)
                    USING CASE WHEN r.alvo = 'code' THEN codigos ELSE p_ids END;
            EXCEPTION
                WHEN foreign_key_violation OR restrict_violation THEN
                    NULL; -- ainda tem filha: fica para a próxima passada
                WHEN OTHERS THEN
                    -- Tabela imutável (auditoria): gatilho recusa o DELETE. Só
                    -- nesta instrução os gatilhos ficam desligados (exige
                    -- superusuário; sem ele, a linha fica).
                    BEGIN
                        PERFORM set_config('session_replication_role', 'replica', true);
                        EXECUTE format('DELETE FROM %I.%I WHERE %I = ANY($1)', r.esquema, r.tabela, r.coluna)
                            USING CASE WHEN r.alvo = 'code' THEN codigos ELSE p_ids END;
                        PERFORM set_config('session_replication_role', 'origin', true);
                    EXCEPTION WHEN OTHERS THEN
                        PERFORM set_config('session_replication_role', 'origin', true);
                    END;
            END;
        END LOOP;
        -- Netos: linhas sem coluna de empresa que apontam para linhas da
        -- empresa (capacity_requirements → machine_types). Apagadas primeiro
        -- na passada seguinte, a filha delas sai depois.
        FOR r IN
            SELECT DISTINCT n.nspname AS esquema, filho.relname AS tabela, fa.attname AS coluna,
                   pn.nspname AS esquema_pai, pai.relname AS tabela_pai, pa.attname AS chave_pai,
                   ea.attname AS empresa_pai, er.attname AS alvo
              FROM pg_constraint fk
              JOIN pg_class filho ON filho.oid = fk.conrelid
              JOIN pg_namespace n ON n.oid = filho.relnamespace
              JOIN pg_class pai ON pai.oid = fk.confrelid
              JOIN pg_namespace pn ON pn.oid = pai.relnamespace
              JOIN pg_attribute fa ON fa.attrelid = fk.conrelid AND fa.attnum = fk.conkey[1]
              JOIN pg_attribute pa ON pa.attrelid = fk.confrelid AND pa.attnum = fk.confkey[1]
              JOIN pg_constraint efk ON efk.conrelid = pai.oid AND efk.contype = 'f' AND array_length(efk.conkey, 1) = 1
              JOIN pg_class ent ON ent.oid = efk.confrelid AND ent.relname = 'enterprise'
              JOIN pg_attribute ea ON ea.attrelid = pai.oid AND ea.attnum = efk.conkey[1]
              JOIN pg_attribute er ON er.attrelid = efk.confrelid AND er.attnum = efk.confkey[1]
             WHERE fk.contype = 'f' AND array_length(fk.conkey, 1) = 1 AND pai.relname <> 'enterprise'
               AND filho.oid <> pai.oid
        LOOP
            BEGIN
                EXECUTE format('DELETE FROM %I.%I WHERE %I IN (SELECT %I FROM %I.%I WHERE %I = ANY($1))',
                               r.esquema, r.tabela, r.coluna, r.chave_pai, r.esquema_pai, r.tabela_pai, r.empresa_pai)
                    USING CASE WHEN r.alvo = 'code' THEN codigos ELSE p_ids END;
            EXCEPTION WHEN OTHERS THEN
                NULL; -- tem filha ou é imutável: a passada seguinte tenta de novo
            END;
        END LOOP;
        -- Uma a uma: a empresa ainda presa não impede as outras.
        FOREACH empresa IN ARRAY p_ids LOOP
            BEGIN
                DELETE FROM enterprise WHERE id = empresa;
            EXCEPTION WHEN foreign_key_violation OR restrict_violation THEN
                NULL;
            END;
        END LOOP;
        SELECT count(*) INTO restantes FROM enterprise WHERE id = ANY(p_ids);
        EXIT WHEN restantes = 0;
    END LOOP;
    SELECT count(*) INTO restantes FROM enterprise WHERE id = ANY(p_ids);
    RETURN restantes;
END
$f$;

SELECT count(*) AS empresas_de_teste_antes FROM enterprise WHERE code <> 1;
SELECT pg_temp.apagar_empresas(ARRAY(SELECT id FROM enterprise WHERE code <> 1)) AS empresas_que_sobraram;
