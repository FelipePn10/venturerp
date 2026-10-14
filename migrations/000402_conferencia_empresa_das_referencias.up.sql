-- Conferência de empresa nas referências entre tabelas.
--
-- A FK garante que o registro apontado EXISTE; não garante que é da MESMA
-- empresa. Um cadastro da empresa B apontando para o item, o cliente ou o
-- depósito da empresa A passava pelo banco e só aparecia (quando aparecia) na
-- auditoria scripts/audit-referencias-entre-empresas.sh. Este gatilho fecha a
-- porta na gravação.
--
-- instalar_conferencia_de_empresa() lê o catálogo e gera, para cada tabela com
-- coluna de empresa, uma função de gatilho com as consultas ESTÁTICAS de cada
-- vínculo (FK de uma coluna para tabela que também tem empresa, mais item_code,
-- que não tem FK mas é sempre item da empresa). É idempotente: migration que
-- criar tabela ou FK nova chama de novo `SELECT instalar_conferencia_de_empresa();`
-- — o teste de integração TestConferenciaDeEmpresaCobreTodosOsVinculos falha
-- se esquecer.
--
-- As duas convenções (enterprise_id / enterprise_code) são comparadas pela
-- tabela enterprise. Não confere quando:
--   * a empresa do filho ou do pai é nula (cadastro global legado);
--   * o pai não existe (é papel da FK — e item_code sem item segue livre);
--   * no UPDATE, nem a referência nem a empresa mudaram (linha antiga intocada).
--
-- Exceções, por serem cruzadas DE PROPÓSITO:
--   * fluxo interfábricas: mrp_*, planning_*, planned_orders e
--     production_plan_inter_factories — o plano de uma fábrica carrega demanda
--     e itens da fábrica que ela abastece;
--   * tabelas de vínculo *_enterprises — ligam um cadastro a OUTRAS empresas.
--
-- A violação sai como foreign_key_violation (23503), que a API já traduz para
-- "um dos vínculos informados não existe na empresa autenticada" — do ponto de
-- vista de quem grava, é exatamente isso.

CREATE OR REPLACE FUNCTION conferencia_empresa_excecao(tabela text) RETURNS boolean
LANGUAGE sql IMMUTABLE AS $$
    SELECT tabela LIKE 'mrp\_%'
        OR tabela LIKE 'planning\_%'
        OR tabela IN ('planned_orders', 'production_plan_inter_factories')
        OR tabela LIKE '%\_enterprises'
$$;

-- Vínculos conferidos (filho, coluna, pai, coluna_pai e a convenção de empresa
-- de cada lado). Usa o catálogo do pg direto: information_schema é lento demais
-- para ser lido a cada tabela.
CREATE OR REPLACE FUNCTION conferencia_empresa_vinculos()
RETURNS TABLE (filho text, coluna text, pai text, coluna_pai text, empresa_filho text, empresa_pai text)
LANGUAGE plpgsql STABLE AS $vinculos$
BEGIN
    RETURN QUERY
    WITH col AS (
        SELECT c.relname::text AS t,
               bool_or(a.attname = 'enterprise_id')   AS tem_id,
               bool_or(a.attname = 'enterprise_code') AS tem_code
          FROM pg_class c
          JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = 'public'
          JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped
         WHERE c.relkind IN ('r', 'p')
         GROUP BY c.relname
    ),
    fk AS (
        SELECT cl.relname::text AS filho, a.attname::text AS coluna, pcl.relname::text AS pai, pa.attname::text AS coluna_pai
          FROM pg_constraint k
          JOIN pg_class cl   ON cl.oid = k.conrelid
          JOIN pg_class pcl  ON pcl.oid = k.confrelid
          JOIN pg_namespace n ON n.oid = cl.relnamespace AND n.nspname = 'public'
          JOIN pg_attribute a  ON a.attrelid = k.conrelid  AND a.attnum = k.conkey[1]
          JOIN pg_attribute pa ON pa.attrelid = k.confrelid AND pa.attnum = k.confkey[1]
         WHERE k.contype = 'f' AND array_length(k.conkey, 1) = 1
           AND cl.relname <> pcl.relname AND pcl.relname <> 'enterprise'
           AND a.attname NOT IN ('enterprise_id', 'enterprise_code')
        UNION
        SELECT c.relname::text, 'item_code', 'items', 'code'
          FROM pg_class c
          JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = 'public'
          JOIN pg_attribute a ON a.attrelid = c.oid AND a.attname = 'item_code' AND NOT a.attisdropped
         WHERE c.relkind IN ('r', 'p') AND c.relname <> 'items'
           AND a.atttypid IN ('bigint'::regtype, 'integer'::regtype)
    )
    SELECT fk.filho, fk.coluna, fk.pai, fk.coluna_pai,
           CASE WHEN cf.tem_id THEN 'enterprise_id' ELSE 'enterprise_code' END,
           CASE WHEN cp.tem_id THEN 'enterprise_id' ELSE 'enterprise_code' END
      FROM fk
      JOIN col cf ON cf.t = fk.filho
      JOIN col cp ON cp.t = fk.pai
     WHERE (cf.tem_id OR cf.tem_code) AND (cp.tem_id OR cp.tem_code)
       AND NOT conferencia_empresa_excecao(fk.filho);
END
$vinculos$;

CREATE OR REPLACE FUNCTION instalar_conferencia_de_empresa() RETURNS integer
LANGUAGE plpgsql AS $instalador$
DECLARE
    t       record;
    v       record;
    corpo   text;
    filho_e text;   -- empresa do filho, sempre no espaço de enterprise.id
    pai_e   text;   -- empresa do pai, idem
    funcao  text;
    n       integer := 0;
BEGIN
    CREATE TEMP TABLE IF NOT EXISTS _vinculos ON COMMIT DROP AS SELECT * FROM conferencia_empresa_vinculos();

    -- Remove as conferências de tabelas que deixaram de ter vínculo (ou viraram exceção).
    FOR t IN
        SELECT c.relname AS tabela
          FROM pg_trigger g JOIN pg_class c ON c.oid = g.tgrelid
         WHERE g.tgname = 'zz_conferir_empresa_ref'
           AND c.relname NOT IN (SELECT x.filho FROM _vinculos x)
    LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS zz_conferir_empresa_ref ON %I', t.tabela);
        EXECUTE format('DROP FUNCTION IF EXISTS %I()', 'conferir_empresa_' || t.tabela);
    END LOOP;

    FOR t IN SELECT DISTINCT x.filho, x.empresa_filho FROM _vinculos x ORDER BY x.filho LOOP
        corpo := '';
        filho_e := CASE WHEN t.empresa_filho = 'enterprise_id' THEN 'NEW.enterprise_id'
                        ELSE '(SELECT e.id FROM enterprise e WHERE e.code = NEW.enterprise_code)' END;
        FOR v IN SELECT * FROM _vinculos x WHERE x.filho = t.filho ORDER BY x.coluna LOOP
            pai_e := CASE WHEN v.empresa_pai = 'enterprise_id' THEN 'p.enterprise_id'
                          ELSE '(SELECT e.id FROM enterprise e WHERE e.code = p.enterprise_code)' END;
            corpo := corpo || format($bloco$
    IF NEW.%1$I IS NOT NULL AND NEW.%2$I IS NOT NULL
       AND (TG_OP = 'INSERT' OR NEW.%1$I IS DISTINCT FROM OLD.%1$I OR NEW.%2$I IS DISTINCT FROM OLD.%2$I) THEN
        SELECT %3$s INTO empresa_pai FROM %4$I p WHERE p.%5$I = NEW.%1$I LIMIT 1;
        IF empresa_pai IS NOT NULL AND empresa_pai IS DISTINCT FROM %6$s THEN
            RAISE EXCEPTION '%7$s.%1$s aponta para %4$s de outra empresa (%%)', NEW.%1$I
                USING ERRCODE = '23503', CONSTRAINT = 'conferencia_empresa_%7$s_%1$s', TABLE = %8$L;
        END IF;
    END IF;
$bloco$, v.coluna, t.empresa_filho, pai_e, v.pai, v.coluna_pai, filho_e, t.filho, t.filho);
        END LOOP;

        funcao := 'conferir_empresa_' || t.filho;
        EXECUTE format($f$
CREATE OR REPLACE FUNCTION %I() RETURNS trigger LANGUAGE plpgsql AS $corpo$
DECLARE
    empresa_pai bigint;
BEGIN%s
    RETURN NEW;
END
$corpo$$f$, funcao, corpo);
        EXECUTE format('DROP TRIGGER IF EXISTS zz_conferir_empresa_ref ON %I', t.filho);
        EXECUTE format('CREATE TRIGGER zz_conferir_empresa_ref BEFORE INSERT OR UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION %I()',
                       t.filho, funcao);
        n := n + 1;
    END LOOP;
    DROP TABLE _vinculos;
    RETURN n;
END
$instalador$;

SELECT instalar_conferencia_de_empresa();
