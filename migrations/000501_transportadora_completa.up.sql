-- Cadastro de transportadora completo (VSUP0140).
--
-- O que faltava, comparado com os cadastros de transportadora de outros ERPs
-- (Senior F073TRA/F073VEI, TOTVS GFE, SAP TM, Oracle Shipping) e com o que o
-- MDF-e e o CT-e pedem do veículo:
--   * veículo: UF da placa, RENAVAM, tara, tipo de rodado e de carroceria (os
--     códigos do MDF-e), tipo do proprietário e RNTRC do proprietário;
--   * motorista: CPF, categoria e validade da CNH — CNH vencida é multa e carga
--     parada na fiscalização;
--   * capa: fator de cubagem (frete cobrado pelo maior entre peso real e peso
--     cubado — o que pesa em móveis e chapas leves e volumosas), taxa de
--     despacho por embarque e se aceita produto perigoso;
--   * cadastros de apoio por empresa: tipos de veículo e tipos de ocorrência
--     (antes texto livre e lista fixa na tela).
--
-- Convenção de empresa: `enterprise_id`, igual ao pai (`shipping_carriers`).

ALTER TABLE shipping_carriers
    ADD COLUMN cubage_factor           NUMERIC(10,3) NOT NULL DEFAULT 0,
    ADD COLUMN dispatch_fee            NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN accepts_dangerous_goods BOOLEAN NOT NULL DEFAULT FALSE,
    ADD CONSTRAINT shipping_carriers_cubagem_check CHECK (cubage_factor >= 0 AND dispatch_fee >= 0);

COMMENT ON COLUMN shipping_carriers.cubage_factor IS 'kg por m3 do peso cubado (rodoviario costuma usar 300). Zero = cobra so o peso real.';
COMMENT ON COLUMN shipping_carriers.dispatch_fee IS 'Taxa fixa por embarque (despacho/emissao do CT-e), somada ao frete.';

-- Tipos de veículo: o que o usuário escolhia digitando (TRUCK, CARRETA…) vira
-- cadastro com os códigos que o MDF-e exige e a capacidade típica.
CREATE TABLE IF NOT EXISTS shipping_vehicle_types (
    id            BIGSERIAL PRIMARY KEY,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id),
    code          VARCHAR(20) NOT NULL,
    description   VARCHAR(120) NOT NULL,
    -- MDF-e tpRod: 01 truck, 02 toco, 03 cavalo mecânico, 04 VAN, 05 utilitário, 06 outros.
    wheel_type    CHAR(2),
    -- MDF-e tpCar: 00 não aplicável, 01 aberta, 02 fechada/baú, 03 granelera, 04 porta-contêiner, 05 sider.
    body_type     CHAR(2),
    axles         SMALLINT,
    capacity_kg   NUMERIC(15,3) NOT NULL DEFAULT 0,
    capacity_m3   NUMERIC(15,3) NOT NULL DEFAULT 0,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT shipping_vehicle_types_unico UNIQUE (enterprise_id, code),
    CONSTRAINT shipping_vehicle_types_mdfe_check CHECK (
        (wheel_type IS NULL OR wheel_type IN ('01','02','03','04','05','06')) AND
        (body_type IS NULL OR body_type IN ('00','01','02','03','04','05')) AND
        (axles IS NULL OR axles BETWEEN 2 AND 12) AND capacity_kg >= 0 AND capacity_m3 >= 0
    )
);
CREATE INDEX IF NOT EXISTS idx_shipping_vehicle_types_empresa ON shipping_vehicle_types (enterprise_id);

ALTER TABLE shipping_carrier_vehicles
    ADD COLUMN vehicle_type_id     BIGINT REFERENCES shipping_vehicle_types(id),
    ADD COLUMN plate_state         CHAR(2),
    ADD COLUMN renavam             VARCHAR(11),
    ADD COLUMN tare_kg             NUMERIC(15,3) NOT NULL DEFAULT 0,
    ADD COLUMN wheel_type          CHAR(2),
    ADD COLUMN body_type           CHAR(2),
    -- MDF-e tpProp: 0 TAC agregado, 1 TAC independente, 2 outros. Vazio = frota própria da transportadora.
    ADD COLUMN owner_type          CHAR(1),
    ADD COLUMN driver_cnh_category VARCHAR(3),
    ADD COLUMN driver_cnh_expiry   DATE,
    ADD CONSTRAINT shipping_carrier_vehicles_mdfe_check CHECK (
        (wheel_type IS NULL OR wheel_type IN ('01','02','03','04','05','06')) AND
        (body_type IS NULL OR body_type IN ('00','01','02','03','04','05')) AND
        (owner_type IS NULL OR owner_type IN ('0','1','2')) AND tare_kg >= 0
    );
CREATE INDEX IF NOT EXISTS idx_shipping_carrier_vehicles_tipo ON shipping_carrier_vehicles (vehicle_type_id);

-- Tipos de ocorrência: a lista fixa da tela vira cadastro. `counts_as_failure`
-- diz se a ocorrência pesa contra a transportadora na avaliação (avaria e
-- extravio pesam; reentrega a pedido do cliente não).
CREATE TABLE IF NOT EXISTS shipping_occurrence_types (
    id                BIGSERIAL PRIMARY KEY,
    enterprise_id     BIGINT NOT NULL REFERENCES enterprise(id),
    code              VARCHAR(40) NOT NULL,
    description       VARCHAR(120) NOT NULL,
    counts_as_failure BOOLEAN NOT NULL DEFAULT TRUE,
    is_active         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT shipping_occurrence_types_unico UNIQUE (enterprise_id, code)
);
CREATE INDEX IF NOT EXISTS idx_shipping_occurrence_types_empresa ON shipping_occurrence_types (enterprise_id);

-- A lista que a tela tinha fixa, para cada empresa, mais qualquer código já
-- gravado em ocorrência (nada fica órfão).
INSERT INTO shipping_occurrence_types (enterprise_id, code, description, counts_as_failure)
SELECT e.id, v.code, v.description, v.falha
  FROM enterprise e
 CROSS JOIN (VALUES
    ('ATRASO', 'Atraso na entrega', TRUE),
    ('AVARIA', 'Avaria da mercadoria', TRUE),
    ('EXTRAVIO', 'Extravio', TRUE),
    ('ENTREGA_PARCIAL', 'Entrega parcial', TRUE),
    ('DEVOLUCAO', 'Devolução', TRUE),
    ('COBRANCA_INDEVIDA', 'Cobrança indevida de frete', TRUE),
    ('RECUSA_DESTINATARIO', 'Recusa do destinatário', FALSE),
    ('REENTREGA', 'Reentrega a pedido do cliente', FALSE),
    ('ELOGIO', 'Elogio', FALSE)
 ) AS v(code, description, falha)
ON CONFLICT DO NOTHING;

INSERT INTO shipping_occurrence_types (enterprise_id, code, description)
SELECT DISTINCT o.enterprise_id, o.occurrence_type, initcap(replace(o.occurrence_type, '_', ' '))
  FROM shipping_carrier_occurrences o
ON CONFLICT DO NOTHING;

-- Ocorrência ligada também ao romaneio (expedição), não só ao pedido.
ALTER TABLE shipping_carrier_occurrences ADD COLUMN shipment_id BIGINT REFERENCES shipments(id);

-- A ocorrência era apagada junto com a transportadora (ON DELETE CASCADE): a
-- exclusão de um cadastro levava o histórico de avaliação. Agora a
-- transportadora com histórico não se apaga — inativa-se.
ALTER TABLE shipping_carrier_occurrences DROP CONSTRAINT IF EXISTS shipping_carrier_occurrences_carrier_id_fkey;
ALTER TABLE shipping_carrier_occurrences
    ADD CONSTRAINT shipping_carrier_occurrences_carrier_id_fkey FOREIGN KEY (carrier_id) REFERENCES shipping_carriers(id);

SELECT instalar_conferencia_de_empresa();
