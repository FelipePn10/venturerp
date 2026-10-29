package testutil

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FuncaoApagarEmpresas apaga as empresas e, antes delas, as linhas de todas
// as tabelas que as referenciam (por enterprise.id ou enterprise.code), em
// passadas: a linha que ainda tem filha fica para a passada seguinte. A
// mesma função está em scripts/limpar-empresas-de-teste.sql.
//
// Os testes que criam empresa apagavam só a empresa; a linha de parâmetros
// semeados, de notificação ou de ordem que o teste gerou segurava a empresa,
// e o banco de teste acumulava empresas "Isolamento A/B" a cada execução.
const FuncaoApagarEmpresas = `
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
$f$`

// ApagarEmpresas apaga as empresas de teste com tudo o que as referencia.
// Empresa que ainda sobrar (filha de filha sem coluna de empresa) é avisada
// no log do teste, não derruba o teste: é limpeza.
func ApagarEmpresas(t *testing.T, pool *pgxpool.Pool, ids ...int64) {
	t.Helper()
	if len(ids) == 0 {
		return
	}
	ctx := context.Background()
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Logf("limpeza das empresas de teste: %v", err)
		return
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, FuncaoApagarEmpresas); err != nil {
		t.Logf("limpeza das empresas de teste: %v", err)
		return
	}
	var restantes int
	if err := conn.QueryRow(ctx, `SELECT pg_temp.apagar_empresas($1)`, ids).Scan(&restantes); err != nil {
		t.Logf("limpeza das empresas de teste: %v", err)
		return
	}
	if restantes > 0 {
		t.Logf("limpeza: %d empresa(s) de teste ficaram no banco (filha de filha sem coluna de empresa)", restantes)
	}
}

// ApagarEmpresasPorCodigo é ApagarEmpresas pelo código da empresa (para o teste
// que cria as empresas num INSERT só e não guarda os ids). Registre logo depois
// de criar: t.Cleanup roda em ordem inversa, então ela roda por último e
// termina o que a limpeza do próprio teste não conseguiu apagar.
func ApagarEmpresasPorCodigo(t *testing.T, pool *pgxpool.Pool, codigos ...int64) {
	t.Helper()
	if len(codigos) == 0 {
		return
	}
	rows, err := pool.Query(context.Background(), `SELECT id FROM enterprise WHERE code = ANY($1)`, codigos)
	if err != nil {
		t.Logf("limpeza das empresas de teste: %v", err)
		return
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	ApagarEmpresas(t, pool, ids...)
}
