# Fluxo completo de um produto — do Pedido de Venda à Entrega

Documenta o caminho **real, hoje**, de um produto final pelo ERP: entrada do pedido de
venda → planejamento (MRP) → capacidade/sequenciamento (CRP/APS) → ordens
(produção e compra, com aprovação/rejeição) → recebimento no almoxarifado →
fabricação → estoque de acabados → saída fiscal.

Usa como exemplo o **Suporte Soldado SS-100** cadastrado em
[`cadastros-item.md`](cadastros-item.md).

> Convenções: `Authorization: Bearer <JWT>`, `application/json`.
> Pontos marcados com **✅ automático** já foram implementados (ver `../MELHORIAS.txt`);
> os marcados com **⚙️ melhoria** seguem pendentes.

---

## Visão geral

```
Pedido de Venda (cliente)                         Cadastros de apoio
      │  PlanCode                                 (item, BOM, roteiro,
      ▼                                            fornecedor, fiscal)
Demanda do plano  ─────────────┐
 (demanda independente +       │
  previsão de vendas)          ▼
                           MRP (run)
        explosão BOM (LLC) · necessidades líquidas · lead time (CPM do roteiro)
                               │
            ┌──────────────────┼───────────────────────┐
            ▼                  ▼                        ▼
   Ordens planejadas      Exceções MRP            (CRP capacidade /
   PRODUCTION / PURCHASE  (atrasos, gargalos)      APS sequenciamento)
            │
   ┌────────┴─────────────────────────┐
   ▼                                   ▼
 PURCHASE (sugestão de compra)      PRODUCTION (sugestão de fabricação)
   │ aprovar/rejeitar (PCP/Compras)   │ firmar → Ordem de Produção (OF)
   ▼                                   ▼
 Pedido de Compra ──► NF-e Entrada   Ordem de Produção
   │ (FocusNFE)                        │ start → consumo → apontamento → complete
   ▼                                   ▼
 Estoque ENTRADA (almoxarifado)      Estoque de produto acabado (ENTRADA)
                                        │
                                        ▼
                                  Atende o Pedido de Venda → NF-e Saída + baixa de estoque
```

---

## Etapa 1 — Pedido de Venda entra

`POST /api/sales-order/create` (capa) + `POST /api/sales-order/items/create` (itens).

- O pedido referencia um **plano de produção** (`PlanCode`) — é o elo com o MRP.
- Status do pedido: rascunho → ... (ver `PATCH /api/sales-order/{code}/status`),
  bloqueio/desbloqueio por crédito (`/block`, `/unblock`).
- Item do pedido: item, quantidade, data de entrega.

**Como o pedido vira demanda do MRP:** o MRP, ao rodar um plano, lê **demandas
independentes** (`/api/independent-demand`) e **previsões de venda**
(`/api/sales-forecast`).

> ✅ **automático:** ao mudar o status do pedido para **"P" (Pedido / confirmado)**
> via `PATCH /api/sales-order/{code}/status`, o sistema cria automaticamente uma
> **demanda independente por item** do pedido (item, quantidade solicitada e data =
> data de entrega da linha ou, na falta, do cabeçalho). O código da demanda é
> derivado da linha (`código_pedido × 100000 + sequência`), tornando a
> reconfirmação **idempotente**. Implementado em
> `sales_order_uc/manage_sales_order_uc.go` (`ChangeStatusSalesOrderUseCase`).

Registro manual (ainda disponível para demandas avulsas / previsões):
`POST /api/independent-demand/create` para o item desejado (qtd e data de
necessidade).

---

## Etapa 2 — Plano de Produção

`POST /api/production-plan/create`. O plano define escopo (itens permitidos), origem da
demanda (independente: não/todas/a partir de data) e parâmetros. É o objeto que o MRP
recebe (`plan_code`).

---

## Etapa 3 — MRP roda

`POST /api/mrp-calculation/run` com `{ "plan_code": <P>, "generate_llc": true }`.

O motor (`mrp_calculation/service`):
1. Carrega demandas (independentes + previsões) do plano.
2. **Explode a BOM** por nível (LLC), aplicando **fórmula de perdas** por componente.
3. Calcula **necessidades líquidas**: `necessidade − estoque − suprimento firme`
   (ordens já firmes/aprovadas e em trânsito contam como suprimento; sugestões **não**).
4. Calcula datas retrocedendo o **lead time** — para itens fabricados, usa o **caminho
   crítico (CPM)** do roteiro; para comprados, o lead time do fornecedor/item.
5. Gera, por item:
   - **Ordens planejadas** `PRODUCTION` (itens fabricados) e `PURCHASE` (comprados),
     ambas como **sugestões** (`is_firm = false`, status `PLANNED`);
   - **Mensagens de exceção** (atraso, item sem configuração, gargalo).

Consultas:
- `GET /api/mrp-calculation/profile/{item_code}/{plan_code}` — perfil por período.
- `GET /api/mrp-calculation/exceptions/{plan_code}` — exceções (e alertas por e-mail/webhook).
- `GET /api/planned-order/list` — ordens planejadas geradas.

Modos suportados por item (`TipoMRP`): MRP, MIN_MAX, Kanban, Ponto de Pedido (ROP), MPS.

---

## Etapa 4 — Capacidade (CRP) e Sequenciamento (APS)

Sobre as ordens planejadas de produção:

- **CRP** — `POST /api/crp/calculate` (`{plan_code}`): soma horas requeridas por centro
  de trabalho/dia (operações do roteiro × quantidade), compara com a capacidade
  disponível (menos paradas de manutenção) e marca **sobrecarga** (`load_pct > 100`).
  Consulta: `GET /api/crp/{planCode}` e `/overloaded`.
- **APS** — `POST /api/aps/sequence`: sequenciamento de **capacidade finita** (EDD),
  alocando as ordens nas máquinas, pulando fins de semana e respeitando paradas.
  Gantt: `GET /api/aps/gantt/order/{orderID}` e `POST /api/aps/gantt/work-center`.

> ✅ **automático (pipeline):** `POST /api/planning/run-pipeline`
> (`{plan_code, generate_llc, start_from}`) encadeia **MRP → CRP → APS** num único
> disparo e devolve um **parecer de viabilidade consolidado** (itens/ordens do MRP,
> entradas e sobrecarga do CRP, operações sequenciadas do APS e o veredito
> `viable`). Implementado em `planning_uc.RunPlanningPipelineUseCase`. As chamadas
> individuais (`/api/mrp-calculation/run`, `/api/crp/calculate`, `/api/aps/sequence`)
> seguem disponíveis.

---

## Etapa 5 — Aprovação das sugestões (PCP / Compras)

As ordens planejadas são **sugestões** que precisam de decisão humana.

### 5a. Sugestões de COMPRA (matéria-prima)
Fluxo implementado (ver `manufatura-e-compras.md` §13 e supplier):
- Listar: `GET /api/purchase-order/suggestions`
- **Aprovar:** `POST /api/purchase-order/suggestions/{code}/approve`
  → gera **Pedido de Compra** (`origin = MRP`, `APPROVED`, firme) com o fornecedor
  escolhido (ou o **preferencial** do item) e os defaults do fornecedor (condição de
  pagamento, tabela de preço, tipo de NF, frete); torna a ordem planejada firme.
- **Rejeitar:** `POST /api/purchase-order/suggestions/{code}/reject`.

Caminhos alternativos de compra:
- **Solicitação → Geração de Pedidos:** `/api/purchase-requisitions` (+`/generate-orders`),
  agrupando por fornecedor.
- **Cotação:** `/api/purchase-quotations` (liberar → preços → selecionar → gerar pedidos).

### 5b. Sugestões de FABRICAÇÃO (itens produzidos)
- **Firmar** a ordem planejada: `GET /api/planned-order/{code}/firm`
  (marca `is_firm = true`, status `RELEASED`).

> ✅ **automático:** firmar uma ordem planejada do tipo **PRODUCTION** agora **gera a
> Ordem de Produção (OF)** automaticamente (status `OPEN`, vinculada à ordem
> planejada via `planned_order_id`, com item/quantidade/centro de custo/máquina/
> datas copiados do planejamento), espelhando o aprovar→pedido do lado de compras.
> A criação ocorre só na **primeira** firmação (guarda contra duplicidade lendo o
> `is_firm` anterior). Implementado em
> `planned_order_uc/firm_planned_order_uc.go`.

- **Criação manual da OF** (ainda disponível p/ casos avulsos):
  `POST /api/production-order/create` informando `planned_order_id`, `item_code`,
  `planned_qty`, `route_id`.

---

## Etapa 6 — Compra → Recebimento no almoxarifado

1. **Pedido de Compra** enviado ao fornecedor (`/api/purchase-order`).
2. **Recebimento via NF-e de entrada:** `POST /api/fiscal/entries/upload-nfe` (ou
   importação por chave via FocusNFE). O sistema casa o **CNPJ do emitente** ao
   fornecedor e, na importação, **lança movimentos `IN`** no estoque para cada
   item numérico da nota (ver `fiscal-financeiro.md` §5).
3. **Conferência/aprovação** da entrada: `POST /api/fiscal/entries/{code}/approve`
   (gera créditos fiscais).
4. Saldo do almoxarifado atualizado: `GET /api/stock/balances/item/{itemCode}`.

> ✅ **automático (saldo):** os movimentos de estoque agora **atualizam o saldo**
> (`stock_balances`) na **mesma transação** — quantidade, custo médio ponderado e
> último custo. O movimento de entrada padronizou o tipo para `IN`
> (consistente com os relatórios financeiros que filtram `movement_type='IN'`).
> Implementado em `repository/stock/stock_repository_pg.go` (`CreateMovement`).
>
> ✅ **automático (baixa do pedido de compra):** quando a importação recebe um
> `purchase_order_code`, as quantidades da NF-e **baixam os itens do pedido de
> compra** (`received_qty`), recalculando o status de cada linha e do pedido
> (`PARTIAL`/`RECEIVED`). Implementado em
> `PurchaseOrderRepository.RegisterReceipts` consumido por `ImportNFePurchaseUseCase`.
>
> ⚙️ **melhoria (pendente):** etapa explícita de **conferência de recebimento**
> (status `CONFERRED`) com tratamento de divergências/tolerância antes de somar ao
> estoque.

---

## Etapa 7 — Fabricação (Ordem de Produção)

Ciclo de vida da OF (`/api/production-order`):

| Passo | Endpoint | Efeito |
|---|---|---|
| Iniciar | `POST /{id}/start` | OPEN → IN_PROGRESS |
| Consumir matéria-prima | `POST /consumption` | baixa de insumos + movimento `OUT` |
| Apontar produção | `POST /appointment` | registra tempo + quantidade produzida |
| Concluir | `POST /{id}/complete` | IN_PROGRESS → COMPLETED + movimento `IN` do acabado |
| Fechar | `POST /{id}/close` | COMPLETED → CLOSED |

> ✅ **automático (estoque):**
> - O **consumo** (`POST /api/production-order/consumption`) gera automaticamente um
>   movimento **`OUT`** do insumo quando a linha traz `warehouse_id` — reduzindo o
>   saldo do almoxarifado. Referência: `PRODUCTION_ORDER` + id da OF.
> - A **conclusão** (`POST /api/production-order/{id}/complete`) gera o movimento
>   **`IN`** do produto acabado quando o corpo traz `warehouse_id` (depósito de
>   acabados); usa a quantidade produzida ou, na falta, a planejada.
>
> Implementado em `production_order_uc/add_consumption_uc.go` e
> `complete_production_order_uc.go`. Como os movimentos atualizam `stock_balances`
> (ver Etapa 6), o saldo de insumos e de acabados fica consistente sem lançamento
> manual.

> ⚙️ **melhoria (pendente):** *backflush* (baixa automática de componentes pela BOM ao
> apontar a operação) e baixa automática de **reservas** ao consumir.

Reservas de estoque (disponíveis): `POST /api/stock/reservations/create`,
`PATCH /{id}/release`, `PATCH /{id}/consume`.

---

## Etapa 8 — Atendimento do Pedido de Venda e saída fiscal

1. Com o produto acabado em estoque, o pedido de venda é atendido.
2. **NF-e de saída:** `POST /api/fiscal/exits/create` → `POST /{code}/authorize`
   (emissão via FocusNFE) — o **motor tributário** calcula ICMS/IPI/PIS/COFINS,
   diferimento, DIFAL/FCP (ver `fiscal-financeiro.md` §3–§4).
3. A autorização dispara, de forma encadeada, a baixa de estoque, a baixa de
   reservas e a baixa do pedido de venda + o financeiro.

> ✅ **automático:** ao **autorizar** a NF-e de saída, além de gerar a **Conta a
> Receber**, o sistema agora:
> - posta um movimento **`OUT`** por item (depósito resolvido a partir do item do
>   **pedido de venda** vinculado), reduzindo o saldo de acabados;
> - **consome as reservas** ativas do pedido de venda;
> - marca o pedido de venda como **Faturado** (`status = "F"`).
>
> Implementado em `fiscal_uc/authorize_fiscal_exit_uc.go`.

### Expedição / Carregamento (logística de saída)
> ✅ **disponível:** módulo de **expedição** (romaneio) em `/api/shipments`:
> criar romaneio (`POST /`), adicionar itens (`POST /{code}/items`), **conferir**
> a separação (`POST /items/confer`, `POST /{code}/confer`) e **despachar**
> (`POST /{code}/ship`, exige todos os itens conferidos). Migration
> `000146_shipments`.

> ⚙️ **melhoria (pendente):** logística mais rica (peso/volumes por caixa, etiquetas,
> integração com transportadora).

---

## Resumo dos status por documento

| Documento | Status |
|---|---|
| Pedido de Venda | `R` rascunho → `P` pedido/confirmado → (bloqueado) → `F` faturado / cancelado |
| Romaneio (Expedição) | `OPEN` → `SEPARATED` → `CONFERRED` → `SHIPPED` (`CANCELLED`) |
| Ordem Planejada | `PLANNED` (sugestão) → `RELEASED` (firme) → `CANCELLED` |
| Pedido de Compra | `DRAFT` → `REQUESTED` → `APPROVED` → `PARTIAL` → `RECEIVED` → `CANCELLED` |
| NF-e Entrada | `PENDING` → `CONFERRED` → `APPROVED` → `WRITTEN_OFF`/`CANCELLED` |
| Ordem de Produção | `OPEN` → `IN_PROGRESS` → `COMPLETED` → `CLOSED` (`CANCELLED`) |
| NF-e Saída | `DRAFT` → `AUTHORIZED` → `CANCELLED`/`REJECTED` |
| Solicitação/Cotação | `OPEN` → `PARTIAL`/`QUOTED` → `ATTENDED`/`CLOSED` → `CANCELLED` |

---

## Ordem prática de execução (cookbook)

1. Cadastros: item + BOM + roteiro + fornecedor + classificação fiscal + conversão UM.
2. `POST /api/production-plan/create`.
3. `POST /api/sales-order/create` (+ itens) e confirmar com
   `PATCH /api/sales-order/{code}/status` → `"P"` (✅ gera a demanda independente
   automaticamente; não é mais preciso chamar `/api/independent-demand/create`).
4. `POST /api/mrp-calculation/run`.
5. `POST /api/crp/calculate` e `POST /api/aps/sequence` (analisar capacidade).
6. Compras: aprovar sugestões (`/suggestions/{code}/approve`) → pedido de compra.
7. Produção: firmar ordem planejada (`GET /api/planned-order/{code}/firm`) → ✅ a OF é
   criada automaticamente (não é mais preciso `POST /api/production-order/create`).
8. Recebimento: `POST /api/fiscal/entries/upload-nfe` → `approve` (estoque entra com
   movimento `IN` + saldo atualizado).
9. Fabricar: `start` → `consumption` (✅ `OUT` dos insumos) → `appointment`
   (opcional `backflush_warehouse_id` → ✅ baixa componentes da BOM) →
   `complete` com `warehouse_id` (✅ `IN` do acabado) → `close`.
10. Faturar: `POST /api/fiscal/exits/create` → `authorize` (✅ NF-e + Conta a Receber +
    `OUT` do estoque + baixa de reservas + pedido de venda → Faturado).
11. Expedir: `POST /api/shipments` (+ itens, conferência) → `ship`.

> 💡 Atalho de planejamento: em vez dos passos 4–5, use
> `POST /api/planning/run-pipeline` para rodar MRP→CRP→APS de uma vez.
> Fiscal pós-venda: `POST /api/fiscal/manifestacao` (manifestação do destinatário)
> e `POST /api/fiscal/inutilizacao` (inutilização de numeração).

> As oportunidades de melhoria levantadas durante este mapeamento estão em
> [`../MELHORIAS.txt`](../MELHORIAS.txt).
