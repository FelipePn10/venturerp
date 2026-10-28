-- Certificado de qualidade ao cliente e controle de documentos da qualidade.

-- Certificado: o conteúdo (medições resumidas, decisão, corrida) é congelado na
-- emissão — o documento entregue ao cliente não muda se o lote for revisto.
CREATE TABLE quality_certificates (
    id               BIGSERIAL     PRIMARY KEY,
    enterprise_id    BIGINT        NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    numero           BIGINT        NOT NULL,
    customer_code    BIGINT,
    fiscal_exit_id   BIGINT,
    sales_order_code BIGINT,
    item_code        BIGINT        NOT NULL,
    mask             TEXT          NOT NULL DEFAULT '',
    lote             TEXT,
    quantidade       NUMERIC(18,6) NOT NULL CHECK (quantidade > 0),
    corrida          TEXT,
    certificado_usina TEXT,
    lotes_inspecao   BIGINT[]      NOT NULL,
    conteudo         JSONB         NOT NULL,
    observacoes      TEXT,
    status           VARCHAR(10)   NOT NULL DEFAULT 'EMITIDO' CHECK (status IN ('EMITIDO', 'CANCELADO')),
    motivo_cancelamento TEXT,
    emitido_em       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    emitido_por      UUID          REFERENCES users(id),
    cancelado_em     TIMESTAMPTZ,
    cancelado_por    UUID          REFERENCES users(id),
    UNIQUE (enterprise_id, numero),
    UNIQUE (enterprise_id, id)
);
CREATE INDEX idx_quality_certificates_item ON quality_certificates(enterprise_id, item_code, emitido_em DESC);

-- Só o cancelamento muda o certificado emitido.
CREATE FUNCTION quality_certificates_congelado() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.status = 'CANCELADO' THEN
        RAISE EXCEPTION 'certificado cancelado não muda';
    END IF;
    IF NEW.status <> 'CANCELADO' OR NEW.conteudo IS DISTINCT FROM OLD.conteudo OR NEW.numero <> OLD.numero
       OR NEW.item_code <> OLD.item_code OR NEW.quantidade <> OLD.quantidade OR NEW.lote IS DISTINCT FROM OLD.lote
       OR NEW.customer_code IS DISTINCT FROM OLD.customer_code OR NEW.fiscal_exit_id IS DISTINCT FROM OLD.fiscal_exit_id THEN
        RAISE EXCEPTION 'o certificado emitido não pode ser alterado: cancele e emita outro';
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER trg_quality_certificates_congelado BEFORE UPDATE ON quality_certificates
    FOR EACH ROW EXECUTE FUNCTION quality_certificates_congelado();

-- Documentos controlados.
CREATE TABLE quality_documents (
    id                 BIGSERIAL   PRIMARY KEY,
    enterprise_id      BIGINT      NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    codigo             VARCHAR(30) NOT NULL,
    titulo             TEXT        NOT NULL,
    tipo               VARCHAR(16) NOT NULL CHECK (tipo IN ('IT', 'PROCEDIMENTO', 'FORMULARIO', 'PLANO_INSPECAO', 'ESPECIFICACAO', 'OUTRO')),
    item_code          BIGINT,
    route_operation_id BIGINT,
    area               TEXT,
    ativo              BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by         UUID        REFERENCES users(id),
    UNIQUE (enterprise_id, codigo),
    UNIQUE (enterprise_id, id)
);
CREATE INDEX idx_quality_documents_item ON quality_documents(enterprise_id, item_code) WHERE item_code IS NOT NULL;

CREATE TABLE quality_document_revisions (
    id                BIGSERIAL   PRIMARY KEY,
    document_id       BIGINT      NOT NULL,
    enterprise_id     BIGINT      NOT NULL,
    revisao           INTEGER     NOT NULL CHECK (revisao >= 0),
    status            VARCHAR(10) NOT NULL DEFAULT 'RASCUNHO' CHECK (status IN ('RASCUNHO', 'VIGENTE', 'OBSOLETA')),
    descricao_mudanca TEXT        NOT NULL,
    conteudo          TEXT,
    arquivo_nome      TEXT,
    arquivo_tipo      TEXT,
    arquivo           BYTEA,
    elaborado_por     UUID        NOT NULL REFERENCES users(id),
    elaborado_em      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    aprovado_por      UUID        REFERENCES users(id),
    vigente_desde     TIMESTAMPTZ,
    obsoleta_em       TIMESTAMPTZ,
    UNIQUE (document_id, revisao),
    FOREIGN KEY (enterprise_id, document_id) REFERENCES quality_documents(enterprise_id, id) ON DELETE CASCADE,
    CHECK (conteudo IS NOT NULL OR arquivo IS NOT NULL),
    CHECK (status = 'RASCUNHO' OR (aprovado_por IS NOT NULL AND vigente_desde IS NOT NULL)),
    CHECK (aprovado_por IS NULL OR aprovado_por <> elaborado_por)
);
-- Um rascunho e uma vigente por documento.
CREATE UNIQUE INDEX uq_quality_doc_rascunho ON quality_document_revisions(document_id) WHERE status = 'RASCUNHO';
CREATE UNIQUE INDEX uq_quality_doc_vigente ON quality_document_revisions(document_id) WHERE status = 'VIGENTE';

-- Revisão aprovada não muda de conteúdo; só VIGENTE → OBSOLETA.
CREATE FUNCTION quality_document_revisions_congeladas() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.status = 'RASCUNHO' THEN
        RETURN NEW;
    END IF;
    IF OLD.status = 'VIGENTE' AND NEW.status = 'OBSOLETA'
       AND NEW.conteudo IS NOT DISTINCT FROM OLD.conteudo AND NEW.arquivo IS NOT DISTINCT FROM OLD.arquivo
       AND NEW.descricao_mudanca = OLD.descricao_mudanca AND NEW.revisao = OLD.revisao THEN
        RETURN NEW;
    END IF;
    RAISE EXCEPTION 'revisão aprovada não pode ser alterada: crie uma nova revisão';
END $$;
CREATE TRIGGER trg_quality_document_revisions_congeladas BEFORE UPDATE ON quality_document_revisions
    FOR EACH ROW EXECUTE FUNCTION quality_document_revisions_congeladas();

SELECT instalar_conferencia_de_empresa();
