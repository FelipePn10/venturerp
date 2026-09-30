-- Auditoria e histórico do beneficiamento (material de terceiros).
--
-- Por que uma trilha própria, já existindo `audit_log`:
--   `audit_log` é a trilha do protocolo — quem chamou qual rota, com que status e
--   em quanto tempo. Ela responde "quem mexeu", mas não "o que era antes e o que
--   ficou depois". O levantamento da Usimac (documento 10) pede nominalmente
--   informação anterior, nova informação e motivo, por registro afetado; e o
--   documento 6 pede histórico de alteração de descrição e NCM do material do
--   cliente. Nada disso é derivável do log de requisição.
--
-- Por que trigger e não escrita no caso de uso:
--   o material de terceiros é patrimônio de outra empresa e a trilha é a defesa
--   em divergência de saldo. Escrita no caso de uso deixa de fora todo caminho
--   que não passa por ele — script de correção, carga, `psql` do plantão. No
--   banco, não existe caminho que escape. Mesmo motivo do
--   `record_manufacturing_structural_audit()` (migração 000325), e o formato
--   segue aquele, de propósito: before_state/after_state em JSONB, índice por
--   empresa, trilha imutável.
--
-- Custo: cada linha gravada nas três tabelas gera uma linha aqui. É o preço da
-- trilha, e foi aceito — o volume do beneficiamento é de notas de remessa, não de
-- apontamento de chão de fábrica. Movimento é insert-only, então sua trilha não
-- cresce além do próprio razão.
--
-- O ator: UPDATE não tem coluna de autor nas três tabelas (só `created_by` no
-- insert), por isso o repositório publica o usuário da sessão em
-- `venture.cm_actor` com `set_config(...,true)` — local à transação, como o
-- `venture.execution_actor` da migração 000359. Sem esse ajuste a trilha ainda é
-- gravada, com ator nulo: perder o autor é ruim, perder o fato é pior.

BEGIN;

CREATE TABLE IF NOT EXISTS customer_material_audit (
    id             BIGSERIAL PRIMARY KEY,
    -- Nenhuma coluna aponta para o que ela audita, de propósito: a trilha é
    -- imutável (trigger abaixo), então uma FK aqui impediria para sempre de
    -- apagar a empresa ou a remessa auditada — o registro ficaria preso pela
    -- própria trilha. A empresa é válida por construção: só chega aqui valor
    -- copiado de linha que já tem a FK.
    enterprise_id  BIGINT NOT NULL,
    entity_type    TEXT NOT NULL,
    entity_id      BIGINT NOT NULL,
    remittance_id  BIGINT,
    action         TEXT NOT NULL CHECK (action IN ('INSERT','UPDATE','DELETE')),
    changed_fields TEXT[],
    before_state   JSONB,
    after_state    JSONB,
    reason         TEXT,
    actor_id       UUID,
    occurred_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- A consulta real é "a trilha desta remessa, do mais recente para o mais antigo".
CREATE INDEX IF NOT EXISTS idx_customer_material_audit_remessa
    ON customer_material_audit (enterprise_id, remittance_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_customer_material_audit_tenant
    ON customer_material_audit (enterprise_id, occurred_at DESC);

CREATE OR REPLACE FUNCTION record_customer_material_audit() RETURNS TRIGGER AS $$
DECLARE
    old_row   JSONB := CASE WHEN TG_OP = 'INSERT' THEN NULL ELSE to_jsonb(OLD) END;
    new_row   JSONB := CASE WHEN TG_OP = 'DELETE' THEN NULL ELSE to_jsonb(NEW) END;
    tenant_id BIGINT;
    remessa   BIGINT;
    alterados TEXT[];
    motivo    TEXT;
    ator      UUID;
BEGIN
    tenant_id := COALESCE((new_row->>'enterprise_id')::BIGINT, (old_row->>'enterprise_id')::BIGINT);

    IF TG_TABLE_NAME = 'customer_material_remittances' THEN
        remessa := COALESCE((new_row->>'id')::BIGINT, (old_row->>'id')::BIGINT);
    ELSIF TG_TABLE_NAME = 'customer_material_items' THEN
        remessa := COALESCE((new_row->>'remittance_id')::BIGINT, (old_row->>'remittance_id')::BIGINT);
    ELSE
        -- Em DELETE em cascata a linha do item já pode ter ido; a trilha fica sem
        -- a remessa, e não sem o fato.
        SELECT i.remittance_id INTO remessa FROM customer_material_items i
        WHERE i.id = COALESCE((new_row->>'remittance_item_id')::BIGINT,
                              (old_row->>'remittance_item_id')::BIGINT);
    END IF;

    IF TG_OP = 'UPDATE' THEN
        SELECT array_agg(n.key ORDER BY n.key) INTO alterados
        FROM jsonb_each(new_row) AS n(key, value)
        WHERE n.key <> 'updated_at'
          AND (old_row -> n.key) IS DISTINCT FROM n.value;
        -- Recálculo de status toca `updated_at` a cada movimento; linha de trilha
        -- sem mudança de conteúdo é ruído que esconde a alteração de verdade.
        IF alterados IS NULL THEN
            RETURN NEW;
        END IF;
    END IF;

    -- Motivo e autor: o parâmetro de sessão manda, porque é o que a pessoa
    -- informou na operação. Na falta dele, só vale coluna que ESTA operação
    -- gravou. Ler `block_reason` numa alteração que não mexeu nele carimbaria o
    -- motivo de um bloqueio antigo em cima de uma mudança sem relação; e cair em
    -- `created_by` atribuiria a alteração a quem criou o registro — pior que não
    -- saber, porque nomeia quem não fez.
    motivo := NULLIF(current_setting('venture.cm_reason', true), '');
    IF motivo IS NULL THEN
        IF TG_OP = 'INSERT' THEN
            motivo := COALESCE(new_row->>'reason', new_row->>'block_reason',
                               new_row->>'divergence_reason');
        ELSIF TG_OP = 'UPDATE' THEN
            motivo := COALESCE(
                CASE WHEN (old_row->>'block_reason') IS DISTINCT FROM (new_row->>'block_reason')
                     THEN new_row->>'block_reason' END,
                CASE WHEN (old_row->>'close_reason') IS DISTINCT FROM (new_row->>'close_reason')
                     THEN new_row->>'close_reason' END,
                CASE WHEN (old_row->>'divergence_reason') IS DISTINCT FROM (new_row->>'divergence_reason')
                     THEN new_row->>'divergence_reason' END);
        END IF;
    END IF;

    ator := NULLIF(current_setting('venture.cm_actor', true), '')::UUID;
    IF ator IS NULL THEN
        IF TG_OP = 'INSERT' THEN
            ator := NULLIF(new_row->>'created_by', '')::UUID;
        ELSIF TG_OP = 'UPDATE' THEN
            ator := COALESCE(
                CASE WHEN (old_row->>'closed_by') IS DISTINCT FROM (new_row->>'closed_by')
                     THEN NULLIF(new_row->>'closed_by', '') END,
                CASE WHEN (old_row->>'divergence_settled_by') IS DISTINCT FROM (new_row->>'divergence_settled_by')
                     THEN NULLIF(new_row->>'divergence_settled_by', '') END)::UUID;
        END IF;
    END IF;

    INSERT INTO customer_material_audit(
        enterprise_id, entity_type, entity_id, remittance_id, action,
        changed_fields, before_state, after_state, reason, actor_id)
    VALUES (
        tenant_id, TG_TABLE_NAME,
        COALESCE((new_row->>'id')::BIGINT, (old_row->>'id')::BIGINT),
        remessa, TG_OP, alterados, old_row, new_row, motivo, ator);

    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION prevent_customer_material_audit_mutation() RETURNS TRIGGER AS $$
BEGIN RAISE EXCEPTION 'auditoria de beneficiamento é imutável'; END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_customer_material_audit_immutable ON customer_material_audit;
CREATE TRIGGER trg_customer_material_audit_immutable
BEFORE UPDATE OR DELETE ON customer_material_audit
FOR EACH ROW EXECUTE FUNCTION prevent_customer_material_audit_mutation();

DO $$
DECLARE tabela TEXT;
BEGIN
    FOREACH tabela IN ARRAY ARRAY['customer_material_remittances',
                                  'customer_material_items',
                                  'customer_material_movements'] LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS trg_%s_audit ON %I', tabela, tabela);
        EXECUTE format('CREATE TRIGGER trg_%s_audit AFTER INSERT OR UPDATE OR DELETE ON %I '
                       'FOR EACH ROW EXECUTE FUNCTION record_customer_material_audit()',
                       tabela, tabela);
    END LOOP;
END $$;

COMMIT;
