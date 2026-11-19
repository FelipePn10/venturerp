#!/usr/bin/env bash
# ============================================================================
# Aceite da cadeia OF filha → OF pai (migração 503), pela API e com o MRP de
# verdade: o caso da Tecnofer — o RN precisa da bucha (BU) e do tubo (TP), que
# também são fabricados, cada um com roteiro de duas etapas.
#
#   pedido 1 (RN ×10) e pedido 2 (RN ×5) → MRP → firmar (de baixo para cima e
#   de cima para baixo) → OFs ligadas à OF pai e ao pedido → conferência do
#   início (parâmetro 52) → etapas na ordem do roteiro → entrega da bucha
#   reservada para a OF do pedido 1 → a OF do pedido 2 não a leva → a OF do
#   pedido 1 consome → encerrada, libera a sobra.
#
# Monta os próprios cadastros (itens, estrutura, roteiro, almoxarifado,
# pedidos) direto no banco e conduz o processo só pela API. Pode rodar de novo
# no mesmo banco (códigos novos a cada rodada).
#
# Variáveis:
#   BASE_URL        (padrão http://localhost:5079)
#   PGURL           (obrigatória — o banco da API, para os cadastros)
#   ADMIN_EMAIL / ADMIN_PASSWORD  (obrigatórias — usuário ADMIN da empresa 1)
#   EMPRESA         (código da empresa, padrão 1)
#   PARAR_APOS      (opcional: número da seção em que o roteiro para, deixando
#                    o cenário montado para a simulação da tela; os códigos vão
#                    para ESTADO, padrão /dev/stdout, em JSON)
#
# Uso: PGURL=postgres://... ADMIN_EMAIL=... ADMIN_PASSWORD=... bash scripts/test-cadeia-of.sh
# ============================================================================
set -euo pipefail
: "${PGURL:?Informe PGURL (o banco da API)}"
: "${ADMIN_EMAIL:?Informe ADMIN_EMAIL}"
: "${ADMIN_PASSWORD:?Informe ADMIN_PASSWORD}"
export BASE_URL="${BASE_URL:-http://localhost:5079}" EMPRESA="${EMPRESA:-1}" PARAR_APOS="${PARAR_APOS:-}" ESTADO="${ESTADO:-/dev/stdout}"
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


def call(method, path, body=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(BASE + path, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    req.add_header("Idempotency-Key", str(uuid.uuid4()))
    if TOKEN:
        req.add_header("Authorization", "Bearer " + TOKEN)
    try:
        with urllib.request.urlopen(req, timeout=300) as r:
            raw, status = r.read(), r.status
    except urllib.error.HTTPError as e:
        raw, status = e.read(), e.code
    try:
        return status, json.loads(raw) if raw else None
    except ValueError:
        return status, raw.decode(errors="replace")


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
        print(f"\033[0;31m  ✗ {label}\033[0m {str(detalhe)[:400]}")
    return cond


def exige(cond, label, detalhe=""):
    if not check(cond, label, detalhe):
        resumo()


def resumo():
    print(f"\nTotal: {ok_n + fail_n}   \033[0;32mPASS: {ok_n}\033[0m   \033[0;31mFAIL: {fail_n}\033[0m")
    sys.exit(1 if fail_n else 0)


def d(v):
    return D(str(v if v is not None else 0))


def marco(secao_n, **estado):
    """Para aqui se PARAR_APOS pedir, gravando o cenário para a simulação da tela."""
    if os.environ.get("PARAR_APOS") == str(secao_n):
        with open(os.environ["ESTADO"], "w") as f:
            json.dump(estado, f)
        resumo()


def concluir_etapas(of, qtd):
    """Executa o roteiro da OF na ordem: cada etapa em andamento e concluída."""
    st, etapas = call("GET", f"/api/production-order/{of}/operations")
    for e in sorted(etapas or [], key=lambda e: e.get("sequence", 0)):
        for status in ("IN_PROGRESS", "DONE"):
            st, r = call("POST", "/api/production-order/operations/advance",
                         {"operation_id": e["id"], "status": status, "actual_hours": 1, "produced_qty": qtd})
            if not check(st == 200, f"OF {of}: etapa {e.get('sequence')} → {status}", (st, r)):
                return


hoje = date.today()

# ── 0. Acesso ────────────────────────────────────────────────────────────────
secao("0. API e acesso")
st, login = call("POST", "/users/login", {"email": os.environ["ADMIN_EMAIL"], "password": os.environ["ADMIN_PASSWORD"]})
exige(st == 200 and login.get("token"), "login", login)
TOKEN = login["token"]

# ── 1. Cadastros ─────────────────────────────────────────────────────────────
secao("1. Cadastros: itens, estrutura, roteiro, almoxarifado, pedidos")
emp = int(sql(f"SELECT id FROM enterprise WHERE code={EMPRESA}"))
ator = sql(f"SELECT id FROM users WHERE email='{os.environ['ADMIN_EMAIL']}'")
u = int(time.time()) % 1_000_000
almox = int(sql(f"""INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
    VALUES('CAD-{u}','CADEIA {u}','{ator}','INTERNO','NORMAL',true,true,{emp}) RETURNING id"""))
cod = {}
for n, (nome, tipo) in enumerate([("RN", 0), ("BU", 0), ("TP", 0), ("MP", 1)]):
    c = 91_000_000_000 + u * 10 + n
    cod[nome] = c
    sql(f"""INSERT INTO items(code,business_code,name,warehouse_code,created_by,enterprise_id,engineering_type,planning_purchase_lead_time_days)
        VALUES({c},'{nome}-{u}','{nome} cadeia {u}',{almox},'{ator}',{emp},{tipo},2)""")
RN, BU, TP, MP = cod["RN"], cod["BU"], cod["TP"], cod["MP"]
for pai, filho, q, seq in [(RN, BU, 2, 1), (RN, TP, 1, 2), (BU, MP, 1, 1), (TP, MP, 1, 1)]:
    sql(f"INSERT INTO item_structures(parent_code,child_code,quantity,sequence,created_by) VALUES({pai},{filho},{q},{seq},'{ator}')")
check(True, f"itens RN={RN} BU={BU} TP={TP} MP={MP}, estrutura RN←2 BU + 1 TP; BU,TP←MP")

# Roteiro de duas etapas em cada fabricado, pela API.
ops = []
for nome in ("Corte", "Usinagem"):
    st, op = call("POST", "/api/routing/operations/", {"name": f"{nome} {u}", "origin": "INTERNA", "setup_time": 0,
                                                       "run_time": 6, "run_base_qty": 1, "time_unit": "MIN", "created_by": ator})
    exige(st in (200, 201) and op.get("id"), f"operação {nome}", op)
    ops.append(op["id"])
for item in (RN, BU, TP):
    st, rt = call("POST", "/api/routing/routes/", {"item_code": item, "alternative": 1, "description": f"Roteiro {item}",
                                                   "is_standard": True, "created_by": ator})
    exige(st in (200, 201) and rt.get("id"), f"roteiro do item {item}", rt)
    for seq, op in zip((10, 20), ops):
        st, ro = call("POST", f"/api/routing/route-operations/{rt['id']}/", {"route_id": rt["id"], "sequence": seq,
                                                                           "operation_id": op, "situation": "APROVADA"})
        exige(st in (200, 201), f"etapa {seq} no roteiro de {item}", ro)

# Matéria-prima em estoque: o BLOQUEAR só barra pela cadeia, não pela compra.
st, r = call("POST", "/api/stock/movements/create", {"item_code": MP, "warehouse_id": almox, "movement_type": "IN",
                                                     "quantity": 1000, "unit_price": 1, "reference_type": "MANUAL",
                                                     "notes": "aceite cadeia"})
exige(st in (200, 201), "matéria-prima em estoque (1000)", r)

cliente = 700_000 + u % 100_000
sql(f"INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES({cliente},'CLIENTE CADEIA {u}','11444777000161','{ator}',{emp})")


def pedido(qtd, dias):
    so = int(sql(f"""INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,created_by,delivery_date)
        VALUES({u * 100 + dias},{EMPRESA},'P',{cliente},'{ator}',CURRENT_DATE+{dias}) RETURNING code"""))
    linha = int(sql(f"""INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price,delivery_date)
        VALUES({so},1,{RN},{qtd},100,CURRENT_DATE+{dias}) RETURNING code"""))
    return so, linha


ped1, lin1 = pedido(10, 40)
ped2, lin2 = pedido(5, 60)
check(True, f"pedido 1 ({ped1}): RN ×10 em 40 dias; pedido 2 ({ped2}): RN ×5 em 60 dias")

# ── 2. MRP ───────────────────────────────────────────────────────────────────
secao("2. MRP: explosão de baixo para cima")
st, planos = call("GET", "/api/production-plan/list")
plano = max([int(p["code"]) for p in planos or []] + [9000]) + 1
st, r = call("POST", "/api/production-plan/create", {"code": plano, "name": f"Cadeia {u}", "independent_demands": "ALL",
                                                    "group_same_date_orders": False, "planning_types": ["MRP"]})
exige(st == 201, f"plano {plano}", r)
st, res = call("POST", "/api/planning/run-pipeline", {"plan_code": plano, "initial_order_number": 600000 + u % 1000 * 100,
                                                      "generate_llc": True, "start_from": hoje.isoformat() + "T08:00:00-03:00"})
exige(st == 200 and res.get("mrp_status") in ("COMPLETED", "COMPLETED_WITH_ERRORS"), "MRP rodou", res)
st, sug = call("GET", f"/api/mrp-calculation/suggestions/{plano}")
meu = lambda s: int(s.get("legacy_item_code", s.get("item_code")))
por_item = {}
for s in sug or []:
    por_item.setdefault(meu(s), []).append(s)
rn = sorted(por_item.get(RN, []), key=lambda s: s["need_date"])
bu = sorted(por_item.get(BU, []), key=lambda s: s["need_date"])
tp = sorted(por_item.get(TP, []), key=lambda s: s["need_date"])
exige(len(rn) == 2 and [d(s["quantity"]) for s in rn] == [10, 5], "RN: uma sugestão por pedido (10 e 5)", rn)
exige(len(bu) == 2 and [d(s["quantity"]) for s in bu] == [20, 10], "BU: 2 por RN (20 e 10)", bu)
exige(len(tp) == 2 and [d(s["quantity"]) for s in tp] == [10, 5], "TP: 1 por RN (10 e 5)", tp)
check(all(s["order_type"] == "FABRICACAO" for s in rn + bu + tp), "RN, BU e TP são fabricação")
for filho in (bu[0], tp[0]):
    check(str(filho["need_date"])[:10] <= str(rn[0]["start_date"])[:10],
          f"{meu(filho)} fica pronto ({str(filho['need_date'])[:10]}) até o início do RN ({str(rn[0]['start_date'])[:10]})")
peg = sql(f"""SELECT COUNT(*) FROM mrp_pegging WHERE enterprise_id={emp} AND plan_code={plano}
    AND demand_kind='SUGESTAO' AND demand_code={rn[0]['code']} AND supply_code IN ({bu[0]['code']},{tp[0]['code']})""")
check(peg == "2", "pegging: BU e TP do pedido 1 cobrem a sugestão do RN do pedido 1", peg)

# ── 3. Firmar ────────────────────────────────────────────────────────────────
secao("3. Firmar: bucha antes do RN, tubo depois")


def firmar(s):
    st, f = call("POST", f"/api/mrp-calculation/suggestions/{s['code']}/firm", {})
    exige(st in (200, 201) and f.get("planned_code"), f"sugestão {s['code']} ({meu(s)}) firmada", f)
    pid = int(sql(f"SELECT id FROM planned_orders WHERE code={f['planned_code']} AND enterprise_id={emp}"))
    of = sql(f"SELECT id FROM production_orders WHERE planned_order_id={pid} AND enterprise_id={emp} AND is_active")
    exige(of != "", f"OF gerada para a planejada {f['planned_code']}", of)
    return pid, int(of)


bu1_pl, bu1 = firmar(bu[0])
rn1_pl, rn1 = firmar(rn[0])
tp1_pl, tp1 = firmar(tp[0])
rn2_pl, rn2 = firmar(rn[1])
firmar(rn[0])  # de novo: não duplica
elos = sql(f"SELECT COUNT(*) FROM production_chain_links WHERE enterprise_id={emp} AND parent_planned_order_id={rn1_pl}")
check(elos == "2", "OF do RN (pedido 1) tem 2 filhas ligadas: bucha e tubo", elos)
cruzado = sql(f"SELECT COUNT(*) FROM production_chain_links WHERE enterprise_id={emp} AND parent_planned_order_id={rn2_pl}")
check(cruzado == "0", "nenhuma filha do pedido 1 ficou ligada ao RN do pedido 2", cruzado)
for nome, pl in (("bucha", bu1_pl), ("tubo", tp1_pl), ("RN", rn1_pl)):
    v = sql(f"SELECT COUNT(*) FROM sales_order_planned_orders WHERE planned_order_id={pl} AND sales_order_item_code={lin1}")
    check(v == "1", f"{nome} ligada à linha do pedido 1", v)
v = sql(f"SELECT COUNT(*) FROM sales_order_planned_orders WHERE planned_order_id IN ({bu1_pl},{tp1_pl}) AND sales_order_item_code={lin2}")
check(v == "0", "bucha e tubo do pedido 1 não pegaram o pedido 2", v)

st, cad = call("GET", f"/api/production-order/{bu1}/chain")
exige(st == 200, "cadeia da OF da bucha", cad)
check([p["sales_order_code"] for p in cad.get("sales_orders") or []] == [ped1], "a cadeia da bucha mostra o pedido 1", cad.get("sales_orders"))
check([p.get("production_order_id") for p in cad.get("parents") or []] == [rn1], "a cadeia da bucha mostra a OF do RN como pai", cad.get("parents"))
st, cad = call("GET", f"/api/production-order/{rn1}/chain")
check(sorted(f.get("production_order_id") for f in cad.get("children") or []) == sorted([bu1, tp1]), "a OF do RN mostra bucha e tubo como filhas", cad)

# ── 4. Conferência do início ─────────────────────────────────────────────────
secao("4. Parâmetro 52: o RN não começa antes das filhas")
st, r = call("PUT", "/api/planning-params/update", {"param_number": 52, "value": "BLOQUEAR"})
exige(st == 200, "parâmetro 52 = BLOQUEAR", r)
st, r = call("POST", f"/api/production-order/{rn1}/start", {"id": rn1, "start_date": hoje.isoformat()})
check(st == 422 and "ainda não foi concluída" in json.dumps(r, ensure_ascii=False), "BLOQUEAR: OF do RN recusada com as filhas abertas", (st, r))

# ── 5. Bucha: roteiro na ordem e entrega reservada ───────────────────────────
secao("5. OF da bucha: etapas na ordem do roteiro, entrega reservada para o RN")
st, r = call("POST", f"/api/production-order/{bu1}/start", {"id": bu1, "start_date": hoje.isoformat()})
exige(st == 200, "OF da bucha iniciada (só depende da matéria-prima, que tem)", r)
st, etapas = call("GET", f"/api/production-order/{bu1}/operations")
etapas = sorted(etapas or [], key=lambda e: e.get("sequence", 0))
exige(len(etapas) == 2, "a OF da bucha tem as 2 etapas do roteiro", etapas)
st, r = call("POST", "/api/production-order/operations/advance", {"operation_id": etapas[1]["id"], "status": "IN_PROGRESS", "actual_hours": 1, "produced_qty": 1})
check(st == 422, "etapa 20 não começa antes da 10", (st, r))
concluir_etapas(bu1, 20)


def entregar(of, qtd, final):
    return call("POST", f"/api/production-order/{of}/complete", {"id": of, "end_date": hoje.isoformat(), "warehouse_id": almox,
                                                                "quantity": qtd, "final": final, "idempotency_key": str(uuid.uuid4())})


st, r = entregar(bu1, 20, True)
exige(st == 200, "bucha entregue (20)", r)


def reservado(of):
    return d(sql(f"""SELECT COALESCE(SUM(quantity),0) FROM stock_reservations WHERE enterprise_id={emp}
        AND reference_type='OF_COMPONENTE' AND reference_code={of} AND status='ACTIVE'"""))


check(reservado(rn1) == 20, "as 20 buchas ficaram reservadas para a OF do RN do pedido 1", reservado(rn1))
st, disp = call("GET", f"/api/production-order/{rn1}/material-availability")
linha_bu = next((l for l in (disp or {}).get("lines", []) if int(l.get("legacy_item_code", l.get("item_code"))) == BU), {})
check(d(linha_bu.get("reserved_for_order")) == 20 and d(linha_bu.get("shortage")) == 0,
      "disponibilidade da OF do RN: bucha reservada, sem falta", linha_bu)

marco(5, rn1=rn1, rn2=rn2, bu1=bu1, tp1=tp1, ped1=ped1, ped2=ped2, bu=BU, tp=TP, rn=RN, almox=almox, sufixo=u)

# ── 6. Outro pedido não leva a bucha ─────────────────────────────────────────
secao("6. A OF do RN do pedido 2 não leva a bucha do pedido 1")
st, disp2 = call("GET", f"/api/production-order/{rn2}/material-availability")
linha2 = next((l for l in (disp2 or {}).get("lines", []) if int(l.get("legacy_item_code", l.get("item_code"))) == BU), {})
check(d(linha2.get("shortage")) == 10, "para a OF do pedido 2, a bucha está em falta (a do pedido 1 é reservada)", linha2)
st, r = call("PUT", "/api/planning-params/update", {"param_number": 52, "value": "AVISAR"})
st, ini2 = call("POST", f"/api/production-order/{rn2}/start", {"id": rn2, "start_date": hoje.isoformat()})
check(st == 200 and any("falta" in w for w in ini2.get("warnings") or []), "AVISAR: OF do pedido 2 inicia com aviso de falta", ini2)
st, r = call("POST", "/api/production-order/consumption", {"production_order_id": rn2, "item_code": BU, "consumed_qty": 10,
                                                          "warehouse_id": almox, "consumption_date": hoje.isoformat()})
check(st == 422 and "reservado" in json.dumps(r, ensure_ascii=False), "consumo da bucha na OF do pedido 2 recusado", (st, r))
st, r = call("POST", "/api/stock/movements/create", {"item_code": BU, "warehouse_id": almox, "movement_type": "OUT",
                                                     "quantity": 1, "reference_type": "MANUAL", "notes": "saída avulsa"})
check(st == 422, "saída manual da bucha reservada recusada", (st, r))

# ── 7. Tubo, início e consumo do RN ──────────────────────────────────────────
secao("7. Tubo pronto, RN começa e consome o que é dele")
st, r = call("PUT", "/api/planning-params/update", {"param_number": 52, "value": "BLOQUEAR"})
st, r = call("POST", f"/api/production-order/{tp1}/start", {"id": tp1, "start_date": hoje.isoformat()})
concluir_etapas(tp1, 10)
st, r = entregar(tp1, 10, True)
exige(st == 200, "tubo entregue (10)", r)
st, r = call("POST", f"/api/production-order/{rn1}/start", {"id": rn1, "start_date": hoje.isoformat()})
check(st == 200 and not r.get("warnings"), "BLOQUEAR: com as filhas concluídas a OF do RN começa", (st, r))
st, r = call("POST", "/api/production-order/consumption", {"production_order_id": rn1, "item_code": BU, "consumed_qty": 12,
                                                          "warehouse_id": almox, "consumption_date": hoje.isoformat()})
check(st == 201, "a OF do RN baixa 12 buchas", (st, r))
check(reservado(rn1) == 18, "a reserva do RN tem a bucha (8) e o tubo (10) que faltam baixar", reservado(rn1))

# ── 8. Encerramento libera ───────────────────────────────────────────────────
secao("8. OF do RN concluída: a sobra é liberada")
concluir_etapas(rn1, 10)
st, r = entregar(rn1, 10, True)
exige(st == 200, "RN entregue (10), OF concluída", r)
check(reservado(rn1) == 0, "nada mais reservado para a OF concluída", reservado(rn1))
# Baixa automática (mig 506): a entrega final tira da lista da OF o que ainda
# faltava — as 8 buchas que o consumo manual (12) não cobriu. Nada sobra.
saiu_bu = d(sql(f"""SELECT COALESCE(SUM(quantity),0) FROM stock_movements WHERE enterprise_id={emp}
                   AND reference_type='PRODUCTION_ORDER' AND reference_code={rn1} AND item_code={BU}
                   AND movement_type IN ('OUT','REP')"""))
check(saiu_bu == 20, "a entrega final baixou sozinha as 8 buchas que faltavam (20 no total)", saiu_bu)

# ── 9. Uma OF de bucha para os dois RNs ──────────────────────────────────────
secao("9. Lote mínimo: uma OF de bucha abastece os RNs dos dois pedidos")
rn_b, bu_b = RN + 5, BU + 5
for c, nome in ((rn_b, "RNL"), (bu_b, "BUL")):
    sql(f"""INSERT INTO items(code,business_code,name,warehouse_code,created_by,enterprise_id,engineering_type,planning_minimum_lot)
        VALUES({c},'{nome}-{u}','{nome} cadeia {u}',{almox},'{ator}',{emp},0,{50 if nome == 'BUL' else 0})""")
sql(f"INSERT INTO item_structures(parent_code,child_code,quantity,sequence,created_by) VALUES({rn_b},{bu_b},2,1,'{ator}')")
sql(f"INSERT INTO item_structures(parent_code,child_code,quantity,sequence,created_by) VALUES({bu_b},{MP},1,1,'{ator}')")
for item in (rn_b, bu_b):
    st, rt = call("POST", "/api/routing/routes/", {"item_code": item, "alternative": 1, "description": f"Roteiro {item}",
                                                   "is_standard": True, "created_by": ator})
    for seq, op in zip((10, 20), ops):
        call("POST", f"/api/routing/route-operations/{rt['id']}/", {"route_id": rt["id"], "sequence": seq, "operation_id": op, "situation": "APROVADA"})
lins = []
for qtd, dias in ((10, 41), (5, 61)):
    so = int(sql(f"""INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,created_by,delivery_date)
        VALUES({u * 100 + dias},{EMPRESA},'P',{cliente},'{ator}',CURRENT_DATE+{dias}) RETURNING code"""))
    lins.append(int(sql(f"""INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price,delivery_date)
        VALUES({so},1,{rn_b},{qtd},100,CURRENT_DATE+{dias}) RETURNING code""")))
st, res = call("POST", "/api/planning/run-pipeline", {"plan_code": plano, "initial_order_number": 700000 + u % 1000 * 100,
                                                      "generate_llc": True, "start_from": hoje.isoformat() + "T08:00:00-03:00"})
exige(st == 200, "MRP de novo", res)
st, sug = call("GET", f"/api/mrp-calculation/suggestions/{plano}")
rnb = sorted([s for s in sug or [] if meu(s) == rn_b], key=lambda s: s["need_date"])
bub = [s for s in sug or [] if meu(s) == bu_b]
exige(len(rnb) == 2 and len(bub) == 1 and d(bub[0]["quantity"]) == 50, "2 RNs e UMA bucha de 50 (lote mínimo)", (rnb, bub))
bub_pl, bub_of = firmar(bub[0])
r1_pl, r1_of = firmar(rnb[0])
r2_pl, r2_of = firmar(rnb[1])
pais = sql(f"SELECT COUNT(*) FROM production_chain_links WHERE enterprise_id={emp} AND child_planned_order_id={bub_pl}")
check(pais == "2", "a bucha ficou ligada às duas OFs de RN", pais)
ped = sql(f"SELECT COUNT(*) FROM sales_order_planned_orders WHERE planned_order_id={bub_pl} AND sales_order_item_code IN ({lins[0]},{lins[1]})")
check(ped == "2", "a bucha leva os dois pedidos (ela atende os dois)", ped)
call("POST", f"/api/production-order/{bub_of}/start", {"id": bub_of, "start_date": hoje.isoformat()})
concluir_etapas(bub_of, 50)
st, r = entregar(bub_of, 25, False)
exige(st == 200, "entrega parcial de 25 buchas", r)
check(reservado(r1_of) == 20 and reservado(r2_of) == 5, "parcial: primeiro o RN que começa antes (20), o resto para o outro (5)",
      (reservado(r1_of), reservado(r2_of)))
st, r = entregar(bub_of, 25, True)
check(reservado(r1_of) == 20 and reservado(r2_of) == 10, "completa: cada RN com o que precisa (20 e 10); 20 livres",
      (reservado(r1_of), reservado(r2_of)))
st, r = call("POST", "/api/stock/movements/create", {"item_code": bu_b, "warehouse_id": almox, "movement_type": "OUT",
                                                     "quantity": 20, "reference_type": "MANUAL", "notes": "sobra livre"})
check(st in (200, 201), "as 20 buchas que sobram do lote mínimo são livres", (st, r))
st, r = call("POST", "/api/stock/movements/create", {"item_code": bu_b, "warehouse_id": almox, "movement_type": "OUT",
                                                     "quantity": 1, "reference_type": "MANUAL", "notes": "reservada"})
check(st == 422, "mas a 21ª já é reservada", (st, r))
st, res = call("POST", "/api/planning/run-pipeline", {"plan_code": plano, "initial_order_number": 800000 + u % 1000 * 100,
                                                      "generate_llc": True, "start_from": hoje.isoformat() + "T08:00:00-03:00"})
st, sug = call("GET", f"/api/mrp-calculation/suggestions/{plano}")
check(not [s for s in sug or [] if meu(s) == bu_b], "MRP de novo: a bucha reservada cobre os RNs, nenhuma bucha a mais",
      [s for s in sug or [] if meu(s) == bu_b])
check(not [s for s in sug or [] if meu(s) == rn_b], "nem RN a mais (as OFs firmes cobrem os pedidos)",
      [s for s in sug or [] if meu(s) == rn_b])

# ── 10. A bucha fica pronta antes de o RN ser firmado ────────────────────────
secao("10. Bucha entregue antes de existir a OF do RN: a OF do RN, ao nascer, reserva")
rn_t, bu_t = RN + 7, BU + 7
for c, nome in ((rn_t, "RNT"), (bu_t, "BUT")):
    sql(f"""INSERT INTO items(code,business_code,name,warehouse_code,created_by,enterprise_id,engineering_type)
        VALUES({c},'{nome}-{u}','{nome} cadeia {u}',{almox},'{ator}',{emp},0)""")
sql(f"INSERT INTO item_structures(parent_code,child_code,quantity,sequence,created_by) VALUES({rn_t},{bu_t},2,1,'{ator}')")
sql(f"INSERT INTO item_structures(parent_code,child_code,quantity,sequence,created_by) VALUES({bu_t},{MP},1,1,'{ator}')")
for item in (rn_t, bu_t):
    st, rt = call("POST", "/api/routing/routes/", {"item_code": item, "alternative": 1, "description": f"Roteiro {item}",
                                                   "is_standard": True, "created_by": ator})
    for seq, op in zip((10, 20), ops):
        call("POST", f"/api/routing/route-operations/{rt['id']}/", {"route_id": rt["id"], "sequence": seq, "operation_id": op, "situation": "APROVADA"})
so = int(sql(f"""INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,created_by,delivery_date)
    VALUES({u * 100 + 71},{EMPRESA},'P',{cliente},'{ator}',CURRENT_DATE+71) RETURNING code"""))
sql(f"""INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price,delivery_date)
    VALUES({so},1,{rn_t},10,100,CURRENT_DATE+71)""")
st, res = call("POST", "/api/planning/run-pipeline", {"plan_code": plano, "initial_order_number": 900000 + u % 1000 * 100,
                                                      "generate_llc": True, "start_from": hoje.isoformat() + "T08:00:00-03:00"})
exige(st == 200, "MRP do RN novo", res)
st, sug = call("GET", f"/api/mrp-calculation/suggestions/{plano}")
rnt = [s for s in sug or [] if meu(s) == rn_t]
but = [s for s in sug or [] if meu(s) == bu_t]
exige(len(rnt) == 1 and len(but) == 1 and d(but[0]["quantity"]) == 20, "1 RN e 1 bucha de 20", (rnt, but))
but_pl, but_of = firmar(but[0])
call("POST", f"/api/production-order/{but_of}/start", {"id": but_of, "start_date": hoje.isoformat()})
concluir_etapas(but_of, 20)
st, r = entregar(but_of, 20, True)
exige(st == 200, "bucha entregue (20) sem OF do RN ainda", r)
livres = d(sql(f"SELECT COALESCE(SUM(quantity-reserved_qty),0) FROM stock_balances WHERE enterprise_id={emp} AND item_code={bu_t}"))
check(livres == 20, "sem OF pai, as 20 ficam no estoque (nada a reservar ainda)", livres)
rnt_pl, rnt_of = firmar(rnt[0])
check(reservado(rnt_of) == 20, "a OF do RN, criada depois, reservou as 20 buchas que já estavam prontas", reservado(rnt_of))
st, r = call("POST", "/api/stock/movements/create", {"item_code": bu_t, "warehouse_id": almox, "movement_type": "OUT",
                                                     "quantity": 1, "reference_type": "MANUAL", "notes": "outro uso"})
check(st == 422, "outra saída já não leva essas buchas", (st, r))
st, disp = call("GET", f"/api/production-order/{rnt_of}/material-availability")
lt = next((l for l in (disp or {}).get("lines", []) if int(l.get("legacy_item_code", l.get("item_code"))) == bu_t), {})
check(d(lt.get("reserved_for_order")) == 20 and d(lt.get("shortage")) == 0, "a tela de materiais da OF do RN mostra as 20 reservadas", lt)

# ── 11. Programação (APS): a OF do RN só começa depois da bucha ──────────────
secao("11. APS: a OF do RN é programada depois de a OF da bucha terminar")
# Um centro para a bucha e outro para o RN: a máquina ocupada não explica a
# espera — só a cadeia.
ops_wc = {}
for papel, n in (("bu", 3), ("rn", 4)):
    mt_code = u * 10 + n
    mt_id = int(sql(f"""INSERT INTO machine_types(code,name,type,created_by,enterprise_id,capacity_hours)
        VALUES({mt_code},'Centro {papel} {u}',(enum_range(NULL::machine_type_enum))[1],'{ator}',{emp},8) RETURNING id"""))
    sql(f"""INSERT INTO machines(code,name,created_by,machine_type_code,enterprise_id,available_hours_per_day,efficiency_rate)
        VALUES({mt_code},'Máquina {papel} {u}','{ator}',{mt_code},{emp},8,1)""")
    ops_wc[papel] = []
    for nome in ("Torneamento", "Acabamento"):
        st, op = call("POST", "/api/routing/operations/", {"name": f"{nome} {papel} {u}", "origin": "INTERNA", "setup_time": 0,
                                                           "run_time": 30, "run_base_qty": 1, "time_unit": "MIN",
                                                           "default_work_center_id": mt_id, "created_by": ator})
        exige(st in (200, 201) and op.get("id"), f"operação {nome} ({papel}) no centro de trabalho", op)
        ops_wc[papel].append(op["id"])
rn_a, bu_a = RN + 9, BU + 9
for c, nome in ((rn_a, "RNA"), (bu_a, "BUA")):
    sql(f"""INSERT INTO items(code,business_code,name,warehouse_code,created_by,enterprise_id,engineering_type)
        VALUES({c},'{nome}-{u}','{nome} cadeia {u}',{almox},'{ator}',{emp},0)""")
sql(f"INSERT INTO item_structures(parent_code,child_code,quantity,sequence,created_by) VALUES({rn_a},{bu_a},2,1,'{ator}')")
sql(f"INSERT INTO item_structures(parent_code,child_code,quantity,sequence,created_by) VALUES({bu_a},{MP},1,1,'{ator}')")
for item, papel in ((rn_a, "rn"), (bu_a, "bu")):
    st, rt = call("POST", "/api/routing/routes/", {"item_code": item, "alternative": 1, "description": f"Roteiro {item}",
                                                   "is_standard": True, "created_by": ator})
    for seq, op in zip((10, 20), ops_wc[papel]):
        call("POST", f"/api/routing/route-operations/{rt['id']}/", {"route_id": rt["id"], "sequence": seq, "operation_id": op, "situation": "APROVADA"})
so = int(sql(f"""INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,created_by,delivery_date)
    VALUES({u * 100 + 81},{EMPRESA},'P',{cliente},'{ator}',CURRENT_DATE+81) RETURNING code"""))
sql(f"""INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price,delivery_date)
    VALUES({so},1,{rn_a},10,100,CURRENT_DATE+81)""")
st, res = call("POST", "/api/planning/run-pipeline", {"plan_code": plano, "initial_order_number": 950000 + u % 1000 * 10,
                                                      "generate_llc": True, "start_from": hoje.isoformat() + "T08:00:00-03:00"})
exige(st == 200, "MRP do RN com roteiro em centro de trabalho", res)
st, sug = call("GET", f"/api/mrp-calculation/suggestions/{plano}")
rna = [s for s in sug or [] if meu(s) == rn_a]
bua = [s for s in sug or [] if meu(s) == bu_a]
exige(len(rna) == 1 and len(bua) == 1, "1 RN e 1 bucha", (rna, bua))
bua_pl, bua_of = firmar(bua[0])
rna_pl, rna_of = firmar(rna[0])


def janela(of):
    r = sql(f"""SELECT to_char(MIN(scheduled_start),'YYYY-MM-DD"T"HH24:MI') || '|' || to_char(MAX(scheduled_end),'YYYY-MM-DD"T"HH24:MI')
        FROM production_sequences WHERE enterprise_id={emp} AND production_order_id={of}""")
    return r.split("|") if r else ["", ""]


inicio_aps = (hoje + timedelta(days=1)).isoformat() + "T08:00:00-03:00"
# O RN vai primeiro na lista e tem entrega antes da bucha? Não importa: a cadeia manda.
st, r = call("POST", "/api/aps/sequence", {"start_from": inicio_aps, "order_ids": [rna_of, bua_of]})
exige(st == 200, "APS programou as duas OFs", r)
ini_rn, fim_rn = janela(rna_of)
ini_bu, fim_bu = janela(bua_of)
exige(ini_rn and fim_bu, "as duas OFs têm programação", (ini_rn, fim_bu))
check(ini_rn >= fim_bu, f"o RN começa ({ini_rn}) depois de a bucha terminar ({fim_bu})", (ini_bu, fim_bu, ini_rn, fim_rn))
st, r = call("POST", "/api/aps/sequence", {"start_from": inicio_aps, "order_ids": [rna_of]})
exige(st == 200, "APS reprogramou só a OF do RN", r)
ini_rn2, _ = janela(rna_of)
check(ini_rn2 >= fim_bu, f"reprogramando só o RN, ele continua depois da bucha ({ini_rn2} ≥ {fim_bu})", ini_rn2)
st, r = call("POST", "/api/aps/sequence", {"start_from": inicio_aps, "order_ids": [rna_of, bua_of], "direction": "BACKWARD"})
exige(st == 200, "APS regressivo (a partir da entrega)", r)
ini_rn3, fim_rn3 = janela(rna_of)
ini_bu3, fim_bu3 = janela(bua_of)
check(fim_bu3 <= ini_rn3, f"regressivo: a bucha termina ({fim_bu3}) até o RN começar ({ini_rn3})", (ini_bu3, fim_bu3, ini_rn3, fim_rn3))
resumo()
PY
