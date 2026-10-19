-- Precisão da unidade interna do pedido de compra.
--
-- A linha comprada em KG de um item estocado em M guarda a quantidade e o preço
-- convertidos (internal_qty / internal_price), e o recebimento lança no estoque
-- received × internal_qty / requested_qty. Com 4 casas, 100 KG × 0,15530965
-- viravam 15,5310 M em vez de 15,530965 — o estoque recebia uma quantidade que
-- não é a da conversão. Mesmas escalas do estoque (migração 407): quantidade com
-- 6 casas, preço unitário com 10.
ALTER TABLE purchase_order_items
    ALTER COLUMN internal_qty   TYPE NUMERIC(18,6),
    ALTER COLUMN internal_price TYPE NUMERIC(24,10);
