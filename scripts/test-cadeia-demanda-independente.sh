#!/usr/bin/env bash
# ============================================================================
# Aceite da cadeia OF filha → OF pai SEM pedido de venda: a Tecnofer faz o
# RN-01001 para ESTOQUE (demanda independente). O RN precisa da bucha (BU) e do
# tubo (TP), também fabricados, cada um com roteiro de duas etapas.
#
#   A. demanda 1 (RN ×10) e demanda 2 (RN ×5) → MRP → firmar → OFs ligadas à
#      OF pai e à demanda → parâmetro 52 → etapas na ordem → entrega da bucha
#      reservada para o RN da demanda 1 → o RN da demanda 2 não a leva →
#      consumo → encerramento libera a sobra → a demanda fica atendida.
#   B. "Gerar OF" direto da demanda (sem MRP para o RN) → MRP → a bucha e o
#      tubo ficam ligados à OF do RN gerada à mão.
#   C. APS e CRP: o RN para estoque é programado depois da bucha.
#
# Variáveis: BASE_URL (padrão http://localhost:5079), PGURL, ADMIN_EMAIL,
# ADMIN_PASSWORD (ADMIN da empresa 1). Roda de novo no mesmo banco.
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
secao("1. Cadastros: itens, estrutura, roteiro, almoxarifado")
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


def demanda(qtd, dias, n):
    st, r = call("POST", "/api/independent-demand/create", {
        "code_demand": u * 10 + n, "item_code": str(RN), "quantity": str(qtd),
        "demand_date": (hoje + timedelta(days=dias)).isoformat(), "motivo": "ESTOQUE", "warehouse_id": almox})
    exige(st == 201 and r.get("code_demand"), f"demanda independente RN ×{qtd} em {dias} dias (produção para estoque)", (st, r))
    return int(r["code_demand"]), int(sql(f"SELECT id FROM independent_demands WHERE code={r['code_demand']} AND enterprise_id={emp}"))


dem1, dem1_id = demanda(10, 40, 1)
dem2, dem2_id = demanda(5, 60, 2)
pedidos_rn = sql(f"SELECT COUNT(*) FROM sales_order_items WHERE item_code={RN}")
check(pedidos_rn == "0", "nenhum pedido de venda do RN: a fábrica trabalha só com a demanda para estoque", pedidos_rn)

# ── A2. MRP ──────────────────────────────────────────────────────────────────
secao("A2. MRP: a demanda independente explode de baixo para cima")
st, planos = call("GET", "/api/production-plan/list")
plano = max([int(p["code"]) for p in planos or []] + [9000]) + 1
st, r = call("POST", "/api/production-plan/create", {"code": plano, "name": f"Estoque {u}", "independent_demands": "ALL",
                                                    "group_same_date_orders": False, "planning_types": ["MRP"]})
exige(st == 201, f"plano {plano}", r)


def rodar_mrp(base):
    st, res = call("POST", "/api/planning/run-pipeline", {"plan_code": plano, "initial_order_number": base + u % 1000 * 100,
                                                          "generate_llc": True, "start_from": hoje.isoformat() + "T08:00:00-03:00"})
    exige(st == 200 and res.get("mrp_status") in ("COMPLETED", "COMPLETED_WITH_ERRORS"), "MRP rodou", res)
    st, sug = call("GET", f"/api/mrp-calculation/suggestions/{plano}")
    return sug or []


meu = lambda s: int(s.get("legacy_item_code", s.get("item_code")))
sug = rodar_mrp(600000)
do_item = lambda sug, it: sorted([s for s in sug if meu(s) == it], key=lambda s: s["need_date"])
rn, bu, tp = do_item(sug, RN), do_item(sug, BU), do_item(sug, TP)
exige(len(rn) == 2 and [d(s["quantity"]) for s in rn] == [10, 5], "RN: uma sugestão por demanda (10 e 5)", rn)
exige(len(bu) == 2 and [d(s["quantity"]) for s in bu] == [20, 10], "BU: 2 por RN (20 e 10)", bu)
exige(len(tp) == 2 and [d(s["quantity"]) for s in tp] == [10, 5], "TP: 1 por RN (10 e 5)", tp)
check(all(s["order_type"] == "FABRICACAO" for s in rn + bu + tp), "RN, BU e TP são fabricação")
for filho in (bu[0], tp[0]):
    check(str(filho["need_date"])[:10] <= str(rn[0]["start_date"])[:10],
          f"{meu(filho)} fica pronto ({str(filho['need_date'])[:10]}) até o início do RN ({str(rn[0]['start_date'])[:10]})")
orig = sql(f"""SELECT string_agg(DISTINCT demand_kind, ',') FROM mrp_pegging WHERE enterprise_id={emp} AND plan_code={plano}
    AND supply_code={rn[0]['code']}""")
check(orig != "" and "SUGESTAO" not in orig, f"o RN é coberto pela demanda independente (pegging: {orig})", orig)
peg = sql(f"""SELECT COUNT(*) FROM mrp_pegging WHERE enterprise_id={emp} AND plan_code={plano}
    AND demand_kind='SUGESTAO' AND demand_code={rn[0]['code']} AND supply_code IN ({bu[0]['code']},{tp[0]['code']})""")
check(peg == "2", "pegging: BU e TP da demanda 1 cobrem a sugestão do RN da demanda 1", peg)

# ── A3. Firmar ───────────────────────────────────────────────────────────────
secao("A3. Firmar: bucha antes do RN, tubo depois")


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
check(elos == "2", "OF do RN (demanda 1) tem 2 filhas ligadas: bucha e tubo", elos)
cruzado = sql(f"SELECT COUNT(*) FROM production_chain_links WHERE enterprise_id={emp} AND parent_planned_order_id={rn2_pl}")
check(cruzado == "0", "nenhuma filha da demanda 1 ficou ligada ao RN da demanda 2", cruzado)
v = sql(f"SELECT COUNT(*) FROM independent_demand_orders WHERE enterprise_id={emp} AND demand_id={dem1_id} AND planned_order_id={rn1_pl}")
check(v == "1", "a ordem do RN ficou ligada à demanda 1", v)
v = sql(f"SELECT COUNT(*) FROM independent_demand_orders WHERE enterprise_id={emp} AND planned_order_id IN ({bu1_pl},{tp1_pl})")
check(v == "0", "bucha e tubo não contam como cobertura da demanda do RN (item diferente)", v)
v = sql(f"SELECT COUNT(*) FROM independent_demand_orders WHERE enterprise_id={emp} AND demand_id={dem2_id} AND planned_order_id={rn1_pl}")
check(v == "0", "a ordem do RN da demanda 1 não cobre a demanda 2", v)

st, cad = call("GET", f"/api/production-order/{bu1}/chain")
exige(st == 200, "cadeia da OF da bucha", cad)
check([p.get("production_order_id") for p in cad.get("parents") or []] == [rn1], "a cadeia da bucha mostra a OF do RN como pai", cad.get("parents"))
check(not cad.get("sales_orders"), "sem pedido de venda na cadeia", cad.get("sales_orders"))
check([x.get("demand_code") for x in cad.get("independent_demands") or []] == [dem1],
      "a cadeia da bucha mostra a demanda 1 (produção para estoque) herdada do RN", cad.get("independent_demands"))
st, cad = call("GET", f"/api/production-order/{rn1}/chain")
check(sorted(f.get("production_order_id") for f in cad.get("children") or []) == sorted([bu1, tp1]), "a OF do RN mostra bucha e tubo como filhas", cad)
check([x.get("demand_code") for x in cad.get("independent_demands") or []] == [dem1], "a cadeia do RN mostra a demanda 1", cad.get("independent_demands"))
marco(3, rn1=rn1, bu1=bu1, tp1=tp1, dem1=dem1, rn=RN, bu=BU, sufixo=u)
st, rd = call("GET", f"/api/independent-demand/get-by-code/{dem1}")
check(st == 200, "a demanda 1 abre com as ordens", rd)
print("     demanda 1:", json.dumps({k: rd.get(k) for k in ("situacao", "coberta", "a_planejar", "sem_ordem", "atendida") if isinstance(rd, dict)}, ensure_ascii=False))

# ── A4. Conferência do início ────────────────────────────────────────────────
secao("A4. Parâmetro 52: o RN não começa antes das filhas")
st, r = call("PUT", "/api/planning-params/update", {"param_number": 52, "value": "BLOQUEAR"})
exige(st == 200, "parâmetro 52 = BLOQUEAR", r)
st, r = call("POST", f"/api/production-order/{rn1}/start", {"id": rn1, "start_date": hoje.isoformat()})
check(st == 422 and "ainda não foi concluída" in json.dumps(r, ensure_ascii=False), "BLOQUEAR: OF do RN recusada com as filhas abertas", (st, r))

# ── A5. Bucha ────────────────────────────────────────────────────────────────
secao("A5. OF da bucha: etapas na ordem do roteiro, entrega reservada para o RN")
st, r = call("POST", f"/api/production-order/{bu1}/start", {"id": bu1, "start_date": hoje.isoformat()})
exige(st == 200, "OF da bucha iniciada", r)
st, etapas = call("GET", f"/api/production-order/{bu1}/operations")
etapas = sorted(etapas or [], key=lambda e: e.get("sequence", 0))
exige(len(etapas) == 2, "a OF da bucha tem as 2 etapas do roteiro", etapas)
st, r = call("POST", "/api/production-order/operations/advance", {"operation_id": etapas[1]["id"], "status": "IN_PROGRESS", "actual_hours": 1, "produced_qty": 1})
check(st == 422, "etapa 20 não começa antes da 10", (st, r))
concluir_etapas(bu1, 20)


def entregar(of, qtd, final):
    return call("POST", f"/api/production-order/{of}/complete", {"id": of, "end_date": hoje.isoformat(), "warehouse_id": almox,
                                                                "quantity": qtd, "final": final, "idempotency_key": str(uuid.uuid4())})


def reservado(of):
    return d(sql(f"""SELECT COALESCE(SUM(quantity),0) FROM stock_reservations WHERE enterprise_id={emp}
        AND reference_type='OF_COMPONENTE' AND reference_code={of} AND status='ACTIVE'"""))


st, r = entregar(bu1, 20, True)
exige(st == 200, "bucha entregue (20)", r)
check(reservado(rn1) == 20, "as 20 buchas ficaram reservadas para a OF do RN da demanda 1", reservado(rn1))

# ── A6. Outra demanda não leva a bucha ───────────────────────────────────────
secao("A6. A OF do RN da demanda 2 não leva a bucha da demanda 1")
st, disp2 = call("GET", f"/api/production-order/{rn2}/material-availability")
linha2 = next((l for l in (disp2 or {}).get("lines", []) if int(l.get("legacy_item_code", l.get("item_code"))) == BU), {})
check(d(linha2.get("shortage")) == 10, "para a OF da demanda 2, a bucha está em falta (a da demanda 1 é reservada)", linha2)
st, r = call("PUT", "/api/planning-params/update", {"param_number": 52, "value": "AVISAR"})
st, ini2 = call("POST", f"/api/production-order/{rn2}/start", {"id": rn2, "start_date": hoje.isoformat()})
check(st == 200 and any("falta" in w for w in ini2.get("warnings") or []), "AVISAR: OF da demanda 2 inicia com aviso de falta", ini2)
st, r = call("POST", "/api/production-order/consumption", {"production_order_id": rn2, "item_code": BU, "consumed_qty": 10,
                                                          "warehouse_id": almox, "consumption_date": hoje.isoformat()})
check(st == 422 and "reservado" in json.dumps(r, ensure_ascii=False), "consumo da bucha na OF da demanda 2 recusado", (st, r))

# ── A7. Tubo, início e consumo do RN ─────────────────────────────────────────
secao("A7. Tubo pronto, RN começa e consome o que é dele")
st, r = call("PUT", "/api/planning-params/update", {"param_number": 52, "value": "BLOQUEAR"})
st, r = call("POST", f"/api/production-order/{tp1}/start", {"id": tp1, "start_date": hoje.isoformat()})
concluir_etapas(tp1, 10)
st, r = entregar(tp1, 10, True)
exige(st == 200, "tubo entregue (10)", r)
st, r = call("POST", f"/api/production-order/{rn1}/start", {"id": rn1, "start_date": hoje.isoformat()})
check(st == 200 and not r.get("warnings"), "BLOQUEAR: com as filhas concluídas a OF do RN começa", (st, r))
st, r = call("POST", "/api/production-order/consumption", {"production_order_id": rn1, "item_code": BU, "consumed_qty": 20,
                                                          "warehouse_id": almox, "consumption_date": hoje.isoformat()})
check(st == 201, "a OF do RN baixa as 20 buchas", (st, r))
st, r = call("POST", "/api/production-order/consumption", {"production_order_id": rn1, "item_code": TP, "consumed_qty": 10,
                                                          "warehouse_id": almox, "consumption_date": hoje.isoformat()})
check(st == 201, "a OF do RN baixa os 10 tubos", (st, r))

# ── A8. Encerramento: RN vai para o estoque e a demanda fica atendida ────────
secao("A8. OF do RN concluída: o RN entra no estoque e a demanda 1 fica coberta")
saldo_antes = d(sql(f"SELECT COALESCE(SUM(quantity),0) FROM stock_balances WHERE item_code={RN} AND warehouse_id={almox}"))
concluir_etapas(rn1, 10)
st, r = entregar(rn1, 10, True)
exige(st == 200, "RN entregue (10), OF concluída", r)
check(reservado(rn1) == 0, "nada mais reservado para a OF concluída", reservado(rn1))
saldo = d(sql(f"SELECT COALESCE(SUM(quantity),0) FROM stock_balances WHERE item_code={RN} AND warehouse_id={almox}"))
check(saldo - saldo_antes == 10, f"as 10 RN entraram no estoque ({saldo_antes} → {saldo})", saldo)
st, rd = call("GET", f"/api/independent-demand/get-by-code/{dem1}")
print("     demanda 1 depois:", json.dumps({k: rd.get(k) for k in ("situacao", "coberta", "a_planejar", "sem_ordem", "atendida", "produzida") if isinstance(rd, dict)}, ensure_ascii=False))
ordens = rd.get("ordens") or rd.get("orders") or []
check(any(str(o.get("status")) in ("COMPLETED", "CONCLUIDA", "FINISHED") for o in ordens), "a demanda 1 mostra a OF do RN concluída", ordens)
sug = rodar_mrp(650000)
check(not [s for s in do_item(sug, RN) if d(s["quantity"]) == 10], "novo MRP não sugere de novo o RN da demanda 1", do_item(sug, RN))

# ── B. Gerar OF direto da demanda ────────────────────────────────────────────
secao("B. Gerar OF direto da demanda: o MRP liga bucha e tubo a essa OF")
dem3, dem3_id = demanda(4, 70, 3)
st, r = call("POST", f"/api/independent-demand/{dem3}/gerar-of", {})
exige(st in (200, 201), "OF do RN gerada direto da demanda 3", (st, r))
rn3 = int(sql(f"SELECT production_order_id FROM independent_demand_orders WHERE enterprise_id={emp} AND demand_id={dem3_id} AND production_order_id IS NOT NULL"))
sug = rodar_mrp(700000)
check(not [s for s in do_item(sug, RN) if d(s["quantity"]) == 4], "o MRP não sugere de novo o RN que já tem OF", do_item(sug, RN))
peg = lambda it: [s for s in do_item(sug, it) if sql(f"""SELECT COUNT(*) FROM mrp_pegging WHERE enterprise_id={emp} AND plan_code={plano}
    AND supply_kind='SUGESTAO' AND supply_code={s['code']} AND demand_kind='ORDEM_PRODUCAO' AND demand_code={rn3}""") != "0"]
bu3, tp3 = peg(BU), peg(TP)
exige(len(bu3) == 1 and d(bu3[0]["quantity"]) == 8, "bucha (8) sugerida pelo empenho da OF gerada à mão", [(s["code"], s["quantity"]) for s in do_item(sug, BU)])
exige(len(tp3) == 1 and d(tp3[0]["quantity"]) == 4, "tubo (4) sugerido pelo empenho da OF gerada à mão", [(s["code"], s["quantity"]) for s in do_item(sug, TP)])
_, bu3_of = firmar(bu3[0])
_, tp3_of = firmar(tp3[0])
st, cad = call("GET", f"/api/production-order/{rn3}/chain")
check(sorted(f.get("production_order_id") for f in cad.get("children") or []) == sorted([bu3_of, tp3_of]),
      "a OF do RN gerada à mão mostra bucha e tubo como filhas", cad.get("children"))
st, cad = call("GET", f"/api/production-order/{bu3_of}/chain")
check([x.get("demand_code") for x in cad.get("independent_demands") or []] == [dem3],
      "a bucha mostra a demanda 3 (herdada da OF do RN)", cad.get("independent_demands"))
st, r = call("POST", f"/api/production-order/{rn3}/start", {"id": rn3, "start_date": hoje.isoformat()})
check(st == 422 and "ainda não foi concluída" in json.dumps(r, ensure_ascii=False), "BLOQUEAR vale também para a OF gerada à mão", (st, r))
st, r = call("PUT", "/api/planning-params/update", {"param_number": 52, "value": "AVISAR"})

# ── D. Venda consome a demanda para estoque ─────────────────────────────────
secao("D. Venda do RN consome a demanda para estoque: planeja o maior, não a soma")
dem4, dem4_id = demanda(6, 90, 5)
cliente = 700_000 + u % 100_000
sql(f"INSERT INTO customers(code,name,document_number,created_by,enterprise_id) VALUES({cliente},'CLIENTE ESTOQUE {u}','11444777000161','{ator}',{emp})")
so = int(sql(f"""INSERT INTO sales_orders(order_number,enterprise_code,status,customer_code,created_by,delivery_date)
    VALUES({u * 100 + 95},{EMPRESA},'P',{cliente},'{ator}',CURRENT_DATE+95) RETURNING code"""))
sql(f"""INSERT INTO sales_order_items(sales_order_code,sequence,item_code,requested_qty,unit_price,delivery_date)
    VALUES({so},1,{RN},4,100,CURRENT_DATE+95)""")
sug = rodar_mrp(750000)
janela_d = [s for s in do_item(sug, RN) if (hoje + timedelta(days=80)).isoformat() <= str(s["need_date"])[:10] <= (hoje + timedelta(days=96)).isoformat()]
total = sum(d(s["quantity"]) for s in janela_d)
check(total == 6, f"demanda 6 + venda 4 dentro da janela: o MRP planeja 6 RN, não 10 ({total})", [(s["need_date"], s["quantity"]) for s in janela_d])
bu_d = [s for s in do_item(sug, BU) if sql(f"""SELECT COUNT(*) FROM mrp_pegging WHERE enterprise_id={emp} AND plan_code={plano}
    AND supply_kind='SUGESTAO' AND supply_code={s['code']} AND demand_kind='SUGESTAO' AND demand_code IN ({",".join(str(x["code"]) for x in janela_d) or 0})""") != "0"]
check(sum(d(s["quantity"]) for s in bu_d) == 12, "a bucha segue o RN planejado (12 = 6 × 2)", [(s["code"], s["quantity"]) for s in bu_d])

# ── E. Demanda de finalidade própria (amostra) ───────────────────────────────
secao("E. Amostra: a cadeia abastece o RN e o RN pronto fica reservado para a demanda")
st, r = call("POST", "/api/independent-demand/create", {"code_demand": u * 10 + 6, "item_code": str(RN), "quantity": "2",
                                                        "demand_date": (hoje + timedelta(days=100)).isoformat(), "motivo": "AMOSTRA", "warehouse_id": almox})
exige(st == 201, "demanda de amostra RN ×2", (st, r))
dem5, dem5_id = int(r["code_demand"]), int(sql(f"SELECT id FROM independent_demands WHERE code={r['code_demand']} AND enterprise_id={emp}"))
st, r = call("POST", f"/api/independent-demand/{dem5}/gerar-of", {})
exige(st in (200, 201), "OF da amostra gerada direto da demanda", (st, r))
rn5 = int(sql(f"SELECT production_order_id FROM independent_demand_orders WHERE enterprise_id={emp} AND demand_id={dem5_id} AND production_order_id IS NOT NULL"))
sug = rodar_mrp(800000)
peg5 = lambda it: [s for s in do_item(sug, it) if sql(f"""SELECT COUNT(*) FROM mrp_pegging WHERE enterprise_id={emp} AND plan_code={plano}
    AND supply_kind='SUGESTAO' AND supply_code={s['code']} AND demand_kind='ORDEM_PRODUCAO' AND demand_code={rn5}""") != "0"]
bu5, tp5 = peg5(BU), peg5(TP)
exige(len(bu5) == 1 and d(bu5[0]["quantity"]) == 4 and len(tp5) == 1 and d(tp5[0]["quantity"]) == 2,
      "bucha (4) e tubo (2) da amostra sugeridos pelo empenho da OF", ([(s["code"], s["quantity"]) for s in do_item(sug, BU)], [(s["code"], s["quantity"]) for s in do_item(sug, TP)]))
_, bu5_of = firmar(bu5[0])
_, tp5_of = firmar(tp5[0])
for of, qtd in ((bu5_of, 4), (tp5_of, 2)):
    st, r = call("POST", f"/api/production-order/{of}/start", {"id": of, "start_date": hoje.isoformat()})
    concluir_etapas(of, qtd)
    st, r = entregar(of, qtd, True)
    exige(st == 200, f"OF {of} entregue ({qtd})", r)
check(reservado(rn5) == 6, "bucha (4) e tubo (2) reservados para a OF da amostra", reservado(rn5))
st, cad = call("GET", f"/api/production-order/{bu5_of}/chain")
check([(x.get("demand_code"), x.get("reason")) for x in cad.get("independent_demands") or []] == [(dem5, "AMOSTRA")],
      "a bucha mostra a demanda de amostra", cad.get("independent_demands"))
st, r = call("POST", f"/api/production-order/{rn5}/start", {"id": rn5, "start_date": hoje.isoformat()})
exige(st == 200, "OF da amostra iniciada", (st, r))
for it, q in ((BU, 4), (TP, 2)):
    st, r = call("POST", "/api/production-order/consumption", {"production_order_id": rn5, "item_code": it, "consumed_qty": q,
                                                              "warehouse_id": almox, "consumption_date": hoje.isoformat()})
    check(st == 201, f"a OF da amostra baixa {q} do item {it}", (st, r))
concluir_etapas(rn5, 2)
st, r = entregar(rn5, 2, True)
exige(st == 200, "amostra entregue (2)", r)
res_dem = d(sql(f"""SELECT COALESCE(SUM(quantity),0) FROM stock_reservations WHERE enterprise_id={emp} AND item_code={RN}
    AND status='ACTIVE' AND reference_type <> 'OF_COMPONENTE'"""))
check(res_dem == 2, f"as 2 RN da amostra ficaram reservadas para a demanda ({res_dem})", res_dem)
check(reservado(rn5) == 0, "nada sobrou reservado para a OF da amostra concluída", reservado(rn5))

# ── C. APS e CRP ─────────────────────────────────────────────────────────────
secao("C. APS e CRP: o RN para estoque é programado depois da bucha")
ops_wc, wc = {}, {}
for papel, n in (("bu", 3), ("rn", 4)):
    mt_code = u * 10 + n
    mt_id = int(sql(f"""INSERT INTO machine_types(code,name,type,created_by,enterprise_id,capacity_hours)
        VALUES({mt_code},'Centro {papel} {u}',(enum_range(NULL::machine_type_enum))[1],'{ator}',{emp},8) RETURNING id"""))
    wc[papel] = mt_id
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
st, r = call("POST", "/api/independent-demand/create", {"code_demand": u * 10 + 4, "item_code": str(rn_a), "quantity": "10",
                                                        "demand_date": (hoje + timedelta(days=81)).isoformat(), "motivo": "ESTOQUE", "warehouse_id": almox})
exige(st == 201, "demanda independente do RNA (10, para estoque)", (st, r))
sug = rodar_mrp(950000)
rna, bua = do_item(sug, rn_a), do_item(sug, bu_a)
exige(len(rna) == 1 and len(bua) == 1, "1 RN e 1 bucha", (rna, bua))
st, crp = call("POST", "/api/crp/calculate", {"plan_code": plano})
exige(st == 200, "CRP calculado com as sugestões da demanda para estoque", (st, crp))
st, cargas = call("GET", f"/api/crp/{plano}")
horas = lambda papel: sum(d(c.get("required_hours")) for c in cargas or [] if int(c.get("work_center_id", 0)) == wc[papel])
check(horas("rn") == 10, f"CRP: o centro do RN recebe 10 h (10 RN × 2 etapas × 30 min) — {horas('rn')}", horas("rn"))
check(horas("bu") == 20, f"CRP: o centro da bucha recebe 20 h (20 BU × 2 etapas × 30 min) — {horas('bu')}", horas("bu"))
bua_pl, bua_of = firmar(bua[0])
rna_pl, rna_of = firmar(rna[0])


def janela(of):
    r = sql(f"""SELECT to_char(MIN(scheduled_start),'YYYY-MM-DD"T"HH24:MI') || '|' || to_char(MAX(scheduled_end),'YYYY-MM-DD"T"HH24:MI')
        FROM production_sequences WHERE enterprise_id={emp} AND production_order_id={of}""")
    return r.split("|") if r else ["", ""]


inicio_aps = (hoje + timedelta(days=1)).isoformat() + "T08:00:00-03:00"
st, r = call("POST", "/api/aps/sequence", {"start_from": inicio_aps, "order_ids": [rna_of, bua_of]})
exige(st == 200, "APS programou as duas OFs", r)
ini_rn, fim_rn = janela(rna_of)
ini_bu, fim_bu = janela(bua_of)
exige(ini_rn and fim_bu, "as duas OFs têm programação", (ini_rn, fim_bu))
check(ini_rn >= fim_bu, f"o RN começa ({ini_rn}) depois de a bucha terminar ({fim_bu})", (ini_bu, fim_bu, ini_rn, fim_rn))
st, r = call("POST", "/api/aps/sequence", {"start_from": inicio_aps, "order_ids": [rna_of, bua_of], "direction": "BACKWARD"})
exige(st == 200, "APS regressivo (a partir da data da demanda)", r)
ini_rn3, fim_rn3 = janela(rna_of)
ini_bu3, fim_bu3 = janela(bua_of)
check(fim_bu3 <= ini_rn3, f"regressivo: a bucha termina ({fim_bu3}) até o RN começar ({ini_rn3})", (ini_bu3, fim_bu3, ini_rn3, fim_rn3))
check(fim_rn3[:10] <= (hoje + timedelta(days=81)).isoformat(), f"regressivo: o RN termina ({fim_rn3}) até a data da demanda", fim_rn3)
resumo()
PY
