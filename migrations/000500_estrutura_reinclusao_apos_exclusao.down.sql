-- Volta às restrições que contam também as linhas inativas. Para isso precisa
-- sobrar uma linha por par: as inativas que convivem com outra linha do mesmo
-- par são descartadas (a ativa, ou a inativa mais recente, fica).
DELETE FROM item_structures s
 USING item_structures o
 WHERE s.parent_code = o.parent_code
   AND s.child_code  = o.child_code
   AND s.parent_mask IS NOT DISTINCT FROM o.parent_mask
   AND s.is_active = FALSE
   AND (o.is_active = TRUE OR o.id > s.id);

DROP INDEX IF EXISTS uq_item_structure_com_mascara;
DROP INDEX IF EXISTS uq_item_structure_sem_mascara;

CREATE UNIQUE INDEX uq_item_structure_sem_mascara
    ON item_structures (parent_code, child_code)
 WHERE parent_mask IS NULL;

ALTER TABLE item_structures
    ADD CONSTRAINT uq_item_structure_unique
        UNIQUE (parent_code, child_code, parent_mask);
