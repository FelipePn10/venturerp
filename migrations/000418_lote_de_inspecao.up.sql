-- Lote de inspeção único (estilo SAP QM): a inspeção de recebimento de compras
-- (receiving_inspection_orders + resultados + análise; procurement_records do
-- tipo RECEIVING_INSPECTION + destinação) e o registro de qualidade da produção
-- (quality_records + medições) passam a ser um só objeto, do módulo de
-- qualidade, com:
--   - as características copiadas do plano na abertura (o lote não muda se o
--     roteiro mudar depois);
--   - o plano de amostragem NBR 5426 calculado na abertura (2.1);
--   - resultado por característica × unidade da amostra;
--   - decisão de uso imutável, gravada junto com as transferências de estoque.
--
-- As tabelas antigas ficam (sem escrita nova) e os dados são copiados. O id da
-- ordem de inspeção de recebimento é preservado como id do lote; registro de
-- qualidade e registro de procurement ficam mapeados em legacy_kind/legacy_id.
-- A sequência nova começa acima de todos os ids antigos, então as rotas antigas
-- (adaptadores) não confundem um id novo com um antigo.

CREATE TABLE quality_inspection_lots (
    id                       BIGSERIAL     PRIMARY KEY,
    enterprise_id            BIGINT        NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    numero                   BIGINT        NOT NULL,
    origem                   VARCHAR(12)   NOT NULL CHECK (origem IN ('RECEBIMENTO', 'PROCESSO', 'FINAL', 'DEVOLUCAO', 'MANUAL')),
    status                   VARCHAR(10)   NOT NULL DEFAULT 'ABERTO' CHECK (status IN ('ABERTO', 'DECIDIDO', 'CANCELADO')),
    item_code                BIGINT        NOT NULL,
    mask                     TEXT          NOT NULL DEFAULT '',
    lote                     TEXT,
    serie                    TEXT,
    quantidade               NUMERIC(18,6) NOT NULL CHECK (quantidade > 0),
    -- Almoxarifado onde o material espera a decisão (INSPECAO). NULL: inspeção
    -- sem estoque retido (processo).
    warehouse_id             BIGINT        REFERENCES warehouse(id),
    supplier_code            BIGINT,
    purchase_order_code      BIGINT,
    purchase_order_item_code BIGINT,
    fiscal_entry_code        BIGINT,
    receiving_notice_code    BIGINT,
    production_order_id      BIGINT,
    route_operation_id       BIGINT,
    receiving_route_id       BIGINT,
    inspection_plan_id       BIGINT,
    sampling_procedure_id    BIGINT,
    chave_amostragem         VARCHAR(40),
    -- Plano de amostragem usado (n, Ac, Re, severidade, letra) — o da abertura.
    plano                    JSONB,
    -- Skip-lot: liberado sem amostra.
    pulado                   BOOLEAN       NOT NULL DEFAULT FALSE,
    certificado              TEXT,
    nota_fornecedor          TEXT,
    modelo                   TEXT,
    observacoes              TEXT,
    motivo_cancelamento      TEXT,
    legacy_kind              VARCHAR(20)   CHECK (legacy_kind IN ('RECEIVING_ORDER', 'QUALITY_RECORD', 'PROCUREMENT_RECORD')),
    legacy_id                BIGINT,
    created_at               TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_by               UUID          REFERENCES users(id),
    updated_at               TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, numero),
    UNIQUE (enterprise_id, id),
    UNIQUE (legacy_kind, legacy_id),
    FOREIGN KEY (enterprise_id, sampling_procedure_id) REFERENCES quality_sampling_procedures(enterprise_id, id)
);
CREATE INDEX idx_quality_lots_status ON quality_inspection_lots(enterprise_id, status, created_at DESC);
CREATE INDEX idx_quality_lots_item ON quality_inspection_lots(enterprise_id, item_code, created_at DESC);
CREATE INDEX idx_quality_lots_po ON quality_inspection_lots(enterprise_id, production_order_id) WHERE production_order_id IS NOT NULL;
CREATE INDEX idx_quality_lots_supplier ON quality_inspection_lots(enterprise_id, supplier_code, created_at DESC) WHERE supplier_code IS NOT NULL;

CREATE TABLE quality_lot_characteristics (
    id                       BIGSERIAL     PRIMARY KEY,
    lot_id                   BIGINT        NOT NULL,
    enterprise_id            BIGINT        NOT NULL,
    sequencia                INTEGER       NOT NULL CHECK (sequencia > 0),
    nome                     TEXT          NOT NULL,
    -- VARIAVEL (mede um valor), ATRIBUTO (conforme / não conforme), ESTRUTURA.
    tipo                     VARCHAR(10)   NOT NULL CHECK (tipo IN ('VARIAVEL', 'ATRIBUTO', 'ESTRUTURA')),
    obrigatoria              BOOLEAN       NOT NULL DEFAULT TRUE,
    critica                  BOOLEAN       NOT NULL DEFAULT FALSE,
    nominal                  NUMERIC(18,6),
    minimo                   NUMERIC(18,6),
    maximo                   NUMERIC(18,6),
    unidade                  TEXT,
    instrumento              TEXT,
    norma                    TEXT,
    referencia               TEXT,
    origem_step_id           BIGINT,
    origem_characteristic_id BIGINT,
    UNIQUE (lot_id, sequencia),
    FOREIGN KEY (enterprise_id, lot_id) REFERENCES quality_inspection_lots(enterprise_id, id) ON DELETE CASCADE,
    CHECK (minimo IS NULL OR maximo IS NULL OR minimo <= maximo)
);

CREATE TABLE quality_lot_results (
    id               BIGSERIAL     PRIMARY KEY,
    lot_id           BIGINT        NOT NULL,
    enterprise_id    BIGINT        NOT NULL,
    characteristic_id BIGINT       NOT NULL REFERENCES quality_lot_characteristics(id) ON DELETE CASCADE,
    amostra_indice   INTEGER       NOT NULL CHECK (amostra_indice > 0),
    valor            NUMERIC(18,6),
    atributo         TEXT,
    conforme         BOOLEAN       NOT NULL,
    observacao       TEXT,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_by       UUID          REFERENCES users(id),
    UNIQUE (characteristic_id, amostra_indice),
    FOREIGN KEY (enterprise_id, lot_id) REFERENCES quality_inspection_lots(enterprise_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_quality_lot_results_lot ON quality_lot_results(lot_id);

CREATE TABLE quality_lot_decisions (
    lot_id             BIGINT        PRIMARY KEY,
    enterprise_id      BIGINT        NOT NULL,
    -- ACEITO, ACEITO_COM_RESTRICAO, REJEITADO ou PARCIAL — derivado das quantidades.
    decisao            VARCHAR(22)   NOT NULL CHECK (decisao IN ('ACEITO', 'ACEITO_COM_RESTRICAO', 'REJEITADO', 'PARCIAL')),
    qtd_aceita         NUMERIC(18,6) NOT NULL DEFAULT 0 CHECK (qtd_aceita >= 0),
    qtd_restrita       NUMERIC(18,6) NOT NULL DEFAULT 0 CHECK (qtd_restrita >= 0),
    qtd_retrabalho     NUMERIC(18,6) NOT NULL DEFAULT 0 CHECK (qtd_retrabalho >= 0),
    qtd_rejeitada      NUMERIC(18,6) NOT NULL DEFAULT 0 CHECK (qtd_rejeitada >= 0),
    qtd_sucata         NUMERIC(18,6) NOT NULL DEFAULT 0 CHECK (qtd_sucata >= 0),
    destino_aceito     BIGINT        REFERENCES warehouse(id),
    destino_restrito   BIGINT        REFERENCES warehouse(id),
    destino_retrabalho BIGINT        REFERENCES warehouse(id),
    destino_rejeitado  BIGINT        REFERENCES warehouse(id),
    destino_sucata     BIGINT        REFERENCES warehouse(id),
    -- Tratamento da parte não aceita (devolver, sucatear, retrabalhar, selecionar, concessão).
    tratamento         VARCHAR(20)   CHECK (tratamento IN ('DEVOLVER', 'SUCATEAR', 'RETRABALHAR', 'SELECIONAR', 'CONCESSAO')),
    nao_conformes      INTEGER       CHECK (nao_conformes >= 0),
    -- Resultado do lote pela amostragem (aceito/recusado), independente de quanto foi aproveitado.
    aceito_amostragem  BOOLEAN,
    afeta_iqf          BOOLEAN       NOT NULL DEFAULT TRUE,
    motivo             TEXT,
    decided_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    decided_by         UUID          REFERENCES users(id),
    FOREIGN KEY (enterprise_id, lot_id) REFERENCES quality_inspection_lots(enterprise_id, id) ON DELETE CASCADE
);

CREATE FUNCTION quality_lot_decisions_imutaveis() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'a decisão de uso do lote de inspeção não pode ser alterada';
END $$;
CREATE TRIGGER trg_quality_lot_decisions_imutaveis BEFORE UPDATE ON quality_lot_decisions
    FOR EACH ROW EXECUTE FUNCTION quality_lot_decisions_imutaveis();

CREATE TABLE quality_lot_supplier_reports (
    lot_id            BIGINT      NOT NULL,
    enterprise_id     BIGINT      NOT NULL,
    quality_report_id BIGINT      NOT NULL,
    linked_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    linked_by         UUID        NOT NULL REFERENCES users(id),
    PRIMARY KEY (lot_id, quality_report_id),
    FOREIGN KEY (enterprise_id, lot_id) REFERENCES quality_inspection_lots(enterprise_id, id) ON DELETE CASCADE,
    FOREIGN KEY (enterprise_id, quality_report_id) REFERENCES item_supplier_quality_reports(enterprise_id, id) ON DELETE RESTRICT
);

-- O plano diz como amostrar.
ALTER TABLE receiving_inspection_routes ADD COLUMN sampling_procedure_id BIGINT REFERENCES quality_sampling_procedures(id);
ALTER TABLE inspection_plans ADD COLUMN sampling_procedure_id BIGINT REFERENCES quality_sampling_procedures(id);
-- A NC passa a apontar o lote.
ALTER TABLE non_conformances ADD COLUMN lot_id BIGINT REFERENCES quality_inspection_lots(id);

-- ─── Dados antigos ────────────────────────────────────────────────────────────

-- 1) Ordens de inspeção de recebimento: mesmo id.
INSERT INTO quality_inspection_lots (id, enterprise_id, numero, origem, status, item_code, mask, lote, serie, quantidade,
        warehouse_id, supplier_code, purchase_order_code, purchase_order_item_code, fiscal_entry_code, receiving_notice_code,
        receiving_route_id, pulado, certificado, nota_fornecedor, modelo, observacoes, legacy_kind, legacy_id,
        created_at, created_by, updated_at)
SELECT o.id, e.id, o.order_number,
       CASE WHEN o.source::text = 'MANUAL' THEN 'MANUAL' ELSE 'RECEBIMENTO' END,
       CASE WHEN o.status::text IN ('PENDING_INSPECTION', 'PENDING_ANALYSIS') THEN 'ABERTO'
            WHEN o.status::text = 'CANCELLED' THEN 'CANCELADO' ELSE 'DECIDIDO' END,
       o.item_code, o.mask, o.lot, o.serial_number, o.quantity,
       o.warehouse_id, o.supplier_code, o.purchase_order_code, o.purchase_order_item_code, o.fiscal_entry_code, o.receiving_notice_code,
       o.route_id, o.status::text = 'SKIPPED', o.certificate, o.supplier_note, o.model, o.notes, 'RECEIVING_ORDER', o.id,
       o.created_at, o.created_by, o.updated_at
  FROM receiving_inspection_orders o
  JOIN enterprise e ON e.code = o.enterprise_code;

INSERT INTO quality_lot_characteristics (lot_id, enterprise_id, sequencia, nome, tipo, obrigatoria, nominal, minimo, maximo,
        unidade, instrumento, norma, referencia, origem_step_id)
SELECT l.id, l.enterprise_id, s.sequence, s.inspection_name,
       CASE s.kind::text WHEN 'VALUE' THEN 'VARIAVEL' WHEN 'ATTRIBUTE' THEN 'ATRIBUTO' ELSE 'ESTRUTURA' END,
       s.is_required, s.nominal_value, s.min_value, s.max_value, s.sample_unit, s.instrument_group, s.norm, s.reference, s.id
  FROM quality_inspection_lots l
  JOIN receiving_inspection_route_steps s ON s.route_id = l.receiving_route_id
 WHERE l.legacy_kind = 'RECEIVING_ORDER'
   AND (s.min_value IS NULL OR s.max_value IS NULL OR s.min_value <= s.max_value);
-- Resultado sem etapa no roteiro: característica pela sequência gravada.
INSERT INTO quality_lot_characteristics (lot_id, enterprise_id, sequencia, nome, tipo, obrigatoria)
SELECT DISTINCT l.id, l.enterprise_id, r.sequence, 'Sequência ' || r.sequence,
       CASE WHEN BOOL_OR(r.measured_value IS NOT NULL) OVER (PARTITION BY l.id, r.sequence) THEN 'VARIAVEL' ELSE 'ATRIBUTO' END, FALSE
  FROM quality_inspection_lots l
  JOIN receiving_inspection_results r ON r.order_id = l.id
 WHERE l.legacy_kind = 'RECEIVING_ORDER'
   AND NOT EXISTS (SELECT 1 FROM quality_lot_characteristics c WHERE c.lot_id = l.id AND c.sequencia = r.sequence);
INSERT INTO quality_lot_results (lot_id, enterprise_id, characteristic_id, amostra_indice, valor, atributo, conforme, observacao, created_at, created_by)
SELECT DISTINCT ON (c.id, GREATEST(r.sample_index, 1))
       l.id, l.enterprise_id, c.id, GREATEST(r.sample_index, 1), r.measured_value, r.attribute_description, r.is_approved, r.notes, r.created_at, r.created_by
  FROM quality_inspection_lots l
  JOIN receiving_inspection_results r ON r.order_id = l.id
  JOIN quality_lot_characteristics c ON c.lot_id = l.id AND c.sequencia = r.sequence
 WHERE l.legacy_kind = 'RECEIVING_ORDER'
 ORDER BY c.id, GREATEST(r.sample_index, 1), r.created_at DESC, r.id DESC;
-- Decisão: soma das análises; sem análise, as quantidades gravadas na ordem.
INSERT INTO quality_lot_decisions (lot_id, enterprise_id, decisao, qtd_aceita, qtd_restrita, qtd_retrabalho, qtd_rejeitada,
        tratamento, afeta_iqf, motivo, decided_at, decided_by)
SELECT l.id, l.enterprise_id,
       CASE WHEN q.rej + q.ret = 0 AND q.res = 0 THEN 'ACEITO'
            WHEN q.rej + q.ret = 0 THEN 'ACEITO_COM_RESTRICAO'
            WHEN q.ace + q.res = 0 THEN 'REJEITADO' ELSE 'PARCIAL' END,
       q.ace, q.res, q.ret, q.rej, q.trat, q.iqf, q.notas, q.quando, q.quem
  FROM quality_inspection_lots l
  JOIN receiving_inspection_orders o ON o.id = l.id
  CROSS JOIN LATERAL (
      SELECT COALESCE(SUM(a.conform_qty), o.approved_qty) ace, COALESCE(SUM(a.restricted_qty), o.restricted_qty) res,
             COALESCE(SUM(a.rework_qty), o.rework_qty) ret, COALESCE(SUM(a.rejected_qty), o.rejected_qty) rej,
             (ARRAY_AGG(CASE a.treatment::text WHEN 'RETURN_TO_SUPPLIER' THEN 'DEVOLVER' WHEN 'SCRAP' THEN 'SUCATEAR'
                  WHEN 'REWORK' THEN 'RETRABALHAR' WHEN 'SORTING' THEN 'SELECIONAR' WHEN 'CONCESSION' THEN 'CONCESSAO' END
                  ORDER BY a.analyzed_at DESC))[1] trat,
             COALESCE(BOOL_OR(a.affects_supplier_score), TRUE) iqf,
             STRING_AGG(a.notes, ' | ' ORDER BY a.analyzed_at) notas,
             COALESCE(MAX(a.analyzed_at), o.updated_at) quando,
             (ARRAY_AGG(a.analyzed_by ORDER BY a.analyzed_at DESC))[1] quem
        FROM receiving_inspection_analyses a WHERE a.order_id = o.id) q
 WHERE l.legacy_kind = 'RECEIVING_ORDER' AND l.status = 'DECIDIDO';

INSERT INTO quality_lot_supplier_reports (lot_id, enterprise_id, quality_report_id, linked_at, linked_by)
SELECT r.inspection_order_id, r.enterprise_id, r.quality_report_id, r.linked_at, r.linked_by
  FROM receiving_inspection_quality_reports r
  JOIN quality_inspection_lots l ON l.id = r.inspection_order_id AND l.enterprise_id = r.enterprise_id;

-- A partir daqui os ids novos ficam acima de todos os antigos.
SELECT setval(pg_get_serial_sequence('quality_inspection_lots', 'id'), GREATEST(
    (SELECT COALESCE(MAX(id), 0) FROM receiving_inspection_orders),
    (SELECT COALESCE(MAX(id), 0) FROM quality_records),
    (SELECT COALESCE(MAX(id), 0) FROM procurement_records),
    1));

-- 2) Registros de qualidade da produção: empresa pelo item (código global).
INSERT INTO quality_inspection_lots (enterprise_id, numero, origem, status, item_code, lote, quantidade, production_order_id,
        route_operation_id, inspection_plan_id, observacoes, legacy_kind, legacy_id, created_at, created_by, updated_at)
SELECT it.enterprise_id,
       COALESCE((SELECT MAX(numero) FROM quality_inspection_lots x WHERE x.enterprise_id = it.enterprise_id), 0)
         + ROW_NUMBER() OVER (PARTITION BY it.enterprise_id ORDER BY qr.id),
       CASE p.point_type::text WHEN 'RECEBIMENTO' THEN 'MANUAL' WHEN 'EXPEDICAO' THEN 'FINAL' ELSE 'PROCESSO' END,
       CASE WHEN qr.result::text = 'PENDENTE' THEN 'ABERTO' ELSE 'DECIDIDO' END,
       qr.item_code, qr.lot,
       COALESCE(NULLIF(qr.inspected_qty, 0), NULLIF(po.planned_qty, 0), 1),
       qr.production_order_id, p.route_operation_id, qr.plan_id, qr.notes, 'QUALITY_RECORD', qr.id,
       qr.created_at, qr.created_by, qr.created_at
  FROM quality_records qr
  JOIN items it ON it.code = qr.item_code
  JOIN inspection_plans p ON p.id = qr.plan_id
  LEFT JOIN production_orders po ON po.id = qr.production_order_id;

INSERT INTO quality_lot_characteristics (lot_id, enterprise_id, sequencia, nome, tipo, critica, nominal, minimo, maximo, unidade, origem_characteristic_id)
SELECT l.id, l.enterprise_id, ROW_NUMBER() OVER (PARTITION BY l.id ORDER BY c.id), c.name,
       CASE WHEN c.nominal IS NULL AND c.tolerance_lower IS NULL AND c.tolerance_upper IS NULL THEN 'ATRIBUTO' ELSE 'VARIAVEL' END,
       c.is_critical, c.nominal, c.tolerance_lower, c.tolerance_upper, c.unit, c.id
  FROM quality_inspection_lots l
  JOIN inspection_plan_characteristics c ON c.plan_id = l.inspection_plan_id
 WHERE l.legacy_kind = 'QUALITY_RECORD';
INSERT INTO quality_lot_results (lot_id, enterprise_id, characteristic_id, amostra_indice, valor, conforme)
SELECT l.id, l.enterprise_id, c.id, ROW_NUMBER() OVER (PARTITION BY c.id ORDER BY m.id), m.measured_value, m.is_conformant
  FROM quality_inspection_lots l
  JOIN quality_measurements m ON m.record_id = l.legacy_id
  JOIN quality_lot_characteristics c ON c.lot_id = l.id AND c.origem_characteristic_id = m.characteristic_id
 WHERE l.legacy_kind = 'QUALITY_RECORD';
INSERT INTO quality_lot_decisions (lot_id, enterprise_id, decisao, qtd_aceita, qtd_restrita, qtd_rejeitada, aceito_amostragem, decided_at, decided_by)
SELECT l.id, l.enterprise_id,
       CASE qr.result::text WHEN 'APROVADO' THEN 'ACEITO' WHEN 'CONDICIONAL' THEN 'ACEITO_COM_RESTRICAO'
            ELSE CASE WHEN qr.approved_qty > 0 THEN 'PARCIAL' ELSE 'REJEITADO' END END,
       CASE WHEN qr.result::text = 'CONDICIONAL' THEN 0 ELSE qr.approved_qty END,
       CASE WHEN qr.result::text = 'CONDICIONAL' THEN qr.approved_qty ELSE 0 END,
       qr.rejected_qty, qr.result::text <> 'REJEITADO', qr.inspected_at, qr.created_by
  FROM quality_inspection_lots l
  JOIN quality_records qr ON qr.id = l.legacy_id
 WHERE l.legacy_kind = 'QUALITY_RECORD' AND l.status = 'DECIDIDO';
UPDATE non_conformances n SET lot_id = l.id
  FROM quality_inspection_lots l WHERE l.legacy_kind = 'QUALITY_RECORD' AND l.legacy_id = n.quality_record_id;

-- 3) Registros de procurement do tipo inspeção de recebimento (+ destinação).
INSERT INTO quality_inspection_lots (enterprise_id, numero, origem, status, item_code, mask, quantidade, warehouse_id,
        supplier_code, purchase_order_code, purchase_order_item_code, observacoes, legacy_kind, legacy_id, created_at, created_by, updated_at)
SELECT e.id,
       COALESCE((SELECT MAX(numero) FROM quality_inspection_lots x WHERE x.enterprise_id = e.id), 0)
         + ROW_NUMBER() OVER (PARTITION BY e.id ORDER BY r.id),
       'RECEBIMENTO',
       CASE WHEN r.status::text IN ('CANCELLED') THEN 'CANCELADO'
            WHEN r.status::text IN ('APPROVED', 'REJECTED', 'PARTIAL', 'CLOSED') OR EXISTS (SELECT 1 FROM procurement_inspection_dispositions d WHERE d.record_id = r.id) THEN 'DECIDIDO'
            ELSE 'ABERTO' END,
       r.item_code, r.mask, r.quantity, r.warehouse_id, r.supplier_code, r.purchase_order_code, r.purchase_order_item_code, r.reference,
       'PROCUREMENT_RECORD', r.id, r.opened_at, r.created_by, r.updated_at
  FROM procurement_records r
  JOIN enterprise e ON e.code = r.enterprise_code
 WHERE r.record_type::text = 'RECEIVING_INSPECTION' AND r.item_code IS NOT NULL AND r.quantity > 0;
INSERT INTO quality_lot_decisions (lot_id, enterprise_id, decisao, qtd_aceita, qtd_rejeitada, destino_aceito, destino_rejeitado, motivo, decided_at, decided_by)
SELECT l.id, l.enterprise_id,
       CASE WHEN q.rej = 0 THEN 'ACEITO' WHEN q.ace = 0 THEN 'REJEITADO' ELSE 'PARCIAL' END,
       q.ace, q.rej, q.dest, q.quar, q.motivo, q.quando, q.quem
  FROM quality_inspection_lots l
  CROSS JOIN LATERAL (
      SELECT COALESCE(SUM(d.approved_qty), 0) ace, COALESCE(SUM(d.rejected_qty), 0) rej,
             (ARRAY_AGG(d.destination_warehouse_id ORDER BY d.disposed_at DESC))[1] dest,
             (ARRAY_AGG(d.quarantine_warehouse_id ORDER BY d.disposed_at DESC))[1] quar,
             STRING_AGG(d.reason, ' | ' ORDER BY d.disposed_at) motivo,
             COALESCE(MAX(d.disposed_at), l.updated_at) quando,
             (ARRAY_AGG(d.disposed_by ORDER BY d.disposed_at DESC))[1] quem
        FROM procurement_inspection_dispositions d WHERE d.record_id = l.legacy_id) q
 WHERE l.legacy_kind = 'PROCUREMENT_RECORD' AND l.status = 'DECIDIDO';

-- Conferência de empresa (migração 000402) nos vínculos novos: lote, decisão,
-- resultados e também as tabelas de amostragem da 000417.
SELECT instalar_conferencia_de_empresa();
