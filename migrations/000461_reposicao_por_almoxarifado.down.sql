DROP INDEX IF EXISTS ix_stock_reservations_documento;

-- Os eventos são imutáveis: só a reversão da migração pode apagá-los.
ALTER TABLE aprovacao_eventos DISABLE TRIGGER trg_aprovacao_eventos_imutaveis;
DELETE FROM aprovacao_eventos e USING aprovacao_solicitacoes s
 WHERE e.enterprise_id = s.enterprise_id AND e.solicitacao_id = s.id AND s.tipo = 'REQUISICAO_MATERIAL';
ALTER TABLE aprovacao_eventos ENABLE TRIGGER trg_aprovacao_eventos_imutaveis;
DELETE FROM aprovacao_solicitacoes WHERE tipo = 'REQUISICAO_MATERIAL';
DELETE FROM aprovacao_politicas WHERE tipo = 'REQUISICAO_MATERIAL';
ALTER TABLE aprovacao_politicas DROP CONSTRAINT IF EXISTS aprovacao_politicas_tipo_check;
ALTER TABLE aprovacao_politicas ADD CONSTRAINT aprovacao_politicas_tipo_check
    CHECK (tipo IN ('LIBERACAO_CREDITO', 'ORCAMENTO_POLITICA_COMERCIAL', 'PEDIDO_POLITICA_COMERCIAL', 'AJUSTE_INVENTARIO'));

-- A reserva da requisição volta ao saldo: sem isso o reservado ficaria preso.
UPDATE stock_balances b SET reserved_qty = GREATEST(b.reserved_qty - x.q, 0)
  FROM (SELECT enterprise_id, item_code, mask, warehouse_id, SUM(quantity) q
          FROM stock_reservations WHERE reference_type = 'REQUISICAO_MATERIAL' AND status = 'ACTIVE'
         GROUP BY 1, 2, 3, 4) x
 WHERE b.enterprise_id = x.enterprise_id AND b.item_code = x.item_code AND b.mask = x.mask
   AND b.warehouse_id = x.warehouse_id AND b.address = '';
UPDATE stock_reservations SET status = 'CANCELLED', updated_at = NOW()
 WHERE reference_type = 'REQUISICAO_MATERIAL' AND status = 'ACTIVE';

DELETE FROM material_requisitions WHERE kind = 'TRANSFERENCIA';
UPDATE material_requisitions SET status = 'CANCELADA' WHERE status IN ('PENDENTE_APROVACAO', 'REPROVADA');
ALTER TABLE material_requisitions DROP CONSTRAINT IF EXISTS material_requisitions_status_check;
ALTER TABLE material_requisitions ADD CONSTRAINT material_requisitions_status_check
    CHECK (status IN ('ABERTA', 'PARCIAL', 'ATENDIDA', 'CANCELADA'));
ALTER TABLE material_requisitions DROP CONSTRAINT IF EXISTS ck_requisicao_destino;
ALTER TABLE material_requisitions DROP COLUMN IF EXISTS destination_warehouse_id, DROP COLUMN IF EXISTS kind;

DROP TABLE IF EXISTS stock_replenishment_policies;
