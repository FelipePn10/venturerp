-- Adiantamentos passam a ser contabilizados (VFIN0600).
-- Pago a fornecedor: D Adiantamentos a fornecedores (ativo) / C Banco.
-- Recebido de cliente: D Banco / C Adiantamentos de clientes (passivo).
-- A aplicação num título transfere o saldo para Fornecedores/Clientes (ou para
-- a despesa/receita do título avulso), e o cancelamento estorna a entrada.
ALTER TABLE public.accounting_posting_params
    ADD COLUMN IF NOT EXISTS adiantamento_fornecedores_account_id BIGINT REFERENCES public.accounting_accounts(id),
    ADD COLUMN IF NOT EXISTS adiantamento_clientes_account_id     BIGINT REFERENCES public.accounting_accounts(id);
