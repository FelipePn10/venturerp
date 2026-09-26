-- Volta as funções de comissão para a versão da migração 322 (só a capa do
-- pedido). O rateio continua gravado; apenas deixa de gerar lançamento.

CREATE OR REPLACE FUNCTION public.record_invoice_commission() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    order_row public.sales_orders%ROWTYPE;
    setting public.commercial_commission_settings%ROWTYPE;
    original_code BIGINT;
    share NUMERIC(7,4);
BEGIN
    IF NEW.sales_order_code IS NULL OR NEW.status NOT IN ('AUTHORIZED','CANCELLED') OR NEW.status IS NOT DISTINCT FROM OLD.status THEN RETURN NEW; END IF;
    SELECT * INTO order_row FROM public.sales_orders WHERE code=NEW.sales_order_code AND enterprise_code=NEW.enterprise_id;
    IF NOT FOUND OR order_row.representative_code IS NULL OR order_row.commission_pct <= 0 THEN RETURN NEW; END IF;
    SELECT * INTO setting FROM public.commercial_commission_settings WHERE enterprise_id=NEW.enterprise_id;
    IF NOT FOUND THEN setting.competence_event:='FATURAMENTO'; setting.invoice_share_pct:=100; setting.receipt_share_pct:=0; END IF;
    IF setting.competence_event='RECEBIMENTO' THEN RETURN NEW; END IF;
    share := CASE WHEN setting.competence_event='RATEIO' THEN setting.invoice_share_pct ELSE 100 END;
    IF NEW.status='AUTHORIZED' THEN
        INSERT INTO public.commercial_commission_ledger(enterprise_id,representative_code,sales_order_code,fiscal_exit_id,event_type,competence_date,base_amount,commission_pct,amount,idempotency_key,actor_id)
        VALUES(NEW.enterprise_id,order_row.representative_code,order_row.code,NEW.id,'COMPETENCIA_FATURAMENTO',NEW.data_emissao::date,NEW.valor_total,order_row.commission_pct,ROUND(NEW.valor_total*order_row.commission_pct/100*share/100,4),'fiscal:'||NEW.id||':authorized',NEW.created_by)
        ON CONFLICT(enterprise_id,idempotency_key) DO NOTHING;
    ELSE
        SELECT code INTO original_code FROM public.commercial_commission_ledger WHERE enterprise_id=NEW.enterprise_id AND idempotency_key='fiscal:'||NEW.id||':authorized';
        IF original_code IS NOT NULL THEN
            INSERT INTO public.commercial_commission_ledger(enterprise_id,representative_code,sales_order_code,fiscal_exit_id,event_type,competence_date,base_amount,commission_pct,amount,status,reversal_of,idempotency_key,actor_id)
            SELECT enterprise_id,representative_code,sales_order_code,fiscal_exit_id,'ESTORNO_FATURAMENTO',CURRENT_DATE,base_amount,commission_pct,-amount,'CONCILIADO',code,'fiscal:'||NEW.id||':cancelled',NEW.created_by FROM public.commercial_commission_ledger WHERE code=original_code
            ON CONFLICT(enterprise_id,idempotency_key) DO NOTHING;
            UPDATE public.commercial_commission_ledger SET status='ESTORNADO' WHERE code=original_code;
        END IF;
    END IF;
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_record_invoice_commission ON public.fiscal_exits;
CREATE TRIGGER trg_record_invoice_commission AFTER UPDATE OF status ON public.fiscal_exits FOR EACH ROW EXECUTE FUNCTION public.record_invoice_commission();

CREATE OR REPLACE FUNCTION public.record_receipt_commission() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    fiscal_row public.fiscal_exits%ROWTYPE;
    order_row public.sales_orders%ROWTYPE;
    setting public.commercial_commission_settings%ROWTYPE;
    original_code BIGINT;
    share NUMERIC(7,4);
    tenant_id BIGINT;
BEGIN
    IF NEW.fiscal_exit_id IS NULL OR NEW.status NOT IN ('RECEBIDO','CANCELADO') OR NEW.status IS NOT DISTINCT FROM OLD.status THEN RETURN NEW; END IF;
    SELECT * INTO fiscal_row FROM public.fiscal_exits WHERE id=NEW.fiscal_exit_id;
    IF NOT FOUND OR fiscal_row.sales_order_code IS NULL THEN RETURN NEW; END IF;
    tenant_id:=fiscal_row.enterprise_id;
    SELECT * INTO order_row FROM public.sales_orders WHERE code=fiscal_row.sales_order_code AND enterprise_code=tenant_id;
    IF NOT FOUND OR order_row.representative_code IS NULL OR order_row.commission_pct <= 0 THEN RETURN NEW; END IF;
    SELECT * INTO setting FROM public.commercial_commission_settings WHERE enterprise_id=tenant_id;
    IF NOT FOUND THEN RETURN NEW; END IF;
    IF setting.competence_event='FATURAMENTO' THEN RETURN NEW; END IF;
    share:=CASE WHEN setting.competence_event='RATEIO' THEN setting.receipt_share_pct ELSE 100 END;
    IF NEW.status='RECEBIDO' THEN
        INSERT INTO public.commercial_commission_ledger(enterprise_id,representative_code,sales_order_code,fiscal_exit_id,receivable_id,event_type,competence_date,base_amount,commission_pct,amount,idempotency_key,actor_id)
        VALUES(tenant_id,order_row.representative_code,order_row.code,fiscal_row.id,NEW.id,'COMPETENCIA_RECEBIMENTO',COALESCE(NEW.data_recebimento,CURRENT_DATE),COALESCE(NEW.valor_recebido,NEW.valor_bruto),order_row.commission_pct,ROUND(COALESCE(NEW.valor_recebido,NEW.valor_bruto)*order_row.commission_pct/100*share/100,4),'receivable:'||NEW.id||':received',COALESCE(NEW.baixado_por,NEW.criado_por))
        ON CONFLICT(enterprise_id,idempotency_key) DO NOTHING;
    ELSE
        SELECT code INTO original_code FROM public.commercial_commission_ledger WHERE enterprise_id=tenant_id AND idempotency_key='receivable:'||NEW.id||':received';
        IF original_code IS NOT NULL THEN
            INSERT INTO public.commercial_commission_ledger(enterprise_id,representative_code,sales_order_code,fiscal_exit_id,receivable_id,event_type,competence_date,base_amount,commission_pct,amount,status,reversal_of,idempotency_key,actor_id)
            SELECT enterprise_id,representative_code,sales_order_code,fiscal_exit_id,receivable_id,'ESTORNO_RECEBIMENTO',CURRENT_DATE,base_amount,commission_pct,-amount,'CONCILIADO',code,'receivable:'||NEW.id||':cancelled',COALESCE(NEW.baixado_por,NEW.criado_por) FROM public.commercial_commission_ledger WHERE code=original_code
            ON CONFLICT(enterprise_id,idempotency_key) DO NOTHING;
            UPDATE public.commercial_commission_ledger SET status='ESTORNADO' WHERE code=original_code;
        END IF;
    END IF;
    RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS trg_record_receipt_commission ON public.contas_receber;
CREATE TRIGGER trg_record_receipt_commission AFTER UPDATE OF status ON public.contas_receber FOR EACH ROW EXECUTE FUNCTION public.record_receipt_commission();
