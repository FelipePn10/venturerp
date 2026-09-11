-- `uq_item_structure_unique UNIQUE (parent_code, child_code, parent_mask)` não
-- impede nada no caso normal. No PostgreSQL dois NULL são sempre distintos num
-- índice único, e estrutura sem máscara — que é a maioria — grava
-- `parent_mask` NULL. Resultado: o mesmo componente entrava quantas vezes
-- fosse salvo, e a quantidade dele passava a contar em dobro no MRP e no custo.
--
-- Em produção isso já aconteceu: o mesmo par pai/filho aparecia duas vezes.

-- 1. Remove as duplicatas existentes, mantendo a linha mais recente (é a que
--    tem os valores que o usuário enxergou por último).
DELETE FROM item_structures s
 USING item_structures mais_nova
 WHERE s.parent_code = mais_nova.parent_code
   AND s.child_code  = mais_nova.child_code
   AND s.parent_mask IS NULL
   AND mais_nova.parent_mask IS NULL
   AND s.id < mais_nova.id;

-- 2. Índice parcial para o caso sem máscara, onde a restrição atual é inerte.
CREATE UNIQUE INDEX IF NOT EXISTS uq_item_structure_sem_mascara
    ON item_structures (parent_code, child_code)
 WHERE parent_mask IS NULL;
