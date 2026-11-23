-- Bug AB: as OFs firmadas pelo MRP (as filhas) nasciam sem almoxarifado de
-- destino; concluída em outro almoxarifado, a peça não estava onde a OF pai a
-- baixa. A criação passou a escolher o destino; aqui as OFs que ficaram sem ele
-- recebem o mesmo, sempre um almoxarifado da empresa da OF:
--   1. OF filha aberta: onde a OF pai baixa a peça (linha de material);
--   2. OF com produção entregue (aberta ou concluída): onde a produção entrou —
--      pelo registro da entrega ou, sem ele, pela entrada do item no estoque;
--   3. OF aberta: o almoxarifado do cadastro do item.
-- OF cancelada sem entrega fica sem destino: nada foi produzido.

UPDATE production_orders po
   SET warehouse_id = x.almox, updated_at = NOW()
  FROM (SELECT DISTINCT ON (filha.id) filha.id, mat.warehouse_id AS almox
          FROM production_orders filha
          JOIN production_chain_links l ON l.enterprise_id = filha.enterprise_id AND l.child_planned_order_id = filha.planned_order_id
          JOIN production_orders pai ON pai.enterprise_id = l.enterprise_id
               AND (pai.id = l.parent_production_order_id
                    OR (l.parent_production_order_id IS NULL AND pai.planned_order_id = l.parent_planned_order_id))
          JOIN production_order_materials mat ON mat.production_order_id = pai.id AND mat.enterprise_id = pai.enterprise_id
               AND mat.material_kind = 'DEMAND' AND mat.item_code = filha.item_code AND mat.warehouse_id IS NOT NULL
          JOIN warehouse w ON w.id = mat.warehouse_id AND w.enterprise_id = filha.enterprise_id
         WHERE filha.warehouse_id IS NULL AND filha.status IN ('OPEN', 'RELEASED', 'IN_PROGRESS')
         ORDER BY filha.id, COALESCE(pai.start_date, 'infinity'::date), pai.id, mat.id) x
 WHERE po.id = x.id;

UPDATE production_orders po
   SET warehouse_id = d.warehouse_id, updated_at = NOW()
  FROM (SELECT DISTINCT ON (production_order_id) production_order_id, warehouse_id, enterprise_id
          FROM production_deliveries
         WHERE warehouse_id IS NOT NULL
         ORDER BY production_order_id, delivered_at DESC, id DESC) d
 WHERE po.id = d.production_order_id AND po.enterprise_id = d.enterprise_id AND po.warehouse_id IS NULL
   AND EXISTS (SELECT 1 FROM warehouse w WHERE w.id = d.warehouse_id AND w.enterprise_id = po.enterprise_id);

UPDATE production_orders po
   SET warehouse_id = m.warehouse_id, updated_at = NOW()
  FROM (SELECT DISTINCT ON (reference_code, enterprise_id) reference_code, enterprise_id, item_code, warehouse_id
          FROM stock_movements
         WHERE reference_type = 'PRODUCTION_ORDER' AND movement_type IN ('EP', 'EPP', 'EPE', 'IN')
         ORDER BY reference_code, enterprise_id, created_at DESC, id DESC) m
 WHERE po.warehouse_id IS NULL AND m.reference_code = po.id AND m.enterprise_id = po.enterprise_id AND m.item_code = po.item_code
   AND EXISTS (SELECT 1 FROM warehouse w WHERE w.id = m.warehouse_id AND w.enterprise_id = po.enterprise_id);

UPDATE production_orders po
   SET warehouse_id = i.warehouse_code, updated_at = NOW()
  FROM items i
 WHERE po.warehouse_id IS NULL
   AND po.status IN ('OPEN', 'RELEASED', 'IN_PROGRESS')
   AND i.code = po.item_code AND i.enterprise_id = po.enterprise_id
   AND i.warehouse_code IS NOT NULL
   AND EXISTS (SELECT 1 FROM warehouse w WHERE w.id = i.warehouse_code AND w.enterprise_id = po.enterprise_id);
