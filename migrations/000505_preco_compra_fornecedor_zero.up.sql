-- Linha de tabela de preço de compra com fornecedor 0 = preço GENÉRICO.
-- O índice único (COALESCE(supplier_code, 0)) e a tela já tratavam 0 como
-- genérico, mas a busca de preço do pedido trata só o vazio (NULL): as linhas
-- gravadas com 0 não casavam com fornecedor nenhum e a tabela inteira era
-- ignorada (Tecnofer: as 5 linhas da tabela estavam com 0). O 0 vira NULL e um
-- gatilho impede que volte, por qualquer caminho de gravação.
-- Não há colisão no índice único: 0 e NULL já eram a mesma chave nele.

UPDATE purchase_price_table_items SET supplier_code = NULL WHERE supplier_code <= 0;
UPDATE purchase_price_tables      SET supplier_code = NULL WHERE supplier_code <= 0;

CREATE OR REPLACE FUNCTION erp_preco_compra_fornecedor_generico() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.supplier_code IS NOT NULL AND NEW.supplier_code <= 0 THEN
        NEW.supplier_code := NULL;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_preco_compra_item_fornecedor_generico
    BEFORE INSERT OR UPDATE OF supplier_code ON purchase_price_table_items
    FOR EACH ROW EXECUTE FUNCTION erp_preco_compra_fornecedor_generico();

CREATE TRIGGER trg_preco_compra_tabela_fornecedor_generico
    BEFORE INSERT OR UPDATE OF supplier_code ON purchase_price_tables
    FOR EACH ROW EXECUTE FUNCTION erp_preco_compra_fornecedor_generico();
