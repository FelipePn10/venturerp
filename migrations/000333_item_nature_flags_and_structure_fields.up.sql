-- Três frentes: natureza do item combinável, campos que faltavam na estrutura
-- de produto e histórico de alterações da estrutura.

-- ─── 1. Natureza do item deixa de ser exclusiva ──────────────────────────────
--
-- `nature` guardava um único valor (0 genérico, 1 configurado, 2 base), então
-- um item não podia ser base *e* configurado ao mesmo tempo — combinação comum
-- em quem usa configurador (um modelo reutilizável que também tem variações).
-- FoccoERP e SAP tratam isso como marcadores independentes; passamos a fazer o
-- mesmo. `nature` continua preenchido para compatibilidade com o que já lê a
-- coluna, refletindo o marcador de maior especificidade.
ALTER TABLE items
    ADD COLUMN IF NOT EXISTS is_base         boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_configured   boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_prototype    boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_tool         boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_process_item boolean NOT NULL DEFAULT false;

COMMENT ON COLUMN items.is_base IS 'Serve de modelo para criar outros itens.';
COMMENT ON COLUMN items.is_configured IS 'Tem variações resolvidas pelo configurador.';
COMMENT ON COLUMN items.is_prototype IS 'Em desenvolvimento; controle manual.';
COMMENT ON COLUMN items.is_tool IS 'Ferramenta, entra no controle de vida útil.';
COMMENT ON COLUMN items.is_process_item IS 'Item de processo em terceiros.';

UPDATE items SET is_configured = true WHERE nature = 1 AND NOT is_configured;
UPDATE items SET is_base       = true WHERE nature = 2 AND NOT is_base;

-- O gatilho de alerta olhava `nature=1`; passa a olhar o marcador, senão um
-- item base+configurado deixaria de ser avaliado.
CREATE OR REPLACE FUNCTION notification_sync_configured_item(p_item_code bigint)
RETURNS void LANGUAGE plpgsql AS $$
DECLARE item_row RECORD; has_active_question BOOLEAN; next_cycle INTEGER;
BEGIN
    SELECT id,enterprise_id,business_code,pdm_description_technique,engineering_item_base_code,nature,is_configured,created_by
      INTO item_row FROM items WHERE code=p_item_code;
    IF NOT FOUND OR NOT COALESCE(item_row.is_configured, item_row.nature = 1) THEN RETURN; END IF;
    SELECT EXISTS(SELECT 1 FROM cfg_item_characteristics ic JOIN cfg_characteristics c ON c.id=ic.characteristic_id WHERE ic.item_code=p_item_code AND c.is_active) INTO has_active_question;
    IF has_active_question THEN
        UPDATE notification_alerts SET state='RESOLVIDO',resolved_at=NOW(),resolution_reason='Primeira característica ativa associada' WHERE enterprise_id=item_row.enterprise_id AND event_key='CADASTRO_ITEM_CONFIGURADO_SEM_PERGUNTAS' AND aggregate_internal_id=item_row.id::text AND state='ABERTO';
        UPDATE notification_outbox SET state='CANCELADO',processed_at=NOW(),lease_owner=NULL,lease_until=NULL WHERE enterprise_id=item_row.enterprise_id AND event_key='CADASTRO_ITEM_CONFIGURADO_SEM_PERGUNTAS' AND aggregate_internal_id=item_row.id::text AND state IN ('PENDENTE','FALHOU');
        RETURN;
    END IF;
    IF EXISTS(SELECT 1 FROM notification_alerts WHERE enterprise_id=item_row.enterprise_id AND event_key='CADASTRO_ITEM_CONFIGURADO_SEM_PERGUNTAS' AND aggregate_internal_id=item_row.id::text AND state='ABERTO') THEN RETURN; END IF;
    SELECT COALESCE(MAX(cycle),0)+1 INTO next_cycle FROM notification_alerts WHERE enterprise_id=item_row.enterprise_id AND event_key='CADASTRO_ITEM_CONFIGURADO_SEM_PERGUNTAS' AND aggregate_internal_id=item_row.id::text;
    INSERT INTO notification_outbox(enterprise_id,event_key,event_version,aggregate_type,aggregate_internal_id,aggregate_public_id,payload,deduplication_key,originator_user_id)
    VALUES(item_row.enterprise_id,'CADASTRO_ITEM_CONFIGURADO_SEM_PERGUNTAS',1,'ITEM',item_row.id::text,item_row.business_code,jsonb_build_object('codigo',item_row.business_code,'descricao',item_row.pdm_description_technique,'item_base',item_row.engineering_item_base_code,'criador_usuario_id',item_row.created_by,'data',NOW(),'link','/items/'||item_row.business_code),'item:'||item_row.id::text||':ciclo:'||next_cycle::text,item_row.created_by)
    ON CONFLICT(enterprise_id,event_key,deduplication_key) DO NOTHING;
END;
$$;

DROP TRIGGER IF EXISTS trg_notification_item_configured ON items;
CREATE TRIGGER trg_notification_item_configured
    AFTER INSERT OR UPDATE OF nature, is_configured, enterprise_id ON items
    FOR EACH ROW EXECUTE FUNCTION notification_item_configured_trigger();

-- ─── 2. Campos da estrutura que só existiam no FoccoERP/SAP ──────────────────
ALTER TABLE item_structures
    -- Almoxarifado de onde o componente é baixado; sem ele o sistema cai no
    -- almoxarifado do cadastro do item.
    ADD COLUMN IF NOT EXISTS warehouse_code        bigint,
    -- Almoxarifado de linha (junto ao operador), usado na requisição.
    ADD COLUMN IF NOT EXISTS line_warehouse_code   bigint,
    -- Perda de preparação da máquina: quantidade consumida uma vez por ordem.
    ADD COLUMN IF NOT EXISTS setup_loss            double precision NOT NULL DEFAULT 0,
    -- A perda de custo é separada da perda de engenharia: a de engenharia
    -- dimensiona a necessidade, a de custo entra no cálculo do custo.
    ADD COLUMN IF NOT EXISTS cost_loss_type        varchar(10)   NOT NULL DEFAULT 'PERCENTUAL',
    ADD COLUMN IF NOT EXISTS cost_loss             double precision NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cost_center_code      bigint,
    -- Item crítico é o filtro de análise do plano mestre (MPS).
    ADD COLUMN IF NOT EXISTS is_critical_mps       boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS generates_inspection  boolean NOT NULL DEFAULT false;

ALTER TABLE item_structures
    DROP CONSTRAINT IF EXISTS ck_item_structures_cost_loss_type;
ALTER TABLE item_structures
    ADD CONSTRAINT ck_item_structures_cost_loss_type
    CHECK (cost_loss_type IN ('PERCENTUAL', 'QUANTIDADE'));

ALTER TABLE item_structures
    DROP CONSTRAINT IF EXISTS ck_item_structures_setup_loss;
ALTER TABLE item_structures
    ADD CONSTRAINT ck_item_structures_setup_loss CHECK (setup_loss >= 0);

COMMENT ON COLUMN item_structures.setup_loss IS 'Perda de preparação: consumida uma vez por ordem.';
COMMENT ON COLUMN item_structures.cost_loss IS 'Perda usada no custo, independente da perda de engenharia.';
COMMENT ON COLUMN item_structures.is_critical_mps IS 'Entra na linha crítica analisada pelo plano mestre.';

-- ─── 3. Histórico da estrutura ───────────────────────────────────────────────
--
-- Nem FoccoERP nem SAP guardam quem mudou o quê na estrutura com esse
-- detalhe. Uma estrutura errada gera ordem errada; sem histórico não há como
-- responder "quem trocou a quantidade e quando".
CREATE TABLE IF NOT EXISTS item_structure_history (
    id            bigserial   PRIMARY KEY,
    structure_id  bigint      NOT NULL,
    parent_code   bigint      NOT NULL,
    child_code    bigint      NOT NULL,
    -- INCLUSAO, ALTERACAO ou EXCLUSAO.
    action        varchar(12) NOT NULL,
    changed_by    uuid        REFERENCES users(id),
    changed_at    timestamptz NOT NULL DEFAULT now(),
    -- Estado antes e depois, para o usuário ver exatamente o que mudou.
    before_state  jsonb,
    after_state   jsonb,
    CONSTRAINT ck_item_structure_history_action CHECK (action IN ('INCLUSAO', 'ALTERACAO', 'EXCLUSAO'))
);

CREATE INDEX IF NOT EXISTS idx_item_structure_history_parent
    ON item_structure_history (parent_code, changed_at DESC);
CREATE INDEX IF NOT EXISTS idx_item_structure_history_structure
    ON item_structure_history (structure_id, changed_at DESC);

COMMENT ON TABLE item_structure_history IS
    'Quem alterou o quê na estrutura de produto, com o antes e o depois.';
