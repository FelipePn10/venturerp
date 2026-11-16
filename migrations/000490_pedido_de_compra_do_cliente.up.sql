-- Número do pedido de compra do cliente no pedido de venda. Vai para a NF-e
-- (det/prod/xPed nos itens e compra/xPed no cabeçalho): cliente industrial
-- costuma recusar nota sem ele. O cliente que exige (customers.requires_customer_po)
-- não tem pedido confirmado sem o número.
-- 15 caracteres: o limite do xPed do item na NF-e.
ALTER TABLE sales_orders ADD COLUMN customer_po_number VARCHAR(15);
