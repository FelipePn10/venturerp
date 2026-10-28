-- CIAP — Controle de Crédito do ICMS do Ativo Permanente (LC 87/1996, art.
-- 20, § 5º; EFD ICMS/IPI, Bloco G). O ICMS do bem do ativo imobilizado não é
-- creditado de uma vez na nota de entrada: entra no CIAP e é apropriado em
-- parcelas mensais (48, salvo legislação da UF), cada uma multiplicada pela
-- participação das saídas tributadas e de exportação no total de saídas do mês.

-- Parâmetros da empresa: o código de ajuste da apuração (tabela 5.1.1 da UF,
-- VFIS0510) com que o crédito do mês entra no E111, a conta contábil do ativo
-- e o número de parcelas.
CREATE TABLE ciap_parametros (
    enterprise_id     BIGINT       PRIMARY KEY REFERENCES enterprise(id),
    codigo_ajuste     VARCHAR(10),
    conta_ativo_id    BIGINT       REFERENCES accounting_accounts(id),
    numero_parcelas   SMALLINT     NOT NULL DEFAULT 48 CHECK (numero_parcelas BETWEEN 1 AND 999),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by        UUID
);

-- Bens no CIAP. Os da nota de entrada aprovada (CFOP de ativo x551/x406 com
-- ICMS) entram sozinhos; os anteriores ao sistema entram à mão (saldo inicial).
CREATE TABLE ciap_bens (
    id                    BIGSERIAL     PRIMARY KEY,
    enterprise_id         BIGINT        NOT NULL REFERENCES enterprise(id),
    codigo                VARCHAR(60)   NOT NULL,
    descricao             TEXT          NOT NULL,
    funcao                TEXT,
    vida_util_meses       SMALLINT,
    conta_ativo_id        BIGINT        REFERENCES accounting_accounts(id),
    cost_center_id        BIGINT        REFERENCES cost_centers(id),
    origem                VARCHAR(12)   NOT NULL CHECK (origem IN ('NF_ENTRADA', 'MANUAL')),
    fiscal_entry_item_id  BIGINT        UNIQUE REFERENCES fiscal_entry_items(id),
    data_entrada          DATE          NOT NULL,
    icms_op               NUMERIC(15,2) NOT NULL DEFAULT 0 CHECK (icms_op >= 0),
    icms_st               NUMERIC(15,2) NOT NULL DEFAULT 0 CHECK (icms_st >= 0),
    icms_frete            NUMERIC(15,2) NOT NULL DEFAULT 0 CHECK (icms_frete >= 0),
    icms_difal            NUMERIC(15,2) NOT NULL DEFAULT 0 CHECK (icms_difal >= 0),
    numero_parcelas       SMALLINT      NOT NULL DEFAULT 48 CHECK (numero_parcelas BETWEEN 1 AND 999),
    situacao              VARCHAR(10)   NOT NULL DEFAULT 'ATIVO' CHECK (situacao IN ('ATIVO', 'BAIXADO')),
    data_baixa            DATE,
    tipo_baixa            VARCHAR(2)    CHECK (tipo_baixa IN ('AT', 'PE', 'OT')),
    motivo_baixa          TEXT,
    created_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_by            UUID,
    CONSTRAINT ck_ciap_bem_icms CHECK (icms_op + icms_st + icms_frete + icms_difal > 0),
    CONSTRAINT ck_ciap_bem_baixa CHECK ((situacao = 'BAIXADO') = (data_baixa IS NOT NULL AND tipo_baixa IS NOT NULL))
);
CREATE UNIQUE INDEX uq_ciap_bens_codigo ON ciap_bens (enterprise_id, upper(codigo));
CREATE INDEX idx_ciap_bens_empresa ON ciap_bens (enterprise_id, situacao, data_entrada);

-- Apuração do mês (G110) e as parcelas apropriadas de cada bem (G125). A
-- apuração é refeita enquanto o mês não for fechado.
CREATE TABLE ciap_apuracoes (
    id              BIGSERIAL     PRIMARY KEY,
    enterprise_id   BIGINT        NOT NULL REFERENCES enterprise(id),
    ano             SMALLINT      NOT NULL CHECK (ano BETWEEN 2000 AND 2100),
    mes             SMALLINT      NOT NULL CHECK (mes BETWEEN 1 AND 12),
    saldo_inicial   NUMERIC(15,2) NOT NULL,
    soma_parcelas   NUMERIC(15,2) NOT NULL,
    saidas_trib_exp NUMERIC(15,2) NOT NULL CHECK (saidas_trib_exp >= 0),
    saidas_total    NUMERIC(15,2) NOT NULL CHECK (saidas_total >= 0),
    indice          NUMERIC(10,8) NOT NULL CHECK (indice BETWEEN 0 AND 1),
    icms_apropriado NUMERIC(15,2) NOT NULL,
    saidas_informadas BOOLEAN     NOT NULL DEFAULT FALSE,
    fechada         BOOLEAN       NOT NULL DEFAULT FALSE,
    apurado_em      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    apurado_por     UUID,
    UNIQUE (enterprise_id, ano, mes),
    CONSTRAINT ck_ciap_trib_total CHECK (saidas_trib_exp <= saidas_total)
);

CREATE TABLE ciap_apuracao_parcelas (
    apuracao_id     BIGINT        NOT NULL REFERENCES ciap_apuracoes(id) ON DELETE CASCADE,
    bem_id          BIGINT        NOT NULL REFERENCES ciap_bens(id),
    tipo_mov        VARCHAR(2)    NOT NULL CHECK (tipo_mov IN ('SI', 'IM', 'BA', 'AT', 'PE', 'OT')),
    numero_parcela  SMALLINT,
    valor_parcela   NUMERIC(15,2),
    PRIMARY KEY (apuracao_id, bem_id, tipo_mov)
);
CREATE INDEX idx_ciap_parcelas_bem ON ciap_apuracao_parcelas (bem_id);

SELECT instalar_conferencia_de_empresa();
