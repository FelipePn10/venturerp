-- A quantidade da estrutura passa a ter uma unidade confiável.
--
-- Hoje a linha de estrutura declara qualquer unidade em
-- `item_structures.unit_of_measurement`, sem relação com a unidade em que o
-- item é estocado (`items.warehouse_unit_of_measurement`). Dá para cadastrar a
-- chapa em KG no item e escrever "2 M2" na estrutura. Ninguém reclama, e a
-- partir daí o MRP reserva 2 KG, a ordem consome 2 KG e o custo rateia sobre
-- 2 KG — quando a engenharia quis dizer 2 metros quadrados, que podem ser
-- 31 kg. O erro não aparece em lugar nenhum: aparece no estoque, meses depois,
-- como uma diferença que ninguém explica.
--
-- O cadastro de conversões por item (`item_unit_conversions`) já existe e já é
-- usado pela compra (UM de compra ↔ UM de estoque). O que faltava era a
-- estrutura usá-lo.
--
-- A partir daqui a linha guarda TRÊS coisas:
--   • `quantity` + `unit_of_measurement` — o que a engenharia escreveu, como
--     escreveu. Continua sendo o que a tela mostra.
--   • `quantity_stock_uom`  — a mesma quantidade na unidade de ESTOQUE do item.
--     É o número que MRP, ordem, custo e apontamento passam a ler.
--   • `conversion_factor`   — o fator usado no momento em que a linha foi
--     gravada.
--
-- O fator fica CONGELADO na linha de propósito. Se alguém corrigir a densidade
-- da chapa amanhã, as estruturas já aprovadas não podem mudar de significado
-- sozinhas: isso reescreveria silenciosamente o consumo de ordens em aberto.
-- A tela compara o fator congelado com o vigente e avisa quando divergem —
-- aí a decisão de atualizar é de quem desenha o produto, não do banco.

ALTER TABLE item_structures
    ADD COLUMN quantity_stock_uom NUMERIC(18,6),
    ADD COLUMN conversion_factor  NUMERIC(18,6);

-- Retrocompatibilidade: toda linha existente foi criada com a unidade da
-- estrutura igual à de estoque (era o default do cadastro), então o fator é 1
-- e a quantidade convertida é a própria. Nenhum número muda hoje.
UPDATE item_structures SET quantity_stock_uom = quantity, conversion_factor = 1
WHERE quantity_stock_uom IS NULL;

ALTER TABLE item_structures
    ADD CONSTRAINT chk_item_structures_conversion
        CHECK (conversion_factor IS NULL OR conversion_factor > 0);

COMMENT ON COLUMN item_structures.quantity_stock_uom IS
    'Quantidade na unidade de ESTOQUE do item filho. É o número que MRP, ordem, custo e apontamento leem. Nulo só em linhas anteriores à migração 354.';
COMMENT ON COLUMN item_structures.conversion_factor IS
    'Fator aplicado (1 unidade_da_estrutura = fator × unidade_de_estoque), congelado na gravação. Não acompanha mudanças posteriores no cadastro de conversões.';
COMMENT ON COLUMN item_structures.unit_of_measurement IS
    'Unidade em que a engenharia escreveu a quantidade. Pode diferir da unidade de estoque do item — nesse caso é obrigatória uma conversão cadastrada.';
