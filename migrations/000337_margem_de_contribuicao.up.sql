-- Margem de contribuição: quanto sobra de cada venda depois de tudo.
--
-- Equivale ao FCST0108 (parâmetros), FCST0254 (geração) e FCST0320 (apuração)
-- do FoccoERP. Sem isso o sistema responde "quanto vendemos" e "quanto custou",
-- mas não "em quais produtos e clientes ganhamos dinheiro" — que é a pergunta
-- que decide mix, preço e desconto.

-- Parâmetros do mês: percentuais e prazos que valem para todas as vendas do
-- período. Cadastrados uma vez por mês, como no FCST0108.
CREATE TABLE IF NOT EXISTS margin_parameters (
    id            bigserial PRIMARY KEY,
    enterprise_id bigint  NOT NULL,
    year          int     NOT NULL,
    month         int     NOT NULL,

    ir_pct                 numeric(9,4) NOT NULL DEFAULT 0,
    admin_pct              numeric(9,4) NOT NULL DEFAULT 0,
    freight_pct            numeric(9,4) NOT NULL DEFAULT 0,
    financial_rate_monthly numeric(9,4) NOT NULL DEFAULT 0,

    avg_sales_term_days    int NOT NULL DEFAULT 0,
    avg_purchase_term_days int NOT NULL DEFAULT 0,
    production_cycle_days  int NOT NULL DEFAULT 0,

    material_payment_days int NOT NULL DEFAULT 0,
    labor_payment_days    int NOT NULL DEFAULT 0,
    ipi_payment_days      int NOT NULL DEFAULT 0,
    icms_payment_days     int NOT NULL DEFAULT 0,
    pis_payment_days      int NOT NULL DEFAULT 0,
    cofins_payment_days   int NOT NULL DEFAULT 0,

    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT ck_margin_parameters_mes CHECK (month BETWEEN 1 AND 12),
    CONSTRAINT ck_margin_parameters_ano CHECK (year BETWEEN 2000 AND 2199),
    CONSTRAINT ck_margin_parameters_percentuais CHECK (
        ir_pct >= 0 AND admin_pct >= 0 AND freight_pct >= 0 AND financial_rate_monthly >= 0
    ),
    CONSTRAINT ux_margin_parameters_periodo UNIQUE (enterprise_id, year, month)
);

COMMENT ON TABLE margin_parameters IS
    'Percentuais e prazos do mês usados no cálculo da margem de contribuição.';
COMMENT ON COLUMN margin_parameters.financial_rate_monthly IS
    'Taxa financeira ao mês (%); ajustada ao ciclo de caixa no cálculo.';

-- Resultado por linha de venda. Guardamos a cascata inteira porque o custo e os
-- parâmetros mudam com o tempo: reapurar depois daria outro número, e a análise
-- precisa refletir as condições da época da venda.
CREATE TABLE IF NOT EXISTS contribution_margin (
    id            bigserial PRIMARY KEY,
    enterprise_id bigint NOT NULL,

    -- Origem: nota fiscal de saída ou pedido de venda (FCST0254 aceita as duas).
    source        varchar(12) NOT NULL,
    source_id     bigint      NOT NULL,
    source_item   int         NOT NULL DEFAULT 0,
    issue_date    date        NOT NULL,

    customer_code bigint,
    item_code     bigint,
    quantity      numeric(18,6) NOT NULL DEFAULT 0,

    gross_revenue           numeric(18,4) NOT NULL DEFAULT 0,
    ipi                     numeric(18,4) NOT NULL DEFAULT 0,
    merchandise_revenue     numeric(18,4) NOT NULL DEFAULT 0,
    icms                    numeric(18,4) NOT NULL DEFAULT 0,
    pis_cofins              numeric(18,4) NOT NULL DEFAULT 0,
    material_cost           numeric(18,4) NOT NULL DEFAULT 0,
    conversion_cost         numeric(18,4) NOT NULL DEFAULT 0,
    gross_profit            numeric(18,4) NOT NULL DEFAULT 0,
    admin_expense           numeric(18,4) NOT NULL DEFAULT 0,
    commission              numeric(18,4) NOT NULL DEFAULT 0,
    freight                 numeric(18,4) NOT NULL DEFAULT 0,
    other_expense           numeric(18,4) NOT NULL DEFAULT 0,
    financial_expense       numeric(18,4) NOT NULL DEFAULT 0,
    income_tax_provision    numeric(18,4) NOT NULL DEFAULT 0,
    margin                  numeric(18,4) NOT NULL DEFAULT 0,
    margin_pct              numeric(9,4)  NOT NULL DEFAULT 0,

    -- Qual custo foi usado: médio (real) ou padrão. O FCST0320 depende dessa
    -- escolha para explicar o número, então ela fica gravada com o resultado.
    cost_basis  varchar(10) NOT NULL DEFAULT 'PADRAO',
    calculated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT ck_contribution_margin_origem CHECK (source IN ('NOTA', 'PEDIDO')),
    CONSTRAINT ck_contribution_margin_base CHECK (cost_basis IN ('MEDIO', 'PADRAO')),
    CONSTRAINT ux_contribution_margin_linha UNIQUE (enterprise_id, source, source_id, source_item)
);

COMMENT ON TABLE contribution_margin IS
    'Margem apurada por linha de venda, com a cascata congelada na data do cálculo.';

CREATE INDEX IF NOT EXISTS idx_contribution_margin_periodo
    ON contribution_margin (enterprise_id, issue_date);
CREATE INDEX IF NOT EXISTS idx_contribution_margin_item
    ON contribution_margin (enterprise_id, item_code);
CREATE INDEX IF NOT EXISTS idx_contribution_margin_cliente
    ON contribution_margin (enterprise_id, customer_code);
