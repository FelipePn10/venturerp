-- Representantes por empresa.
--
-- O representante era um cadastro compartilhado, ligado às empresas por
-- representative_enterprises. A leitura filtrava pelo vínculo, mas as escritas
-- dos filhos não: pelo código, uma empresa se vinculava ao representante da
-- outra e incluía telefones, e-mails, contatos, regiões e contabilização nele.
--
-- O representante passa a ter UMA empresa dona (enterprise_code, convenção
-- comercial). O código continua único no banco (como item e fornecedor):
--   * representante usado por várias empresas fica com a menor delas; cada uma
--     das outras recebe uma CÓPIA com código novo (telefones, e-mails, contatos,
--     endereços e interesses copiados), e os dados e documentos dela — vínculo,
--     contabilização, regiões, segmentos, planos de venda, pedidos, orçamentos,
--     comissões, venda recorrente — passam para a cópia;
--   * representante sem uso nenhum (não era visível a ninguém) fica com a menor
--     empresa.

ALTER TABLE public.representatives ADD COLUMN IF NOT EXISTS enterprise_code BIGINT;

CREATE TEMP TABLE _uso_rep (code BIGINT NOT NULL, enterprise_code BIGINT NOT NULL) ON COMMIT DROP;
INSERT INTO _uso_rep
SELECT representative_code, enterprise_code FROM public.representative_enterprises WHERE enterprise_code IS NOT NULL
UNION SELECT representative_code, enterprise_code FROM public.representative_accounting WHERE enterprise_code IS NOT NULL
UNION SELECT representative_code, enterprise_code FROM public.representative_regions WHERE enterprise_code IS NOT NULL
UNION SELECT representative_code, enterprise_code FROM public.representative_segments WHERE enterprise_code IS NOT NULL
UNION SELECT representative_code, enterprise_code FROM public.representative_sales_plans WHERE enterprise_code IS NOT NULL
UNION SELECT representative_code, enterprise_code FROM public.sales_orders WHERE representative_code IS NOT NULL
UNION SELECT representative_code, enterprise_code FROM public.sales_order_representatives WHERE representative_code IS NOT NULL
UNION SELECT representative_code, enterprise_code FROM public.sales_quotations WHERE representative_code IS NOT NULL
UNION SELECT representative_code, enterprise_code FROM public.sales_quotation_representatives WHERE representative_code IS NOT NULL
UNION SELECT generic_representative_code, enterprise_code FROM public.recurring_sales_parameters WHERE generic_representative_code IS NOT NULL
UNION SELECT rr.representative_code, rs.enterprise_code FROM public.recurring_sales_representatives rr JOIN public.recurring_sales rs ON rs.code = rr.recurring_sale_code
UNION SELECT l.representative_code, e.code FROM public.commercial_commission_ledger l JOIN public.enterprise e ON e.id = l.enterprise_id WHERE l.representative_code IS NOT NULL;
DELETE FROM _uso_rep u WHERE NOT EXISTS (SELECT 1 FROM public.enterprise e WHERE e.code = u.enterprise_code)
                          OR NOT EXISTS (SELECT 1 FROM public.representatives r WHERE r.code = u.code);

-- Dono: a empresa de menor id entre as que usam; sem uso, a menor empresa.
UPDATE public.representatives r SET enterprise_code = d.dono
  FROM (SELECT DISTINCT ON (u.code) u.code, u.enterprise_code AS dono
          FROM _uso_rep u JOIN public.enterprise e ON e.code = u.enterprise_code
         ORDER BY u.code, e.id) d
 WHERE d.code = r.code;
UPDATE public.representatives SET enterprise_code = (SELECT code FROM public.enterprise ORDER BY id LIMIT 1)
 WHERE enterprise_code IS NULL;

-- Código novo para cada empresa que não é a dona.
CREATE TEMP TABLE _mapa_rep ON COMMIT DROP AS
SELECT DISTINCT u.code AS antigo, u.enterprise_code, 0::BIGINT AS novo
  FROM _uso_rep u JOIN public.representatives r ON r.code = u.code
 WHERE u.enterprise_code <> r.enterprise_code;
UPDATE _mapa_rep SET novo = nextval('public.representatives_code_seq');

DO $$
DECLARE cols TEXT; rcols TEXT; t TEXT;
BEGIN
    SELECT string_agg(quote_ident(column_name), ',' ORDER BY ordinal_position),
           string_agg('r.' || quote_ident(column_name), ',' ORDER BY ordinal_position)
      INTO cols, rcols
      FROM information_schema.columns
     WHERE table_schema = 'public' AND table_name = 'representatives' AND column_name NOT IN ('code', 'enterprise_code');
    EXECUTE format('INSERT INTO public.representatives (code, enterprise_code, %s)
                    SELECT m.novo, m.enterprise_code, %s FROM _mapa_rep m JOIN public.representatives r ON r.code = m.antigo',
                   cols, rcols);

    -- Filhos sem empresa: copiados para cada cópia.
    FOREACH t IN ARRAY ARRAY['representative_phones', 'representative_emails', 'representative_correspondence_addresses',
                             'representative_contacts', 'representative_interests'] LOOP
        SELECT string_agg(quote_ident(column_name), ',' ORDER BY ordinal_position),
               string_agg('c.' || quote_ident(column_name), ',' ORDER BY ordinal_position)
          INTO cols, rcols
          FROM information_schema.columns
         WHERE table_schema = 'public' AND table_name = t AND column_name NOT IN ('id', 'representative_code');
        EXECUTE format('INSERT INTO public.%1$I (representative_code, %2$s)
                        SELECT m.novo, %3$s FROM _mapa_rep m JOIN public.%1$I c ON c.representative_code = m.antigo',
                       t, cols, rcols);
    END LOOP;

    -- Dados e documentos da empresa passam para a cópia dela.
    FOREACH t IN ARRAY ARRAY['representative_enterprises', 'representative_accounting', 'representative_regions',
                             'representative_segments', 'representative_sales_plans', 'sales_orders',
                             'sales_order_representatives', 'sales_quotations', 'sales_quotation_representatives'] LOOP
        EXECUTE format('UPDATE public.%I x SET representative_code = m.novo FROM _mapa_rep m
                         WHERE m.antigo = x.representative_code AND m.enterprise_code = x.enterprise_code', t);
    END LOOP;
END $$;

UPDATE public.recurring_sales_parameters x SET generic_representative_code = m.novo
  FROM _mapa_rep m WHERE m.antigo = x.generic_representative_code AND m.enterprise_code = x.enterprise_code;
UPDATE public.recurring_sales_representatives x SET representative_code = m.novo
  FROM _mapa_rep m, public.recurring_sales rs
 WHERE rs.code = x.recurring_sale_code AND m.antigo = x.representative_code AND m.enterprise_code = rs.enterprise_code;
UPDATE public.commercial_commission_ledger x SET representative_code = m.novo
  FROM _mapa_rep m, public.enterprise e
 WHERE e.id = x.enterprise_id AND m.antigo = x.representative_code AND m.enterprise_code = e.code;

ALTER TABLE public.representatives ALTER COLUMN enterprise_code SET NOT NULL;
CREATE INDEX IF NOT EXISTS ix_representatives_enterprise ON public.representatives (enterprise_code, code);
