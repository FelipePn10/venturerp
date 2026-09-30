-- Beneficiamento e estoque de terceiros.
--
-- A Usimac recebe matéria-prima do cliente por NF-e de remessa (CFOP 5901),
-- processa conforme o roteiro e devolve na MESMA nota em que fatura o serviço:
-- CFOP 5124 para o serviço de industrialização e CFOP 5902 para o material do
-- cliente voltando. Sobra, perda e sucata devolvidas saem por CFOP 5903.
--
-- Hoje esse controle é feito à mão. Não existe no sistema nenhuma noção de
-- material que está aqui mas NÃO é nosso.
--
-- Por que um livro-razão separado, e não uma coluna de proprietário em
-- `stock_balances`:
--
--   1. Material de terceiro não pode entrar em valoração de estoque, custeio nem
--      no líquido do MRP. Se morasse em `stock_balances`, toda consulta que hoje
--      soma saldo passaria a incluir, em silêncio, material que não é da empresa
--      — isso é erro contábil, não divergência de tela.
--   2. Ele tem dimensões que o estoque próprio não tem: cliente proprietário,
--      NF-e de remessa e linha da remessa. O saldo precisa fechar POR REMESSA,
--      porque é a remessa que tem prazo fiscal de retorno.
--   3. `stock_balances` é UNIQUE(item_code, mask, warehouse_id). Acrescentar
--      proprietário mexeria na tabela mais central do ERP, em produção, sem
--      benefício para quem já usa o sistema.
--
-- O estoque próprio continua exatamente como está. A tela de inventário mostra os
-- dois lados juntos, somando cada um no seu lugar.
--
-- Nome `customer_material_*`, e não `third_party_*`, de propósito: já existe o
-- módulo `third_party_service_*` (migration 000233), que é o sentido OPOSTO — nós
-- mandando operação de roteiro para fora (galvanização, têmpera, zincagem). Aqui é
-- material DO CLIENTE em poder da empresa. Os dois convivem e confundir um pelo
-- outro custaria caro.
--
-- Convenção de empresa: `enterprise_id`, como o resto do domínio operacional.

-- ABERTA: nada retornou ainda. PARCIAL: retornou parte. ENCERRADA: saldo zero, ou
-- encerramento manual aprovado com saldo remanescente registrado.
CREATE TYPE customer_material_remittance_status_enum AS ENUM ('ABERTA', 'PARCIAL', 'ENCERRADA', 'CANCELADA');

-- RECEIPT entra; os demais saem. LEFTOVER é material não utilizado voltando,
-- SCRAP é sucata gerada no processo, ADJUSTMENT é correção justificada.
CREATE TYPE customer_material_movement_type_enum AS ENUM ('RECEIPT', 'RETURN', 'LEFTOVER', 'SCRAP', 'ADJUSTMENT');

-- Destinação da sucata, conforme as regras que o cliente documentou.
CREATE TYPE customer_material_scrap_destination_enum AS ENUM ('CLIENTE', 'DESCARTE', 'RETENCAO', 'OUTRA');

-- ── Remessa recebida do cliente ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS customer_material_remittances (
    id                     BIGSERIAL PRIMARY KEY,
    enterprise_id          BIGINT NOT NULL REFERENCES enterprise(id),
    -- Cliente proprietário do material. Guardado por CÓDIGO público, como o resto
    -- do módulo comercial, e não por id interno.
    customer_code          BIGINT NOT NULL,
    nfe_number             BIGINT NOT NULL,
    nfe_series             VARCHAR(10) NOT NULL DEFAULT '1',
    -- Chave de acesso de 44 dígitos, quando o XML está disponível.
    nfe_key                VARCHAR(44),
    cfop                   VARCHAR(4) NOT NULL DEFAULT '5901',
    issue_date             DATE NOT NULL,
    received_at            DATE NOT NULL,
    -- Prazo fiscal para o retorno: 30 dias na operação da Usimac. Fica gravado
    -- por remessa porque é o que dispara a cobrança de quem está perto de vencer.
    fiscal_return_deadline DATE NOT NULL,
    total_value            NUMERIC(15,2) NOT NULL DEFAULT 0,
    status                 customer_material_remittance_status_enum NOT NULL DEFAULT 'ABERTA',
    -- Pedido de beneficiamento. Nulo é caso real e previsto: "material recebido
    -- sem pedido previamente cadastrado" deve entrar e ficar BLOQUEADO.
    sales_order_code       BIGINT,
    -- Bloqueio cobre material sem pedido e recebimento divergente. Enquanto
    -- bloqueado, o material não pode ser usado na produção.
    blocked                BOOLEAN NOT NULL DEFAULT FALSE,
    block_reason           TEXT,
    -- Encerramento com saldo exige aprovação formal; o motivo e quem aprovou
    -- ficam registrados, porque é exceção.
    closed_at              TIMESTAMPTZ,
    closed_by              UUID REFERENCES users(id),
    close_reason           TEXT,
    notes                  TEXT,
    created_by             UUID NOT NULL REFERENCES users(id),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- A mesma NF-e do mesmo cliente não entra duas vezes. Por empresa, porque em
    -- bases distintas de clientes distintos o número pode repetir.
    CONSTRAINT customer_material_remittances_nfe_unica
        UNIQUE (enterprise_id, customer_code, nfe_number, nfe_series),
    -- Bloqueio sem motivo é bloqueio que ninguém sabe resolver.
    CONSTRAINT customer_material_remittances_bloqueio_com_motivo
        CHECK (blocked = FALSE OR block_reason IS NOT NULL),
    CONSTRAINT customer_material_remittances_prazo_apos_emissao
        CHECK (fiscal_return_deadline >= issue_date)
);

CREATE INDEX idx_customer_material_remittances_cliente
    ON customer_material_remittances (enterprise_id, customer_code, status);
-- Sustenta a cobrança por prazo fiscal: remessas abertas vencendo primeiro.
CREATE INDEX idx_customer_material_remittances_prazo
    ON customer_material_remittances (enterprise_id, fiscal_return_deadline)
    WHERE status IN ('ABERTA', 'PARCIAL');
CREATE INDEX idx_customer_material_remittances_pedido
    ON customer_material_remittances (enterprise_id, sales_order_code)
    WHERE sales_order_code IS NOT NULL;

-- ── Itens da remessa: é aqui que o saldo de terceiro vive ────────────────────
CREATE TABLE IF NOT EXISTS customer_material_items (
    id                  BIGSERIAL PRIMARY KEY,
    enterprise_id       BIGINT NOT NULL REFERENCES enterprise(id),
    remittance_id       BIGINT NOT NULL REFERENCES customer_material_remittances(id) ON DELETE CASCADE,
    line_number         INT NOT NULL,
    -- Código do item COMO VEIO na nota do cliente. Não é o nosso código: o
    -- retorno fiscal tem de sair com a identificação que o cliente reconhece.
    customer_item_code  VARCHAR(60) NOT NULL,
    -- Item interno correspondente, quando existir cadastro. Opcional de propósito:
    -- o controle do material de terceiro não depende de cadastrá-lo como item
    -- nosso, e forçar isso poluiria o cadastro com peças de cliente.
    item_code           BIGINT,
    description         VARCHAR(200) NOT NULL,
    -- NCM da nota de ENTRADA. O retorno tem de sair com o mesmo NCM, e por isso
    -- ele é copiado aqui em vez de buscado no cadastro.
    ncm                 VARCHAR(10) NOT NULL,
    -- CST da entrada: 050 (suspensão) na remessa para industrialização.
    cst                 VARCHAR(3),
    uom                 VARCHAR(6) NOT NULL,
    qty_invoiced        NUMERIC(18,6) NOT NULL,
    -- Quantidade FÍSICA conferida. Pode divergir da nota, e a divergência é
    -- registrada em vez de silenciada.
    qty_received        NUMERIC(18,6) NOT NULL,
    unit_value          NUMERIC(15,6) NOT NULL DEFAULT 0,
    -- Consumidas pelos movimentos, mantidas na mesma transação do movimento.
    qty_returned        NUMERIC(18,6) NOT NULL DEFAULT 0,
    qty_leftover        NUMERIC(18,6) NOT NULL DEFAULT 0,
    qty_scrapped        NUMERIC(18,6) NOT NULL DEFAULT 0,
    -- Saldo de terceiro em poder da Usimac. Derivado da própria linha, então não
    -- existe estado para divergir do que os movimentos dizem.
    balance_qty         NUMERIC(18,6)
        GENERATED ALWAYS AS (qty_received - qty_returned - qty_leftover - qty_scrapped) STORED,
    divergence_qty      NUMERIC(18,6) GENERATED ALWAYS AS (qty_received - qty_invoiced) STORED,
    divergence_reason   TEXT,
    divergence_settled_by UUID REFERENCES users(id),
    divergence_settled_at TIMESTAMPTZ,
    -- Onde o material do cliente está guardado. Separar fisicamente é exigência
    -- do cliente: material próprio e de terceiro não se misturam.
    warehouse_id        BIGINT,
    address             VARCHAR(100),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT customer_material_items_linha_unica
        UNIQUE (remittance_id, line_number),
    CONSTRAINT customer_material_items_qtd_positiva
        CHECK (qty_invoiced > 0 AND qty_received >= 0),
    CONSTRAINT customer_material_items_consumos_nao_negativos
        CHECK (qty_returned >= 0 AND qty_leftover >= 0 AND qty_scrapped >= 0),
    -- Rede de segurança no banco: devolver mais do que entrou é impossível, mesmo
    -- que uma regra de aplicação falhe.
    CONSTRAINT customer_material_items_saldo_nao_negativo
        CHECK (qty_received - qty_returned - qty_leftover - qty_scrapped >= 0),
    -- Divergência conferida sem motivo não explica nada a quem for regularizar.
    CONSTRAINT customer_material_items_divergencia_com_motivo
        CHECK (qty_received = qty_invoiced OR divergence_reason IS NOT NULL)
);

CREATE INDEX idx_customer_material_items_remessa
    ON customer_material_items (remittance_id);
-- Sustenta "quanto deste item eu tenho deste cliente": o saldo por item.
CREATE INDEX idx_customer_material_items_saldo
    ON customer_material_items (enterprise_id, customer_item_code)
    WHERE balance_qty > 0;
CREATE INDEX idx_customer_material_items_item_interno
    ON customer_material_items (enterprise_id, item_code)
    WHERE item_code IS NOT NULL;

-- ── Livro-razão: todo movimento de material de terceiro ──────────────────────
CREATE TABLE IF NOT EXISTS customer_material_movements (
    id                  BIGSERIAL PRIMARY KEY,
    enterprise_id       BIGINT NOT NULL REFERENCES enterprise(id),
    remittance_item_id  BIGINT NOT NULL REFERENCES customer_material_items(id) ON DELETE CASCADE,
    movement_type       customer_material_movement_type_enum NOT NULL,
    quantity            NUMERIC(18,6) NOT NULL,
    unit_value          NUMERIC(15,6) NOT NULL DEFAULT 0,
    -- CFOP com que o movimento saiu: 5902 no retorno com o serviço, 5903 em sobra
    -- e sucata devolvidas.
    cfop                VARCHAR(4),
    -- Ordem de produção que consumiu, quando houver.
    production_order_id BIGINT,
    -- NF de saída que documentou o movimento. Nulo até a nota ser emitida.
    fiscal_exit_id      BIGINT,
    scrap_destination   customer_material_scrap_destination_enum,
    reason              TEXT,
    -- Mesma disciplina do módulo de serviços de terceiros: repetir a requisição
    -- devolve o movimento original em vez de duplicar a baixa.
    idempotency_key     VARCHAR(120) NOT NULL,
    created_by          UUID NOT NULL REFERENCES users(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT customer_material_movements_idempotencia
        UNIQUE (enterprise_id, idempotency_key),
    CONSTRAINT customer_material_movements_qtd_positiva
        CHECK (quantity > 0),
    -- Sucata precisa dizer para onde foi; as regras do cliente exigem destinação
    -- registrada em toda sucata gerada.
    CONSTRAINT customer_material_movements_sucata_com_destino
        CHECK (movement_type <> 'SCRAP' OR scrap_destination IS NOT NULL),
    -- Ajuste sem justificativa é ajuste que ninguém audita.
    CONSTRAINT customer_material_movements_ajuste_com_motivo
        CHECK (movement_type <> 'ADJUSTMENT' OR reason IS NOT NULL)
);

CREATE INDEX idx_customer_material_movements_item
    ON customer_material_movements (remittance_item_id, created_at);
CREATE INDEX idx_customer_material_movements_nota
    ON customer_material_movements (enterprise_id, fiscal_exit_id)
    WHERE fiscal_exit_id IS NOT NULL;
CREATE INDEX idx_customer_material_movements_ordem
    ON customer_material_movements (enterprise_id, production_order_id)
    WHERE production_order_id IS NOT NULL;
