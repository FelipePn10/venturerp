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

> Relatórios de custo (histórico, ficha técnica, produtos produzidos) ficam no módulo
> financeiro — ver [`fiscal-financeiro.md`](fiscal-financeiro.md).
