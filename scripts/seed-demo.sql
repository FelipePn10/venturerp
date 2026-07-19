-- ============================================================================
-- PanossoERP — SEED DE APRESENTAÇÃO (demo)
-- ----------------------------------------------------------------------------
-- Popula o banco de DEMONSTRAÇÃO com ~1 ano de operação fictícia de uma
-- metalúrgica: catálogo de produtos, clientes, fornecedores, estoque, pedidos
-- de venda/compra, ordens de fabricação, notas fiscais e financeiro (AR/AP).
--
-- Idempotente: faz TRUNCATE ... RESTART IDENTITY das tabelas que popula antes
-- de inserir. Seguro para reexecutar.
--
-- Uso:
--   make demo-seed
--   (ou) psql "<DATABASE_URL_DEMO>" -v ON_ERROR_STOP=1 -f scripts/seed-demo.sql
--
-- Login gerado: admin@panossoerp.demo / senha fornecida por DEMO_ADMIN_PASSWORD.
-- ============================================================================

\set ON_ERROR_STOP on
\set ADMIN '00000000-0000-0000-0000-000000000001'
\set START_TS '2025-06-15 08:00:00-03'

BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;
SELECT setseed(0.4242);

-- ─── Reset ───────────────────────────────────────────────────────────────────
TRUNCATE TABLE
  users, enterprise, warehouse,
  condicoes_pagamento, payment_conditions, contas_bancarias, cost_centers, fiscal_classifications,
  employees, machine_types, machines, operations, manufacturing_routes,
  items, products, item_structures, stock_balances, stock_movements,
  customers, suppliers,
  sales_orders, sales_order_items, sales_order_sequences,
  purchase_orders, purchase_order_items, purchase_order_sequences,
  production_orders, fiscal_exits, fiscal_exit_items,
  contas_receber, contas_pagar
  RESTART IDENTITY CASCADE;

-- ─── 1. Usuário admin ────────────────────────────────────────────────────────
INSERT INTO users (id, name, email, password, role, created_at, updated_at)
VALUES (:'ADMIN'::uuid, 'Administrador Demo', 'admin@panossoerp.demo',
        crypt(:'admin_password', gen_salt('bf', 12)), 'ADMIN', :'START_TS', :'START_TS');

-- ─── 2. Empresa ──────────────────────────────────────────────────────────────
INSERT INTO enterprise (id, code, name, created_at, created_by)
VALUES (1, 1, 'Panosso Metalurgia Ltda', :'START_TS', :'ADMIN'::uuid);

INSERT INTO user_enterprises (user_id, enterprise_id, role)
VALUES (:'ADMIN'::uuid, 1, 'ADMIN');

-- ─── 3. Depósitos (warehouse) ────────────────────────────────────────────────
INSERT INTO warehouse (id, code, description, created_at, created_by, location, type, disposition, reservations_allowed) VALUES
 (1, '1', 'Almoxarifado de Matéria-Prima', :'START_TS', :'ADMIN'::uuid, 'NORMAL', 'INTERNO', true, true),
 (2, '2', 'Depósito de Produtos Acabados', :'START_TS', :'ADMIN'::uuid, 'NORMAL', 'INTERNO', true, true),
 (3, '3', 'Linha de Produção',            :'START_TS', :'ADMIN'::uuid, 'LINHA_DE_PRODUCAO', 'INTERNO', true, false),
 (4, '4', 'Depósito de Rejeição',         :'START_TS', :'ADMIN'::uuid, 'NORMAL', 'REJEICAO', false, false);

-- ─── 4. Cadastros de apoio ───────────────────────────────────────────────────
INSERT INTO condicoes_pagamento (id, nome, parcelas, ativo, created_at, updated_at) VALUES
 (1, 'À vista',            '[{"dias":0,"percentual":100}]', true, :'START_TS', :'START_TS'),
 (2, '28 dias',            '[{"dias":28,"percentual":100}]', true, :'START_TS', :'START_TS'),
 (3, '28/56 dias',         '[{"dias":28,"percentual":50},{"dias":56,"percentual":50}]', true, :'START_TS', :'START_TS'),
 (4, '30/60/90 dias',      '[{"dias":30,"percentual":34},{"dias":60,"percentual":33},{"dias":90,"percentual":33}]', true, :'START_TS', :'START_TS');

-- Condições de pagamento (cadastro usado por clientes/fornecedores — FK)
INSERT INTO payment_conditions (id, code, description, analysis_type, parcel_start, average_term, is_at_sight, is_active, created_at) VALUES
 (1, 1, 'À vista',        'LIBERA_SEM_ANALISE', 'EMISSAO', 0,  true,  true, :'START_TS'),
 (2, 2, '28 dias',        'LIBERA_SEM_ANALISE', 'EMISSAO', 28, false, true, :'START_TS'),
 (3, 3, '28/56 dias',     'SEMPRE_ANALISA',     'EMISSAO', 42, false, true, :'START_TS'),
 (4, 4, '30/60/90 dias',  'SEMPRE_ANALISA',     'EMISSAO', 60, false, true, :'START_TS');

INSERT INTO contas_bancarias (id, banco, agencia, conta, digito, descricao, titular, saldo_inicial, is_active, created_at, updated_at, created_by) VALUES
 (1, '341', '1234', '56789',  '0', 'Itaú — Conta Movimento', 'Panosso Metalurgia Ltda', 250000.00, true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (2, '001', '4567', '11223',  '5', 'Banco do Brasil — Conta Pagamentos', 'Panosso Metalurgia Ltda', 80000.00, true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (3, '237', '8899', '99887',  '1', 'Bradesco — Conta Recebimentos', 'Panosso Metalurgia Ltda', 120000.00, true, :'START_TS', :'START_TS', :'ADMIN'::uuid);

INSERT INTO cost_centers (id, code, description, type, start_date, is_active, created_at, updated_at, created_by) VALUES
 (1, 100, 'Produção — Corte e Dobra',  'PRODUCTIVE',     '2025-06-01', true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (2, 200, 'Produção — Soldagem',       'PRODUCTIVE',     '2025-06-01', true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (3, 300, 'Produção — Acabamento',     'PRODUCTIVE',     '2025-06-01', true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (4, 400, 'Manutenção',                'AUXILIARY',      '2025-06-01', true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (5, 500, 'Administrativo',            'ADMINISTRATIVE', '2025-06-01', true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (6, 600, 'Comercial',                 'COMMERCIAL',     '2025-06-01', true, :'START_TS', :'START_TS', :'ADMIN'::uuid);

INSERT INTO fiscal_classifications (id, code, description, ncm, ipi_rate, pis_rate, cofins_rate, created_at, updated_at, created_by) VALUES
 (1, 1, 'Estruturas metálicas',          '73089000', 5.0, 0.65, 3.0, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (2, 2, 'Artefatos de ferro/aço',        '73269000', 5.0, 0.65, 3.0, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (3, 3, 'Chapas de aço laminadas',       '72104900', 0.0, 1.65, 7.6, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (4, 4, 'Parafusos e fixadores',         '73181500', 5.0, 1.65, 7.6, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (5, 5, 'Serviços de industrialização',  NULL,       0.0, 0.65, 3.0, :'START_TS', :'START_TS', :'ADMIN'::uuid);

INSERT INTO employees (id, code, name, situation, role, created_at, updated_at, created_by) VALUES
 (1, 1001, 'Carlos Henrique Souza',  'ACTIVE', 'Operador de Produção', :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (2, 1002, 'Marcos Antônio Lima',    'ACTIVE', 'Soldador',             :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (3, 1003, 'Roberto Carlos Dias',    'ACTIVE', 'Operador CNC',         :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (4, 1004, 'João Pedro Alves',       'ACTIVE', 'Montador',             :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (5, 1005, 'Fernanda Oliveira',      'ACTIVE', 'Planejadora (PCP)',    :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (6, 1006, 'Ana Paula Martins',      'ACTIVE', 'Compradora',           :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (7, 1007, 'Ricardo Gomes',          'ACTIVE', 'Vendedor',             :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (8, 1008, 'Patrícia Ferreira',      'ACTIVE', 'Vendedora',            :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (9, 1009, 'Eduardo Nunes',          'ACTIVE', 'Inspetor de Qualidade',:'START_TS', :'START_TS', :'ADMIN'::uuid),
 (10,1010, 'Juliana Castro',         'ACTIVE', 'Analista Financeiro',  :'START_TS', :'START_TS', :'ADMIN'::uuid);

INSERT INTO machine_types (id, code, name, type, setup_time, requires_operator, is_active, created_at, updated_at, created_by) VALUES
 (1, 1, 'Corte a Laser',     'CUT',      15, true,  true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (2, 2, 'Dobradeira CNC',    'BEND',     20, true,  true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (3, 3, 'Solda MIG/MAG',     'WELD',     10, true,  true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (4, 4, 'Centro de Usinagem','MILL',     30, true,  true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (5, 5, 'Cabine de Pintura', 'PAINT',    25, true,  true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (6, 6, 'Prensa Hidráulica', 'PRESS',    12, true,  true, :'START_TS', :'START_TS', :'ADMIN'::uuid);

INSERT INTO machines (id, code, name, capacity, efficiency_rate, machine_type_code, cost_center_code, capacity_unit, capacity_period, is_active, created_at, updated_at, created_by) VALUES
 (1, 1001, 'Laser Fibra 3kW #1',   120, 0.92, 1, 100, 'CHAPAS', 'DIA', true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (2, 1002, 'Laser Fibra 3kW #2',   120, 0.90, 1, 100, 'CHAPAS', 'DIA', true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (3, 1003, 'Dobradeira 100t',       80, 0.88, 2, 100, 'PEÇAS',  'DIA', true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (4, 1004, 'Solda MIG Robô #1',     60, 0.95, 3, 200, 'PEÇAS',  'DIA', true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (5, 1005, 'Solda MIG Manual #1',   40, 0.85, 3, 200, 'PEÇAS',  'DIA', true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (6, 1006, 'Centro Usinagem VMC',   50, 0.90, 4, 100, 'PEÇAS',  'DIA', true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (7, 1007, 'Cabine Pintura Pó',    100, 0.93, 5, 300, 'PEÇAS',  'DIA', true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (8, 1008, 'Prensa 250t',           70, 0.87, 6, 100, 'PEÇAS',  'DIA', true, :'START_TS', :'START_TS', :'ADMIN'::uuid);

INSERT INTO operations (id, code, name, description, origin, situation, standard_time, setup_time, is_active, created_at, updated_at, created_by) VALUES
 (1, 10, 'Corte a Laser',      'Corte de chapas em laser de fibra', 'INTERNA', 'APROVADA', 0.083, 0.250, true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (2, 20, 'Dobra',              'Dobra em dobradeira CNC',           'INTERNA', 'APROVADA', 0.066, 0.333, true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (3, 30, 'Soldagem MIG',       'Soldagem MIG/MAG',                  'INTERNA', 'APROVADA', 0.167, 0.250, true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (4, 40, 'Usinagem',           'Usinagem CNC',                      'INTERNA', 'APROVADA', 0.250, 0.500, true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (5, 50, 'Pintura Eletrostática','Pintura a pó',                    'INTERNA', 'APROVADA', 0.100, 0.416, true, :'START_TS', :'START_TS', :'ADMIN'::uuid),
 (6, 60, 'Montagem',           'Montagem final',                    'INTERNA', 'APROVADA', 0.200, 0.166, true, :'START_TS', :'START_TS', :'ADMIN'::uuid);

-- ─── 5. Catálogo de itens (temp) ─────────────────────────────────────────────
CREATE TEMP TABLE item_catalog (
  code bigint PRIMARY KEY,
  kind text NOT NULL,
  name text NOT NULL,
  uom  text NOT NULL,
  unit_cost numeric(15,2) NOT NULL,
  sale_price numeric(15,2) NOT NULL,
  ncm text,
  weight numeric(15,4) NOT NULL
);

-- 50 produtos acabados (10001..10050)
INSERT INTO item_catalog (code, kind, name, uom, unit_cost, sale_price, ncm, weight)
SELECT 10000 + i, 'FINISHED',
  (ARRAY['Suporte Soldado','Estrutura Metálica','Chassi Industrial','Bandeja Porta-Cabos',
         'Mão Francesa Reforçada','Grade de Proteção','Caixa de Comando','Perfil U Dobrado',
         'Cantoneira Montada','Flange Usinada','Eixo Torneado','Engrenagem Reta',
         'Polia em V','Carrinho Industrial','Mesa de Apoio'])[1 + (i % 15)]
    || ' Mod. ' || lpad(i::text, 3, '0'),
  'UN', c.cost, round(c.cost * 1.6, 2), '73269000', round((2 + random()*45)::numeric, 4)
FROM generate_series(1, 50) i
CROSS JOIN LATERAL (SELECT round((80 + random()*520)::numeric, 2) AS cost, i AS k) c;

-- 80 matérias-primas / componentes (20001..20080)
INSERT INTO item_catalog (code, kind, name, uom, unit_cost, sale_price, ncm, weight)
SELECT 20000 + i, 'RAW',
  (ARRAY['Chapa de Aço','Barra Redonda','Tubo Quadrado','Cantoneira','Parafuso Sextavado',
         'Porca Sextavada','Arruela Lisa','Eletrodo','Tinta a Pó','Disco de Corte',
         'Perfil U','Vergalhão','Arame Solda MIG','Chapa Galvanizada','Tarugo de Alumínio'])[1 + (i % 15)]
    || ' ' || (ARRAY['3mm','1/2"','40x40','#8','M8x30','M8','3/16"','6013','RAL9005','4.1/2"'])[1 + (i % 10)],
  (ARRAY['KG','KG','M','UN','UN'])[1 + (i % 5)],
  c.cost, round(c.cost * 1.3, 2), '72104900', round((0.1 + random()*8)::numeric, 4)
FROM generate_series(1, 80) i
CROSS JOIN LATERAL (SELECT round((2 + random()*58)::numeric, 2) AS cost, i AS k) c;

-- 15 serviços (30001..30015)
INSERT INTO item_catalog (code, kind, name, uom, unit_cost, sale_price, ncm, weight)
SELECT 30000 + i, 'SERVICE',
  (ARRAY['Serviço de Corte a Laser','Dobra CNC','Pintura Eletrostática','Galvanização',
         'Usinagem CNC','Solda Especializada','Tratamento Térmico','Jateamento',
         'Montagem Industrial','Inspeção Dimensional'])[1 + (i % 10)],
  'UN', c.cost, round(c.cost * 1.5, 2), NULL, 0
FROM generate_series(1, 15) i
CROSS JOIN LATERAL (SELECT round((50 + random()*250)::numeric, 2) AS cost, i AS k) c;

-- items (a partir do catálogo)
INSERT INTO items (id, warehouse_code, code, health, created_by, created_at, nature, situation,
  pdm_group_code, pdm_modifier_code, pdm_attributes, pdm_description_technique,
  warehouse_unit_of_measurement, warehouse_automatic_low, warehouse_minimum_stock,
  engineering_weight, engineering_type, engineering_type_struct, engineering_oem,
  planning_type_mrp, planning_llc, planning_ghost, supplies_type_of_use)
SELECT ic.code,
  CASE WHEN ic.kind = 'FINISHED' THEN 2 ELSE 1 END,
  ic.code, 'ATIVO', :'ADMIN'::uuid, :'START_TS', 2, 0,
  10, 0, '[]'::jsonb, ic.name,
  ic.uom::unit_of_measurement_enum, false, 20,
  jsonb_build_object('gross', ic.weight, 'net', round(ic.weight*0.95, 4), 'unit', 'KG'),
  CASE ic.kind WHEN 'FINISHED' THEN 2 WHEN 'SERVICE' THEN 4 ELSE 0 END,
  CASE ic.kind WHEN 'FINISHED' THEN 0 ELSE 1 END,
  false,
  CASE ic.kind WHEN 'FINISHED' THEN 0 ELSE 2 END,
  CASE ic.kind WHEN 'FINISHED' THEN 0 ELSE 1 END,
  false,
  CASE ic.kind WHEN 'FINISHED' THEN 1 WHEN 'SERVICE' THEN 3 ELSE 0 END
FROM item_catalog ic;

-- products (espelho p/ módulos que usam a tabela products)
INSERT INTO products (id, code, group_code, name, created_by, created_at)
SELECT ic.code, ic.code::text, '10', ic.name, :'ADMIN'::uuid, :'START_TS'
FROM item_catalog ic;

-- Estrutura de produto (BOM) — cada acabado consome 2-4 matérias-primas
INSERT INTO item_structures (parent_code, child_code, quantity, unit_of_measurement, loss_percentage, sequence, health, created_by, created_at, updated_at)
SELECT f.code, 20000 + (1 + ((f.code * 7 + s.seq) % 80)),
  round((0.5 + random()*8)::numeric, 4), 'UN', round((random()*6)::numeric, 2),
  s.seq, 'ATIVO', :'ADMIN'::uuid, :'START_TS', :'START_TS'
FROM item_catalog f
CROSS JOIN LATERAL generate_series(1, 2 + (f.code % 3)) s(seq)
WHERE f.kind = 'FINISHED';

-- Roteiros de fabricação — 1 por acabado (id = code-10000, p/ FK das OFs)
INSERT INTO manufacturing_routes (id, code, item_code, alternative, description, situation, is_standard, is_active, created_at, updated_at, created_by)
SELECT f.code - 10000, f.code - 10000, f.code, 1, 'Roteiro padrão — ' || f.name, 'APROVADA', true, true, :'START_TS', :'START_TS', :'ADMIN'::uuid
FROM item_catalog f WHERE f.kind = 'FINISHED';

-- ─── 6. Clientes (50) e Fornecedores (35) ────────────────────────────────────
INSERT INTO customers (id, code, name, trade_name, document_type, document_number, state_registration,
  payment_condition_id, payment_cond_visibility, credit_limit, is_active, created_at, created_by, updated_at)
SELECT i, i,
  (ARRAY['Estruturas Metálicas','Indústria Mecânica','Construtora','Montadora','Serralheria',
         'Metalúrgica','Implementos','Caldeiraria','Engenharia','Equipamentos'])[1 + (i % 10)]
    || ' ' || (ARRAY['São Paulo','Paraná','Minas','Sul','Brasil','Nacional','Industrial','Alfa','Beta','Premium'])[1 + ((i/10) % 10)]
    || ' Ltda',
  'Cliente ' || lpad(i::text, 3, '0'),
  'CNPJ', lpad((10000000000000 + i*137)::text, 14, '0'), lpad((100000000 + i*271)::text, 9, '0'),
  1 + (i % 4), 'TODOS', (10000 + (i % 10) * 9000)::numeric, true, :'START_TS', :'ADMIN'::uuid, :'START_TS'
FROM generate_series(1, 50) i;

INSERT INTO suppliers (id, code, name, trade_name, person_type, document_type, document_number, state_registration,
  payment_condition_id, register_date, homologated, is_active, created_at, created_by, updated_at)
SELECT i, i,
  (ARRAY['Siderúrgica','Aços','Distribuidora de Metais','Parafusos','Tintas Industriais',
         'Abrasivos','Ferragens','Insumos Soldagem','Alumínios','Comercial de Aços'])[1 + (i % 10)]
    || ' ' || (ARRAY['Nacional','do Brasil','Sul','Express','Prime','Forte','União','Líder','Global','Master'])[1 + ((i/10) % 10)]
    || ' Ltda',
  'Fornecedor ' || lpad(i::text, 3, '0'),
  'JURIDICA', 'CNPJ', lpad((20000000000000 + i*311)::text, 14, '0'), lpad((200000000 + i*419)::text, 9, '0'),
  1 + (i % 4), '2025-06-01', true, true, :'START_TS', :'ADMIN'::uuid, :'START_TS'
FROM generate_series(1, 35) i;

-- ─── 7. Saldos de estoque ────────────────────────────────────────────────────
INSERT INTO stock_balances (id, item_code, warehouse_id, quantity, minimum_stock, maximum_stock, safety_stock,
  avg_cost, last_cost, total_cost, last_movement_at, updated_at)
SELECT row_number() OVER (), ic.code, w.wid,
  q.qty, 20, 5000, 30,
  ic.unit_cost, round(ic.unit_cost*1.02, 4), round(q.qty * ic.unit_cost, 4),
  (current_date - (floor(random()*25))::int)::timestamptz, now()
FROM item_catalog ic
CROSS JOIN (VALUES (1), (2)) w(wid)
CROSS JOIN LATERAL (SELECT round((40 + random()*1800)::numeric, 4) AS qty, ic.code AS k) q
WHERE (w.wid = 1 AND ic.kind = 'RAW')
   OR (w.wid = 2 AND ic.kind = 'FINISHED');

-- ─── 8. PEDIDOS DE VENDA (1500) + itens ──────────────────────────────────────
INSERT INTO sales_orders (code, order_number, enterprise_code, status, origin, emission_date, delivery_date,
  digit_date, customer_code, payment_term_code, currency_code, total_gross, total_net, total_net_no_st,
  total_with_ipi_with_st, is_firm, is_active, created_by, created_at, updated_at)
SELECT g, g, 1,
  (ARRAY['F','F','F','F','F','P','P','A','R','OF'])[1 + floor(random()*10)::int],
  'NORMAL', dd.d, dd.d + (5 + floor(random()*25)::int), dd.d,
  1 + floor(random()*50)::int, 1 + (g % 4), 'BRL', 0, 0, 0, 0,
  true, true, :'ADMIN'::uuid, dd.d::timestamptz + interval '9 hour', dd.d::timestamptz + interval '9 hour'
FROM generate_series(1, 1500) g
CROSS JOIN LATERAL (SELECT ('2025-07-01'::date + (random()*(current_date - '2025-07-01'::date))::int) AS d, g AS k) dd;

INSERT INTO sales_order_items (code, sales_order_code, sequence, item_code, sales_uom, warehouse_code,
  requested_qty, unit_price, attended_qty, discount_pct, ipi_pct, icms_pct, pis_pct, cofins_pct,
  total_gross, total_net, total_net_with_ipi, total_ipi, status, created_at, updated_at)
SELECT row_number() OVER (), so.code, gs.seq, ic.code, ic.uom, 2,
  q.qty, ic.sale_price,
  CASE WHEN so.status = 'F' THEN q.qty ELSE 0 END,
  d.disc, 5, 18, 0.65, 3.0,
  round(q.qty * ic.sale_price, 4),
  round(q.qty * ic.sale_price * (1 - d.disc/100), 4),
  round(q.qty * ic.sale_price * (1 - d.disc/100) * 1.05, 4),
  round(q.qty * ic.sale_price * (1 - d.disc/100) * 0.05, 4),
  CASE WHEN so.status = 'F' THEN 'DELIVERED' WHEN so.status = 'P' THEN 'PARTIAL' ELSE 'OPEN' END,
  so.created_at, so.created_at
FROM sales_orders so
CROSS JOIN LATERAL generate_series(1, 1 + (so.code % 4)) gs(seq)
CROSS JOIN LATERAL (SELECT (10001 + floor(random()*50)::int) AS pick, so.code AS k) p
JOIN item_catalog ic ON ic.code = p.pick
CROSS JOIN LATERAL (SELECT (1 + floor(random()*20))::numeric AS qty, gs.seq AS k) q
CROSS JOIN LATERAL (SELECT round((random()*10)::numeric, 2) AS disc, gs.seq AS k) d;

UPDATE sales_orders so SET
  total_gross = t.tg, total_net = t.tn, total_net_no_st = t.tn, total_with_ipi_with_st = t.twi
FROM (SELECT sales_order_code, sum(total_gross) tg, sum(total_net) tn, sum(total_net_with_ipi) twi
      FROM sales_order_items GROUP BY sales_order_code) t
WHERE so.code = t.sales_order_code;

INSERT INTO sales_order_sequences (enterprise_code, last_number) VALUES (1, 1500);

-- ─── 9. PEDIDOS DE COMPRA (700) + itens ──────────────────────────────────────
INSERT INTO purchase_orders (code, order_number, enterprise_code, status, origin, emission_date, delivery_date,
  supplier_code, payment_term_code, currency_code, total_gross, total_net, total_discount,
  is_firm, is_active, created_by, created_at, updated_at)
SELECT g, g, 1,
  (ARRAY['RECEIVED','RECEIVED','RECEIVED','RECEIVED','APPROVED','APPROVED','PARTIAL','REQUESTED','DRAFT','CANCELLED'])[1 + floor(random()*10)::int],
  'NORMAL', dd.d, dd.d + (7 + floor(random()*20)::int),
  1 + floor(random()*35)::int, 1 + (g % 4), 'BRL', 0, 0, 0,
  true, true, :'ADMIN'::uuid, dd.d::timestamptz + interval '10 hour', dd.d::timestamptz + interval '10 hour'
FROM generate_series(1, 700) g
CROSS JOIN LATERAL (SELECT ('2025-07-01'::date + (random()*(current_date - '2025-07-01'::date))::int) AS d, g AS k) dd;

INSERT INTO purchase_order_items (code, purchase_order_code, sequence, item_code, requested_qty, received_qty,
  unit_price, total_price, discount_pct, ipi_pct, icms_pct, status, purchase_uom, created_at, updated_at)
SELECT row_number() OVER (), po.code, gs.seq, ic.code, q.qty,
  CASE WHEN po.status IN ('RECEIVED') THEN q.qty WHEN po.status = 'PARTIAL' THEN round(q.qty/2,4) ELSE 0 END,
  ic.unit_cost, round(q.qty * ic.unit_cost, 4), 0, 5, 18,
  CASE WHEN po.status = 'RECEIVED' THEN 'RECEIVED' WHEN po.status = 'PARTIAL' THEN 'PARTIAL' WHEN po.status = 'CANCELLED' THEN 'CANCELLED' ELSE 'OPEN' END,
  ic.uom, po.created_at, po.created_at
FROM purchase_orders po
CROSS JOIN LATERAL generate_series(1, 1 + (po.code % 3)) gs(seq)
CROSS JOIN LATERAL (SELECT (20001 + floor(random()*80)::int) AS pick, gs.seq AS k) p
JOIN item_catalog ic ON ic.code = p.pick
CROSS JOIN LATERAL (SELECT (10 + floor(random()*200))::numeric AS qty, gs.seq AS k) q;

UPDATE purchase_orders po SET total_gross = t.tp, total_net = t.tp
FROM (SELECT purchase_order_code, sum(total_price) tp FROM purchase_order_items GROUP BY purchase_order_code) t
WHERE po.code = t.purchase_order_code;

INSERT INTO purchase_order_sequences (enterprise_code, last_number) VALUES (1, 700);

-- ─── 10. ORDENS DE FABRICAÇÃO (600) ──────────────────────────────────────────
INSERT INTO production_orders (id, order_number, item_code, planned_qty, produced_qty, scrapped_qty, status,
  start_date, end_date, machine_id, cost_center_id, employee_id, route_id, priority, is_active, created_by, created_at, updated_at)
SELECT g, g, 10001 + (g % 50), q.qty,
  CASE st.s WHEN 'COMPLETED' THEN q.qty WHEN 'CLOSED' THEN q.qty WHEN 'IN_PROGRESS' THEN round(q.qty*0.6,4) ELSE 0 END,
  CASE WHEN st.s IN ('COMPLETED','CLOSED') THEN round(q.qty*0.03,4) ELSE 0 END,
  st.s, dd.d, CASE WHEN st.s IN ('COMPLETED','CLOSED') THEN dd.d + (2+floor(random()*8)::int) ELSE NULL END,
  1 + floor(random()*8)::int, 1 + floor(random()*3)::int, 1 + floor(random()*4)::int,
  (10001 + (g % 50)) - 10000, 'NORMAL', true, :'ADMIN'::uuid,
  dd.d::timestamptz + interval '8 hour', dd.d::timestamptz + interval '8 hour'
FROM generate_series(1, 600) g
CROSS JOIN LATERAL (SELECT (ARRAY['COMPLETED','COMPLETED','COMPLETED','CLOSED','IN_PROGRESS','OPEN'])[1 + floor(random()*6)::int] AS s, g AS k) st
CROSS JOIN LATERAL (SELECT ('2025-07-01'::date + (random()*(current_date - '2025-07-01'::date))::int) AS d, g AS k) dd
CROSS JOIN LATERAL (SELECT (5 + floor(random()*60))::numeric AS qty, g AS k) q;

-- ─── 11. NOTAS FISCAIS DE SAÍDA (dos pedidos faturados) + itens ──────────────
INSERT INTO fiscal_exits (id, numero_nf, serie, data_emissao, data_saida, cnpj_destinatario, razao_social_destinatario,
  uf_destinatario, cfop, natureza_operacao, valor_produtos, valor_icms, valor_ipi, valor_pis, valor_cofins, valor_total,
  sales_order_code, status, chave_acesso, is_active, created_by, created_at, updated_at)
SELECT row_number() OVER (ORDER BY so.code),
  row_number() OVER (ORDER BY so.code),
  '1', so.emission_date, so.emission_date, c.document_number, c.name, 'SP',
  '5101', 'Venda de produção do estabelecimento',
  so.total_net, round(so.total_net*0.18, 2), round(so.total_net*0.05, 2),
  round(so.total_net*0.0065, 2), round(so.total_net*0.03, 2),
  round(so.total_net*1.05, 2), so.code, 'AUTHORIZED',
  lpad((35250000000000000000000000000000000000000000 + so.code)::numeric::text, 44, '0'),
  true, :'ADMIN'::uuid, so.created_at, so.created_at
FROM sales_orders so JOIN customers c ON c.code = so.customer_code
WHERE so.status = 'F';

INSERT INTO fiscal_exit_items (id, fiscal_exit_id, sequence, item_code, ncm, cfop, quantity, unit_price, total_price,
  base_icms, aliq_icms, valor_icms, valor_pis, valor_cofins, origem_mercadoria, description, created_at)
SELECT row_number() OVER (), fe.id, soi.sequence, soi.item_code, '73269000', '5101',
  soi.requested_qty, soi.unit_price, soi.total_net,
  soi.total_net, 18, round(soi.total_net*0.18, 2), round(soi.total_net*0.0065, 2), round(soi.total_net*0.03, 2),
  '0', ic.name, fe.created_at
FROM fiscal_exits fe
JOIN sales_order_items soi ON soi.sales_order_code = fe.sales_order_code
LEFT JOIN item_catalog ic ON ic.code = soi.item_code;

-- ─── 12. CONTAS A RECEBER (dos pedidos faturados, parceladas) ────────────────
INSERT INTO contas_receber (id, numero_documento, cliente_id, sales_order_id, data_emissao, data_vencimento,
  data_recebimento, valor_bruto, valor_recebido, parcela_numero, parcela_total, status, forma_pagamento,
  condicao_pagamento_id, criado_por, created_at, updated_at)
SELECT row_number() OVER (), 'NF' || fe.numero_nf || '-' || pr.n, so.customer_code, so.code,
  so.emission_date, venc.dt,
  CASE WHEN r.recebido THEN venc.dt + (floor(random()*6))::int ELSE NULL END,
  round(so.total_net*1.05 / pt.parc, 2),
  CASE WHEN r.recebido THEN round(so.total_net*1.05 / pt.parc, 2) ELSE 0 END,
  pr.n, pt.parc,
  CASE WHEN r.recebido THEN 'RECEBIDO' WHEN venc.dt < current_date THEN 'VENCIDO' ELSE 'PENDENTE' END,
  'BOLETO', so.payment_term_code, :'ADMIN'::uuid, so.created_at, so.created_at
FROM sales_orders so
JOIN fiscal_exits fe ON fe.sales_order_code = so.code
CROSS JOIN LATERAL (SELECT (1 + (so.code % 3)) AS parc) pt
CROSS JOIN LATERAL generate_series(1, pt.parc) pr(n)
CROSS JOIN LATERAL (SELECT (so.emission_date + (pr.n * 28)::int)::date AS dt) venc
CROSS JOIN LATERAL (SELECT (venc.dt < current_date AND random() < 0.85) AS recebido) r
WHERE so.status = 'F';

-- ─── 13. CONTAS A PAGAR (dos pedidos de compra recebidos) ────────────────────
INSERT INTO contas_pagar (id, numero_documento, tipo_documento, fornecedor_id, purchase_order_id, data_emissao,
  data_vencimento, data_pagamento, valor_bruto, valor_pago, valor_adiantamento_abatido, parcela_numero, parcela_total, status,
  status_aprovacao, forma_pagamento, fornecedor_cnpj, criado_por, created_at, updated_at)
SELECT row_number() OVER (), 'NFC' || po.code || '-' || pr.n, 'NF_COMPRA', po.supplier_code, po.code,
  po.emission_date, venc.dt,
  CASE WHEN p.pago THEN venc.dt + (floor(random()*4))::int ELSE NULL END,
  round(po.total_net / pt.parc, 2),
  CASE WHEN p.pago THEN round(po.total_net / pt.parc, 2) ELSE 0 END, 0,
  pr.n, pt.parc,
  CASE WHEN p.pago THEN 'PAGO' WHEN venc.dt < current_date THEN 'VENCIDO' ELSE 'PENDENTE' END,
  'APROVADO', 'TED', s.document_number, :'ADMIN'::uuid, po.created_at, po.created_at
FROM purchase_orders po
JOIN suppliers s ON s.code = po.supplier_code
CROSS JOIN LATERAL (SELECT (1 + (po.code % 3)) AS parc) pt
CROSS JOIN LATERAL generate_series(1, pt.parc) pr(n)
CROSS JOIN LATERAL (SELECT (po.emission_date + (pr.n * 30)::int)::date AS dt) venc
CROSS JOIN LATERAL (SELECT (venc.dt < current_date AND random() < 0.9) AS pago) p
WHERE po.status = 'RECEIVED' AND po.total_net > 0;

-- ─── 14. MOVIMENTOS DE ESTOQUE ───────────────────────────────────────────────
-- Entradas (compras recebidas)
INSERT INTO stock_movements (id, item_code, warehouse_id, movement_type, quantity, unit_price, total_price,
  reference_type, reference_code, notes, created_at, created_by)
SELECT row_number() OVER (), poi.item_code, 1, 'IN', poi.received_qty, poi.unit_price,
  round(poi.received_qty * poi.unit_price, 4), 'PURCHASE_ORDER', poi.purchase_order_code,
  'Recebimento de compra', po.created_at, :'ADMIN'::uuid
FROM purchase_order_items poi JOIN purchase_orders po ON po.code = poi.purchase_order_code
WHERE poi.received_qty > 0;

-- Saídas (vendas faturadas)
INSERT INTO stock_movements (id, item_code, warehouse_id, movement_type, quantity, unit_price, total_price,
  reference_type, reference_code, notes, created_at, created_by)
SELECT (SELECT COALESCE(max(id),0) FROM stock_movements) + row_number() OVER (),
  soi.item_code, 2, 'OUT', soi.attended_qty, soi.unit_price,
  round(soi.attended_qty * soi.unit_price, 4), 'SALES_ORDER', soi.sales_order_code,
  'Faturamento de venda', so.created_at, :'ADMIN'::uuid
FROM sales_order_items soi JOIN sales_orders so ON so.code = soi.sales_order_code
WHERE soi.attended_qty > 0;

-- Entradas de produção (OFs concluídas)
INSERT INTO stock_movements (id, item_code, warehouse_id, movement_type, quantity, unit_price, total_price,
  reference_type, reference_code, notes, created_at, created_by)
SELECT (SELECT COALESCE(max(id),0) FROM stock_movements) + row_number() OVER (),
  po.item_code, 2, 'IN', po.produced_qty, ic.unit_cost,
  round(po.produced_qty * ic.unit_cost, 4), 'PRODUCTION_ORDER', po.id,
  'Conclusão de produção', po.created_at, :'ADMIN'::uuid
FROM production_orders po JOIN item_catalog ic ON ic.code = po.item_code
WHERE po.produced_qty > 0;

-- ─── 15. Ajuste das sequências (p/ a API continuar inserindo) ────────────────
SELECT setval('enterprise_id_seq',            (SELECT max(id) FROM enterprise));
SELECT setval('warehouse_id_seq',             (SELECT max(id) FROM warehouse));
SELECT setval('condicoes_pagamento_id_seq',   (SELECT max(id) FROM condicoes_pagamento));
SELECT setval('payment_conditions_id_seq',    (SELECT max(id) FROM payment_conditions));
SELECT setval('contas_bancarias_id_seq',      (SELECT max(id) FROM contas_bancarias));
SELECT setval('cost_centers_id_seq',          (SELECT max(id) FROM cost_centers));
SELECT setval('fiscal_classifications_id_seq', (SELECT max(id) FROM fiscal_classifications));
SELECT setval('employees_id_seq',             (SELECT max(id) FROM employees));
SELECT setval('machine_types_id_seq',         (SELECT max(id) FROM machine_types));
SELECT setval('machines_id_seq',              (SELECT max(id) FROM machines));
SELECT setval('operations_id_seq',            (SELECT max(id) FROM operations));
SELECT setval('manufacturing_routes_id_seq',  (SELECT max(id) FROM manufacturing_routes));
SELECT setval('items_id_seq',                 (SELECT max(id) FROM items));
SELECT setval('item_structures_id_seq',       (SELECT max(id) FROM item_structures));
SELECT setval('stock_balances_id_seq',        (SELECT max(id) FROM stock_balances));
SELECT setval('stock_movements_id_seq',       (SELECT max(id) FROM stock_movements));
SELECT setval('customers_id_seq',             (SELECT max(id) FROM customers));
SELECT setval('suppliers_id_seq',             (SELECT max(id) FROM suppliers));
SELECT setval('sales_orders_code_seq',        (SELECT max(code) FROM sales_orders));
SELECT setval('sales_order_items_code_seq',   (SELECT max(code) FROM sales_order_items));
SELECT setval('purchase_orders_code_seq',     (SELECT max(code) FROM purchase_orders));
SELECT setval('purchase_order_items_code_seq', (SELECT max(code) FROM purchase_order_items));
SELECT setval('production_orders_id_seq',      (SELECT max(id) FROM production_orders));
SELECT setval('fiscal_exits_id_seq',          (SELECT max(id) FROM fiscal_exits));
SELECT setval('fiscal_exit_items_id_seq',     (SELECT max(id) FROM fiscal_exit_items));
SELECT setval('contas_receber_id_seq',        (SELECT max(id) FROM contas_receber));
SELECT setval('contas_pagar_id_seq',          (SELECT max(id) FROM contas_pagar));

COMMIT;

-- ─── Resumo ──────────────────────────────────────────────────────────────────
\echo '──────────────────────────────────────────────'
\echo ' SEED DEMO concluído. Contagens:'
SELECT 'items' t, count(*) n FROM items
UNION ALL SELECT 'customers', count(*) FROM customers
UNION ALL SELECT 'suppliers', count(*) FROM suppliers
UNION ALL SELECT 'stock_balances', count(*) FROM stock_balances
UNION ALL SELECT 'sales_orders', count(*) FROM sales_orders
UNION ALL SELECT 'sales_order_items', count(*) FROM sales_order_items
UNION ALL SELECT 'purchase_orders', count(*) FROM purchase_orders
UNION ALL SELECT 'purchase_order_items', count(*) FROM purchase_order_items
UNION ALL SELECT 'production_orders', count(*) FROM production_orders
UNION ALL SELECT 'fiscal_exits', count(*) FROM fiscal_exits
UNION ALL SELECT 'fiscal_exit_items', count(*) FROM fiscal_exit_items
UNION ALL SELECT 'contas_receber', count(*) FROM contas_receber
UNION ALL SELECT 'contas_pagar', count(*) FROM contas_pagar
UNION ALL SELECT 'stock_movements', count(*) FROM stock_movements
ORDER BY t;
