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

## Cadastros relacionados
Armazém (`/api/warehouse`) e Localização (`/api/location` — países/UFs não; ver
[`cadastros-apoio.md`](cadastros-apoio.md) para armazém/localização física). Tipos de
armazém/localização: ver enums `WarehouseType`/`LocationType`.
