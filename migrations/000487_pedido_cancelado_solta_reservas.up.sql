-- Pedidos cancelados antes da correção do cancelamento.
--
-- Até aqui cancelar o pedido de venda só trocava o status: as linhas ficavam
-- OPEN/PARTIAL (o MRP e a carteira continuavam vendo a demanda) e as reservas
-- de estoque da confirmação ficavam ACTIVE (o disponível ficava menor para
-- sempre). O cancelamento agora fecha as linhas e solta as reservas; esta
-- migração faz o mesmo com os pedidos que já estavam cancelados.
--
-- A reserva solta devolve o reservado ao saldo pela MESMA regra do
-- repositório de estoque (closeReservation/adjustReservedTx): primeiro o
-- reservado livre (o que não está preso a reserva de endereço), depois o resto,
-- endereço a endereço, sem deixar o reservado negativo.

-- 1. Linhas: o saldo não atendido vira cancelado.
UPDATE public.sales_order_items soi
   SET cancelled_qty = GREATEST(soi.requested_qty - soi.attended_qty, 0),
       status = CASE WHEN soi.attended_qty > 0 THEN 'DELIVERED' ELSE 'CANCELLED' END,
       updated_at = NOW()
  FROM public.sales_orders so
 WHERE so.code = soi.sales_order_code
   AND so.status = 'CANCELLED'
   AND soi.status IN ('OPEN', 'PARTIAL');

-- 2. Reservas ativas do pedido cancelado (da mesma empresa: o pedido é por
-- enterprise_code, a reserva por enterprise_id).
DO $$
DECLARE
    res   RECORD;
    bal   RECORD;
    resta NUMERIC;
    q     NUMERIC;
    teto  NUMERIC;
    passo INT;
BEGIN
    FOR res IN
        SELECT r.id, r.enterprise_id, r.item_code, r.mask, r.warehouse_id, r.quantity
          FROM public.stock_reservations r
          JOIN public.enterprise e ON e.id = r.enterprise_id
          JOIN public.sales_orders so ON so.code = r.reference_code AND so.enterprise_code = e.code
         WHERE r.reference_type = 'SALES_ORDER'
           AND r.status = 'ACTIVE'
           AND so.status = 'CANCELLED'
         ORDER BY r.id
    LOOP
        UPDATE public.stock_reservations SET status = 'CANCELLED', updated_at = NOW() WHERE id = res.id;
        resta := res.quantity;
        FOR passo IN 1..2 LOOP
            FOR bal IN
                SELECT b.id, b.reserved_qty,
                       COALESCE((SELECT SUM(x.quantity) FROM public.stock_reservations x
                                  WHERE x.enterprise_id = b.enterprise_id AND x.item_code = b.item_code AND x.mask = b.mask
                                    AND x.warehouse_id = b.warehouse_id AND x.address = b.address AND x.address <> ''
                                    AND x.status = 'ACTIVE'), 0) AS fixa
                  FROM public.stock_balances b
                 WHERE b.enterprise_id = res.enterprise_id AND b.item_code = res.item_code
                   AND b.mask = res.mask AND b.warehouse_id = res.warehouse_id
                 ORDER BY b.address
                 FOR UPDATE OF b
            LOOP
                EXIT WHEN resta <= 0;
                IF passo = 1 THEN
                    teto := GREATEST(bal.reserved_qty - bal.fixa, 0);
                ELSE
                    teto := bal.reserved_qty;
                END IF;
                q := LEAST(teto, resta);
                IF q > 0 THEN
                    UPDATE public.stock_balances
                       SET reserved_qty = GREATEST(reserved_qty - q, 0), updated_at = NOW()
                     WHERE id = bal.id;
                    resta := resta - q;
                END IF;
            END LOOP;
        END LOOP;
    END LOOP;
END $$;
