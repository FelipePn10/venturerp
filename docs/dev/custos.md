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
| GET `/cadastro` | lista ativos e inativos (tela VCTB0102) |
| GET `/{costCenterCode}/uso` | `{"em_uso", "usos": [{"onde", "quantidade"}]}` |
| PUT `/{costCenterCode}` | altera (ADMIN e USER) |
| DELETE `/{costCenterCode}` | exclui — só ADMIN, só sem uso |

**Manutenção** (`cost_center_uc/cost_center_manutencao.go`): o código nunca muda —
máquinas, estruturas, ordens, requisições, pedidos de compra, vendas recorrentes e
rateios guardam o centro por ele (metade sem FK). Descrição, data final e situação
mudam sempre; tipo, centro-pai, "rateio de absorção" e data inicial só sem uso
(409 listando onde é usado). O pai é conferido (existe na empresa, ativo, sem
ciclo) também no `POST /create`. Desativar com filho ativo → 409. `end_date`
ausente no `PUT` **limpa** a data final — mande a atual para mantê-la.

Enum de classificação: `CostCenterEnum`.

**Cadastro único (migração 447, 2026-10-27).** Não há mais dois cadastros: o
financeiro (`centros_custo`, VFIN0130) foi migrado para `cost_centers`, empresa a
empresa, com o código alfanumérico preservado e a hierarquia montada pelo
prefixo (`CC-FAB-CORTE` abaixo de `CC-FAB`); as referências financeiras (títulos,
rateios, itens e parcelas de nota, frete, regras de rateio, ECD) apontam para
`cost_centers.id`. O `code` numérico continua a chave do lado industrial; o
código da empresa fica em `cost_center_gestao` (gatilho cria a linha de todo
centro novo; `POST /create` com `code` 0 usa o próximo livre).

| Método | Rota | Ação |
|---|---|---|
| GET `/gestao` | centros com a gestão (código da empresa, responsável, aprovador, conta padrão) |
| PUT `/{costCenterCode}/gestao` | grava a gestão (código único na empresa; aprovar requisições exige aprovador; usuários e conta da empresa) |
| GET `/usuarios` | usuários da empresa (responsável e aprovador) |
| GET/PUT `/orcamento?ano=` | orçado mensal × realizado; o PUT grava os meses (`valor` em texto decimal) |
| GET `/arvore?ano=&mes=` | árvore com orçado e realizado próprios e consolidados, `estourado` (mes 0 = ano) |
| GET `/aprovacoes?status=&minhas=&requisicao=` | aprovações de requisição pelo centro |
| POST `/aprovacoes/{id}/aprovar` · `/reprovar` | decide (só o aprovador do centro; quem pediu não aprova; reprovar exige motivo; duas decisões simultâneas: uma passa) |

**Realizado** = contas a pagar não cancelados pela data de emissão: o rateio do
título por centro, ou o centro do título sem rateio (a nota de entrada aprovada
gera o título com o rateio dos itens, então não há dupla contagem).

**Requisição de compra:** criar ou acrescentar item de centro com
`aprova_requisicoes` abre (ou reabre, se o item novo chegou depois da
aprovação, ou depois da reprovação) a aprovação do aprovador do centro; a
geração de pedidos pula a requisição pendente ou reprovada, com o motivo. A
requisição de serviço gerada ao firmar a ordem planejada (MRP, operação
externa) leva o centro de custo da ordem e abre a mesma aprovação, dentro da
transação da firmação (`cost_center.AprovacoesNaTransacao`).

**Alertas:** `FINANCEIRO_CENTRO_ORCAMENTO_ESTOURADO` (realizado do mês acima do
orçado; resolve quando volta para dentro ou o mês acaba) e
`COMPRAS_REQUISICAO_AGUARDANDO_CENTRO` (resolve quando o aprovador decide).

Domínio: `internal/domain/cost_center/gestao`.

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
