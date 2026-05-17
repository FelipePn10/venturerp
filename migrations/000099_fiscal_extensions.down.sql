BEGIN;

DROP VIEW IF EXISTS public.saldo_contas;
DROP TABLE IF EXISTS public.usuarios_perfis;
DROP TABLE IF EXISTS public.permissoes;
DROP TABLE IF EXISTS public.perfis_usuario;
DROP TABLE IF EXISTS public.focus_nfe_logs;
DROP TABLE IF EXISTS public.extrato_bancario;
DROP TABLE IF EXISTS public.carta_correcao;
DROP TABLE IF EXISTS public.cte_nfe_association;
DROP TABLE IF EXISTS public.fiscal_cte;

ALTER TABLE public.fiscal_exits
    DROP COLUMN IF EXISTS motivo_cancelamento,
    DROP COLUMN IF EXISTS data_cancelamento,
    DROP COLUMN IF EXISTS cancelado_por,
    DROP COLUMN IF EXISTS emitida_contingencia,
    DROP COLUMN IF EXISTS condicao_pagamento_id,
    DROP COLUMN IF EXISTS tipo_pagamento;

ALTER TABLE public.contas_receber DROP COLUMN IF EXISTS condicao_pagamento_id;
ALTER TABLE public.contas_pagar DROP COLUMN IF EXISTS fornecedor_cnpj;

COMMIT;
