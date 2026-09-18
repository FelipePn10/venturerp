-- Roteiro no nível dos grandes ERPs: refugo por operação, ponto de inspeção e
-- documento de processo.
--
-- ─── 1. Refugo por operação ──────────────────────────────────────────────────
--
-- A perda que existia até aqui é a da ESTRUTURA: quanto de um componente se
-- perde ao consumi-lo. Ela não responde a outra pergunta, que é a que aparece
-- no chão de fábrica: se o corte refuga 3% e a solda 1%, quantas peças preciso
-- SOLTAR na primeira operação para entregar 100 boas no fim?
--
-- Sem essa conta a ordem sai curta. O planejador solta 100, o corte entrega 97,
-- a solda entrega 96, e faltam 4 peças descobertas só na expedição — quando não
-- há mais tempo de refazer. Focco, SAP (campo "refugo da operação") e Oracle
-- calculam assim; é o mínimo para o roteiro ser confiável.
--
-- A conta é a MESMA da estrutura (`QuantidadeComPerda`, fórmula "divide"):
--   entra = sai / (1 − refugo/100)
-- Uma fórmula só no sistema inteiro — foi a divergência entre duas contas de
-- perda que já custou uma correção antes.
--
-- Fica em dois lugares pelo mesmo motivo do modelo de tempo: a operação de
-- biblioteca traz o padrão e a etapa do roteiro sobrepõe, porque o mesmo
-- "rebarbar" refuga diferente numa peça delicada e numa peça bruta.

ALTER TABLE operations
    ADD COLUMN scrap_pct NUMERIC(6,3) NOT NULL DEFAULT 0;

ALTER TABLE operations
    ADD CONSTRAINT chk_operations_scrap CHECK (scrap_pct >= 0 AND scrap_pct < 100);

ALTER TABLE route_operations
    ADD COLUMN scrap_pct NUMERIC(6,3);

ALTER TABLE route_operations
    ADD CONSTRAINT chk_route_ops_scrap
        CHECK (scrap_pct IS NULL OR (scrap_pct >= 0 AND scrap_pct < 100));

COMMENT ON COLUMN operations.scrap_pct IS
    'Refugo padrão da operação, em %. Quanto do que entra não sai bom. Nunca 100: a operação inteira seria perda.';
COMMENT ON COLUMN route_operations.scrap_pct IS
    'Refugo desta etapa, em %. Nulo = herda o refugo da operação de biblioteca.';

-- A quantidade planejada de cada etapa da ordem passa a ser gravada: com
-- refugo ela é diferente em cada operação, e o apontamento precisa saber
-- quantas peças aquela etapa deveria entregar.
ALTER TABLE production_order_operations
    ADD COLUMN planned_qty NUMERIC(15,4);

COMMENT ON COLUMN production_order_operations.planned_qty IS
    'Quantidade que deve ENTRAR nesta operação para a ordem fechar a quantidade boa, já considerando o refugo das operações seguintes.';

-- ─── 2. Ponto de inspeção no roteiro ─────────────────────────────────────────
--
-- `inspection_plans.route_operation_id` já existia, mas nada no roteiro dizia
-- que aquela etapa tem inspeção — o vínculo só podia nascer do lado da
-- qualidade, então na prática ninguém criava. Marcar no roteiro é o que faz a
-- inspeção existir para quem desenha o processo.

ALTER TABLE route_operations
    ADD COLUMN inspection_required BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN route_operations.inspection_required IS
    'Etapa com inspeção de qualidade. A ordem de produção gera o registro de inspeção ao chegar nela.';

-- ─── 3. Documento de processo ────────────────────────────────────────────────
--
-- Desenho, ficha de processo, instrução de trabalho. O operador precisa abrir
-- isso no posto; hoje o desenho vive num diretório de rede e ninguém garante
-- que é a revisão certa.
--
-- Dois vínculos, exatamente um preenchido: a instrução genérica ("como rebarbar")
-- pertence à operação de biblioteca e vale em todo roteiro que a usa; o desenho
-- pertence à ETAPA, porque é do item.

CREATE TABLE operation_documents (
    id                 BIGSERIAL PRIMARY KEY,
    operation_id       BIGINT REFERENCES operations(id) ON DELETE CASCADE,
    route_operation_id BIGINT REFERENCES route_operations(id) ON DELETE CASCADE,
    kind               VARCHAR(20)  NOT NULL DEFAULT 'INSTRUCAO',
    title              VARCHAR(200) NOT NULL,
    reference          VARCHAR(300),
    revision           VARCHAR(30),
    instructions       TEXT,
    is_active          BOOLEAN      NOT NULL DEFAULT true,
    enterprise_id      BIGINT       NOT NULL REFERENCES enterprise(id),
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_by         UUID         NOT NULL,

    CONSTRAINT chk_operation_documents_vinculo CHECK (
        (operation_id IS NOT NULL AND route_operation_id IS NULL) OR
        (operation_id IS NULL AND route_operation_id IS NOT NULL)
    ),
    CONSTRAINT chk_operation_documents_kind CHECK (
        kind IN ('DESENHO', 'INSTRUCAO', 'FICHA', 'NORMA', 'FOTO', 'OUTRO')
    ),
    CONSTRAINT chk_operation_documents_titulo CHECK (length(btrim(title)) > 0)
);

CREATE INDEX idx_operation_documents_op    ON operation_documents(operation_id)       WHERE operation_id IS NOT NULL;
CREATE INDEX idx_operation_documents_rop   ON operation_documents(route_operation_id) WHERE route_operation_id IS NOT NULL;
CREATE INDEX idx_operation_documents_tenant ON operation_documents(enterprise_id, is_active);

COMMENT ON TABLE operation_documents IS
    'Desenho, instrução de trabalho e ficha de processo da operação. Vinculado à operação de biblioteca (vale em todo roteiro) OU a uma etapa de roteiro (é do item).';
COMMENT ON COLUMN operation_documents.reference IS
    'Onde o documento está: caminho, URL ou código no controle de documentos. O sistema não guarda o arquivo, guarda a referência e a revisão vigente.';
