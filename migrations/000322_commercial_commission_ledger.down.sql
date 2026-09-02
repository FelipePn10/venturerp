DROP TRIGGER IF EXISTS trg_record_receipt_commission ON public.contas_receber;
DROP FUNCTION IF EXISTS public.record_receipt_commission();
DROP TRIGGER IF EXISTS trg_record_invoice_commission ON public.fiscal_exits;
DROP FUNCTION IF EXISTS public.record_invoice_commission();
DROP TABLE IF EXISTS public.commercial_commission_ledger;
DROP TABLE IF EXISTS public.commercial_commission_settings;
DROP INDEX IF EXISTS public.ux_commercial_commission_ledger_tenant_code;
