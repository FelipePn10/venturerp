-- Dados do cadastro que a nota avulsa (VFIS0200) puxa sozinha: endereço e contato
-- do cliente, classificação fiscal (NCM) do item e preço em tabela de venda. Sem
-- eles, o demo não mostra o preenchimento automático. Só completa o que falta,
-- então pode rodar mais de uma vez. Endereços e contatos são fictícios.
\set ON_ERROR_STOP on
BEGIN;

INSERT INTO customer_addresses (customer_id, address_type, zip_code, street, number, neighborhood, city, uf, is_default)
SELECT c.id, 'ENTREGA',
       CASE WHEN c.code % 7 = 0 THEN '30130000' WHEN c.code % 5 = 0 THEN '80010000' ELSE '01310100' END,
       CASE WHEN c.code % 7 = 0 THEN 'AV AFONSO PENA' WHEN c.code % 5 = 0 THEN 'RUA XV DE NOVEMBRO' ELSE 'AV PAULISTA' END,
       (100 + c.code * 7)::text,
       CASE WHEN c.code % 7 = 0 THEN 'CENTRO' WHEN c.code % 5 = 0 THEN 'CENTRO' ELSE 'BELA VISTA' END,
       CASE WHEN c.code % 7 = 0 THEN 'BELO HORIZONTE' WHEN c.code % 5 = 0 THEN 'CURITIBA' ELSE 'SAO PAULO' END,
       CASE WHEN c.code % 7 = 0 THEN 'MG' WHEN c.code % 5 = 0 THEN 'PR' ELSE 'SP' END,
       true
FROM customers c
WHERE NOT EXISTS (SELECT 1 FROM customer_addresses a WHERE a.customer_id = c.id);

-- Um cliente com dois endereços, para mostrar a escolha do endereço da nota.
INSERT INTO customer_addresses (customer_id, address_type, zip_code, street, number, neighborhood, city, uf, is_default)
SELECT c.id, 'COBRANCA', '13010000', 'RUA BARAO DE JAGUARA', '500', 'CENTRO', 'CAMPINAS', 'SP', true
FROM customers c
WHERE c.code = 1 AND NOT EXISTS (SELECT 1 FROM customer_addresses a WHERE a.customer_id = c.id AND a.address_type = 'COBRANCA');

-- Os CNPJs semeados dos fornecedores não têm dígito verificador válido, e a nota de
-- entrada os recusa. Só recalcula os dois últimos dígitos: a raiz do número fica.
DO $$
DECLARE
  f RECORD; base TEXT; d1 INT; d2 INT; soma INT; i INT;
  pesos1 INT[] := ARRAY[5,4,3,2,9,8,7,6,5,4,3,2];
  pesos2 INT[] := ARRAY[6,5,4,3,2,9,8,7,6,5,4,3,2];
BEGIN
  FOR f IN SELECT id, document_number FROM suppliers WHERE length(regexp_replace(document_number, '\D', '', 'g')) = 14 LOOP
    base := left(regexp_replace(f.document_number, '\D', '', 'g'), 12);
    soma := 0;
    FOR i IN 1..12 LOOP soma := soma + substr(base, i, 1)::int * pesos1[i]; END LOOP;
    d1 := CASE WHEN soma % 11 < 2 THEN 0 ELSE 11 - soma % 11 END;
    soma := 0;
    FOR i IN 1..13 LOOP soma := soma + substr(base || d1, i, 1)::int * pesos2[i]; END LOOP;
    d2 := CASE WHEN soma % 11 < 2 THEN 0 ELSE 11 - soma % 11 END;
    UPDATE suppliers SET document_number = base || d1 || d2 WHERE id = f.id AND document_number <> base || d1 || d2;
  END LOOP;
END $$;

-- Os clientes também: a NF-e recusa destinatário com CNPJ inválido. Os rascunhos
-- acompanham; nota já autorizada fica como foi emitida.
DO $$
DECLARE
  f RECORD; base TEXT; d1 INT; d2 INT; soma INT; i INT;
  pesos1 INT[] := ARRAY[5,4,3,2,9,8,7,6,5,4,3,2];
  pesos2 INT[] := ARRAY[6,5,4,3,2,9,8,7,6,5,4,3,2];
BEGIN
  FOR f IN SELECT id, code, enterprise_id, document_number FROM customers WHERE length(regexp_replace(document_number, '\D', '', 'g')) = 14 LOOP
    base := left(regexp_replace(f.document_number, '\D', '', 'g'), 12);
    soma := 0;
    FOR i IN 1..12 LOOP soma := soma + substr(base, i, 1)::int * pesos1[i]; END LOOP;
    d1 := CASE WHEN soma % 11 < 2 THEN 0 ELSE 11 - soma % 11 END;
    soma := 0;
    FOR i IN 1..13 LOOP soma := soma + substr(base || d1, i, 1)::int * pesos2[i]; END LOOP;
    d2 := CASE WHEN soma % 11 < 2 THEN 0 ELSE 11 - soma % 11 END;
    UPDATE customers SET document_number = base || d1 || d2 WHERE id = f.id AND document_number <> base || d1 || d2;
    UPDATE fiscal_exits SET cnpj_destinatario = base || d1 || d2
     WHERE status = 'DRAFT' AND customer_code = f.code AND enterprise_id = f.enterprise_id;
  END LOOP;
END $$;

-- Fornecedores: a UF do endereço define o CFOP do lançamento manual de entrada (VFIS0210).
INSERT INTO supplier_addresses (supplier_id, address_type, zip_code, street, number, neighborhood, city, uf, is_default)
SELECT f.id, 'COMERCIAL',
       CASE WHEN f.code % 3 = 0 THEN '80010000' ELSE '01310100' END,
       CASE WHEN f.code % 3 = 0 THEN 'RUA XV DE NOVEMBRO' ELSE 'AV PAULISTA' END,
       (200 + f.code * 3)::text,
       CASE WHEN f.code % 3 = 0 THEN 'CENTRO' ELSE 'BELA VISTA' END,
       CASE WHEN f.code % 3 = 0 THEN 'CURITIBA' ELSE 'SAO PAULO' END,
       CASE WHEN f.code % 3 = 0 THEN 'PR' ELSE 'SP' END,
       true
FROM suppliers f
WHERE NOT EXISTS (SELECT 1 FROM supplier_addresses a WHERE a.supplier_id = f.id);

INSERT INTO customer_contacts (customer_id, name, email, phone, is_primary, is_active)
SELECT c.id, 'Compras', 'compras' || c.code || '@cliente.demo', '1130000' || lpad(c.code::text, 3, '0'), true, true
FROM customers c
WHERE NOT EXISTS (SELECT 1 FROM customer_contacts t WHERE t.customer_id = c.id);

UPDATE items SET accounting_sale_fiscal_classification_code = CASE
    WHEN pdm_description_technique ILIKE '%estrutura%' THEN '1'
    WHEN pdm_description_technique ILIKE '%chapa%' THEN '3'
    WHEN pdm_description_technique ILIKE '%parafuso%' THEN '4'
    ELSE '2' END,
  accounting_origin = COALESCE(accounting_origin, 0)
WHERE COALESCE(accounting_sale_fiscal_classification_code, '') = '' AND enterprise_id = 1;

INSERT INTO sales_tables (code, description, enterprise_id)
SELECT 900, 'Tabela padrão', 1 WHERE NOT EXISTS (SELECT 1 FROM sales_tables WHERE code = 900 AND enterprise_id = 1);

INSERT INTO sales_table_prices (sales_table_id, item_code, price, ume)
SELECT t.id, i.business_code, 50 + (i.code % 40) * 12.5, i.warehouse_unit_of_measurement::text
FROM items i JOIN sales_tables t ON t.code = 900 AND t.enterprise_id = i.enterprise_id
WHERE i.enterprise_id = 1
ON CONFLICT (sales_table_id, item_code) DO NOTHING;

-- Operações de entrada (TES): sem elas os campos de VFIS0210 e VFIS0360 ficam vazios.
INSERT INTO entry_operation_types
  (enterprise_id, code, description, nature_operation, movimenta_estoque, gera_financeiro,
   credita_icms, credita_ipi, credita_pis_cofins, created_by)
SELECT 1, v.code, v.description, v.natureza, v.estoque, v.financeiro, v.icms, v.ipi, v.piscofins,
       (SELECT id FROM users WHERE email = 'admin@panossoerp.demo')
FROM (VALUES
  (1::bigint, 'Compra para industrialização', '1101', true,  true,  true,  true,  true),
  (2,         'Compra para comercialização',  '1102', true,  true,  true,  false, true),
  (3,         'Compra para uso e consumo',    '1556', true,  true,  false, false, false),
  (4,         'Compra para o ativo imobilizado', '1551', true, true, false, false, false),
  (5,         'Serviço tomado',               '1933', false, true,  false, false, true),
  (6,         'Bonificação recebida',         '1910', true,  false, true,  true,  true),
  (7,         'Remessa de material do cliente para beneficiamento', '1901', false, false, false, false, false),
  (8,         'Retorno de conserto',          '1916', false, false, false, false, false),
  (9,         'Devolução de venda',           '1202', true,  false, true,  true,  true)
) AS v(code, description, natureza, estoque, financeiro, icms, ipi, piscofins)
ON CONFLICT (enterprise_id, code) DO NOTHING;

INSERT INTO supplier_types (enterprise_id, code, description)
SELECT 1, v.code, v.description
FROM (VALUES (1::bigint, 'Indústria'), (2, 'Distribuidor'), (3, 'Prestador de serviço')) AS v(code, description)
ON CONFLICT (enterprise_id, code) DO NOTHING;

-- Grupos de estado que a operação de entrada pode exigir.
INSERT INTO state_groups (enterprise_id, code, description, created_by)
SELECT 1, 1, 'Sudeste e Sul', (SELECT id FROM users WHERE email = 'admin@panossoerp.demo')
ON CONFLICT (enterprise_id, code) DO NOTHING;
INSERT INTO state_group_ufs (enterprise_id, state_group_code, uf)
SELECT 1, 1, u FROM unnest(ARRAY['SP','RJ','MG','ES','PR','SC','RS']) AS u
ON CONFLICT (enterprise_id, state_group_code, uf) DO NOTHING;

-- Plano de contas para classificar os itens da nota e distribuir as parcelas.
INSERT INTO plano_contas (enterprise_id, codigo, descricao, tipo, natureza, nivel)
SELECT 1, v.codigo, v.descricao, 'DESPESA', 'DEBITO', 2
FROM (VALUES
  ('4.1.01', 'Matéria-prima'),
  ('4.1.02', 'Material de embalagem'),
  ('4.2.01', 'Material de uso e consumo'),
  ('4.2.02', 'Equipamentos de proteção (EPI)'),
  ('4.3.01', 'Serviços de terceiros'),
  ('4.3.02', 'Fretes e carretos')
) AS v(codigo, descricao)
ON CONFLICT (enterprise_id, codigo) DO NOTHING;

-- As condições semeadas não tinham parcelas: "28/56 dias" valia como à vista.
INSERT INTO payment_condition_installments (payment_condition_id, installment_number, due_days, base_event)
SELECT pc.id, v.numero, v.dias, 'EMISSAO'::payment_base_event_enum
FROM payment_conditions pc
JOIN (VALUES (2::bigint, 1::smallint, 28::smallint), (3, 1, 28), (3, 2, 56), (4, 1, 30), (4, 2, 60), (4, 3, 90))
  AS v(cond, numero, dias) ON v.cond = pc.code
WHERE pc.enterprise_id = 1
  AND NOT EXISTS (SELECT 1 FROM payment_condition_installments i WHERE i.payment_condition_id = pc.id);

-- Percentual combinado com parcelas: entrada + parcelas e parcelas + saldo no final.
INSERT INTO payment_conditions (enterprise_id, code, description, average_term)
SELECT 1, v.code, v.description, v.prazo
FROM (VALUES (5::bigint, '30% de entrada + 28/56 dias', 30::smallint),
             (6::bigint, '3x em 30/60/90 + 40% em 120 dias', 84::smallint)) AS v(code, description, prazo)
WHERE NOT EXISTS (SELECT 1 FROM payment_conditions p WHERE p.code = v.code);

INSERT INTO payment_condition_installments (payment_condition_id, installment_number, due_days, percentage, base_event, description)
SELECT pc.id, v.numero, v.dias, v.pct, v.evento::payment_base_event_enum, v.descr
FROM payment_conditions pc
JOIN (VALUES
  (5::bigint, 1::smallint, 0::smallint, 30::numeric, 'ENTRADA', 'Entrada'),
  (5, 2, 28, 35, 'EMISSAO', NULL),
  (5, 3, 56, 35, 'EMISSAO', NULL),
  (6, 1, 30, 20, 'EMISSAO', NULL),
  (6, 2, 60, 20, 'EMISSAO', NULL),
  (6, 3, 90, 20, 'EMISSAO', NULL),
  (6, 4, 120, 40, 'EMISSAO', 'Saldo final')
) AS v(cond, numero, dias, pct, evento, descr) ON v.cond = pc.code
WHERE pc.enterprise_id = 1
  AND NOT EXISTS (SELECT 1 FROM payment_condition_installments i WHERE i.payment_condition_id = pc.id);

-- Vínculo item × fornecedor: o código que o fornecedor usa para o nosso item, tirado do
-- que cada fornecedor já vendeu nos pedidos. É o que concilia a nota sozinho.
INSERT INTO item_preferred_suppliers (enterprise_id, item_code, supplier_code, mask, ranking, supplier_item_code, supplier_description, uom, created_by)
SELECT DISTINCT ON (po.supplier_code, i.code)
       1, i.code, po.supplier_code, '', 1,
       'F' || po.supplier_code || '-' || i.code,
       upper(left(COALESCE(NULLIF(i.pdm_description_technique, ''), i.business_code), 200)),
       i.warehouse_unit_of_measurement::text,
       (SELECT id FROM users WHERE email = 'admin@panossoerp.demo')
FROM purchase_order_items poi
JOIN purchase_orders po ON po.code = poi.purchase_order_code
JOIN items i ON i.code = poi.item_code AND i.enterprise_id = 1
WHERE po.supplier_code IS NOT NULL AND po.enterprise_code = 1
ON CONFLICT (enterprise_id, item_code, supplier_code, mask) WHERE enterprise_id IS NOT NULL DO NOTHING;

-- Tipos de nota de saída (VCLI0530): o que a nota faz ao ser autorizada. Sem eles a
-- lista "Tipo de nota" da VFIS0200 fica vazia. Quem já existe fica como está.
INSERT INTO invoice_types (enterprise_id, code, description, type, stock_movement, icms_type,
                           generates_revenue, updates_inventory, generates_financial_title, baixa_pedido,
                           calc_pis_cofins, calc_substitution_tax, calc_icms_deferral, calc_difal, requires_sales_order)
VALUES
  (1, 1, 'VENDA DE MERCADORIA',        'VENDA',                  'ATUALIZA',     'TRIBUTADO', true,  true,  true,  true,  true,  true,  true,  true,  false),
  (1, 2, 'BONIFICACAO',                'OUTROS',                 'ATUALIZA',     'TRIBUTADO', false, true,  false, false, true,  false, false, true,  false),
  (1, 3, 'REMESSA PARA DEMONSTRACAO',  'DEMONSTRACAO',           'ATUALIZA',     'OUTROS',    false, true,  false, false, false, false, false, false, false),
  (1, 4, 'SIMPLES REMESSA',            'SIMPLES_REMESSA',        'ATUALIZA',     'OUTROS',    false, true,  false, false, false, false, false, false, false),
  (1, 5, 'TRANSFERENCIA ENTRE FILIAIS','TRANSFERENCIA',          'ATUALIZA',     'TRIBUTADO', false, true,  false, false, false, false, false, false, false),
  (1, 6, 'DEVOLUCAO DE COMPRA',        'DEVOLUCAO',              'ATUALIZA',     'TRIBUTADO', false, true,  false, false, false, false, false, false, false),
  (1, 7, 'RETORNO DE BENEFICIAMENTO',  'RETORNO_BENEFICIAMENTO', 'NAO_ATUALIZA', 'OUTROS',    false, false, false, false, false, false, false, false, false),
  (1, 8, 'PRESTACAO DE SERVICOS',      'PRESTACAO_SERVICOS',     'NAO_ATUALIZA', 'OUTROS',    true,  false, true,  true,  true,  false, false, false, false)
ON CONFLICT (enterprise_id, code) DO NOTHING;

COMMIT;
