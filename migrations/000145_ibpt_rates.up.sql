BEGIN;

-- IBPT/SCI — Tabela de carga tributária aproximada por NCM (Lei da Transparência,
-- Lei 12.741/2012). Importada do arquivo oficial IBPT (CSV) por UF/versão.
CREATE TABLE IF NOT EXISTS ibpt_rates (
    id               BIGSERIAL PRIMARY KEY,
    ncm              VARCHAR(10)  NOT NULL,
    ex               VARCHAR(3)   NOT NULL DEFAULT '0',
    uf               CHAR(2)      NOT NULL,
    tipo             SMALLINT     NOT NULL DEFAULT 0, -- 0=NCM, 1=NBS, 2=LC116
    descricao        VARCHAR(255) NOT NULL DEFAULT '',
    nacional_federal NUMERIC(7,4) NOT NULL DEFAULT 0, -- % tributos federais p/ nacional
    importado_federal NUMERIC(7,4) NOT NULL DEFAULT 0, -- % tributos federais p/ importado
    estadual         NUMERIC(7,4) NOT NULL DEFAULT 0, -- % tributos estaduais
    municipal        NUMERIC(7,4) NOT NULL DEFAULT 0, -- % tributos municipais
    vigencia_inicio  DATE,
    vigencia_fim     DATE,
    chave            VARCHAR(20),
    versao           VARCHAR(20)  NOT NULL DEFAULT '',
    fonte            VARCHAR(60)  NOT NULL DEFAULT 'IBPT',
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (ncm, ex, uf, versao)
);

CREATE INDEX IF NOT EXISTS idx_ibpt_rates_ncm_uf ON ibpt_rates (ncm, uf);

COMMIT;
