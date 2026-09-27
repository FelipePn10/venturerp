-- Endereço do destinatário na NF-e de saída.
--
-- A nota guardava apenas CNPJ, razão social, IE e UF do destinatário. O layout
-- da NF-e exige o endereço completo (logradouro, número, bairro, município,
-- código IBGE do município e CEP): sem ele a SEFAZ rejeita a autorização.
-- Como nenhuma nota tinha onde guardar esses dados, a primeira emissão real
-- falharia por campo obrigatório ausente.
--
-- As colunas nascem nulas: notas já existentes (todas em rascunho ou de
-- simulação) continuam válidas e a prévia acusa o que falta antes de emitir.
ALTER TABLE public.fiscal_exits
    ADD COLUMN IF NOT EXISTS dest_logradouro       TEXT,
    ADD COLUMN IF NOT EXISTS dest_numero           TEXT,
    ADD COLUMN IF NOT EXISTS dest_complemento      TEXT,
    ADD COLUMN IF NOT EXISTS dest_bairro           TEXT,
    ADD COLUMN IF NOT EXISTS dest_municipio        TEXT,
    ADD COLUMN IF NOT EXISTS dest_codigo_municipio TEXT,
    ADD COLUMN IF NOT EXISTS dest_cep              TEXT,
    ADD COLUMN IF NOT EXISTS dest_email            TEXT,
    ADD COLUMN IF NOT EXISTS dest_telefone         TEXT,
    ADD COLUMN IF NOT EXISTS customer_code         BIGINT;

COMMENT ON COLUMN public.fiscal_exits.dest_codigo_municipio IS 'Código IBGE do município do destinatário (7 dígitos).';
COMMENT ON COLUMN public.fiscal_exits.customer_code IS 'Cliente de onde o endereço e a condição de pagamento foram resolvidos.';

CREATE INDEX IF NOT EXISTS idx_fiscal_exits_customer ON public.fiscal_exits(enterprise_id, customer_code);
