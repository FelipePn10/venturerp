-- fiscal_exits.customer_code entrou na 000369 sem preencher as notas antigas.
-- Quem lê o faturamento por cliente (Visão 360, última nota) não via essas
-- notas. Preenche pelo pedido de venda da nota e, sem pedido, pelo CNPJ/CPF
-- do destinatário quando ele é de um único cliente da mesma empresa.
-- Devoluções a fornecedor (supplier_code) e notas de entrada ficam de fora.
UPDATE fiscal_exits e
   SET customer_code = o.customer_code
  FROM sales_orders o
  JOIN enterprise en ON en.code = o.enterprise_code
 WHERE e.customer_code IS NULL
   AND e.supplier_code IS NULL
   AND COALESCE(e.tipo_documento, 1) = 1
   AND e.sales_order_code = o.code
   AND en.id = e.enterprise_id
   AND o.customer_code IS NOT NULL;

UPDATE fiscal_exits e
   SET customer_code = c.code
  FROM customers c
 WHERE e.customer_code IS NULL
   AND e.supplier_code IS NULL
   AND COALESCE(e.tipo_documento, 1) = 1
   AND c.enterprise_id = e.enterprise_id
   AND regexp_replace(c.document_number, '\D', '', 'g') = regexp_replace(COALESCE(e.cnpj_destinatario, ''), '\D', '', 'g')
   AND regexp_replace(c.document_number, '\D', '', 'g') <> ''
   AND NOT EXISTS (
         SELECT 1 FROM customers c2
          WHERE c2.enterprise_id = c.enterprise_id AND c2.code <> c.code
            AND regexp_replace(c2.document_number, '\D', '', 'g') = regexp_replace(c.document_number, '\D', '', 'g'));
