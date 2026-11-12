-- Reserva do pedido de venda aponta para a LINHA do pedido.
--
-- A confirmação do pedido gravava em reference_item_code o código do PRODUTO;
-- o romaneio, a reprogramação da entrega e o cancelamento da linha leem ali a
-- linha do pedido. Na prática a reprogramação não mudava a validade da reserva
-- e o cancelamento da linha não a soltava.
--
-- Corrige as reservas ativas em que dá para saber a linha sem dúvida: o
-- produto aparece numa linha ativa só do pedido. As demais ficam como estão
-- (a baixa pela NF-e e a onda casam pelo produto e seguem funcionando).
UPDATE public.stock_reservations r
   SET reference_item_code = soi.code, updated_at = NOW()
  FROM public.sales_order_items soi
  JOIN public.sales_orders so ON so.code = soi.sales_order_code
  JOIN public.enterprise e ON e.code = so.enterprise_code
 WHERE r.reference_type = 'SALES_ORDER'
   AND r.status = 'ACTIVE'
   AND r.enterprise_id = e.id
   AND r.reference_code = so.code
   AND r.item_code = soi.item_code
   AND soi.is_active
   AND (r.reference_item_code IS NULL OR r.reference_item_code = r.item_code)
   AND r.reference_item_code IS DISTINCT FROM soi.code
   AND NOT EXISTS (SELECT 1 FROM public.sales_order_items outra
                    WHERE outra.sales_order_code = soi.sales_order_code
                      AND outra.item_code = soi.item_code
                      AND outra.is_active
                      AND outra.code <> soi.code);
