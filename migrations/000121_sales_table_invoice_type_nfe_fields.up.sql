-- Sales table composition (for export orders)
CREATE TYPE table_composition_enum AS ENUM ('EXWORK', 'CIF', 'FOB');

-- Sales table type (normal vs promotional)
CREATE TYPE table_type_enum AS ENUM ('NORMAL', 'PROMOCIONAL');

-- Base date for price calculation on Custo Médio tables
CREATE TYPE base_date_enum AS ENUM ('PEDIDO', 'DATA_ATUAL');

-- ─── SalesTable new columns ───────────────────────────────────────────────────
ALTER TABLE sales_tables
    ADD COLUMN composition                  table_composition_enum  NOT NULL DEFAULT 'FOB',
    ADD COLUMN table_type                   table_type_enum         NOT NULL DEFAULT 'NORMAL',
    ADD COLUMN base_date                    base_date_enum          NOT NULL DEFAULT 'PEDIDO',
    ADD COLUMN allow_items_below_cent       BOOLEAN                 NOT NULL DEFAULT FALSE,
    ADD COLUMN icms_interestadual_por_dentro BOOLEAN                NOT NULL DEFAULT FALSE,
    ADD COLUMN observation                  TEXT;

-- ─── InvoiceType NF-e fields (FocusNFE integration) ──────────────────────────
-- CST codes are SEFAZ-defined strings, not enums
ALTER TABLE invoice_types
    -- NF-e model: '55' = NF-e, '65' = NFC-e
    ADD COLUMN model_nf                 VARCHAR(2),
    -- CST/CSOSN codes per tax (sent directly in NF-e XML via FocusNFE)
    ADD COLUMN cst_icms                 VARCHAR(3),
    ADD COLUMN csosn_icms               VARCHAR(4),
    ADD COLUMN cst_ipi                  VARCHAR(2),
    ADD COLUMN cst_pis                  VARCHAR(2),
    ADD COLUMN cst_cofins               VARCHAR(2),
    -- Operational flags
    ADD COLUMN baixa_pedido             BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN gera_titulo_dev          BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN exige_suframa            BOOLEAN NOT NULL DEFAULT FALSE,
    -- Lucro Presumido presumption percentages
    ADD COLUMN ir_pct_presumption       NUMERIC(5,2) NOT NULL DEFAULT 0,
    ADD COLUMN csll_pct_presumption     NUMERIC(5,2) NOT NULL DEFAULT 0;
