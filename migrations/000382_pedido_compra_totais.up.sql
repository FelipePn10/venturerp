-- Pedido de compra: os totais da capa nunca eram somados a partir das linhas
-- (ficavam em 0), e a alteração da capa apagava a origem. A aplicação passa a
-- recalcular os totais a cada mudança de linha (entity.CalcularTotais); aqui
-- se recalcula, com a MESMA regra, o que já existe:
--   bruto    = Σ (pedida − cancelada) × preço                (linhas válidas)
--   desconto = Σ bruto da linha × desconto%
--   IPI      = Σ (bruto − desconto) da linha × IPI%
--   frete    = só FOB (o comprador paga): valor fechado, por unidade ou % da mercadoria
--   líquido  = bruto − desconto + IPI + frete
WITH linhas AS (
    SELECT i.purchase_order_code AS code,
           ROUND(GREATEST(i.requested_qty - i.cancelled_qty, 0) * i.unit_price, 2) AS bruto,
           GREATEST(i.requested_qty - i.cancelled_qty, 0) AS qtd,
           i.discount_pct, i.ipi_pct
      FROM public.purchase_order_items i
     WHERE i.is_active AND i.status <> 'CANCELLED'
), por_linha AS (
    SELECT code, qtd, bruto,
           ROUND(bruto * discount_pct / 100, 2) AS desconto,
           ROUND((bruto - ROUND(bruto * discount_pct / 100, 2)) * ipi_pct / 100, 2) AS ipi
      FROM linhas
), soma AS (
    SELECT code, SUM(qtd) AS qtd, SUM(bruto) AS bruto, SUM(desconto) AS desconto, SUM(ipi) AS ipi
      FROM por_linha GROUP BY code
)
UPDATE public.purchase_orders po
   SET total_gross    = COALESCE(s.bruto, 0),
       total_discount = COALESCE(s.desconto, 0),
       total_net      = COALESCE(s.bruto, 0) - COALESCE(s.desconto, 0) + COALESCE(s.ipi, 0)
                      + CASE
                          WHEN upper(btrim(po.freight_type)) <> 'FOB' OR COALESCE(po.freight_value, 0) <= 0 THEN 0
                          WHEN upper(COALESCE(po.freight_value_type, '')) = 'PERCENTUAL'
                            THEN ROUND((COALESCE(s.bruto, 0) - COALESCE(s.desconto, 0)) * po.freight_value / 100, 2)
                          WHEN upper(COALESCE(po.freight_value_mode, '')) = 'UNITARIO'
                            THEN ROUND(COALESCE(s.qtd, 0) * po.freight_value, 2)
                          ELSE ROUND(po.freight_value, 2)
                        END
  FROM (SELECT p.code, s.qtd, s.bruto, s.desconto, s.ipi
          FROM public.purchase_orders p LEFT JOIN soma s ON s.code = p.code) s
 WHERE s.code = po.code;

UPDATE public.purchase_orders SET origin = 'NORMAL' WHERE origin IS NULL OR btrim(origin) = '';
UPDATE public.purchase_orders SET alcada_status = 'N' WHERE alcada_status IS NULL OR btrim(alcada_status) = '';

-- Pedido cancelado leva as linhas junto (o cancelamento antigo só marcava a capa).
UPDATE public.purchase_order_items i
   SET status = 'CANCELLED', updated_at = NOW()
  FROM public.purchase_orders po
 WHERE po.code = i.purchase_order_code AND po.status = 'CANCELLED' AND i.status <> 'CANCELLED';
