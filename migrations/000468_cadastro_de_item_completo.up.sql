-- Cadastro de item completo (VENT0200).
--
-- A comparação com SAP, Oracle, Dynamics e TOTVS mostrou o que faltava ao mestre
-- de item, e parte do que faltava já quebrava obrigação fiscal:
--  * a NF-e saía sem GTIN (cEAN/cEANTrib) e com a unidade tributável sempre igual
--    à comercial;
--  * o 0200 da EFD deduzia o TIPO_ITEM e nunca chegava a 03, 05, 06 ou 10, embora
--    o 0210 só exista para o tipo 03; o COD_BARRA só vinha do XML de entrada;
--  * validade, tolerância de recebimento, prazo, comprador e produto perigoso não
--    tinham onde morar, e a tela perdia 13 campos no salvar.
--
-- As colunas ficam em `items` de propósito: o histórico campo a campo
-- (item_registrar_historico) e a completude da governança (Ficha) leem a linha
-- do item, então tudo aqui nasce auditado e verificável.
ALTER TABLE items
    ADD COLUMN long_description                      TEXT,
    ADD COLUMN general_notes                         VARCHAR(1000),
    ADD COLUMN revision                              VARCHAR(20),
    ADD COLUMN previous_code                         VARCHAR(60),
    ADD COLUMN manufacturer_name                     VARCHAR(120),
    ADD COLUMN manufacturer_part_number              VARCHAR(60),
    ADD COLUMN warehouse_shelf_life_days             INTEGER,
    ADD COLUMN warehouse_min_remaining_life_days     INTEGER,
    ADD COLUMN warehouse_storage_conditions          VARCHAR(120),
    ADD COLUMN warehouse_hazardous                   BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN warehouse_un_number                   CHARACTER(4),
    ADD COLUMN warehouse_hazard_class                VARCHAR(10),
    ADD COLUMN engineering_volume                    NUMERIC(18,6),
    ADD COLUMN planning_buyer_employee_code          BIGINT,
    ADD COLUMN planning_purchase_lead_time_days      INTEGER,
    ADD COLUMN planning_manufacturing_lead_time_days INTEGER,
    ADD COLUMN planning_notes                        VARCHAR(1000),
    ADD COLUMN supplies_under_delivery_tolerance_pct NUMERIC(9,4),
    ADD COLUMN supplies_over_delivery_tolerance_pct  NUMERIC(9,4),
    ADD COLUMN supplies_receipt_routing              VARCHAR(10) NOT NULL DEFAULT 'PADRAO',
    ADD COLUMN supplies_classification_code          VARCHAR(40),
    ADD COLUMN accounting_gtin                       VARCHAR(14),
    ADD COLUMN accounting_gtin_tax                   VARCHAR(14),
    ADD COLUMN accounting_efd_item_type              CHARACTER(2),
    ADD COLUMN accounting_tax_unit                   VARCHAR(6),
    ADD COLUMN accounting_tax_unit_factor            NUMERIC(18,6),
    ADD COLUMN pdm_signature                         VARCHAR(64);

ALTER TABLE items
    ADD CONSTRAINT chk_items_shelf_life CHECK (
        (warehouse_shelf_life_days IS NULL OR warehouse_shelf_life_days > 0)
        AND (warehouse_min_remaining_life_days IS NULL OR warehouse_min_remaining_life_days >= 0)),
    ADD CONSTRAINT chk_items_un_number CHECK (warehouse_un_number IS NULL OR warehouse_un_number ~ '^[0-9]{4}$'),
    ADD CONSTRAINT chk_items_lead_times CHECK (
        (planning_purchase_lead_time_days IS NULL OR planning_purchase_lead_time_days >= 0)
        AND (planning_manufacturing_lead_time_days IS NULL OR planning_manufacturing_lead_time_days >= 0)),
    ADD CONSTRAINT chk_items_delivery_tolerance CHECK (
        (supplies_under_delivery_tolerance_pct IS NULL OR supplies_under_delivery_tolerance_pct BETWEEN 0 AND 100)
        AND (supplies_over_delivery_tolerance_pct IS NULL OR supplies_over_delivery_tolerance_pct >= 0)),
    ADD CONSTRAINT chk_items_receipt_routing CHECK (supplies_receipt_routing IN ('DIRETO', 'PADRAO', 'INSPECAO')),
    ADD CONSTRAINT chk_items_gtin CHECK (
        (accounting_gtin IS NULL OR accounting_gtin ~ '^([0-9]{8}|[0-9]{12,14})$')
        AND (accounting_gtin_tax IS NULL OR accounting_gtin_tax ~ '^([0-9]{8}|[0-9]{12,14})$')),
    -- Tabela do campo TIPO_ITEM do Registro 0200 (Guia Prático EFD-ICMS/IPI).
    ADD CONSTRAINT chk_items_efd_item_type CHECK (accounting_efd_item_type IS NULL OR accounting_efd_item_type IN
        ('00','01','02','03','04','05','06','07','08','09','10','99')),
    ADD CONSTRAINT chk_items_tax_unit_factor CHECK (accounting_tax_unit_factor IS NULL OR accounting_tax_unit_factor > 0);

-- O GTIN identifica um produto: dentro da empresa não pode haver dois itens com
-- o mesmo. Entre empresas pode, porque cada uma é dona do seu cadastro.
CREATE UNIQUE INDEX uq_items_enterprise_gtin ON items (enterprise_id, accounting_gtin) WHERE accounting_gtin IS NOT NULL;
CREATE INDEX idx_items_enterprise_previous_code ON items (enterprise_id, previous_code) WHERE previous_code IS NOT NULL;
CREATE INDEX idx_items_enterprise_mpn ON items (enterprise_id, manufacturer_part_number) WHERE manufacturer_part_number IS NOT NULL;
CREATE INDEX idx_items_enterprise_pdm_signature ON items (enterprise_id, pdm_signature) WHERE pdm_signature IS NOT NULL;
CREATE INDEX idx_items_buyer ON items (enterprise_id, planning_buyer_employee_code) WHERE planning_buyer_employee_code IS NOT NULL;

-- PDM: o padrão descritivo de verdade. Até aqui grupo e modificador eram duas
-- listas soltas e o atributo era texto livre que nenhuma tela gravava; a descrição
-- técnica dependia de cada pessoa. Agora o modificador pertence a grupos e cada
-- par grupo+modificador tem o seu dicionário de atributos, que gera a descrição.
ALTER TABLE modifier ADD COLUMN abbreviation VARCHAR(20);

CREATE TABLE pdm_grupo_modificadores (
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    group_code    INTEGER NOT NULL,
    modifier_id   BIGINT NOT NULL REFERENCES modifier(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (enterprise_id, group_code, modifier_id)
);

CREATE TABLE pdm_atributos (
    id              BIGSERIAL PRIMARY KEY,
    enterprise_id   BIGINT NOT NULL REFERENCES enterprise(id),
    group_code      INTEGER NOT NULL,
    modifier_id     BIGINT NOT NULL REFERENCES modifier(id) ON DELETE CASCADE,
    name            VARCHAR(60) NOT NULL,
    position        INTEGER NOT NULL DEFAULT 0,
    value_type      VARCHAR(8) NOT NULL DEFAULT 'TEXTO' CHECK (value_type IN ('TEXTO', 'NUMERO', 'LISTA')),
    unit            VARCHAR(12),
    prefix          VARCHAR(12),
    decimal_places  SMALLINT NOT NULL DEFAULT 0 CHECK (decimal_places BETWEEN 0 AND 6),
    is_required     BOOLEAN NOT NULL DEFAULT FALSE,
    in_description  BOOLEAN NOT NULL DEFAULT TRUE,
    in_signature    BOOLEAN NOT NULL DEFAULT TRUE,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_by      UUID NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, group_code, modifier_id, name)
);
CREATE INDEX idx_pdm_atributos_modelo ON pdm_atributos (enterprise_id, group_code, modifier_id, position);

CREATE TABLE pdm_atributo_valores (
    id           BIGSERIAL PRIMARY KEY,
    attribute_id BIGINT NOT NULL REFERENCES pdm_atributos(id) ON DELETE CASCADE,
    value        VARCHAR(60) NOT NULL,
    abbreviation VARCHAR(20),
    position     INTEGER NOT NULL DEFAULT 0,
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (attribute_id, value)
);

-- Os pares já usados viram o vínculo inicial, por empresa: grupo sem vínculo
-- segue aceitando qualquer modificador, então nada que funciona deixa de
-- funcionar.
INSERT INTO pdm_grupo_modificadores (enterprise_id, group_code, modifier_id)
SELECT DISTINCT i.enterprise_id, i.pdm_group_code::INTEGER, i.pdm_modifier_code
  FROM items i
  JOIN modifier m ON m.id = i.pdm_modifier_code AND m.enterprise_id = i.enterprise_id
  JOIN groups g ON g.code = i.pdm_group_code AND g.enterprise_id = i.enterprise_id
 WHERE i.pdm_group_code <> 0 AND i.pdm_modifier_code <> 0;

-- Política por processo: o que cada status do cadastro permite em cada documento.
-- Sem linha vale o que sempre valeu (status diferente de LIBERADO bloqueia).
CREATE TABLE item_status_politicas (
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    status        VARCHAR(20) NOT NULL CHECK (status IN ('EM_CADASTRO', 'BLOQUEADO', 'DESCONTINUADO')),
    processo      VARCHAR(20) NOT NULL CHECK (processo IN ('ORCAMENTO', 'VENDA', 'COMPRA', 'REQUISICAO', 'PRODUCAO', 'FATURAMENTO')),
    politica      VARCHAR(10) NOT NULL CHECK (politica IN ('LIBERADO', 'AVISO', 'BLOQUEADO')),
    updated_by    UUID,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (enterprise_id, status, processo)
);

ALTER TABLE item_historico DROP CONSTRAINT item_historico_evento_check;
ALTER TABLE item_historico ADD CONSTRAINT item_historico_evento_check CHECK (evento IN
    ('CRIACAO', 'ALTERACAO', 'STATUS', 'LIBERACAO', 'ESTORNO_LIBERACAO', 'USO_COM_AVISO'));

CREATE OR REPLACE FUNCTION item_conferir_utilizavel() RETURNS trigger AS $$
DECLARE
    g        item_governanca%ROWTYPE;
    codigo   TEXT;
    subst    TEXT;
    v_processo TEXT;
    v_regra    TEXT;
BEGIN
    IF NEW.item_code IS NULL THEN
        RETURN NEW;
    END IF;
    IF TG_OP = 'UPDATE' AND NEW.item_code IS NOT DISTINCT FROM OLD.item_code THEN
        RETURN NEW; -- documento já aberto segue (alterar quantidade, status…)
    END IF;
    SELECT * INTO g FROM item_governanca WHERE item_code = NEW.item_code;
    IF NOT FOUND OR g.status = 'LIBERADO' THEN
        RETURN NEW;
    END IF;
    v_processo := CASE TG_TABLE_NAME
        WHEN 'sales_quotation_items'      THEN 'ORCAMENTO'
        WHEN 'sales_order_items'          THEN 'VENDA'
        WHEN 'purchase_order_items'       THEN 'COMPRA'
        WHEN 'purchase_requisition_items' THEN 'REQUISICAO'
        WHEN 'production_orders'          THEN 'PRODUCAO'
        WHEN 'fiscal_exit_items'          THEN 'FATURAMENTO'
    END;
    SELECT p.politica INTO v_regra FROM item_status_politicas p
     WHERE p.enterprise_id = g.enterprise_id AND p.status = g.status AND p.processo = v_processo;
    IF v_regra = 'LIBERADO' THEN
        RETURN NEW;
    END IF;
    IF v_regra = 'AVISO' THEN
        -- Passa, mas fica no histórico: o uso consciente tem de deixar rastro.
        INSERT INTO item_historico (enterprise_id, item_code, evento, campo, depois, motivo, por)
        VALUES (g.enterprise_id, NEW.item_code, 'USO_COM_AVISO', v_processo, to_jsonb(g.status), g.motivo, item_ator());
        RETURN NEW;
    END IF;
    SELECT business_code INTO codigo FROM items WHERE code = NEW.item_code;
    SELECT business_code INTO subst FROM items WHERE code = g.substituto_code;
    RAISE EXCEPTION USING ERRCODE = 'P0001', MESSAGE =
        CASE g.status
            WHEN 'EM_CADASTRO'   THEN format('o item %s ainda está em cadastro: as áreas precisam liberar o cadastro antes de usá-lo', codigo)
            WHEN 'BLOQUEADO'     THEN format('o item %s está bloqueado (%s)', codigo, g.motivo)
            ELSE format('o item %s foi descontinuado (%s)%s', codigo, g.motivo,
                        CASE WHEN subst IS NOT NULL THEN format(': use o item %s', subst) ELSE '' END)
        END;
END;
$$ LANGUAGE plpgsql;

-- O histórico guarda o que muda no cadastro; a assinatura do PDM é derivada dos
-- atributos, que já estão lá.
CREATE OR REPLACE FUNCTION item_registrar_historico() RETURNS trigger AS $$
DECLARE
    antes  JSONB;
    depois JSONB;
    k      TEXT;
    motivo TEXT := NULLIF(current_setting('venture.item_reason', true), '');
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO item_historico(enterprise_id, item_code, evento, por, motivo)
        VALUES (NEW.enterprise_id, NEW.code, 'CRIACAO', COALESCE(item_ator(), NEW.created_by), motivo);
        RETURN NEW;
    END IF;
    -- Colunas calculadas pelo sistema (classe ABC, LLC, política de contagem,
    -- assinatura do PDM) não são alteração de cadastro.
    antes  := to_jsonb(OLD) - ARRAY['id','abc_consumption_value','abc_share_pct','abc_calculated_at','planning_llc','cyclical_count_policy_activated_at','pdm_signature'];
    depois := to_jsonb(NEW) - ARRAY['id','abc_consumption_value','abc_share_pct','abc_calculated_at','planning_llc','cyclical_count_policy_activated_at','pdm_signature'];
    FOR k IN SELECT jsonb_object_keys(depois) LOOP
        IF antes -> k IS DISTINCT FROM depois -> k THEN
            INSERT INTO item_historico(enterprise_id, item_code, evento, campo, antes, depois, por, motivo)
            VALUES (NEW.enterprise_id, NEW.code, 'ALTERACAO', k, antes -> k, depois -> k, item_ator(), motivo);
        END IF;
    END LOOP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

SELECT instalar_conferencia_de_empresa();
