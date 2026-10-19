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
> `repository/stock/stock_repository_pg.go` (`CreateMovementTx`).

Regras do lançamento (todas em `CreateMovementTx`, o único ponto que grava movimento):

- **Custo médio único por item e almoxarifado.** A quantidade é por endereço
  (uma linha por endereço), mas o custeio é feito sobre o total do almoxarifado e o
  custo resultante vale para todas as linhas. Saída sem preço é valorizada por ele.
- **Saldo negativo bloqueado** (migração 405): saída que deixa o almoxarifado — ou o
  endereço — abaixo de zero responde 422, salvo `warehouse.allow_negative_stock`
  (padrão verdadeiro na linha de produção, por causa da baixa automática). Ajuste de
  inventário e de contagem cíclica passam (`PermiteSaldoNegativo`).
- **Período fechado** (`manufacturing_stock_closed_periods`) recusa qualquer lançamento
  com data dentro dele — antes só o refugo da produção respeitava.
- Preço sem total: o total é quantidade × preço (chamadores internos gravavam zero).
- **Precisão** (migração 407): quantidade e custo andam em `decimal` do repositório
  ao domínio (`shopspring/decimal`), nunca em `float64` — dez entradas de 0,1 somam
  1 e a saída do saldo inteiro zera quantidade e valor. Colunas: quantidade
  `NUMERIC(18,6)`; custo unitário (`unit_price`, `avg_cost`, `last_cost`,
  `unit_cost` do inventário) `NUMERIC(24,10)` — R$ 400 por barra de 6000 mm dá
  R$ 0,0666666667/mm, que com 4 casas virava 0,0667; totais em dinheiro seguem com
  4 casas. Custo médio = valor ÷ quantidade com 10 casas (`entity.CasasCusto`),
  sempre sobre o valor já em 4 casas (`entity.CasasValor`): o médio gravado é
  exatamente total_cost ÷ quantity. A saída é valorizada em centavos, então o
  médio pode variar na 7ª casa — é o custo médio móvel, como nos ERPs de mercado.
  O pedido de compra guarda a unidade interna com as mesmas escalas (migração
  408): 100 KG × 0,15530965 entram como 15,530965 M, não 15,5310. A inspeção de
  recebimento (ordem, análise, disposição, registro) também (migração 409), e a
  análise confere o saldo da inspeção antes de gravar.
  As respostas JSON continuam número (conversão só na fronteira do DTO); os
  read models de kardex, posição e análise devolvem texto decimal.
- **Tipo de movimento configurado** (`movement_type_code`, sigla da VEST0200): no
  lançamento manual decide a direção, exige centro de custo (`requires_cost_center`),
  recusa quando `allows_manual` é falso e, com `updates_avg_cost` falso, a entrada
  entra pelo custo médio atual. `cost_center_code` fica no movimento.

| Método | Rota | Ação |
|---|---|---|
| POST | `/{id}/reverse` | **Estorno** `{"reason"}`: movimento contrário pelo mesmo valor, ligado por `reverses_movement_id`/`reversed_by_movement_id`. Só manual ou transferência (as duas pernas); documento (nota, OF, inventário) é desfeito pelo documento; estorno duplo → 409 |

### Transferência entre almoxarifados — `POST /api/stock/transfers`

`{item_code, from_warehouse_id, to_warehouse_id, quantity, lot?, from_address?, to_address?}`.
Uma operação: `TRANSFER_OUT` na origem ao custo médio e `TRANSFER_IN` no destino pelo
mesmo custo; as pernas se referenciam (`reference_type TRANSFERENCIA`, `reference_code`
= id da saída). Trânsito: transfira para um almoxarifado de localização `TRANSITO`.
Transferência originada de documento (liberação da inspeção de recebimento) usa a
mesma operação com `ReferenciaTipo`/`ReferenciaCodigo` do documento: as pernas
guardam a referência dele e se desfazem pelo documento, não pelo estorno avulso.
Toda entrada de transferência vale o que saiu — inclusive a automática para o
almoxarifado de linha da OF.

### Conversão de unidade (compras e vendas)

- **Nota de entrada:** a linha em unidade diferente da de estoque recebe o fator do
  cadastro do item (`item_unit_conversions`, direta ou inversa, sem máscara) ou, sem
  ele, da conversão global (`global_unit_conversions`) — depois do vínculo com o
  fornecedor e do pedido. Sem conversão nenhuma é **pendência impeditiva**. A
  aprovação reaplica as conversões nas notas gravadas antes. (Caso Tecnofer, NF
  78402: 115,8975 KG de tubo entravam como 115,8975 M.)
- **Pedido de compra:** `purchase_order_uc.UnidadeInterna` grava a quantidade e o
  preço na unidade de estoque do cadastro em todas as origens (linha, criação,
  sugestão do MRP, requisição, cotação, recebimento de linha antiga); unidade de
  compra sem conversão → 422.
- **NF-e de saída:** a baixa converte da unidade da nota (`uCom`) para a de estoque;
  a prévia barra a transmissão sem conversão ou sem saldo.

### Extrato e posição

| Método | Rota | Ação |
|---|---|---|
| GET | `/api/stock/kardex/{itemCode}?warehouse_id&from&to` | Saldo anterior, movimentos com saldo acumulado (qtd e valor), saldo final. Dias contados no fuso da empresa (`enterprise_notification_settings.timezone`, padrão America/Sao_Paulo): antes o fim do dia era meia-noite UTC e o lançamento depois das 21:00 sumia de "até hoje" |
| GET | `/api/stock/position?date&warehouse_id` | Saldo por item/almoxarifado ao fim do dia, valorizado, com a 2ª unidade (unidade de compra) quando há conversão |

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

## 4. Inventário (`/api/inventory`; `/api/stock/inventories` por compatibilidade)

| Método | Rota | Ação |
|---|---|---|
| POST | `/` | Abre: `{code, description, warehouse_id, start_date, item_codes?}` — grava a **foto** do saldo (qtd e custo médio) |
| GET | `/` · `/{id}` · `/{id}/items` | Consulta |
| POST | `/count` | Conta (upsert): diferença = contado − foto; recontagem guarda a anterior (`previous_counted_qty`, `count_number`) |
| POST | `/{id}/close` | Fecha `{"zerar_nao_contados"}` — lança `ADJUSTMENT` de cada diferença ao custo médio (ref. `INVENTORY`), numa transação; responde o resumo |

Até a migração 405 o inventário não mexia no saldo: abrir não gerava itens, contar
atualizava linha inexistente e fechar só trocava a situação. Um almoxarifado não tem
dois inventários abertos. A **contagem cíclica** (VEST0500) também ajusta: dentro da
tolerância na conclusão; divergente, só na aprovação (ref. `CONTAGEM_CICLICA`).

## 5. Tipos de movimento (`/api/estoque/tipos-movimento`)

Classificação (com sigla) de cada lançamento de estoque. JSON em snake_case
(`entry_exit`, `requires_cost_center`, `allows_manual`, `updates_avg_cost`,
`considers_consumption`…) — antes sem tags, as flags com sublinhado não chegavam.

| Método | Rota | Ação |
|---|---|---|
| POST `/` · PUT `/` · GET `/` · GET `/{id}` · GET `/sigla/{sigla}` | CRUD + busca por sigla |

---

## 6. Disponível para promessa (ATP)

| Método | Rota | Ação |
|---|---|---|
| GET | `/api/stock/balances/atp/{itemCode}` | Disponível = saldo − reservas dos almoxarifados **prometíveis** (`disposition` e localização fora de `INSPECAO`/`REJEICAO`); o resto vai em `total_not_available`, e cada almoxarifado traz `stock_status` (LIVRE, INSPECAO, BLOQUEADO) |

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

Migration `000154`. Calculado das saídas de **consumo** (`OUT`/`SAIDA`/`REP`, sem
estornos e sem tipos com `considers_consumption` falso) numa janela móvel (padrão 6
meses). Transferência deixou de contar como consumo (vale também para a curva ABC).

| Método | Rota | Ação |
|---|---|---|
| POST | `/api/stock/consumption-average/recalc` | Recalcula (item específico via `item_code`, ou todos) |
| GET | `/api/stock/consumption-average/{itemCode}` | Consulta consumo médio mensal do item |

## 9. Reposição, análise e lotes a vencer

| Método | Rota | Ação |
|---|---|---|
| GET | `/api/stock/replenishment` | Itens com projeção (disponível − reservado + pedido de compra aberto + requisição de compra aberta) ≤ ponto de pedido. PP = TR×CM/CR+ES do cadastro; sem CM, o consumo médio apurado; sem fórmula, o estoque mínimo. Sugerido = PP + segurança − projeção, ≥ lote mínimo, no múltiplo |
| POST | `/api/stock/replenishment/purchase-requisition` | `{items:[{item_code, quantity}]}` → requisição de compra (VSUP0300) |
| GET | `/api/stock/analysis?months&idle_days` | Valor, consumo, consumo médio, cobertura (meses), giro anualizado, dias sem movimento, parado, ABC |
| GET | `/api/stock/lots/expiring?days` | Saldo em lote vencido ou a vencer |

## 10. Requisição ao almoxarifado (`/api/stock/material-requisitions`)

Migração 406. Pedido interno (solicitante, centro de custo, almoxarifado, itens).

| Método | Rota | Ação |
|---|---|---|
| GET | `/?status=` · `/{id}` | Consulta (ABERTA, PARCIAL, ATENDIDA, CANCELADA) |
| POST | `/` | Cria |
| POST | `/{id}/issue` | Atende `{lines:[{item_id, quantity, lot?, address?}]}` — saída no centro de custo (ref. `REQUISICAO_MATERIAL`), até o pendente |
| POST | `/{id}/cancel` | Cancela o pendente (o entregue fica) |

---

## Cadastros relacionados
Armazém (`/api/warehouse`; `PUT /{code}` altera — só ADMIN — inclusive `allow_negative_stock`) e Localização (`/api/location` — países/UFs não; ver
[`cadastros-apoio.md`](cadastros-apoio.md) para armazém/localização física). Tipos de
armazém/localização: ver enums `WarehouseType`/`LocationType`.
