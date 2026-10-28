-- Regra do cadastro de parâmetros de ICMS/IPI (VFIS0320) que tributou a linha
-- da nota de saída. Referência histórica, sem FK: a regra pode ser alterada ou
-- excluída depois, e a nota continua dizendo de onde veio a tributação.
ALTER TABLE fiscal_exit_items ADD COLUMN IF NOT EXISTS regra_icms_ipi_id bigint;
COMMENT ON COLUMN fiscal_exit_items.regra_icms_ipi_id IS
    'icms_ipi_tax_params.id aplicado no cálculo; NULL = regra geral da empresa';

-- Percentual de redução da base do ICMS (pRedBC), exigido no XML nos CST 20 e
-- 70. Zero = sem redução de base.
ALTER TABLE fiscal_exit_items ADD COLUMN IF NOT EXISTS icms_reducao_base_pct numeric(7,4) NOT NULL DEFAULT 0;

-- Percentual de diferimento da linha (pDif, CST 51). Antes o XML levava o da
-- configuração da empresa; com diferimento por regra, a linha tem o seu.
-- Zero = linha antiga (vale o da configuração) ou sem diferimento.
ALTER TABLE fiscal_exit_items ADD COLUMN IF NOT EXISTS icms_diferimento_pct numeric(7,4) NOT NULL DEFAULT 0;
