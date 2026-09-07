-- Correção de dados: as telas do configurador gravavam o código de negócio do
-- item (o que o usuário digita, ex.: 900700) nas tabelas cfg_*, enquanto o
-- painel da estrutura de produto procura pela chave interna de `items.code`.
-- O resultado eram características órfãs: cadastradas, mas invisíveis para o
-- configurador. O handler já traduz as duas chaves; aqui as linhas antigas são
-- reapontadas.
--
-- Só é reapontada a linha cujo valor atual NÃO existe como `items.code` e
-- existe como `items.business_code` — assim quem já estava correto não é
-- tocado, e a migração é idempotente.

DO $$
DECLARE
    alvo RECORD;
BEGIN
    FOR alvo IN
        SELECT * FROM (VALUES
            ('cfg_item_characteristics',        'item_code'),
            ('cfg_item_descriptions',           'item_code'),
            ('cfg_item_rules',                  'item_code'),
            ('cfg_characteristic_receiving_items', 'item_code'),
            ('cfg_equivalent_rules',            'parent_item_code'),
            ('cfg_equivalent_rules',            'child_item_code')
        ) AS t(tabela, coluna)
    LOOP
        EXECUTE format($f$
            UPDATE %I AS alvo
               SET %I = itens.code
              FROM items AS itens
             WHERE itens.business_code = alvo.%I::text
               AND NOT EXISTS (SELECT 1 FROM items AS ok WHERE ok.code = alvo.%I)
        $f$, alvo.tabela, alvo.coluna, alvo.coluna, alvo.coluna);
    END LOOP;
END $$;
