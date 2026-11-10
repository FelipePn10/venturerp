-- O "atendido" da lista de materiais da OP (production_order_materials.
-- attended_quantity) nunca era atualizado: o consumo lançado na OP baixava o
-- estoque, mas a lista continuava dizendo que nada tinha saído. O MRP passa a
-- usar o que FALTA sair (necessário − atendido) como empenho da OP; sem este
-- gatilho o empenho nunca baixaria.
--
-- Regra: o consumido do item na OP é distribuído pelas linhas do mesmo item
-- (ou do substituto) na ordem das etapas — início da ordem primeiro —, cada
-- linha até a sua quantidade; o que passar da necessidade fica na última.
-- O gatilho cobre todo caminho que grava consumo (tela, baixa automática,
-- leitor, importação) e o estorno (DELETE/UPDATE). Só linhas de demanda
-- (DEMAND): a linha de retorno (RETURN) não é material a sair.

CREATE OR REPLACE FUNCTION erp_recalcular_atendido_da_op(p_ordem bigint, p_item bigint)
RETURNS void LANGUAGE sql AS $$
    WITH total AS (
        SELECT COALESCE(SUM(c.consumed_qty), 0) AS consumido
          FROM production_consumptions c
         WHERE c.production_order_id = p_ordem AND c.item_code = p_item
    ), linhas AS (
        SELECT m.id, m.quantity,
               SUM(m.quantity) OVER (ORDER BY m.route_operation_sequence NULLS FIRST, m.id) - m.quantity AS antes
          FROM production_order_materials m
         WHERE m.production_order_id = p_ordem
           AND m.material_kind = 'DEMAND'
           AND COALESCE(m.substituted_item_code, m.item_code) = p_item
    ), ultima AS (
        SELECT id FROM production_order_materials m
         WHERE m.production_order_id = p_ordem
           AND m.material_kind = 'DEMAND'
           AND COALESCE(m.substituted_item_code, m.item_code) = p_item
         ORDER BY m.route_operation_sequence DESC NULLS LAST, m.id DESC
         LIMIT 1
    )
    UPDATE production_order_materials m
       SET attended_quantity = CASE
               WHEN m.id = (SELECT id FROM ultima) THEN GREATEST(0, t.consumido - l.antes)
               ELSE GREATEST(0, LEAST(l.quantity, t.consumido - l.antes))
           END,
           updated_at = NOW()
      FROM linhas l, total t
     WHERE m.id = l.id
       AND m.attended_quantity IS DISTINCT FROM CASE
               WHEN m.id = (SELECT id FROM ultima) THEN GREATEST(0, t.consumido - l.antes)
               ELSE GREATEST(0, LEAST(l.quantity, t.consumido - l.antes))
           END;
$$;

CREATE OR REPLACE FUNCTION erp_atendido_da_op_pelo_consumo()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP IN ('UPDATE', 'DELETE') THEN
        PERFORM erp_recalcular_atendido_da_op(OLD.production_order_id, OLD.item_code);
    END IF;
    IF TG_OP IN ('INSERT', 'UPDATE') THEN
        PERFORM erp_recalcular_atendido_da_op(NEW.production_order_id, NEW.item_code);
    END IF;
    RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS atendido_da_op_pelo_consumo ON production_consumptions;
CREATE TRIGGER atendido_da_op_pelo_consumo
    AFTER INSERT OR UPDATE OR DELETE ON production_consumptions
    FOR EACH ROW EXECUTE FUNCTION erp_atendido_da_op_pelo_consumo();

-- Histórico: o consumo que já existe passa a aparecer como atendido.
SELECT erp_recalcular_atendido_da_op(c.production_order_id, c.item_code)
  FROM (SELECT DISTINCT production_order_id, item_code FROM production_consumptions) c;
