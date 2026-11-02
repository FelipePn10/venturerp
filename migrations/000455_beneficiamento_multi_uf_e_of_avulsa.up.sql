-- Texto legal da nota de retorno do beneficiamento: cada UF tem o seu dispositivo.
-- Em SP vale a Portaria CAT 22/2007, que o sistema já conhece; nas demais a empresa
-- informa o da sua legislação, e sem ele a nota é recusada.
ALTER TABLE fiscal_configs ADD COLUMN obs_retorno_industrializacao TEXT;

-- cBenef da suspensão (CST 50) no material que volta. O do diferimento já existe em
-- cbenef_diferimento; o PR recusa os dois CST sem código (rejeição 930).
ALTER TABLE fiscal_configs ADD COLUMN cbenef_suspensao VARCHAR(10);

-- Pedido da OF aberta à mão. A OF do MRP chega ao pedido pela ordem planejada
-- (sales_order_planned_orders); a avulsa e a do subconjunto não têm esse caminho, e
-- saíam sem cliente no relatório de horas de beneficiamento.
ALTER TABLE production_orders
    ADD COLUMN sales_order_code BIGINT REFERENCES sales_orders(code);
CREATE INDEX idx_production_orders_sales_order ON production_orders (sales_order_code)
    WHERE sales_order_code IS NOT NULL;

SELECT instalar_conferencia_de_empresa();
