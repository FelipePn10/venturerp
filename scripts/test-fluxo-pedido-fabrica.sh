#!/usr/bin/env bash
set -euo pipefail

# Aceite do fluxo do pedido de venda até a fábrica, pela API, com conferência
# dos números: pedido → MRP → sugestões → firmar → pedido de compra →
# recebimento → OF → consumo → apontamento → entrega → CRP → APS → execução
# automática.
#
# Requer uma API no ar sobre um banco com o seed do demo (make demo-seed ou
# scripts/seed-demo.sql). O roteiro cria o próprio plano, tabela de preço e
# pedido; pode rodar mais de uma vez no mesmo banco.
#
# Variáveis:
#   BASE_URL        (padrão http://localhost:5072 — a API do demo)
#   ADMIN_EMAIL     (padrão admin@panossoerp.demo)
#   ADMIN_PASSWORD  (obrigatória — a senha do administrador do demo)
#   ITEM            (padrão 10002 — acabado do seed com estrutura e roteiro)
#   QTD             (padrão 5000 — acima do estoque, para forçar o planejamento)
#
# Uso: ADMIN_PASSWORD=... BASE_URL=http://localhost:5072 bash scripts/test-fluxo-pedido-fabrica.sh

: "${ADMIN_PASSWORD:?Informe ADMIN_PASSWORD (senha do administrador do demo)}"
export BASE_URL="${BASE_URL:-http://localhost:5072}"
export ADMIN_EMAIL="${ADMIN_EMAIL:-admin@panossoerp.demo}"
export ITEM="${ITEM:-10002}"
export QTD="${QTD:-5000}"

command -v python3 >/dev/null || { echo "dependência ausente: python3" >&2; exit 1; }

exec python3 - <<'PY'
import json, os, sys, uuid, urllib.request, urllib.error
from datetime import date, timedelta
from decimal import Decimal as D

BASE = os.environ["BASE_URL"].rstrip("/")
ITEM = int(os.environ["ITEM"])
QTD = D(os.environ["QTD"])
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


def secao(t):
    print(f"\n\033[1;36m── {t}\033[0m")


def check(cond, label, detalhe=""):
    global ok_n, fail_n
    if cond:
        ok_n += 1
        print(f"\033[0;32m  ✓ {label}\033[0m")
    else:
        fail_n += 1
        print(f"\033[0;31m  ✗ {label}\033[0m {str(detalhe)[:300]}")
    return cond


def exige(cond, label, detalhe=""):
    if not check(cond, label, detalhe):
        resumo()


def resumo():
    print(f"\nTotal: {ok_n + fail_n}   \033[0;32mPASS: {ok_n}\033[0m   \033[0;31mFAIL: {fail_n}\033[0m")
    sys.exit(1 if fail_n else 0)


def d(v):
    return D(str(v if v is not None else 0))


def legado(row):
    """Código interno do item numa linha da API (item_code pode vir como código de negócio)."""
    v = row.get("legacy_item_code", row.get("item_code"))
    return int(v) if v is not None and str(v).lstrip("-").isdigit() else None


def saldo(item):
    st, rows = call("GET", f"/api/stock/balances/item/{item}")
    return sum((d(r.get("quantity")) for r in rows or []), D(0)) if st == 200 else D(0)


def sugestoes(plano):
    st, rows = call("GET", f"/api/mrp-calculation/suggestions/{plano}")
    return rows or [] if st == 200 else []


def pipeline(plano):
    return call("POST", "/api/planning/run-pipeline", {
        "plan_code": plano, "initial_order_number": 500000, "generate_llc": True,
        "start_from": date.today().isoformat() + "T08:00:00-03:00"})


hoje = date.today()
entrega = (hoje + timedelta(days=120)).isoformat()

# ── 0. Acesso ────────────────────────────────────────────────────────────────
secao("0. API e acesso")
st, h = call("GET", "/health")
exige(st == 200, "API no ar", h)
st, login = call("POST", "/users/login", {"email": os.environ["ADMIN_EMAIL"], "password": os.environ["ADMIN_PASSWORD"]})
exige(st == 200 and isinstance(login, dict) and login.get("token"), "login do administrador", login)
TOKEN = login["token"]

# ── 1. Cadastros de apoio do roteiro ─────────────────────────────────────────
secao("1. Plano e tabela de preço")
st, planos = call("GET", "/api/production-plan/list")
plano = max([int(p["code"]) for p in planos or []] + [9000]) + 1
st, r = call("POST", "/api/production-plan/create", {
    "code": plano, "name": f"Aceite fluxo {plano}", "independent_demands": "ALL",
    "group_same_date_orders": True, "planning_types": ["MRP", "MPS"]})
exige(st == 201, f"plano de produção {plano} criado", r)

st, tabela = call("POST", "/api/customers/support/sales-tables", {
    "description": f"Aceite fluxo {plano}", "price_formation": "INFORMADO", "decimal_places": 2,
    "composition": "EXWORK", "table_type": "NORMAL", "base_date": "PEDIDO"})
exige(st == 201, "tabela de preço criada", tabela)
st, r = call("POST", f"/api/customers/support/sales-tables/{tabela['code']}/prices/", {
    "sales_table_id": tabela["id"], "sales_table_code": tabela["code"], "item_code": str(ITEM),
    "price": 50, "price_conv": 50, "situation": "ATIVO"})
exige(st in (200, 201), "preço do item na tabela", r)

# Outros pedidos em carteira sem suprimento (de testes manuais, por exemplo)
# também viram sugestão do item: a rodada de base, antes do pedido do roteiro,
# mede isso para a conferência olhar só o acréscimo que o pedido causa.
st, r = pipeline(plano)
exige(st == 200, "rodada de base do planejamento (antes do pedido)", r)
base_fab = sum(d(s["quantity"]) for s in sugestoes(plano) if legado(s) == ITEM and s["order_type"] == "FABRICACAO")
if base_fab > 0:
    print(f"    (já havia {base_fab} un. a fabricar do item por outros pedidos em carteira)")


# ── 2. Pedido de venda ───────────────────────────────────────────────────────
secao("2. Pedido de venda")
capa = {"enterprise_code": 1, "status": "R", "origin": "NORMAL", "emission_date": hoje.isoformat(),
        "delivery_date": entrega, "customer_code": 2, "plan_code": plano, "currency_code": "BRL",
        "price_table_code": tabela["code"]}
st, vazio = call("POST", "/api/sales-order/create", capa)
exige(st == 201, "pedido (rascunho) criado", vazio)
st, r = call("PATCH", f"/api/sales-order/{vazio['code']}/status", {"status": "P"})
check(st == 422, "pedido sem itens NÃO pode ser confirmado", (st, r))
st, r = call("PATCH", f"/api/sales-order/{vazio['code']}/status", {"status": "A"})
check(st == 422, "pedido sem itens NÃO vai para análise", (st, r))

pedido = vazio["code"]
st, linha = call("POST", "/api/sales-order/items/create", {
    "sales_order_code": pedido, "sequence": 1, "item_code": str(ITEM), "mask": "",
    "digit_date": hoje.isoformat(), "warehouse_code": 1, "requested_qty": float(QTD),
    "unit_price": 50, "delivery_date": entrega})
exige(st == 201, f"linha de {QTD} un. do item {ITEM}", linha)
check(d(linha.get("balance")) == QTD, "saldo da linha = quantidade pedida", linha.get("balance"))

st, r = call("PATCH", f"/api/sales-order/{pedido}/status", {"status": "P"})
exige(st == 204, "pedido confirmado", (st, r))
st, so = call("GET", f"/api/sales-order/{pedido}")
if so.get("is_blocked"):
    print("    (bloqueado pelo limite de crédito — liberando)")
    st, r = call("PATCH", f"/api/sales-order/{pedido}/unblock", {"reason": "aceite do fluxo"})
    st, so = call("GET", f"/api/sales-order/{pedido}")
check(so.get("status") == "P" and not so.get("is_blocked"), "pedido em 'P' e liberado", (so.get("status"), so.get("is_blocked")))

# ── 3. Planejamento (MRP → CRP → APS) ────────────────────────────────────────
secao("3. Planejamento: MRP → CRP → APS")
estoque_acabado = saldo(ITEM)
st, res = pipeline(plano)
exige(st == 200, "pipeline executado", res)
# Com a fábrica carregada, uma ordem pode não caber na agenda da máquina: o MRP
# termina "com erros" dizendo qual, e as demais continuam programadas.
check(res.get("mrp_status") in ("COMPLETED", "COMPLETED_WITH_ERRORS"), f"MRP concluído ({res.get('mrp_status')})", res)
check(res.get("mrp_orders", 0) > 0, f"MRP gerou sugestões ({res.get('mrp_orders')})")
check(res.get("crp_entries", 0) > 0, f"CRP calculou carga ({res.get('crp_entries')} centro/dia)")

sug = sugestoes(plano)
fab = [s for s in sug if legado(s) == ITEM and s["order_type"] == "FABRICACAO"]
exige(len(fab) >= 1, "sugestão de FABRICAÇÃO para o item do pedido", [s for s in sug if legado(s) == ITEM])
qtd_fab = sum(d(s["quantity"]) for s in fab)
acrescimo = qtd_fab - base_fab
check(D(0) < acrescimo <= QTD, f"o pedido acrescenta {acrescimo} a fabricar (total {qtd_fab}), entre 0 e o pedido ({QTD})")
check(qtd_fab >= QTD - estoque_acabado, f"cobre o que o estoque ({estoque_acabado}) não cobre")
check(all(d(s["quantity"]) > 0 for s in sug), "nenhuma sugestão com quantidade zero",
      [s for s in sug if d(s["quantity"]) <= 0][:2])
com_maquina = [s for s in sug if s["order_type"] == "FABRICACAO" and s.get("machine_code")]
check(len(com_maquina) >= 1, f"ordens de fabricação programadas em máquina ({len(com_maquina)})")
s0 = fab[0]
check(str(s0.get("start_date"))[:10] < str(s0.get("need_date"))[:10],
      f"prazo do roteiro aplicado (início {str(s0.get('start_date'))[:10]} < necessidade {str(s0.get('need_date'))[:10]})")

compras = [s for s in sug if s["order_type"] == "COMPRA" and str(s.get("parent_item_code")) == str(ITEM)]
check(len(compras) >= 1, f"componentes explodidos em sugestões de COMPRA ({len(compras)})")
for c in compras:
    check(str(c.get("need_date"))[:10] <= str(s0.get("need_date"))[:10],
          f"componente {legado(c)} chega antes da entrega do acabado")

st, res2 = pipeline(plano)
sug2 = sugestoes(plano)
fab2 = [s for s in sug2 if legado(s) == ITEM and s["order_type"] == "FABRICACAO"]
check(st == 200 and sum(d(s["quantity"]) for s in fab2) == qtd_fab and len(sug2) == len(sug),
      "recálculo é idempotente (mesmas sugestões, sem duplicar)", (len(sug), len(sug2)))

st, exc = call("GET", f"/api/mrp-calculation/exceptions/{plano}")
check(st == 200 and isinstance(exc, list), f"mensagens de exceção consultadas ({len(exc or [])})")

# ── 4. Firmar a fabricação → OF ──────────────────────────────────────────────
secao("4. Firmar a sugestão de fabricação → ordem de fabricação")
alvo = sorted(fab2, key=lambda s: -d(s["quantity"]))[0]
st, firmada = call("POST", f"/api/mrp-calculation/suggestions/{alvo['code']}/firm", {})
exige(st in (200, 201) and firmada.get("is_firm") is True, "sugestão firmada (ordem planejada firme)", firmada)
st, de_novo = call("POST", f"/api/mrp-calculation/suggestions/{alvo['code']}/firm", {})
check(st in (200, 201) and de_novo.get("planned_code") == firmada.get("planned_code"),
      "firmar de novo não cria outra ordem planejada")

st, planejadas = call("GET", "/api/planned-order/list")
planejada = next((p for p in planejadas or [] if p.get("code") == firmada.get("planned_code")), None)
exige(planejada is not None and planejada.get("is_firm"), "ordem planejada firme na lista", firmada)
st, ofs = call("GET", "/api/production-order/list")
minhas = [o for o in ofs or [] if o.get("planned_order_id") == planejada.get("id")]
exige(len(minhas) == 1, "exatamente UMA OF gerada pela firmação", [o.get("id") for o in minhas])
of = minhas[0]
check(of.get("status") == "OPEN", "OF nasce aberta", of.get("status"))

st, res3 = pipeline(plano)
fab3 = [s for s in sugestoes(plano) if legado(s) == ITEM and s["order_type"] == "FABRICACAO"]
check(sum(d(s["quantity"]) for s in fab3) == qtd_fab - d(alvo["quantity"]),
      "recálculo desconta a OF firme (não sugere de novo o que já virou ordem)",
      [s["quantity"] for s in fab3])

# ── 5. Compra: firmar → pedido de compra → recebimento ───────────────────────
secao("5. Compra do componente")
comp = [s for s in sugestoes(plano) if s["order_type"] == "COMPRA" and d(s["quantity"]) > 0]
exige(len(comp) >= 1, "há sugestão de compra para firmar")
c = sorted(comp, key=lambda s: -d(s["quantity"]))[0]
item_c, qtd_c = legado(c), d(c["quantity"])
st, cf = call("POST", f"/api/mrp-calculation/suggestions/{c['code']}/firm", {})
exige(st in (200, 201) and cf.get("is_firm"), f"sugestão de compra firmada (item {item_c}, {qtd_c})", cf)
st, fila = call("GET", "/api/purchase-order/suggestions")
check(any(o.get("code") == cf["planned_code"] for o in fila or []), "aparece na fila do comprador")
st, r = call("POST", f"/api/purchase-order/suggestions/{cf['planned_code']}/approve", {"unit_price": "3.50"})
check(st == 422, "aprovar sem fornecedor é recusado", (st, r))
st, pc = call("POST", f"/api/purchase-order/suggestions/{cf['planned_code']}/approve", {"supplier_code": 1, "unit_price": "3.50"})
exige(st == 201 and pc.get("items"), "pedido de compra gerado", pc)
pci = pc["items"][0]
check(d(pci["requested_qty"]) == qtd_c, "quantidade do pedido de compra = sugestão", pci["requested_qty"])
st, fila = call("GET", "/api/purchase-order/suggestions")
check(not any(o.get("code") == cf["planned_code"] for o in fila or []), "saiu da fila do comprador")

antes = saldo(item_c)
almox = pci.get("warehouse_id") or 1
metade = (qtd_c / 2).quantize(D("0.0001"))
st, r = call("POST", f"/api/purchase-order/{pc['code']}/receipts", {"items": [
    {"purchase_order_item_code": pci["code"], "quantity": float(metade), "warehouse_id": almox}]})
check(st == 200, "recebimento parcial", r)
check(saldo(item_c) == antes + metade, f"estoque subiu exatamente o recebido ({metade})", saldo(item_c) - antes)
st, r = call("POST", f"/api/purchase-order/{pc['code']}/receipts", {"items": [
    {"purchase_order_item_code": pci["code"], "quantity": float(qtd_c * 3), "warehouse_id": almox}]})
check(st == 422, "receber além do saldo do pedido é recusado", (st, r))
st, r = call("POST", f"/api/purchase-order/{pc['code']}/receipts", {"items": [
    {"purchase_order_item_code": pci["code"], "quantity": float(qtd_c - metade), "warehouse_id": almox}]})
check(st == 200 and saldo(item_c) == antes + qtd_c, "recebimento total: estoque = anterior + pedido", saldo(item_c) - antes)
st, pc2 = call("GET", f"/api/purchase-order/{pc['code']}")
check(pc2.get("status") == "RECEIVED", "pedido de compra recebido", pc2.get("status"))

st, _ = pipeline(plano)
resto = [s for s in sugestoes(plano) if s["order_type"] == "COMPRA" and legado(s) == item_c]
check(sum((d(s["quantity"]) for s in resto), D(0)) < qtd_c,
      "recálculo não repete a compra já recebida", [s["quantity"] for s in resto])
check(all(d(s["quantity"]) > 0 for s in sugestoes(plano)), "sem sugestão de quantidade zero depois da compra")

# ── 6. Produção ──────────────────────────────────────────────────────────────
secao("6. Ordem de fabricação: materiais, consumo, apontamento e entrega")
oid = of["id"]
st, mats = call("GET", f"/api/production-order/{oid}/materials")
check(st == 200 and len(mats or []) >= 1, f"OF com materiais empenhados ({len(mats or [])})", mats)
st, disp = call("GET", f"/api/production-order/{oid}/material-availability")
check(st == 200 and "lines" in (disp or {}), "disponibilidade de materiais da OF")
st, ops = call("GET", f"/api/production-order/{oid}/operations")
exige(st == 200 and len(ops or []) >= 1, f"OF com etapas do roteiro ({len(ops or [])})", ops)

st, ini = call("POST", f"/api/production-order/{oid}/start", {"id": oid, "start_date": hoje.isoformat()})
exige(st == 200, "OF iniciada", ini)
check(ini.get("status") == "IN_PROGRESS" and "Status" not in ini,
      "resposta do início no contrato da API (snake_case)", list(ini.keys())[:4])

m_item = int(mats[0].get("legacy_item_code", item_c))
m_antes = saldo(m_item)
consumir = min(m_antes, D("10")).quantize(D("0.0001"))
if consumir > 0:
    st, cons = call("POST", "/api/production-order/consumption", {
        "production_order_id": oid, "item_code": m_item, "consumed_qty": float(consumir),
        "warehouse_id": mats[0].get("warehouse_id") or 1, "consumption_date": hoje.isoformat()})
    check(st == 201 and "consumed_qty" in (cons or {}), "consumo lançado (snake_case)", cons)
    check(saldo(m_item) == m_antes - consumir, f"estoque do insumo baixou exatamente {consumir}", m_antes - saldo(m_item))
st, r = call("POST", "/api/production-order/consumption", {
    "production_order_id": oid, "item_code": m_item, "consumed_qty": 99999999,
    "warehouse_id": mats[0].get("warehouse_id") or 1, "consumption_date": hoje.isoformat()})
check(st == 422, "consumo acima do saldo é recusado", (st, r))

apont = {"production_order_id": oid, "employee_id": 1, "machine_id": 1,
         "appointment_date": hoje.isoformat(), "produced_qty": 10, "scrapped_qty": 0}
st, r = call("POST", "/api/production-order/appointment", {**apont, "start_time": "2026-01-01T08:00:00-03:00"})
check(st == 422, "hora em formato inválido é recusada com mensagem (não erro interno)", (st, r))
st, r = call("POST", "/api/production-order/appointment", {**apont, "employee_id": None})
check(st == 422, "apontamento sem operador é recusado", (st, r))
st, ap = call("POST", "/api/production-order/appointment", {**apont, "start_time": "08:00", "end_time": "16:00"})
check(st == 201 and "produced_qty" in (ap or {}), "apontamento lançado (snake_case)", ap)

st, etapa = call("POST", "/api/production-order/operations/advance",
                 {"operation_id": ops[0]["id"], "status": "IN_PROGRESS", "actual_hours": 8, "produced_qty": 10})
check(st == 200 and etapa.get("status") == "IN_PROGRESS", "primeira etapa em andamento", etapa)
if len(ops) > 1:
    st, r = call("POST", "/api/production-order/operations/advance",
                 {"operation_id": ops[-1]["id"], "status": "IN_PROGRESS", "actual_hours": 1, "produced_qty": 1})
    check(st == 422, "etapa seguinte não começa antes da predecessora", (st, r))

fg_antes = saldo(ITEM)
entregar = {"id": oid, "end_date": hoje.isoformat(), "warehouse_id": 2, "quantity": 10, "final": False,
            "idempotency_key": f"aceite-{plano}", "lot": f"L-ACEITE-{plano}"}
st, ent = call("POST", f"/api/production-order/{oid}/complete", entregar)
check(st == 200 and saldo(ITEM) == fg_antes + 10, "entrega parcial: acabado entrou no estoque (+10)", saldo(ITEM) - fg_antes)
st, ent2 = call("POST", f"/api/production-order/{oid}/complete", entregar)
check(saldo(ITEM) == fg_antes + 10, "repetir a mesma entrega não entra em dobro", saldo(ITEM) - fg_antes)
st, of2 = call("GET", f"/api/production-order/{oid}")
check(d(of2.get("produced_qty")) >= 10 and of2.get("status") == "IN_PROGRESS", "OF segue em andamento com o produzido", of2)

# ── 7. Capacidade e sequenciamento ───────────────────────────────────────────
secao("7. CRP e APS")
st, res4 = pipeline(plano)
exige(st == 200, "pipeline com a OF em andamento", res4)
check(res4.get("crp_entries", 0) > 0, f"CRP com carga ({res4.get('crp_entries')})")
check(res4.get("aps_operations", 0) > 0, f"APS sequenciou operações ({res4.get('aps_operations')})")
st, carga = call("GET", f"/api/crp/{plano}")
check(st == 200 and all(d(e["required_hours"]) > 0 for e in carga or []), "toda linha de carga tem horas requeridas")
st, gantt = call("GET", f"/api/aps/gantt/order/{oid}")
check(st == 200 and len(gantt or []) >= 1, "OF aparece no Gantt", gantt)
if gantt:
    check(all(g["scheduled_end"] > g["scheduled_start"] for g in gantt), "toda operação termina depois de começar")
    fim = max(g["scheduled_end"] for g in gantt)[:10]
    necessidade = str(alvo.get("need_date"))[:10]
    atrasada = oid in (res4.get("aps_late_orders") or [])
    check(atrasada == (fim > necessidade),
          f"atraso apontado corretamente (fim {fim}, necessidade {necessidade}, atrasada={atrasada})")
    check(res4.get("viable") is False if atrasada else True, "plano com ordem atrasada não é dado como viável")
st, regr = call("POST", "/api/aps/sequence", {"start_from": hoje.isoformat() + "T08:00:00-03:00", "direction": "BACKWARD"})
check(st == 200 and regr.get("scheduled_operations", 0) > 0, "programação regressiva executa", regr)

# ── 8. Parâmetros e execução automática ──────────────────────────────────────
secao("8. Parâmetros e execução automática")
st, p51 = call("GET", "/api/planning-params/51")
check(st == 200 and str(p51.get("value")) == "30", "horizonte de reprogramação ligado por padrão (30 dias)", p51)
st, r = call("PUT", "/api/planning-params/update", {"param_number": 51, "value": "-1"})
check(st == 422, "valor inválido do parâmetro é recusado", (st, r))
st, auto = call("GET", "/api/planning/auto-run")
check(st == 200 and "run_hour" in (auto or {}), "parametrização da execução automática consultada")

# ── 9. Pegging ───────────────────────────────────────────────────────────────
secao("9. Rastreamento")
st, peg = call("GET", f"/api/mrp-calculation/pegging/sales-order/{pedido}")
check(st == 200, "pegging do pedido de venda consultado", peg)

resumo()
PY
