-- Governança do cadastro de item (Fase 4).
--
-- O item nascia utilizável com campos faltando, e a pendência só aparecia
-- quando a transação falhava (NF sem NCM, OF sem roteiro…). Agora:
--  * o cadastro tem status (EM_CADASTRO → LIBERADO; BLOQUEADO; DESCONTINUADO),
--    como o status do material do SAP e a fase do ciclo de vida da Oracle;
--  * cada área libera a sua visão (engenharia, fiscal, suprimentos,
--    planejamento, comercial, custos) quando os campos dela estão completos;
--  * toda alteração do cadastro fica num histórico campo a campo imutável;
--  * item bloqueado, descontinuado ou ainda em cadastro não entra em documento
--    novo (pedido de venda/compra, orçamento, requisição, OF, NF).
--
-- A governança fica em tabelas próprias: item sem linha em item_governanca é
-- LIBERADO — os itens que já existem continuam funcionando como antes. Exigir
-- a liberação para item novo é escolha da empresa (item_governanca_parametros).

CREATE TABLE item_governanca_parametros (
    enterprise_id   BIGINT      PRIMARY KEY REFERENCES enterprise(id),
    exige_liberacao BOOLEAN     NOT NULL DEFAULT FALSE,
    visoes          TEXT[]      NOT NULL DEFAULT ARRAY['ENGENHARIA','FISCAL','SUPRIMENTOS','PLANEJAMENTO','COMERCIAL','CUSTOS'],
    updated_by      UUID        REFERENCES users(id),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (visoes <@ ARRAY['ENGENHARIA','FISCAL','SUPRIMENTOS','PLANEJAMENTO','COMERCIAL','CUSTOS']::TEXT[])
);

CREATE TABLE item_governanca (
    enterprise_id    BIGINT      NOT NULL REFERENCES enterprise(id),
    item_code        BIGINT      NOT NULL REFERENCES items(code) ON DELETE CASCADE,
    status           VARCHAR(14) NOT NULL CHECK (status IN ('EM_CADASTRO', 'LIBERADO', 'BLOQUEADO', 'DESCONTINUADO')),
    substituto_code  BIGINT      REFERENCES items(code),
    motivo           TEXT,
    alterado_por     UUID        REFERENCES users(id),
    alterado_em      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (enterprise_id, item_code),
    CHECK (substituto_code IS NULL OR substituto_code <> item_code),
    CHECK (status NOT IN ('BLOQUEADO', 'DESCONTINUADO') OR btrim(COALESCE(motivo, '')) <> '')
);
CREATE INDEX idx_item_governanca_status ON item_governanca(enterprise_id, status);

-- Liberação de cada visão do item (quem e quando).
CREATE TABLE item_visoes_liberadas (
    enterprise_id BIGINT      NOT NULL REFERENCES enterprise(id),
    item_code     BIGINT      NOT NULL REFERENCES items(code) ON DELETE CASCADE,
    visao         VARCHAR(14) NOT NULL CHECK (visao IN ('ENGENHARIA','FISCAL','SUPRIMENTOS','PLANEJAMENTO','COMERCIAL','CUSTOS')),
    liberado_por  UUID        REFERENCES users(id),
    liberado_em   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (enterprise_id, item_code, visao)
);

-- Ajuste da empresa sobre o catálogo de campos obrigatórios (o padrão está no
-- domínio): ligar ou desligar a obrigatoriedade de um campo.
CREATE TABLE item_regras_campo (
    enterprise_id BIGINT      NOT NULL REFERENCES enterprise(id),
    campo         VARCHAR(60) NOT NULL,
    obrigatorio   BOOLEAN     NOT NULL,
    updated_by    UUID        REFERENCES users(id),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (enterprise_id, campo)
);

-- Histórico campo a campo do cadastro, gravado pelo banco em toda alteração
-- (venha da tela, da importação ou de outro processo). O autor e o motivo vêm
-- do parâmetro de transação venture.item_actor / venture.item_reason.
CREATE TABLE item_historico (
    id            BIGSERIAL   PRIMARY KEY,
    enterprise_id BIGINT      NOT NULL,
    item_code     BIGINT      NOT NULL,
    evento        VARCHAR(16) NOT NULL CHECK (evento IN ('CRIACAO', 'ALTERACAO', 'STATUS', 'LIBERACAO', 'ESTORNO_LIBERACAO')),
    campo         VARCHAR(80),
    antes         JSONB,
    depois        JSONB,
    motivo        TEXT,
    por           UUID,
    ocorrido_em   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_item_historico_item ON item_historico(enterprise_id, item_code, ocorrido_em DESC);

CREATE OR REPLACE FUNCTION item_historico_imutavel() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'o histórico do cadastro de item não pode ser alterado nem apagado';
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_item_historico_imutavel BEFORE UPDATE OR DELETE ON item_historico
    FOR EACH ROW WHEN (pg_trigger_depth() = 0) EXECUTE FUNCTION item_historico_imutavel();

-- Valor que não é UUID (sessão de serviço, teste) vira autor desconhecido, nunca
-- falha da alteração do item.
CREATE OR REPLACE FUNCTION item_ator() RETURNS UUID AS $$
    SELECT CASE WHEN v ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' THEN v::uuid END
      FROM (SELECT current_setting('venture.item_actor', true) AS v) s
$$ LANGUAGE sql STABLE;

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
    -- Colunas calculadas pelo sistema (classe ABC, LLC, política de contagem)
    -- não são alteração de cadastro.
    antes  := to_jsonb(OLD) - ARRAY['id','abc_consumption_value','abc_share_pct','abc_calculated_at','planning_llc','cyclical_count_policy_activated_at'];
    depois := to_jsonb(NEW) - ARRAY['id','abc_consumption_value','abc_share_pct','abc_calculated_at','planning_llc','cyclical_count_policy_activated_at'];
    FOR k IN SELECT jsonb_object_keys(depois) LOOP
        IF antes -> k IS DISTINCT FROM depois -> k THEN
            INSERT INTO item_historico(enterprise_id, item_code, evento, campo, antes, depois, por, motivo)
            VALUES (NEW.enterprise_id, NEW.code, 'ALTERACAO', k, antes -> k, depois -> k, item_ator(), motivo);
        END IF;
    END LOOP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_items_historico AFTER INSERT OR UPDATE ON items
    FOR EACH ROW EXECUTE FUNCTION item_registrar_historico();

-- Item que não pode entrar em documento novo. A mensagem vai ao usuário
-- (SQLSTATE P0001, tratado como regra de negócio na API).
CREATE OR REPLACE FUNCTION item_conferir_utilizavel() RETURNS trigger AS $$
DECLARE
    g      item_governanca%ROWTYPE;
    codigo TEXT;
    subst  TEXT;
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

CREATE TRIGGER trg_item_utilizavel BEFORE INSERT OR UPDATE OF item_code ON sales_order_items
    FOR EACH ROW EXECUTE FUNCTION item_conferir_utilizavel();
CREATE TRIGGER trg_item_utilizavel BEFORE INSERT OR UPDATE OF item_code ON sales_quotation_items
    FOR EACH ROW EXECUTE FUNCTION item_conferir_utilizavel();
CREATE TRIGGER trg_item_utilizavel BEFORE INSERT OR UPDATE OF item_code ON purchase_order_items
    FOR EACH ROW EXECUTE FUNCTION item_conferir_utilizavel();
CREATE TRIGGER trg_item_utilizavel BEFORE INSERT OR UPDATE OF item_code ON purchase_requisition_items
    FOR EACH ROW EXECUTE FUNCTION item_conferir_utilizavel();
CREATE TRIGGER trg_item_utilizavel BEFORE INSERT OR UPDATE OF item_code ON production_orders
    FOR EACH ROW EXECUTE FUNCTION item_conferir_utilizavel();
CREATE TRIGGER trg_item_utilizavel BEFORE INSERT OR UPDATE OF item_code ON fiscal_exit_items
    FOR EACH ROW EXECUTE FUNCTION item_conferir_utilizavel();

SELECT instalar_conferencia_de_empresa();

-- Produtores dos alertas de cadastro ligados (ScheduleOperationalAlerts).
UPDATE notification_event_catalog SET producer_status='ATIVO', producer_description='Produtor conectado e monitorado pelo backend.'
 WHERE event_key IN ('CADASTRO_ITEM_SEM_CLASSIFICACAO_FISCAL','CADASTRO_ITEM_COMPRADO_SEM_FORNECEDOR','CADASTRO_ITEM_FABRICADO_SEM_ESTRUTURA',
                     'CADASTRO_ITEM_FABRICADO_SEM_ROTEIRO','CADASTRO_ITEM_INATIVO_COM_DEMANDA_ABERTA');
