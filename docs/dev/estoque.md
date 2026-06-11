# Estoque e Almoxarifado — Documentação técnica

Cobre movimentos, saldos, reservas, inventário e tipos de movimento. Versão de
negócio em [`../apresentacao/estoque.md`](../apresentacao/estoque.md). Detalhe de
movimentos/saldos/reservas também em [`visao-geral.md`](visao-geral.md) §5.2.

> Convenções: `Authorization: Bearer <JWT>`, `Content-Type: application/json`,
> papel `ADMIN`/`USER`.

---

## 1. Movimentos (`/api/stock/movements`)

| Método | Rota | Ação |
|---|---|---|
| POST | `/create` | Lança movimento |
| GET | `/list` | Lista |
| GET | `/item/{itemCode}` | Por item |
| GET | `/warehouse/{warehouseId}` | Por armazém |

> ✅ Movimentos atualizam `stock_balances` **na mesma transação** (quantidade, custo
> médio ponderado e último custo). Tipo de entrada padronizado em `IN`. Ver
> `repository/stock/stock_repository_pg.go` (`CreateMovement`).

## 2. Saldos (`/api/stock/balances`)

| Método | Rota | Ação |
|---|---|---|
| GET | `/get` | Saldo (item+armazém) |
| GET | `/list` | Lista saldos |
| GET | `/warehouse/{warehouseId}` | Por armazém |
| GET | `/item/{itemCode}` | Por item |

## 3. Reservas (`/api/stock/reservations`)

| Método | Rota | Ação |
|---|---|---|
| POST | `/create` | Cria reserva |
| PATCH | `/{id}/release` | Libera |
| PATCH | `/{id}/consume` | Consome |

> A autorização da NF-e de saída consome as reservas do pedido automaticamente.

## 4. Inventário (`/api/stock/inventories`)

| Método | Rota | Ação |
|---|---|---|
| POST | `/create` | Cria inventário |
| GET | `/list` | Lista |
| GET | `/{id}` | Consulta |
| POST | `/{id}/close` | Fecha |
| POST | `/count` | Registra contagem de um item |
| POST | `/adjust` | Ajusta diferença (gera movimento de acerto) |
| GET | `/{id}/items` | Lista itens do inventário |

Fluxo: criar → contar → ajustar → fechar. O ajuste gera o movimento de acerto de saldo.

## 5. Tipos de movimento (`/api/estoque/tipos-movimento`)

Classificação (com sigla) de cada lançamento de estoque.

| Método | Rota | Ação |
|---|---|---|
| POST `/` · PUT `/` · GET `/` · GET `/{id}` · GET `/sigla/{sigla}` | CRUD + busca por sigla |

---

## 6. Disponível para promessa (ATP)

| Método | Rota | Ação |
|---|---|---|
| GET | `/api/stock/balances/atp/{itemCode}` | Disponível = saldo − reservas (todos os depósitos; opcional `?mask=`) |

As **reservas** mantêm o `reserved_qty` do saldo consistente (criar/liberar/consumir,
na mesma transação), então o ATP reflete o que realmente pode ser prometido.
Confirmar um pedido de venda (`"P"`) reserva o disponível por linha automaticamente
(ver [`00-fluxo-geral.md`](00-fluxo-geral.md)).

## 7. Rastreabilidade de lote / corrida (genealogia)

Migration `000153`. Saldo segregado por lote (`stock_lot_balances`) + registro de
corrida/certificado (`stock_lots`). Todo movimento com `lot` atualiza o saldo do lote.

| Método | Rota | Ação |
|---|---|---|
| POST | `/api/stock/lots/register` | Registra lote: `heat_number` (corrida), `certificate`, fornecedor, recebimento |
| GET | `/api/stock/lots/item/{itemCode}` | Saldos por lote do item |
| GET | `/api/stock/lots/genealogy/{itemCode}/{lot}` | Genealogia bidirecional do lote |

A **genealogia** devolve: registro (corrida/certificado), saldos do lote,
`consumed_in` (OFs que consumiram o lote → item produzido) e `produced_by` (OFs que
produziram o lote + `input_lots` que o compõem). O lote do acabado é gravado ao
concluir a OF com `lot`.

## 8. Consumo médio mensal (ROP)

Migration `000154`. Calculado das saídas (`OUT`/`TRANSFER_OUT`) numa janela móvel
(padrão 6 meses); alimenta o ponto de reposição.

| Método | Rota | Ação |
|---|---|---|
| POST | `/api/stock/consumption-average/recalc` | Recalcula (item específico via `item_code`, ou todos) |
| GET | `/api/stock/consumption-average/{itemCode}` | Consulta consumo médio mensal do item |

---

## Cadastros relacionados
Armazém (`/api/warehouse`) e Localização (`/api/location` — países/UFs não; ver
[`cadastros-apoio.md`](cadastros-apoio.md) para armazém/localização física). Tipos de
armazém/localização: ver enums `WarehouseType`/`LocationType`.
