-- Tributação por cenário: a nota deixa de depender de uma alíquota única da
-- empresa. O que muda o imposto de uma venda para outra passa a ter lugar no
-- banco: a finalidade da compra (revenda, industrialização, uso e consumo), o
-- regime do emitente (Simples Nacional com CSOSN e crédito), a MVA da regra de
-- ICMS/IPI (VFIS0320) e o tipo de nota da saída.

-- ── Configuração fiscal ──────────────────────────────────────────────────────
-- exigir_regra_icms_ipi: ligado, a linha de nota sem regra no cadastro de
-- parâmetros de ICMS/IPI não é transmitida (a prévia aponta a pendência).
-- Desligado, vale a regra geral da empresa e a prévia só avisa.
-- aliquota_credito_simples: pCredSN (%) do optante do Simples Nacional, o
-- crédito de ICMS que o comprador pode aproveitar (CSOSN 101/201).
ALTER TABLE fiscal_configs
    ADD COLUMN IF NOT EXISTS exigir_regra_icms_ipi    boolean      NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS aliquota_credito_simples numeric(7,4) NOT NULL DEFAULT 0,
    ADD CONSTRAINT chk_fiscal_configs_credito_simples
        CHECK (aliquota_credito_simples >= 0 AND aliquota_credito_simples <= 100);

-- ── Finalidade da compra ─────────────────────────────────────────────────────
-- REVENDA e INDUSTRIALIZACAO: o destinatário revende ou transforma (IPI fora da
-- base do ICMS entre contribuintes, sem consumidor final). USO_CONSUMO: uso,
-- consumo ou ativo imobilizado (consumidor final, IPI na base do ICMS, mesmo com
-- IE). NULL = deduzir: com inscrição estadual não é consumidor final; sem ela, é.
ALTER TABLE customers
    ADD COLUMN IF NOT EXISTS finalidade_compra varchar(20),
    ADD CONSTRAINT chk_customers_finalidade_compra
        CHECK (finalidade_compra IN ('REVENDA', 'INDUSTRIALIZACAO', 'USO_CONSUMO'));

ALTER TABLE sales_orders
    ADD COLUMN IF NOT EXISTS finalidade_compra varchar(20),
    ADD CONSTRAINT chk_sales_orders_finalidade_compra
        CHECK (finalidade_compra IN ('REVENDA', 'INDUSTRIALIZACAO', 'USO_CONSUMO'));

-- A nota guarda a finalidade que valeu no cálculo e o tipo de nota (VFIS0110)
-- cujos indicadores tributaram a saída.
ALTER TABLE fiscal_exits
    ADD COLUMN IF NOT EXISTS finalidade_compra varchar(20),
    ADD COLUMN IF NOT EXISTS invoice_type_id   bigint,
    ADD CONSTRAINT chk_fiscal_exits_finalidade_compra
        CHECK (finalidade_compra IN ('REVENDA', 'INDUSTRIALIZACAO', 'USO_CONSUMO')),
    ADD CONSTRAINT fiscal_exits_invoice_type_id_empresa_fkey
        FOREIGN KEY (enterprise_id, invoice_type_id) REFERENCES invoice_types (enterprise_id, id);
CREATE INDEX IF NOT EXISTS idx_fiscal_exits_invoice_type ON fiscal_exits (invoice_type_id) WHERE invoice_type_id IS NOT NULL;

-- ── Regra de ICMS/IPI ────────────────────────────────────────────────────────
-- finalidade_compra: restringe a regra a uma finalidade (NULL = qualquer).
-- mva_pct: MVA original da substituição tributária (40 = 40%); zero = sem ST.
-- mva_ajustar_interestadual: na venda interestadual a MVA é ajustada pela
-- fórmula do Convênio ICMS 142/18 (cláusula 13ª); desligado, vale como está.
ALTER TABLE icms_ipi_tax_params
    ADD COLUMN IF NOT EXISTS finalidade_compra         varchar(20),
    ADD COLUMN IF NOT EXISTS mva_pct                   numeric(7,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS mva_ajustar_interestadual boolean      NOT NULL DEFAULT true,
    ADD CONSTRAINT chk_icms_ipi_tax_params_finalidade_compra
        CHECK (finalidade_compra IN ('REVENDA', 'INDUSTRIALIZACAO', 'USO_CONSUMO')),
    ADD CONSTRAINT chk_icms_ipi_tax_params_mva CHECK (mva_pct >= 0);

-- ── Linha da nota de saída ───────────────────────────────────────────────────
-- Crédito do Simples (pCredSN/vCredICMSSN) e o grupo ICMSUFDest (DIFAL da venda
-- interestadual a consumidor final não contribuinte, EC 87/2015). Antes o DIFAL
-- era calculado e descartado: a nota ia sem o grupo e a SEFAZ a recusava.
ALTER TABLE fiscal_exit_items
    ADD COLUMN IF NOT EXISTS aliq_credito_simples      numeric(7,4)  NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS valor_credito_simples     numeric(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS icms_uf_dest_base         numeric(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS icms_uf_dest_aliq_interna numeric(7,4)  NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS icms_aliq_interestadual   numeric(7,4)  NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS icms_uf_dest_valor        numeric(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS fcp_uf_dest_pct           numeric(7,4)  NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS fcp_uf_dest_valor         numeric(15,2) NOT NULL DEFAULT 0;
