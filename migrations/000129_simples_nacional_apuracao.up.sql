-- Apuração do Simples Nacional
CREATE TYPE simples_nacional_annex_enum AS ENUM ('I', 'II', 'III', 'IV', 'V', 'VI');

CREATE TABLE simples_nacional_apuracoes (
    id                      BIGSERIAL PRIMARY KEY,
    period                  VARCHAR(7) NOT NULL,   -- YYYY-MM
    annex                   simples_nacional_annex_enum NOT NULL,
    receita_interna         NUMERIC(15,2) NOT NULL DEFAULT 0,
    receita_externa         NUMERIC(15,2) NOT NULL DEFAULT 0,
    folha_pagamento         NUMERIC(15,2) NOT NULL DEFAULT 0,
    receita_bruta_12m       NUMERIC(15,2) NOT NULL DEFAULT 0,
    simples_recolhido       NUMERIC(15,2) NOT NULL DEFAULT 0,
    aliquota_nominal        NUMERIC(7,4) NOT NULL DEFAULT 0,
    aliquota_efetiva        NUMERIC(7,4) NOT NULL DEFAULT 0,
    aliquota_efetiva_icms   NUMERIC(7,4) NOT NULL DEFAULT 0,
    parcela_deduzir         NUMERIC(15,2) NOT NULL DEFAULT 0,
    observation             TEXT,
    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (period, annex)
);
