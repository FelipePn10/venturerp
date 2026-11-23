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
| POST | `/operations/explode` | Aplica o roteiro do item na OF sem etapas (legado; o `start` faz sozinho) |
| GET | `/{id}/operations` | Lista operações da OF e andamento |
| POST | `/operations/advance` | Avança a operação (conclui etapa, libera a próxima) |

Corpo de `/operations/advance`: `{ "operation_id": 123, "status": "IN_PROGRESS", "actual_hours": 0, "reason": "Retomada" }`.

Transições permitidas:

- `PENDING` → `IN_PROGRESS` ou `SKIPPED` (dispensa com motivo).
- `IN_PROGRESS` → `PAUSED`, `INTERRUPTED` ou `DONE`.
- `PAUSED`/`INTERRUPTED` → `IN_PROGRESS`.
- `DONE`/`SKIPPED` são terminais. Pausa e interrupção exigem motivo.

O início depende das predecessoras concluídas/dispensadas. Sem rede explícita,
valem as etapas de sequência anterior. A conclusão libera as sucessoras, mas não
inicia máquinas automaticamente. `started_at` preserva o primeiro início;
`completed_at` identifica a conclusão/dispensa. A lista retorna `can_start` e
`execution_history`, incluindo autor, estado e horário de cada alteração.

As dependências são copiadas para a OF na geração das etapas (migração 358).
Alterar a rede de engenharia depois não altera a rede da OF existente. A migração
preenche ordens anteriores usando a rede disponível no momento da atualização;
não recupera versões históricas que não foram armazenadas.

O histórico de execução é registrado por trigger na mesma transação, com fonte
`OPERATION` ou `SCANNER` e usuário autenticado (migração 359). Atualização/exclusão
direta desses eventos é bloqueada. Atualizações legadas sem contexto registram
fonte `LEGACY` e não inventam um autor. Isso não substitui um sistema completo de
estorno contábil nem identifica uma estação física não informada pelo cliente.

A criação manual e a liberação de planejadas na API mantêm OF, materiais, etapas,
inspeções e aplicação da programação de máquinas na mesma transação. Na liberação,
a mudança de situação, requisição e ordens de serviço também participam dela.
Bloqueios por ordem planejada impedem que duas liberações gerem duas OFs.
Ordens planejadas que já geraram OF não retornam a PLANNED por essa ação: ajustes
seguem na OF correspondente. A numeração da OF é reservada atomicamente por
empresa (migração 360); falhas de criação podem deixar lacunas na numeração.

A entrega final exige etapas concluídas/dispensadas, ausência de inspeção pendente
ou rejeitada e ausência de pedido de serviço pendente. Inspeção obrigatória sem
plano ativo impede a criação da OF. A aprovação da qualidade continua sendo uma
ação explícita. Concluir a última etapa não equivale à entrada do acabado: o
scanner orienta a entrega de produção. Entrega, movimentos e saldos são atômicos;
repetir a chave da entrega com outros dados é rejeitado. Ordens concluídas ou
canceladas não podem ser reabertas por uma entrega posterior.

> ✅ **Baixa automática (mig 506):** a fonte é a **lista de materiais da OF**
> (`production_order_materials`, `material_kind = DEMAND`, `automatic_issue = true`),
> não a estrutura atual — respeita substitutos e materiais incluídos depois. O
> apontamento baixa `OUT` na fração `(apontado antes + bom) ÷ planejado`; a entrega
> baixa `REP` na fração entregue (1 na final). Cada baixa desconta o que já saiu
> para a OF (`BaixasDaOrdem`), então consumo manual do mesmo item nunca sai duas
> vezes. Material de baixa manual fica de fora. `backflush_warehouse_id` só
> sobrescreve o almoxarifado do material. Ver `baixa_automatica_da_of.go`.
> A mig 506 liga `items.warehouse_automatic_low` (padrão TRUE) e os materiais das
> OFs abertas/em produção.

> ✅ **Etapas sem trabalho manual:** a OF nasce com as etapas do roteiro aprovado;
> se não tiver (item ganhou roteiro depois), `start` chama `ExplodeRoute` e recusa
> iniciar só se ainda assim não houver etapa.

> 🔧 **Ficha de Produção da Ferramenta:** define qual **série** de cada ferramenta roda
> cada operação da OF, com substituição rastreada e débito de vida útil por série no
> apontamento. Ver [`ficha-producao-ferramenta.md`](ficha-producao-ferramenta.md).

---

## Origem da OF
**Número que o usuário vê:** "OF n" em tela, impresso e código de barras
(`OF%06d`) é `production_orders.id`. `order_number` é interno e difere do id nas
OFs firmadas pelo MRP; listas e mensagens (inclusive
`GetPlannedOrderActiveProductionOrder` e o APS) usam o id.

**OF manual e estrutura:** criar a OF copiando a estrutura recalcula antes os
fatores de conversão das linhas daquele pai (`RecalcularFatores.RecalcularDoPai`),
sem esperar o MRP; linha sem conversão entre a unidade da estrutura e a de
estoque recusa a OF com a lista das linhas (422).

A OF normalmente nasce ao **firmar** uma ordem planejada `PRODUCTION`
(`GET /api/planned-order/{code}/firm`), que cria a OF automaticamente na primeira
firmação. Ver `planned_order_uc/firm_planned_order_uc.go` e
[`mrp-calculo.md`](mrp-calculo.md).

### Cliente, projeto e identificação (mig 506)

- `production_orders.customer_code` e `project_id` (FK composta por empresa).
  `projects` é cadastro por empresa (`/api/projects`, tela VPLA0120), código único
  por `(enterprise_id, upper(btrim(code)))`; a mig semeia a partir de
  `sales_orders.project_code` e `independent_demands.projeto`.
- Herança no banco, só preenche nulos (`erp_of_herdar_origem`): pedido de venda
  (`sales_order_code` ou `sales_order_planned_orders`) → demanda independente
  (`independent_demand_orders`) → OF pai (`production_chain_links`). Gatilhos no
  INSERT da OF e na criação tardia dos vínculos. Trocar na pai propaga para as
  filhas que estavam nulas ou com o valor antigo da pai.
- `PUT /api/production-order/{id}/origin` `{customer_code, project_code}` (0/"" limpa);
  projeto precisa estar ATIVO e, com cliente, preenche/valida o da OF. Recusa OF
  CLOSED/CANCELLED.
- `list` e `get` devolvem `origin`: item de negócio e nome, cliente, projeto,
  pedidos, demandas, OF pai (id, número, item), raiz da cadeia, nível e nº de filhas
  (`IdentificacaoDasOFs`, CTE recursiva pela cadeia).

---

### Cronômetro da etapa

`POST /operations/advance` sem `actual_hours` ao sair de `IN_PROGRESS` (pausa,
interrupção, conclusão) grava o tempo corrido desde a última entrada em
andamento, lido da trilha `production_operation_execution_events`
(`entity.InicioDoTrecho` / `HorasDoTrecho`; sem trilha, `started_at`). Sem
`produced_qty` na conclusão, a vida útil da ferramenta usa a quantidade da OF.
Horas digitadas continuam valendo. A resposta traz `recorded_hours`,
`hours_from_clock`, `recorded_qty`, `qty_proposed`; a listagem traz
`running_since` da etapa em andamento. A leitura por código de barras
(`ExecuteScan`, concluir sem horas) aplica a mesma regra em SQL. Trecho acima
de `entity.HorasEtapaEsquecida` (12 h) não é gravado sozinho: tela e leitura
devolvem 400 pedindo as horas (etapa esquecida aberta). Testes:
`TestCronometroDaEtapa` (unidade), `TestCronometroDaEtapaIntegration` e
`TestLeituraConcluiPeloCronometro`.

## 3. Custo real da OF

| Método | Rota | Ação |
|---|---|---|
| POST | `/{id}/settle-cost` | Recalcula o custo real sob demanda (a tela não usa mais) |
| GET | `/{id}/cost` | Consulta a apuração + variâncias vs. padrão |

> ✅ **automático:** apontamento, consumo, entrega e devolução de sucata chamam
> `SettleProductionCostUseCase.Atualizar` depois do sucesso; falha vira
> `custo_pendente` com o motivo (o movimento não é desfeito). Horas gravadas na
> etapa (`/operations/advance` com `recorded_hours > 0`) e o apontamento pela
> leitura do código de barras (`ExecuteScan`, ação de apontar, não repetida)
> também atualizam. Fechar a OF (`/{id}/close`) também apura.
>
> **OFs antigas sem custo:** na subida da API, `custoof.Rotina` apura em lotes de
> 200 as OFs não canceladas que têm apontamento, horas de etapa ou movimento de
> estoque e nenhuma linha em `production_order_costs`
> (`OFsSemCusto`), no contexto da empresa de cada OF. Falha de uma OF é
> registrada no log e não para as outras. Detalhes da
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

A montagem da lista é `CreateProductionOrderUseCase.MateriaisDaEstrutura`
(componentes vigentes, máscara, perda, substituto primário, quantidade fixa,
sem co-produto nem material do cliente) e cada linha recebe a etapa do roteiro
(`route_operation_sequence`, da estrutura) — é o que a aba Execução usa para
mostrar o material de cada etapa e o que a ficha da OF imprime.

**OF sem lista (criada antes da regra ou pela carga do demo).** Na subida da
API, `materiaisof.Rotina` procura OFs `OPEN`/`IN_PROGRESS` sem nenhuma demanda e
cujo item tem estrutura ativa (`OFsSemMateriais`, lotes de 200, todas as
empresas) e completa cada uma pela estrutura vigente, no contexto da empresa da
OF (`CompletarMateriais`: trava a OF com `FOR UPDATE`, confere empresa,
situação e ausência de lista na mesma transação — idempotente; OF de outra
empresa, fechada ou já completada não muda). A observação da linha diz "lista
completada pela estrutura vigente". Etapa só é gravada onde está nula. OF que
já produziu (peças apontadas, entregue, refugo ou movimento de estoque por ela)
recebe a lista com `automatic_issue=false`: a baixa automática sai na proporção
do TOTAL produzido e lançaria hoje, de uma vez, o consumo do passado. Testes:
`TestCompletarMateriaisDaOFSemLista` (duas empresas) e
`TestListaCompletadaEmOFJaProduzidaFicaComBaixaManual`.

A quantidade de cada componente usa a mesma conta do MRP
(`QuantidadeDoComponente`): perda pela fórmula do parâmetro 20 dos parâmetros de
planejamento, 6 casas no cálculo e arredondamento para cima em 4 casas — a
escala do pedido de compra e do estoque. O mesmo vale para a baixa automática
no apontamento e na finalização, para o recebimento de co-produto e para o custo
padrão; antes a OF usava sempre a fórmula padrão e pedia 0,00002 kg a mais que o
comprado. A baixa na finalização usa a quantidade convertida para a unidade de
estoque do componente.

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

### Faltas de material antes de liberar

`GET /api/production-order/{id}/material-availability` devolve, por material
de demanda da OF: quanto falta baixar (`remaining`), o saldo livre do depósito
do material (saldo − reservado), o que as **OFs abertas que estão na frente**
vão consumir do mesmo item (`committed_earlier` — início anterior, ou mesmo dia
e criada antes), o disponível para esta OF e a falta. Situação por linha e da
OF: `COMPLETA`, `PARCIAL`, `SEM_SALDO` ou `SEM_LISTA`.
No detalhe (`/{id}/material-availability`), o material que já saiu todo
também vem na lista (`remaining` 0, situação `COMPLETA`) para a tela mostrar
previsto / já saiu / falta sair de tudo; ele não conta no resumo nem nas faltas.

`GET /api/production-order/material-availability?ids=1,2,3` (até 500) devolve
só o resumo por OF, para o selo da lista.

A peça reservada para a OF (`reserved_for_order`, o componente que a OF filha
entregou para ela — ver "Cadeia OF filha → OF pai") está fora do saldo livre mas
conta como disponível para ela; no `committed_earlier` das outras OFs, a parte
já reservada para cada uma não pesa na fila.

### Conferência ao iniciar a OF (parâmetro 52)

`POST /api/production-order/{id}/start` e o coletor (ação `INICIAR`) conferem,
pelo parâmetro de planejamento 52 (`CONFERE_COMPONENTES_INICIO_OF`, migração
503), o material a baixar sem disponível e as OFs filhas ainda não concluídas
(ou ordem planejada filha sem OF):

| Valor | Efeito |
|---|---|
| `NAO` | não confere |
| `AVISAR` (padrão) | inicia e devolve os avisos em `warnings` (no coletor, em `message`) |
| `BLOQUEAR` | recusa o início (422) listando cada pendência |

### Cadeia OF filha → OF pai (migração 503)

Ver `.ai/tasks/cadeia-of-pai-filha.md`. Firmada a sugestão do MRP de um
componente fabricado, `production_chain_links` liga a ordem planejada dele à
ordem pai (a sugestão do pai já firmada, a planejada firme ou a OF do empenho),
e o pedido de venda do pai desce para `sales_order_planned_orders` de toda a
cadeia abaixo. Vale firmando de cima para baixo ou de baixo para cima; o
pegging só é lido na hora de firmar (`VincularSugestao`).

A entrega da OF filha reserva o produzido para a OF pai (`stock_reservations`,
`reference_type = OF_COMPONENTE`, `reference_code` = id da OF pai,
`reference_item_code` = linha de material), até o que falta baixar dela, no
almoxarifado da entrega — material da OF pai em outro almoxarifado não é
reservado. A saída com referência `PRODUCTION_ORDER` da OF pai (baixa,
requisição, abastecimento da linha) consome a reserva primeiro; a saída de
qualquer outro documento que deixaria o saldo abaixo do reservado é recusada.
Saídas liberadas (inventário, estorno, ajuste, devolução) passam, e a reserva
cai até o saldo que sobrou — da OF pai que começa mais tarde para a que começa
antes —, para não ficar reserva de peça que já saiu travando o item.
Com uma OF filha abastecendo mais de uma OF pai (lote mínimo, lote por
período), a entrega reserva primeiro para a OF pai que começa antes; o que
passa da necessidade dos pais fica livre. A OF pai
concluída, encerrada ou cancelada libera a sobra (gatilho
`trg_liberar_reservas_de_componente`). A reserva pode ser cancelada à mão para
liberar a peça a outra OF.

OF pai criada **depois** da entrega da filha (firmar de baixo para cima: a
bucha é feita antes de o RN ser firmado, ou a planejada do RN só é liberada
depois): ao nascer, a OF pai reserva o que as filhas ligadas a ela entregaram
e ainda está livre (`ReservarEntregasDasFilhasTx`, chamado em
`CreateWithMaterialsAndCallback`). Teto por material: o que falta baixar e o
entregue pelas filhas menos o já reservado ou consumido pelas outras OFs pai
dessas filhas; vai nas linhas (almoxarifado, endereço, lote) das entregas,
mais nova primeiro, nunca além do saldo livre. Peça comprada livre no mesmo
almoxarifado não é tomada.

Sequenciamento (APS): a OF filha entra antes da OF pai, e a pai só começa
depois do término das filhas programadas na mesma rodada ou, fora dela, do
término já gravado em `production_sequences` (`CadeiaRepository`). Na
programação regressiva a pai é programada primeiro e a filha termina até o
início dela. OF filha concluída, encerrada ou cancelada não segura nada.

`GET /api/production-order/{id}/chain` devolve os pedidos de venda, as
demandas independentes (`independent_demands`), as ordens pai e as ordens
filhas da OF (com o reservado de cada filha para ela).

**Demanda independente (produção para estoque, sem pedido).** A cadeia em si
não depende da origem: os elos vêm do pegging (SUGESTAO, ORDEM_PLANEJADA ou
ORDEM_PRODUCAO, este último para a OF criada pelo "Gerar OF" da demanda). A
demanda NÃO é copiada para as filhas — `independent_demand_orders` conta como
cobertura do item da demanda, e gravar a bucha ali atenderia o RN com bucha.
A leitura sobe pelos ancestrais (CTE recursiva em `CadeiaDaOF`, pela planejada
pai ou pela OF pai, sempre com `enterprise_id`) e junta as demandas ligadas a
qualquer um deles. Demanda de estoque (ESTOQUE/REPOSICAO/ANTECIPACAO_VENDA)
fica ABERTA depois da OF concluída: é consumida pela venda ou vence; demanda de
finalidade própria (AMOSTRA…) recebe a reserva do produzido na entrega.
Aceite: `scripts/test-cadeia-demanda-independente.sh` (148 checagens: MRP,
firmar, elos, reserva OF_COMPONENTE, parâmetro 52, ordem do roteiro, consumo,
entrada em estoque sem nova sugestão, Gerar OF, demanda + pedido no mesmo
item, AMOSTRA, CRP e APS progressivo/regressivo; `PARAR_APOS=3` deixa o
cenário para `scripts/ui-sim/cadeia-demanda.mjs` do front).

Aceite pela API, com MRP real: `scripts/test-cadeia-of.sh` (monta os próprios
cadastros; precisa de `PGURL`, `ADMIN_EMAIL` e `ADMIN_PASSWORD`; `PARAR_APOS=5`
deixa o cenário montado para a simulação da tela no front,
`scripts/ui-sim/cadeia-of.mjs`).

## 7. Atomicidade e serviços de terceiros

Entrega, linhas EPP/EPE, REP, co-produtos, movimentos e saldos são gravados em
uma única transação. Uma entrega que cruza o planejado é dividida (por exemplo,
`1 EPP + 2 EPE`) sem perder a chave idempotente. O encadeamento
OF → requisição de serviço → OCS é persistido e usado pelo bloqueio de
encerramento com OCS pendente.

## Rodada de validação (22/11/2026)

- **Baixa automática grava consumo**: apontamento da tela, leitura do coletor (APONTAR) e entrega gravam `production_consumptions` junto com a saída; o atendido do material sobe pelo gatilho da 479. O custo real não dobra (só lê consumo sem almoxarifado).
- **Leitura do coletor**: passou a fazer a baixa automática (antes só gravava o apontamento), aceita `machine_id` e devolve `auto_issues`. Regra única em `entity.BaixasAutomaticas`.
- **Início da OF**: material comprado (sugestão de compra firmada na cadeia) não é mais "componente sem OF" — só `order_type = PRODUCTION`.
- **OF só com etapa externa**: sem fim do agendamento, a data de fim é a de necessidade.
