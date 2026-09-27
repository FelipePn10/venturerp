-- Cadastro de transportadora.
--
-- Hoje a "transportadora" é só um fornecedor com `freight_type`, e a tabela
-- `carriers` deste banco é outra coisa: é o PORTADOR financeiro (carteira,
-- limite de crédito, dias de recebimento). Não havia onde guardar RNTRC, modal,
-- tabela de frete, veículo, motorista, seguro nem prazo por região — o que o
-- FoccoERP tem no cadastro de transportadora, o TOTVS em GU6/GU7 (transportadora
-- e veículo) e o SAP no parceiro "transportador" com Shipment Costs.
--
-- O desenho aqui é: a transportadora CONTINUA sendo o fornecedor (para pagar o
-- frete pelo contas a pagar de sempre) e ganha um perfil de transporte ao lado.
-- Nenhum cadastro existente muda de significado.
--
-- Convenção de empresa: `enterprise_id`, igual ao pai (`suppliers`).

CREATE TYPE carrier_modal_enum AS ENUM ('RODOVIARIO', 'AEREO', 'MARITIMO', 'FERROVIARIO', 'DUTOVIARIO', 'MULTIMODAL');

-- Categorias da ANTT: ETC (empresa), CTC (cooperativa), TAC (autônomo).
CREATE TYPE carrier_shipper_type_enum AS ENUM ('ETC', 'CTC', 'TAC');

CREATE TABLE IF NOT EXISTS shipping_carriers (
    id                   BIGSERIAL PRIMARY KEY,
    enterprise_id        BIGINT NOT NULL REFERENCES enterprise(id),
    supplier_code        BIGINT NOT NULL REFERENCES suppliers(code),
    -- Registro Nacional de Transportadores Rodoviários de Carga: 8 dígitos. Sem
    -- RNTRC válido o CT-e é rejeitado, então a validade fica junto.
    antt_rntrc           VARCHAR(20),
    antt_expiry          DATE,
    shipper_type         carrier_shipper_type_enum,
    modal                carrier_modal_enum NOT NULL DEFAULT 'RODOVIARIO',
    issues_cte           BOOLEAN NOT NULL DEFAULT TRUE,
    -- Tabela de frete padrão. Cada componente existe porque é cobrado
    -- separadamente na praça: peso, ad valorem sobre a mercadoria, GRIS
    -- (gerenciamento de risco), pedágio por 100 kg e um piso.
    default_freight_type VARCHAR(10),
    freight_min_value    NUMERIC(15,2) NOT NULL DEFAULT 0,
    freight_kg_rate      NUMERIC(15,6) NOT NULL DEFAULT 0,
    freight_pct_value    NUMERIC(9,4)  NOT NULL DEFAULT 0,
    gris_pct             NUMERIC(9,4)  NOT NULL DEFAULT 0,
    toll_per_100kg       NUMERIC(15,2) NOT NULL DEFAULT 0,
    insurance_company    VARCHAR(120),
    insurance_policy     VARCHAR(60),
    insurance_expiry     DATE,
    insurance_coverage   NUMERIC(15,2) NOT NULL DEFAULT 0,
    average_lead_days    SMALLINT,
    tracking_url         VARCHAR(255),
    contact_name         VARCHAR(120),
    contact_phone        VARCHAR(40),
    contact_email        VARCHAR(120),
    notes                TEXT,
    is_active            BOOLEAN NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT shipping_carriers_unico UNIQUE (enterprise_id, supplier_code),
    CONSTRAINT shipping_carriers_percentuais_check CHECK (
        freight_pct_value >= 0 AND freight_pct_value <= 100 AND
        gris_pct >= 0 AND gris_pct <= 100
    ),
    CONSTRAINT shipping_carriers_valores_check CHECK (
        freight_min_value >= 0 AND freight_kg_rate >= 0 AND
        toll_per_100kg >= 0 AND insurance_coverage >= 0 AND
        (average_lead_days IS NULL OR average_lead_days >= 0)
    )
);

CREATE INDEX IF NOT EXISTS idx_shipping_carriers_empresa ON shipping_carriers (enterprise_id);
CREATE INDEX IF NOT EXISTS idx_shipping_carriers_fornecedor ON shipping_carriers (supplier_code);

-- Veículo e motorista. O CT-e e o MDF-e pedem placa, e a capacidade é o que
-- permite dizer se a carga cabe antes de fechar a expedição.
CREATE TABLE IF NOT EXISTS shipping_carrier_vehicles (
    id             BIGSERIAL PRIMARY KEY,
    enterprise_id  BIGINT NOT NULL REFERENCES enterprise(id),
    carrier_id     BIGINT NOT NULL REFERENCES shipping_carriers(id) ON DELETE CASCADE,
    plate          VARCHAR(10) NOT NULL,
    description    VARCHAR(120),
    vehicle_type   VARCHAR(40),
    axles          SMALLINT,
    capacity_kg    NUMERIC(15,3) NOT NULL DEFAULT 0,
    capacity_m3    NUMERIC(15,3) NOT NULL DEFAULT 0,
    antt_owner     VARCHAR(20),
    driver_name    VARCHAR(120),
    driver_document VARCHAR(20),
    driver_license VARCHAR(20),
    is_active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT shipping_carrier_vehicles_unico UNIQUE (carrier_id, plate),
    CONSTRAINT shipping_carrier_vehicles_capacidade_check CHECK (
        capacity_kg >= 0 AND capacity_m3 >= 0 AND (axles IS NULL OR axles BETWEEN 2 AND 12)
    )
);

CREATE INDEX IF NOT EXISTS idx_shipping_carrier_vehicles_empresa ON shipping_carrier_vehicles (enterprise_id);
CREATE INDEX IF NOT EXISTS idx_shipping_carrier_vehicles_transportadora ON shipping_carrier_vehicles (carrier_id);

-- Região atendida: é o que responde "esta transportadora entrega neste CEP, em
-- quantos dias e por quanto". Sem isso o prazo do pedido é chute.
CREATE TABLE IF NOT EXISTS shipping_carrier_service_areas (
    id               BIGSERIAL PRIMARY KEY,
    enterprise_id    BIGINT NOT NULL REFERENCES enterprise(id),
    carrier_id       BIGINT NOT NULL REFERENCES shipping_carriers(id) ON DELETE CASCADE,
    state            CHAR(2),
    city             VARCHAR(120),
    postal_code_from VARCHAR(9),
    postal_code_to   VARCHAR(9),
    lead_days        SMALLINT NOT NULL DEFAULT 0,
    min_value        NUMERIC(15,2) NOT NULL DEFAULT 0,
    kg_rate          NUMERIC(15,6) NOT NULL DEFAULT 0,
    pct_value        NUMERIC(9,4)  NOT NULL DEFAULT 0,
    is_active        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT shipping_carrier_service_areas_check CHECK (
        lead_days >= 0 AND min_value >= 0 AND kg_rate >= 0 AND
        pct_value >= 0 AND pct_value <= 100 AND
        (state IS NOT NULL OR postal_code_from IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_shipping_carrier_areas_empresa ON shipping_carrier_service_areas (enterprise_id);
CREATE INDEX IF NOT EXISTS idx_shipping_carrier_areas_transportadora ON shipping_carrier_service_areas (carrier_id);
CREATE INDEX IF NOT EXISTS idx_shipping_carrier_areas_uf ON shipping_carrier_service_areas (state);

-- Ocorrência de entrega. É a memória que transforma "essa transportadora atrasa"
-- em número: é o que o FoccoERP chama de ocorrência de transporte e o que
-- alimenta a avaliação do fornecedor de frete.
CREATE TABLE IF NOT EXISTS shipping_carrier_occurrences (
    id               BIGSERIAL PRIMARY KEY,
    enterprise_id    BIGINT NOT NULL REFERENCES enterprise(id),
    carrier_id       BIGINT NOT NULL REFERENCES shipping_carriers(id) ON DELETE CASCADE,
    occurrence_date  DATE NOT NULL DEFAULT CURRENT_DATE,
    occurrence_type  VARCHAR(40) NOT NULL,
    sales_order_code BIGINT,
    delay_days       SMALLINT NOT NULL DEFAULT 0,
    cost_impact      NUMERIC(15,2) NOT NULL DEFAULT 0,
    description      TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by       UUID,
    CONSTRAINT shipping_carrier_occurrences_check CHECK (delay_days >= 0)
);

CREATE INDEX IF NOT EXISTS idx_shipping_carrier_ocorr_empresa ON shipping_carrier_occurrences (enterprise_id);
CREATE INDEX IF NOT EXISTS idx_shipping_carrier_ocorr_transportadora ON shipping_carrier_occurrences (carrier_id, occurrence_date DESC);

COMMENT ON TABLE shipping_carriers IS 'Perfil de transporte do fornecedor transportadora (RNTRC, modal, tabela de frete, seguro). Nao confundir com public.carriers, que e o portador financeiro.';
COMMENT ON TABLE shipping_carrier_vehicles IS 'Frota e motoristas da transportadora (placa, capacidade, ANTT do proprietario).';
COMMENT ON TABLE shipping_carrier_service_areas IS 'Regioes atendidas com prazo e tabela propria; responde prazo e custo por UF/CEP.';
COMMENT ON TABLE shipping_carrier_occurrences IS 'Ocorrencias de entrega usadas para avaliar a transportadora.';
