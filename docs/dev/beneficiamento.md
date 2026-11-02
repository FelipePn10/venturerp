# Beneficiamento — material do cliente em poder da empresa

A Usimac recebe matéria-prima do cliente por NF-e de remessa (CFOP 5901), processa
conforme o roteiro e devolve **na mesma nota** em que fatura o serviço: CFOP 5124
para a industrialização e CFOP 5902 para o material voltando. Sobra e sucata
devolvidas saem por CFOP 5903.

Antes disso o controle era manual. O sistema não tinha noção de material que está
aqui mas **não é nosso**.

## Por que um razão separado do estoque próprio

Tabelas: `customer_material_remittances`, `customer_material_items`,
`customer_material_movements` (migration `000370`).

1. Material de terceiro não pode entrar em valoração, custeio nem no líquido do
   MRP. Em `stock_balances`, toda consulta que hoje soma saldo passaria a incluir,
   em silêncio, material que não é da empresa — erro contábil, não de tela.
2. Ele tem dimensões que o estoque próprio não tem: cliente proprietário, NF-e de
   remessa e linha da remessa. O saldo fecha **por remessa**, porque é a remessa
   que tem prazo fiscal.
3. `stock_balances` é `UNIQUE(item_code, mask, warehouse_id)`. Acrescentar
   proprietário mexeria na tabela mais central do ERP, em produção, sem benefício
   para quem já usa o sistema.

O nome é `customer_material_*`, e não `third_party_*`, porque já existe
`third_party_service_*` (migration 000233) — o sentido **oposto**: nós mandando
operação de roteiro para fora (galvanização, têmpera, zincagem).

## Onde as regras moram

| Regra | Onde |
|---|---|
| Saldo nunca negativo (retorno + sobra + sucata ≤ recebido) | `CHECK` no banco |
| Sucata exige destinação; ajuste exige justificativa | `CHECK` no banco |
| Divergência de conferência exige motivo | `CHECK` no banco |
| Mesma NF-e do mesmo cliente não entra duas vezes | `UNIQUE` no banco |
| Movimento idempotente | `UNIQUE (enterprise_id, idempotency_key)` |
| Baixa da linha na mesma transação do movimento | repositório |
| Trava da linha contra faturamento simultâneo | `FOR UPDATE` no repositório |
| Situação derivada do saldo (ABERTA→PARCIAL→ENCERRADA) | repositório, mesma transação |
| Prazo fiscal de 30 dias | caso de uso |
| Material sem pedido entra bloqueado | caso de uso |
| Recebimento divergente entra bloqueado | caso de uso |
| CFOP automático (5902 retorno, 5903 sobra/sucata) | caso de uso |

As invariantes de saldo ficam no banco de propósito: são as que não podem ser
contornadas nem por outro caminho de código, nem por SQL manual numa correção às
pressas. `balance_qty` é coluna **gerada** — não existe estado para divergir dos
movimentos.

## API

Todas as rotas começam por `/api/customer-material` e exigem JWT. Consultar é
`USER`; movimentar, bloquear e encerrar mexem em saldo de material que não é da
empresa e exigem `ADMIN`.

| Método | Rota | Uso |
|---|---|---|
| GET | `/` | remessas, com filtro de cliente, NF-e, pedido, situação, bloqueio, saldo e prazo |
| GET | `/balance` | saldo de terceiros agregado por cliente e item |
| GET | `/{id}` | remessa com as linhas e o saldo de cada uma |
| GET | `/items/{itemId}/movements` | trilha do item |
| GET | `/{id}/audit` | histórico e auditoria da remessa (`?limit=`, teto 500, padrão 200) |
| GET | `/movements` | relatório de movimentação (entrada e saída), com filtro de cliente, período, tipo e estornados |
| GET | `/hours` | relatório de horas por OF de beneficiamento, com filtro de cliente, pedido e período |
| GET | `/{id}/invoice-proposal` | proposta de faturamento: linhas do pedido com saldo e devolução proposta |
| POST | `/notes/{fiscalExitId}/reverse` | estornar a baixa de uma nota cancelada (idempotente) |
| POST | `/` | receber a NF-e de remessa do cliente |
| POST | `/items/{itemId}/movements` | retorno, sobra, sucata ou ajuste |
| POST | `/{id}/block` · `/{id}/unblock` | reter ou liberar material |
| POST | `/{id}/close` | encerrar (com saldo, exige motivo) |
| POST | `/{id}/sales-order` | vincular ou trocar o pedido de beneficiamento (ADMIN) |
| POST | `/notes/{fiscalExitId}/discard` | descartar a nota em rascunho ou rejeitada: devolve o saldo e tira o atendido do pedido (ADMIN) |
| POST | `/hours/{ofId}/sales-order` | ligar a OF aberta à mão a um pedido de venda (ADMIN; só OF com componente do cliente) |

Quantidades e valores viajam como **texto** no JSON: passar por float perderia
centavo e a nota de retorno sairia divergente da de entrada.

`RECEIPT` não é aceito em `/movements`. A entrada é criada pelo recebimento da
remessa; aceitá-la ali permitiria inflar o saldo do cliente sem nota que o sustente.

`dias_para_o_prazo` vem calculado do servidor para a tela não depender do relógio
da máquina do usuário.

## Filtro por prazo

`GET /?with_balance=true&due_until=AAAA-MM-DD` é a fila que evita perder os 30
dias: remessas com material ainda aqui, ordenadas pelo prazo mais próximo.

## Pedido de beneficiamento

O pedido é um **pedido de venda** do cliente, com a natureza de beneficiamento. A
remessa guarda `sales_order_code`; a API devolve também `sales_order_number`, que é
o que a tela mostra, porque é o número que o cliente conhece.

- **Receber ou vincular** confere se o pedido existe nesta empresa, é do mesmo
  cliente e não está cancelado nem faturado. Sem pedido, a remessa entra bloqueada.
- **Faturar** traz o serviço da linha do pedido. Com mais de uma linha com saldo, o
  usuário escolhe qual a nota fatura. A linha é dada por atendida **ao autorizar** a
  nota, não ao criar o rascunho.
- ⚠️ `sales_orders` usa `enterprise_code` (`tenant.Code`); o razão de terceiros usa
  `enterprise_id` (`tenant.ID`). O banco ainda recusa vínculo com pedido de outra
  empresa por gatilho.

## Componente fornecido pelo cliente

`item_structures.customer_supplied` (migração 000453) marca o componente que é do
cliente. A OF cujo item tem um componente assim é **OF de beneficiamento**
(`entity.TemMaterialDoCliente`). O componente marcado:

- não gera necessidade no MRP nem sugestão de compra;
- não entra no custo padrão, na ficha técnica nem na demanda do plano de corte;
- não vira material da OF nem sofre baixa automática no apontamento.

Ao concluir a OF de beneficiamento, o produto **não entra no estoque próprio**: ele
volta ao cliente pela nota de retorno. O apontamento registra só horas.

### Como o MRP enxerga a OF de beneficiamento

O produto pronto não entra em `stock_balances` e o pedido só é atendido quando a nota
é autorizada. Entre uma coisa e outra a linha do pedido continua aberta, sem estoque
que a cubra. Por isso:

- a peça pronta que a nota ainda não levou **conta como estoque do cálculo**
  (`ProducaoAguardandoRetorno`: produzido nas OFs não canceladas − atendido nos
  pedidos, só para item com componente do cliente);
- a ordem firme é suprimento só **pelo que falta chegar**
  (`ListFirmPlannedOrdersByItems`, regra geral do MRP): com a OF concluída ou
  cancelada ela deixa de contar. Antes ela cobria para sempre os pedidos seguintes do
  item, e o segundo pedido não era planejado. Na ordem de compra, o que abate é o
  recebido e o cancelado das linhas do pedido de compra ligadas à ordem.

A OF aberta à mão não tem pedido de origem: **Vincular pedido** em VBEN0300 grava
`production_orders.sales_order_code`, que tem prioridade na ligação das horas ao
cliente. Vale também para subconjunto em nível dependente.

## Relatórios

- **Movimentação** (`/movements`): razão de terceiros mais uma linha sintética
  `RECEIPT` por item recebido, porque o recebimento não é movimento do razão. O `id`
  dessa linha é o da linha da remessa: a chave única na tela é tipo + id. Remessa
  cancelada e, por padrão, movimento estornado ficam de fora.
- **Inventário** (`/balance`): saldo atual por cliente e item, sem remessa encerrada
  ou cancelada: quem encerra com saldo declara que aquele material não volta. O saldo
  segue visível na própria remessa, como "Saldo ao encerrar", e ela não aceita mais
  baixa nem faturamento.
- **Horas** (`/hours`): a OF gerada pelo MRP chega ao cliente por
  `planned_orders → sales_order_planned_orders → sales_order_items → sales_orders`
  (migração 000454). O vínculo é gravado ao firmar a sugestão, a partir do pegging,
  que é refeito a cada cálculo. Se ele não existir, vale
  `planned_orders.sales_order_code → sales_order_demands`. Uma OF que atende vários
  pedidos mostra o de maior quantidade. OF aberta à mão, sem pedido, aparece sem
  cliente.

Movimento avulso (sobra, sucata, ajuste) sem valor informado grava o valor unitário da
nota do cliente; a tela não pede esse valor.

## Regras fiscais da nota de saída

Confirmadas pela contadora da Usimac em **29/09/2026**, sobre a NF-e 5.911. O
levantamento dizia "ICMS 5,93%" para o serviço; a contadora confirmou que a nota
real está certa e que **não há hoje situação em que os 5,93% sejam destacados**.

| Linha | CFOP | CST ICMS | PIS / COFINS |
|---|---|---|---|
| Serviço de industrialização | 5124 | 051 — diferimento (Portaria CAT 22/2007) | CST 01 — 0,65% e 3,00% |
| Material do cliente retornando | 5902 | 050 — suspensão | sem destaque |
| Sobra, perda e sucata devolvidas | 5903 | 050 — suspensão | sem destaque |

- **IRPJ (1,2%) e CSLL (1,08%) não são indicados na NF-e.**
- Sobre R$ 230,00 de serviço: PIS R$ 1,50 + COFINS R$ 6,90 = R$ 8,40. O valor de
  R$ 9,66 que aparece na NF-e 5.911 é o "tributo aproximado" da Lei 12.741/2012,
  coisa diferente do PIS/COFINS destacado.
- **Somente CFOP interno.** A Usimac não faz industrialização interestadual hoje, e
  os CFOPs 6124/6902/6903 exigem nova validação fiscal antes do primeiro uso. O
  construtor RECUSA destinatário de outra UF em vez de inventar o CFOP: emitir com
  CFOP interno para outra UF seria nota errada, não nota a menos.
- **A UF vem da empresa** (`uf_empresa` na configuração fiscal); sem ela a nota é
  recusada. Em SP, a observação padrão é a da Portaria CAT 22/2007. Em outra UF, a nota
  só sai com o **texto legal** que a contadora daquele estado definir
  (`obs_retorno_industrializacao`): o sistema não inventa fundamento legal.
- **cBenef por empresa.** Nas UFs que exigem o código (o PR, por exemplo), a linha do
  serviço leva `cbenef_diferimento` e a de retorno leva `cbenef_suspensao`. O código
  correto é decisão da contadora.
- **Destinatário com IE.** CST 51 exige contribuinte; sem inscrição estadual a nota é
  recusada antes de existir, com a mensagem pedindo o cadastro.
- A observação da nota carrega o fundamento por escrito (é o que sustenta o ICMS zero
  no serviço), e a remessa do cliente entra em `NFref` quando a chave foi informada.

### Validação em homologação

Notas transmitidas à SEFAZ-PR de homologação pela Focus NF-e em 31/10/2026 (Tecnofer,
`tpAmb 2`). Cada rejeição virou correção no código:

| Retorno | Causa | Correção |
|---|---|---|
| 508 | CST 51 para destinatário sem IE | recusa antes de gerar a nota |
| 930 | CST 51 e CST 50 sem cBenef | `cbenef_diferimento` e `cbenef_suspensao` |
| 929 | CST 51 com ICMS diferido zero não enviava o grupo `pDif` | envia `pDif` 100 também com valor zero |
| — | a rejeição prendia o saldo do cliente | `DescartarNota` |
| — | o texto legal não chegava à nota | `InformacoesComplementares` |

Resultado: cStat 100 (5124/CST 51/`pDif` 100 e 5902/CST 50), título só do serviço,
pedido atendido, nenhum movimento em estoque próprio. Cancelamento real: cStat 135,
com saldo, pedido e título revertidos. **SP não foi transmitida**: a homologação usou a
empresa do PR, e os códigos cBenef usados foram só de teste.

Código: `internal/application/usecase/customer_material_uc/nota_de_retorno.go`.
O teste `TestNotaReproduzANF5911` reconstrói a nota real linha por linha — mesmos
CFOPs, CSTs, NCMs e os valores 230,00 + 182,60 = 412,60.

## Testes

```bash
go test ./internal/application/usecase/customer_material_uc/
TEST_DATABASE_URL=... go test -tags=integration \
  ./internal/infrastructure/repository/customer_material/
```

O teste `TestFaturamentosSimultaneosNaoEstouramOSaldo` dispara dois faturamentos
concorrentes da mesma linha: exatamente um passa. Sem o `FOR UPDATE`, os dois liam
o mesmo saldo e o material do cliente somava mais saída do que entrada.

## Faturamento

**Valor cobrado** (`fiscal/entity/valor_cobrado.go`): duplicata, conta a receber e
receita consideram só o serviço; as linhas de retorno (x902/x903) ficam de fora. Se
nada é cobrado, a nota sai sem cobrança (`tPag 90`). O total da nota continua somando
serviço e material, como a SEFAZ exige.

`POST /api/customer-material/{id}/invoice` (ADMIN) cria a nota e baixa o saldo. A
tela VBEN0100 tem a aba **Faturamento**, que propõe devolver o saldo
inteiro de cada linha e deixa escolher, por linha, entre processado (5902), sobra e
sucata (5903).

### A ordem das operações é a proteção

Não há transação entre o módulo fiscal e o razão de terceiros, então a ordem é o que
garante a consistência:

1. monta e valida as linhas — puro, sem efeito; CFOP, CST, saldo e UF conferidos aqui
2. cria a nota de saída, em **rascunho**, ainda não transmitida
3. baixa o saldo no razão, vinculado à nota (`fiscal_exit_id`), idempotente pela nota
4. a autorização, em passo separado, **confere que a baixa existe** antes de transmitir

Uma falha entre 2 e 3 deixa uma nota em rascunho sem baixa: recuperável repetindo o
faturamento (a chave de idempotência é `nota-<id>-item-<id>-<tipo>`, então não
duplica) e inofensiva porque nada foi transmitido. O inverso — nota autorizada na
SEFAZ sem o saldo baixado, material que saiu fiscalmente e continua aparecendo como
presente — é o que a ordem torna impossível.

A trava do passo 4 é `BeneficiamentoGuard` em `AuthorizeFiscalExitUseCase`, acionada
quando `source_type = BENEFICIAMENTO`. Verificado: nota sem baixa recebe **409** com
a explicação; nota com baixa passa.

### Destinatário

Resolvido no servidor pelo cadastro do cliente da remessa, com
`fiscal_uc.ResolvedorDeDestinatario` — o mesmo caminho da nota de venda (entrega →
cobrança → padrão → primeiro). A tela **não** monta endereço fiscal: fazer isso em
dois lugares faria o mesmo cliente sair com endereço diferente em cada tipo de nota.

`DadosDoDestinatario` é um **alias** do tipo do módulo fiscal, não uma cópia — assim
o resolvedor satisfaz a interface sem conversão manual, que é onde um campo novo some.

⚠️ O **código IBGE do município** fica vazio: a tabela de endereço do cliente não tem
a coluna, e a criação de nota de venda já se comporta assim. A prévia da NF-e sinaliza
a ausência antes da transmissão.

## Cancelamento da nota: o estorno

O documento 6 diz, no item de cancelamento: "o sistema deverá estornar as
movimentações relacionadas". Sem isso, cancelar a NF-e de retorno deixa o saldo
baixado — o sistema afirma que o material voltou ao cliente enquanto ele continua
no pátio. É a pior classe de erro deste módulo.

`CancelFiscalExitUseCase` chama `EstornarNotaCancelada` **depois** do cancelamento
na SEFAZ e da baixa no banco. A ordem é deliberada: cancelar na SEFAZ pode falhar,
e devolver saldo de uma nota que continua válida inventaria material no pátio.

A falha do estorno **não é engolida** — diferente do estorno de contas a receber,
que é best-effort. Se o saldo não voltar, a resposta diz que a nota foi cancelada e
o saldo não foi devolvido, e aponta a rota `POST /notes/{fiscalExitId}/reverse`
para concluir. O estorno é idempotente, então refazer é seguro.

Duas decisões:

**Marca, não apaga.** O movimento fica no razão com `reversed_at`, `reversed_by` e
`reversal_reason` (migração 000372), e a quantidade volta ao saldo pelo decremento
da coluna de consumo. "Saiu e voltou" é informação de conferência; apagar a linha
faria o razão contar uma história que não aconteceu. A tela mostra a linha riscada,
com o motivo.

**Encerramento manual não é reaberto.** O estorno recalcula a situação e reabre a
remessa que havia encerrado sozinha ao zerar o saldo. Remessa encerrada **por
pessoa** (`closed_by` preenchido) fica como está: era decisão aprovada, e desfazê-la
em silêncio trocaria um problema de saldo por um de governança. O saldo volta nos
dois casos — a quantidade física é fato, a situação é decisão.

`MovimentosDaNota` ignora movimento estornado. É o que faz a conferência antes de
autorizar na SEFAZ tratar uma baixa desfeita como baixa que não existe.

**Nota rejeitada pela SEFAZ.** A baixa acontece no faturamento, com a nota em
rascunho, antes da autorização — é essa ordem que impede autorizar nota sem baixa.
Se a SEFAZ rejeita, a nota não vira cancelada (cancelar exige nota autorizada) e o
estorno automático não dispara. Corrigir e retransmitir a MESMA nota mantém a baixa
certa, porque o id da saída não muda. Abandonar a nota exige o estorno pela rota
`POST /notes/{fiscalExitId}/reverse` — é para esse caso, além da falha no meio do
cancelamento, que a rota existe.

## Histórico e auditoria

A Usimac pede, no levantamento (documento 10): usuário, data e hora, operação
realizada, cadastro ou documento afetado, informação anterior, nova informação e
motivo. O documento 6 pede, em particular, histórico de alteração de descrição e
NCM do material do cliente. O `audit_log` que já existe não atende: ele é a trilha
do protocolo — quem chamou qual rota, com que status — e não guarda valor anterior.

A migração `000371` cria `customer_material_audit` e um trigger
`AFTER INSERT OR UPDATE OR DELETE` nas três tabelas do módulo. Cada evento guarda
o registro inteiro antes e depois (`JSONB`), a lista de campos que mudaram, o
motivo e o autor.

**Custo, medido.** Uma remessa de 200 linhas com 200 retornos gera 601 linhas de
trilha e passa de 115 ms para 211 ms — cerca de 0,16 ms por linha auditada, a
1,1 KB por linha. Para o volume do beneficiamento (notas de remessa, não
apontamento de chão de fábrica) é barato. Se um dia pesar, a alavanca é gravar só
os campos alterados no UPDATE em vez do registro inteiro — perdendo a propriedade
de responder pergunta que ninguém fez na hora de gravar.

Três decisões que não são óbvias:

**Trigger, não caso de uso.** Escrita no caso de uso deixaria de fora todo caminho
que não passa por ele: script de correção, carga inicial, `psql` do plantão. O
material é patrimônio de outra empresa e a trilha é a defesa em divergência de
saldo — não pode ter porta de serviço. Mesmo padrão do
`record_manufacturing_structural_audit()` (migração 000325).

**O autor vem de parâmetro de sessão.** As três tabelas só têm autor no insert
(`created_by`); numa alteração o banco não sabe quem foi. O repositório publica o
usuário em `venture.cm_actor` com `set_config(...,true)` — local à transação, como
o `venture.execution_actor` da migração 000359 —, e é por isso que `Bloquear`,
`Desbloquear` e `atualizarBloqueio` rodam em transação mesmo alterando uma linha
só: fora dela o parâmetro morre antes do `UPDATE`.

Na falta do parâmetro, só vale coluna que **esta** operação gravou. Cair em
`created_by` numa alteração atribuiria a mudança a quem criou o registro — pior
que não saber, porque nomeia quem não fez. Ator nulo significa alteração feita
fora do sistema.

**`updated_at` não conta como mudança.** O recálculo de situação toca
`updated_at` a cada movimento; sem esse filtro a trilha encheria de linhas sem
conteúdo e a alteração de verdade sumiria no meio.

A trilha não tem FK para empresa nem para a remessa, de propósito: é imutável
(trigger `BEFORE UPDATE OR DELETE` levanta exceção), então uma FK prenderia para
sempre o registro auditado. E ela **sobrevive** ao registro apagado — se caísse
junto, não era trilha.

Na tela, VBEN0100 mostra o histórico da remessa selecionada em
**Histórico e auditoria**: uma linha por campo alterado, com valor anterior, novo,
usuário, data e motivo.

### Mensagens ao usuário

Motivo de recusa vira texto da tela, e fica gravado na nota: só a mensagem, sem status
HTTP, código ou corpo em JSON (`recusaDaFocus`). Recusa da Focus não é rejeição da
SEFAZ (`RejeicaoError.DaFocus`), e o token recusado aponta para a VFIS0100. Falha de
rede vai para o log; a tela diz que a nota ficou aguardando e o que fazer.

### Conferências do recebimento

Emissão no futuro (um dia de folga, a data vem sem fuso), CFOP fora de 4 dígitos, NCM
fora de 8 ou 2 dígitos e chave de acesso com dígito verificador errado são recusados no
recebimento: esses dados voltam na nota de retorno, e a recusa viria só da SEFAZ.

### Pedido com nota pendente

Com nota pendente, a remessa não troca de pedido: a linha da nota aponta para o pedido
atual, e a autorização atenderia um pedido que a remessa não tem mais. Descarte a nota
antes. Faturamentos simultâneos podem tirar o mesmo número (maior + 1): quem perde tenta
de novo, e a disputa pela própria remessa diz que a nota já existe. Ainda sem proteção no
banco: dois faturamentos da mesma remessa no mesmo instante podem repetir a linha do
rascunho, porque `fiscal_exit_items` não tem índice único por sequência.

### Nota pendente

A baixa acontece ao gerar a nota, então a remessa pode ficar sem saldo com a nota ainda
em rascunho. O `pending_exit` da remessa traz a nota em `DRAFT`, `REJECTED` ou
`AGUARDANDO_AUTORIZACAO`; a tela mostra essa situação em vez de "Encerrada", permite
**Transmitir** e **Descartar**. Rejeitada por dado do cliente: descartar e faturar de
novo, porque a nota guarda o destinatário de quando foi gerada.

## Pendente

- Transmitir uma nota real de SP em homologação, com a Usimac (a validação foi no PR).
- Confirmar com a contabilidade o que fazer com o número da nota rejeitada e descartada
  (inutilização).
- Preencher o código IBGE do município no endereço do cliente (lacuna que vem de
  antes deste módulo e afeta toda emissão, não só o beneficiamento).
- Confirmar com a Usimac se o encerramento de remessa com saldo deve exigir a
  **destinação** do saldo em campo próprio. O documento 6 pede "o saldo existente,
  sua destinação, o motivo e os usuários responsáveis": hoje ficam registrados o
  saldo (nas linhas), o motivo (`close_reason`), o autor (`closed_by`) e a trilha —
  a destinação depende do texto do motivo. Transformar em campo fechado
  (devolução / descarte / retenção) é decisão de processo deles, não nossa.
