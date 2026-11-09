# Unidades de medida e conversões (VSUP0110)

Migração `000474_catalogo_e_conversao_de_unidades`. Este documento é a referência
técnica; o manual da tela fica no venture-help.

## O problema que motivou a reescrita

- Cinco SQLs diferentes respondiam "qual o fator de X para Y deste item":
  - a compra olhava a máscara e ignorava a conversão padrão;
  - a nota de entrada olhava a padrão e ignorava a máscara;
  - a sucata da OP não olhava nenhuma das duas;
  - o saldo em segunda unidade só achava conversão que tocasse a unidade de estoque.
- Não havia encadeamento: com KG→UN e UN→MM cadastrados, KG→MM exigia mais um cadastro.
- A unidade era um enum fixo de 15 valores, mas o `uCom` da NF-e é texto livre (UND, PÇ, "Kg.").
- O XML trazia `uTrib`/`qTrib` (a mesma mercadoria em outra unidade) e o sistema ignorava.
- Não havia vigência, histórico, análise de impacto nem sinônimos.
- A edição da tela fazia POST de novo e recebia 409.

## Modelo

| Tabela | Papel |
|---|---|
| `units_of_measure` | Catálogo por empresa: código (1–10, A-Z0-9), descrição, grandeza (CONTAGEM, COMPRIMENTO, MASSA, AREA, VOLUME, TEMPO, OUTRA), código fiscal (≤ 6, o que vai no XML), casas decimais. Semeado com 29 unidades por `erp_semear_unidades_de_medida()` nas empresas existentes e, por gatilho, em toda empresa nova. |
| `unit_of_measure_aliases` | Sinônimos: texto recebido → unidade. `supplier_code` nulo vale para todos os fornecedores; preenchido vale só para aquele fornecedor e tem precedência. O texto é normalizado sem acento, pontuação ou caixa, então "Kg." = "KG". |
| `global_unit_conversions` | Conversão padrão da empresa, **só dentro da mesma grandeza** (KG↔G, M↔MM). Conversão entre grandezas (KG→UN de uma chapa) é sempre do item. |
| `item_unit_conversions` | Conversão do item (e da máscara), em **fração exata** (`from_quantity`/`to_quantity`, como UMREZ/UMREN do SAP), com vigência (`valid_from`/`valid_to`), origem (MANUAL, NOTA_FISCAL, PEDIDO_COMPRA, IMPORTACAO) e observação. |
| `item_lot_unit_conversions` | Fator medido de um lote (chapa pesada, bobina do certificado). |
| `item_unit_conversion_events` | Histórico: CRIADA, ALTERADA, NOVA_VIGENCIA, EXCLUIDA, com antes/depois e motivo. |

`items.warehouse_unit_of_measurement` e `item_structures.unit_of_measurement`
deixaram de ser enum (`unit_of_measurement_enum` foi removido). A unidade de
estoque do item tem FK composta para o catálogo da empresa
(`fk_items_warehouse_uom`).

### Gatilhos

- `trg_item_conversion_sem_sobreposicao`: duas versões ativas do mesmo fato não
  podem se sobrepor no tempo. "1 UN = 23 KG" e "1 KG = 0,0435 UN" são o mesmo fato,
  então a inversa também conta. Erro `23P01`.
- `trg_aa_*_conversion_fracao`: mantém `factor` = `to_quantity / from_quantity`.
  Quem grava só `factor` (integração, script, código antigo) recebe a fração
  `1 : factor`. O prefixo `aa` faz o gatilho rodar antes do de sobreposição.
- `instalar_conferencia_de_empresa()` cobre `unit_of_measure_aliases.supplier_code`.

## Motor de conversão (`internal/domain/item_conversion/service/grafo.go`)

Conversões são **arestas**; o fator é o **menor caminho** (BFS, até 4 passos).

- **Precedência** quando dois cadastros falam do mesmo par: LOTE > ITEM_MASCARA > ITEM > SEGUNDA_UNIDADE > EMBALAGEM > PADRAO.
- **Segunda unidade do item** (`items.warehouse_secondary_uom`/`warehouse_secondary_factor`, VENT0200) entra no grafo como aresta `SEGUNDA_UNIDADE`: sem conversão explícita na VSUP0110, compra, nota e saldo já enxergam 1 UN = N KG pelo cadastro do item.
- **Embalagem do item** (`item_gtins`: VENT0200 › Embalagens e GTIN, `package_quantity` = total na unidade de estoque) entra como aresta `EMBALAGEM`: a caixa cadastrada no item (1 CX = 12 UN) já converte a nota e o pedido em CX. A conversão explícita da VSUP0110 do mesmo par ganha; com fator diferente, a VSUP0110 avisa (a embalagem não é alterada sozinha, porque é também o GTIN da nota de saída). A cobertura lista a embalagem (papel EMBALAGEM).
- **Fração exata do começo ao fim:** 1/6000 × 6000 volta a ser 1, sem virar 0,9999999.
- `Caminho.Explicacao()` descreve os passos ("1 KG = 0,0424 M2 (item) → 1 M2 = 0,3333 UN (item)").
- `VerificarConsistencia` recusa conversão nova que contradiga um ciclo existente (tolerância 0,0001), ignorando ciclos feitos só de conversões padrão.

**Leitura única:** `internal/infrastructure/repository/uomgraph` carrega numa
consulta só as arestas do item (com a máscara e o lote) e as padrão da empresa,
vigentes na data. Usam esse pacote:

- compra (fator de compra);
- nota de entrada;
- estrutura (fator congelado na linha, migrações 354/356);
- saldo em segunda unidade;
- sucata da OP;
- serviços de terceiros.

A empresa do item confere por `EXISTS (items … enterprise_id)`.

## VSUP0110

### Conversões do item

**Create:**
- confere se as duas unidades estão ativas no catálogo;
- recusa o mesmo fato vigente, inclusive a inversa ("use Alterar");
- confere a consistência com as demais conversões;
- avisa onde o item é usado em outra unidade.

**Update (PUT /{id}):**
- Trocar a fração exige **motivo**.
- Sem `valid_from`, **corrige** a versão no lugar (o cadastro estava errado; vale para o passado).
- Com `valid_from`, abre **nova vigência** e encerra a anterior na véspera (mudou o material ou o fornecedor; o passado mantém o fator).
- Avisos devolvidos com a gravação:
  - SPED 0220: o fator não muda dentro do mês;
  - pedidos em aberto;
  - notas pendentes;
  - estruturas com fator congelado diferente.

**Delete** exige motivo e grava o evento.

**Sincronização com a segunda unidade (VENT0200).** Ao criar uma conversão
(sem máscara, vigente hoje), ou ao corrigir o fator dela, no par
estoque ↔ segunda unidade do item, o caso de uso grava o mesmo fator em
`items.warehouse_secondary_factor` (6 casas, no sentido do cadastro) e devolve
o aviso com o valor anterior. Assim o gatilho `trg_stock_movement_segunda_unidade`
(que congela o peso nominal no movimento) passa a usar o fator novo. Os
movimentos já gravados mantêm o peso que tinham. Uma nova vigência com data futura
não sincroniza; nesse caso, a VENT0200 continua com o fator atual até alguém
alterá-lo. Uma falha na sincronização vira aviso, não erro.

### Outras abas e endpoints

| Recurso | O que faz |
|---|---|
| Impacto (`GET /{id}/impact`) | Mostrado antes de alterar ou excluir. |
| Cobertura (`/item/{code}/coverage`) | Cada papel de unidade do item (estoque, compra, compra fiscal, venda, segunda unidade) mais o uso real (estrutura, vínculo com fornecedor, pedido, nota), com status BASE, OK, SEM_CONVERSAO ou FORA_DO_CATALOGO. |
| Simulador (`/simulate`) | Três partes na tela: as conversões que o item já tem (cadastradas, segunda unidade, embalagem, padrão), com quantidade editável e ⇄ para inverter; o **teste antes de gravar** (fração exata na tela, compara com o que o sistema faz hoje, avisa arredondamento de item sem fração e leva a conversão para o cadastro); e o passo a passo com data, lote e máscara. |
| Catálogo (`/api/units-of-measure`) | Unidades, sinônimos e conversões padrão. A conversão padrão só é gravada por ADMIN, e o mesmo vale para ativar ou desativar unidade. Unidade em uso não pode ser desativada. A grandeza de unidade do sistema não muda. |

### Regras que valem fora da VSUP0110

- **Cadastro do item** (criar e alterar, inclusive o cadastro a partir da nota):
  unidade de estoque, de compra, segunda unidade e unidades fiscais de compra e
  venda precisam estar ativas no catálogo. Na alteração só as que mudaram são
  conferidas, para item antigo com unidade legada continuar editável.
- **Segunda unidade na VENT0200:** se a VSUP0110 tem conversão vigente do par
  estoque ↔ segunda unidade, o fator digitado no item precisa ser o mesmo
  (tolerância 0,0001). Caso contrário, a gravação é recusada com a mensagem
  "use o mesmo fator ou altere a conversão na VSUP0110". O campo aceita vírgula
  decimal ("84,78").
- **Saldo em duas unidades:** soma o `secondary_quantity` gravado no movimento
  (peso medido ou nominal congelado) e, no movimento antigo sem peso, usa o
  fator do grafo (estoque → segunda unidade).
- **Estrutura (BOM):** a quantidade na unidade de estoque é calculada pela
  fração exata (42,5 KG com 1 UN = 85 KG dá 0,5 UN, não 0,5000000008).
- **Unidade do catálogo:**
  - não muda de grandeza enquanto tiver conversão padrão;
  - o uso que impede desativar conta itens (todos os papéis), estruturas,
    conversões do item, do lote e padrão.
- **Conversão padrão** só entre unidades ativas e da mesma grandeza. Isso vale
  também pela tela de serviços de terceiros (VTPS0100), que grava na mesma tabela
  e passou a oferecer só unidades do catálogo.
- **Pedido de compra e saída da NF** usam a conversão da máscara da linha antes
  da do item.
- **Unidade de compra do pedido passa pelo catálogo** (`UnidadeInterna.Catalogo`):
  sinônimo ou código fiscal vira o código do catálogo (BMB → BOMBONA, LT → L)
  antes de procurar a conversão. Antes, o sinônimo era tratado como outra
  unidade e o pedido pedia "cadastre a conversão BMB→BOMBONA". Unidade que o
  catálogo não reconhece é recusada como fora do catálogo.
- **Análise de impacto:** nota pendente conta só a linha na unidade da
  conversão, e pedido cancelado não conta.
- **Conversão de máscara:** a máscara informada precisa existir no item
  (`item_masks`); a VSUP0110 tem o campo Máscara no cadastro e no simulador.
- **Sucata da OP:** sem conversão no item de sucata, usa a do material de origem.
- **Telas de compra** (Tabela de Preço VSUP0120, Solicitação VSUP0300) escolhem
  a unidade do catálogo (`SelectUnidade`), não texto livre. O servidor também
  confere (`unit_of_measure_uc.ConferirNoCatalogo`): unidade fora do catálogo ou
  inativa é recusada. A solicitação valida todos os itens antes de gravar, então
  nunca fica gravada pela metade.
- **Pedido e orçamento de venda:** a unidade de venda da linha também é
  conferida no catálogo (a nota de saída usa essa unidade e converte para a de
  estoque). A tela começa em "A do item" em vez de UN fixo.
- **Unidade escolhida sempre do catálogo, com o nome** (`SelectUnidade`,
  `useUnidades`): cadastro do item (estoque, compra, venda, segunda unidade e
  níveis de embalagem — estes, unidades de contagem com até 6 letras), pedido e
  orçamento de venda, desenho (VDES0100), plano de corte e serviço do
  beneficiamento. A lista vem em ordem alfabética, mostra "GL — Galão" e é
  relida quando a janela volta ao foco (unidade criada na VSUP0110 aparece nas
  telas já abertas; a volta do foco ignora o cache de 30 s, e os campos da tela
  dividem uma leitura). Também usam o catálogo o contrato de fornecimento
  (VCON0200), o aviso de recebimento (VAVR0200) e a nota de entrada digitada
  (NovaNotaForm). Ficam em texto livre só os campos que registram o texto do
  documento de terceiros (unidade da nota importada, uTrib, código da unidade do
  fornecedor), que o catálogo traduz por sinônimo, e as unidades de medição que
  não são unidade de estoque (resultado de inspeção, característica do PDM,
  capacidade de máquina).
- **Código fiscal da unidade na NF-e e no SPED:** a autorização da NF-e troca
  a unidade do catálogo pelo `fiscal_code` em uCom e uTrib (TONELADA → T,
  MICROMETRO → UM) e recusa, antes de transmitir, unidade que nem traduzida
  caiba em 6 caracteres. A EFD faz o mesmo em 0190 (com a descrição do
  catálogo), 0200, 0220, C170 e H010. Antes a unidade ia com o código interno e
  a SEFAZ recusava as de mais de 6 caracteres. Por isso os níveis de embalagem
  do item aceitam qualquer unidade do catálogo.
  - A prévia da NF-e mostra a unidade já traduzida e aponta como pendência que
    impede (`itens.unidade`) a que não tem código fiscal que caiba.
  - Migração 000475: `fiscal_exit_items.unidade_comercial` passou de
    VARCHAR(6) para VARCHAR(10), o tamanho do código do catálogo. A NF de
    saída de um item vendido em BOMBONA não gravava. O down devolve o código
    fiscal antes de voltar a 6.
- **Frete:** o peso para o frete reconhece MG e T/TON/TN/TONELADA além de G e KG.
- **Item cadastrado a partir da nota:** a unidade sugerida é a já traduzida pelo
  catálogo (sinônimos), não mais a tabela fixa de variações.

## Nota fiscal de entrada (`fiscal_uc/entrada_unidade.go`)

Cada linha grava:
- `unidade_tributavel` e `quantidade_tributavel` (do XML);
- `unidade_normalizada` (a unidade da nota no catálogo);
- `origem_conversao`;
- `caminho_conversao` (a explicação).

**Precedência do fator.** Os três primeiros são **fixos**: nunca são recalculados.
1. `VINCULO_FORNECEDOR` — o fator do vínculo produto × fornecedor, só se a unidade do vínculo casa com a da nota.
2. `PEDIDO_COMPRA` — o fator da linha do pedido vinculada.
3. `MANUAL` — digitado na conciliação ou na resolução. Reenviar o mesmo fator mantém a origem que ele já tinha.

Os demais são **automáticos** e refeitos a cada preparação ou aprovação:
4. `MESMA_UNIDADE` / `SINONIMO` — a unidade da nota, traduzida pelo catálogo, é a de estoque.
5. `CADASTRO_ITEM` / `LOTE` / `PADRAO` — o caminho no grafo.
6. `XML_TRIBUTAVEL` — `qTrib/qCom`, quando `uTrib` é outra unidade e chega à de estoque, direto ou por conversão cadastrada.

"Automático com evidência forte": sem nenhuma das evidências acima, a linha fica
sem fator e a conferência **impede** a aprovação.

**Conferência:**
- IMPEDE: linha que movimenta estoque em unidade diferente, sem fator.
- ATENÇÃO: custo por unidade de estoque 5× maior ou menor que o da última entrada aprovada do item. É o sinal clássico de fator invertido.

**Sugestões** (`GET /api/fiscal/entries/{id}/unidades/sugestoes`). Para cada
linha em unidade diferente, os candidatos e o que cada um faz com a quantidade
e o custo:

| Fonte | Confiança |
|---|---|
| Conversão cadastrada | ALTA |
| XML tributável | ALTA |
| Pedido | ALTA |
| Última nota aprovada do mesmo fornecedor | MÉDIA |
| Peso líquido do item | MÉDIA |
| Custo da última entrada | BAIXA |

Candidato que deixa o custo fora da faixa de 5× cai para BAIXA. Sem candidatos,
a lista vem vazia (`[]`), nunca `null`.

**Resolução** (`POST …/unidades/resolver`):

| `memorizar` | Efeito |
|---|---|
| `NENHUM` | Só a linha, como MANUAL. |
| `ITEM` | Cria a conversão do item pela mesma validação da VSUP0110, com origem NOTA_FISCAL. |
| `FORNECEDOR` | Grava no vínculo produto × fornecedor. |
| `SINONIMO` | O texto da nota vira sinônimo da unidade escolhida, só para aquele fornecedor; o sinônimo geral é cadastro do catálogo. |

## Comparação com outros ERPs (fontes oficiais)

- **SAP**: unidades alternativas por material (MARM, fração UMREZ/UMREN), unidade de pedido e de saída. Temos a mesma fração exata, mais encadeamento, vigência e lote.
- **Oracle**: conversões standard, intraclass e interclass, política por item e conversão por lote. Nossa padrão = intraclass; a do item cobre a interclass; a do lote, a lot-specific.
- **TOTVS Protheus**: B1_SEGUM/B1_CONV/B1_TIPCONV — uma segunda unidade com fator multiplicar/dividir. Ficamos acima (N unidades, caminho, vigência).
- **Senior / Guardião XML**: conversão por produto + fornecedor e prompt quando falta. Equivale ao nosso vínculo mais a resolução na nota; temos também sugestões com custo e confiança.
- **Focco**: sem documentação pública encontrada.

Diferenciais:
- uso de `uTrib/qTrib` como evidência;
- sinônimos por fornecedor;
- cobertura do item;
- análise de impacto antes de alterar;
- sanidade de custo na conferência;
- memorização em três níveis a partir da nota.

## Testes

| Arquivo | O que cobre |
|---|---|
| `item_conversion/service/grafo_test.go` | Motor. |
| `item_conversion_uc/*_test.go` | Casos de uso e versionamento. |
| `fiscal_uc/entrada_unidade_test.go` | Sinônimo, encadeamento, uTrib, fator decidido, custo fora do padrão. |
| `repository/item_conversion/conversao_unidade_integration_test.go` | Duas empresas: catálogo semeado, grafo, inversa recusada, isolamento de conversão e sinônimo. |
| `repository/item_conversion/segunda_unidade_integration_test.go` | Aresta da segunda unidade, saldo em duas unidades, `FatorDaConversaoDoItem`, sincronização e isolamento. |
| `item_uc/segunda_unidade_test.go`, `item_conversion_uc/segunda_unidade_test.go` | Fator da VENT0200 contra a VSUP0110; seletividade da sincronização. |
| `unit_of_measure_uc/conferir_test.go`, `structure_uc/unidade_de_estoque_test.go` | Unidade no catálogo (preço/solicitação); quantidade exata na estrutura. |
| `purchase_order_uc/unidade_interna_test.go` | Sinônimo da unidade de compra traduzido (BMB → BOMBONA); unidade fora do catálogo recusada. |
| `fiscal_uc/unidade_fiscal_nfe_test.go`, `fiscal/sped/unidade_fiscal_test.go` | Código fiscal da unidade em uCom/uTrib e na EFD. |
| `fiscal/frete/frete_test.go` | Peso em KG a partir de MG e TONELADA. |
