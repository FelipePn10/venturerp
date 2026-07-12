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

> ⚠️ **Campo de consumo:** `POST /consumption` usa **`consumed_qty`** (não
> `quantity`). Enviar `quantity` é ignorado (grava 0 e não baixa estoque). Campos:
> `production_order_id`, `item_code`, `consumed_qty`, `warehouse_id?`, `lot?`,
> `consumption_date?`, `notes?`.
>
> **Datas reais:** `start_date`/`end_date`/`consumption_date`/`appointment_date`
> aceitam `YYYY-MM-DD` ou ISO-8601; quando omitidas assumem **agora** (não mais
> `0001-01-01`). `POST /create` exige `item_code` e `planned_qty > 0` (422).

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

Corpo de `/operations/advance`: `{ "operation_id": 123, "status": "IN_PROGRESS", "actual_hours": 0 }`.
`status` ∈ `PENDING` · `IN_PROGRESS` · `DONE` · `SKIPPED` (422 para outros valores).
`IN_PROGRESS` carimba `started_at`; `DONE`/`SKIPPED` carimbam `completed_at`.

> ✅ **Backflush:** no apontamento com `backflush_warehouse_id`, os componentes da BOM
> são baixados automaticamente, proporcional à qtd produzida. Respeita quantidade fixa
> por OF, ignora co-produtos e usa o componente primário de grupos substitutos. Ver
> [`manufatura-e-compras.md`](manufatura-e-compras.md) §18.

> 🔧 **Ficha de Produção da Ferramenta:** define qual **série** de cada ferramenta roda
> cada operação da OF, com substituição rastreada e débito de vida útil por série no
> apontamento. Ver [`ficha-producao-ferramenta.md`](ficha-producao-ferramenta.md).

---

## Origem da OF
A OF normalmente nasce ao **firmar** uma ordem planejada `PRODUCTION`
(`GET /api/planned-order/{code}/firm`), que cria a OF automaticamente na primeira
firmação. Ver `planned_order_uc/firm_planned_order_uc.go` e
[`mrp-calculo.md`](mrp-calculo.md).

---

## 3. Custo real da OF

| Método | Rota | Ação |
|---|---|---|
| POST | `/{id}/settle-cost` | Apura/recalcula o custo real (material + conversão) |
| GET | `/{id}/cost` | Consulta a apuração + variâncias vs. padrão |

> ✅ **automático:** fechar a OF (`/{id}/close`) apura o custo real. Detalhes da
> fórmula em [`custos.md`](custos.md) §4.

## 4. Sucata / retalho valorizado

| Método | Rota | Ação |
|---|---|---|
| POST | `/{id}/scrap-return` | Retorna sucata/retalho como subproduto valorizado (`IN`) |

Corpo: `scrap_item_code`, `warehouse_id`, `quantity`, `unit_value`, `lot?`, `notes?`.
O movimento `IN` valoriza a sucata no estoque (custo médio do item de sucata), para
revenda ou reaproveitamento de retalho de chapa/barra.

## 5. Lote produzido (rastreabilidade)

Ao concluir a OF (`/{id}/complete`) informando `lot`, o lote do acabado é gravado no
movimento `IN`, habilitando a **genealogia** em
`GET /api/stock/lots/genealogy/{itemCode}/{lot}` (ver [`estoque.md`](estoque.md)).

### Entrega parcial ou final da OF

`POST /api/production-order/{id}/complete` também funciona como entrega de
produção. O corpo aceita `quantity`, `final`, `warehouse_id`, `lot` e
`idempotency_key`. Sem `warehouse_id`, utiliza o almoxarifado gravado na OF —
inclusive o destino de assistência técnica propagado pelo MRP.

- `EP`: entrada normal quando o tratamento de excedentes não está habilitado.
- `EPP`: entrega até a quantidade planejada quando `production_excess_treatment`
  está habilitado nos parâmetros de planejamento.
- `EPE`: entrega que ultrapassa a quantidade planejada.
- `REP`: saída automática dos componentes da estrutura cujo item possui
  `warehouse_automatic_low=true`, considerando quantidade fixa e perda.

A chave de idempotência é única por empresa e evita entrega duplicada. Uma
entrega parcial mantém a OF em `IN_PROGRESS`; `final=true` muda para
`COMPLETED`. Finalização com quantidade zero é bloqueada quando a OF possui uma
OCS vinculada ainda não recebida/cancelada. Esses vínculos são persistidos em
`production_order_service_links`.

### Consulta operacional consolidada

`GET /api/production-order/{id}/operational` retorna em uma única resposta:

- dados e prioridade da OF;
- entregas parciais/finais e classes EP/EPP/EPE;
- apontamentos e refugos;
- consumos com lotes;
- movimentos de estoque vinculados, incluindo REP;
- totais planejado, produzido, entregue, refugado e pendente.

A consulta exige acesso à empresa autenticada. Desenhos/revisões e roteiro
continuam disponíveis nos endpoints especializados de `drawings` e
`production-order/{id}/operations`, evitando duplicação desses cadastros.

## 6. Materiais, substituições, lotes e WMS

O cadastro manual da OF gera, na mesma transação, as demandas do primeiro nível
da estrutura. Co-produtos e substitutos secundários não viram demandas; perdas,
quantidade fixa e baixa automática são preservadas. Se um componente for o
próprio item fabricado, a OF recebe a observação `ORDEM DE RETRABALHO`.

| Método | Rota | Uso |
|---|---|---|
| GET | `/{id}/materials?kind=DEMAND|RETURN` | Demandas/devoluções da OF |
| POST | `/materials` | Inclui demanda ou devolução |
| POST | `/materials/replace` | Substituição parcial/total rastreável |
| DELETE | `/materials/{materialID}` | Exclui somente sem atendimento/movimento/WMS |
| POST | `/materials/lots` | Seleciona lotes; lista vazia aplica FIFO automático |
| POST | `/materials/lots/batch` | Distribui lotes entre várias OFs por ordem crescente |
| POST | `/scrap-destinations` | Destina refugo da OF ou de uma demanda |
| GET | `/delivery-candidates` | OFs liberadas/em produção filtradas para entrega |
| PUT | `/{id}` | Mantém quantidade, datas, máquina, prioridade e observação |
| PUT | `/wms-settings` | Configura almoxarifado WMS e intermediário de saída |

As quantidades novas usam decimal e o estoque persiste seis casas. Em uma
substituição, o componente original permanece com a quantidade restante e os
substitutos guardam `substituted_item_code`, inclusive para rastreio fiscal.
Solicitação WMS não cancelada bloqueia alteração/exclusão. Um almoxarifado WMS
exige intermediário de saída e os lotes são consumidos desse intermediário.

OF movimentada, apontada, consumida ou com separação WMS não pode ser alterada.
A quantidade não pode ficar abaixo do produzido nem ser fracionária quando o
item não aceitar frações. OF Kanban ou comercial não pode ser mantida/cancelada
por esse fluxo. Os controles equivalentes aos parâmetros 10 e 14 ficam
registrados na OF para autorizar mudanças de quantidade e datas.

O parâmetro 66 bloqueia demanda de retrabalho divergente da quantidade da OF. O
parâmetro 45 bloqueia liberação com apontamento por ordem + baixa no
cadastro/liberação, roteiro misto com terceiros ou remessa de terceiros que não
utilize itens da demanda.

## 7. Atomicidade e serviços de terceiros

Entrega, linhas EPP/EPE, REP, co-produtos, movimentos e saldos são gravados em
uma única transação. Uma entrega que cruza o planejado é dividida (por exemplo,
`1 EPP + 2 EPE`) sem perder a chave idempotente. O encadeamento
OF → requisição de serviço → OCS é persistido e usado pelo bloqueio de
encerramento com OCS pendente.
