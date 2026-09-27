-- Volta a somar o ST dentro de `total_net_with_ipi`, que era o comportamento
-- anterior — inclusive a distorção que ele carregava.
UPDATE sales_quotation_items SET
    total_net_with_ipi = COALESCE(total_net,0) + total_ipi + total_st;

ALTER TABLE sales_quotations
    DROP COLUMN IF EXISTS total_with_ipi,
    DROP COLUMN IF EXISTS total_st,
    DROP COLUMN IF EXISTS total_ipi;

ALTER TABLE sales_quotation_items
    DROP COLUMN IF EXISTS total_st,
    DROP COLUMN IF EXISTS total_ipi;
