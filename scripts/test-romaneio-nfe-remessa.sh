#!/usr/bin/env bash
# Simulação ponta a ponta dos bugs Y, U, R, Q, S, V, W, Z e H (rodada 7,
# .ai/tasks/rodada-validacao-bugs-e-melhorias.md).
#
# Roda CONTRA UMA CÓPIA do banco (grava romaneios, NF-e, OS de terceiro,
# movimentos de estoque e muda a configuração fiscal por alguns segundos).
# Nunca aponte para produção.
#
#   BASE_URL      API (ex.: http://127.0.0.1:5097)
#   ERP_EMAIL     usuário ADMIN da empresa
#   ERP_PASSWORD  senha
#   DATABASE_URL  o MESMO banco da API
#   CONFIRMO_QUE_E_COPIA=1   obrigatório
#
# A autorização na SEFAZ não acontece numa cópia (sem token da Focus): o roteiro
# marca a nota como autorizada e roda os efeitos pela rota de reprocesso — o
# mesmo código que a autorização executa. Sai com 1 se algum cenário falhar.
set -euo pipefail
: "${BASE_URL:?}" "${ERP_EMAIL:?}" "${ERP_PASSWORD:?}" "${DATABASE_URL:?}"
[[ "${CONFIRMO_QUE_E_COPIA:-}" == "1" ]] || { echo "defina CONFIRMO_QUE_E_COPIA=1 (o roteiro grava dados)"; exit 2; }
case "${DATABASE_URL}" in *venturerp_usimac*|*venturerp_truckparts*|*"/venturerp?"*|*"/venturerp") echo "recusado: parece o banco de produção"; exit 2;; esac
exec python3 -I - "$@" <<'PY'
import datetime, json, os, random, subprocess, urllib.error, urllib.request
from decimal import Decimal
B, DB = os.environ["BASE_URL"].rstrip("/"), os.environ["DATABASE_URL"]
falhas, total = [], 0
def sql(q):
    r = subprocess.run(["psql", DB, "-q", "-v", "ON_ERROR_STOP=1", "-Atc", q], capture_output=True, text=True)
    if r.returncode: raise SystemExit("psql: " + r.stderr + "\n" + q)
    return r.stdout.strip()
def api(m, path, body=None, tok=None):
    req = urllib.request.Request(B + path, data=None if body is None else json.dumps(body).encode(), method=m)
    req.add_header("Content-Type", "application/json")
    if tok: req.add_header("Authorization", "Bearer " + tok)
    try:
        with urllib.request.urlopen(req, timeout=120) as r:
            t = r.read().decode(); return r.status, (json.loads(t) if t.strip()[:1] in ("{", "[") else t)
    except urllib.error.HTTPError as e:
        t = e.read().decode()
        try: return e.code, json.loads(t)
        except Exception: return e.code, t
def ok(cond, cenario, extra=""):
    global total; total += 1
    print(("OK    " if cond else "FALHA ") + cenario + ("" if cond else f"  → {str(extra)[:400]}"))
    if not cond: falhas.append(cenario)
def secao(t): print(f"\n== {t}")
def dado(r, *ks):
    r = r.get("data", r) if isinstance(r, dict) else r
    for k in ks:
        if isinstance(r, dict) and k in r: return r[k]
    return None

if sql("select count(*) from fiscal_configs where coalesce(focus_nfe_token,'') <> ''") != "0":
    raise SystemExit("a cópia tem token da Focus: apague-o (UPDATE fiscal_configs SET focus_nfe_token=NULL) — a NF-e não pode sair daqui")
st, l = api("POST", "/users/login", {"email": os.environ["ERP_EMAIL"], "password": os.environ["ERP_PASSWORD"]})
if st != 200: raise SystemExit(f"login: {st} {l}")
T = l["token"]; hoje = datetime.date.today().isoformat()
EID = int(sql("select id from enterprise order by id limit 1"))

# ── Pedido → romaneio → NF-e ────────────────────────────────────────────────
secao("Romaneio e NF-e do pedido (U, Y, W, Z)")
# Pedido sem nota nem romaneio, com almoxarifado em toda linha e saldo livre
# nele: a reserva feita aqui tem de voltar depois do cancelamento da NF-e.
so = sql("""with livre as (select b.item_code, b.warehouse_id, sum(b.quantity - b.reserved_qty) q from stock_balances b group by 1, 2)
 select so.code from sales_orders so where so.status='P' and so.enterprise_code=(select code from enterprise where id=%d)
 and not exists(select 1 from shipments s where s.sales_order_code=so.code)
 and not exists(select 1 from fiscal_exits e where e.sales_order_code=so.code and e.status <> 'CANCELLED')
 and exists(select 1 from sales_order_items l where l.sales_order_code=so.code and l.is_active)
 and not exists (select 1 from sales_order_items l left join livre on livre.item_code=l.item_code and livre.warehouse_id=l.warehouse_code
   where l.sales_order_code=so.code and l.is_active and (l.warehouse_code is null
     or (l.requested_qty-coalesce(l.attended_qty,0)-coalesce(l.cancelled_qty,0)) > coalesce(livre.q,0)))
 order by so.code desc limit 1""" % EID)
if not so:
    ok(False, "há pedido confirmado com saldo em estoque na cópia", "nenhum")
else:
    so = int(so)
    pendente = lambda: Decimal(sql(f"select coalesce(sum(requested_qty-attended_qty-cancelled_qty),0) from sales_order_items where sales_order_code={so} and is_active"))
    atendido = lambda: Decimal(sql(f"select coalesce(sum(attended_qty),0) from sales_order_items where sales_order_code={so} and is_active"))
    status = lambda: sql(f"select status from sales_orders where code={so}")
    reservado = lambda: Decimal(sql(f"select coalesce(sum(quantity),0) from stock_reservations where status='ACTIVE' and reference_type='SALES_ORDER' and reference_code={so}"))
    for linha, item, almox, qtd in [x.split("|") for x in sql(f"""select l.code, i.business_code, l.warehouse_code, l.requested_qty-l.attended_qty-l.cancelled_qty
            from sales_order_items l join items i on i.code=l.item_code where l.sales_order_code={so} and l.is_active""").splitlines()]:
        st, r = api("POST", "/api/stock/reservations/create", {"item_code": item, "mask": "", "warehouse_id": int(almox), "quantity": qtd,
            "reference_type": "SALES_ORDER", "reference_code": so, "reference_item_code": int(linha)}, T)
        if st >= 300: ok(False, f"reserva da linha {linha} do pedido {so}", (st, r))
    reserva_antes = reservado()
    ok(reserva_antes > 0, f"pedido {so} reservado antes da NF-e", reserva_antes)
    st, r = api("POST", "/api/shipments/auto-fill/sales-order", {"sales_order_code": so}, T)
    rom = dado(r, "code")
    ok(st < 300 and rom, f"U: 1º romaneio do pedido {so}", r)
    st, r = api("POST", "/api/shipments/auto-fill/sales-order", {"sales_order_code": so}, T)
    ok(st == 422 and "saldo" in str(r), "U: 2º romaneio com o mesmo saldo é recusado", (st, r))

    _, pv = api("GET", f"/api/fiscal/exits/sales-order/{so}/previa", tok=T)
    nota_dto = {"serie": "1", "data_emissao": hoje, "cfop": dado(pv, "cfop_sugerido") or "5102",
                "natureza_operacao": dado(pv, "natureza_sugerida") or "VENDA"}
    def emitir():
        st, r = api("POST", f"/api/shipments/{rom}/nfe", nota_dto, T)
        return st, r, dado(r, "id")
    st, r, e1 = emitir()
    ok(st < 300 and e1, "Y: NF-e do romaneio criada", (st, r))
    if e1:
        sem_linha = sql(f"select count(*) from fiscal_exit_items where fiscal_exit_id={e1} and sales_order_item_code is null")
        ok(sem_linha == "0", "Y: toda linha da NF-e aponta a linha do pedido", f"{sem_linha} linha(s) sem")
        ok(sql(f"select nfe_number is null from shipments where code={rom}") == "t", "Z: romaneio sem número enquanto a nota é rascunho (antes: NF-e 0)")
        st, r = api("POST", "/api/fiscal/exits/from-sales-order", dict(nota_dto, sales_order_code=so), T)
        ok(st >= 400, "U: faturar o pedido de novo com tudo em nota é recusado", (st, r))
        st, r = api("POST", f"/api/shipments/{rom}/nfe", nota_dto, T)
        ok(st >= 400, "U: 2ª NF-e do mesmo romaneio é recusada", (st, r))
        st, r = api("POST", f"/api/shipments/{rom}/cancel", {"reason": "teste"}, T)
        ok(st == 422 and "NF-e" in str(r), "W: cancelar o romaneio com NF-e é recusado", (st, r))
        st, r = api("POST", f"/api/fiscal/exits/{e1}/descartar", {"motivo": "romaneio refeito"}, T)
        ok(st == 200 and sql(f"select status||'|'||coalesce(chave_acesso,'') from fiscal_exits where id={e1}") == "CANCELLED|", "W: rascunho descartado (cancelado, sem chave)", (st, r))
        ok(sql(f"select fiscal_exit_id is null from shipments where code={rom}") == "t", "W: o descarte solta o romaneio")
        st, r = api("POST", f"/api/fiscal/exits/{e1}/descartar", {}, T)
        ok(st == 422, "W: descartar de novo é recusado", (st, r))

    st, r, e2 = emitir()
    ok(st < 300 and e2, "W: depois do descarte o romaneio fatura de novo", (st, r))
    if e2:
        # Autorização simulada: o que a Focus devolveria + os efeitos pela rota de reprocesso.
        numero = int(sql("select coalesce(max(numero_nf),0)+1 from fiscal_exits"))
        chave = "35" + "".join(random.choice("0123456789") for _ in range(42))
        sql(f"""update fiscal_exits set status='AUTHORIZED', numero_nf={numero}, chave_acesso='{chave}', protocolo='1' where id={e2};
                insert into fiscal_exit_pendencias(enterprise_id, fiscal_exit_id, etapa) values ({EID}, {e2}, 'AUTORIZACAO')""")
        st, r = api("POST", f"/api/fiscal/exits/{e2}/reprocessar-efeitos", {}, T)
        ok(st == 200 and sql(f"select count(*) from fiscal_exit_pendencias where fiscal_exit_id={e2} and situacao='PENDENTE'") == "0", "Y: efeitos da autorização aplicados", (st, r))
        ok(pendente() == 0 and atendido() > 0, "Y: a autorização soma o atendido das linhas", f"pendente {pendente()} atendido {atendido()}")
        ok(status() == "F", "Y: pedido faturado", status())
        ok(sql(f"select nfe_number::text||'|'||nfe_key from shipments where code={rom}") == f"{numero}|{chave}", "Z: romaneio com o número e a chave da nota autorizada",
           sql(f"select nfe_number, nfe_key from shipments where code={rom}"))
        st, r = api("POST", "/api/shipments/auto-fill/sales-order", {"sales_order_code": so}, T)
        ok(st == 422, "U: pedido faturado não monta outro romaneio", (st, r))

        st, r = api("POST", f"/api/fiscal/exits/{e2}/cancel", {"justificativa": "teste de cancelamento da rodada 7"}, T)
        ok(st == 200, "Y: NF-e autorizada cancelada", (st, r))
        ok(atendido() == 0 and status() == "P", "Y: o cancelamento devolve o atendido e o pedido volta a pedido", f"atendido {atendido()} status {status()}")
        ok(reservado() == reserva_antes, "Y: a reserva do pedido volta como estava", f"antes {reserva_antes}, agora {reservado()}")
        ok(sql(f"select fiscal_exit_id is null from shipments where code={rom}") == "t", "Y: o romaneio é desligado da nota cancelada")
        st, r, e3 = emitir()
        ok(st < 300 and e3, "Y: depois do cancelamento o pedido fatura de novo (antes: 'já foi faturado por inteiro')", (st, r))
        if e3: api("POST", f"/api/fiscal/exits/{e3}/descartar", {}, T)
        st, r = api("POST", f"/api/shipments/{rom}/cancel", {"reason": "fim da simulação"}, T)
        ok(st < 300, "W: sem nota viva, o romaneio cancela", (st, r))

# ── Industrialização por terceiro: remessa, NF-e, reparo e saldo em poder ──
secao("Remessa ao terceiro (R, Q, S), reparo das remessas antigas e saldo em poder (Z)")
ator = sql(f"select id from users where email='{os.environ['ERP_EMAIL']}'")
cods = sql(f"""select i.code||'|'||i.business_code||'|'||b.warehouse_id from items i
   join stock_balances b on b.item_code=i.code and b.enterprise_id=i.enterprise_id and b.mask='' and b.quantity - b.reserved_qty > 50
   join warehouse w on w.id=b.warehouse_id and w.location::text not in ('TERCEIRO','INSPECAO','REJEICAO','TRANSITO') and w.supplier_code is null
   join fiscal_classifications fc on fc.enterprise_id=i.enterprise_id and fc.code::text=COALESCE(NULLIF(i.accounting_sale_fiscal_classification_code,''), i.accounting_purchase_fiscal_classification_code)
  where i.enterprise_id={EID} and coalesce(fc.ncm,'') <> '' order by i.code limit 2""").splitlines()
if len(cods) < 2:
    ok(False, "há dois itens com NCM e saldo para montar a OS de terceiro", cods)
else:
    (peca, peca_bc, almox), (acabado, acabado_bc, almox_acab) = [c.split("|") for c in cods]
    n = random.randint(10**8, 9*10**8)
    forn = int(sql(f"""insert into suppliers(code,name,document_type,document_number,created_by,enterprise_id)
        values ({n},'TRATE simulação','CNPJ','11222333000181','{ator}',{EID}) returning code"""))
    sql(f"""insert into supplier_addresses(supplier_id,uf,street,number,neighborhood,city,zip_code,is_default)
        select id,'SP','Rua A','1','Centro','São Paulo','01001000',true from suppliers where code={forn};
      insert into item_preferred_suppliers(enterprise_id,item_code,supplier_code,mask,ranking,is_preferred,created_by)
        values ({EID},{acabado},{forn},'',1,true,'{ator}')""")
    op = sql(f"insert into operations(code,name,origin,lead_time_days,created_by,enterprise_id) values ({n},'Zincagem','TERCEIROS',7,'{ator}',{EID}) returning id")
    rot = sql(f"insert into manufacturing_routes(code,item_code,alternative,is_standard,created_by,enterprise_id) values ({n},{acabado},{100 + n % 20000},false,'{ator}',{EID}) returning id")
    ro10 = sql(f"insert into route_operations(route_id,sequence,operation_id) values ({rot},10,{op}) returning id")
    ro20 = sql(f"insert into route_operations(route_id,sequence,operation_id) values ({rot},20,{op}) returning id")
    of = sql(f"insert into production_orders(order_number,item_code,mask,planned_qty,status,route_id,enterprise_id,created_by) values ({n % 1000000},{acabado},'',40,'OPEN',{rot},{EID},'{ator}') returning id")
    sql(f"""insert into production_order_materials(production_order_id,enterprise_id,material_kind,item_code,quantity,warehouse_id,created_by,route_operation_sequence)
            values ({of},{EID},'DEMAND',{peca},40,{almox},'{ator}',10)""")
    def nova_os(ro, tipo):
        return int(sql(f"""insert into third_party_service_orders(code,enterprise_id,production_order_id,route_operation_id,operation_id,item_code,supplier_code,uom,quantity,start_date,due_date,status,created_by,remittance_type)
            values ((select coalesce(max(code),0)+1 from third_party_service_orders where enterprise_id={EID}),{EID},{of},{ro},{op},{acabado},{forn},'UN',40,current_date,current_date+10,'RELEASED_WITHOUT_PO','{ator}','{tipo}') returning id"""))
    terc = lambda: sql(f"select id from warehouse where enterprise_id={EID} and supplier_code={forn}")
    saldo = lambda item, w: Decimal(sql(f"select coalesce(sum(quantity),0) from stock_balances where enterprise_id={EID} and item_code={item} and warehouse_id={w}") or "0")
    def remeter(os_id, q, w):
        st, r = api("POST", f"/api/third-party-services/orders/{os_id}/movements", {"movement_type": "REMITTANCE", "quantity": str(q),
            "occurred_at": datetime.datetime.now().astimezone().isoformat(), "idempotency_key": f"sim-{os_id}-{random.random()}", "warehouse_id": int(w)}, T)
        return st, r, dado(r, "id")

    os1 = nova_os(ro10, "DEMAND_ITEMS")
    acab_antes, peca_antes = saldo(acabado, almox_acab), saldo(peca, almox)
    st, r, mov = remeter(os1, 5, almox)
    ok(st < 300 and mov, "Q: remessa da OS registrada", (st, r))
    ok(saldo(peca, almox) == peca_antes - 5 and saldo(peca, terc()) == 5, "Q: a remessa leva a PEÇA da etapa ao almoxarifado do terceiro",
       f"peça fábrica {peca_antes}→{saldo(peca, almox)}, terceiro {saldo(peca, terc())}")
    ok(saldo(acabado, almox_acab) == acab_antes and saldo(acabado, terc()) == 0, "Q: o item da OF (acabado) não se move")

    st, r = api("POST", f"/api/third-party-services/orders/{os1}/nfe-remessa", {"movement_id": mov, "quantity": "5", "data_emissao": hoje}, T)
    nf = dado(r, "id")
    ok(st < 300 and nf, "R: NF-e de remessa gravada (origem com 24 caracteres)", (st, r))
    if nf:
        ok(sql(f"select source_type from fiscal_exits where id={nf}") == "REMESSA_INDUSTRIALIZACAO", "R: origem REMESSA_INDUSTRIALIZACAO na nota")
        ok(sql(f"select item_code from fiscal_exit_items where fiscal_exit_id={nf}") == peca, "Q: a NF-e de remessa leva a peça")
        ok(Decimal(sql(f"select unit_price from fiscal_exit_items where fiscal_exit_id={nf}")) > 0, "Q: a peça sai pelo custo médio (não zero)")
        st, r = api("POST", f"/api/fiscal/exits/{nf}/descartar", {}, T)
        ok(st == 200, "W: rascunho da remessa descartado", (st, r))

    st, r = api("GET", "/api/third-party-services/em-poder", tok=T)
    linhas = [x for x in (r if isinstance(r, list) else (r or {}).get("data") or []) if str(x.get("supplier_code")) == str(forn)]
    ok(len(linhas) == 1 and "in_hands" in linhas[0] and "InHands" not in linhas[0], "Z: saldo em poder em snake_case", r if not linhas else linhas[0])
    if linhas:
        ok(Decimal(str(linhas[0]["value"])) > 0, "Q: o valor em poder do terceiro não é zero", linhas[0])

    # S: no PR a suspensão exige cBenef. A configuração fiscal da cópia muda e volta.
    cfg = sql("select id||'|'||uf_empresa||'|'||coalesce(cbenef_suspensao,'') from fiscal_configs order by id limit 1").split("|")
    try:
        sql(f"update fiscal_configs set uf_empresa='PR', cbenef_suspensao=NULL where id={cfg[0]}")
        st, r, mov2 = remeter(os1, 2, almox)
        st, r = api("POST", f"/api/third-party-services/orders/{os1}/nfe-remessa", {"movement_id": mov2, "quantity": "2", "data_emissao": hoje}, T)
        ok(st == 422 and "cBenef" in str(r), "S: no PR a remessa sem cBenef é recusada antes da SEFAZ", (st, r))
        sql(f"update fiscal_configs set cbenef_suspensao='PR999999' where id={cfg[0]}")
        st, r = api("POST", f"/api/third-party-services/orders/{os1}/nfe-remessa", {"movement_id": mov2, "quantity": "2", "data_emissao": hoje}, T)
        nf2 = dado(r, "id")
        ok(st < 300 and nf2 and sql(f"select codigo_beneficio_fiscal from fiscal_exit_items where fiscal_exit_id={nf2}") == "PR999999",
           "S: com o cBenef da configuração a nota sai com ele", (st, r))
        if nf2:
            campos = lambda pv: [p.get("campo", "") for p in (dado(pv, "pendencias") or [])]
            st, pv = api("GET", f"/api/fiscal/exits/{nf2}/previa", tok=T)
            ok(st == 200 and not any("codigo_beneficio_fiscal" in c for c in campos(pv)), "S: a prévia não acusa cBenef quando ele está na linha", campos(pv))
            sql(f"update fiscal_exit_items set codigo_beneficio_fiscal=NULL where fiscal_exit_id={nf2}")
            st, pv = api("GET", f"/api/fiscal/exits/{nf2}/previa", tok=T)
            ok(st == 200 and any("codigo_beneficio_fiscal" in c for c in campos(pv)), "S: a prévia acusa a linha sem cBenef", campos(pv))
            st, r = api("POST", f"/api/fiscal/exits/{nf2}/authorize", {}, T)
            ok(st >= 400 and "cBenef" in str(r), "S: a autorização recusa a linha sem cBenef (antes: erro 930 da SEFAZ)", (st, r))
            api("POST", f"/api/fiscal/exits/{nf2}/descartar", {}, T)
    finally:
        sql(f"update fiscal_configs set uf_empresa='{cfg[1]}', cbenef_suspensao={repr(cfg[2]) if cfg[2] else 'NULL'} where id={cfg[0]}")

    # Remessa antiga (antes do Q): a OS levou o acabado. O reparo devolve e manda a peça.
    os2 = nova_os(ro20, "ORDER_ITEM")
    acab_antes, peca_antes = saldo(acabado, almox_acab), saldo(peca, almox)
    st, r, _ = remeter(os2, 3, almox_acab)
    ok(st < 300 and saldo(acabado, terc()) == 3, "reparo: simulada a remessa antiga (acabado no terceiro)", (st, r))
    sql(f"update third_party_service_orders set remittance_type='DEMAND_ITEMS' where id={os2}")
    code2 = sql(f"select code from third_party_service_orders where id={os2}")
    st, r = api("POST", "/api/third-party-services/corrigir-remessas", {}, T)
    sim = [x for x in (r if isinstance(r, list) else []) if str(x.get("service_order_code")) == code2]
    ok(st == 200 and len(sim) == 1 and sim[0]["status"] == "SIMULADA" and saldo(acabado, terc()) == 3, "reparo: simulação lista a OS sem gravar", (st, r))
    st, r = api("POST", "/api/third-party-services/corrigir-remessas?gravar=true", {}, T)
    feitas = [x for x in (r if isinstance(r, list) else []) if str(x.get("service_order_code")) == code2]
    ok(st == 200 and len(feitas) == 1 and feitas[0]["status"] == "CORRIGIDA", "reparo: OS corrigida", (st, r))
    # Pelos movimentos desta OS: o reparo também corrige outras OS antigas da cópia.
    movs = lambda item, tipo: Decimal(sql(f"""select coalesce(sum(quantity),0) from stock_movements where enterprise_id={EID}
        and reference_type='OS_TERCEIRO' and reference_code={code2} and item_code={item} and warehouse_id={terc()} and movement_type='{tipo}'"""))
    ok(saldo(acabado, terc()) == 0 and movs(acabado, "TRANSFER_OUT") == 3 and movs(peca, "TRANSFER_IN") == 3,
       "reparo: o acabado volta e a peça vai ao terceiro", f"acabado no terceiro {saldo(acabado, terc())}, acabado saiu {movs(acabado, 'TRANSFER_OUT')}, peça entrou {movs(peca, 'TRANSFER_IN')}")
    st, r = api("POST", "/api/third-party-services/corrigir-remessas?gravar=true", {}, T)
    ok(st == 200 and not [x for x in (r if isinstance(r, list) else []) if str(x.get("service_order_code")) == code2], "reparo: repetir não refaz", (st, r))

# ── Promessa de entrega ─────────────────────────────────────────────────────
secao("Promessa de entrega (V)")
def dias_uteis(n):
    d = datetime.date.today()
    while n > 0:
        d += datetime.timedelta(days=1)
        if d.weekday() < 5: n -= 1
    return d.isoformat()
ator = sql(f"select id from users where email='{os.environ['ERP_EMAIL']}'")
n = random.randint(10**8, 9*10**8)
comprado, fabricado = 9_100_000_000 + n % 10**8, 9_200_000_000 + n % 10**8
sql(f"""insert into items(code,business_code,warehouse_code,created_by,enterprise_id,engineering_type,planning_purchase_lead_time_days)
          values ({comprado},'SIM-C-{n}',1,'{ator}',{EID},1,5);
        insert into items(code,business_code,warehouse_code,created_by,enterprise_id,engineering_type,planning_manufacturing_lead_time_days)
          values ({fabricado},'SIM-F-{n}',1,'{ator}',{EID},0,0);
        insert into suppliers(code,name,document_type,document_number,created_by,enterprise_id) values ({n},'Fornecedor sem prazo','CNPJ','{n}','{ator}',{EID});
        insert into item_preferred_suppliers(enterprise_id,item_code,supplier_code,mask,ranking,is_preferred,created_by,lead_time_days)
          values ({EID},{comprado},{n},'',1,true,'{ator}',0)""")
op = sql(f"insert into operations(code,name,origin,lead_time_days,created_by,enterprise_id) values ({n},'Tratamento','TERCEIROS',7,'{ator}',{EID}) returning id")
rot = sql(f"insert into manufacturing_routes(code,item_code,alternative,is_standard,created_by,enterprise_id) values ({n},{fabricado},1,true,'{ator}',{EID}) returning id")
sql(f"insert into route_operations(route_id,sequence,operation_id) values ({rot},10,{op})")
st, r = api("POST", "/api/delivery-promise/simular", {"lines": [{"item_code": f"SIM-C-{n}", "quantity": "50"}, {"item_code": f"SIM-F-{n}", "quantity": "50"}]}, T)
lst = r if isinstance(r, list) else (r or {}).get("data") or []
datas = [str(x.get("promised_date", ""))[:10] for x in lst]
ok(st == 200 and len(datas) == 2 and datas[0] == dias_uteis(5), "V: comprado com prazo 0 no fornecedor promete pelos 5 dias do item (como o MRP)", (st, datas, r))
ok(len(datas) == 2 and datas[1] == dias_uteis(5), "V: fabricado soma a etapa no terceiro (7 corridos ≈ 5 úteis)", (datas, r))

# ── Poeira de conversão ─────────────────────────────────────────────────────
secao("Saída sem poeira (H)")
item_h = 9_300_000_000 + n % 10**8
sql(f"insert into items(code,business_code,warehouse_code,created_by,enterprise_id) values ({item_h},'SIM-H-{n}',1,'{ator}',{EID})")
almox_h = sql(f"select id from warehouse where enterprise_id={EID} and supplier_code is null and location::text not in ('TERCEIRO','INSPECAO','REJEICAO','TRANSITO') order by id limit 1")
mov = lambda t, q, p: api("POST", "/api/stock/movements/create", {"item_code": f"SIM-H-{n}", "mask": "", "warehouse_id": int(almox_h), "movement_type": t, "quantity": q, "unit_price": p, "total_price": "0"}, T)
st, r = mov("IN", "1", "400")
ok(st < 300, "H: entrada de 1 chapa", (st, r))
for _ in range(3):
    st, r = mov("OUT", "0.333333", "0")
    if st >= 300: ok(False, "H: saída de 1/3", (st, r))
sobra = sql(f"select coalesce(sum(quantity),0) from stock_balances where enterprise_id={EID} and item_code={item_h}")
ok(Decimal(sobra) == 0, "H: três saídas de 1/3 zeram o saldo (antes sobrava 0,000001)", sobra)

print(f"\n{total - len(falhas)}/{total} cenários OK")
if falhas:
    print("Falharam:\n  " + "\n  ".join(falhas)); raise SystemExit(1)
PY
