-- Projetos e origem da OF (cliente e projeto) + baixa automática por padrão.
--
-- 1. Cadastro de Projetos por empresa (código único na empresa, sem diferenciar
--    maiúsculas). O pedido de venda (project_code/project_name) e a demanda
--    independente (projeto) já guardavam o CÓDIGO como texto: o cadastro nasce
--    com esses códigos, então nada do que existe deixa de valer.
-- 2. A OF guarda cliente e projeto. Herda sozinha, nesta ordem: pedido de venda
--    (direto ou pela ordem planejada), demanda independente, OF pai da cadeia.
--    A OF filha nascida antes da ligação com o pai herda quando a ligação chega;
--    quando a OF pai ganha cliente/projeto, as filhas sem eles herdam junto.
--    O que o usuário preencheu na OF nunca é sobrescrito.
-- 3. Consumo de material da OF automático por padrão: a marca "baixa
--    automática" do cadastro do item passa a nascer ligada e é ligada nos itens
--    existentes; os materiais das OFs abertas acompanham. Item que deve sair só
--    por requisição/consumo manual é desmarcado no cadastro (VENT0200).

CREATE TABLE projects (
    id            BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT       NOT NULL REFERENCES enterprise(id),
    code          VARCHAR(50)  NOT NULL CHECK (btrim(code) <> ''),
    name          VARCHAR(200) NOT NULL CHECK (btrim(name) <> ''),
    customer_code BIGINT,
    status        VARCHAR(20)  NOT NULL DEFAULT 'ATIVO' CHECK (status IN ('ATIVO', 'ENCERRADO', 'CANCELADO')),
    start_date    DATE,
    end_date      DATE,
    notes         TEXT,
    created_by    UUID REFERENCES users(id),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT projects_empresa_id UNIQUE (enterprise_id, id),
    CONSTRAINT projects_cliente_fk FOREIGN KEY (enterprise_id, customer_code) REFERENCES customers(enterprise_id, code),
    CONSTRAINT projects_datas CHECK (end_date IS NULL OR start_date IS NULL OR end_date >= start_date)
);
CREATE UNIQUE INDEX ux_projects_codigo ON projects(enterprise_id, upper(btrim(code)));
CREATE INDEX ix_projects_cliente ON projects(enterprise_id, customer_code);

-- Projeto pelo código, criando-o quando ainda não está no cadastro (o código
-- veio digitado no pedido ou na demanda). Vazio = sem projeto.
CREATE OR REPLACE FUNCTION erp_projeto_do_codigo(p_empresa BIGINT, p_codigo TEXT, p_nome TEXT, p_cliente BIGINT)
RETURNS BIGINT LANGUAGE plpgsql AS $$
DECLARE
    v_id BIGINT;
BEGIN
    IF p_codigo IS NULL OR btrim(p_codigo) = '' THEN
        RETURN NULL;
    END IF;
    SELECT id INTO v_id FROM projects WHERE enterprise_id = p_empresa AND upper(btrim(code)) = upper(btrim(p_codigo));
    IF v_id IS NOT NULL THEN
        RETURN v_id;
    END IF;
    INSERT INTO projects(enterprise_id, code, name, customer_code)
    VALUES (p_empresa, btrim(p_codigo), COALESCE(NULLIF(btrim(p_nome), ''), btrim(p_codigo)),
            (SELECT c.code FROM customers c WHERE c.enterprise_id = p_empresa AND c.code = p_cliente))
    ON CONFLICT DO NOTHING
    RETURNING id INTO v_id;
    IF v_id IS NULL THEN
        SELECT id INTO v_id FROM projects WHERE enterprise_id = p_empresa AND upper(btrim(code)) = upper(btrim(p_codigo));
    END IF;
    RETURN v_id;
END;
$$;

-- Carga: os códigos que já estão nos pedidos de venda e nas demandas.
SELECT erp_projeto_do_codigo(e.id, so.project_code, so.project_name, so.customer_code)
  FROM (SELECT DISTINCT ON (enterprise_code, upper(btrim(project_code))) enterprise_code, project_code, project_name, customer_code
          FROM sales_orders WHERE COALESCE(btrim(project_code), '') <> ''
         ORDER BY enterprise_code, upper(btrim(project_code)), code DESC) so
  JOIN enterprise e ON e.code = so.enterprise_code;
SELECT erp_projeto_do_codigo(d.enterprise_id, d.projeto, NULL, d.customer_code)
  FROM (SELECT DISTINCT ON (enterprise_id, upper(btrim(projeto))) enterprise_id, projeto, customer_code
          FROM independent_demands WHERE COALESCE(btrim(projeto), '') <> ''
         ORDER BY enterprise_id, upper(btrim(projeto)), id DESC) d;

ALTER TABLE production_orders
    ADD COLUMN customer_code BIGINT,
    ADD COLUMN project_id    BIGINT,
    ADD CONSTRAINT production_orders_cliente_fk FOREIGN KEY (enterprise_id, customer_code) REFERENCES customers(enterprise_id, code),
    ADD CONSTRAINT production_orders_projeto_fk FOREIGN KEY (enterprise_id, project_id) REFERENCES projects(enterprise_id, id);
CREATE INDEX ix_production_orders_cliente ON production_orders(enterprise_id, customer_code) WHERE customer_code IS NOT NULL;
CREATE INDEX ix_production_orders_projeto ON production_orders(enterprise_id, project_id) WHERE project_id IS NOT NULL;

-- Preenche cliente e projeto que faltam na OF, pela origem dela.
CREATE OR REPLACE FUNCTION erp_of_herdar_origem(p_of BIGINT) RETURNS VOID LANGUAGE plpgsql AS $$
DECLARE
    o        production_orders%ROWTYPE;
    v_cli    BIGINT;
    v_cod    TEXT;
    v_nome   TEXT;
    v_proj   BIGINT;
    v_achou  BOOLEAN := FALSE;
BEGIN
    SELECT * INTO o FROM production_orders WHERE id = p_of;
    IF NOT FOUND OR (o.customer_code IS NOT NULL AND o.project_id IS NOT NULL) THEN
        RETURN;
    END IF;

    -- 1. Pedido de venda: o da OF ou o da ordem planejada dela.
    SELECT so.customer_code, so.project_code, so.project_name INTO v_cli, v_cod, v_nome
      FROM sales_orders so
      JOIN enterprise e ON e.code = so.enterprise_code AND e.id = o.enterprise_id
     WHERE so.code = o.sales_order_code
        OR (o.planned_order_id IS NOT NULL AND so.code IN (
              SELECT si.sales_order_code FROM sales_order_planned_orders sp
                JOIN sales_order_items si ON si.code = sp.sales_order_item_code
               WHERE sp.enterprise_id = o.enterprise_id AND sp.planned_order_id = o.planned_order_id))
     ORDER BY (so.code = o.sales_order_code) DESC NULLS LAST, so.code
     LIMIT 1;
    v_achou := FOUND;

    -- 2. Demanda independente: ligada à OF ou à ordem planejada dela.
    IF NOT v_achou THEN
        SELECT d.customer_code, d.projeto, NULL INTO v_cli, v_cod, v_nome
          FROM independent_demand_orders v
          JOIN independent_demands d ON d.id = v.demand_id AND d.enterprise_id = v.enterprise_id
         WHERE v.enterprise_id = o.enterprise_id
           AND (v.production_order_id = o.id OR (o.planned_order_id IS NOT NULL AND v.planned_order_id = o.planned_order_id))
         ORDER BY d.demand_date, d.id
         LIMIT 1;
        v_achou := FOUND;
    END IF;

    -- 3. OF pai da cadeia (pela planejada pai ou pela OF pai): o que ela já tem.
    IF NOT v_achou AND o.planned_order_id IS NOT NULL THEN
        SELECT p.customer_code, p.project_id INTO v_cli, v_proj
          FROM production_chain_links l
          JOIN production_orders p ON p.enterprise_id = l.enterprise_id
               AND (p.id = l.parent_production_order_id
                    OR (l.parent_production_order_id IS NULL AND p.planned_order_id = l.parent_planned_order_id AND p.is_active))
         WHERE l.enterprise_id = o.enterprise_id AND l.child_planned_order_id = o.planned_order_id
           AND (p.customer_code IS NOT NULL OR p.project_id IS NOT NULL)
         ORDER BY p.id
         LIMIT 1;
        v_achou := FOUND;
    END IF;

    IF NOT v_achou THEN
        RETURN;
    END IF;
    -- Cliente que não existe na empresa da OF não é levado (pedido antigo).
    IF v_cli IS NOT NULL AND NOT EXISTS (SELECT 1 FROM customers c WHERE c.enterprise_id = o.enterprise_id AND c.code = v_cli) THEN
        v_cli := NULL;
    END IF;
    IF v_proj IS NULL THEN
        v_proj := erp_projeto_do_codigo(o.enterprise_id, v_cod, v_nome, v_cli);
    END IF;
    UPDATE production_orders
       SET customer_code = COALESCE(customer_code, v_cli),
           project_id    = COALESCE(project_id, v_proj)
     WHERE id = p_of
       AND ((customer_code IS NULL AND v_cli IS NOT NULL) OR (project_id IS NULL AND v_proj IS NOT NULL));
END;
$$;

CREATE OR REPLACE FUNCTION trg_of_herdar_origem() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    PERFORM erp_of_herdar_origem(NEW.id);
    RETURN NULL;
END;
$$;
CREATE TRIGGER trg_of_herdar_origem AFTER INSERT ON production_orders
    FOR EACH ROW EXECUTE FUNCTION trg_of_herdar_origem();

-- OF pai ganhou ou trocou cliente/projeto: as filhas que não tinham, ou que
-- tinham o valor herdado (igual ao antigo da pai), acompanham. Valor próprio
-- da filha (diferente do da pai) fica. Desce a cadeia pelo mesmo gatilho.
CREATE OR REPLACE FUNCTION trg_of_passar_origem_as_filhas() RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    UPDATE production_orders c
       SET customer_code = CASE WHEN c.customer_code IS NULL OR c.customer_code IS NOT DISTINCT FROM OLD.customer_code
                                THEN NEW.customer_code ELSE c.customer_code END,
           project_id    = CASE WHEN c.project_id IS NULL OR c.project_id IS NOT DISTINCT FROM OLD.project_id
                                THEN NEW.project_id ELSE c.project_id END
      FROM production_chain_links l
     WHERE l.enterprise_id = NEW.enterprise_id
       AND c.enterprise_id = l.enterprise_id AND c.planned_order_id = l.child_planned_order_id AND c.is_active
       AND c.id <> NEW.id
       AND (l.parent_production_order_id = NEW.id
            OR (l.parent_production_order_id IS NULL AND NEW.planned_order_id IS NOT NULL AND l.parent_planned_order_id = NEW.planned_order_id))
       AND ((c.customer_code IS NULL OR c.customer_code IS NOT DISTINCT FROM OLD.customer_code) AND c.customer_code IS DISTINCT FROM NEW.customer_code
            OR (c.project_id IS NULL OR c.project_id IS NOT DISTINCT FROM OLD.project_id) AND c.project_id IS DISTINCT FROM NEW.project_id);
    RETURN NULL;
END;
$$;
CREATE TRIGGER trg_of_passar_origem_as_filhas AFTER UPDATE OF customer_code, project_id ON production_orders
    FOR EACH ROW WHEN (NEW.customer_code IS DISTINCT FROM OLD.customer_code OR NEW.project_id IS DISTINCT FROM OLD.project_id)
    EXECUTE FUNCTION trg_of_passar_origem_as_filhas();

-- Ligação nova (pedido, demanda, cadeia) depois de a OF existir.
CREATE OR REPLACE FUNCTION trg_origem_ligada_a_planejada() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE
    f BIGINT;
    v_planejada BIGINT;
BEGIN
    IF TG_TABLE_NAME = 'production_chain_links' THEN
        v_planejada := NEW.child_planned_order_id;
    ELSE
        v_planejada := NEW.planned_order_id;
    END IF;
    -- Só a demanda liga direto à OF. Em IF separado: o PL/pgSQL não deixa de
    -- avaliar NEW.production_order_id num AND e as outras tabelas não têm a coluna.
    IF TG_TABLE_NAME = 'independent_demand_orders' THEN
        IF NEW.production_order_id IS NOT NULL THEN
            PERFORM erp_of_herdar_origem(NEW.production_order_id);
        END IF;
    END IF;
    IF v_planejada IS NOT NULL THEN
        FOR f IN SELECT id FROM production_orders WHERE enterprise_id = NEW.enterprise_id AND planned_order_id = v_planejada AND is_active LOOP
            PERFORM erp_of_herdar_origem(f);
        END LOOP;
    END IF;
    RETURN NULL;
END;
$$;
CREATE TRIGGER trg_origem_pela_cadeia AFTER INSERT ON production_chain_links
    FOR EACH ROW EXECUTE FUNCTION trg_origem_ligada_a_planejada();
CREATE TRIGGER trg_origem_pelo_pedido AFTER INSERT ON sales_order_planned_orders
    FOR EACH ROW EXECUTE FUNCTION trg_origem_ligada_a_planejada();
CREATE TRIGGER trg_origem_pela_demanda AFTER INSERT ON independent_demand_orders
    FOR EACH ROW EXECUTE FUNCTION trg_origem_ligada_a_planejada();

-- Carga das OFs existentes: de cima para baixo (a pai antes da filha); o
-- gatilho de passagem às filhas completa o que a ordem por id não pegar.
SELECT erp_of_herdar_origem(id) FROM production_orders ORDER BY id;

-- 3. Baixa automática por padrão.
ALTER TABLE items ALTER COLUMN warehouse_automatic_low SET DEFAULT TRUE;
UPDATE items SET warehouse_automatic_low = TRUE WHERE NOT warehouse_automatic_low;
UPDATE production_order_materials m SET automatic_issue = TRUE, updated_at = NOW()
  FROM production_orders o
 WHERE o.id = m.production_order_id AND o.status IN ('OPEN', 'IN_PROGRESS') AND NOT m.automatic_issue;

SELECT instalar_conferencia_de_empresa();
