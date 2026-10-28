# Qualidade

Módulo de qualidade para a indústria metalúrgica e moveleira, sem certificação
formal (sem PPAP/APQP). Plano e decisões em `.ai/tasks/qualidade-fase2.md`.

## 1. Amostragem NBR 5426 (migration 000417)

Amostragem por atributos, plano simples, da NBR 5426 (que reproduz a ISO
2859-1). As tabelas estão em `internal/domain/quality/amostragem` e foram
conferidas contra a IS 2500-1:2000, idêntica à ISO 2859-1:1999:

| Tabela | Conteúdo | Como está no código |
|---|---|---|
| 1 | Letra-código por tamanho de lote × nível (S1–S4, I, II, III) | faixas literais |
| 2-A | Normal | diagonal (p = coluna + linha − 14) + sequência |
| 2-B | Apertada (inclui a linha S, n 3150) | diagonal (p = coluna + linha − 15) + sequência |
| 2-C | Atenuada | diagonal a partir da linha C; linhas A e B transcritas |

**Setas.** Uma seta aponta para o primeiro plano na direção dela. Quando não há
plano nessa direção (topo ou fundo da tabela), vale o primeiro plano do outro
lado, como o desenho da norma indica. A letra do plano usado pode ser diferente
da letra do lote (`letra_da_tabela`).

**100 %.** Se a amostra alcança o tamanho do lote, o lote inteiro é inspecionado
com o mesmo Ac.

**Os testes** (`amostragem_test.go`) comparam com valores lidos da imagem das
tabelas, não com a fórmula, e verificam que toda combinação letra × NQA ×
severidade resolve para um plano.

### Troca de severidade (cláusula 9)

O estado fica por **unidade de controle**: item × fornecedor (`F:<código>`) no
recebimento, item × operação (`O:<id>`) no processo, ou geral (`-`).

| De | Para | Regra |
|---|---|---|
| Normal | Apertada | 2 lotes recusados em até 5 consecutivos (9.3.1) |
| Apertada | Normal | 5 lotes consecutivos aceitos (9.3.2) |
| Normal | Atenuada | pontuação ≥ 30, produção regular e atenuada permitida no procedimento (9.3.3) |
| Atenuada | Normal | 1 lote recusado ou produção irregular (9.3.4) |
| Apertada | Suspensa | 5 recusas acumuladas em apertada (9.4) |

**Pontuação (9.3.3.2).**
- Com Ac ≥ 2: soma 3 se o lote seria aceito com o NQA um passo mais exigente; senão, zera.
- Com Ac 0 ou 1: soma 2 se o lote é aceito; senão, zera.

**Suspensão.** Para liberar, use `POST /api/quality/amostragem/estados/liberar`,
que é a ação **APROVAR** do módulo Qualidade e exige um motivo. A liberação
recomeça a inspeção em apertada. Enquanto suspenso, o resultado de lote é
recusado.

**Skip-lot.** É regra da empresa, não da norma (o skip-lot normativo é a ISO
2859-3, que não foi usada). Depois de N lotes aceitos seguidos, só 1 a cada K é
inspecionado. Uma recusa volta a inspecionar todo lote, e o skip-lot nunca vale
em apertada.

**Concorrência.** `AtualizarEstado` cria a linha se não existir, trava com
`FOR UPDATE` e grava estado e evento na mesma transação. O teste de integração
registra 5 recusas em paralelo e confere que nenhuma se perde.

**Histórico.** `quality_sampling_events` é imutável: um gatilho recusa UPDATE.

### Procedimentos

| Tipo | Campos | Uso |
|---|---|---|
| `NORMA` | nível, NQA, `permite_atenuada` | peças contáveis |
| `FIXO` | n, Ac, Re | plano próprio; aceita lote em kg/m |
| `CEM` | — | lote inteiro |

Os três tipos aceitam skip-lot (`skip_apos`, `skip_frequencia`), exceto o `CEM`.
A norma conta unidades: quantidade fracionada com `NORMA` é recusada com
mensagem.

### API (`/api/quality/amostragem`)

| Método | Rota | Uso |
|---|---|---|
| GET | `/tabelas` | NQAs, níveis e severidades válidos |
| GET | `/calcular?lote=&nivel=&nqa=&severidade=` | calculadora, sem estado |
| GET/POST | `/procedimentos` | lista / cria |
| GET/PUT | `/procedimentos/{id}` | lê / altera |
| GET | `/plano-do-lote?procedimento=&item=&chave=&quantidade=` | plano pela severidade corrente e se o skip-lot libera o lote (botão **Simular** na aba Severidade por item da VINS0110) |
| GET | `/estados?item=` | severidade por unidade de controle |
| GET | `/eventos?item=` | histórico de trocas |
| POST | `/estados/liberar` | levanta a suspensão (APROVAR) |

O registro do resultado de um lote (`RegistrarResultado`) e o pulo do skip-lot
(`PularLote`) não têm rota própria. Eles são chamados pelo lote de inspeção
(bloco 2.2), com o plano gravado no lote; não se usa o plano do momento, porque
a severidade pode ter mudado com outro lote no meio.

Tela: **VINS0110**.

## 2. Lote de inspeção (migration 000418)

Um só objeto para a inspeção de recebimento, de processo e final, no estilo do
lote de inspeção do SAP QM. Ele substitui:
- a ordem de inspeção de recebimento (`receiving_inspection_orders`, com
  resultados e análise);
- o registro de procurement do tipo `RECEIVING_INSPECTION`, com a destinação;
- o registro de qualidade da produção (`quality_records`, com as medições).

Domínio em `internal/domain/quality/lote`, caso de uso em
`internal/application/usecase/lote_uc`, tela **VINS0300**.

### Abertura

| Origem | Quem abre | Estoque retido | Plano | Chave da amostragem |
|---|---|---|---|---|
| `RECEBIMENTO` | recebimento do pedido de compra e aprovação da NF de entrada, quando o item tem roteiro de inspeção | almoxarifado de inspeção do roteiro | roteiro de recebimento (empresa da sessão) | `F:<fornecedor>` |
| `PROCESSO` | criação da OP, para cada etapa marcada com plano de inspeção ativo (na transação da OP) | não | plano de inspeção da operação | `O:<operação>` |
| `MANUAL`, `FINAL`, `DEVOLUCAO` | VINS0300 / `POST /api/quality/lotes` | se informado | roteiro do item | fornecedor ou geral |

Na abertura, o lote copia as características do plano (nome, tipo, limites,
crítica, instrumento). Assim, mudar o plano depois não muda o lote. O lote
também calcula o plano de amostragem da seção 1 pela severidade corrente e
guarda o plano usado. Quando o skip-lot libera o lote, ele abre `pulado`: a
decisão não exige resultados, e o estado registra o pulo.

A numeração é por empresa, com trava consultiva para não haver duplicata.

### Resultados

Os resultados são gravados por característica × unidade da amostra, de 1 a n.
Gravar de novo a mesma unidade corrige o resultado enquanto o lote está aberto.

A conformidade da característica VARIAVEL é decidida pelo servidor, pelos
limites copiados do plano. O `conforme` que a tela envia só vale para atributo
e para variável sem limites.

A avaliação calcula:
- **d**: unidades da amostra com alguma não conformidade (uma unidade com duas
  características ruins conta uma vez);
- **característica crítica não conforme**: recusa o lote mesmo com d ≤ Ac;
- **completa**: toda característica obrigatória tem as n unidades.

### Decisão de uso (`POST /api/quality/lotes/{id}/decisao`, permissão APROVAR)

As partes (aceita, restrita, retrabalho, rejeitada, sucata) precisam somar a
quantidade do lote.

Destinos:

| Parte | Vai para |
|---|---|
| Aceita / restrita | almoxarifado disponível (não INSPECAO/REJEICAO, com disposição) |
| Rejeitada | REJEICAO ou almoxarifado sem disposição |
| Retrabalho | qualquer almoxarifado, menos o de inspeção |
| Sucata | sem destino é baixada; com destino, é transferida |

Regras:
- Aproveitar alguma parte exige a inspeção completa. Rejeitar não exige.
- Aceitar tudo um lote recusado pela amostragem exige o tratamento `CONCESSAO`
  e um motivo. Separar a parte ruim (`SELECIONAR`) não exige.
- Toda decisão que não seja aceite simples exige motivo.
- Com a inspeção suspensa (seção 1), só é possível rejeitar ou devolver.

Tudo vai numa transação, com o lote travado (`FOR UPDATE`):
- a decisão, que é imutável (gatilho);
- as transferências de cada parte (`QUALITY_LOT`), pelo custo médio da saída;
- a troca de severidade da amostragem;
- a NC, aberta quando há parte restrita, em retrabalho, rejeitada, sucateada ou
  aceita por concessão. A severidade da NC é CRITICA se uma característica
  crítica reprovou e MENOR se só houve restrição.

Duas decisões simultâneas do mesmo lote: uma passa e a outra recebe conflito,
e o estoque é movido uma vez só (teste de integração).

O cancelamento (permissão EXCLUIR) só é aceito sem estoque retido ou depois do
estorno da entrada. Com o material ainda na inspeção, a saída é a decisão de uso.

Laudos do fornecedor: `POST /api/quality/lotes/{id}/laudos` (`quality_report_id`)
e `DELETE /api/quality/lotes/{id}/laudos/{laudo}`. Na VINS0300, o bloco **Laudos
do fornecedor** lista os laudos do vínculo item × fornecedor (do fornecedor do
lote, ou de todos os fornecedores do item quando o lote não tem fornecedor) e
vincula ou desvincula enquanto o lote está aberto.

### Efeitos em outros módulos

- **Estoque:** bloqueio de saída do almoxarifado de inspeção (ver
  `docs/dev/estoque.md`).
- **MRP:** não conta REJEICAO nem almoxarifado sem disposição.
- **OP:** a entrega final espera os lotes de processo/final da ordem estarem
  decididos e não rejeitados.
- **IQF do fornecedor:** lê os lotes decididos com `afeta_iqf`. Recusa é lote
  com parte rejeitada, sucateada ou em retrabalho, ou recusado pela amostragem.

### Compatibilidade

As rotas antigas continuam respondendo, como adaptadores sobre o lote
(`procurement_uc/lote_compat.go`, `quality_uc/lote_compat.go`):

| Rota antiga | Vira |
|---|---|
| `POST/GET /api/procurement/receiving-inspection-orders` | abre/lista lotes de recebimento (o id da ordem é o id do lote) |
| `.../{id}/results` | resultado da característica pela etapa (`step_id`) ou sequência |
| `.../{id}/analysis` | decisão de uso (conforme/restrita/retrabalho/rejeitada + destinos) |
| `.../{id}/quality-reports` | laudos do fornecedor ligados ao lote |
| `POST /api/procurement/receiving-inspections/{id}/disposition` | decisão do lote ligado ao registro de procurement (aberto na hora, se o registro é novo) |
| `/api/quality/records` | lote de processo com o plano informado; as quantidades e o resultado viram a decisão |

**Mudanças de comportamento intencionais:**
- a análise ou destinação precisa distribuir o lote inteiro e sempre move o
  estoque, na mesma transação;
- não se analisa duas vezes;
- a conformidade é a dos limites;
- o roteiro é procurado na empresa da sessão (antes, a empresa 1 estava fixa
  no código).

O desktop novo usa `/api/quality/lotes`.

**Dados antigos:** a migração copia ordens, resultados, análises, registros de
qualidade, medições, registros de procurement com destinação e vínculos de
laudo. As tabelas antigas ficam sem escrita nova.

## 3. CAPA / 8D (migration 000419)

A não conformidade é tratada por uma CAPA (ação corretiva ou preventiva).
Domínio em `internal/domain/quality/capa`, caso de uso em
`internal/application/usecase/capa_uc`, tela **VINS0500**.

**Métodos**

| Método | Etapas |
|---|---|
| `8D` | D0 preparação → D1 equipe → D2 problema (5W2H) → D3 contenção → D4 causa raiz e ponto de escape → D5 ações corretivas → D6 implementação → D7 prevenção → D8 eficácia |
| `SIMPLES` | ANALISE → ACAO → VERIFICACAO |

**Origens:**
- NC (ao menos uma NC);
- RECLAMACAO_CLIENTE (cliente) — também aberta pelo chamado de assistência
  (`POST /api/pos-venda/chamados/{code}/capa`, uma CAPA por chamado, ligada em
  `technical_assistance_calls.capa_id`; ver `vendas.md` §10.5);
- FORNECEDOR (fornecedor);
- AUDITORIA;
- INTERNA.

Várias NCs podem estar ligadas a uma CAPA.

**O que cada etapa exige para avançar** (`Capa.Pendencias`):

| Etapa | Exige |
|---|---|
| D1 | líder e ao menos 2 pessoas da empresa |
| D2 | o quê, onde, quando, quanto |
| D3 | ação de contenção, ou a justificativa de por que não é necessária |
| D4 / ANALISE | método escolhido; 5 porquês com ≥ 3 respostas, ou Ishikawa com ≥ 1 causa nos 6M; causa raiz; no 8D, também o ponto de escape |
| D5 | ao menos uma ação corretiva do ciclo |
| D6 / ACAO | todas as ações corretivas e de contenção do ciclo concluídas com evidência |
| D7 | o texto da prevenção e as ações preventivas concluídas |

**Encerramento** (`POST /capas/{id}/encerrar`, permissão APROVAR): só na
última etapa, com a descrição de como a eficácia foi verificada.
- **Eficaz:** encerra a CAPA.
- **Ineficaz:** a CAPA volta à causa raiz (D4/ANALISE) num ciclo novo, sem a
  causa anterior. As ações do ciclo antigo ficam no histórico, não mudam mais e
  não contam para o ciclo novo.

**Ações:**
- tipos: contenção, corretiva, preventiva;
- cada uma com responsável da empresa e prazo;
- concluir exige evidência.

O painel "minhas ações" lista as pendentes e as atrasadas.

**Histórico:** `quality_capa_eventos` é imutável (gatilho) e registra
abertura, alteração, troca de etapa, ações, ineficácia, encerramento e
cancelamento.

**NC:** `GET /api/quality/ncs` lista as NCs da empresa com o lote de inspeção,
o fornecedor do lote e as CAPAs ligadas. Os filtros são só abertas e sem CAPA.
A disposição do material continua na NC: ela diz o que fazer com as peças, e a
CAPA diz por que elas saíram ruins.

### API (`/api/quality/capas`)

| Método | Rota | Uso |
|---|---|---|
| GET/POST | `/` | lista (`status`, `minhas`, `atrasadas`) / abre |
| GET | `/{id}` | CAPA com pendências da etapa e histórico |
| PUT | `/{id}/analise` | grava problema, contenção, causa, prevenção e equipe (só o que vier) |
| POST | `/{id}/avancar` | conclui a etapa |
| POST | `/{id}/encerrar` | eficácia (APROVAR) |
| POST | `/{id}/cancelar` | cancela (EXCLUIR) |
| POST/PUT | `/{id}/acoes`, `/{id}/acoes/{acao}` | inclui / muda andamento |
| POST | `/{id}/ncs` | liga outra NC |
| GET | `/acoes-pendentes`, `/usuarios` | minhas ações; pessoas da empresa |

## 4. Metrologia (migration 000420)

Instrumentos de medição, calibrações e o bloqueio da medição com instrumento
inválido. Domínio em `internal/domain/quality/metrologia`, tela **VINS0600**.

**Instrumento:** código, grupo, faixa, resolução, EMA (erro máximo admissível)
e frequência de calibração em dias.
- O grupo é normalizado (maiúsculas, sem acento: "Paquímetro" = "PAQUIMETRO")
  e é o mesmo `instrument_group` da etapa do roteiro, que o lote copia para a
  característica.
- Status: ATIVO, BLOQUEADO (calibração reprovada) e BAIXADO (fora de uso).

**Calibração** (`quality_calibrations`, imutável):
- Com erro encontrado e EMA, o servidor decide: aprovada se |erro| + incerteza
  ≤ EMA. É a regra conservadora (zona de guarda): um erro no limite com
  incerteza grande não aprova.
- Sem os números, vale o resultado informado.
- Calibração externa exige o número do certificado.
- Aprovada: próxima = data + frequência, e o instrumento bloqueado volta a
  ativo.
- Reprovada: bloqueia o instrumento e devolve a **análise de impacto**, isto é,
  os lotes com resultado medido por ele desde a última calibração aprovada.
- Calibração mais antiga que a última registrada não mexe nas datas.

**Medição no lote:** o resultado guarda `instrument_id`.
- Instrumento informado precisa estar ativo, com a calibração em dia e ser do
  grupo da característica.
- Característica com grupo exige instrumento **quando a empresa já cadastrou
  instrumentos daquele grupo**. É uma implantação gradual: quem não usa
  metrologia, e as rotas antigas, continuam funcionando.

### API (`/api/quality/instrumentos`)

| Método | Rota | Uso |
|---|---|---|
| GET/POST | `/` | lista (`situacao` = VENCIDOS, A_VENCER, BLOQUEADOS; `grupo`; `dias`) / cadastra |
| GET/PUT | `/{id}` | instrumento com calibrações / altera |
| POST | `/{id}/calibracoes` | registra calibração (devolve o impacto se reprovar) |
| GET | `/{id}/impacto?desde=` | impacto sob demanda (bloco **Lotes medidos com o instrumento** da VINS0600) |
| POST | `/{id}/baixar` | baixa (EXCLUIR) |

## 5. Certificado de qualidade e documentos (migration 000421)

### Certificado ao cliente (VINS0700, `/api/quality/certificados`)

Segue o espírito do certificado de inspeção 3.1 da EN 10204. Declara o que foi
inspecionado no material vendido:
- item, quantidade, lote, corrida e certificado da usina (de `stock_lots`);
- cliente e NF de saída. O cliente da NF tem de bater com o informado.
- **por lote de inspeção:** a decisão e, por característica, a especificação,
  n, mínimo, máximo e média medidos, os não conformes e os instrumentos usados.
  O lote aceito por **concessão** aparece como tal.

**Regras**
- Só emite com lote de inspeção decidido e aproveitado do item. Os lotes vêm
  informados ou são achados pelo lote/corrida.
- A quantidade, somada à dos certificados emitidos sobre esses lotes, não pode
  passar do aprovado. Certificados parciais (entregas separadas) são aceitos; o
  cancelado libera a quantidade. A conferência é refeita sob a trava de
  numeração, então duas emissões simultâneas não certificam em dobro.
- O conteúdo é congelado na emissão (JSON). Um gatilho recusa qualquer
  alteração que não seja o cancelamento com motivo (EXCLUIR). Para corrigir,
  cancela-se e emite-se outro.
- O PDF (`/{id}/pdf`) usa o `pdfkit` do sistema.

### Documentos controlados (VINS0800, `/api/quality/documentos`)

Tipos de documento: IT, procedimento, formulário, plano de inspeção,
especificação e outro, opcionalmente ligados a um item ou operação. O desenho de
engenharia continua no cadastro de desenhos (`/api/drawings`), que já tem
revisões.

**Revisão:** RASCUNHO → VIGENTE → OBSOLETA.
- Há um rascunho e uma vigente por documento, garantido por índices únicos.
- O rascunho leva texto e/ou arquivo (até 10 MB, em base64 no JSON).
- **Aprovar** (`POST /{id}/aprovar`, permissão APROVAR): torna o rascunho
  vigente e a vigente anterior obsoleta, na mesma transação.
- **Segregação:** quem elaborou não aprova (regra no domínio e CHECK no banco).
- Revisão aprovada não muda de conteúdo (gatilho). Só a transição
  VIGENTE → OBSOLETA é aceita.
