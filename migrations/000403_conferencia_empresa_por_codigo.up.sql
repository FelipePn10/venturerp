-- Conferência de empresa também nas colunas de código sem FK (continua a 000402).
--
-- `fiscal_exits.sales_order_code`, `sales_orders.customer_code`,
-- `purchase_orders.supplier_code`… apontam para outra tabela pelo CÓDIGO, sem FK
-- — e a 000402 só via FK e item_code. Agora a coluna X_code sem FK que tem uma
-- tabela X(e)s com `code` e empresa também é conferida.
--
-- O código pode repetir entre empresas (cada empresa tem a sua cópia do cliente,
-- com o mesmo código), então a regra é: é violação quando o código NÃO existe na
-- empresa do registro e EXISTE em outra. Código que não existe em lugar nenhum
-- segue livre (como sem FK), e cadastro global (empresa nula) vale para todas.
--
-- Fora, além das exceções da 000402: o plano de corte (fora do escopo desta
-- conferência), a demanda do planejamento (`sales_order_demands`, fluxo
-- interfábricas) e `warehouse_code`, cujo nome coincide com a tabela `warehouse`
-- sem ser a mesma coisa.

CREATE OR REPLACE FUNCTION conferencia_empresa_excecao_coluna(tabela text, coluna text) RETURNS boolean
LANGUAGE sql IMMUTABLE AS $$
    SELECT tabela LIKE 'cutting\_%'
        OR tabela = 'sales_order_demands'
        OR coluna = 'warehouse_code'
$$;

DROP FUNCTION IF EXISTS conferencia_empresa_vinculos();

-- Vínculos conferidos (filho, coluna, pai, coluna_pai e a convenção de empresa
-- de cada lado). Usa o catálogo do pg direto: information_schema é lento demais
-- para ser lido a cada tabela.
CREATE OR REPLACE FUNCTION conferencia_empresa_vinculos()
RETURNS TABLE (filho text, coluna text, pai text, coluna_pai text, empresa_filho text, empresa_pai text, por_codigo boolean)
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
    ),
    -- Colunas X_code sem FK que apontam para a tabela X(e)s pelo código. O código
    -- pode repetir entre empresas (cada uma tem a sua cópia do cliente), então a
    -- conferência é "existe na minha empresa", não "o pai é da minha empresa".
    porcodigo AS (
        SELECT c.relname::text AS filho, a.attname::text AS coluna, p.relname::text AS pai, 'code'::text AS coluna_pai
          FROM pg_class c
          JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = 'public'
          JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped
          JOIN pg_class p ON p.relnamespace = n.oid AND p.relkind IN ('r', 'p')
           AND p.relname IN (regexp_replace(a.attname, '_code$', '') || 's',
                             regexp_replace(a.attname, '_code$', '') || 'es',
                             regexp_replace(regexp_replace(a.attname, '_code$', ''), 'y$', 'ies'))
          JOIN pg_attribute pa ON pa.attrelid = p.oid AND pa.attname = 'code' AND NOT pa.attisdropped
         WHERE c.relkind IN ('r', 'p') AND c.relname <> p.relname
           AND a.attname ~ '_code$' AND a.attname NOT IN ('enterprise_code', 'item_code')
           AND a.atttypid IN ('bigint'::regtype, 'integer'::regtype)
           AND NOT EXISTS (SELECT 1 FROM pg_constraint k
                            WHERE k.conrelid = c.oid AND k.contype = 'f' AND k.conkey = ARRAY[a.attnum])
           AND NOT conferencia_empresa_excecao_coluna(c.relname::text, a.attname::text)
    )
    SELECT fk.filho, fk.coluna, fk.pai, fk.coluna_pai,
           CASE WHEN cf.tem_id THEN 'enterprise_id' ELSE 'enterprise_code' END,
           CASE WHEN cp.tem_id THEN 'enterprise_id' ELSE 'enterprise_code' END,
           false
      FROM fk
      JOIN col cf ON cf.t = fk.filho
      JOIN col cp ON cp.t = fk.pai
     WHERE (cf.tem_id OR cf.tem_code) AND (cp.tem_id OR cp.tem_code)
       AND NOT conferencia_empresa_excecao(fk.filho)
    UNION ALL
    SELECT pc.filho, pc.coluna, pc.pai, pc.coluna_pai,
           CASE WHEN cf.tem_id THEN 'enterprise_id' ELSE 'enterprise_code' END,
           CASE WHEN cp.tem_id THEN 'enterprise_id' ELSE 'enterprise_code' END,
           true
      FROM porcodigo pc
      JOIN col cf ON cf.t = pc.filho
      JOIN col cp ON cp.t = pc.pai
     WHERE (cf.tem_id OR cf.tem_code) AND (cp.tem_id OR cp.tem_code)
       AND NOT conferencia_empresa_excecao(pc.filho);
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
            IF v.por_codigo THEN
                corpo := corpo || format($bloco$
    IF NEW.%1$I IS NOT NULL AND NEW.%2$I IS NOT NULL
       AND (TG_OP = 'INSERT' OR NEW.%1$I IS DISTINCT FROM OLD.%1$I OR NEW.%2$I IS DISTINCT FROM OLD.%2$I)
       AND NOT EXISTS (SELECT 1 FROM %4$I p WHERE p.%5$I = NEW.%1$I AND (%3$s IS NULL OR %3$s = %6$s))
       AND EXISTS (SELECT 1 FROM %4$I p WHERE p.%5$I = NEW.%1$I) THEN
        RAISE EXCEPTION '%7$s.%1$s aponta para %4$s de outra empresa (%%)', NEW.%1$I
            USING ERRCODE = '23503', CONSTRAINT = 'conferencia_empresa_%7$s_%1$s', TABLE = %8$L;
    END IF;
$bloco$, v.coluna, t.empresa_filho, pai_e, v.pai, v.coluna_pai, filho_e, t.filho, t.filho);
                CONTINUE;
            END IF;
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
