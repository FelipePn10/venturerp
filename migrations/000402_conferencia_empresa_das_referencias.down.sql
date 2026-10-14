DO $$
DECLARE t record;
BEGIN
    FOR t IN
        SELECT c.relname AS tabela
          FROM pg_trigger g JOIN pg_class c ON c.oid = g.tgrelid
         WHERE g.tgname = 'zz_conferir_empresa_ref'
    LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS zz_conferir_empresa_ref ON %I', t.tabela);
        EXECUTE format('DROP FUNCTION IF EXISTS %I()', 'conferir_empresa_' || t.tabela);
    END LOOP;
END $$;

DROP FUNCTION IF EXISTS instalar_conferencia_de_empresa();
DROP FUNCTION IF EXISTS conferencia_empresa_vinculos();
DROP FUNCTION IF EXISTS conferencia_empresa_excecao(text);
