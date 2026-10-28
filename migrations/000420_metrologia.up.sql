-- Metrologia: instrumentos de medição, calibrações e o bloqueio da medição com
-- instrumento vencido, reprovado ou de outro grupo.
--
-- O roteiro de inspeção já pedia um "grupo de instrumento" na etapa
-- (instrument_group) e o lote copia esse grupo para a característica, mas nada
-- conferia o instrumento usado: medir com paquímetro vencido passava igual.

CREATE TABLE quality_instruments (
    id                BIGSERIAL     PRIMARY KEY,
    enterprise_id     BIGINT        NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    codigo            VARCHAR(30)   NOT NULL,
    descricao         TEXT          NOT NULL,
    -- Grupo comparado com o instrumento da característica (ex.: PAQUIMETRO).
    grupo             VARCHAR(60)   NOT NULL,
    fabricante        TEXT,
    numero_serie      TEXT,
    faixa             TEXT,
    resolucao         NUMERIC(18,6) CHECK (resolucao > 0),
    -- Erro máximo admissível: critério de aceitação da calibração.
    ema               NUMERIC(18,6) CHECK (ema > 0),
    unidade           TEXT,
    localizacao       TEXT,
    frequencia_dias   INTEGER       NOT NULL CHECK (frequencia_dias > 0),
    status            VARCHAR(10)   NOT NULL DEFAULT 'ATIVO' CHECK (status IN ('ATIVO', 'BLOQUEADO', 'BAIXADO')),
    motivo_status     TEXT,
    ultima_calibracao DATE,
    proxima_calibracao DATE,
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_by        UUID          REFERENCES users(id),
    UNIQUE (enterprise_id, codigo),
    UNIQUE (enterprise_id, id)
);
CREATE INDEX idx_quality_instruments_vencimento ON quality_instruments(enterprise_id, proxima_calibracao) WHERE status = 'ATIVO';

CREATE TABLE quality_calibrations (
    id               BIGSERIAL     PRIMARY KEY,
    enterprise_id    BIGINT        NOT NULL,
    instrument_id    BIGINT        NOT NULL,
    data             DATE          NOT NULL,
    executor         VARCHAR(8)    NOT NULL CHECK (executor IN ('INTERNO', 'EXTERNO')),
    laboratorio      TEXT,
    certificado      TEXT,
    erro_encontrado  NUMERIC(18,6),
    incerteza        NUMERIC(18,6),
    ema              NUMERIC(18,6),
    resultado        VARCHAR(10)   NOT NULL CHECK (resultado IN ('APROVADO', 'REPROVADO')),
    ajustado         BOOLEAN       NOT NULL DEFAULT FALSE,
    proxima          DATE,
    observacoes      TEXT,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_by       UUID          REFERENCES users(id),
    FOREIGN KEY (enterprise_id, instrument_id) REFERENCES quality_instruments(enterprise_id, id) ON DELETE CASCADE
);
CREATE INDEX idx_quality_calibrations_instr ON quality_calibrations(instrument_id, data DESC);

CREATE FUNCTION quality_calibrations_imutaveis() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'a calibração registrada não pode ser alterada: registre uma nova';
END $$;
CREATE TRIGGER trg_quality_calibrations_imutaveis BEFORE UPDATE ON quality_calibrations
    FOR EACH ROW EXECUTE FUNCTION quality_calibrations_imutaveis();

-- O resultado do lote guarda o instrumento usado: base da análise de impacto
-- quando a calibração reprova.
ALTER TABLE quality_lot_results ADD COLUMN instrument_id BIGINT REFERENCES quality_instruments(id);
CREATE INDEX idx_quality_lot_results_instr ON quality_lot_results(instrument_id, created_at) WHERE instrument_id IS NOT NULL;

SELECT instalar_conferencia_de_empresa();
