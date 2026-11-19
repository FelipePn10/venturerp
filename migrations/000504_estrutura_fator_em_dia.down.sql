-- O recálculo não é desfeito (os números antigos estavam errados); volta só a
-- visão e os comentários.
DROP VIEW IF EXISTS vw_estrutura_unidade_a_conferir;
COMMENT ON COLUMN item_structures.conversion_factor IS
    'Fator aplicado (1 unidade_da_estrutura = fator × unidade_de_estoque), congelado na gravação. Dez casas porque a inversa de um fator grande (1/6000) não cabe em seis.';
COMMENT ON COLUMN item_structures.quantity_stock_uom IS
    'Quantidade na unidade de ESTOQUE do item filho. É o número que MRP, ordem, custo e apontamento leem. Nulo só em linhas anteriores à migração 354.';
