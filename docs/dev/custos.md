# Custos — Documentação técnica

Cobre Custo Padrão (rollup), custos de centro de trabalho, custo de compra, centro de
custo, overhead e base de alocação. Versão de negócio em
[`../apresentacao/custos.md`](../apresentacao/custos.md). Fundamentos do custo padrão
também em [`manufatura-e-compras.md`](manufatura-e-compras.md) §4.

> Convenções: `Authorization: Bearer <JWT>`, papel `ADMIN`/`USER`.

---

## 1. Custo Padrão (`/api/standard-cost`)

| Método | Rota | Ação |
|---|---|---|
| POST | `/rollup` | Recalcula o custo subindo pela estrutura (material + transformação + overhead) |
| GET | `/items/{itemCode}` | Consulta o custo padrão do item |

### Custos de centro de trabalho (`/work-center-costs`)
| POST `/` · GET `/` | Upsert e listagem do custo/hora por centro de trabalho |

### Custos de compra (`/purchase-costs`)
| POST `/` · GET `/{itemCode}` | Upsert e consulta do custo de compra por item |

Fórmula (resumo): `custo = Σ material(BOM) + Σ (tempo_operação × custo/hora_centro) + overhead`.
O rollup respeita o LLC para compor o custo dos intermediários antes do produto final.

---

## 2. Centro de Custo (`/api/cost-center`)

| Método | Rota | Ação |
|---|---|---|
| POST `/create` · GET `/list` · GET `/{costCenterCode}` | CRUD/consulta |

Enum de classificação: `CostCenterEnum`. Também há centros de custo no financeiro
(`/api/financial/centros-custo`) para o plano financeiro.

---

## 3. Overhead e Base de Alocação

### Base de alocação (`/api/allocations`)
Critério de rateio (horas de máquina, quantidade, valor de material…).

| POST `/create` · GET `/list` | CRUD |

### Alocação de overhead (`/api/overhead-allocation`)
Regra que distribui os custos indiretos usando a base escolhida.

| POST `/create` · GET `/list` | CRUD |

---

---

## 4. Custo Real da Ordem de Produção (apuração + variância)

Enquanto o **Custo Padrão** (§1) é o custo *planejado* do item, a OF apura o custo
**real** incorrido no chão de fábrica e o compara com o padrão.

| Método | Rota | Ação |
|---|---|---|
| POST | `/api/production-order/{id}/settle-cost` | Apura/recalcula o custo real da OF |
| GET | `/api/production-order/{id}/cost` | Consulta a apuração + variâncias |

**Como é apurado** (`production_order_costs`, migration `000152`):

- **Material real** = Σ (consumo × custo médio do item consumido). O custo médio vem
  do saldo (`stock_balances.avg_cost`), preferindo o depósito do consumo.
- **Conversão (mão-de-obra) real** = Σ (horas apontadas × custo/hora do centro de
  trabalho). A hora vem de `end_time − start_time` do apontamento e o custo/hora do
  `work_center_costs` do tipo da máquina apontada.
- **Overhead real** = aplicado proporcionalmente à mão-de-obra real pela razão
  padrão `overhead/mão-de-obra` (quando o padrão tem mão-de-obra).
- **Padrão (snapshot)** = custo unitário padrão do item × quantidade produzida.
- **Variância** = real − padrão, por componente (material, MO, overhead, total).
  Positivo = gastou mais que o padrão.

> ✅ **automático:** ao **fechar** a OF (`/{id}/close`), a apuração roda
> automaticamente (best-effort: uma falha de custeio não desfaz o fechamento).
> A apuração é **idempotente** — reexecutar `settle-cost` recalcula a linha única
> por OF. Implementado em `entity.BuildSettlement` (função pura, testada) +
> `SettleProductionCostUseCase`.

---

> Relatórios de custo (histórico, ficha técnica, produtos produzidos) ficam no módulo
> financeiro — ver [`fiscal-financeiro.md`](fiscal-financeiro.md).
