-- Contas da contabilização da NF-e de saída para IBS e CBS (a partir de 2027,
-- quando o destaque vira tributo a recolher). Mesmo tratamento do ICMS: o
-- total da nota e as duplicatas não somam IBS/CBS por fora, então o imposto
-- sai da receita como dedução (D "sobre vendas") contra o passivo (C "a recolher").
ALTER TABLE public.accounting_posting_params
    ADD COLUMN IF NOT EXISTS ibs_vendas_account_id   BIGINT REFERENCES public.accounting_accounts(id),
    ADD COLUMN IF NOT EXISTS ibs_recolher_account_id BIGINT REFERENCES public.accounting_accounts(id),
    ADD COLUMN IF NOT EXISTS cbs_vendas_account_id   BIGINT REFERENCES public.accounting_accounts(id),
    ADD COLUMN IF NOT EXISTS cbs_recolher_account_id BIGINT REFERENCES public.accounting_accounts(id);
