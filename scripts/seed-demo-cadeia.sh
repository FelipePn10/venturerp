#!/usr/bin/env bash
# Cenário de cadeia de fabricação do demo: o RN-DEMO precisa de 2 buchas
# (BU-DEMO) e 1 tubo (TP-DEMO), que também são fabricados. Pedido de venda do
# cliente 50 no projeto OBRA-DEMO, ordens planejadas ligadas (filha → pai) e as
# três OFs criadas pela API ao firmar — com a lista de materiais da estrutura e
# as etapas do roteiro, como nasceriam de verdade. A OF da bucha já começa em
# produção. Não refaz nada se o RN-DEMO já existir, para não desfazer o que foi
# testado na tela.
set -euo pipefail

export BASE_URL="${BASE_URL:-http://localhost:5072}"
export DEMO_ADMIN_EMAIL="${DEMO_ADMIN_EMAIL:-admin@panossoerp.demo}"
export DEMO_ADMIN_PASSWORD="${DEMO_ADMIN_PASSWORD:-Demo@12345}"
export DB_CONTAINER=panossoerp-postgres-demo
export DB_USER=panossoerp_demo
export DB_NAME="${DB_NAME:-panossoerpdatabase_demo}"

exec python3 - <<'PY'
import json, os, subprocess, sys, uuid, urllib.request, urllib.error
from datetime import date, timedelta

ADMIN = "00000000-0000-0000-0000-000000000001"
RN, BU, TP = 10901, 10902, 10903
CHAPA, TUBO = 20013, 20029  # matérias-primas do demo, estocadas em UN


def sql(q):
    r = subprocess.run(["docker", "exec", "-i", os.environ["DB_CONTAINER"], "psql", "-U", os.environ["DB_USER"],
                        "-d", os.environ["DB_NAME"], "-v", "ON_ERROR_STOP=1", "-qtAX"], input=q, text=True, capture_output=True)
    if r.returncode != 0:
        sys.exit(f"falha no SQL do cenário de cadeia: {r.stderr.strip()}")
    return r.stdout.strip()


ITENS = f"{RN},{BU},{TP}"
# Tempo de roteiro sem unidade é ambíguo: o banco do demo guardou "300/250"
# de testes antigos no roteiro copiado, a OF leu como horas e a bucha pediu
# 6.000 h de laser — o APS inteiro parou. Sem unidade, a etapa herda o tempo
# da operação da biblioteca; as OFs abertas do cenário são refeitas por ele.
# Roda sempre (corrige o demo já carregado) e não muda nada na segunda vez.
sql(f"""BEGIN;
UPDATE route_operations ro SET standard_time = NULL, setup_time = NULL, run_time = NULL, labor_time = NULL
  FROM manufacturing_routes r WHERE r.id = ro.route_id AND r.item_code IN ({ITENS}) AND ro.time_unit IS NULL
   AND (ro.standard_time IS NOT NULL OR ro.setup_time IS NOT NULL OR ro.run_time IS NOT NULL OR ro.labor_time IS NOT NULL);
UPDATE production_order_operations o
   SET planned_hours = round(COALESCE(NULLIF(lib.run_time, 0), lib.standard_time, 0) * p.planned_qty, 4), setup_hours = COALESCE(lib.setup_time, 0)
  FROM production_orders p, route_operations ro, operations lib
 WHERE p.id = o.production_order_id AND p.item_code IN ({ITENS}) AND o.status IN ('PENDING','PAUSED','INTERRUPTED','IN_PROGRESS')
   AND ro.id = o.route_operation_id AND lib.id = ro.operation_id AND COALESCE(lib.time_unit, 'HORA') = 'HORA'
   AND ro.time_unit IS NULL AND ro.run_time IS NULL AND ro.setup_time IS NULL
   AND (o.planned_hours > 100 OR o.setup_hours > 24);
COMMIT;""")
if sql(f"SELECT count(*) FROM production_orders WHERE item_code = {RN}") != "0":
    print("▸ Cenário de cadeia já carregado (RN-DEMO, OBRA-DEMO).")
    sys.exit(0)
if sql(f"SELECT count(*) FROM items WHERE code = {RN}") != "0":
    # Carga anterior parou no meio (sem OF): desfaz o que ficou e recomeça.
    sql(f"""BEGIN;
DELETE FROM production_chain_links WHERE child_planned_order_id IN (SELECT id FROM planned_orders WHERE item_code IN ({ITENS}))
   OR parent_planned_order_id IN (SELECT id FROM planned_orders WHERE item_code IN ({ITENS}));
DELETE FROM sales_order_planned_orders WHERE planned_order_id IN (SELECT id FROM planned_orders WHERE item_code IN ({ITENS}));
DELETE FROM planned_orders WHERE item_code IN ({ITENS});
DELETE FROM sales_order_demands WHERE item_code IN ({ITENS});
DELETE FROM sales_order_items WHERE item_code IN ({ITENS});
DELETE FROM sales_orders s WHERE s.project_code = 'OBRA-DEMO' AND NOT EXISTS (SELECT 1 FROM sales_order_items i WHERE i.sales_order_code = s.code);
DELETE FROM route_operations WHERE route_id IN (SELECT id FROM manufacturing_routes WHERE item_code IN ({ITENS}));
DELETE FROM manufacturing_routes WHERE item_code IN ({ITENS});
DELETE FROM item_structures WHERE parent_code IN ({ITENS});
DELETE FROM items WHERE code IN ({ITENS});
COMMIT;""")

TOKEN = None


def call(method, path, body=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(os.environ["BASE_URL"].rstrip("/") + path, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    req.add_header("Idempotency-Key", str(uuid.uuid4()))
    if TOKEN:
        req.add_header("Authorization", "Bearer " + TOKEN)
    try:
        with urllib.request.urlopen(req, timeout=120) as r:
            raw, status = r.read(), r.status
    except urllib.error.HTTPError as e:
        raw, status = e.read(), e.code
    try:
        return status, json.loads(raw) if raw else None
    except ValueError:
        return status, raw.decode(errors="replace")


# Itens: cópia do cadastro do 10002 (acabado do seed, com roteiro), outros códigos.
colunas = sql("""SELECT string_agg(quote_ident(column_name), ',' ORDER BY ordinal_position) FROM information_schema.columns
  WHERE table_schema = 'public' AND table_name = 'items' AND column_name NOT IN ('id','code','business_code','name')""")
for code, negocio, nome in [(RN, "RN-DEMO", "ROLDANA RN DEMO"), (BU, "BU-DEMO", "BUCHA DEMO"), (TP, "TP-DEMO", "TUBO DEMO")]:
    sql(f"INSERT INTO items(code,business_code,name,{colunas}) SELECT {code},'{negocio}','{nome}',{colunas} FROM items WHERE code = 10002")

sql(f"""
INSERT INTO item_structures(parent_code,child_code,quantity,sequence,route_operation_sequence,created_by) VALUES
  ({RN},{BU},2,1,20,'{ADMIN}'), ({RN},{TP},1,2,20,'{ADMIN}'), ({RN},{CHAPA},1,3,10,'{ADMIN}'),
  ({BU},{CHAPA},1,1,10,'{ADMIN}'), ({TP},{TUBO},1,1,10,'{ADMIN}');
""")

# Roteiros: cópia do roteiro do 10002 (corte, solda, pintura); a bucha não pinta, o tubo só corta.
rota = sql("SELECT id FROM manufacturing_routes WHERE item_code = 10002 ORDER BY id LIMIT 1")
rcols = sql("""SELECT string_agg(quote_ident(column_name), ',' ORDER BY ordinal_position) FROM information_schema.columns
  WHERE table_schema = 'public' AND table_name = 'manufacturing_routes' AND column_name NOT IN ('id','code','item_code','is_active')""")
ocols = sql("""SELECT string_agg(quote_ident(column_name), ',' ORDER BY ordinal_position) FROM information_schema.columns
  WHERE table_schema = 'public' AND table_name = 'route_operations' AND column_name NOT IN ('id','route_id')""")
for item, ate in [(RN, 30), (BU, 20), (TP, 10)]:
    nova = sql(f"""INSERT INTO manufacturing_routes(code,item_code,is_active,{rcols})
      SELECT (SELECT COALESCE(MAX(code),0)+1 FROM manufacturing_routes),{item},TRUE,{rcols} FROM manufacturing_routes WHERE id = {rota} RETURNING id""").splitlines()[0]
    sql(f"INSERT INTO route_operations(route_id,{ocols}) SELECT {nova},{ocols} FROM route_operations WHERE route_id = {rota} AND sequence <= {ate}")
    # Sem unidade, o tempo copiado não diz se é segundo, minuto ou hora: herda o da operação.
    sql(f"""UPDATE route_operations SET is_active = TRUE,
      standard_time = CASE WHEN time_unit IS NULL THEN NULL ELSE standard_time END, setup_time = CASE WHEN time_unit IS NULL THEN NULL ELSE setup_time END,
      run_time = CASE WHEN time_unit IS NULL THEN NULL ELSE run_time END, labor_time = CASE WHEN time_unit IS NULL THEN NULL ELSE labor_time END
      WHERE route_id = {nova}""")

# Projeto e pedido de venda do cliente 50.
sql(f"""INSERT INTO projects(enterprise_id,code,name,customer_code,status,created_by)
  SELECT 1,'OBRA-DEMO','Obra de demonstração — linha de roldanas',50,'ATIVO','{ADMIN}'
  WHERE NOT EXISTS (SELECT 1 FROM projects WHERE enterprise_id = 1 AND upper(code) = 'OBRA-DEMO')""")
entrega = (date.today() + timedelta(days=30)).isoformat()
pedido = sql(f"""INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,project_code,project_name,delivery_date,created_by)
  VALUES((SELECT COALESCE(MAX(order_number),0)+1 FROM sales_orders),1,'P',50,'OBRA-DEMO','Obra de demonstração — linha de roldanas','{entrega}','{ADMIN}') RETURNING code""").splitlines()[0]
linha = sql(f"""INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price,delivery_date)
  VALUES({pedido},1,{RN},10,450,'{entrega}') RETURNING code""").splitlines()[0]
demanda = sql(f"""INSERT INTO sales_order_demands(sales_order_code,item_code,quantity,delivery_date,enterprise_id)
  VALUES({pedido},{RN},10,'{entrega}',1) RETURNING code""").splitlines()[0]


def planejada(item, qtd, dias):
    return sql(f"""INSERT INTO planned_orders(order_number,item_code,quantity,order_type,demand_type,need_date,start_date,created_by,demand_code,enterprise_id,sales_order_code)
      VALUES((SELECT COALESCE(MAX(order_number),0)+1 FROM planned_orders),{item},{qtd},'PRODUCTION','SALES_ORDER',CURRENT_DATE+{dias},CURRENT_DATE+{dias - 3},'{ADMIN}',1,1,{demanda})
      RETURNING id || '|' || code""").splitlines()[0].split("|")


pl_rn, pl_bu, pl_tp = planejada(RN, 10, 20), planejada(BU, 20, 12), planejada(TP, 10, 12)
sql(f"""INSERT INTO sales_order_planned_orders(enterprise_id,sales_order_item_code,planned_order_id,quantity,created_by) VALUES(1,{linha},{pl_rn[0]},10,'{ADMIN}');
INSERT INTO production_chain_links(enterprise_id,child_planned_order_id,parent_planned_order_id,quantity,created_by) VALUES
  (1,{pl_bu[0]},{pl_rn[0]},20,'{ADMIN}'), (1,{pl_tp[0]},{pl_rn[0]},10,'{ADMIN}');""")

st, login = call("POST", "/users/login", {"email": os.environ["DEMO_ADMIN_EMAIL"], "password": os.environ["DEMO_ADMIN_PASSWORD"]})
if st != 200 or not isinstance(login, dict) or not login.get("token"):
    sys.exit(f"cenário de cadeia: login no demo falhou ({st})")
TOKEN = login["token"]
ofs = {}
for nome, pl in [("BU", pl_bu), ("TP", pl_tp), ("RN", pl_rn)]:  # de baixo para cima
    st, r = call("GET", f"/api/planned-order/{pl[1]}/firm")
    if st not in (200, 201):
        sys.exit(f"cenário de cadeia: firmar a planejada do {nome} falhou ({st}): {r}")
    ofs[nome] = sql(f"SELECT id FROM production_orders WHERE planned_order_id = {pl[0]}")
st, r = call("POST", f"/api/production-order/{ofs['BU']}/start", {})
if st not in (200, 201):
    print(f"  (a OF da bucha não começou: {st} {r})")
print(f"▸ Cenário de cadeia: pedido {pedido} (OBRA-DEMO) → OF {ofs['RN']} RN-DEMO, com OF {ofs['BU']} bucha e OF {ofs['TP']} tubo.")
PY
