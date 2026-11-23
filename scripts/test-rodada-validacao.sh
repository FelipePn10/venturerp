#!/usr/bin/env bash
# Simulação ponta a ponta da rodada de validação (.ai/tasks/rodada-validacao-bugs-e-melhorias.md).
#
# Roda CONTRA UMA CÓPIA do banco (grava pedidos, romaneios, NF-e em rascunho,
# baixas, links do portal). Nunca aponte para produção.
#
#   BASE_URL      API (ex.: http://127.0.0.1:5099)
#   ERP_EMAIL     usuário ADMIN da empresa
#   ERP_PASSWORD  senha
#   DATABASE_URL  o MESMO banco da API (as conferências leem o resultado nele)
#   CONFIRMO_QUE_E_COPIA=1   obrigatório
#   INTER_WEBHOOK_TOKEN      opcional: o segredo configurado na API (testa 200/503)
#
# Sai com código 1 se algum cenário falhar. Pré-requisito: Focus sem token na
# cópia (a transmissão da NF-e deve PARAR com mensagem) e integrações externas
# desligadas (o roteiro confere as mensagens de "não configurada").
set -euo pipefail
: "${BASE_URL:?}" "${ERP_EMAIL:?}" "${ERP_PASSWORD:?}" "${DATABASE_URL:?}"
[[ "${CONFIRMO_QUE_E_COPIA:-}" == "1" ]] || { echo "defina CONFIRMO_QUE_E_COPIA=1 (o roteiro grava dados)"; exit 2; }
case "${DATABASE_URL}" in *venturerp_usimac*|*venturerp_truckparts*|*"/venturerp?"*|*"/venturerp") echo "recusado: parece o banco de produção"; exit 2;; esac
exec python3 -I - "$@" <<'PY'
import datetime, json, os, subprocess, sys, urllib.error, urllib.request
from decimal import Decimal
B, DB = os.environ["BASE_URL"].rstrip("/"), os.environ["DATABASE_URL"]
falhas, total = [], 0
def sql(q):
    r = subprocess.run(["psql", DB, "-Atc", q], capture_output=True, text=True)
    if r.returncode: raise SystemExit("psql: " + r.stderr)
    return r.stdout.strip()
def api(m, path, body=None, tok=None, raw=False):
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
    print(("OK    " if cond else "FALHA ") + cenario + ("" if cond else f"  → {str(extra)[:300]}"))
    if not cond: falhas.append(cenario)
def secao(t): print(f"\n== {t}")

st, l = api("POST", "/users/login", {"email": os.environ["ERP_EMAIL"], "password": os.environ["ERP_PASSWORD"]})
if st != 200: raise SystemExit(f"login: {st} {l}")
T = l["token"]; hoje = datetime.date.today().isoformat()
empresa_code = sql("select code from enterprise order by id limit 1")

secao("Expedição guiada (bugs J, L e melhoria 6)")
so = sql("""with livre as (select item_code, sum(quantity) q from stock_balances group by 1),
 res as (select item_code, sum(quantity) q from stock_reservations where status='ACTIVE' group by 1)
 select so.code from sales_orders so where so.status='P' and not exists(select 1 from shipments s where s.sales_order_code=so.code)
 and exists(select 1 from sales_order_items l where l.sales_order_code=so.code and l.is_active)
 and not exists (select 1 from sales_order_items l left join livre on livre.item_code=l.item_code left join res on res.item_code=l.item_code
   where l.sales_order_code=so.code and l.is_active and (l.requested_qty-coalesce(l.attended_qty,0)-coalesce(l.cancelled_qty,0)) > coalesce(livre.q,0)-coalesce(res.q,0))
 order by so.code desc limit 1""")
if not so:
    ok(False, "há pedido confirmado com saldo em estoque para simular a expedição", "nenhum pedido adequado na cópia")
else:
    so = int(so); passos = []
    for _ in range(8):
        st, f = api("GET", f"/api/shipments/fluxo/pedido/{so}", tok=T)
        acoes = [s["next_action"] for s in (f.get("shipments") or []) if s.get("next_action")] or ([f["next_action"]] if f.get("next_action") else [])
        if not acoes: break
        a = acoes[0]; passos.append(a["code"])
        if a["code"] == "MONTAR_ROMANEIO": st, r = api("POST", a["path"], {"sales_order_code": so}, T)
        elif a["code"] == "CONFERIR":
            code = a["path"].split("/")[3]; _, det = api("GET", f"/api/shipments/{code}", tok=T)
            for it in (det.get("items") or []):
                if not it.get("is_conferred"): api("POST", f"/api/shipments/{code}/items/confer", {"item_id": it["id"], "conferred_qty": it["quantity"]}, T)
            st, r = api("POST", a["path"], {}, T)
        elif a["code"] == "EMITIR_NFE":
            _, pv = api("GET", f"/api/fiscal/exits/sales-order/{so}/previa", tok=T)
            st, r = api("POST", a["path"], {"serie": "1", "data_emissao": hoje, "cfop": (pv or {}).get("cfop_sugerido") or "5101",
                "natureza_operacao": (pv or {}).get("natureza_sugerida") or "VENDA", "valor_frete": "0", "valor_seguro": "0", "valor_desconto": "0"}, T)
        else: st, r = api(a["method"], a["path"], {}, T)
        if st >= 300: break
    ok(passos[:5] == ["MONTAR_ROMANEIO", "SEPARAR", "CONFERIR", "EMITIR_NFE", "AUTORIZAR_NFE"], "pedido → romaneio → separação → conferência → NF-e", passos)
    ok(st == 422 and "Focus" in str(r), "transmissão sem token Focus para com mensagem (sem 500)", f"{st} {r}")
    sh = sql(f"select status||'|'||(fiscal_exit_id is not null) from shipments where sales_order_code={so} order by code desc limit 1")
    ok(sh == "CONFERRED|true", "romaneio conferido e ligado à NF-e", sh)
    ok(Decimal(sql(f"select coalesce(sum(quantity),0) from stock_reservations where reference_type='SALES_ORDER' and reference_code={so} and status='ACTIVE'") or 0) > 0,
       "separação reservou para o pedido no depósito padrão do item")
    romaneio = sql(f"select code from shipments where sales_order_code={so} order by code desc limit 1")
    st, r = api("POST", f"/api/shipments/{romaneio}/cotacao-online", {}, T)
    ok(st == 422, "cotação de frete sem Frete Rápido: 422 explicando", f"{st} {r}")

secao("Repetir pedido (melhoria 13) e Portal do cliente (fase 0)")
cli = sql(f"select so.customer_code from sales_orders so join sales_order_items l on l.sales_order_code=so.code where so.status in ('P','F') and so.enterprise_code={empresa_code} and coalesce(l.price_table_code, so.price_table_code) is not null group by 1 order by count(*) desc limit 1")
outro = sql(f"select customer_code from sales_orders where enterprise_code={empresa_code} and status in ('P','F') and customer_code <> {cli or 0} limit 1") if cli else ""
if not cli:
    ok(False, "há cliente com pedido com tabela de preço", "nenhum na cópia")
else:
    st, link = api("POST", f"/api/customers/{cli}/portal-links", {"days": 7, "description": "simulação"}, T)
    ok(st == 201 and len(link.get("token", "")) >= 40, "gerar link do portal", f"{st} {link}")
    tk = link.get("token", "")
    # A cópia guarda os pedidos do portal de rodadas anteriores; como o portal
    # barra repetir o que ainda aguarda (e mais de 5 abertos), a simulação
    # começa cancelando as sobras deste cliente — só na cópia (guarda no topo).
    sql(f"update sales_orders set status='CANCELLED' where origin='PORTAL' and status='R' and customer_code={cli} and enterprise_code={empresa_code}")
    st, res = api("GET", f"/api/portal/{tk}")
    ok(st == 200 and res.get("orders"), "cliente abre o portal e vê os pedidos dele", f"{st} {str(res)[:200]}")
    meus = {p["code"] for p in res.get("orders", [])}
    ok(all(sql(f"select customer_code from sales_orders where code={c}") == cli for c in meus), "portal só mostra pedidos do próprio cliente")
    if outro:
        alheio = sql(f"select code from sales_orders where customer_code={outro} and enterprise_code={empresa_code} and status in ('P','F') limit 1")
        st, _ = api("POST", f"/api/portal/{tk}/pedidos/{alheio}/repetir", {"note": "x"})
        ok(st == 404, "portal não repete pedido de outro cliente", st)
    alvo = sql(f"select so.code from sales_orders so join sales_order_items l on l.sales_order_code=so.code where so.customer_code={cli} and so.enterprise_code={empresa_code} and so.status in ('P','F') and coalesce(l.price_table_code, so.price_table_code) is not null limit 1")
    st, r = api("POST", f"/api/portal/{tk}/pedidos/{alvo}/repetir", {"note": "simulação do portal"})
    ok(st == 201 and r.get("lines", 0) >= 1, "cliente pede de novo pelo portal", f"{st} {r}")
    ok(sql(f"select count(*) from sales_orders where origin='PORTAL' and status='R' and customer_code={cli}") != "0", "o pedido do portal vira rascunho PORTAL")
    st, _ = api("POST", f"/api/portal/{tk}/pedidos/{alvo}/repetir", {"note": "de novo"})
    ok(st == 422, "o mesmo pedido não é repetido enquanto o anterior aguarda", st)
    orc = sql(f"select code from sales_orders where customer_code={cli} and enterprise_code={empresa_code} and status='OF' limit 1")
    if orc:
        ok(api("POST", f"/api/portal/{tk}/pedidos/{orc}/repetir", {"note": "x"})[0] == 404, "orçamento não se repete pelo portal")
    ok(sql(f"select count(*) from notification_outbox where event_key='COMERCIAL_PEDIDO_PORTAL' and aggregate_public_id='{r.get('order_number')}'") == "1", "alerta COMERCIAL_PEDIDO_PORTAL registrado para o comercial")
    sem = sql(f"select so.code from sales_orders so where so.customer_code={cli} and so.enterprise_code={empresa_code} and so.status in ('P','F') and so.price_table_code is null and exists(select 1 from sales_order_items l where l.sales_order_code=so.code and l.is_active and l.price_table_code is null) limit 1")
    if sem:
        st, r2 = api("POST", f"/api/portal/{tk}/pedidos/{sem}/repetir", {"note": "sem tabela"})
        ok(st == 201 and sql(f"select status from sales_orders where order_number={r2.get('order_number', 0)} and enterprise_code={empresa_code}") == "R",
           "pedido do portal sem tabela de preço chega mesmo assim (rascunho com a lista)", f"{st} {r2}")
        _, res3 = api("GET", f"/api/portal/{tk}")
        p3 = [p for p in res3.get("orders", []) if p["number"] == r2.get("order_number")]
        ok(bool(p3) and p3[0]["lines"] and all(l.get("to_be_priced") for l in p3[0]["lines"]), "cliente vê os itens que pediu, com preço a confirmar", str(p3)[:200])
    st, pend = api("GET", "/api/pendencias/dia", tok=T)
    ok(any(p.get("key") == "pedidos_portal" for p in (pend or [])), "Central de Pendências mostra o pedido do portal", str(pend)[:200])
    ok(int(sql(f"select count(*) from portal_events where link_id={link.get('id', 0)}")) >= 2, "auditoria do portal (acesso + pedido)")
    st, _ = api("DELETE", f"/api/customers/portal-links/{link.get('id')}", tok=T)
    st2, _ = api("GET", f"/api/portal/{tk}")
    ok(st == 204 and st2 == 404, "link revogado deixa de abrir", f"{st} {st2}")
    ok(api("GET", "/api/portal/token-inventado-que-nao-existe-xxxxxxxx")[0] == 404, "token inventado: 404")
    st, r = api("POST", f"/api/sales-order/{alvo}/repetir", tok=T)
    ok(st == 201, "vendedor repete o pedido (VVND0200)", f"{st} {r}")
    vazio = sql(f"select so.code from sales_orders so where so.enterprise_code={empresa_code} and not exists(select 1 from sales_order_items l where l.sales_order_code=so.code and l.is_active and l.requested_qty>0) limit 1")
    if vazio:
        antes = sql("select max(code) from sales_orders")
        st, _ = api("POST", f"/api/sales-order/{vazio}/repetir", tok=T)
        ok(st == 422 and sql(f"select count(*) from sales_orders where code>{antes} and status<>'CANCELLED'") == "0", "repetir pedido sem linha não deixa rascunho vazio")

secao("Portal: notas fiscais (DANFE e XML)")
nota = sql(f"select x.id||'|'||so.customer_code from fiscal_exits x join sales_orders so on so.code=x.sales_order_code and so.enterprise_code={empresa_code} where x.status='AUTHORIZED' and x.danfe_path is not null limit 1")
if not nota:
    print("      (nenhuma nota com DANFE gravada na cópia: download não simulado — em produção vem da Focus)")
else:
    nid, ncli = nota.split("|")
    _, lk = api("POST", f"/api/customers/{ncli}/portal-links", {"days": 1}, T)
    req = urllib.request.Request(f"{B}/api/portal/{lk['token']}/notas/{nid}/danfe")
    try:
        with urllib.request.urlopen(req, timeout=60) as r3: ok(r3.status == 200 and r3.headers.get("Content-Type") == "application/pdf", "cliente baixa a DANFE em PDF")
    except urllib.error.HTTPError as e: ok(False, "cliente baixa a DANFE em PDF", e.code)
    alheia = sql(f"select x.id from fiscal_exits x join sales_orders so on so.code=x.sales_order_code where x.status='AUTHORIZED' and so.customer_code<>{ncli} limit 1")
    ok(api("GET", f"/api/portal/{lk['token']}/notas/{alheia}/xml")[0] == 404, "nota de outro cliente: 404")

secao("Cobrança, retorno CNAB e aviso do banco (melhoria 10)")
tit = sql("select id from contas_receber where status in ('PENDENTE','VENCIDO') and is_active and coalesce(desconto,0)=0 order by id desc limit 1")
st, r = api("POST", f"/api/financial/contas-receber/{tit}/pix", {}, T)
ok(st == 422 and "Inter" in str(r), "boleto+PIX sem Banco Inter: 422 explicando", f"{st} {r}")
st, r = api("POST", f"/api/financial/contas-receber/{tit}/aviso", {"channel": "WHATSAPP", "to": ["44999990000"]}, T)
ok(st == 422 and st != 500, "aviso por WhatsApp desligado: 422", f"{st} {r}")
bruto, rec = sql(f"select valor_bruto, coalesce(valor_recebido,0) from contas_receber where id={tit}").split("|")
saldo = Decimal(bruto) - Decimal(rec); nn = f"9{tit}"[-10:]
sql(f"update contas_receber set nosso_numero='{nn}' where id={tit}")
conta = sql("select id from contas_bancarias where is_active order by id limit 1")
def linha(c):
    b = [" "] * 240
    for p, v in c.items(): b[p-1:p-1+len(v)] = list(v)
    return "".join(b)
cent = lambda d: str(int((d * 100).quantize(Decimal(1)))).zfill(15)
dmy = datetime.date.today().strftime("%d%m%Y")
arq = "\r\n".join([linha({1: "077", 8: "0"}),
    linha({8: "3", 14: "T", 16: "06", 38: nn.zfill(20), 82: cent(saldo)}),
    linha({8: "3", 14: "U", 16: "06", 18: cent(Decimal("1.50")), 33: cent(Decimal(0)), 78: cent(saldo + Decimal("1.50")), 138: dmy, 146: dmy}),
    linha({8: "3", 14: "T", 16: "06", 38: "88888888".zfill(20), 82: cent(Decimal(10))}),
    linha({8: "3", 14: "U", 16: "06", 78: cent(Decimal(10)), 138: dmy, 146: dmy})])
st, r = api("POST", "/api/financial/retorno-bancario", {"conta_bancaria_id": int(conta), "content": arq}, T)
ok(st == 200 and r.get("settled") == [nn] and r.get("not_found") == ["88888888"], "retorno CNAB baixa o título e lista o desconhecido", f"{st} {r}")
ok(sql(f"select status||'|'||coalesce(juros,0) from contas_receber where id={tit}") == "RECEBIDO|1.50", "juros informados pelo banco entram como encargo")
st, r = api("POST", "/api/financial/retorno-bancario", {"conta_bancaria_id": int(conta), "content": arq}, T)
ok(r.get("already_settled") == [nn] and sql(f"select count(*) from fluxo_caixa where contas_receber_id={tit}") == "1", "segunda leitura do mesmo retorno não baixa de novo", r)
W = os.environ.get("INTER_WEBHOOK_TOKEN", "")
ok(api("POST", "/api/integracoes/inter/webhook/segredo-errado", [])[0] == 404, "webhook do Inter com segredo errado: 404")
if W:
    ok(api("POST", f"/api/integracoes/inter/webhook/{W}", [])[0] == 200, "webhook de teste (lista vazia): 200")
    ok(api("POST", f"/api/integracoes/inter/webhook/{W}", [{"codigoSolicitacao": "x"}])[0] == 503, "webhook com banco desligado: 503 (o Inter reenvia)")
st, s = api("GET", "/api/integracoes/status", tok=T)
ok(st == 200 and s.get("bank") is False and s.get("whatsapp") is False, "situação das integrações (todas desligadas na cópia)", s)

secao("Pedido de compra por WhatsApp (melhoria 13)")
po = sql(f"select code from purchase_orders where enterprise_code={empresa_code} and status in ('APPROVED','PARTIAL') limit 1")
if po:
    st, r = api("POST", f"/api/purchase-order/{po}/enviar-whatsapp", {"para": ["(44) 99999-0000"]}, T)
    ok(st == 422 and "WhatsApp" in str(r), "WhatsApp do pedido de compra desligado: 422 explicando", f"{st} {r}")
    st, r = api("POST", f"/api/purchase-order/{po}/enviar-whatsapp", {"para": ["123"]}, T)
    ok(st == 422, "celular inválido é recusado", f"{st} {r}")

secao("Assistente, promessa, custo, indicadores, pendências, implantação, terceiros, simulação")
item = sql("select i.code from items i join stock_balances b on b.item_code=i.code group by i.code having sum(b.quantity) > 10 order by i.code limit 1")
bc = sql(f"select coalesce(nullif(business_code,''), code::text) from items where code={item}")
med = {"item_code": bc, "shape": "CHAPA", "material": "ACO", "length_mm": "1000", "width_mm": "500", "thickness_mm": "3"}
st, r = api("POST", "/api/item-conversions/assistente", med, T)
ok(st == 200 and r.get("weight_kg") == "11.775", "assistente: chapa 1000×500×3 aço = 11,775 kg", f"{st} {r}")
st, r = api("POST", "/api/item-conversions/assistente", {k: v for k, v in med.items() if k != "thickness_mm"}, T)
ok(st == 422, "assistente: chapa sem espessura recusada", st)
livre = Decimal(sql(f"select coalesce(sum(quantity),0) from stock_balances where item_code={item}"))
st, r = api("POST", "/api/delivery-promise/simular", {"lines": [{"item_code": bc, "quantity": "1"}, {"item_code": bc, "quantity": str(int(livre) + 1000)}]}, T)
ok(st == 200 and r[0]["source"] == "ESTOQUE" and r[1]["source"] != "ESTOQUE", "promessa: a segunda linha disputa o estoque da primeira", f"{st} {str(r)[:200]}")
for path, cond, nome in [
    ("/api/implantacao/painel", lambda r: "percent" in r, "painel de implantação"),
    ("/api/pendencias/dia", lambda r: isinstance(r, list), "central de pendências"),
    ("/api/indicadores", lambda r: "sales" in r, "indicadores (vendas do período, OEE, giro)"),
    ("/api/financial/fluxo-projetado-pedidos?dias=60", lambda r: len(r) >= 8, "fluxo projetado com pedidos"),
    ("/api/standard-cost/variacoes", lambda r: "orders" in r, "variações padrão × real"),
    ("/api/third-party-services/em-poder", lambda r: isinstance(r, list), "em poder de terceiros"),
    ("/api/system/simulacao/status", lambda r: "state" in r, "situação do modo simulação"),
]:
    st, r = api("GET", path, tok=T); ok(st == 200 and cond(r), nome, f"{st} {str(r)[:200]}")
for path, esperado, nome in [("/api/financial/fluxo-projetado-pedidos?dias=0", 422, "fluxo com 0 dias recusado"),
    ("/api/indicadores?de=2026-12-01&ate=2026-01-01", 422, "indicadores com período invertido recusado"),
    ("/api/standard-cost/margem/pedido/99999999", 404, "margem de pedido inexistente: 404")]:
    ok(api("GET", path, tok=T)[0] == esperado, nome)
st, r = api("GET", "/api/financial/dre-gerencial", tok=T)
ok(st in (200, 422), "DRE pelo plano: 200 ou 422 pedindo o plano", f"{st} {r}")

print(f"\n{total - len(falhas)} de {total} cenários OK")
if falhas:
    print("FALHAS:"); [print(" -", f) for f in falhas]; sys.exit(1)
PY
