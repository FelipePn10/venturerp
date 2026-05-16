BEGIN;

DROP INDEX IF EXISTS idx_tax_competencia;
DROP INDEX IF EXISTS idx_fc_conta;
DROP INDEX IF EXISTS idx_fc_data;
DROP INDEX IF EXISTS idx_cr_cliente;
DROP INDEX IF EXISTS idx_cr_status;
DROP INDEX IF EXISTS idx_cr_vencimento;
DROP INDEX IF EXISTS idx_cp_fornecedor;
DROP INDEX IF EXISTS idx_cp_status;
DROP INDEX IF EXISTS idx_cp_vencimento;

DROP TABLE IF EXISTS public.tax_assessments;
DROP TABLE IF EXISTS public.fluxo_caixa;
DROP TABLE IF EXISTS public.contas_receber;
DROP TABLE IF EXISTS public.contas_pagar;
DROP TABLE IF EXISTS public.centros_custo;
DROP TABLE IF EXISTS public.plano_contas;
DROP TABLE IF EXISTS public.formas_pagamento;
DROP TABLE IF EXISTS public.condicoes_pagamento;
DROP TABLE IF EXISTS public.contas_bancarias;

COMMIT;
