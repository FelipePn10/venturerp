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
| GET | `/list` | Lista por página (`limit` até 2000, padrão 500; `offset`, `warehouse_id`, `from`, `to`) |
| GET | `/item/{itemCode}` | Por item |
| GET | `/warehouse/{warehouseId}` | Por armazém |

> ✅ Movimentos atualizam `stock_balances` **na mesma transação** (quantidade, custo
> médio ponderado e último custo). Tipo de entrada padronizado em `IN`. Ver
> `repository/stock/stock_repository_pg.go` (`CreateMovementTx`).

Regras do lançamento (todas em `CreateMovementTx`, o único ponto que grava movimento):

- **Item e almoxarifado têm de ser da empresa.** As tabelas de estoque não têm
  chave estrangeira para eles; a conferência é feita aqui.
- **Endereço:** num almoxarifado com endereços cadastrados, entrada só em endereço
  ativo; endereço bloqueado não recebe nem entrega. Inventário, contagem cíclica e
  estorno passam, porque são o que acerta um endereço bloqueado.
- **Saída sem endereço é repartida** pelos endereços (`baixa_estoque.go`): primeiro
  a linha sem endereço, depois os endereços em ordem, usando o livre e só então o
  reservado, sem tocar em endereço bloqueado. O que faltar só fica negativo onde o
  almoxarifado permite, sempre na linha sem endereço.
- **Lote:** saída com lote sai das linhas do lote e não passa do saldo dele. Sem
  nenhum saldo em lote do item no almoxarifado, o lote digitado é só referência do
  documento. Saída sem lote baixa os lotes que sobrarem no endereço, primeiro o que
  vence antes, e registra isso na observação do movimento.
- **Valor da saída é sempre o custo médio**, mesmo com preço digitado; só o desfazer
  de uma entrada (`SaidaPeloPreco`) sai pelo preço dela. Sobra de ajuste sem preço
  entra pelo médio.
- **Entrada sobre saldo negativo:** cobre primeiro o que saiu sem saldo, e o que
  sobra fica pelo preço da entrada.

- **Operações do mesmo item são serializadas** (`TravarItensTx`): toda rotina que
  mexe em saldo, lote ou reserva trava o item antes de tocar nas linhas, em ordem
  crescente de código. O fechamento do mês trava a mesma família por inteiro, então
  lançamento em curso não entra nem sai da foto. Conflito que ainda aconteça responde
  409 com "nada foi gravado, tente de novo".
- **Lote de referência:** sem saldo em lote no almoxarifado, o lote da saída é só
  referência do documento e não grava linha de lote.
- **Transferência entre endereços sem lote** só move o que está fora de lote; o resto
  pede o lote. Lote que não permite transferência é recusado também entre endereços.
- **Entrada de produção valorizada:** o produto entra pelo que a ordem gastou (material
  baixado e mão de obra, menos o que entregas anteriores levaram). Sem custo apurado vale
  o custo-padrão; sem padrão, o médio do item. Nunca a zero.
- **Baixa da ordem não repete:** a baixa automática da entrega e o backflush do apontamento
  tiram só o que a produção acumulada exige menos o que já saiu para a ordem.
- **Nota do pedido com onda:** a saída herda o lote e o endereço separados
  (`invoiced_qty` na linha da onda). Lote ou endereço digitados na nota têm prioridade.
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
Transferência originada de documento usa a mesma operação com
`ReferenciaTipo`/`ReferenciaCodigo` do documento: as pernas guardam a referência
dele e se desfazem pelo documento, não pelo estorno avulso.

**Bloqueio do almoxarifado de inspeção (migração 418).** Material num
almoxarifado de localização `INSPECAO` só sai pela decisão de uso do lote de
inspeção (`reference_type QUALITY_LOT`, VINS0300). `CreateMovementTx` recusa
qualquer saída (`OUT`, `TRANSFER_OUT`, `REP`, ajuste negativo) dele com outra
referência: requisição, apontamento/consumo de OF, separação, faturamento,
transferência e movimento manual. Continuam passando:
- o estorno ou cancelamento da entrada (`NF_ENTRADA`, `PURCHASE_ORDER`) — o cancelamento da entrada também cancela os títulos a pagar da nota e os tira da lista (`is_active=false`; migração 000422 corrige os que ficaram visíveis);
- o estorno de movimento (`ESTORNO`);
- o inventário (`INVENTORY`, `CONTAGEM_CICLICA`);
- as referências antigas da inspeção de recebimento.

Antes, o almoxarifado de inspeção só ficava fora do ATP e da reposição.
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

`GET /list` lista as reservas (`item_code` e `active=true` opcionais). A reserva
sem endereço é repartida pelo livre de cada linha de endereço; `GetBalance` soma os
endereços. Almoxarifado com "aceita reservas" desligado recusa a reserva manual e a
onda; pedido e romaneio seguem sem reservar. Reserva encerrada não muda de situação.
Reserva de onda ou de demanda independente só se encerra pelo próprio documento. A
NF-e de saída consome só o que faturou (`ConsumirReservaParcial`).

| Método | Rota | Ação |
|---|---|---|
| POST | `/create` | Cria reserva |
| PATCH | `/{id}/release` | Libera |
| PATCH | `/{id}/consume` | Consome |

> A autorização da NF-e de saída consome as reservas do pedido automaticamente.

**Reserva da demanda independente** (`reference_type = DEMANDA_INDEPENDENTE`,
`reserva_demanda_pg.go`): a entrega da OF de uma demanda de finalidade própria
(amostra, feira, consumo interno, assistência, projeto) reserva o que entrou, na
mesma transação e na linha do movimento — almoxarifado, endereço e lote, porque a
onda de separação lê o saldo do lote. A saída com essa referência consome a
reserva; o estorno da saída a refaz; encerrar ou cancelar a demanda libera o resto.

## 4. Inventário (`/api/inventory`; `/api/stock/inventories` por compatibilidade)

| Método | Rota | Ação |
|---|---|---|
| POST | `/` | Abre e grava a foto do saldo |
| GET | `/`, `/{id}`, `/{id}/items` | Consulta |
| POST | `/count` | Conta uma linha (item + `address` + `lot`) |
| POST | `/{id}/finish-count` | Encerra a contagem e vai para a conferência |
| POST | `/{id}/recount` | Devolve linhas à contagem (`line_ids`; vazio = as fora da tolerância) |
| POST | `/{id}/close` | Fecha (`reason`, `zerar_nao_contados`) ou manda para a alçada |

Situações: `OPEN` → `EM_CONFERENCIA` → (`AGUARDANDO_APROVACAO`) → `CLOSED`. Um
almoxarifado tem um só inventário em andamento (índice único).

- **Foto por endereço e lote:** cada lote do endereço é uma linha, e o que o
  endereço tem além dos lotes é a linha sem lote. Item fora da lista entra como
  linha nova, com foto zero.
- **Inventário sem bloqueio:** saída do mesmo endereço durante a contagem pode deixar o
  endereço negativo depois do ajuste, porque a diferença vale sobre a foto. O lote nunca
  fica negativo por isso, e a Conferência do saldo acusa o endereço. Quem não aceita
  esse risco abre o inventário com "bloquear movimentos".
- **O ajuste é contado − foto**, aplicado ao saldo atual, no endereço e no lote da
  linha, ao custo médio. O que entrou ou saiu depois da abertura continua valendo.
- **`blocks_movements`:** recusa movimento dos itens no almoxarifado (no parcial, só
  os itens do inventário) até terminar. O próprio ajuste passa. Efeito de NF-e
  recusado por isso fica em pendência e é reprocessado.
- **`blind_count`:** enquanto `OPEN`, o saldo, a diferença e a contagem anterior não
  saem do servidor (`GetInventoryUseCase.ListItems`). Fechar exige passar pela
  conferência.
- **Aprovação (`approval_mode`):** `NUNCA`, `FORA_TOLERANCIA` (`tolerance_pct`,
  `tolerance_value`) ou `SEMPRE`. Usa o motor de aprovação, tipo `AJUSTE_INVENTARIO`;
  sem política ativa o fechamento é recusado. Aprovar lança os ajustes
  (`AprovacaoDeInventario`); reprovar volta para `EM_CONFERENCIA`.

A **contagem cíclica** (VEST0500) também ajusta: dentro da tolerância na conclusão;
divergente, só na aprovação (ref. `CONTAGEM_CICLICA`).

## 4.2 Controle de material

Rotas em `/api/stock`: `item-control/{itemCode}`, `lots/status`, `lots/implant`,
`lot-statuses` e `parameters`. Tela: VEST0100 › Lotes e VEST0200 › Situações de lote.

- **Controle do item** (`controls_lot`, `controls_address`, `lot_issue_rule`):
  lançamento manual, requisição e transferência digitados exigem o lote e o endereço
  (`StockMovement.ExigirRastreio`). NF-e e apontamento seguem pela baixa automática,
  na regra do item (FEFO ou FIFO).
- **Situação de lote:** cadastro da empresa dizendo se o lote permite saída,
  transferência e se conta no disponível. Lote sem situação está liberado. Toda troca
  fica em `stock_lot_status_events`. Almoxarifado, endereço e série têm controle próprio.
- **Lote vencido** (`expired_lot_action`, padrão `BLOQUEAR`): sai do disponível, da
  separação e da saída. Transferência continua permitida e o descarte é por `ADJUSTMENT`.
- **Saída sem lote** não alcança o que está retido em lote sem saída ou vencido.
- **Passam sem olhar a situação:** inventário, contagem cíclica, estorno, decisão de
  uso, desfazer entrada e ajuste.
- **Implantar lote:** dá lote a saldo que entrou sem lote, no mesmo endereço; a
  quantidade não muda e o registro fica no extrato (`IMPLANTACAO_LOTE`).
- **Entrada com lote** cria o cadastro do lote com a validade do movimento.
- **NF-e de saída:** a linha guarda máscara, lote e endereço (`fiscal_exit_items`). A
  máscara vem da linha do pedido. O MRP ainda conta o saldo retido em lote.

## 4.3 Data do documento e fechamento mensal

- **`movement_date`** é a data que vale para kardex, posição e registro de inventário;
  `created_at` é só quando foi gravado. Lançamento manual aceita data passada dentro
  do período aberto. Nota de entrada usa a data de entrada e inventário a data da foto;
  se ela cair em mês fechado, o movimento entra hoje (`DataAjustavel`).
- **O custo segue em tempo real.** Lançamento com data passada marca `backdated`.
- **Fechamento** (`/api/stock/closings`, VEST0200 › Fechamento mensal): `simulate` faz
  tudo e desfaz. Impedem fechar: mês em curso, mês fora de sequência, inconsistência de
  saldo e inventário em andamento.
- **Recálculo** (`entity.RecalcularCusto`): só dos itens com `backdated` no mês, a partir
  do primeiro deles, na ordem data + número. Os demais não são tocados. O valor anterior
  fica em `stock_cost_recalc_log`. Recalcula em níveis: a diferença do consumo vai para a entrada da ordem e dali para o produto. Antes não recalculava a entrada de produção pelo novo custo
  dos componentes.
- **Foto:** `stock_closing_balances`, o saldo inicial do mês seguinte.
- **Diferenças** (`stock_cost_variances`): `RECALCULO` (saída que mudou de valor) e
  `SALDO_NEGATIVO` (valor da entrada que cobriu saldo negativo). O líquido vira um
  lançamento CMV × Estoque no último dia do mês, se as contas estiverem configuradas.
- **Reabrir:** só o último mês, com motivo; estorna o lançamento. Os custos
  recalculados ficam.

## 4.1 Conferência do saldo (`/api/stock/inconsistencies`)

`GET` compara o saldo com os movimentos, os lotes com o saldo e o reservado com as
reservas ativas (filtros `warehouse_id`, `item_code`). `POST /fix` (só ADMIN, com
`reason`) refaz o saldo a partir da origem sem criar movimento e registra em
`stock_inconsistency_fixes`, que só cresce. Corrige quantidade; o custo segue o
médio do item. Período de estoque fechado recusa o acerto.

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
| GET | `/api/stock/replenishment` | Sugestões de reposição (regras abaixo) |
| POST | `/api/stock/replenishment/generate` | `{lines:[{item_code, warehouse_id?, source_type, source_warehouse_id?, quantity}]}` → ordem de reposição, requisição de compra e ordem planejada |
| POST | `/api/stock/replenishment/purchase-requisition` | Rota antiga, mantida para o desktop já instalado: `{items:[{item_code, quantity}]}` → requisição de compra |
| GET · PUT | `/api/stock/replenishment/policies` | Lista e grava a política do item no almoxarifado |
| DELETE | `/api/stock/replenishment/policies/{id}` | Exclui a política; o item volta a seguir o cadastro |
| GET | `/api/stock/analysis?months&idle_days` | Valor, consumo, consumo médio, cobertura (meses), giro anualizado, dias sem movimento, parado, ABC |
| GET | `/api/stock/accuracy?months&warehouse_id&tol_a&tol_b&tol_c` | Acuracidade das contagens encerradas (inventário fechado e contagem cíclica): por linha, por valor, por mês e por classe, com item e endereço de erro repetido |
| GET | `/api/stock/aging` | Saldo em faixas de tempo sem movimento real |
| GET | `/api/stock/lots/expiring?days` | Saldo em lote vencido ou a vencer |

### Regras da análise

- **Acuracidade por linha:** contagens dentro da tolerância ÷ contagens. Tolerância padrão: classe A exata, B 2%, C 5%; item sem classe usa a C.
- **Acuracidade por valor:** 1 − Σ|contado − esperado| × custo ÷ Σ esperado × custo.
- **Erro repetido:** item ou endereço fora da tolerância duas vezes ou mais na janela.
- **Parado e idade:** o último movimento real é o de `movement_date`, sem contar inventário, contagem cíclica, transferência, troca de endereço, ajuste de custo e estorno. Consumo usa `movement_date`.

### Regras da reposição (migração 461)

- **Quem entra:** item ativo no planejamento, liberado na governança e que não é fantasma. Item de terceiro e serviço ficam fora.
- **Política por almoxarifado:** mínimo, máximo, segurança e origem (`COMPRA`, `TRANSFERENCIA`, `PRODUCAO`). Campo vazio herda do cadastro. Item com política só é planejado nos almoxarifados dela; sem política, vale o cadastro sobre o saldo somado dos almoxarifados de promessa, e a origem vem do tipo do item.
- **Ponto:** TR×CM/CR+ES do cadastro, ou o mínimo. Ele já contém a segurança: a fórmula a soma, e no mínimo ela é o piso. Nunca soma de novo na quantidade.
- **Projeção:** físico − reservado − lote retido − requisição aberta sem reserva + o que está a caminho. A caminho: pedido de compra, ordem de produção aberta, ordem planejada não firme e transferência pendente. Requisição de compra não tem almoxarifado: cobre uma vez só, o primeiro destino por compra que precisar.
- **Quantidade:** projeção ≤ ponto dispara e repõe até o máximo, ou até o ponto quando não há máximo. Compra e produção respeitam lote mínimo e múltiplo.
- **Transferência:** limitada ao que a origem pode ceder sem cair abaixo do próprio ponto; sem saldo, a linha aparece com quantidade zero e aviso.
- **Alerta `ESTOQUE_ABAIXO_MINIMO`:** lê a política e compara o disponível somado do almoxarifado, um alerta por item e almoxarifado. As colunas de mínimo de `stock_balances` nunca foram preenchidas.

Depois de toda saída, o reservado que passar do saldo de um endereço vai para o livre dos outros (`reequilibrarReservadoTx`): a reserva é do item, não do endereço. O MRP não conta lote que a situação tira do disponível.

## 9.1 Transferência em duas etapas (`/api/stock/transit`)

Migração 462. O material sai da origem para um almoxarifado real de localização `TRANSITO` e só entra no destino ao ser recebido. Sendo um almoxarifado comum, segue no custo, na posição, no Registro de Inventário e no fechamento. Não conta para promessa; na reposição, conta como a caminho no destino. Vale dentro da mesma empresa: entre CNPJs a transferência exige nota fiscal.

| Método | Rota | Ação |
|---|---|---|
| GET | `/api/stock/transit?open=true` | Lista os envios |
| POST | `/api/stock/transit` | Envia `{item_code, from_warehouse_id, to_warehouse_id, transit_warehouse_id?, quantity, lot?, from_address?, expected_date?, notes?}`. Sem `transit_warehouse_id`, usa o único de trânsito da empresa |
| POST | `/api/stock/transit/{id}/settle` | `{action: RECEBER\|DEVOLVER\|PERDA, quantity, address?, reason?}`; devolução e perda exigem motivo |

Os movimentos têm referência `TRANSITO` e não se estornam soltos: o caminho é devolver à origem. A entrega de requisição de transferência com `via_transit` cria o envio. O custo em trânsito é a média do almoxarifado de trânsito, que mistura envios simultâneos do mesmo item.

## 9.2 Onda de separação (`/api/stock/waves`)

| Método | Rota | Ação |
|---|---|---|
| GET | `/?status=` | Lista as ondas, com as linhas |
| GET | `/order-lines/{orderCode}?warehouse_id=` | O que falta separar do pedido: pendente, menos onda aberta, menos separado ainda não faturado |
| POST | `/` | Cria. Sem `code`, usa o próximo número da empresa |
| POST | `/{code}/confirm` | `{lines:[{id, picked_qty}]}`, opcional; linha fora da lista foi separada inteira |
| POST | `/{code}/cancel` | Devolve o reservado |

Linha com referência `SALES_ORDER` usa a reserva do pedido e **não baixa o estoque**: quem baixa é a nota fiscal. Ao criar, a reserva do pedido passa para a onda (`order_reserved_qty`); ao encerrar, volta ao pedido. A nota ainda não herda o lote e o endereço separados. Linha sem pedido baixa o separado, com referência `ONDA_SEPARACAO`.

## 9.3 Coletor (VEST0150, `GET /api/stock/scan?code=&warehouse_id=`)

Tela do desktop para leitor USB ou Bluetooth, que funciona como teclado: lê o código e manda Enter. Modos: consultar, separar onda, contar inventário, mover entre endereços, receber envio em trânsito e etiquetas. A separação guarda o andamento no aparelho até a conclusão.

A leitura tenta, nesta ordem: endereço do almoxarifado em uso, código do item, código de barras do fornecedor, EAN das notas de entrada e lote. Etiqueta GS1 (01, 10, 17, 30, 37) traz item, lote, validade e quantidade. Código ligado a mais de um item é recusado com mensagem. O item não tem código de barras próprio: a etiqueta interna é o próprio código do item.

## 10. Requisição ao almoxarifado (`/api/stock/material-requisitions`)

Migrações 406 e 461. Pedido interno (solicitante, centro de custo, almoxarifado, itens). Com `destination_warehouse_id` é uma **ordem de reposição**: a entrega transfere ao destino, ao custo da origem, em vez de baixar no centro de custo.

Ao abrir, reserva o pendente de cada linha até o saldo livre (`stock_reservations`, referência `REQUISICAO_MATERIAL` e `reference_item_code` = linha). O que não coube fica pendente sem reserva e pesa na reposição. Almoxarifado que não aceita reserva só não reserva. Com política ativa de aprovação `REQUISICAO_MATERIAL`, nasce `PENDENTE_APROVACAO` e só reserva depois de aprovada; a ordem gerada pela reposição já nasce aprovada, como a move order de reposição do Oracle.

| Método | Rota | Ação |
|---|---|---|
| GET | `/?status=` · `/{id}` | Consulta (PENDENTE_APROVACAO, ABERTA, PARCIAL, ATENDIDA, CANCELADA, REPROVADA) |
| POST | `/` | Cria |
| POST | `/{id}/issue` | Atende `{lines:[{item_id, quantity, lot?, address?, destination_address?, via_transit?}]}` — consome a reserva e lança a saída ou a transferência (ref. `REQUISICAO_MATERIAL`), até o pendente |
| POST | `/{id}/cancel` | Cancela o pendente e devolve a reserva (o entregue fica) |

A entrega de consumo se desfaz pelo estorno do movimento: o material volta, a linha fica pendente sem reserva e a requisição reabre. A entrega de ordem de reposição se desfaz lançando a transferência de volta. Reserva da requisição só se encerra pela própria requisição.

---

## Cadastros relacionados
Armazém (`/api/warehouse`; `PUT /{code}` altera — só ADMIN — inclusive `allow_negative_stock`) e Localização (`/api/location` — países/UFs não; ver
[`cadastros-apoio.md`](cadastros-apoio.md) para armazém/localização física). Tipos de
armazém/localização: ver enums `WarehouseType`/`LocationType`.
