-- Fonte de suprimento por item (VSUP0130 — Fornecedor Preferencial por Item).
--
-- O que faltava, comparado com o registro de fonte dos grandes ERPs (SAP:
-- registro info + lista de fontes + quota; Oracle: Approved Supplier List +
-- sourcing rule; Senior F403FPR):
--   * quantidade mínima e múltiplo de compra POR FORNECEDOR (o MRP arredonda a
--     sugestão de compra pelo do fornecedor escolhido);
--   * cota (%) entre fornecedores: o MRP manda cada necessidade inteira ao
--     fornecedor mais atrasado em relação à sua cota (índice de cota do SAP),
--     sem picotar o pedido;
--   * bloqueio com motivo, quem e quando — em vez de excluir — com histórico;
--   * vigência (de/até) conferida em todo documento de compra;
--   * FONTE OBRIGATÓRIA ("source list requirement" do SAP / ASL do Oracle):
--     pedido de compra e a LIBERAÇÃO/REMESSA da ordem de serviço de terceiro só
--     aceitam item com fornecedor ativo, vigente e não bloqueado — e o
--     fornecedor do documento tem de ser um desses. Sem isso o MRP não sabe de
--     quem comprar nem em quanto tempo chega.
--     A REQUISIÇÃO fica livre, como no SAP e no Oracle: é pedido interno, sem
--     fornecedor, e nasce sozinha ao firmar a OF (serviço da operação externa).
--     Travá-la deixava a firmação pela metade. A geração de pedidos a partir
--     da requisição deixa de fora, com o motivo, a linha sem fonte.
--
-- A empresa pode desligar a exigência em procurement_parameters
-- (domínio SUPPLIER, chave EXIGE_FONTE_DE_SUPRIMENTO = N).
--
-- Convenção de empresa: `enterprise_id`, igual ao pai (`item_preferred_suppliers`).

ALTER TABLE item_preferred_suppliers
    ADD COLUMN IF NOT EXISTS min_order_qty  NUMERIC(18,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS order_multiple NUMERIC(18,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS quota_percent  NUMERIC(7,4),
    ADD COLUMN IF NOT EXISTS quota_since    DATE,
    ADD COLUMN IF NOT EXISTS blocked        BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS blocked_reason TEXT,
    ADD COLUMN IF NOT EXISTS blocked_at     TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS blocked_by     UUID;

ALTER TABLE item_preferred_suppliers
    ADD CONSTRAINT item_preferred_suppliers_lote_check
        CHECK (min_order_qty >= 0 AND order_multiple >= 0),
    ADD CONSTRAINT item_preferred_suppliers_cota_check
        CHECK (quota_percent IS NULL OR (quota_percent > 0 AND quota_percent <= 100)),
    ADD CONSTRAINT item_preferred_suppliers_bloqueio_check
        CHECK (NOT blocked OR NULLIF(btrim(blocked_reason), '') IS NOT NULL),
    ADD CONSTRAINT item_preferred_suppliers_vigencia_check
        CHECK (valid_from IS NULL OR valid_until IS NULL OR valid_from <= valid_until);

COMMENT ON COLUMN item_preferred_suppliers.min_order_qty IS 'Quantidade minima de compra deste fornecedor (unidade de estoque). Zero = sem minimo.';
COMMENT ON COLUMN item_preferred_suppliers.order_multiple IS 'Multiplo de compra deste fornecedor (unidade de estoque). Zero = sem multiplo.';
COMMENT ON COLUMN item_preferred_suppliers.quota_percent IS 'Cota da compra do item para este fornecedor. Vazio = fora da divisao por cota.';
COMMENT ON COLUMN item_preferred_suppliers.quota_since IS 'Dia em que a cota atual foi definida. A conta do ja comprado por cota comeca na mudanca de cota mais recente do item.';

-- Fornecedor escolhido pelo MRP, gravado na ordem planejada ao firmar a
-- sugestão: o recálculo do plano apaga as sugestões, e a ordem firmada (ainda
-- sem pedido) perdia o fornecedor — e a cota, a conta do que já foi a cada um.
ALTER TABLE planned_orders ADD COLUMN IF NOT EXISTS supplier_code BIGINT REFERENCES suppliers(code);
COMMENT ON COLUMN planned_orders.supplier_code IS 'Fornecedor escolhido pelo MRP (VSUP0130) para a ordem de compra; vem da sugestao ao firmar.';

-- Histórico de bloqueio/desbloqueio do vínculo.
CREATE TABLE IF NOT EXISTS item_supplier_block_history (
    id               BIGSERIAL PRIMARY KEY,
    enterprise_id    BIGINT NOT NULL REFERENCES enterprise(id),
    item_supplier_id BIGINT NOT NULL REFERENCES item_preferred_suppliers(id) ON DELETE CASCADE,
    action           VARCHAR(12) NOT NULL CHECK (action IN ('BLOQUEIO','DESBLOQUEIO')),
    reason           TEXT NOT NULL CHECK (btrim(reason) <> ''),
    actor_id         UUID NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_item_supplier_block_history
    ON item_supplier_block_history (enterprise_id, item_supplier_id, created_at DESC);

-- A empresa exige fonte de suprimento? Ligado por padrão.
CREATE OR REPLACE FUNCTION erp_exige_fonte_de_suprimento(p_enterprise_id BIGINT) RETURNS BOOLEAN
LANGUAGE sql STABLE AS $$
    SELECT NOT EXISTS (
        SELECT 1 FROM procurement_parameters p
          JOIN enterprise e ON e.code = p.enterprise_code
         WHERE e.id = p_enterprise_id AND p.domain = 'SUPPLIER'
           AND p.param_key = 'EXIGE_FONTE_DE_SUPRIMENTO'
           AND upper(btrim(p.param_value)) IN ('N','NAO','NÃO','FALSE','0'))
$$;

-- Há fonte válida para o item (e, se informado, deste fornecedor) na data?
-- A empresa é a do ITEM (items.code é único no banco; items.enterprise_id é a verdade).
CREATE OR REPLACE FUNCTION erp_fonte_de_suprimento_valida(p_item BIGINT, p_mask TEXT, p_supplier BIGINT, p_on DATE) RETURNS BOOLEAN
LANGUAGE sql STABLE AS $$
    SELECT EXISTS (
        SELECT 1 FROM item_preferred_suppliers s
          JOIN items i ON i.code = s.item_code AND i.enterprise_id = s.enterprise_id
         WHERE s.item_code = p_item AND s.is_active AND NOT s.blocked
           AND (s.mask = '' OR s.mask = COALESCE(p_mask, ''))
           AND (s.valid_from IS NULL OR s.valid_from <= p_on)
           AND (s.valid_until IS NULL OR s.valid_until >= p_on)
           AND (p_supplier IS NULL OR s.supplier_code = p_supplier))
$$;

-- Confere uma linha de compra; a mensagem vai ao usuário (P0001 → 422).
CREATE OR REPLACE FUNCTION erp_exigir_fonte_de_suprimento(p_item BIGINT, p_mask TEXT, p_supplier BIGINT) RETURNS VOID
LANGUAGE plpgsql AS $$
DECLARE
    empresa BIGINT;
    codigo  TEXT;
    forn    TEXT;
BEGIN
    IF p_item IS NULL THEN
        RETURN;
    END IF;
    SELECT enterprise_id, COALESCE(NULLIF(business_code, ''), code::text) INTO empresa, codigo FROM items WHERE code = p_item;
    IF NOT FOUND OR NOT erp_exige_fonte_de_suprimento(empresa) THEN
        RETURN;
    END IF;
    IF NOT erp_fonte_de_suprimento_valida(p_item, p_mask, NULL, CURRENT_DATE) THEN
        RAISE EXCEPTION USING ERRCODE = 'P0001', MESSAGE = format(
            'o item %s não tem fornecedor ativo, vigente e desbloqueado: cadastre-o em VSUP0130 — Fornecedor Preferencial por Item antes de comprar ou enviar ao terceiro', codigo);
    END IF;
    IF p_supplier IS NOT NULL AND NOT erp_fonte_de_suprimento_valida(p_item, p_mask, p_supplier, CURRENT_DATE) THEN
        SELECT COALESCE(NULLIF(trade_name, ''), name, code::text) INTO forn FROM suppliers WHERE code = p_supplier;
        RAISE EXCEPTION USING ERRCODE = 'P0001', MESSAGE = format(
            'o fornecedor %s não é fonte do item %s (sem vínculo, bloqueado ou fora da vigência): vincule-o em VSUP0130 — Fornecedor Preferencial por Item ou escolha um fornecedor do item',
            COALESCE(forn, p_supplier::text), codigo);
    END IF;
END;
$$;

-- OS de terceiro: o que se compra do terceiro é o item de serviço da operação;
-- sem ele, o próprio item enviado.
CREATE OR REPLACE FUNCTION erp_exigir_fonte_da_os(p_servico BIGINT, p_item BIGINT, p_mask TEXT, p_supplier BIGINT) RETURNS VOID
LANGUAGE plpgsql AS $$
BEGIN
    IF p_servico IS NOT NULL THEN
        PERFORM erp_exigir_fonte_de_suprimento(p_servico, '', p_supplier);
    ELSE
        PERFORM erp_exigir_fonte_de_suprimento(p_item, p_mask, p_supplier);
    END IF;
END;
$$;

CREATE OR REPLACE FUNCTION erp_conferir_fonte_de_suprimento() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    fornecedor BIGINT;
    linha      RECORD;
    os         third_party_service_orders%ROWTYPE;
BEGIN
    IF TG_TABLE_NAME = 'purchase_order_items' THEN
        IF TG_OP = 'UPDATE' AND NEW.item_code IS NOT DISTINCT FROM OLD.item_code
           AND NEW.mask IS NOT DISTINCT FROM OLD.mask THEN
            RETURN NEW;
        END IF;
        SELECT supplier_code INTO fornecedor FROM purchase_orders WHERE code = NEW.purchase_order_code;
        PERFORM erp_exigir_fonte_de_suprimento(NEW.item_code, NEW.mask, fornecedor);
    ELSIF TG_TABLE_NAME = 'purchase_orders' THEN
        -- Trocar o fornecedor do pedido: todas as linhas têm de aceitá-lo.
        IF NEW.supplier_code IS NULL OR NEW.supplier_code IS NOT DISTINCT FROM OLD.supplier_code THEN
            RETURN NEW;
        END IF;
        FOR linha IN SELECT item_code, mask FROM purchase_order_items WHERE purchase_order_code = NEW.code AND item_code IS NOT NULL LOOP
            PERFORM erp_exigir_fonte_de_suprimento(linha.item_code, linha.mask, NEW.supplier_code);
        END LOOP;
    ELSIF TG_TABLE_NAME = 'third_party_service_orders' THEN
        -- A OS nasce com a OF (planejamento) e não é barrada aí: travar a OS
        -- travaria a fábrica, não a compra. Barra-se a LIBERAÇÃO ao terceiro.
        IF NEW.status NOT IN ('RELEASED_WITH_PO','RELEASED_WITHOUT_PO') THEN
            RETURN NEW;
        END IF;
        IF TG_OP = 'UPDATE' AND OLD.status = NEW.status
           AND NEW.supplier_code IS NOT DISTINCT FROM OLD.supplier_code
           AND NEW.service_item_code IS NOT DISTINCT FROM OLD.service_item_code
           AND NEW.item_code IS NOT DISTINCT FROM OLD.item_code THEN
            RETURN NEW;
        END IF;
        PERFORM erp_exigir_fonte_da_os(NEW.service_item_code, NEW.item_code, NEW.mask, NEW.supplier_code);
    ELSIF TG_TABLE_NAME = 'third_party_service_movements' THEN
        -- Remessa ao terceiro: o envio físico também exige a fonte.
        IF NEW.movement_type <> 'REMITTANCE' THEN
            RETURN NEW;
        END IF;
        SELECT * INTO os FROM third_party_service_orders WHERE id = NEW.service_order_id AND enterprise_id = NEW.enterprise_id;
        IF FOUND THEN
            PERFORM erp_exigir_fonte_da_os(os.service_item_code, os.item_code, os.mask, os.supplier_code);
        END IF;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_fonte_de_suprimento BEFORE INSERT OR UPDATE OF item_code, mask ON purchase_order_items
    FOR EACH ROW EXECUTE FUNCTION erp_conferir_fonte_de_suprimento();
CREATE TRIGGER trg_fonte_de_suprimento BEFORE UPDATE OF supplier_code ON purchase_orders
    FOR EACH ROW EXECUTE FUNCTION erp_conferir_fonte_de_suprimento();
CREATE TRIGGER trg_fonte_de_suprimento BEFORE INSERT OR UPDATE OF status, supplier_code, service_item_code, item_code ON third_party_service_orders
    FOR EACH ROW EXECUTE FUNCTION erp_conferir_fonte_de_suprimento();
CREATE TRIGGER trg_fonte_de_suprimento BEFORE INSERT ON third_party_service_movements
    FOR EACH ROW EXECUTE FUNCTION erp_conferir_fonte_de_suprimento();

SELECT instalar_conferencia_de_empresa();
