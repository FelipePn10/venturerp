-- Tanques / linhas de capacidade da promessa de entrega. O item apontava para um
-- número solto (planning_tank_code) e a capacidade era digitada na consulta, a
-- mesma para todos. Agora cada linha tem nome e capacidade diária própria.
CREATE TABLE delivery_capacity_lines (
    id              BIGSERIAL PRIMARY KEY,
    enterprise_id   BIGINT NOT NULL REFERENCES enterprise(id),
    code            BIGINT NOT NULL CHECK (code > 0),
    description     VARCHAR(120) NOT NULL,
    daily_capacity  NUMERIC(18,4) NOT NULL CHECK (daily_capacity > 0),
    uom             VARCHAR(20) NOT NULL DEFAULT 'UN',
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (enterprise_id, code)
);

-- O item que já tinha um tanque ganha a linha, para nenhum vínculo ficar órfão.
INSERT INTO delivery_capacity_lines (enterprise_id, code, description, daily_capacity)
SELECT DISTINCT enterprise_id, planning_tank_code, 'Tanque ' || planning_tank_code, 1
  FROM items WHERE planning_tank_code IS NOT NULL AND planning_tank_code > 0
ON CONFLICT DO NOTHING;

SELECT instalar_conferencia_de_empresa();
