-- Excluir um componente da estrutura não apaga a linha: marca is_active = FALSE
-- (o histórico e a auditoria apontam para ela). Mas os dois índices únicos da
-- estrutura contavam também as linhas inativas, então o componente excluído
-- nunca mais podia voltar para o mesmo pai — a tela não o mostrava e a inclusão
-- respondia "este componente já está na estrutura deste item".
--
-- Em produção (Tecnofer, 17/11/2026): TP-0120/50-E excluído do BU-120/50 em
-- 16/11 não pôde ser incluído de novo.
--
-- A regra passa a valer só para as linhas ativas, como já é a de sequência
-- (uq_structure_sequence). Todas as leituras de estrutura (tela, MRP, custo,
-- explosão, ciclo) filtram is_active, então a linha inativa que convive com a
-- nova ativa não conta em dobro.

ALTER TABLE item_structures DROP CONSTRAINT IF EXISTS uq_item_structure_unique;
DROP INDEX IF EXISTS uq_item_structure_unique;
DROP INDEX IF EXISTS uq_item_structure_sem_mascara;

CREATE UNIQUE INDEX uq_item_structure_sem_mascara
    ON item_structures (parent_code, child_code)
 WHERE parent_mask IS NULL AND is_active = TRUE;

CREATE UNIQUE INDEX uq_item_structure_com_mascara
    ON item_structures (parent_code, child_code, parent_mask)
 WHERE parent_mask IS NOT NULL AND is_active = TRUE;
