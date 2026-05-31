-- ─── Países e UFs ────────────────────────────────────────────────────────────

CREATE TABLE countries (
    id          BIGSERIAL    PRIMARY KEY,
    sigla       VARCHAR(3)   NOT NULL UNIQUE,   -- ISO 3166-1 alpha-3 (BRA, USA, ARG …)
    name        VARCHAR(100) NOT NULL,
    ddi         VARCHAR(5),                     -- international dialing code (+55, +1 …)
    bacen_code  VARCHAR(4),                     -- Banco Central do Brasil country code
    sis_comex   VARCHAR(4),                     -- SisComEx foreign trade code
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE ufs (
    id          BIGSERIAL   PRIMARY KEY,
    sigla       CHAR(2)     NOT NULL UNIQUE,    -- AC, AM, BA, RJ, SP …
    name        VARCHAR(100) NOT NULL,
    country_id  BIGINT      NOT NULL REFERENCES countries(id) ON DELETE RESTRICT,
    ibge_code   VARCHAR(10),
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed Brazil
INSERT INTO countries (sigla, name, ddi, bacen_code, sis_comex)
VALUES ('BRA', 'Brasil', '+55', '1058', '1058');

-- Seed Brazilian states (using the IBGE state codes)
INSERT INTO ufs (sigla, name, country_id, ibge_code) VALUES
    ('AC', 'Acre',               (SELECT id FROM countries WHERE sigla='BRA'), '12'),
    ('AL', 'Alagoas',            (SELECT id FROM countries WHERE sigla='BRA'), '27'),
    ('AM', 'Amazonas',           (SELECT id FROM countries WHERE sigla='BRA'), '13'),
    ('AP', 'Amapá',              (SELECT id FROM countries WHERE sigla='BRA'), '16'),
    ('BA', 'Bahia',              (SELECT id FROM countries WHERE sigla='BRA'), '29'),
    ('CE', 'Ceará',              (SELECT id FROM countries WHERE sigla='BRA'), '23'),
    ('DF', 'Distrito Federal',   (SELECT id FROM countries WHERE sigla='BRA'), '53'),
    ('ES', 'Espírito Santo',     (SELECT id FROM countries WHERE sigla='BRA'), '32'),
    ('GO', 'Goiás',              (SELECT id FROM countries WHERE sigla='BRA'), '52'),
    ('MA', 'Maranhão',           (SELECT id FROM countries WHERE sigla='BRA'), '21'),
    ('MG', 'Minas Gerais',       (SELECT id FROM countries WHERE sigla='BRA'), '31'),
    ('MS', 'Mato Grosso do Sul', (SELECT id FROM countries WHERE sigla='BRA'), '50'),
    ('MT', 'Mato Grosso',        (SELECT id FROM countries WHERE sigla='BRA'), '51'),
    ('PA', 'Pará',               (SELECT id FROM countries WHERE sigla='BRA'), '15'),
    ('PB', 'Paraíba',            (SELECT id FROM countries WHERE sigla='BRA'), '25'),
    ('PE', 'Pernambuco',         (SELECT id FROM countries WHERE sigla='BRA'), '26'),
    ('PI', 'Piauí',              (SELECT id FROM countries WHERE sigla='BRA'), '22'),
    ('PR', 'Paraná',             (SELECT id FROM countries WHERE sigla='BRA'), '41'),
    ('RJ', 'Rio de Janeiro',     (SELECT id FROM countries WHERE sigla='BRA'), '33'),
    ('RN', 'Rio Grande do Norte',(SELECT id FROM countries WHERE sigla='BRA'), '24'),
    ('RO', 'Rondônia',           (SELECT id FROM countries WHERE sigla='BRA'), '11'),
    ('RR', 'Roraima',            (SELECT id FROM countries WHERE sigla='BRA'), '14'),
    ('RS', 'Rio Grande do Sul',  (SELECT id FROM countries WHERE sigla='BRA'), '43'),
    ('SC', 'Santa Catarina',     (SELECT id FROM countries WHERE sigla='BRA'), '42'),
    ('SE', 'Sergipe',            (SELECT id FROM countries WHERE sigla='BRA'), '28'),
    ('SP', 'São Paulo',          (SELECT id FROM countries WHERE sigla='BRA'), '35'),
    ('TO', 'Tocantins',          (SELECT id FROM countries WHERE sigla='BRA'), '17');
