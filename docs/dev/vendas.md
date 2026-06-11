# Vendas e Expedição — Documentação técnica

Cobre Pedido de Venda, Divisão de Vendas, Promessa de Entrega, Reprogramação de
Entrega e Expedição (romaneio). A versão de negócio está em
[`../apresentacao/vendas.md`](../apresentacao/vendas.md). Detalhe aprofundado do
Pedido de Venda também em [`visao-geral.md`](visao-geral.md) §4.

> Convenções: `Authorization: Bearer <JWT>`, `Content-Type: application/json`.
> Salvo indicação, todas as rotas exigem papel `ADMIN` ou `USER`.

---

## 1. Pedido de Venda (`/api/sales-order`)

| Método | Rota | Ação |
|---|---|---|
| POST | `/create` | Cria a capa do pedido |
| GET | `/list` | Lista pedidos |
| GET | `/{code}` | Consulta por código |
| PUT | `/{code}` | Atualiza a capa |
| DELETE | `/{code}/cancel` | Cancela o pedido |
| PATCH | `/{code}/block` | Bloqueia (crédito/manual) |
| PATCH | `/{code}/unblock` | Desbloqueia |
| PATCH | `/{code}/status` | Muda o status |
| GET | `/customer/{customerCode}` | Lista por cliente |
| GET | `/status/{status}` | Lista por status |

### Itens (`/api/sales-order/items`)

| Método | Rota | Ação |
|---|---|---|
| POST | `/create` | Adiciona item (item, qtd, data de entrega) |
| GET | `/{code}` | Lista itens do pedido |
| PUT | `/{itemCode}` | Atualiza item |
| DELETE | `/{itemCode}/cancel` | Cancela item |

**Status do pedido:** `R` (rascunho) → `P` (pedido/confirmado) → `F` (faturado);
`CANCELLED`; estado **bloqueado** ortogonal (crédito/manual).

> ✅ **Automação:** mudar o status para `P` cria, por item, uma **demanda
> independente** (item, qtd, data) de forma **idempotente** — código derivado da linha
> (`código_pedido × 100000 + sequência`). Ver `sales_order_uc/manage_sales_order_uc.go`
> e [`00-fluxo-geral.md`](00-fluxo-geral.md).
>
> ✅ **Automação (crédito):** confirmar (`P`) roda a **checagem de limite de
> crédito** (exposição = contas a receber em aberto + outros pedidos em aberto).
> Excedeu o limite (ou cliente bloqueado) → pedido **bloqueado** automaticamente,
> sem gerar demanda nem reserva. Ver `sales_order_uc/credit_check.go`.
>
> ✅ **Automação (ATP/reserva):** aprovado no crédito, cada linha **reserva o
> estoque disponível** no depósito da linha (limitado ao disponível). ATP em
> `GET /api/stock/balances/atp/{itemCode}`. Ver `sales_order_uc/order_reserve.go`.
>
> ✅ **Automação (faturamento):** a autorização da NF-e de saída posta `OUT` por item,
> consome reservas do pedido e marca o pedido como `F`. Ver
> `fiscal_uc/authorize_fiscal_exit_uc.go` e [`fiscal-financeiro.md`](fiscal-financeiro.md).

---

## 2. Divisão de Vendas (`/api/sales-division`)

Organização comercial (equipe/região/unidade) associável ao pedido para análise de
resultado e regras comerciais.

| Método | Rota | Ação |
|---|---|---|
| POST | `/create` · GET `/list` · GET `/{code}` · PUT `/{code}` · DELETE `/{code}` | CRUD completo |

---

## 3. Promessa de Entrega

Cálculo de data prometida com base em disponibilidade (estoque + capacidade).

### Parâmetros (`/api/delivery-promise-params`)
| Método | Rota | Ação |
|---|---|---|
| GET | `/` | Lê os parâmetros |
| PUT | `/update` | Atualiza os parâmetros |

### Calendário de promessa por item (`/api/item-calendar-promise`)
Disponibilidade (ATP) por item/variante, dia a dia.

| Método | Rota | Ação |
|---|---|---|
| POST | `/create` | Upsert de um dia |
| GET | `/{item_code}/{mask}/{year}/{month}` | Lista o mês |
| GET | `/{item_code}/{mask}/{year}/{month}/workdays` | Dias úteis |
| GET | `/{item_code}/{mask}/{year}/{month}/{day}` | Consulta um dia |
| DELETE | `/{item_code}/{mask}/{year}/{month}/{day}` | Remove um dia |

---

## 4. Reprogramação de Entrega (`/api/delivery-reschedule`)

Histórico de remarcações de data vinculado ao pedido (data original × nova × motivo).

| Método | Rota | Ação |
|---|---|---|
| POST | `/create` | Registra a reprogramação |
| GET | `/list/{sales_order_code}` | Lista as reprogramações do pedido |

---

## 5. Expedição / Romaneio (`/api/shipments`) — migration 000146

| Método | Rota | Ação |
|---|---|---|
| POST | `/` | Cria romaneio |
| GET | `/` | Lista |
| GET | `/{code}` | Consulta |
| POST | `/{code}/items` | Adiciona item |
| POST | `/items/confer` | Confere um item |
| POST | `/{code}/confer` | Confere o romaneio |
| POST | `/{code}/ship` | Despacha (exige tudo conferido) |
| POST | `/{code}/cancel` | Cancela |

**Status:** `OPEN` → `SEPARATED` → `CONFERRED` → `SHIPPED` (`CANCELLED`).
