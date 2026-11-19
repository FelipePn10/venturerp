#!/usr/bin/env bash
# ============================================================================
# Aceite dos bugs de unidade, compra e liberação achados na Tecnofer, pela API
# e com o MRP de verdade:
#
#   1. linha de estrutura com fator velho (tubo passou de MM para M);
#   2. aprovar sugestão de compra sem olhar a tabela de preço (pedido a R$ 0);
#   3. requisição de serviço externo com unidade "SERV" fixa;
#   4. prazo do serviço externo como "hoje + prazo";
#   5. pedido do MRP na unidade de estoque (chapa em kg, não em UN);
#   6. PDF do plano de materiais (pedido sem data derrubava o documento);
#   7. liberação do item não via comprado com estrutura, fator velho nem
#      operação externa sem fornecedor;
#   8. gerar pedido da requisição respondia 201 com nenhum pedido.
#
# Monta os cadastros direto no banco e conduz o processo pela API. Pode rodar
# de novo no mesmo banco (códigos novos a cada rodada).
#
# Variáveis: BASE_URL (padrão http://localhost:5079), PGURL, ADMIN_EMAIL,
# ADMIN_PASSWORD (obrigatórias), EMPRESA (padrão 1).
#
# Uso: PGURL=postgres://... ADMIN_EMAIL=... ADMIN_PASSWORD=... bash scripts/test-unidades-compras.sh
# ============================================================================
set -euo pipefail
: "${PGURL:?Informe PGURL (o banco da API)}"
: "${ADMIN_EMAIL:?Informe ADMIN_EMAIL}"
: "${ADMIN_PASSWORD:?Informe ADMIN_PASSWORD}"
export BASE_URL="${BASE_URL:-http://localhost:5079}" EMPRESA="${EMPRESA:-1}"
command -v python3 >/dev/null || { echo "dependência ausente: python3" >&2; exit 1; }
command -v psql >/dev/null || { echo "dependência ausente: psql" >&2; exit 1; }

exec python3 - <<'PY'
import json, os, subprocess, sys, time, uuid, urllib.request, urllib.error
from datetime import date, timedelta
from decimal import Decimal as D

BASE = os.environ["BASE_URL"].rstrip("/")
PGURL = os.environ["PGURL"]
EMPRESA = int(os.environ["EMPRESA"])
TOKEN = None
ok_n = fail_n = 0


def call(method, path, body=None, raw=False):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(BASE + path, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    req.add_header("Idempotency-Key", str(uuid.uuid4()))
    if TOKEN:
        req.add_header("Authorization", "Bearer " + TOKEN)
    try:
        with urllib.request.urlopen(req, timeout=300) as r:
            corpo, status = r.read(), r.status
    except urllib.error.HTTPError as e:
        corpo, status = e.read(), e.code
    if raw:
        return status, corpo
    try:
        return status, json.loads(corpo) if corpo else None
    except ValueError:
        return status, corpo.decode(errors="replace")


def sql(q):
    out = subprocess.run(["psql", PGURL, "-v", "ON_ERROR_STOP=1", "-qtAX", "-c", q], capture_output=True, text=True)
    if out.returncode != 0:
        print(out.stderr)
        raise SystemExit(f"SQL falhou: {q[:200]}")
    return out.stdout.strip()


def secao(t):
    print(f"\n\033[1;36m── {t}\033[0m")


def check(cond, label, detalhe=""):
    global ok_n, fail_n
    if cond:
        ok_n += 1
        print(f"\033[0;32m  ✓ {label}\033[0m")
    else:
        fail_n += 1
        print(f"\033[0;31m  ✗ {label}\033[0m {str(detalhe)[:500]}")
    return cond


def exige(cond, label, detalhe=""):
    if not check(cond, label, detalhe):
        resumo()


def resumo():
    print(f"\nTotal: {ok_n + fail_n}   \033[0;32mPASS: {ok_n}\033[0m   \033[0;31mFAIL: {fail_n}\033[0m")
    sys.exit(1 if fail_n else 0)


def d(v):
    return D(str(v if v is not None else 0))


txt = lambda r: json.dumps(r, ensure_ascii=False)
hoje = date.today()

secao("0. API e acesso")
st, login = call("POST", "/users/login", {"email": os.environ["ADMIN_EMAIL"], "password": os.environ["ADMIN_PASSWORD"]})
exige(st == 200 and login.get("token"), "login", login)
TOKEN = login["token"]
emp = int(sql(f"SELECT id FROM enterprise WHERE code={EMPRESA}"))
ator = sql(f"SELECT id FROM users WHERE email='{os.environ['ADMIN_EMAIL']}'")
u = int(time.time()) % 1_000_000
almox = int(sql(f"""INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
    VALUES('UNC-{u}','UNIDADES {u}','{ator}','INTERNO','NORMAL',true,true,{emp}) RETURNING id"""))
seq = iter(range(100))


def item(nome, tipo, estoque, compra=None):
    c = 92_000_000_000 + u * 100 + next(seq)
    sql(f"""INSERT INTO items(code,business_code,name,warehouse_code,created_by,enterprise_id,engineering_type,
        warehouse_unit_of_measurement,supplies_purchase_uom,planning_purchase_lead_time_days,engineering_weight)
        VALUES({c},'{nome}-{u}','{nome} {u}',{almox},'{ator}',{emp},{tipo},'{estoque}',{f"'{compra}'" if compra else 'NULL'},2,
               '{{"gross":1,"net":1,"unit":"KG"}}')""")
    NEG[c] = f"{nome}-{u}"
    return c


NEG = {}  # código interno → código de negócio (as rotas do item usam o de negócio)


def fornecedor(nome):
    c = 1_800_000_000 + u * 10 + next(seq)
    sql(f"""INSERT INTO suppliers(code,name,document_number,created_by,enterprise_id)
        VALUES({c},'{nome} {u}','{c:014d}','{ator}',{emp})""")
    return c


def fonte(item_code, supplier, uom=None):
    sql(f"""INSERT INTO item_preferred_suppliers(item_code,supplier_code,uom,is_preferred,created_by,enterprise_id)
        VALUES({item_code},{supplier},{f"'{uom}'" if uom else 'NULL'},true,'{ator}',{emp})""")


cliente = 710_000 + u % 100_000
sql(f"INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES({cliente},'CLIENTE UN {u}','11444777000161','{ator}',{emp})")


def pedido_venda(item_code, qtd, dias, status="P", com_data=True):
    so = int(sql(f"""INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,created_by,delivery_date)
        VALUES({u * 100 + next(seq)},{EMPRESA},'{status}',{cliente},'{ator}',{f"CURRENT_DATE+{dias}" if com_data else 'NULL'}) RETURNING code"""))
    sql(f"""INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price,delivery_date)
        VALUES({so},1,{item_code},{qtd},100,{f"CURRENT_DATE+{dias}" if com_data else 'NULL'})""")
    return so


def linha(pai):
    return sql(f"SELECT quantity_stock_uom::numeric(20,6)||'|'||conversion_factor::numeric(20,10) FROM item_structures WHERE parent_code={pai} AND is_active").split("|")


def novo_plano(nome):
    st, planos = call("GET", "/api/production-plan/list")
    plano = max([int(p["code"]) for p in planos or []] + [9000]) + 1
    st, r = call("POST", "/api/production-plan/create", {"code": plano, "name": f"{nome} {u}", "independent_demands": "ALL",
                                                        "group_same_date_orders": False, "planning_types": ["MRP"]})
    exige(st == 201, f"plano {plano}", r)
    return plano


def rodar_mrp(plano):
    st, res = call("POST", "/api/planning/run-pipeline", {"plan_code": plano, "initial_order_number": 700000 + u % 1000 * 100 + next(seq),
                                                          "generate_llc": True, "start_from": hoje.isoformat() + "T08:00:00-03:00"})
    exige(st == 200 and res.get("mrp_status") in ("COMPLETED", "COMPLETED_WITH_ERRORS"), "MRP rodou", res)
    st, sug = call("GET", f"/api/mrp-calculation/suggestions/{plano}")
    return sug or []


meu = lambda s: int(s.get("legacy_item_code", s.get("item_code")))

# ── 1. Fator velho na estrutura ─────────────────────────────────────────────
secao("1. Tubo passou de MM para M: a estrutura acompanha")
TUBO = item("TUBO", 1, "MM")
RN = item("RN", 0, "UN")
sql(f"""INSERT INTO item_structures(parent_code,child_code,quantity,unit_of_measurement,quantity_stock_uom,conversion_factor,sequence,created_by)
    VALUES({RN},{TUBO},104,'MM',104,1,1,'{ator}')""")
check(linha(RN) == ["104.000000", "1.0000000000"], "linha RN ← 104 MM de tubo, gravada com o tubo em MM (104, fator 1)", linha(RN))
st, r = call("PUT", f"/api/items/{NEG[TUBO]}", {"warehouse": {"unit_of_measurement": "M"}})
exige(st == 200, "tubo passa a ser estocado em metro", (st, r))
avisos = ((r or {}).get("data") or r or {}).get("warnings") or []
check(any("1 linha(s) de estrutura" in w for w in avisos), "a resposta avisa que a estrutura foi recalculada", r)
check(linha(RN) == ["0.104000", "0.0010000000"], "a linha agora diz 0,104 M (fator 0,001) — antes ficava 104", linha(RN))

# O caso do fator de barra que ficou: o dado velho já está no banco.
sql(f"UPDATE item_structures SET quantity_stock_uom=0.0173333333, conversion_factor=0.0001666667 WHERE parent_code={RN} AND is_active")
st, rel = call("GET", f"/api/items/{NEG[RN]}/activation-readiness")
check(st == 200 and not rel.get("ready") and any("unidade velha" in i for i in rel.get("issues") or []),
      "liberação do RN acusa a linha com fator de barra velho (bug 7)", rel)
pedido_venda(RN, 40, 40)
plano = novo_plano("Unidades")
sug = rodar_mrp(plano)
tubo = [s for s in sug if meu(s) == TUBO]
check(len(tubo) == 1 and d(tubo[0]["quantity"]) == D("4.16"),
      "MRP compra 4,16 m de tubo para 40 RN (não 4.160 m nem 0,69 m)", [(s.get("quantity"), s.get("order_type")) for s in tubo])
check(linha(RN) == ["0.104000", "0.0010000000"], "o MRP regravou a linha antes de calcular", linha(RN))
hist = sql(f"SELECT COUNT(*) FROM item_structure_history WHERE parent_code={RN} AND after_state ? 'motivo'")
check(int(hist) >= 2, "o recálculo ficou no histórico da estrutura (troca de unidade e MRP)", hist)

# ── 1b. Linha sem conversão: o cálculo avisa e o log guarda ──────────────────
secao("1b. Linha de estrutura sem conversão: aviso na resposta, no log e no PDF")
TUBOX = item("TUBOX", 1, "M")
RNX = item("RNX", 0, "UN")
sql(f"""INSERT INTO item_structures(parent_code,child_code,quantity,unit_of_measurement,quantity_stock_uom,conversion_factor,sequence,created_by)
    VALUES({RNX},{TUBOX},2,'KG',2,1,1,'{ator}')""")
pedido_venda(RNX, 5, 45)
planox = novo_plano("Sem conversao")
st, run = call("POST", "/api/mrp-calculation/run", {"plan_code": planox, "initial_order_number": 800000 + u % 1000 * 100 + next(seq), "generate_llc": True})
run = (run or {}).get("data") or run or {}
aviso_x = [a for a in (run.get("errors") or {}).get("estrutura_sem_conversao") or [] if NEG[RNX] in a]
check(st in (200, 202) and len(aviso_x) == 1 and NEG[TUBOX] in aviso_x[0], "a resposta do cálculo aponta a linha RNX ← tubo em KG sem conversão", (st, run))
gravado = sql(f"""SELECT COUNT(*) FROM mrp_calculation_logs l, jsonb_array_elements_text(l.errors->'estrutura_sem_conversao') a
    WHERE l.plan_code={planox} AND l.enterprise_id={emp} AND a LIKE '%{NEG[RNX]}%'""")
check(gravado == "1", "o aviso ficou no log gravado do cálculo (antes só ia na resposta)", gravado)
st, pdfx = call("GET", f"/api/impressao/plano-materiais/{planox}", raw=True)
check(st == 200 and pdfx[:4] == b"%PDF", "o PDF do plano com o aviso sai", (st, pdfx[:200]))
# Tira a linha de cena: as próximas seções calculam o MRP da empresa inteira.
sql(f"UPDATE item_structures SET is_active=false WHERE parent_code={RNX}")

# ── 1c. Conversão padrão: gravar recalcula, excluir avisa ────────────────────
secao("1c. Conversão padrão JG → PC: gravar recalcula a estrutura, excluir aponta a linha")
PECAJ = item("PECAJ", 1, "PC")
RNJ = item("RNJ", 0, "UN")
sql(f"""INSERT INTO item_structures(parent_code,child_code,quantity,unit_of_measurement,quantity_stock_uom,conversion_factor,sequence,created_by)
    VALUES({RNJ},{PECAJ},1,'JG',1,1,1,'{ator}')""")
st, cp = call("POST", "/api/units-of-measure/standard-conversions", {"from_uom": "JG", "to_uom": "PC", "from_quantity": "1", "to_quantity": "4"})
cp = (cp or {}).get("data") or cp or {}
exige(st in (200, 201) and cp.get("id"), "conversão padrão 1 JG = 4 PC gravada", (st, cp))
check(any("recalculadas" in w for w in cp.get("warnings") or []), "a gravação avisa que estruturas foram recalculadas", cp)
check(linha(RNJ) == ["4.000000", "4.0000000000"], "a linha RNJ ← 1 JG passa a 4 PC (fator 4)", linha(RNJ))
st, ex = call("DELETE", f"/api/units-of-measure/standard-conversions/{cp['id']}")
ex = (ex or {}).get("data") or ex or {} if isinstance(ex, dict) else {}
check(st == 200 and any(NEG[RNJ] in w and NEG[PECAJ] in w for w in ex.get("warnings") or []),
      "a exclusão aponta a estrutura RNJ que ficou sem conversão (antes: 204 calado)", (st, ex))
sql(f"UPDATE item_structures SET is_active=false WHERE parent_code={RNJ}")

# ── 6. PDF do plano de materiais ────────────────────────────────────────────
secao("6. PDF do plano de materiais")
pedido_venda(RN, 7, 50, status="R")              # só digitado: o MRP não lê
pedido_venda(RN, 3, 0, com_data=False)            # liberado e sem data de entrega
plano6 = novo_plano("Plano PDF")
rodar_mrp(plano6)
st, pdf = call("GET", f"/api/impressao/plano-materiais/{plano6}", raw=True)
check(st == 200 and pdf[:4] == b"%PDF", "o PDF sai com pedido sem data de entrega na demanda (antes: erro)", (st, pdf[:200]))

# ── 7. Comprado com estrutura e roteiro ─────────────────────────────────────
secao("7. Bucha cadastrada como comprada, com estrutura")
BU = item("BU", 1, "UN")
MP = item("MP", 1, "UN")
sql(f"INSERT INTO item_structures(parent_code,child_code,quantity,sequence,created_by) VALUES({BU},{MP},1,1,'{ator}')")
st, rel = call("GET", f"/api/items/{NEG[BU]}/activation-readiness")
check(st == 200 and not rel.get("ready") and any("cadastrado como comprado" in i for i in rel.get("issues") or []),
      "liberação acusa: o MRP compraria a bucha em vez de fabricar", rel)

# ── 2 e 5. Sugestão de compra: unidade de compra e preço ────────────────────
secao("2 e 5. Chapa estocada em kg, comprada em UN, com preço da tabela")
CHAPA = item("CHAPA", 1, "KG", compra="UN")
st, r = call("POST", "/api/item-conversions/", {"item_code": NEG[CHAPA], "from_uom": "UN", "to_uom": "KG", "factor": "84.78",
                                                "rounding_percent": "0", "tolerance_value": "0", "tolerance_type": "PERCENT"})
exige(st in (200, 201), "conversão da chapa: 1 UN = 84,78 kg", (st, r))
FORN = fornecedor("ACO")
fonte(CHAPA, FORN)


def planejada_compra(item_code, qtd):
    return int(sql(f"""INSERT INTO planned_orders(order_number,item_code,quantity,order_type,demand_type,need_date,created_by,demand_code,enterprise_id,warehouse_code)
        VALUES({u * 100 + next(seq)},{item_code},{qtd},'PURCHASE','REPLENISHMENT',CURRENT_DATE+20,'{ator}',0,{emp},{almox}) RETURNING code"""))


pl = planejada_compra(CHAPA, "376.7932")
st, r = call("POST", f"/api/purchase-order/suggestions/{pl}/approve", {"supplier_code": FORN})
check(st == 422 and "não tem preço" in txt(r), "sem preço informado nem tabela: aprovação recusada (antes: pedido a R$ 0,00)", (st, r))
tab = int(sql(f"""INSERT INTO purchase_price_tables(code,description,created_by,enterprise_id,supplier_code)
    VALUES({u * 10 + next(seq)},'ACO {u}','{ator}',{emp},{FORN}) RETURNING id"""))
sql(f"INSERT INTO purchase_price_table_items(table_id,item_code,uom,price) VALUES({tab},{CHAPA},'KG',12.50)")
# Tabela de volume do mesmo fornecedor, em UN: a partir de 10 chapas, 900 cada.
vol = int(sql(f"""INSERT INTO purchase_price_tables(code,description,created_by,enterprise_id,supplier_code)
    VALUES({u * 10 + next(seq)},'ACO VOLUME {u}','{ator}',{emp},{FORN}) RETURNING id"""))
sql(f"INSERT INTO purchase_price_table_items(table_id,item_code,uom,price,min_qty) VALUES({vol},{CHAPA},'UN',900,10)")
st, r = call("POST", f"/api/purchase-order/suggestions/{pl}/approve", {"supplier_code": FORN})
exige(st in (200, 201), "aprovada com o preço da tabela do fornecedor", (st, r))
pc = r.get("code") or (r.get("purchase_order") or {}).get("code")
l = sql(f"""SELECT requested_qty::numeric(18,4)||'|'||purchase_uom||'|'||unit_price::numeric(18,4)||'|'||internal_qty::numeric(18,4)||'|'||internal_uom
    FROM purchase_order_items WHERE planned_order_code={pl}""").split("|")
check(l[:2] == ["5.0000", "UN"], "pedido sai em 5 UN de chapa (376,79 kg ÷ 84,78 = 4,44 → 5), não 376,7932 kg", l)
check(l[2] == "1059.7500", "preço por chapa = 12,50/kg × 84,78 = 1.059,75 (tabela em kg convertida)", l)
check(l[3:] == ["423.9000", "KG"], "no estoque entram 423,9 kg (5 chapas)", l)
check(l[2] != "900.0000", "5 chapas não pegam o preço de volume (a partir de 10 UN) — antes 376 kg ≥ 10 valia", l)
pl12 = planejada_compra(CHAPA, "1017.36")
st, r = call("POST", f"/api/purchase-order/suggestions/{pl12}/approve", {"supplier_code": FORN})
exige(st in (200, 201), "12 chapas aprovadas", (st, r))
l12 = sql(f"SELECT requested_qty::numeric(18,4)||'|'||unit_price::numeric(18,4) FROM purchase_order_items WHERE planned_order_code={pl12}").split("|")
check(l12 == ["12.0000", "900.0000"], "12 chapas alcançam a tabela de volume: 900 por chapa", l12)

# ── 2b. Pedido manual: tabela do vínculo do fornecedor e desconto em cadeia ──
secao("2b. Pedido manual: tabela do fornecedor (id ≠ código) com desconto em cadeia")
PARAF = item("PARAFT", 1, "UN", compra="UN")
FORN2 = fornecedor("PARAF")
fonte(PARAF, FORN2)
cod_tab = u * 10 + next(seq)
tab2 = int(sql(f"""INSERT INTO purchase_price_tables(code,description,created_by,enterprise_id,supplier_code)
    VALUES({cod_tab},'PARAF {u}','{ator}',{emp},{FORN2}) RETURNING id"""))
check(tab2 != cod_tab, f"tabela com id {tab2} ≠ código {cod_tab} (o caso que escondia o erro)", (tab2, cod_tab))
it2 = int(sql(f"INSERT INTO purchase_price_table_items(table_id,item_code,uom,price) VALUES({tab2},{PARAF},'UN',100) RETURNING id"))
sql(f"""INSERT INTO purchase_price_item_adjustments(price_item_id,sequence,adjustment_kind,calculation_type,value)
    VALUES({it2},1,'DISCOUNT','PERCENT',10),({it2},2,'SURCHARGE','FIXED',2)""")
emp_code = sql(f"SELECT code FROM enterprise WHERE id={emp}")
sql(f"""INSERT INTO supplier_enterprises(supplier_id,enterprise_code,purchase_price_table_id)
    SELECT id,{emp_code},{tab2} FROM suppliers WHERE code={FORN2}""")
st, r = call("GET", f"/api/suppliers/{FORN2}/purchasing-defaults")
if st == 200:
    check(r.get("purchase_price_table_code") == cod_tab, "defaults do fornecedor trazem o CÓDIGO da tabela", r)
st, r = call("POST", "/api/purchase-order/", {"status": "DRAFT", "origin": "NORMAL", "emission_date": hoje.isoformat(), "supplier_code": FORN2, "currency_code": "BRL"})
exige(st in (200, 201), "pedido manual criado com o fornecedor", (st, r))
pc2 = r.get("code") or r.get("Code")
cap = sql(f"SELECT COALESCE(price_table_code::text,'') FROM purchase_orders WHERE code={pc2}")
check(cap == str(cod_tab), f"capa do pedido com a tabela {cod_tab} (antes: o id {tab2})", cap)
st, r = call("POST", f"/api/purchase-order/{pc2}/items", {"item_code": PARAF, "requested_qty": "10"})
exige(st in (200, 201), "linha incluída sem preço digitado", (st, r))
pr = sql(f"SELECT unit_price::numeric(18,4) FROM purchase_order_items WHERE purchase_order_code={pc2} AND item_code={PARAF}")
check(pr == "92.0000", "preço líquido da tabela: 100 − 10 % + R$ 2 = 92 (antes: 100, ou nenhum)", pr)

# ── 3, 4 e 8. Serviço externo ───────────────────────────────────────────────
secao("3, 4 e 8. Zincagem terceirizada: requisição, prazo e pedido")
PECA = item("PECA", 0, "UN")
MPZ = item("MPZ", 1, "UN")
ZINC = item("ZINC", 1, "UN", compra="UN")
sql(f"INSERT INTO item_structures(parent_code,child_code,quantity,sequence,created_by) VALUES({PECA},{MPZ},1,1,'{ator}')")
st, r = call("POST", "/api/stock/movements/create", {"item_code": MPZ, "warehouse_id": almox, "movement_type": "IN",
                                                     "quantity": 1000, "unit_price": 1, "reference_type": "MANUAL", "notes": "aceite unidades"})
TERC = fornecedor("ZINCADORA")
fonte(ZINC, TERC)
OUTRO = fornecedor("OUTRO")
st, opz = call("POST", "/api/routing/operations/", {"name": f"Zincagem {u}", "origin": "EXTERNA", "setup_time": 0, "run_time": 6,
                                                   "run_base_qty": 1, "time_unit": "MIN", "created_by": ator, "supplier_id": TERC,
                                                   "service_item_code": NEG[ZINC], "cost_per_unit": "2.5", "lead_time_days": 3})
exige(st in (200, 201) and opz.get("id"), "operação externa de zincagem (prazo 3 dias, item de serviço em UN)", opz)
st, opm = call("POST", "/api/routing/operations/", {"name": f"Montagem {u}", "origin": "INTERNA", "setup_time": 0, "run_time": 6,
                                                   "run_base_qty": 1, "time_unit": "MIN", "created_by": ator})
st, rt = call("POST", "/api/routing/routes/", {"item_code": PECA, "alternative": 1, "description": f"Roteiro peça {u}",
                                              "is_standard": True, "created_by": ator})
exige(st in (200, 201) and rt.get("id"), "roteiro da peça", rt)
for s_, op in ((10, opz["id"]), (20, opm["id"])):
    st, ro = call("POST", f"/api/routing/route-operations/{rt['id']}/", {"route_id": rt["id"], "sequence": s_, "operation_id": op, "situation": "APROVADA"})
    exige(st in (200, 201), f"etapa {s_}", ro)
pedido_venda(PECA, 20, 45)
plano3 = novo_plano("Serviço")
sug = rodar_mrp(plano3)
peca = [s for s in sug if meu(s) == PECA and s["order_type"] == "FABRICACAO"]
exige(len(peca) == 1, "MRP sugeriu a OF da peça", peca)
st, f = call("POST", f"/api/mrp-calculation/suggestions/{peca[0]['code']}/firm", {})
exige(st in (200, 201) and f.get("planned_code"), "sugestão firmada (gera a OF e a requisição do serviço)", (st, f))
ini, nec = sql(f"SELECT start_date||'|'||need_date FROM planned_orders WHERE code={f['planned_code']} AND enterprise_id={emp}").split("|")
req = sql(f"""SELECT ri.id||'|'||COALESCE(ri.uom,'')||'|'||COALESCE(ri.delivery_date::text,'')||'|'||ri.requisition_code
    FROM purchase_requisition_items ri JOIN purchase_requisitions r ON r.code=ri.requisition_code AND r.enterprise_code={EMPRESA}
    WHERE ri.item_code={ZINC} ORDER BY ri.id DESC LIMIT 1""").split("|")
exige(len(req) == 4, "requisição de serviço gerada", req)
req_item, uom, entrega = int(req[0]), req[1], req[2]
check(uom == "UN", "requisição do serviço em UN, a unidade de compra do item (antes: SERV, fora do catálogo)", uom)
esperado = min(date.fromisoformat(ini[:10]) + timedelta(days=3), date.fromisoformat(nec[:10]))
check(entrega == esperado.isoformat(), f"entrega do serviço = início da OF ({ini[:10]}) + 3 dias = {esperado} (antes: hoje + 3)", entrega)

st, r = call("POST", "/api/purchase-requisitions/generate-orders", {"selections": [{"requisition_item_id": req_item, "qty_to_attend": "20", "supplier_code": OUTRO}]})
check(st == 422 and "VSUP0130" in txt(r), "fornecedor que não é fonte: 422 com o motivo (antes: 201 com nenhum pedido)", (st, r))
st, r = call("POST", "/api/purchase-requisitions/generate-orders", {"selections": [{"requisition_item_id": req_item, "qty_to_attend": "20"}]})
exige(st in (200, 201) and (r.get("orders") or r.get("pedidos")), "pedido do serviço gerado para a zincadora", (st, r))
cab = sql(f"""SELECT COALESCE(po.delivery_date::text,'') FROM purchase_orders po JOIN purchase_order_items i ON i.purchase_order_code=po.code
    WHERE i.purchase_requisition_item_id={req_item} LIMIT 1""")
check(cab == esperado.isoformat(), "o cabeçalho do pedido leva a data de entrega (antes: vazio)", cab)
lin = sql(f"SELECT purchase_uom FROM purchase_order_items WHERE purchase_requisition_item_id={req_item}")
check(lin == "UN", "linha do pedido do serviço em UN", lin)

# Operação externa comprando o próprio produto e sem fornecedor (bug 7).
st, op2 = call("POST", "/api/routing/operations/", {"name": f"Pintura {u}", "origin": "EXTERNA", "setup_time": 0, "run_time": 6,
                                                   "run_base_qty": 1, "time_unit": "MIN", "created_by": ator, "service_item_code": NEG[PECA]})
st, ro = call("POST", f"/api/routing/route-operations/{rt['id']}/", {"route_id": rt["id"], "sequence": 30, "operation_id": op2["id"], "situation": "APROVADA"})
st, rel = call("GET", f"/api/items/{NEG[PECA]}/activation-readiness")
check(st == 200 and any("próprio produto" in i for i in rel.get("issues") or []),
      "liberação acusa a operação externa que compraria o próprio produto", rel)

# ── 9. Bugs achados na cópia da Tecnofer (v1.19.8) ──────────────────────────
secao("9A. Linha da tabela com fornecedor 0 é preço genérico")
CH9 = item("CHG", 1, "KG", compra="KG")
FORN9 = fornecedor("GENERICO")
fonte(CH9, FORN9)
tab9_code = u * 10 + next(seq)
sql(f"INSERT INTO purchase_price_tables(code,description,created_by,enterprise_id) VALUES({tab9_code},'GERAL {u}','{ator}',{emp})")
st, r = call("POST", "/api/purchase-price-tables/items", {"table_code": tab9_code, "item_code": NEG[CH9], "price": 14.2, "uom": "KG", "min_qty": 0, "supplier_code": 0})
exige(st in (200, 201), "linha gravada pela tela com o fornecedor limpo (0)", (st, r))
check(sql(f"SELECT COUNT(*) FROM purchase_price_table_items i JOIN purchase_price_tables t ON t.id=i.table_id WHERE t.code={tab9_code} AND t.enterprise_id={emp} AND i.supplier_code IS NULL") == "1",
      "fornecedor 0 vira genérico (vazio) — antes ficava 0 e a busca de preço ignorava a linha")
sql(f"INSERT INTO purchase_price_table_items(table_id,item_code,uom,price,supplier_code) SELECT id,{CH9},'UN',1,0 FROM purchase_price_tables WHERE code={tab9_code} AND enterprise_id={emp} ON CONFLICT DO NOTHING")
check(sql(f"SELECT COUNT(*) FROM purchase_price_table_items i JOIN purchase_price_tables t ON t.id=i.table_id WHERE t.code={tab9_code} AND i.supplier_code = 0") == "0",
      "nem gravando direto no banco o 0 fica (gatilho)")
pl9 = planejada_compra(CH9, "30")
st, r = call("POST", f"/api/purchase-order/suggestions/{pl9}/approve", {"supplier_code": FORN9})
exige(st in (200, 201), "sugestão aprovada pela tabela genérica (antes: recusada por falta de preço)", (st, r))
check(sql(f"SELECT unit_price::numeric(18,4) FROM purchase_order_items WHERE planned_order_code={pl9}") == "14.2000", "preço 14,20 da linha genérica")

secao("9B e 9C. Chapa com CHAPA no cadastro e UN na fonte de suprimento")
CH794 = item("CH794", 1, "KG", compra="CHAPA")
st, r = call("POST", "/api/item-conversions/", {"item_code": NEG[CH794], "from_uom": "CHAPA", "to_uom": "KG", "factor": "62.3",
                                                "rounding_percent": "0", "tolerance_value": "0", "tolerance_type": "PERCENT"})
exige(st in (200, 201), "só a conversão CHAPA→kg cadastrada", (st, r))
FORN794 = fornecedor("ACO 794")
fonte(CH794, FORN794, "UN")
st, rel = call("GET", f"/api/items/{NEG[CH794]}/activation-readiness")
check(st == 200 and not rel.get("ready") and any("UN→KG (fonte de suprimento)" in i for i in rel.get("issues") or []),
      "liberação acusa a compra em UN (fonte de suprimento) sem conversão para kg — antes dava liberada", rel)
st, gov = call("GET", f"/api/items/governanca/itens/{NEG[CH794]}/prontidao")
check(st == 200 and "UN→KG (fonte de suprimento)" in json.dumps(gov, ensure_ascii=False), "a prontidão da governança diz a mesma unidade", (st, gov))

antes = int(sql(f"SELECT COALESCE((SELECT last_number FROM purchase_order_sequences WHERE enterprise_code={EMPRESA}),0)"))
pl794 = planejada_compra(CH794, "124.6")
st, r = call("POST", f"/api/purchase-order/suggestions/{pl794}/approve", {"supplier_code": FORN794})
check(st == 422 and "não tem preço" in txt(r), "aprovação recusada: sem preço", (st, r))
check(int(sql(f"SELECT COALESCE((SELECT last_number FROM purchase_order_sequences WHERE enterprise_code={EMPRESA}),0)")) == antes,
      "a recusa não queimou número de pedido (antes: 9, 10, 13…)")
pl_ok = planejada_compra(CHAPA, "84.78")
st, r = call("POST", f"/api/purchase-order/suggestions/{pl_ok}/approve", {"supplier_code": FORN})
exige(st in (200, 201), "o próximo pedido aprovado", (st, r))
num = int(sql(f"SELECT po.order_number FROM purchase_orders po JOIN purchase_order_items i ON i.purchase_order_code=po.code WHERE i.planned_order_code={pl_ok}"))
check(num == antes + 1, f"o próximo pedido leva o número seguinte ({antes + 1}), sem buraco", num)
st, r = call("POST", "/api/item-conversions/", {"item_code": NEG[CH794], "from_uom": "UN", "to_uom": "KG", "factor": "62.3",
                                                "rounding_percent": "0", "tolerance_value": "0", "tolerance_type": "PERCENT"})
st, rel = call("GET", f"/api/items/{NEG[CH794]}/activation-readiness")
check(st == 200 and not any("UN→KG" in i for i in rel.get("issues") or []), "com UN→kg cadastrada a pendência some", rel)

resumo()
PY
