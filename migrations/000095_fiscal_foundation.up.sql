BEGIN;

-- NCM Tax Table (IPI aliquots, PIS/COFINS, ICMS config per scenario)
CREATE TABLE IF NOT EXISTS public.ncm_tax_table (
    id              BIGSERIAL PRIMARY KEY,
    ncm             VARCHAR(10) NOT NULL,
    aliq_ipi        NUMERIC(7,4) NOT NULL DEFAULT 0,
    aliq_pis         NUMERIC(7,4) NOT NULL DEFAULT 0.0165,
    aliq_cofins      NUMERIC(7,4) NOT NULL DEFAULT 0.0760,
    cst_pis         VARCHAR(2) NOT NULL DEFAULT '01',
    cst_cofins      VARCHAR(2) NOT NULL DEFAULT '01',
    cst_ipi         VARCHAR(2) NOT NULL DEFAULT '50',
    description     VARCHAR(300),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(ncm)
);

-- Tax Scenario Configuration
CREATE TABLE IF NOT EXISTS public.tax_scenarios (
    id                  BIGSERIAL PRIMARY KEY,
    scenario_name       VARCHAR(50) NOT NULL UNIQUE,
    destination_uf      VARCHAR(2),
    destination_type    VARCHAR(20),
    aliq_icms          NUMERIC(7,4) NOT NULL DEFAULT 0,
    dif_icms_pct       NUMERIC(7,4) NOT NULL DEFAULT 0,
    cst_icms           VARCHAR(3) NOT NULL DEFAULT '00',
    calc_difal          BOOLEAN NOT NULL DEFAULT FALSE,
    aliq_fcp           NUMERIC(7,4) NOT NULL DEFAULT 0,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ICMS inter-state aliquot table
CREATE TABLE IF NOT EXISTS public.icms_interstate (
    id              BIGSERIAL PRIMARY KEY,
    origin_uf       VARCHAR(2) NOT NULL,
    destination_uf  VARCHAR(2) NOT NULL,
    aliq_icms      NUMERIC(7,4) NOT NULL,
    description     VARCHAR(100),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE(origin_uf, destination_uf)
);

-- ICMS internal aliquot per UF
CREATE TABLE IF NOT EXISTS public.icms_internal (
    id              BIGSERIAL PRIMARY KEY,
    uf              VARCHAR(2) NOT NULL UNIQUE,
    aliq_icms      NUMERIC(7,4) NOT NULL DEFAULT 0,
    aliq_fcp       NUMERIC(7,4) NOT NULL DEFAULT 0,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fiscal Entry Documents (NF-e de Entrada)
CREATE TABLE IF NOT EXISTS public.fiscal_entries (
    id                      BIGSERIAL PRIMARY KEY,
    chave_acesso            VARCHAR(44),
    numero_nf               BIGINT NOT NULL,
    serie                   VARCHAR(3) NOT NULL DEFAULT '1',
    modelo                  VARCHAR(2) NOT NULL DEFAULT '55',
    data_emissao            DATE NOT NULL,
    data_entrada            DATE NOT NULL DEFAULT CURRENT_DATE,
    cnpj_emitente           VARCHAR(14) NOT NULL,
    razao_social_emitente   VARCHAR(200) NOT NULL,
    ie_emitente             VARCHAR(14),
    uf_emitente             VARCHAR(2),
    valor_produtos          NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_frete             NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_seguro            NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_desconto          NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_ipi               NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_icms              NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_pis               NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_cofins            NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_total             NUMERIC(15,2) NOT NULL DEFAULT 0,
    tipo_documento          VARCHAR(20) NOT NULL DEFAULT 'NFE',
    purchase_order_code     BIGINT,
    cte_code                BIGINT,
    status                  VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    xml_path                VARCHAR(500),
    notes                   TEXT,
    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              UUID NOT NULL
);

-- Fiscal Entry Items
CREATE TABLE IF NOT EXISTS public.fiscal_entry_items (
    id                  BIGSERIAL PRIMARY KEY,
    fiscal_entry_id     BIGINT NOT NULL REFERENCES public.fiscal_entries(id) ON DELETE CASCADE,
    sequence            INT NOT NULL DEFAULT 1,
    item_code           BIGINT,
    ncm                 VARCHAR(10),
    cfop                VARCHAR(4) NOT NULL,
    quantity            NUMERIC(15,4) NOT NULL,
    unit_price          NUMERIC(15,2) NOT NULL,
    total_price         NUMERIC(15,2) NOT NULL,
    base_icms           NUMERIC(15,2) NOT NULL DEFAULT 0,
    aliq_icms           NUMERIC(7,4) NOT NULL DEFAULT 0,
    valor_icms          NUMERIC(15,2) NOT NULL DEFAULT 0,
    base_ipi            NUMERIC(15,2) NOT NULL DEFAULT 0,
    aliq_ipi            NUMERIC(7,4) NOT NULL DEFAULT 0,
    valor_ipi           NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_pis           NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_cofins        NUMERIC(15,2) NOT NULL DEFAULT 0,
    cst_icms            VARCHAR(3),
    cst_ipi             VARCHAR(3),
    cst_pis             VARCHAR(3),
    cst_cofins          VARCHAR(3),
    gera_credito_icms   BOOLEAN NOT NULL DEFAULT TRUE,
    gera_credito_ipi    BOOLEAN NOT NULL DEFAULT TRUE,
    gera_credito_pis    BOOLEAN NOT NULL DEFAULT TRUE,
    gera_credito_cofins BOOLEAN NOT NULL DEFAULT TRUE,
    description         VARCHAR(300),
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fiscal Out Documents (NF-e de Saída)
CREATE TABLE IF NOT EXISTS public.fiscal_exits (
    id                      BIGSERIAL PRIMARY KEY,
    chave_acesso            VARCHAR(44),
    numero_nf               BIGINT NOT NULL,
    serie                   VARCHAR(3) NOT NULL DEFAULT '1',
    data_emissao            DATE NOT NULL,
    data_saida              DATE,
    cnpj_destinatario       VARCHAR(14),
    razao_social_destinatario VARCHAR(200),
    ie_destinatario         VARCHAR(14),
    uf_destinatario         VARCHAR(2),
    cfop                    VARCHAR(4) NOT NULL,
    natureza_operacao       VARCHAR(200) NOT NULL,
    valor_produtos          NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_frete             NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_seguro            NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_desconto          NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_ipi               NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_icms              NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_pis               NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_cofins            NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_total             NUMERIC(15,2) NOT NULL DEFAULT 0,
    sales_order_code        BIGINT,
    status                  VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    protocolo               VARCHAR(50),
    xml_path                VARCHAR(500),
    danfe_path              VARCHAR(500),
    focus_ref               VARCHAR(50),
    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              UUID NOT NULL
);

-- Fiscal Exit Items
CREATE TABLE IF NOT EXISTS public.fiscal_exit_items (
    id                      BIGSERIAL PRIMARY KEY,
    fiscal_exit_id          BIGINT NOT NULL REFERENCES public.fiscal_exits(id) ON DELETE CASCADE,
    sequence                INT NOT NULL DEFAULT 1,
    item_code               BIGINT,
    ncm                     VARCHAR(10),
    cfop                    VARCHAR(4) NOT NULL,
    quantity                NUMERIC(15,4) NOT NULL,
    unit_price              NUMERIC(15,2) NOT NULL,
    total_price             NUMERIC(15,2) NOT NULL,
    base_icms               NUMERIC(15,2) NOT NULL DEFAULT 0,
    aliq_icms               NUMERIC(7,4) NOT NULL DEFAULT 0,
    valor_icms              NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_icms_diferido     NUMERIC(15,2) NOT NULL DEFAULT 0,
    base_ipi                NUMERIC(15,2) NOT NULL DEFAULT 0,
    aliq_ipi                NUMERIC(7,4) NOT NULL DEFAULT 0,
    valor_ipi               NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_pis               NUMERIC(15,2) NOT NULL DEFAULT 0,
    valor_cofins            NUMERIC(15,2) NOT NULL DEFAULT 0,
    cst_icms                VARCHAR(3),
    cst_ipi                 VARCHAR(3),
    cst_pis                 VARCHAR(3),
    cst_cofins              VARCHAR(3),
    origem_mercadoria       VARCHAR(1) NOT NULL DEFAULT '0',
    description             VARCHAR(300),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fiscal Configurations
CREATE TABLE IF NOT EXISTS public.fiscal_configs (
    id                          BIGSERIAL PRIMARY KEY,
    cnpj_empresa                VARCHAR(14) NOT NULL,
    razao_social                VARCHAR(200) NOT NULL,
    ie_empresa                  VARCHAR(14),
    regime_tributario           VARCHAR(20) NOT NULL DEFAULT 'lucro_real',
    uf_empresa                  VARCHAR(2) NOT NULL DEFAULT 'PR',
    icms_interno_aliquota       NUMERIC(7,4) NOT NULL DEFAULT 0.195,
    icms_diferimento_percentual NUMERIC(7,4) NOT NULL DEFAULT 0.3846,
    focus_nfe_token             VARCHAR(200),
    focus_nfe_ambiente          VARCHAR(20) NOT NULL DEFAULT 'homologacao',
    juros_mes                   NUMERIC(7,4) NOT NULL DEFAULT 0.01,
    multa_atraso                NUMERIC(7,4) NOT NULL DEFAULT 0.02,
    vencimento_icms_dia         INT NOT NULL DEFAULT 20,
    vencimento_ipi_dia          INT NOT NULL DEFAULT 25,
    vencimento_pis_cofins_dia   INT NOT NULL DEFAULT 25,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by                  UUID NOT NULL
);

-- Insert default PR fiscal config
INSERT INTO public.fiscal_configs (cnpj_empresa, razao_social, ie_empresa, regime_tributario, uf_empresa, updated_by)
VALUES ('00000000000000', 'Empresa Default', 'ISENTO', 'lucro_real', 'PR', '00000000-0000-0000-0000-000000000000');

-- Seed NCM data with IPI aliquots
INSERT INTO public.ncm_tax_table (ncm, aliq_ipi) VALUES
('2804.21.00', 0), ('2804.40.00', 0), ('3208.90.10', 0.0325), ('3506.10.90', 0), ('3506.91.90', 0),
('3814.00.90', 0.065), ('3824.99.41', 0), ('3920.62.99', 0.0975), ('3923.29.10', 0.0975), ('3926.90.90', 0.0975),
('4009.21.90', 0.065), ('4415.20.00', 0), ('5911.90.00', 0.0325), ('6403.91.90', 0), ('6805.20.00', 0),
('6805.30.90', 0), ('7208.40.00', 0.0325), ('7208.51.00', 0.0325), ('7208.52.00', 0.0325), ('7208.53.00', 0.0325),
('7208.54.00', 0.0325), ('7209.26.00', 0.0325), ('7209.27.00', 0.0325), ('7214.91.00', 0), ('7214.99.10', 0),
('7215.50.00', 0.0325), ('7216.10.00', 0), ('7216.21.00', 0), ('7216.31.00', 0), ('7216.61.10', 0),
('7219.12.00', 0.0325), ('7219.32.00', 0.0325), ('7219.33.00', 0.0325), ('7219.90.90', 0.0325), ('7222.20.00', 0.0325),
('7229.20.00', 0.0325), ('7306.30.10', 0.0325), ('7306.40.00', 0.0325), ('7306.61.00', 0.05), ('7306.69.00', 0.0325),
('7307.19.20', 0.0325), ('7307.99.00', 0.0325), ('7308.20.00', 0), ('7308.40.00', 0), ('7308.90.10', 0),
('7309.00.90', 0), ('7312.90.00', 0.0975), ('7315.82.00', 0.0975), ('7318.12.00', 0.065), ('7318.15.00', 0.065),
('7318.16.00', 0.065), ('7318.19.00', 0.065), ('7318.21.00', 0.065), ('7318.22.00', 0.065), ('7318.29.00', 0.065),
('7323.93.00', 0.065), ('7326.19.00', 0.065), ('7326.90.90', 0.05), ('7407.10.10', 0.0325), ('7407.10.29', 0.0325),
('7606.92.00', 0.0325), ('8204.11.00', 0.052), ('8204.20.00', 0.052), ('8205.10.00', 0.052), ('8207.30.00', 0),
('8207.40.10', 0.052), ('8207.50.19', 0.052), ('8207.90.00', 0.052), ('8302.10.00', 0), ('8302.49.00', 0.065),
('8309.90.00', 0), ('8421.39.90', 0), ('8422.40.90', 0), ('8422.90.90', 0.0325), ('8426.11.00', 0),
('8434.90.00', 0.0325), ('8451.80.00', 0), ('8467.29.99', 0.052), ('8480.60.00', 0.065), ('8481.10.00', 0),
('8481.80.95', 0), ('8482.10.10', 0.078), ('8482.10.90', 0.078), ('8483.30.90', 0.078), ('8483.90.00', 0),
('8504.90.30', 0.065), ('8507.90.10', 0.0975), ('8515.39.00', 0), ('8515.90.00', 0), ('8528.52.00', 0.15),
('8716.80.00', 0.0325), ('8716.90.90', 0.0325), ('9406.90.20', 0), ('9603.40.10', 0);

-- Seed tax scenarios for PR
INSERT INTO public.tax_scenarios (scenario_name, destination_uf, destination_type, aliq_icms, dif_icms_pct, cst_icms) VALUES
('INTERESTADUAL_SUL_SUDESTE', NULL, 'contribuinte', 0.12, 0, '00'),
('INTERESTADUAL_NORTE_NORDESTE', NULL, 'contribuinte', 0.07, 0, '00'),
('INTERESTADUAL_IMPORTADA', NULL, 'contribuinte', 0.04, 0, '00'),
('INTERNA_PR_CONTRIBUINTE', 'PR', 'contribuinte', 0.195, 38.46, '51'),
('INTERNA_PR_NAO_CONTRIBUINTE', 'PR', 'nao_contribuinte', 0.195, 0, '00');

-- ICMS interstate seeds
INSERT INTO public.icms_interstate (origin_uf, destination_uf, aliq_icms) VALUES
('PR', 'RS', 0.12), ('PR', 'SC', 0.12), ('PR', 'SP', 0.12), ('PR', 'RJ', 0.12),
('PR', 'MG', 0.12), ('PR', 'ES', 0.12), ('PR', 'GO', 0.12), ('PR', 'MT', 0.12),
('PR', 'MS', 0.12), ('PR', 'DF', 0.12), ('PR', 'BA', 0.07), ('PR', 'PE', 0.07),
('PR', 'CE', 0.07), ('PR', 'MA', 0.07), ('PR', 'PA', 0.07), ('PR', 'AM', 0.07),
('PR', 'RO', 0.07), ('PR', 'TO', 0.07), ('PR', 'AC', 0.07), ('PR', 'RR', 0.07),
('PR', 'AP', 0.07), ('PR', 'SE', 0.07), ('PR', 'AL', 0.07), ('PR', 'PB', 0.07),
('PR', 'RN', 0.07), ('PR', 'PI', 0.07);

-- ICMS internal per UF seeds (key states)
INSERT INTO public.icms_internal (uf, aliq_icms, aliq_fcp) VALUES
('PR', 0.195, 0.02), ('SP', 0.18, 0.02), ('RJ', 0.18, 0.02), ('MG', 0.18, 0.02),
('RS', 0.17, 0.02), ('SC', 0.17, 0.02), ('ES', 0.17, 0.02), ('BA', 0.18, 0.02),
('GO', 0.17, 0.02), ('MT', 0.17, 0.02), ('MS', 0.17, 0.02), ('DF', 0.18, 0.02);

COMMIT;
