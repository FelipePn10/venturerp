-- Matriz de tempo de preparação dependente da sequência.
--
-- O setup da operação era fixo: trocar de preto para preto custava o mesmo que
-- trocar de branco para preto. Na prática é o contrário — em movelaria a troca
-- de cor obriga a limpar a linha, e em metalúrgica a troca de matriz ou de
-- espessura para a máquina. É esse custo que o sequenciador tenta evitar
-- agrupando itens parecidos, e sem a matriz ele não tem como enxergá-lo.
--
-- Equivale à Matriz do Tempo de Preparação das Máquinas (FPRD0113) do FoccoERP.
--
-- A regra pode ser escrita por item ou por família (classificação do item):
-- família cobre o caso geral com poucas linhas, item afina onde importa.
CREATE TABLE IF NOT EXISTS setup_matrix (
    id             bigserial PRIMARY KEY,
    enterprise_id  bigint      NOT NULL,
    work_center_id bigint      NOT NULL,

    -- De onde e para onde. NULL nos dois lados do mesmo par = "qualquer".
    from_item_code   bigint,
    to_item_code     bigint,
    from_family      varchar(80),
    to_family        varchar(80),

    setup_minutes  numeric(12,4) NOT NULL,
    notes          text,
    is_active      boolean     NOT NULL DEFAULT true,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT ck_setup_matrix_minutos CHECK (setup_minutes >= 0),
    -- Uma linha precisa dizer algo: ou o par de itens, ou o par de famílias.
    CONSTRAINT ck_setup_matrix_tem_criterio CHECK (
        from_item_code IS NOT NULL OR to_item_code IS NOT NULL
        OR from_family IS NOT NULL OR to_family IS NOT NULL
    )
);

COMMENT ON TABLE setup_matrix IS
    'Tempo de preparação por transição (item ou família) em cada centro de trabalho.';
COMMENT ON COLUMN setup_matrix.from_item_code IS 'Item que estava na máquina; NULL = qualquer.';
COMMENT ON COLUMN setup_matrix.to_item_code IS 'Item que vai entrar; NULL = qualquer.';

-- Uma transição não pode ser cadastrada duas vezes para o mesmo centro.
CREATE UNIQUE INDEX IF NOT EXISTS ux_setup_matrix_transicao
    ON setup_matrix (
        enterprise_id, work_center_id,
        COALESCE(from_item_code, -1), COALESCE(to_item_code, -1),
        COALESCE(from_family, ''), COALESCE(to_family, '')
    );

CREATE INDEX IF NOT EXISTS idx_setup_matrix_busca
    ON setup_matrix (enterprise_id, work_center_id, is_active);
