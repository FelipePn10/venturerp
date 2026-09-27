-- Comissão de mais de um representante no mesmo pedido, com percentual próprio.
--
-- O pedido e o orçamento guardavam UM representante e UM percentual na capa.
-- Na prática a venda costuma ser dividida: o representante da região e o
-- parceiro que trouxe o cliente, cada um com a sua taxa, e a taxa muda de
-- pedido para pedido (venda casada, campanha, cliente novo). Com um campo só,
-- a segunda comissão era acertada por fora do sistema.
--
-- É o modelo dos ERPs grandes: no Protheus são os pares vendedor/comissão do
-- cabeçalho (SC5_VEND1..5 / SC5_COMIS1..5); no SAP são funções de parceiro do
-- tipo "representante de vendas" no pedido; no FoccoERP, o rateio de comissão
-- por representante.
--
-- A capa continua com `representative_code` e `commission_pct`: é o
-- representante principal, e nenhum pedido antigo muda de significado. A tabela
-- nova é o rateio completo, inclusive do principal.

CREATE TYPE sales_commission_role_enum AS ENUM ('PRINCIPAL', 'PARCEIRO');

-- Sobre o que a comissão incide. Comissão sobre o total COM IPI paga o
-- representante por imposto que a empresa só repassa — é erro clássico de
-- cadastro, e ter a base explícita no pedido é o que permite auditar depois.
CREATE TYPE sales_commission_base_enum AS ENUM ('TOTAL_PRODUTOS', 'TOTAL_LIQUIDO');

CREATE TABLE IF NOT EXISTS sales_order_representatives (
    id                  BIGSERIAL PRIMARY KEY,
    -- `enterprise_code` e não `enterprise_id`: é a convenção do pedido e do
    -- orçamento, e a filha tem de casar com o pai para o filtro ser um só.
    enterprise_code     BIGINT NOT NULL REFERENCES enterprise(code),
    sales_order_code    BIGINT NOT NULL REFERENCES sales_orders(code) ON DELETE CASCADE,
    representative_code BIGINT NOT NULL REFERENCES representatives(code),
    role                sales_commission_role_enum NOT NULL DEFAULT 'PRINCIPAL',
    commission_pct      NUMERIC(9,4) NOT NULL DEFAULT 0,
    commission_base     sales_commission_base_enum NOT NULL DEFAULT 'TOTAL_PRODUTOS',
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sales_order_representatives_pct_check CHECK (commission_pct >= 0 AND commission_pct <= 100),
    CONSTRAINT sales_order_representatives_unico UNIQUE (sales_order_code, representative_code)
);

CREATE TABLE IF NOT EXISTS sales_quotation_representatives (
    id                   BIGSERIAL PRIMARY KEY,
    enterprise_code      BIGINT NOT NULL REFERENCES enterprise(code),
    sales_quotation_code BIGINT NOT NULL REFERENCES sales_quotations(code) ON DELETE CASCADE,
    representative_code  BIGINT NOT NULL REFERENCES representatives(code),
    role                 sales_commission_role_enum NOT NULL DEFAULT 'PRINCIPAL',
    commission_pct       NUMERIC(9,4) NOT NULL DEFAULT 0,
    commission_base      sales_commission_base_enum NOT NULL DEFAULT 'TOTAL_PRODUTOS',
    notes                TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sales_quotation_representatives_pct_check CHECK (commission_pct >= 0 AND commission_pct <= 100),
    CONSTRAINT sales_quotation_representatives_unico UNIQUE (sales_quotation_code, representative_code)
);

CREATE INDEX IF NOT EXISTS idx_sales_order_representatives_empresa ON sales_order_representatives (enterprise_code);
CREATE INDEX IF NOT EXISTS idx_sales_quotation_representatives_empresa ON sales_quotation_representatives (enterprise_code);
CREATE INDEX IF NOT EXISTS idx_sales_order_representatives_pedido ON sales_order_representatives (sales_order_code);
CREATE INDEX IF NOT EXISTS idx_sales_order_representatives_rep ON sales_order_representatives (representative_code);
CREATE INDEX IF NOT EXISTS idx_sales_quotation_representatives_orcamento ON sales_quotation_representatives (sales_quotation_code);
CREATE INDEX IF NOT EXISTS idx_sales_quotation_representatives_rep ON sales_quotation_representatives (representative_code);

-- O que já existe vira a primeira linha do rateio: o representante da capa,
-- com o percentual da capa. Assim nenhum pedido fica sem rateio e o relatório
-- de comissão lê uma fonte só.
INSERT INTO sales_order_representatives (enterprise_code, sales_order_code, representative_code, role, commission_pct)
SELECT COALESCE(o.enterprise_code, (SELECT MIN(code) FROM enterprise)), o.code, o.representative_code, 'PRINCIPAL',
       COALESCE(o.commission_pct, 0)
FROM sales_orders o
WHERE o.representative_code IS NOT NULL
  AND EXISTS (SELECT 1 FROM representatives r WHERE r.code = o.representative_code)
ON CONFLICT DO NOTHING;

INSERT INTO sales_quotation_representatives (enterprise_code, sales_quotation_code, representative_code, role, commission_pct)
SELECT COALESCE(q.enterprise_code, (SELECT MIN(code) FROM enterprise)), q.code, q.representative_code, 'PRINCIPAL',
       COALESCE(q.commission_pct, 0)
FROM sales_quotations q
WHERE q.representative_code IS NOT NULL
  AND EXISTS (SELECT 1 FROM representatives r WHERE r.code = q.representative_code)
ON CONFLICT DO NOTHING;
