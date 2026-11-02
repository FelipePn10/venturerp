-- Beneficiamento: componente da estrutura que o CLIENTE fornece.
--
-- Na industrialização por encomenda a matéria-prima chega por NF-e de remessa
-- (CFOP 5901) e nunca é da empresa: fica no razão de terceiros
-- (customer_material_*, migração 000370), não em stock_balances. Sem esta marca,
-- a estrutura do produto beneficiado tratava o material do cliente como
-- componente comum — o MRP sugeria COMPRAR o que o cliente manda, a baixa
-- automática consumia estoque próprio que não existe, o custo-padrão somava
-- material que não é custo da empresa e a conclusão da OF dava entrada do
-- produto processado no estoque PRÓPRIO, com valor.
--
-- É marca da LINHA da estrutura (como is_coproduct), não do item: a mesma
-- barra pode ser comprada para um produto e fornecida pelo cliente em outro.
ALTER TABLE item_structures
    ADD COLUMN IF NOT EXISTS customer_supplied BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN item_structures.customer_supplied IS
    'Componente fornecido pelo cliente (beneficiamento): fora do MRP, da baixa automática e do custo; a OF do pai não dá entrada no estoque próprio.';

-- A OF consulta "o pai tem componente do cliente?" na conclusão e no apontamento.
CREATE INDEX IF NOT EXISTS idx_item_structures_customer_supplied
    ON item_structures (parent_code)
    WHERE customer_supplied AND is_active;
