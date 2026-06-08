# Produção — Documentação técnica

Cobre a Ordem de Produção (OF) e suas operações. **Roteiro**, **Qualidade** e
**Manutenção** têm seções dedicadas em
[`manufatura-e-compras.md`](manufatura-e-compras.md) (§1 Roteiro, §5 Qualidade,
§6 Manutenção) e [`maquinas-e-roteiro.md`](maquinas-e-roteiro.md). Versão de negócio
em [`../apresentacao/producao.md`](../apresentacao/producao.md). Detalhe da OF também
em [`visao-geral.md`](visao-geral.md) §5.1.

> Convenções: `Authorization: Bearer <JWT>`, papel `ADMIN`/`USER`.

---

## 1. Ordem de Produção (`/api/production-order`)

| Método | Rota | Ação |
|---|---|---|
| POST | `/create` | Cria OF (item, qtd, roteiro; `planned_order_id` quando vinda do MRP) |
| GET | `/list` | Lista |
| GET | `/{id}` | Consulta |
| POST | `/{id}/start` | OPEN → IN_PROGRESS |
| POST | `/appointment` | Apontamento (tempo + qtd produzida/refugada) |
| POST | `/consumption` | Consumo de insumo (gera `OUT` quando há `warehouse_id`) |
| POST | `/{id}/complete` | IN_PROGRESS → COMPLETED (gera `IN` do acabado com `warehouse_id`) |
| POST | `/{id}/close` | COMPLETED → CLOSED |
| POST | `/{id}/cancel` | Cancela |
| GET | `/{id}/appointments` | Histórico de apontamentos |
| GET | `/{id}/consumptions` | Histórico de consumos |

**Status:** `OPEN` → `IN_PROGRESS` → `COMPLETED` → `CLOSED` (`CANCELLED`).

> ✅ **Automações de estoque:** consumo → `OUT` do insumo; conclusão → `IN` do acabado;
> ambos atualizam saldo/custo. Ver `production_order_uc/add_consumption_uc.go` e
> `complete_production_order_uc.go`.

---

## 2. Operações da OF

| Método | Rota | Ação |
|---|---|---|
| POST | `/operations/explode` | Explode o roteiro do item nas operações da OF |
| GET | `/{id}/operations` | Lista operações da OF e andamento |
| POST | `/operations/advance` | Avança a operação (conclui etapa, libera a próxima) |

> ✅ **Backflush:** no apontamento com `backflush_warehouse_id`, os componentes da BOM
> são baixados automaticamente, proporcional à qtd produzida. Ver
> [`manufatura-e-compras.md`](manufatura-e-compras.md) §18.

---

## Origem da OF
A OF normalmente nasce ao **firmar** uma ordem planejada `PRODUCTION`
(`GET /api/planned-order/{code}/firm`), que cria a OF automaticamente na primeira
firmação. Ver `planned_order_uc/firm_planned_order_uc.go` e
[`mrp-calculo.md`](mrp-calculo.md).
