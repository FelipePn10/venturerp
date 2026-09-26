-- A razão de comissões precisa enxergar o RATEIO, não só a capa do pedido.
--
-- A migração 365 permitiu dois representantes por pedido, cada um com o seu
-- percentual. Mas quem gera o lançamento de comissão é este gatilho, e ele lia
-- `sales_orders.representative_code` / `commission_pct` — a capa. Resultado: o
-- parceiro aparecia no pedido e no orçamento, e **nunca era pago**. A comissão
-- dele continuaria sendo acertada por fora, que é justamente o que a mudança
-- queria acabar.
--
-- Agora o gatilho percorre `sales_order_representatives`. Sem rateio (pedido
-- antigo, ou importado), cai na capa exatamente como antes.
--
-- Chave de idempotência: o PRINCIPAL mantém a chave antiga
-- ('fiscal:<id>:authorized'), para os lançamentos que já existem continuarem
-- casando com o estorno. Os demais ganham o sufixo ':rep:<código>'.
--
-- Base de incidência: no faturamento dá para respeitar a escolha da linha —
-- TOTAL_PRODUTOS usa `valor_produtos` da nota e TOTAL_LIQUIDO usa `valor_total`.
-- No RECEBIMENTO a base é o que entrou no caixa: não existe "parte de produtos"
-- de um pagamento parcial, então as duas bases usam o valor recebido.
--
-- ⚠️ Corrige também uma confusão de convenção que vinha da 322: a nota guarda
-- `enterprise_id` (id da empresa) e o pedido guarda `enterprise_code` (código
-- público), e o gatilho comparava os dois DIRETAMENTE. Na empresa 1 id e código
-- valem 1 e ninguém percebeu; na segunda empresa o pedido nunca era encontrado e
-- a comissão simplesmente não era lançada. Agora a ligação passa por
-- `enterprise`, que é onde as duas chaves se encontram.

CREATE OR REPLACE FUNCTION public.record_invoice_commission() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    order_row public.sales_orders%ROWTYPE;
    setting public.commercial_commission_settings%ROWTYPE;
    share NUMERIC(7,4);
    rep RECORD;
    base NUMERIC(18,4);
    chave VARCHAR(160);
BEGIN
    IF NEW.sales_order_code IS NULL OR NEW.status NOT IN ('AUTHORIZED','CANCELLED') OR NEW.status IS NOT DISTINCT FROM OLD.status THEN RETURN NEW; END IF;
    SELECT o.* INTO order_row
    FROM public.sales_orders o
    JOIN public.enterprise e ON e.code = o.enterprise_code
    WHERE o.code = NEW.sales_order_code AND e.id = NEW.enterprise_id;
    IF NOT FOUND THEN RETURN NEW; END IF;
    SELECT * INTO setting FROM public.commercial_commission_settings WHERE enterprise_id=NEW.enterprise_id;
    IF NOT FOUND THEN setting.competence_event:='FATURAMENTO'; setting.invoice_share_pct:=100; setting.receipt_share_pct:=0; END IF;
    IF setting.competence_event='RECEBIMENTO' THEN RETURN NEW; END IF;
    share := CASE WHEN setting.competence_event='RATEIO' THEN setting.invoice_share_pct ELSE 100 END;

    IF NEW.status='AUTHORIZED' THEN
        FOR rep IN
            -- O rateio quando existe; a capa quando não existe. O principal vem
            -- primeiro para manter a chave antiga estável.
            SELECT r.representative_code, r.commission_pct, r.commission_base::text AS commission_base,
                   (r.role = 'PRINCIPAL') AS principal
            FROM public.sales_order_representatives r
            WHERE r.sales_order_code = order_row.code AND r.enterprise_code = order_row.enterprise_code
              AND r.commission_pct > 0
            UNION ALL
            SELECT order_row.representative_code, order_row.commission_pct, 'TOTAL_LIQUIDO', TRUE
            WHERE order_row.representative_code IS NOT NULL AND order_row.commission_pct > 0
              AND NOT EXISTS (SELECT 1 FROM public.sales_order_representatives r2
                              WHERE r2.sales_order_code = order_row.code AND r2.enterprise_code = order_row.enterprise_code
                                AND r2.commission_pct > 0)
            ORDER BY principal DESC
        LOOP
            base := CASE WHEN rep.commission_base = 'TOTAL_PRODUTOS'
                         THEN COALESCE(NEW.valor_produtos, NEW.valor_total)
                         ELSE NEW.valor_total END;
            chave := 'fiscal:'||NEW.id||':authorized' || CASE WHEN rep.principal THEN '' ELSE ':rep:'||rep.representative_code END;
            INSERT INTO public.commercial_commission_ledger(enterprise_id,representative_code,sales_order_code,fiscal_exit_id,event_type,competence_date,base_amount,commission_pct,amount,idempotency_key,actor_id)
            VALUES(NEW.enterprise_id,rep.representative_code,order_row.code,NEW.id,'COMPETENCIA_FATURAMENTO',NEW.data_emissao::date,base,rep.commission_pct,ROUND(base*rep.commission_pct/100*share/100,4),chave,NEW.created_by)
            ON CONFLICT(enterprise_id,idempotency_key) DO NOTHING;
        END LOOP;
    ELSE
        -- Estorna TODOS os lançamentos daquela nota, não só o do principal.
        INSERT INTO public.commercial_commission_ledger(enterprise_id,representative_code,sales_order_code,fiscal_exit_id,event_type,competence_date,base_amount,commission_pct,amount,status,reversal_of,idempotency_key,actor_id)
        SELECT enterprise_id,representative_code,sales_order_code,fiscal_exit_id,'ESTORNO_FATURAMENTO',CURRENT_DATE,base_amount,commission_pct,-amount,'CONCILIADO',code,
               'fiscal:'||NEW.id||':cancelled:'||code,NEW.created_by
        FROM public.commercial_commission_ledger
        WHERE enterprise_id=NEW.enterprise_id AND fiscal_exit_id=NEW.id
          AND event_type='COMPETENCIA_FATURAMENTO' AND status <> 'ESTORNADO'
        ON CONFLICT(enterprise_id,idempotency_key) DO NOTHING;
        UPDATE public.commercial_commission_ledger SET status='ESTORNADO'
        WHERE enterprise_id=NEW.enterprise_id AND fiscal_exit_id=NEW.id
          AND event_type='COMPETENCIA_FATURAMENTO' AND status <> 'ESTORNADO';
    END IF;
    RETURN NEW;
END $$;

CREATE OR REPLACE FUNCTION public.record_receipt_commission() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    fiscal_row public.fiscal_exits%ROWTYPE;
    order_row public.sales_orders%ROWTYPE;
    setting public.commercial_commission_settings%ROWTYPE;
    share NUMERIC(7,4);
    tenant_id BIGINT;
    rep RECORD;
    base NUMERIC(18,4);
    chave VARCHAR(160);
BEGIN
    IF NEW.fiscal_exit_id IS NULL OR NEW.status NOT IN ('RECEBIDO','CANCELADO') OR NEW.status IS NOT DISTINCT FROM OLD.status THEN RETURN NEW; END IF;
    SELECT * INTO fiscal_row FROM public.fiscal_exits WHERE id=NEW.fiscal_exit_id;
    IF NOT FOUND OR fiscal_row.sales_order_code IS NULL THEN RETURN NEW; END IF;
    tenant_id:=fiscal_row.enterprise_id;
    SELECT o.* INTO order_row
    FROM public.sales_orders o
    JOIN public.enterprise e ON e.code = o.enterprise_code
    WHERE o.code = fiscal_row.sales_order_code AND e.id = tenant_id;
    IF NOT FOUND THEN RETURN NEW; END IF;
    SELECT * INTO setting FROM public.commercial_commission_settings WHERE enterprise_id=tenant_id;
    IF NOT FOUND THEN RETURN NEW; END IF;
    IF setting.competence_event='FATURAMENTO' THEN RETURN NEW; END IF;
    share:=CASE WHEN setting.competence_event='RATEIO' THEN setting.receipt_share_pct ELSE 100 END;
    base:=COALESCE(NEW.valor_recebido,NEW.valor_bruto);

    IF NEW.status='RECEBIDO' THEN
        FOR rep IN
            SELECT r.representative_code, r.commission_pct, (r.role = 'PRINCIPAL') AS principal
            FROM public.sales_order_representatives r
            WHERE r.sales_order_code = order_row.code AND r.enterprise_code = order_row.enterprise_code
              AND r.commission_pct > 0
            UNION ALL
            SELECT order_row.representative_code, order_row.commission_pct, TRUE
            WHERE order_row.representative_code IS NOT NULL AND order_row.commission_pct > 0
              AND NOT EXISTS (SELECT 1 FROM public.sales_order_representatives r2
                              WHERE r2.sales_order_code = order_row.code AND r2.enterprise_code = order_row.enterprise_code
                                AND r2.commission_pct > 0)
            ORDER BY principal DESC
        LOOP
            chave := 'receivable:'||NEW.id||':received' || CASE WHEN rep.principal THEN '' ELSE ':rep:'||rep.representative_code END;
            INSERT INTO public.commercial_commission_ledger(enterprise_id,representative_code,sales_order_code,fiscal_exit_id,receivable_id,event_type,competence_date,base_amount,commission_pct,amount,idempotency_key,actor_id)
            VALUES(tenant_id,rep.representative_code,order_row.code,fiscal_row.id,NEW.id,'COMPETENCIA_RECEBIMENTO',COALESCE(NEW.data_recebimento,CURRENT_DATE),base,rep.commission_pct,ROUND(base*rep.commission_pct/100*share/100,4),chave,COALESCE(NEW.baixado_por,NEW.criado_por))
            ON CONFLICT(enterprise_id,idempotency_key) DO NOTHING;
        END LOOP;
    ELSE
        INSERT INTO public.commercial_commission_ledger(enterprise_id,representative_code,sales_order_code,fiscal_exit_id,receivable_id,event_type,competence_date,base_amount,commission_pct,amount,status,reversal_of,idempotency_key,actor_id)
        SELECT enterprise_id,representative_code,sales_order_code,fiscal_exit_id,receivable_id,'ESTORNO_RECEBIMENTO',CURRENT_DATE,base_amount,commission_pct,-amount,'CONCILIADO',code,
               'receivable:'||NEW.id||':cancelled:'||code,COALESCE(NEW.baixado_por,NEW.criado_por)
        FROM public.commercial_commission_ledger
        WHERE enterprise_id=tenant_id AND receivable_id=NEW.id
          AND event_type='COMPETENCIA_RECEBIMENTO' AND status <> 'ESTORNADO'
        ON CONFLICT(enterprise_id,idempotency_key) DO NOTHING;
        UPDATE public.commercial_commission_ledger SET status='ESTORNADO'
        WHERE enterprise_id=tenant_id AND receivable_id=NEW.id
          AND event_type='COMPETENCIA_RECEBIMENTO' AND status <> 'ESTORNADO';
    END IF;
    RETURN NEW;
END $$;
