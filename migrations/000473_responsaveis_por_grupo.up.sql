-- Responsável pelo planejamento e pelas compras de cada grupo do PDM. Aponta para
-- um SETOR (enterprise_departments), não para uma pessoa: quem sai da empresa ou
-- muda de área sai do setor, e o grupo continua com responsável. O item herda do
-- grupo; não há campo por item.
CREATE TABLE pdm_group_responsibles (
    enterprise_id            BIGINT NOT NULL REFERENCES enterprise(id),
    group_code               INTEGER NOT NULL,
    planning_department_id   UUID REFERENCES enterprise_departments(id) ON DELETE SET NULL,
    purchasing_department_id UUID REFERENCES enterprise_departments(id) ON DELETE SET NULL,
    updated_by               UUID REFERENCES users(id),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (enterprise_id, group_code),
    FOREIGN KEY (enterprise_id, group_code) REFERENCES groups(enterprise_id, code) ON DELETE CASCADE
);

SELECT instalar_conferencia_de_empresa();
