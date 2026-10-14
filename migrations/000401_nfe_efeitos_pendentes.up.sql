-- Efeitos de uma NF-e já autorizada (ou cancelada) na SEFAZ que ainda não
-- foram aplicados no ERP: baixa de estoque, reservas, títulos a receber e
-- situação do pedido. A nota não pode ser desfeita, então os efeitos ficam
-- pendentes, visíveis e reprocessáveis (automático e manual) até entrarem.
CREATE TABLE fiscal_exit_pendencias (
    id             BIGSERIAL PRIMARY KEY,
    enterprise_id  BIGINT      NOT NULL REFERENCES enterprise(id),
    fiscal_exit_id BIGINT      NOT NULL REFERENCES fiscal_exits(id),
    etapa          TEXT        NOT NULL CHECK (etapa IN ('AUTORIZACAO', 'CANCELAMENTO')),
    situacao       TEXT        NOT NULL DEFAULT 'PENDENTE' CHECK (situacao IN ('PENDENTE', 'RESOLVIDA')),
    tentativas     INT         NOT NULL DEFAULT 0,
    ultimo_erro    TEXT,
    proxima_tentativa TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    criado_por     UUID        REFERENCES users(id),
    criado_em      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolvido_em   TIMESTAMPTZ
);
-- Uma pendência aberta por nota e etapa.
CREATE UNIQUE INDEX uq_fiscal_exit_pendencia_aberta
    ON fiscal_exit_pendencias (fiscal_exit_id, etapa) WHERE situacao = 'PENDENTE';
CREATE INDEX idx_fiscal_exit_pendencias_fila
    ON fiscal_exit_pendencias (proxima_tentativa) WHERE situacao = 'PENDENTE';
CREATE INDEX idx_fiscal_exit_pendencias_empresa
    ON fiscal_exit_pendencias (enterprise_id, fiscal_exit_id);
