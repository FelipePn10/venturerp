-- Orçamento: separar valor do produto, valor do IPI e produto + IPI.
--
-- A proposta mostrava "total líquido" e "líquido c/ IPI" e nada mais. Faltava o
-- número que o cliente pede primeiro — quanto é de IPI — e o comprador tinha de
-- subtrair as duas colunas de cabeça. Pior: `total_net_with_ipi` somava IPI **e
-- ST**, apesar do nome. Num item com substituição tributária, o valor
-- apresentado como "com IPI" vinha inflado pelo ST, e esse número seguia para o
-- pedido na conversão do orçamento.
--
-- Depois desta migração cada valor tem uma coluna e um significado só:
--   total_gross        bruto, antes do desconto
--   total_net          o PRODUTO, depois do desconto
--   total_ipi          o IPI, sozinho
--   total_st           o ST, sozinho
--   total_net_with_ipi produto + IPI (sem ST)
--
-- É como o pedido de venda já guardava (`sales_order_items`); era o orçamento
-- que destoava, e a conversão orçamento → pedido levava a distorção junto.

ALTER TABLE sales_quotation_items
    ADD COLUMN IF NOT EXISTS total_ipi NUMERIC(18,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_st  NUMERIC(18,4) NOT NULL DEFAULT 0;

ALTER TABLE sales_quotations
    ADD COLUMN IF NOT EXISTS total_ipi      NUMERIC(18,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_st       NUMERIC(18,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_with_ipi NUMERIC(18,4) NOT NULL DEFAULT 0;

-- Reconstrói os valores das linhas já gravadas a partir das alíquotas, e
-- devolve a `total_net_with_ipi` o significado do nome dela.
UPDATE sales_quotation_items SET
    total_ipi = ROUND(COALESCE(total_net,0) * COALESCE(ipi_pct,0) / 100, 4),
    total_st  = ROUND(COALESCE(total_net,0) * COALESCE(st_pct,0)  / 100, 4);

UPDATE sales_quotation_items SET
    total_net_with_ipi = COALESCE(total_net,0) + total_ipi;

UPDATE sales_quotations q SET
    total_ipi = COALESCE((SELECT SUM(total_ipi) FROM sales_quotation_items i WHERE i.sales_quotation_code=q.code AND i.is_active),0),
    total_st  = COALESCE((SELECT SUM(total_st)  FROM sales_quotation_items i WHERE i.sales_quotation_code=q.code AND i.is_active),0);

UPDATE sales_quotations SET total_with_ipi = COALESCE(total_net,0) + total_ipi;

COMMENT ON COLUMN sales_quotation_items.total_net IS 'Valor do PRODUTO na linha, já com o desconto e sem imposto.';
COMMENT ON COLUMN sales_quotation_items.total_ipi IS 'Valor do IPI da linha, sozinho.';
COMMENT ON COLUMN sales_quotation_items.total_st IS 'Valor do ST da linha, sozinho — não entra em total_net_with_ipi.';
COMMENT ON COLUMN sales_quotation_items.total_net_with_ipi IS 'Produto + IPI. Sem ST: para o total a pagar use produto + IPI + ST.';
