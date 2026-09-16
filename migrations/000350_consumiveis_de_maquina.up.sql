-- Consumíveis da máquina e o consumo que depende do que está sendo produzido.
--
-- Um cilindro de oxigênio não dura "N horas": dura conforme o que a máquina está
-- cortando. Chapa de 3 mm gasta uma vazão; chapa de 12 mm gasta outra. Por isso
-- a AUTONOMIA fica na máquina (quanto rende uma carga, quanto demora a troca) e
-- a TAXA DE CONSUMO fica no par item × máscara × máquina, exatamente no mesmo
-- grão da produtividade — que é onde o sistema já sabe o que está sendo feito.
--
-- Com os dois, o planejamento conta quantas trocas a ordem vai exigir e soma o
-- tempo delas à ocupação da máquina. Sem isso a ordem "cabe" no turno no papel e
-- estoura na prática, que é o erro clássico de quem planeja corte a laser só
-- pelo tempo de máquina.

CREATE TABLE machine_consumables (
    id                  BIGSERIAL PRIMARY KEY,
    enterprise_id       BIGINT NOT NULL REFERENCES enterprise(id),
    machine_code        BIGINT NOT NULL REFERENCES machines(code) ON DELETE CASCADE,
    code                TEXT NOT NULL,
    description         TEXT NOT NULL,
    unit                TEXT NOT NULL,
    -- Quanto rende UMA carga completa, na unidade acima (ex.: 200 m³ por cilindro).
    capacity_per_refill NUMERIC(18,6) NOT NULL CHECK (capacity_per_refill > 0),
    -- Quanto a máquina fica parada para trocar a carga.
    replacement_minutes NUMERIC(10,2) NOT NULL DEFAULT 0 CHECK (replacement_minutes >= 0),
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, machine_code, code),
    -- Redundante como chave, mas é o que permite a chave estrangeira composta
    -- lá embaixo: sem ela o Postgres não aceita referenciar (id, machine_code).
    UNIQUE (id, machine_code)
);

CREATE INDEX machine_consumables_tenant ON machine_consumables (enterprise_id, machine_code);

COMMENT ON COLUMN machine_consumables.capacity_per_refill IS
    'Quanto rende uma carga completa, na unidade do consumível.';
COMMENT ON COLUMN machine_consumables.replacement_minutes IS
    'Minutos de máquina parada para trocar a carga. Entra na ocupação do planejamento.';

ALTER TABLE item_machine_times
    ADD COLUMN consumable_id        BIGINT,
    ADD COLUMN consumption_per_hour NUMERIC(18,6);

-- A taxa vale por hora de USINAGEM — preparação não corta, então não consome.
ALTER TABLE item_machine_times
    ADD CONSTRAINT item_machine_times_consumption_check
        CHECK (consumption_per_hour IS NULL OR consumption_per_hour > 0);

-- Os dois campos andam juntos: taxa sem consumível não diz o que se gasta, e
-- consumível sem taxa não diz quanto. Meio preenchido seria um dado que o
-- planejamento ignora em silêncio.
ALTER TABLE item_machine_times
    ADD CONSTRAINT item_machine_times_consumption_pair
        CHECK ((consumable_id IS NULL) = (consumption_per_hour IS NULL));

-- Chave composta, e não simples: apontar o consumo de um item para o consumível
-- de OUTRA máquina passa a ser impossível no banco, em vez de depender de o
-- código lembrar de conferir.
ALTER TABLE item_machine_times
    ADD CONSTRAINT item_machine_times_consumable_fk
        FOREIGN KEY (consumable_id, machine_code)
        REFERENCES machine_consumables (id, machine_code);

COMMENT ON COLUMN item_machine_times.consumption_per_hour IS
    'Consumo por hora de usinagem deste item nesta máquina, na unidade do consumível.';
