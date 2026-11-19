-- O 0 trocado por NULL não volta: os dois significam "genérico".
DROP TRIGGER IF EXISTS trg_preco_compra_tabela_fornecedor_generico ON purchase_price_tables;
DROP TRIGGER IF EXISTS trg_preco_compra_item_fornecedor_generico ON purchase_price_table_items;
DROP FUNCTION IF EXISTS erp_preco_compra_fornecedor_generico();
