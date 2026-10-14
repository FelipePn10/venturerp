-- Clientes por empresa.
--
-- `customers` não tinha coluna de empresa: o mesmo cadastro aparecia para todas
-- as empresas do banco (GET /api/customers/{code} devolvia o mesmo registro em
-- duas empresas). Cada empresa passa a ter o seu cadastro, e o código do cliente
-- passa a ser único POR EMPRESA.
--
-- Backfill, sem quebrar o histórico:
--   * cada empresa que já usou o cliente em algum documento recebe o seu
--     cadastro, com o MESMO código — os documentos dela continuam apontando
--     para o código certo, agora dentro da própria empresa;
--   * cliente nunca usado vai para todas as empresas (nenhuma perde o que vê
--     hoje; quem não quiser, inativa);
--   * o registro original fica com a menor empresa da lista; as demais recebem
--     cópias, com endereços e contatos copiados;
--   * condição de pagamento e tabela de venda são cadastros por empresa: a
--     referência a um registro de OUTRA empresa é desfeita (nunca mistura).
--
-- Nenhuma FK aponta para customers(code); endereços e contatos apontam para o id.

ALTER TABLE public.customers ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);

-- Empresas que usaram cada código de cliente. Tabelas da convenção
-- enterprise_code são traduzidas para enterprise.id.
CREATE TEMP TABLE _uso_cliente (code BIGINT NOT NULL, enterprise_id BIGINT NOT NULL) ON COMMIT DROP;

INSERT INTO _uso_cliente (code, enterprise_id)
SELECT customer_code, enterprise_id FROM public.commercial_policies WHERE customer_code IS NOT NULL AND enterprise_id IS NOT NULL
UNION SELECT customer_code, enterprise_id FROM public.consumer_service_customer_contacts WHERE customer_code IS NOT NULL AND enterprise_id IS NOT NULL
UNION SELECT customer_code, enterprise_id FROM public.contribution_margin WHERE customer_code IS NOT NULL AND enterprise_id IS NOT NULL
UNION SELECT customer_code, enterprise_id FROM public.customer_material_remittances WHERE customer_code IS NOT NULL AND enterprise_id IS NOT NULL
UNION SELECT customer_code, enterprise_id FROM public.fiscal_exits WHERE customer_code IS NOT NULL AND enterprise_id IS NOT NULL
UNION SELECT customer_code, enterprise_id FROM public.lot_masks WHERE customer_code IS NOT NULL AND enterprise_id IS NOT NULL
UNION SELECT customer_code, enterprise_id FROM public.restrictions WHERE customer_code IS NOT NULL AND enterprise_id IS NOT NULL
UNION SELECT t.customer_code, e.id FROM public.sales_orders t JOIN public.enterprise e ON e.code = t.enterprise_code WHERE t.customer_code IS NOT NULL
UNION SELECT t.customer_code, e.id FROM public.sales_quotations t JOIN public.enterprise e ON e.code = t.enterprise_code WHERE t.customer_code IS NOT NULL
UNION SELECT t.customer_code, e.id FROM public.recurring_sales t JOIN public.enterprise e ON e.code = t.enterprise_code WHERE t.customer_code IS NOT NULL
UNION SELECT t.customer_code, e.id FROM public.consumer_service_calls t JOIN public.enterprise e ON e.code = t.enterprise_code WHERE t.customer_code IS NOT NULL
UNION SELECT t.customer_code, e.id FROM public.technical_assistance_calls t JOIN public.enterprise e ON e.code = t.enterprise_code WHERE t.customer_code IS NOT NULL
UNION SELECT t.customer_code, e.id FROM public.purchase_orders t JOIN public.enterprise e ON e.code = t.enterprise_code WHERE t.customer_code IS NOT NULL;

CREATE TEMP TABLE _destino_cliente ON COMMIT DROP AS
SELECT c.id AS customer_id, c.code, u.enterprise_id
  FROM public.customers c
  JOIN (SELECT DISTINCT code, enterprise_id FROM _uso_cliente) u ON u.code = c.code
UNION
SELECT c.id, c.code, e.id
  FROM public.customers c
 CROSS JOIN public.enterprise e
 WHERE NOT EXISTS (SELECT 1 FROM _uso_cliente u WHERE u.code = c.code);

CREATE TEMP TABLE _originais ON COMMIT DROP AS SELECT id FROM public.customers;

-- O original fica com a menor empresa do destino.
UPDATE public.customers c
   SET enterprise_id = d.dono
  FROM (SELECT customer_id, MIN(enterprise_id) AS dono FROM _destino_cliente GROUP BY customer_id) d
 WHERE d.customer_id = c.id AND c.enterprise_id IS NULL;

ALTER TABLE public.customers DROP CONSTRAINT IF EXISTS customers_code_key;

-- Cópias para as demais empresas, com o mesmo código.
INSERT INTO public.customers (
    code, corporate_code, is_corporate, name, trade_name, document_type, document_number,
    state_registration, municipal_registration, suframa_code, suframa_expiry,
    region_id, market_segment_id, customer_type_id, payment_condition_id, sales_table_id,
    carrier_id, carrier_group_id, invoice_type_id, tax_type_id, payment_cond_visibility,
    credit_limit, website, is_active, blocked, block_reason, created_at, created_by, updated_at,
    enterprise_id)
SELECT c.code, c.corporate_code, c.is_corporate, c.name, c.trade_name, c.document_type, c.document_number,
       c.state_registration, c.municipal_registration, c.suframa_code, c.suframa_expiry,
       c.region_id, c.market_segment_id, c.customer_type_id, c.payment_condition_id, c.sales_table_id,
       c.carrier_id, c.carrier_group_id, c.invoice_type_id, c.tax_type_id, c.payment_cond_visibility,
       c.credit_limit, c.website, c.is_active, c.blocked, c.block_reason, c.created_at, c.created_by, NOW(),
       d.enterprise_id
  FROM _destino_cliente d
  JOIN public.customers c ON c.id = d.customer_id
 WHERE d.enterprise_id <> c.enterprise_id;

-- Endereços e contatos acompanham cada cópia.
INSERT INTO public.customer_addresses (customer_id, address_type, zip_code, street, number, complement,
                                       neighborhood, city, uf, country, is_default, created_at)
SELECT novo.id, a.address_type, a.zip_code, a.street, a.number, a.complement,
       a.neighborhood, a.city, a.uf, a.country, a.is_default, a.created_at
  FROM public.customer_addresses a
  JOIN public.customers orig ON orig.id = a.customer_id AND orig.id IN (SELECT id FROM _originais)
  JOIN public.customers novo ON novo.code = orig.code AND novo.id NOT IN (SELECT id FROM _originais);

INSERT INTO public.customer_contacts (customer_id, contact_type_id, name, email, phone, mobile, position,
                                      is_primary, is_active, created_at)
SELECT novo.id, ct.contact_type_id, ct.name, ct.email, ct.phone, ct.mobile, ct.position,
       ct.is_primary, ct.is_active, ct.created_at
  FROM public.customer_contacts ct
  JOIN public.customers orig ON orig.id = ct.customer_id AND orig.id IN (SELECT id FROM _originais)
  JOIN public.customers novo ON novo.code = orig.code AND novo.id NOT IN (SELECT id FROM _originais);

-- Condição de pagamento e tabela de venda de OUTRA empresa: desfaz a referência.
UPDATE public.customers c SET payment_condition_id = NULL
  FROM public.payment_conditions pc
 WHERE pc.id = c.payment_condition_id AND pc.enterprise_id IS DISTINCT FROM c.enterprise_id;
UPDATE public.customers c SET sales_table_id = NULL
  FROM public.sales_tables st
 WHERE st.id = c.sales_table_id AND st.enterprise_id IS DISTINCT FROM c.enterprise_id;

ALTER TABLE public.customers ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.customers ADD CONSTRAINT customers_enterprise_code_key UNIQUE (enterprise_id, code);
DROP INDEX IF EXISTS public.ix_customers_document;
CREATE INDEX IF NOT EXISTS ix_customers_enterprise_document ON public.customers (enterprise_id, document_number);

-- Títulos a receber guardam o CÓDIGO do cliente (é por ele que a tela filtra e
-- a análise de crédito soma). A emissão da NF-e gravava o id do cadastro:
-- corrige pelos títulos que nasceram de nota.
UPDATE public.contas_receber cr
   SET cliente_id = fe.customer_code
  FROM public.fiscal_exits fe
 WHERE fe.id = cr.fiscal_exit_id AND fe.customer_code IS NOT NULL
   AND cr.cliente_id IS DISTINCT FROM fe.customer_code;
