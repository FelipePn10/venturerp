-- A quantidade da estrutura na unidade de estoque deixa de ficar velha.
--
-- A migração 354 gravou na linha a quantidade convertida e o fator, CONGELADOS
-- de propósito ("corrigir a conversão não pode reescrever o consumo de uma
-- ordem aberta"). Só que a OF copia o material quando nasce
-- (production_order_materials) — a estrutura não reescreve ordem nenhuma — e o
-- congelamento virou erro de compra: na Tecnofer o tubo 101 passou a ser
-- estocado em metro e a linha "104 MM, fator 1" seguiu dizendo 104 (o MRP
-- comprou 4.160 m em vez de 4,16 m); os tubos 102 e 103 guardaram o fator da
-- barra (1/6000) com o estoque já em metro (6 vezes menos).
--
-- A partir de agora o sistema recalcula a linha quando a unidade de estoque ou
-- a conversão do componente muda, e o MRP confere tudo antes de cada cálculo
-- (structure_uc.RecalcularFatores). O que dá para acertar só com SQL é
-- acertado aqui; o resto o primeiro cálculo do MRP acerta.

-- 1. Unidade da linha igual à de estoque: não há conversão, o fator é 1 e a
--    quantidade convertida é a própria. Linha com outra coisa gravada está
--    errada com certeza (a unidade de estoque mudou para a da linha depois).
WITH defasadas AS (
    SELECT s.id, s.quantity_stock_uom AS antes_qtd, s.conversion_factor AS antes_fator,
           s.parent_code, s.child_code, s.quantity, s.unit_of_measurement
      FROM item_structures s
      JOIN items f ON f.code = s.child_code
     WHERE s.is_active
       AND upper(trim(COALESCE(s.unit_of_measurement, ''))) = upper(trim(COALESCE(f.warehouse_unit_of_measurement, '')))
       AND (s.conversion_factor IS DISTINCT FROM 1 OR s.quantity_stock_uom IS DISTINCT FROM s.quantity)
), corrigidas AS (
    UPDATE item_structures s SET quantity_stock_uom = s.quantity, conversion_factor = 1, updated_at = NOW()
      FROM defasadas d WHERE s.id = d.id
    RETURNING s.id
)
INSERT INTO item_structure_history (structure_id, parent_code, child_code, action, before_state, after_state)
SELECT d.id, d.parent_code, d.child_code, 'ALTERACAO',
       jsonb_build_object('quantidade_estoque', d.antes_qtd, 'fator', d.antes_fator),
       jsonb_build_object('quantidade_estoque', d.quantity, 'fator', 1, 'unidade', d.unit_of_measurement,
                          'motivo', 'migração 504: unidade da linha igual à de estoque, fator volta a 1')
  FROM defasadas d JOIN corrigidas c ON c.id = d.id;

-- 2. Para conferência: linhas em que a unidade difere da de estoque. O fator
--    certo depende das conversões do item e do catálogo (o grafo de unidades,
--    no Go), então o recálculo é do sistema; esta visão mostra onde olhar.
--    Fator 1 entre unidades diferentes quase sempre é a unidade de estoque que
--    mudou depois de a linha ser gravada.
CREATE OR REPLACE VIEW vw_estrutura_unidade_a_conferir AS
SELECT p.enterprise_id, s.id AS structure_id, s.parent_code, p.business_code AS parent_business_code,
       s.child_code, f.business_code AS child_business_code,
       s.quantity, s.unit_of_measurement AS unidade_da_linha, f.warehouse_unit_of_measurement AS unidade_de_estoque,
       s.quantity_stock_uom, s.conversion_factor,
       (s.conversion_factor = 1) AS fator_um_entre_unidades_diferentes
  FROM item_structures s
  JOIN items p ON p.code = s.parent_code
  JOIN items f ON f.code = s.child_code
 WHERE s.is_active
   AND upper(trim(COALESCE(s.unit_of_measurement, ''))) <> upper(trim(COALESCE(f.warehouse_unit_of_measurement, '')))
   AND COALESCE(s.unit_of_measurement, '') <> '' AND COALESCE(f.warehouse_unit_of_measurement, '') <> '';

COMMENT ON VIEW vw_estrutura_unidade_a_conferir IS
    'Linhas de estrutura com unidade diferente da de estoque do componente. O sistema recalcula o fator (antes de cada MRP e quando unidade/conversão mudam); use para conferir. Filtre por enterprise_id.';

COMMENT ON COLUMN item_structures.conversion_factor IS
    'Fator aplicado (1 unidade_da_estrutura = fator × unidade_de_estoque) pela unidade de estoque e conversões ATUAIS do componente: recalculado quando elas mudam e antes de cada MRP (migração 504). A OF copia o material na criação e não muda.';
COMMENT ON COLUMN item_structures.quantity_stock_uom IS
    'Quantidade na unidade de ESTOQUE atual do item filho — o número que MRP, ordem, custo e apontamento leem. Recalculada com o fator (migração 504).';
