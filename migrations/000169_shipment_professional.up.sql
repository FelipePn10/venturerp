BEGIN;

-- Romaneio profissional (padrão SAP Outbound Delivery / romaneio de carga BR):
-- pesos líquido+bruto, cubagem, dados da viagem (placa/motorista/ANTT/frete/
-- seguro/lacres/previsão), vínculo com a NF-e (saída fiscal) e auditoria.
ALTER TABLE shipments
    ADD COLUMN IF NOT EXISTS total_net_weight   NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_gross_weight NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS total_cubage_m3    NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS freight_modality   VARCHAR(10),            -- CIF | FOB | TERCEIROS | SEM_FRETE
    ADD COLUMN IF NOT EXISTS freight_value      NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS insurance_value    NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS vehicle_plate      VARCHAR(10),
    ADD COLUMN IF NOT EXISTS driver_name        VARCHAR(120),
    ADD COLUMN IF NOT EXISTS driver_document    VARCHAR(20),
    ADD COLUMN IF NOT EXISTS antt_code          VARCHAR(20),
    ADD COLUMN IF NOT EXISTS seals              VARCHAR(200),           -- lacres, separados por vírgula
    ADD COLUMN IF NOT EXISTS estimated_delivery DATE,
    ADD COLUMN IF NOT EXISTS fiscal_exit_id     BIGINT REFERENCES public.fiscal_exits(id),
    ADD COLUMN IF NOT EXISTS nfe_number         BIGINT,                 -- denormalizado p/ o documento
    ADD COLUMN IF NOT EXISTS nfe_key            VARCHAR(44),
    ADD COLUMN IF NOT EXISTS separated_at       TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS conferred_at       TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS cancelled_at       TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS updated_by         UUID;

-- Pesos unitários do item (para recalcular líquido/bruto a partir das quantidades).
ALTER TABLE shipment_items
    ADD COLUMN IF NOT EXISTS unit_net_weight   NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS unit_gross_weight NUMERIC(15,4) NOT NULL DEFAULT 0;

-- Volumes / Handling Units: cada embalagem física do carregamento.
CREATE TABLE IF NOT EXISTS shipment_volumes (
    id            BIGSERIAL PRIMARY KEY,
    shipment_id   BIGINT       NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    volume_number INT          NOT NULL DEFAULT 0,
    package_type  VARCHAR(20)  NOT NULL DEFAULT 'CAIXA', -- CAIXA|PALLET|FARDO|ENGRADADO|BOBINA|SACO|TAMBOR|AMARRADO
    net_weight    NUMERIC(15,4) NOT NULL DEFAULT 0,
    gross_weight  NUMERIC(15,4) NOT NULL DEFAULT 0,
    length_cm     NUMERIC(10,2) NOT NULL DEFAULT 0,
    width_cm      NUMERIC(10,2) NOT NULL DEFAULT 0,
    height_cm     NUMERIC(10,2) NOT NULL DEFAULT 0,
    cubage_m3     NUMERIC(15,4) NOT NULL DEFAULT 0,
    marking       VARCHAR(120),  -- marca / contramarca / identificação
    contents      TEXT,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_shipment_volumes_shipment ON shipment_volumes(shipment_id);

-- Trilha de auditoria das transições de status.
CREATE TABLE IF NOT EXISTS shipment_events (
    id          BIGSERIAL PRIMARY KEY,
    shipment_id BIGINT       NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    event       VARCHAR(20)  NOT NULL, -- CREATED|SEPARATED|CONFERRED|SHIPPED|CANCELLED|REOPENED|TRANSPORT|NFE_LINKED
    note        TEXT,
    created_by  UUID,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_shipment_events_shipment ON shipment_events(shipment_id);

CREATE INDEX IF NOT EXISTS idx_shipments_status        ON shipments(status);
CREATE INDEX IF NOT EXISTS idx_shipments_carrier       ON shipments(carrier_code);
CREATE INDEX IF NOT EXISTS idx_shipments_fiscal_exit   ON shipments(fiscal_exit_id);

COMMIT;
