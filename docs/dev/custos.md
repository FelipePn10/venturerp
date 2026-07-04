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

Campos: `cost_per_hour` (taxa combinada — usada como fallback de máquina) e o **split
enterprise+** `machine_cost_per_hour` / `labor_cost_per_hour` (migration `000174`).
Quando o split é omitido, a máquina usa `cost_per_hour` e a mão-de-obra fica em 0.

### Custos de compra (`/purchase-costs`)
| POST `/` · GET `/{itemCode}` | Upsert e consulta do custo de compra por item |

Fórmula (resumo): `custo = Σ material(BOM) + custo_de_conversão(roteiro) + overhead`.

**Custo de conversão (Fase 2 — por centro de trabalho real, quantidade-consciente).**
Cada operação do roteiro é debitada na taxa do **seu próprio** centro de trabalho — não
mais a média ingênua de todos os centros (bug corrigido). Usa o modelo de tempo rico:
```
conversão_unitária = ( Σ_operações [ MachineHours(lot) × machine_rate(CT)
                                    + LaborHours(lot)   × labor_rate(CT) ] ) ÷ lot
```
O `POST /rollup` aceita **`lot_size`** (lote de referência, padrão 1): com `lot_size > 1`
o **setup é amortizado** sobre o lote (ex.: setup 1 h em lote de 10 → 0,1 h/peça),
como em uma rotina estruturada de custo padrão industrial.

> **Co-produtos e quantidade fixa (BOM).** Componentes `is_coproduct` **creditam** o
> custo do pai pelo seu valor (`material -= valor_coproduto × qtde`) — recuperação de
> subproduto/sucata; o material nunca fica negativo. Componentes `is_fixed_qty` são
> **amortizados** pelo `lot_size` (`material += valor × qtde ÷ lote`). Grupos de
> substitutos (`substitute_group > 0`) custeiam somente o componente primário
> (`substitute_priority` menor), mantendo o rollup alinhado à explosão do MRP.
> Operações `FANTASMA` não custam. Quando o roteiro não
está disponível, cai no cálculo antigo (`horas_totais × média`). O rollup respeita o LLC
para compor o custo dos intermediários antes do produto final e resolve a BOM pela
`mask` informada no rollup (componentes genéricos + componentes específicos da variante).

> Corpo do rollup: `{ "item_code": 50001, "mask": "", "lot_size": 100, "calculated_by": "<uuid>" }`.

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

> Código duplicado no `POST /create` retorna **409 Conflict** (não mais 500).

### Alocação de overhead (`/api/overhead-allocation`)
Regra que distribui os custos indiretos usando a base escolhida.

| POST `/create` · GET `/list` | CRUD |

> **Campo obrigatório:** `cost_center_code` (código do centro de custo, não
> `cost_center_id`). Cada alvo em `targets[]` também usa `cost_center_code`.
> Ausência → **422** (`cost_center_code is required`). As colunas legadas
> `cost_center_id`/`overhead_id` viraram nullable (migration `000172`) — antes o
> `POST /create` estourava com *null value in column "cost_center_id"*.
>
> Corpo mínimo:
> ```json
> {
>   "cost_center_code": 10,
>   "period_start": "2026-01-01",
>   "period_end": "2026-01-31",
>   "allocation_type": "PERCENTAGE",
>   "targets": [{ "cost_center_code": 20, "percentage": 100 }]
> }
> ```

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
