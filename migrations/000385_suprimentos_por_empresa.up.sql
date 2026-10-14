-- Suprimentos por empresa.
--
-- Registros de suprimentos, ordens de inspeção de recebimento, divergências de
-- recebimento, homologações de fornecedor e snapshots de IQF não tinham coluna
-- de empresa: a sonda cruzada leu e listou os de uma empresa a partir de outra.
-- Convenção do módulo de compras: enterprise_code (enterprise.code).
--
-- Backfill pelo vínculo mais forte disponível: o pedido de compra; o registro
-- ou a rota de origem; o fornecedor (suppliers.enterprise_id); o almoxarifado;
-- e, sem nenhum vínculo, a empresa principal (menor código).

ALTER TABLE public.procurement_records          ADD COLUMN IF NOT EXISTS enterprise_code BIGINT;
ALTER TABLE public.receiving_inspection_orders  ADD COLUMN IF NOT EXISTS enterprise_code BIGINT;
ALTER TABLE public.receiving_divergences        ADD COLUMN IF NOT EXISTS enterprise_code BIGINT;
ALTER TABLE public.supplier_homologations       ADD COLUMN IF NOT EXISTS enterprise_code BIGINT;
ALTER TABLE public.supplier_scorecard_snapshots ADD COLUMN IF NOT EXISTS enterprise_code BIGINT;

CREATE TEMP TABLE _empresa_principal ON COMMIT DROP AS SELECT MIN(code) AS code FROM public.enterprise;

UPDATE public.procurement_records t SET enterprise_code = COALESCE(
    (SELECT po.enterprise_code FROM public.purchase_orders po WHERE po.code = t.purchase_order_code),
    (SELECT e.code FROM public.suppliers s JOIN public.enterprise e ON e.id = s.enterprise_id WHERE s.code = t.supplier_code),
    (SELECT e.code FROM public.warehouse w JOIN public.enterprise e ON e.id = w.enterprise_id WHERE w.id = t.warehouse_id),
    (SELECT code FROM _empresa_principal))
 WHERE t.enterprise_code IS NULL;

UPDATE public.receiving_inspection_orders t SET enterprise_code = COALESCE(
    (SELECT po.enterprise_code FROM public.purchase_orders po WHERE po.code = t.purchase_order_code),
    (SELECT r.enterprise_code FROM public.procurement_records r WHERE r.id = t.procurement_record_id),
    (SELECT rt.enterprise_code FROM public.receiving_inspection_routes rt WHERE rt.id = t.route_id),
    (SELECT e.code FROM public.suppliers s JOIN public.enterprise e ON e.id = s.enterprise_id WHERE s.code = t.supplier_code),
    (SELECT e.code FROM public.warehouse w JOIN public.enterprise e ON e.id = w.enterprise_id WHERE w.id = t.warehouse_id),
    (SELECT code FROM _empresa_principal))
 WHERE t.enterprise_code IS NULL;

UPDATE public.receiving_divergences t SET enterprise_code = COALESCE(
    (SELECT n.enterprise_code FROM public.receiving_notices n WHERE n.id = t.notice_id),
    (SELECT po.enterprise_code FROM public.purchase_orders po WHERE po.code = t.purchase_order_code),
    (SELECT e.code FROM public.suppliers s JOIN public.enterprise e ON e.id = s.enterprise_id WHERE s.code = t.supplier_code),
    (SELECT code FROM _empresa_principal))
 WHERE t.enterprise_code IS NULL;

UPDATE public.supplier_homologations t SET enterprise_code = COALESCE(
    (SELECT e.code FROM public.suppliers s JOIN public.enterprise e ON e.id = s.enterprise_id WHERE s.code = t.supplier_code),
    (SELECT code FROM _empresa_principal))
 WHERE t.enterprise_code IS NULL;

UPDATE public.supplier_scorecard_snapshots t SET enterprise_code = COALESCE(
    (SELECT e.code FROM public.suppliers s JOIN public.enterprise e ON e.id = s.enterprise_id WHERE s.code = t.supplier_code),
    (SELECT code FROM _empresa_principal))
 WHERE t.enterprise_code IS NULL;

-- Banco sem empresa nenhuma não tem como ter registros de compras; nesse caso a
-- coluna fica nula e o NOT NULL abaixo é que denuncia a inconsistência.
ALTER TABLE public.procurement_records          ALTER COLUMN enterprise_code SET NOT NULL;
ALTER TABLE public.receiving_inspection_orders  ALTER COLUMN enterprise_code SET NOT NULL;
ALTER TABLE public.receiving_divergences        ALTER COLUMN enterprise_code SET NOT NULL;
ALTER TABLE public.supplier_homologations       ALTER COLUMN enterprise_code SET NOT NULL;
ALTER TABLE public.supplier_scorecard_snapshots ALTER COLUMN enterprise_code SET NOT NULL;

CREATE INDEX IF NOT EXISTS ix_procurement_records_enterprise ON public.procurement_records (enterprise_code, opened_at DESC);
CREATE INDEX IF NOT EXISTS ix_receiving_inspection_orders_enterprise ON public.receiving_inspection_orders (enterprise_code, created_at DESC);
CREATE INDEX IF NOT EXISTS ix_receiving_divergences_enterprise ON public.receiving_divergences (enterprise_code, created_at DESC);
CREATE INDEX IF NOT EXISTS ix_supplier_homologations_enterprise ON public.supplier_homologations (enterprise_code, supplier_code);

-- O IQF de um fornecedor num período é por empresa.
ALTER TABLE public.supplier_scorecard_snapshots DROP CONSTRAINT IF EXISTS supplier_scorecard_snapshots_supplier_code_period_start_per_key;
ALTER TABLE public.supplier_scorecard_snapshots
    ADD CONSTRAINT supplier_scorecard_snapshots_empresa_periodo_key UNIQUE (enterprise_code, supplier_code, period_start, period_end);
