# Changelog

Todas as mudanças distribuíveis do VentureERP são registradas neste arquivo. Commits comuns não
constituem uma atualização; somente uma tag SemVer criada por `make release VERSION=x.y.z` inicia o
pipeline de release.

## Unreleased

- fix(compras): a linha comprada em outra unidade guarda a quantidade e o preço
  internos com 6 e 10 casas (migração 408) — 100 KG × 0,15530965 entravam no
  estoque como 15,5310 M em vez de 15,530965
- fix(estoque): o custo médio gravado é sempre o valor (em 4 casas) ÷ quantidade;
  antes era calculado sobre o valor ainda sem arredondar e "mudava" na sétima
  casa quando o próximo movimento o relia pelos totais
- docs: exemplo da NF-e de saída com `origem_mercadoria` "0" e `tipo_pessoa` "J"
  (os valores por extenso estouravam a coluna)

## [v1.7.1] — 2026-10-19

- fix(estoque): quantidade e custo em decimal de ponta a ponta; kardex no fuso da empresa (`d80e7b13`)

- fix(estoque): quantidade e custo do estoque em decimal de ponta a ponta (antes
  `float64`): o saldo "zerado" deixava resíduo de 0,0000000001 que o bloqueio de
  saldo negativo acusava, e a soma de frações derivava. Migração 407: custo unitário
  com 10 casas (R$ 0,0666666667/mm em vez de 0,0667, que encarecia o corte em 0,05%)
- fix(estoque): kardex e posição em data contam o dia no fuso da empresa — o
  lançamento depois das 21:00 de Brasília sumia de "até hoje"
- fix(estoque): reserva de quantidade zero ou negativa é recusada (corrompia o
  reservado do saldo)

## [v1.7.0] — 2026-10-17

- fix(nf-e): prévia aponta item sem almoxarifado para a baixa; código sqlc regenerado com as migrações 405/406 (`f16013b3`)
- docs(estoque): regras do lançamento, inventário, transferência, estorno, kardex, reposição e requisição (`2500b99a`)
- feat(estoque): tipos de movimento com centro de custo e lançamento manual; respostas no DTO do movimento (`e4da1951`)
- feat(estoque): transferência entre almoxarifados, estorno, kardex, posição em data, ATP por situação, reposição, análise, requisição ao almoxarifado e tipo de movimento no lançamento (`9ca8f726`)
- wip(estoque): migração 406 (estorno, tipo de movimento no lançamento, requisição ao almoxarifado) — código ainda por fazer (`0906cdec`)
- wip(estoque): alteração de almoxarifado e permissão de saldo negativo (`b49abf44`)
- wip(estoque): inventário que ajusta o saldo, saldo negativo bloqueado, custo único por almoxarifado, período fechado e ajuste da contagem cíclica (`95308e41`)
- wip(estoque): conversão de unidade na entrada da NF, no pedido de compra e na baixa da NF-e (`79be2c19`)
- feat(cadastros): alterar e excluir plano de contas e centro de custo no momento certo (`56ad56c7`)

- fix(estoque): a nota de entrada passa a converter a unidade da nota para a de estoque
  pelo cadastro de conversões do item (ou a global); sem conversão a nota é barrada.
  Caso Tecnofer: 115,8975 KG de tubo entravam como 115,8975 M ao preço do quilo
- fix(compras): pedido de compra grava quantidade e preço na unidade de estoque do
  cadastro em todas as origens; unidade de compra sem conversão é recusada (antes o
  erro era engolido)
- fix(fiscal): a baixa da NF-e converte a unidade de venda para a de estoque; a prévia
  barra a transmissão sem conversão ou sem saldo
- fix(estoque): inventário físico corrige o saldo — foto na abertura, contagem e
  recontagem, ajustes lançados no fechamento (antes não mexia em nada); contagem
  cíclica aprovada também ajusta
- fix(estoque): custo médio único por item e almoxarifado (era por endereço); saída
  sem saldo bloqueada, salvo no almoxarifado que permite (migração 405); período de
  estoque fechado vale para todo lançamento; transferência deixa de contar como
  consumo (média e curva ABC); ATP não promete saldo em inspeção/rejeição
- feat(estoque): transferência entre almoxarifados numa operação, estorno de
  movimento, kardex, posição em data com 2ª unidade, reposição por ponto de pedido
  com geração de requisição de compra, análise (giro, cobertura, parados), lotes a
  vencer, requisição ao almoxarifado por centro de custo e tipo de movimento
  configurado no lançamento (migração 406)
- feat(almoxarifado): `PUT /api/warehouse/{code}` (alterar almoxarifado e a permissão
  de saldo negativo)
- fix(estoque): tipos de movimento aceitam as flags em snake_case (antes ignoradas)

- feat(cadastros): plano de contas e centro de custo (financeiro VFIN0120/VFIN0130 e
  industrial VCTB0102) podem ser alterados, desativados e excluídos no momento certo:
  - em uso (títulos, notas, rateios, máquinas, estruturas, ordens, filhos…), só a
    descrição e a situação mudam (no industrial, também a data final); código, tipo,
    natureza e pai respondem 409 listando onde o cadastro é usado; excluir também
  - sem uso, a estrutura muda com conferência do pai (existe, ativo, sem ciclo)
  - desativar com filho ativo e reativar com pai inativo são recusados
  - excluir é só do ADMIN; a listagem de cadastro traz os inativos para reativar,
    e as escolhas continuam só com ativos
  - rotas: `GET …/cadastro`, `GET …/{id}/uso`, `PUT …/{id}`, `DELETE …/{id}`
- fix(cadastros): conta-pai e centro-pai inexistentes passam a ser recusados no
  cadastro (422) — deixavam galhos soltos na árvore
- fix(fiscal): código IBGE do município da empresa é conferido ao salvar (7 dígitos,
  da UF da empresa) — código de outra UF saía errado no CT-e, na NFS-e e no SPED
- fix(compras): `GET /api/procurement/receiving-inspection-orders/{id}` existia só no
  front (VINS0400 recebia 404)

## [v1.6.3] — 2026-10-16

- fix(nf-e): validado na SEFAZ do PR em homologação; cBenef, rejeição e demora da SEFAZ (`091ab3ba`)

- fix(nf-e): validado contra a SEFAZ do PR em homologação (autorização, efeitos,
  cancelamento e estorno). Correções encontradas no caminho:
  - venda interna a contribuinte sem diferimento configurado saía CST 51 e era
    rejeitada (930, código de benefício fiscal); agora sai CST 00
  - o percentual de diferimento era dividido por 100 duas vezes (ICMS diferido cem
    vezes menor); a configuração guarda fração e o motor passa a usá-la assim
  - no CST 51 o ICMS destacado é o da operação menos o diferido (`icms_valor_operacao`),
    como a SEFAZ confere (rejeição 353)
  - código de benefício fiscal (cBenef) por item: vem do parâmetro de ICMS/IPI do item
    ou do NCM (VFIS0320), vai para a Focus, e a prévia barra a emissão quando o CST
    exige e o código falta (PR, RS, RJ, SC, GO, DF) — migração 000404
  - rejeição da SEFAZ responde 422 com o motivo (era 502) e o motivo fica gravado na
    nota; a consulta de status o mostra
  - a referência da Focus é gravada antes do envio: nota sem resposta (SEFAZ demorando,
    queda) fica "aguardando autorização" e é concluída pela consulta — antes era
    marcada rejeitada e, autorizada depois, ficava sem título, baixa e pedido
  - a consulta que descobre a autorização grava número, chave, protocolo e DANFE
  - reenviar nota que aguarda resposta é recusado (evita duas notas na SEFAZ)
- feat(itens): o campo de código do item aceita também a descrição — um item só abre
  direto; vários listam os códigos para escolher
- fix(faturamento): as mensagens citam o código do item (RN-01001), não a chave interna

## [v1.6.2] — 2026-10-14

- fix: erro conhecido sai do log de erro e mensagem de código de item legível (`ac12a14c`)

- fix(log): erro conhecido (validação, não encontrado, duplicidade) não é mais registrado
  como "internal server error" — só a falha de verdade vai para o log de erro
- fix(itens): a mensagem de código de item inválido vem acentuada, mostra o que foi
  digitado e orienta a buscar pela descrição

## [v1.6.1] — 2026-10-14

- fix: filtros de lista, conferência de empresa por código, crédito em decimal e fuso da capacidade (`3436d5f5`)
- fix(métricas): coleta de métricas de notificação usava coluna inexistente (`3a01e5b1`)

- feat(isolamento): o banco também confere as colunas de código sem FK
  (`sales_order_code`, `customer_code`, `supplier_code`… — 87 vínculos): o código
  precisa existir na empresa do registro. Código repetido entre empresas (cada uma
  com a sua cópia do cliente) é aceito; o da outra empresa, não (migração 000403)
- fix(validação): filtro numérico ou de data mal digitado nas listagens (cerca de 60
  rotas: compras, comissões, auditoria, materiais de cliente, contabilidade,
  representantes, orçamentos, serviços de terceiros, estoque, manutenção…) passa a
  responder 422 dizendo o filtro e o valor, em vez de listar tudo sem o filtro. Filtro
  em branco enviado como `undefined`/`null` continua valendo como "sem filtro".
  Código inválido no endereço (reprogramação de entrega, centro de custo, firmar
  ordem planejada, prioridade) também é recusado — antes virava 0
- fix(crédito): a conferência do limite de crédito soma e compara em decimal; em
  float um pedido que fechava exatamente no limite podia ser bloqueado
- fix(capacidade): com o banco fora do fuso da empresa, o CRP e o agendamento finito
  das máquinas liam as paradas em horários diferentes e o plano saía com mais horas
  que a capacidade do dia. Os dois usam o fuso da empresa
- fix(métricas): as métricas de notificação nunca eram coletadas — a consulta somava
  uma coluna inexistente (`message_count`; o nome é `sent_count`) e o erro era ignorado

## [v1.6.0] — 2026-10-14

- feat: isolamento por empresa, efeitos da NF-e como pendência e erros que não somem mais (`5fab9265`)

- feat(nf-e): os efeitos da NF-e autorizada (títulos a receber, baixa de estoque,
  consumo das reservas e faturamento do pedido) viram uma PENDÊNCIA registrada antes
  de rodar e aplicada numa transação só — ou entram todos, ou nenhum. Falhando, a
  pendência guarda o motivo e é reprocessada sozinha (1, 2, 4… até 60 min) ou pelo botão
  "Reprocessar agora" da tela de NF-e. Antes, uma falha depois da SEFAZ deixava a nota
  autorizada sem título, sem baixa ou com o pedido aberto, e ninguém sabia. O mesmo vale
  para o cancelamento. Rotas: `GET /api/fiscal/exits/{id}/pendencias` e
  `POST /api/fiscal/exits/{id}/reprocessar-efeitos`
- feat(nf-e): a baixa da nota usa o depósito da linha do pedido e, sem ele, o depósito
  padrão do item; sem nenhum dos dois a autorização é recusada ANTES de transmitir
- feat(isolamento): o banco recusa gravar referência a registro de outra empresa
  (migração 000402). Cada vínculo por FK, mais `item_code`, é conferido por um gatilho
  gerado do catálogo — 253 vínculos em 153 tabelas. Ficam de fora, por serem cruzados de
  propósito, o fluxo interfábricas (MRP, planejamento, ordens planejadas) e as tabelas
  de vínculo `*_enterprises`. Migration que criar tabela ou FK chama
  `SELECT instalar_conferencia_de_empresa();` (um teste de integração cobra)
- fix(vendas): confirmar o pedido é tudo ou nada — status, demanda do MRP e reserva de
  estoque entram juntos. Falha na conferência de crédito não confirma e diz o motivo
  (antes o pedido seguia sem a análise)
- fix(expedição): separar, expedir e cancelar romaneio gravam a situação e as reservas
  juntas; criar romaneio e carga grava o evento e a caixa de expedição juntos
- fix(produção): apontamento com baixa automática dos componentes é tudo ou nada; a
  falha na baixa não deixa mais o apontamento gravado sem o consumo
- feat(produção): OF encerrada com falha na apuração do custo fica marcada "custo não
  apurado", com o motivo, até "Apurar custo" dar certo (antes ficava sem custo, calada)
- fix(custos): o rateio de indiretos por centro de trabalho nunca entrava — a consulta
  dos centros do roteiro usava uma coluna inexistente e o erro era engolido. Agora usa o
  centro padrão da operação e erro de leitura aparece
- fix(erros): leituras que falhavam em silêncio e seguiam com valor vazio ou padrão
  passam a devolver o erro (ou ir para o log quando a operação principal já aconteceu):
  MRP (registradas no log do cálculo), configurador (regras e fórmulas avaliadas sem a
  variável), APS/Gantt, devolução de compra, romaneio impresso sem destinatário ou
  preços, defaults de compra do fornecedor, parâmetros 25/45/66, numeração de ordem
  planejada e de OF (caía em "1"), conciliação automática do extrato, histórico do
  pedido de venda, notificações e idempotência
- fix(validação): data mal digitada não grava mais 0001-01-01 em silêncio — títulos a
  pagar/receber, nota de saída, demanda independente, ordem planejada, rateio, centro de
  custo, reserva e movimento de estoque, agenda da máquina, atendimento ao consumidor,
  remessa CNAB, previsão de entrega e relatórios financeiros recusam com a mensagem do
  campo (aceitam AAAA-MM-DD e DD/MM/AAAA)
- fix(validação): corpo opcional com JSON quebrado (motivo do cancelamento/bloqueio,
  atender pedido, expedir, recalcular consumo médio, curva ABC) era ignorado; agora é 400.
  Filtros de data, transportadora e paginação das listas de romaneio e carga também
- fix(aps): reprogramar a operação informa quando a conferência de capacidade falhou
  (`capacity_check_error`) em vez de devolver "sem conflitos"
- fix(clientes): cada empresa tem o seu cadastro de clientes. O cadastro não tinha
  empresa nenhuma e o mesmo cliente aparecia (e podia ser alterado, bloqueado ou ter
  endereço apagado) em todas as empresas do banco. Na atualização, cada empresa que já
  usava o cliente recebe o seu cadastro com o mesmo código; cliente nunca usado vai para
  todas as empresas
- fix(contas a receber): o título gerado pela NF-e guardava o id interno do cliente em vez
  do código — sumia do filtro por cliente e da análise de crédito. Os títulos existentes
  são corrigidos na atualização
- fix(clientes): alterar endereço e contato do cliente passa a gravar de fato
- fix(isolamento): uma empresa não lê nem altera mais os dados de outra em:
  registros de suprimentos, inspeção de recebimento, avisos de recebimento,
  divergências, homologação e IQF de fornecedor, mensagens EDI, processos de
  importação, contratos, histórico de compras, telefones e e-mails de contato do
  fornecedor, ficha de ferramentas e reprogramação de entrega. Na atualização, os
  registros de suprimentos existentes vão para a empresa do pedido/fornecedor de origem
- fix(empresas): a lista de empresas mostra só as empresas do usuário (mostrava todas
  as do banco), e criar empresa exige perfil de administrador; quem cria passa a
  pertencer a ela
- fix(reprogramação de entrega): a prévia e a reprogramação em lote não achavam o
  pedido fora da empresa 1, e linha já atendida aparecia como reprogramável
- fix(suprimentos): "gerar itens do fornecedor pelo histórico" falhava sempre
- fix(isolamento): contabilidade por empresa — plano de contas, contas e
  demonstrativos eram compartilhados (uma empresa via, alterava e lançava no plano
  da outra). Na atualização, cada empresa recebe a sua cópia dos planos que já usava,
  com parâmetros de contabilização, lançamentos, plano financeiro e contas bancárias
  apontando para ela
- fix(isolamento): NFS-e por empresa — pelo id, uma empresa lia, autorizava e
  cancelava (com o próprio token Focus) a nota de serviço da outra
- fix(isolamento): parâmetros ICMS/IPI, redução/substituição de ICMS, linhas e
  totalizadores da apuração de ICMS, apuração do Simples Nacional, cenários fiscais,
  rateio de despesas indiretas e adiantamentos passam a ser da empresa. Apurações
  vão para a empresa fiscal; parametrização é copiada para cada empresa (regra que
  cita item, cliente ou fornecedor vai só para a empresa dona dele)
- fix(fiscal): cadastrar redução/substituição de ICMS falhava sempre (a gravação
  enviava menos valores que colunas)
- fix(fiscal): período mal informado na restituição de ICMS-ST respondia "erro
  interno" em vez de pedir o formato ano-mês
- fix(isolamento): representantes por empresa — o cadastro era compartilhado e,
  pelo código, outra empresa se vinculava ao representante e incluía telefone,
  e-mail, contato, região e contabilização nele. Representante usado por mais de
  uma empresa vira um por empresa (código novo para as demais), com pedidos,
  orçamentos, comissões e venda recorrente de cada empresa apontando para o dela
- fix(isolamento): metas de venda por empresa — períodos, metas e saldos eram do
  banco inteiro, e o relatório somava como realizado os pedidos de todas as empresas
- fix(isolamento): estrutura de produto (BOM) é da empresa do item pai — antes
  outra empresa criava componente, lia, alterava e removia a estrutura alheia; o
  mesmo vale para histórico da estrutura, "onde é usado", máscaras, conversões de
  unidade, calendário de promessa e custos do item na formação de preço
- fix(isolamento): ferramentas e séries, responsáveis de garantia, reservas de
  tanque e parâmetros da promessa de entrega passam a ser por empresa; atendimento
  de SAC e de assistência técnica não aceita mais consumidor, cliente, responsável,
  pedido ou ordem de produção de outra empresa
- fix(ferramentas): a ficha aceitava atribuir série de ferramenta de outra empresa
- fix(reprogramação/assistência técnica/estrutura): erros ao carregar itens,
  notas de retorno e clientes do acompanhamento eram ignorados em silêncio
- fix(isolamento): cadastros de apoio por empresa — regiões, segmentos, tipos de
  cliente/fornecedor/contato, portadores e grupos, tipos de nota, tributação, TES,
  grupos de UF, tipos de movimento de estoque, tipos de representante, tipos de
  chamado e fontes de conhecimento do SAC, grupos e motivos de defeito. Na
  atualização, cada empresa recebe a sua cópia com o MESMO código (tipos de
  representante, de chamado e defeitos ganham código novo nas demais empresas), e
  clientes, fornecedores, políticas, parâmetros fiscais e documentos passam a
  apontar para o catálogo da própria empresa. O banco recusa vínculo com catálogo
  de outra empresa
- fix(fiscal): a lista de cartas de correção mostrava as de nota de outra empresa
- fix(clientes): alterar a tributação não gravava nada
- nota: empresa criada a partir desta versão começa com os cadastros de apoio vazios
- fix(compras): a segunda empresa do banco não conseguia criar cotação nem
  requisição de compra — a numeração era por empresa, mas o código é único no
  banco inteiro, e o código gerado já existia na outra empresa
- fix(isolamento): itens, fornecedores convidados e preços da cotação de compra e
  itens da requisição eram lidos e gravados pelo id, sem olhar a empresa — outra
  empresa incluía item, convidava fornecedor, gravava e selecionava preço na
  cotação alheia e levava item de requisição alheia para a própria cotação. A
  cotação também recusa fornecedor de outra empresa
- fix(isolamento): o vínculo fornecedor×empresa era alterado e excluído pelo id
  sem conferir a empresa, e aceitava tabela de preço de compra de outra empresa
- fix(isolamento): o vínculo item×fornecedor aceitava item ou fornecedor de outra
  empresa; contagem cíclica aceitava almoxarifado de outra empresa
- fix(fiscal): a sincronização das NF-e recebidas (DF-e) falhava em toda nota ao
  gravá-la ("inconsistent types deduced for parameter $2") — nenhuma nota recebida
  era importada, nem pelo agendamento nem pelo botão
- fix(compras): gravar preço da cotação ignorava a falha ao marcá-la como cotada,
  e a importação de NF-e de compra ignorava a falha ao gravar o item da nota (o
  estoque entrava sem o item na nota e o pedido recebia baixa) — agora o motivo
  volta na resposta
- fix(planejamento): empresa criada depois da v1.0 não tinha parâmetros de
  planejamento — a tela vinha vazia e nada podia ser alterado. Cada empresa
  recebe o catálogo de parâmetros (também as criadas daqui em diante), com os
  valores que o MRP já usava como padrão: o cálculo não muda
- fix(produção): a permissão de alterar quantidade e datas da OF lia os
  parâmetros 10 e 14 do catálogo (itens fantasmas e cálculo do MPS), sem relação
  com a OF; agora lê só os próprios parâmetros
- fix(compras): falha ao criar a cotação (item de requisição inválido, fornecedor
  de outra empresa) deixava uma cotação vazia gravada; agora nada é gravado
- fix(mrp): a programação das máquinas esperava os componentes de TODAS as ordens
  do item pai — a ordem de amanhã aguardava a compra do mês seguinte e todas as
  ordens de fabricação saíam "atrasadas" na mesma data. Agora cada ordem espera
  só os componentes pedidos até o início dela
- fix(mrp): número inicial de ordem já em uso (ordem planejada ativa ou sugestão
  de outro plano) fazia o cálculo perder as sugestões e terminar "com erros"
  sem nenhuma ordem; agora a sugestão pega o próximo número livre
- feat(mrp): a lista de sugestões informa a ordem planejada gerada ao firmar
  (`planned_order_code`)
- fix(aps): o turno do calendário da máquina era lido como UTC — o APS
  programava três horas antes do expediente (turno das 07:00 começava às 04:00
  em Brasília). A janela usa o fuso da empresa, o mesmo do MRP
- fix(fiscal): doze cadastros da apuração de ICMS (resumo e notas do resumo,
  redução/substituição, ressarcimento de ST, notas especiais e itens, Simples
  Nacional, linhas, códigos de ajuste, motivos DAPI) eram decodificados direto na
  entidade sem nome de campo: tudo que tinha nome composto (`cfop_id`,
  `icms_base`, `issue_date`…) era descartado em silêncio. Aceitam data só com o
  dia nos campos de data, e o erro do banco não aparece mais cru para o usuário
- fix(fiscal): regra de redução/substituição de ICMS nunca gravava — o alvo da
  redução não informado ia como texto vazio para a coluna enum; vale BASE
- fix(fiscal): item da nota especial de ajuste não podia ser incluído — o código
  do item (texto do documento) era traduzido para a chave interna numérica
- fix(compras): gerar vínculos item × fornecedor pelo histórico (FFOR0204)
  falhava sempre — o ator ia como texto para a coluna uuid
- fix(erros): cerca de 600 validações de regra de negócio (cadastro, planejamento,
  produção, fiscal, financeiro…) voltavam como "erro interno" (500); agora voltam
  como erro de validação (422), registro não encontrado (404) ou conflito (409),
  com a mensagem em português
- fix(máquinas): máquina sem calendário trabalhava a partir da meia-noite no MRP e
  no APS; o turno padrão passa a começar às 08:00 (até completar as horas
  disponíveis por dia)
- fix(planejamento): a execução automática (janela noturna) rodava sem a sessão
  da empresa e falhava sempre; agora roda em nome de quem salvou a configuração.
  Da segunda rodada em diante também falhava (contagem de itens alterados lia
  uma coluna inexistente), e a hora da janela era a do banco, não a da empresa
- feat(planejamento): configuração da execução automática por empresa
  (`GET/PUT /api/planning/auto-run`), "rodar agora" (`POST
  /api/planning/auto-run/run-now`) e histórico de execuções (`GET
  /api/planning/runs`)
- feat(mrp): monitor de cobertura de materiais por plano
  (`GET /api/mrp-reports/coverage`): dias de cobertura do estoque e com ordens
  firmes, falta sem planejada e farol (crítico/atenção/coberto), filtrável por
  planejador e tipo de item
- fix(mrp): mensagens de exceção eram geradas sobre o suprimento já consumido
  pelo cálculo e comparadas com a necessidade líquida e com a primeira data do
  item: ordem firme atrasada só gerava mensagem se chegasse depois da última
  necessidade, o excesso quase nunca aparecia, e uma ordem que cobria a
  necessidade de novembro era "atrasada" em relação à de outubro. Agora cada
  ordem firme é comparada com a necessidade que de fato cobre (depois do estoque)
- feat(mrp): ação direto na exceção (`POST
  /api/mrp-calculation/exceptions/{code}/handle`): reprogramar a ordem planejada
  para a data sugerida, cancelar a ordem sem necessidade ou marcar como tratada
  com observação. A exceção guarda a data sugerida, quem tratou e como
- fix(ordens planejadas): voltar uma ordem para "planejada" podia ser recusado
  por movimento de OUTRA ordem — a checagem ligava a OF pelo código da ordem
  planejada em vez do id
- feat(mrp): rastreamento (pegging) gravado a cada cálculo — de cada pedido ou
  demanda até as compras dos componentes, e de uma ordem até quem depende dela
  (`/api/mrp-calculation/pegging/{plano}/roots|down|up`)
- feat(planejamento): cenários de simulação (`/api/planning/scenarios`) —
  ajustes de demanda, estoque, prazo e capacidade de máquina calculados (MRP +
  CRP) num plano oculto, comparados com o plano real; só a demanda pode ser
  aplicada ao plano real
- fix(mrp): operação externa (ordem de SERVIÇO) era tratada como suprimento do
  item — a necessidade seguinte do item saía menor — e explodia a estrutura de
  novo, pedindo os componentes em dobro
- fix(mrp): o perfil do item gravava "ordens planejadas" negativas quando o
  estoque sobrava (a necessidade líquida negativa); agora grava zero
- fix(mrp): duas pessoas tratando a mesma exceção ao mesmo tempo gravavam as
  duas (a segunda sobrescrevia); agora a segunda recebe conflito
- fix(api): corpo JSON quebrado, incompleto ou com tipo errado respondia com o
  texto técnico do decodificador ("invalid character…", "cannot unmarshal…");
  agora a mensagem é em português
- fix(crp): planos de cenário de simulação apareciam na lista de planos do CRP
- fix(produção): faltas de material de OF inexistente ou de outra empresa
  respondiam "sem lista" em vez de "não encontrada" (404)
- fix(planejamento): o plano oculto do cenário usa uma faixa reservada de
  códigos — antes podia ocupar o código que o usuário escolheria para um plano
  novo, que recebia "já existe" para um plano invisível
- feat(produção): faltas de material da OF antes de liberar
  (`GET /api/production-order/{id}/material-availability` e resumo
  `GET /api/production-order/material-availability?ids=`): saldo livre do
  depósito menos o que as OFs da frente vão consumir
- feat(mrp): atendimento do pedido de venda pelo MRP
  (`GET /api/mrp-calculation/pegging/sales-order/{order_code}`): por plano ativo,
  as linhas do pedido com estoque, ordens, sugestões, falta e a cadeia até as compras
- feat(mrp): o rastreamento marca o elo atrasado — sugestão que termina depois da
  necessidade na programação das máquinas ou que já devia ter começado — com o
  motivo por extenso (`late`, `late_reason`; na raiz, `chain_late`)
- feat(mrp): a exceção de ordem planejada informa os pedidos/demandas que dependem
  dela (`affected_demands`)
- feat(planejamento): a comparação de cenário conta os pedidos/demandas com cadeia
  atrasada (`roots_late`)
- fix(mrp): a mensagem de exceção citava o código interno do item ("Item 9003: ordem
  firme de 80.00 un.…"); agora cita o código de negócio e a quantidade como se lê
  ("RODA-200: ordem firme de 80 un.…", "12,5 un.")
- fix(mrp): no rastreamento, a sugestão que a máquina só termina depois da
  necessidade passa a ter como data o fim programado — "pronto em" (`latest_supply`)
  e o motivo do atraso mostram o dia em que a cadeia fica pronta de fato
- fix(isolamento): a consulta de carga por centro do CRP (sem uso em rota hoje) lia
  planos de todas as empresas e os de cenário; agora só os planos reais da empresa
- fix(isolamento): roteiro — pelo id, outra empresa alterava e inativava etapa,
  incluía, alterava, lia, tornava principal e removia recurso alternativo, e ligava
  ou apagava precedência de roteiro alheio; a própria empresa podia apontar
  operação ou centro de trabalho de outra. Agora tudo filtra pela empresa do
  roteiro e responde "não encontrado"
- fix(isolamento): o vencimento do fornecedor aceitava condição de pagamento de
  outra empresa
- fix(fiscal): CFOP e dispositivo legal valem para todas as empresas do banco; incluir
  e alterar passa a exigir o perfil ADMIN (USER consulta normalmente)
- fix(roteiro): trocar o recurso principal da etapa é tudo ou nada — uma falha no meio
  deixava a etapa sem recurso principal
- fix(validação): data mal digitada era descartada em silêncio e o registro gravado sem
  ela (pedido e item de venda, orçamento, OF, ordem planejada, venda recorrente,
  desenho, SAC, assistência técnica, atributo de exportação). Agora responde 422 dizendo
  o campo e os formatos aceitos (AAAA-MM-DD ou DD/MM/AAAA)
- fix(validação): filtro de data ou número mal digitado nas listas de orçamentos,
  chamados de assistência técnica e acompanhamento de representantes era ignorado (a
  lista vinha inteira); agora responde 422
- fix(mrp): falha ao ler as operações externas do item era ignorada e o cálculo deixava
  de sugerir a ordem de serviço; agora o cálculo falha com o motivo
- fix(desenhos): falha ao ler revisões e distribuições era ignorada (a resposta vinha
  incompleta) e falha de banco virava "não encontrado"
- feat(auditoria): `scripts/audit-referencias-entre-empresas.sh` lista vínculos em que
  um registro aponta para cadastro de outra empresa (somente leitura; serve em produção)
- fix(isolamento): qualidade por empresa — planos de inspeção, características,
  registros e não conformidades eram lidos e alterados por id sem olhar a empresa.
  O filtro é pela empresa do item; alterar registro inexistente ou de outra
  empresa responde "não encontrado"
- fix(isolamento): o quadro mensal (Gantt) do APS mostrava ordens, dependências e
  carga de máquina de todas as empresas, e a carga somava os planos de cenário de
  simulação; o sequenciamento do APS também lia as ordens abertas de todas as empresas
- fix(gantt): a barra da ordem mostra o código e o nome do item
- fix(qualidade): característica com tolerância mínima maior que a máxima, sem nome
  ou de plano inexistente era aceita
- fix(fiscal): CFOP com utilização, indicador de operação ou tipo de utilização
  inválido era gravado (e no PUT, campo vazio virava valor inválido); agora a
  mensagem lista os valores aceitos
- fix(roteiro): alterar uma operação sem informar a situação gravava situação vazia;
  agora mantém a atual, e situação inválida é recusada
- fix(mrp): envio de exceções por webhook/e-mail valida o plano, o endereço e os
  e-mails antes de enviar; endereço recusado e falha de envio respondem em português
- fix(segurança): o webhook recusa também as faixas reservadas (CGNAT 100.64/10,
  redes de documentação e de teste, 0/8, 240/4, NAT64) além das internas que já recusava
- fix(api): erros de validação, "não encontrado", duplicidade e vínculo inexistente
  vindos dos handlers respondiam o texto técnico do banco ou do Go; passam pela
  mesma tradução para português dos demais endpoints. Corpo com campo desconhecido
  nas operações estritas responde qual campo não é aceito

## [v1.5.1] — 2026-10-07

- feat(compras): pedido de compra por empresa, com documento e acompanhamento (`0136247`)
- test(itens): resolução por empresa cria as próprias empresas (`f7696cb`)

- fix(entrada de NF-e): relacionar a linha da nota a um item cadastrado dava "o
  item não existe" quando o código do item na tela era diferente do código
  interno. A tela passa a enviar o código do item como ele aparece no cadastro
  (o mesmo vale para o lançamento manual de nota)
- fix(pedido de compra): incluir item sem escolher o almoxarifado dava "depósito é
  obrigatório" — e na VSUP0200 nem havia o campo. O campo existe nas duas telas e,
  em branco, vale o almoxarifado de suprimentos do cadastro do item
- fix(pedido de compra): os totais do pedido passam a ser calculados (estavam
  sempre em zero), e por isso a alçada de valores volta a funcionar: o pedido
  acima do limite fica aguardando autorização e a autorização o encontra
- fix(pedido de compra): a alteração da capa grava frete, transportadora,
  adiantamento e importação, não apaga mais a origem e não aceita mudar a situação
  por fora da aprovação. Pedido não nasce mais "aprovado" pelo corpo da requisição
- feat(pedido de compra): editar e remover linhas antes da aprovação; depois de
  aprovado, eliminar o saldo que falta receber (com motivo); pedido com material
  recebido não é cancelado inteiro. A grade mostra almoxarifado, nome do item e a
  situação de cada linha
- fix(itens): plano de corte, ordem de produção (item substituído), regras de
  custo indireto, relatório de margem e matriz de setup gravavam o código do item
  da tela como se fosse o código interno
- fix(pedido de compra): o fornecedor do pedido passa a ser conferido contra o
  cadastro da empresa autenticada e tem de estar ativo. `suppliers.code` é único
  no banco inteiro e a FK do pedido não olha a empresa: informar o código de um
  fornecedor de OUTRA empresa gravava o pedido, que depois saía sem fornecedor no
  documento (o leitor filtra por empresa) e sem destinatário para envio — e ainda
  podia receber material. A conferência vale para criar, alterar a capa e aprovar
  sugestão do MRP; na geração em lote (requisição e cotação) o fornecedor
  imprestável não derruba o lote: só as linhas dele ficam de fora, com o motivo
- fix(numeração): `NextPurchaseRequisitionCode`, `NextPurchaseQuotationCode`,
  `NextCuttingPlanCode` e `NextFiscalClassificationCode` devolviam `int32` para
  uma coluna `bigint` (faltava o `::bigint` que as outras consultas já têm), e a
  leitura estourava com código acima de 2^31 — o que deixava a suíte de
  integração intermitente. Os modelos de `purchase_order_envios` e
  `purchase_order_item_followups` (migração 383) também foram gerados

## [v1.5.0] — 2026-10-07

- feat(fiscal): ciclo fiscal completo da NF-e de entrada à EFD (`bc4be66`)

- feat(entrada de NF-e): importar a mesma nota duas vezes ao mesmo tempo não
  duplica mais a entrada. Quem chega em segundo recebe "a nota está sendo
  importada por outro usuário" na hora, em vez de esperar ou criar outra entrada
- feat(notas recebidas): a busca das notas emitidas contra a empresa na SEFAZ
  pode rodar sozinha, a cada hora, ligada por empresa. A tela mostra quando foi a
  última consulta e o último erro, e destaca as notas cujo prazo de manifestação
  (180 dias da emissão) vence em até 30 dias
- feat(contabilidade): o ciclo inteiro passa a ser contabilizado, não só a
  entrada. Pagamento de fornecedor (com juros, multa e desconto), recolhimento das
  retenções, recebimento de cliente, NF-e de saída (receita, impostos sobre a
  venda e custo da mercadoria vendida pelo custo médio) e o estorno de cada um no
  cancelamento. Cada conta bancária pode ser ligada à sua conta contábil
- feat(frete sobre compras): o CT-e da transportadora entra pelo XML, é ligado
  às notas que transportou e rateado entre os itens por valor, quantidade ou peso.
  O frete complementa o custo do que ainda está no estoque (o que já foi consumido
  vai para despesa), gera o título da transportadora e é contabilizado
- feat(devolução de compra): devolver ao fornecedor, total ou parcialmente, a
  partir da nota de entrada. A NF-e de devolução sai com o CFOP de devolução da compra (5201/5202,
  6201/6202...) e a nota
  original referenciada; ao ser autorizada, a mercadoria sai do estoque pelo custo
  médio, o valor abate os títulos em aberto da nota e o que sobrar vira crédito a
  receber do fornecedor. Cancelar a devolução desfaz tudo
- feat(SPED): a EFD ICMS/IPI do mês é gerada a partir das próprias notas, sem
  digitação: entradas aprovadas (C100/C170/C190, com o CFOP de entrada e só o ICMS
  que a empresa credita), NF-e emitidas e canceladas, CT-e de frete (D100/D190),
  apuração do ICMS (E110/E116) e do IPI (E500/E510/E520). Antes de baixar, a tela
  mostra o resumo da apuração e o que falta no cadastro para o PVA aceitar
- feat(entrada de NF-e): nota de emitente que não está no cadastro deixa de ser
  aprovada sem fornecedor (o título nascia sem dono). O fornecedor e os itens que
  faltam se cadastram a partir da própria nota, com os dados do XML; fornecedor
  cadastrado depois se liga sozinho; fornecedor bloqueado ou inativo impede a
  aprovação. Nota com frete ou devolução lançados não se cancela antes deles
- fix(NF-e de saída): a transmissão à Focus usava um formato que a API não aceita
  e lia a resposta com nomes errados — nenhuma nota seria autorizada. Conferido
  na homologação da SEFAZ: venda, devolução e cancelamento autorizados. A nota
  passa a levar o grupo IBS/CBS (obrigatório em 2026), o rateio de frete e
  desconto nos itens, a hora certa no fuso de Brasília e o número que a SEFAZ
  autorizou. Nota rejeitada pode ser corrigida e reenviada; cancelar pela tela
  voltou a funcionar; carta de correção e cancelamento recusados pela SEFAZ não
  são mais registrados como feitos. A data/hora de CT-e e NFS-e tinha o mesmo
  erro de fuso
- feat(SPED): ajustes da apuração (E111) a partir das notas especiais de ajuste,
  inventário (bloco H) a pedido e tipo do item (0200) pelo cadastro
- fix(apuração de impostos): os créditos passam a seguir a mesma regra da EFD —
  só o imposto dos itens com crédito, pela data de entrada, mais o ICMS do frete
- fix(notas especiais de ajuste): a empresa vinha da tela (fixa em 1) e a consulta
  por código não filtrava empresa; agora é sempre a do usuário
- fix(SPED): o arquivo gerado seguia um leiaute que o PVA recusava (versão,
  quantidade de campos por registro, blocos sem abertura e encerramento)
- fix(NF-e de saída): cancelar uma nota autorizada não devolvia a mercadoria ao
  estoque
- fix(CT-e): os conhecimentos de transporte não tinham empresa; cada empresa via
  os CT-e das outras
- fix(financeiro): a baixa parcial de título gravava o lançamento sem a empresa,
  e os juros e a multa eram cobrados sobre o saldo inteiro em vez do valor pago

## [v1.4.3] — 2026-10-02

- fix(tabela de preço): formação inexistente recusada com os valores aceitos (`59a468c`)

- fix(tabela de preço): escolher uma formação de preço que não existe passou a
  dizer o que serve. Antes a tela respondia "Dados inválidos. Revise os campos
  destacados", sem apontar o campo — porque o erro vinha do banco em inglês e o
  sistema esconde texto técnico. Agora a mensagem nomeia o valor recusado e lista
  as formações aceitas

## [v1.4.2] — 2026-10-02

- fix(entrada de NF-e): o XML da nota era aceito e lido como se estivesse vazio (`d2686b6`)
- fix(erros): pendência de cadastro e data mal digitada deixam de virar 500 genérico (`843f34f`)

- fix(erros): situações que são do usuário deixaram de aparecer como "erro interno
  do servidor". A mais séria: quando o roteiro de um item passa por uma operação
  feita por terceiro (galvanização, têmpera, zincagem) e o preço daquele serviço
  ainda não foi cadastrado, o cálculo de custo falhava com uma mensagem genérica —
  agora diz qual operação está sem preço e que o preço se cadastra em VTER0100,
  ou que a operação pode ser marcada como interna. O mesmo valia para data digitada
  fora do formato na baixa de conta a pagar e a receber, no adiantamento e no CT-e,
  e para arquivo que não é XML de NF-e no upload de entrada: todos diziam só "erro
  interno", e dois ainda mostravam texto em inglês
- fix(entrada de NF-e): a importação do XML da nota de compra passou a ler a nota
  de verdade. O arquivo era aceito mas lido como se estivesse vazio, e a entrada
  entrava com número 0, sem CNPJ do emitente e com todos os valores zerados. Além
  disso, o arquivo que a SEFAZ devolve ao autorizar a nota (o que o contador manda)
  era recusado com a mensagem de arquivo inválido. Agora as duas formas do arquivo
  são aceitas, e nota sem número ou sem CNPJ do emitente é recusada dizendo o motivo

## [v1.4.1] — 2026-10-01

- fix(fornecedor): o construtor devolve a condição de ICMS que ele validou (`d899a1f`)
- fix(fornecedor): inscrição estadual segue a condição de ICMS, não o tipo (`2f7a20f`)

- fix(fornecedor): cadastrar fornecedor que **não é contribuinte de ICMS** voltou
  a ser possível. O sistema exigia Inscrição Estadual de todo fornecedor que não
  fosse transportadora, e quem não é contribuinte legitimamente não tem inscrição
  — prestador de serviço, pessoa física e boa parte dos MEI. Restavam duas saídas
  erradas: inventar um número, que entra na apuração de ICMS de toda nota de
  entrada do fornecedor, ou declará-lo transportadora. Agora a inscrição é pedida
  conforme o campo "Contrib. ICMS", que já existia na tela, e a mensagem diz o que
  fazer em vez de só recusar

## [v1.4.0] — 2026-10-01

- fix(fiscal): a máscara do NCM não pode decidir se o item é tributado (`94ce2e8`)
- fix(cadastros): próximo código como bigint nos 17 geradores de sequência (`c84b16a`)
- fix: oito defeitos da revisão do PR, cinco deles em documento fiscal ou custo gravado (`18256d5`)
- fix(fiscal,financeiro): erro de caso de uso deixa de virar "requisição inválida" (`436b20c`)
- feat(custos): esquema de rateio de indiretos, componentes e histórico (`cb535e0`)
- fix(conciliação): importar um JSON respondia sucesso com zero lançamento (`d2450ee`)
- fix(financeiro): o filtro da carteira nunca chegava ao banco (`51892ff`)
- feat(beneficiamento): trilha de auditoria e estorno da nota cancelada (`b778141`)
- feat(beneficiamento): material de terceiro, do recebimento ao retorno fiscal (`ecbe218`)
- feat(multiempresa): banco, container e subdomínio por empresa na mesma VPS (`e92013f`)

- feat(multiempresa): o sistema passa a atender mais de uma empresa com banco,
  usuário de banco e serviço próprios para cada uma. Nenhuma empresa alcança o
  dado da outra nem por consulta direta ao banco — a separação foi verificada nas
  duas direções, com tentativa de acesso cruzado recusada pelo próprio Postgres
- feat(beneficiamento): remessa de material do cliente, estoque de terceiros e a
  nota de retorno de industrialização numa só emissão, com a linha do serviço
  (CFOP 5.124) e a do material devolvido (5.902/5.903) na mesma nota, nos CST
  confirmados pela contabilidade. O faturamento pode ser repetido depois de uma
  falha sem gerar nota duplicada, e a baixa pode ser estornada com motivo
  registrado e histórico de quem fez
- feat(custos): custo do item com os componentes separados (material, preparação,
  máquina, mão de obra, terceiros e indiretos) e um esquema de rateio de indiretos
  por percentual, por hora ou por unidade, com a trilha do que foi aplicado e o
  histórico de cada apuração
- feat(financeiro): contas a pagar e a receber com filtro por cliente,
  fornecedor, período, situação, forma de pagamento e documento — antes o filtro
  era desenhado na tela e não chegava ao servidor
- fix(financeiro): a importação de extrato passou a recusar arquivo que não é OFX.
  Antes qualquer arquivo era aceito e a importação dizia "sucesso" sem importar
  nada
- fix(fiscal): o NCM com máscara deixou de zerar o imposto. A tabela tributária e
  a classificação do item eram comparadas letra por letra: "8466.20.90" num lado e
  "84662090" no outro faziam a nota sair sem IPI, PIS e COFINS, sem erro nenhum.
  O NCM também ia com máscara para o XML, o que a SEFAZ recusa
- fix(cadastros): criar cliente, fornecedor e outros cadastros voltou a funcionar
  em base cujo último código já passou de 2 bilhões. Antes a criação falhava com
  uma mensagem técnica em inglês
- feat(telas): VFIS0110, VFIN0200/0210/0300/0600/0620 e VCUS0100 reconstruídas, com
  busca por lupa nos campos de item, cliente, fornecedor e conta bancária, e sem
  nenhum campo pedindo "ID" ao usuário

## [v1.3.0] — 2026-09-27

- fix: seis achados da revisão do PR, três deles vazando dados entre empresas (`5248a2e`)
- feat(nfe): prévia da nota antes de emitir, com a conferência do que impede (`5595e19`)
- fix: sete defeitos achados simulando a rotina da empresa sobre os dados reais (`ff6e2fa`)
- fix(comissao): a razão de comissões passou a ver o rateio, não só a capa (`67f64cd`)
- test(integração): tirar a dependência de fuso de três testes (`1d15271`)
- feat(sessao): "manter conectado" com renovação e teto de 30 dias (`5d0f725`)
- feat(comercial): condição composta, comissão dividida, IPI separado e transportadora (`4facc9a`)
- fix(ci): rodar o banco do CI no fuso da produção (`6b97ac8`)
- fix(ci): versionar a semente da suíte de integração (`d825956`)

- fix(tenant): o relatório de representantes agregava as junções sem recorte de
  empresa — um representante que atende duas empresas devolvia, para cada uma, o
  percentual de comissão combinado com a OUTRA, as regiões das duas misturadas e
  as contas contábeis do plano alheio
- fix(tenant): o detalhe por cliente do acompanhamento de representantes somava
  orçamentos e pedidos de todas as empresas; só o resumo era recortado
- fix(orcamento): o plano de pagamento era dividido sobre o valor dos produtos,
  sem IPI e sem ICMS-ST. Cada parcela saía menor que a devida e a proposta não
  fechava com os títulos, que nascem do total do documento fiscal
- fix(comissao): limpar o rateio deixava o representante antigo na capa do
  documento, e a leitura seguinte reconstruía o rateio a partir dele — a limpeza
  parecia desfazer a si mesma
- fix(transportadora): a ocorrência de entrega aceitava a data de amanhã durante
  quase todo o dia de hoje, apesar da mensagem dizer que futuro é recusado
- fix(transportadora): região cadastrada por faixa de CEP era cotada como se
  atendesse o estado inteiro quando a cotação vinha só com a UF

- feat(nfe): prévia da nota antes de emitir
  (`GET /api/fiscal/exits/{id}/previa`) — monta o documento exatamente como será
  transmitido à SEFAZ e devolve a conferência do que ainda impede a emissão
  ("Impede") e do que passa mas tem consequência ("Atenção"), cada item com a
  tela onde se resolve. Não transmite, não grava e não muda o status da nota
- fix(nfe): a nota não tinha onde guardar o endereço do destinatário, e o
  documento saía sem logradouro, bairro, município e CEP — campos obrigatórios
  no layout da NF-e. O endereço agora é gravado na nota e vem do cadastro do
  cliente (endereço de ENTREGA, ou o de COBRANÇA na falta dele), sem sobrescrever
  o que for digitado na própria nota para uma entrega pontual
- fix(nfe): o número da nota digitada ficava como veio, e todas nasciam com
  número 0; sem número informado, a nota passa a receber o próximo da sequência
  fiscal da empresa (série vazia vira "1")
- fix(nfe): a forma de pagamento declarada era sempre "01 — dinheiro" com o valor
  inteiro da nota, mesmo numa venda a prazo, e nenhuma duplicata era enviada.
  Agora a condição de pagamento vira uma forma por parcela e duplicata para cada
  parcela cobrada depois
- fix(financeiro): o faturamento gerava SEMPRE um único título vencendo em 30
  dias e sem cliente. Passa a gerar um título por parcela da condição de
  pagamento do pedido, com o cliente, o pedido e a forma de pagamento — aging,
  extrato por cliente e limite de crédito deixam de ficar cegos
- fix(orcamento): a conversão copiava o total do orçamento inteiro para a capa do
  pedido; convertendo o saldo de um orçamento parcialmente atendido, o pedido
  nascia valendo mais que a soma dos seus itens. A capa passa a fechar com as
  linhas convertidas, incluindo IPI e ST
- fix(condicao-pagamento): criar ou alterar a condição sem escolher o início das
  parcelas quebrava com erro de banco; em branco passa a valer EMISSAO e um valor
  inválido responde mensagem de validação
- fix(orcamento): a notificação de "orçamento convertido em pedido" saía com
  todos os itens sem descrição (a busca usava o código da empresa numa coluna que
  guarda o identificador)

## [v1.3.0] — 2026-09-25

- feat(condicao-pagamento): percentual e evento base por parcela — "30% entrada,
  20% entrega e o restante em 28/56 dias" passa a ser cadastrável, com listagem,
  exclusão e simulação em dinheiro da condição
- feat(orcamento): plano de pagamento calculado por orçamento
  (`GET /api/sales-quotation/{code}/payment-schedule`)
- feat(comissao): rateio de comissão com mais de um representante por pedido e
  por orçamento, com base de incidência explícita e percentual por documento
- feat(orcamento): produto, IPI e produto + IPI como valores distintos no item e
  na capa
- feat(transportadora): cadastro de transportadora (RNTRC/ANTT, modal, tabela de
  frete, seguro, frota com motorista, regiões atendidas com prazo, ocorrências de
  entrega) e cotação comparativa de frete aberta em componentes
- feat(conversao-de-um): conversão de unidade liberada para o perfil USER
- feat(sessao): "manter conectado" de verdade — `remember_me` no login emite token
  de 7 dias, `POST /users/session/renew` renova a sessão a cada abertura do app e
  o teto de 30 dias (claim `session_start`) impede sessão eterna; o login passou a
  devolver `expires_at`, então o cliente renova antes de levar 401
- fix(orcamento): `total_net_with_ipi` somava o ST apesar do nome, e o valor
  inflado era copiado para o pedido na conversão
- fix(orcamento): a conversão parcial levava o total da quantidade inteira para o
  item do pedido
- fix(condicao-pagamento): criar condição de pagamento voltou a funcionar — a
  coluna de empresa era NOT NULL e ninguém a preenchia desde a migração 361
- fix(condicao-pagamento): a condição passou a ser lida com as parcelas; sem
  isso todo plano de pagamento era calculado como "à vista"
- fix(tenant): condição de pagamento, representante, relatório e acompanhamento
  de representante isolados por empresa
- fix(representante): cadastrar representante cria o vínculo com a empresa da
  sessão — sem ele o representante nascia invisível para o pedido e o orçamento
- fix(item): orçamento, pedido e OF gravavam no ITEM ERRADO quando o código
  comercial é numérico — o código era traduzido duas vezes (middleware + caso de
  uso) e a chave interna 5 era relida como o comercial "5"; achado simulando a
  rotina sobre os dados reais da Tecnofer
- fix(producao): o chão de fábrica não conseguia apontar — a tela manda o CÓDIGO
  do funcionário e as tabelas de produção exigem a chave interna; a tradução
  agora acontece no caso de uso (`employeeresolution`)
- fix(financeiro): baixar título a pagar e a receber estava quebrado — o
  lançamento no fluxo de caixa não preenchia `enterprise_id`, obrigatório desde a
  migração 362
- fix(crp): roteiro sem máscara ficava invisível para o cálculo de capacidade
  (`mask = ''` não trata nulo) e todas as ordens voltavam como "sem carga"
- fix(fiscal): recusa do provedor (Focus NF-e/SEFAZ) chega ao usuário com o
  motivo em HTTP 502, e "token não configurado" virou aviso acionável — antes
  eram 500 "erro interno do servidor"
- fix(planejamento): regra de transição de ordem planejada respondia 500; agora
  responde 422 com o motivo
- fix(comissao): a razão de comissões passou a lançar para TODOS os
  representantes do rateio, respeitando a base de cada linha — o parceiro
  aparecia no pedido e nunca era pago (migração 368). Corrigida no caminho uma
  confusão de convenção vinda da 322: o gatilho comparava `fiscal_exits.enterprise_id`
  com `sales_orders.enterprise_code`, e só acertava na empresa em que id = código
- test(integração): três testes dependiam do fuso (misturavam `time.Now()` do
  processo com `CURRENT_DATE` do banco) e reprovavam no CI perto da virada do dia;
  a suíte passou a rodar verde em UTC, America/Sao_Paulo e Asia/Tokyo

## [v1.2.0] — 2026-09-24

- ci: rodar gofmt, vet, unidade e integração a cada PR (`5ff1240`)
- chore(sqlc): regenerar código e montar as rotas das mudanças acima (`6061263`)
- fix(mensagens): validações em português, com trava contra regressão (`1896318`)
- feat(auditoria): filtrar o log por ação, caminho e situação (`fe7443d`)
- feat(ferramentaria): limiar na lista de troca e aviso de ferramenta vencida (`46aa686`)
- feat(item): recusar item inativo em lançamento (`658c111`)
- fix(producao): plano de inspeção ausente vira aviso, não recusa da ordem (`ffec59e`)
- feat(roteiro): conferência de prontidão antes de rodar o MRP (`1cd8bfa`)
- fix(mrp): eficiência e paradas de consumível valem sobre o tempo do roteiro (`733819d`)
- feat(seguranca): perfil OPERATOR e permissão própria de apontamento (`2f5576e`)
- fix(tenant): isolar financeiro, restrição e rateio por empresa (`056b3a0`)

## [v1.1.33] — 2026-09-22

- fix(producao): tornar liberação e execução de roteiros transacionais (`1cd187d`)
- docs: reescreve o tutorial de roteiro como tutorial (#169) (`fb1a1ca`)
- docs: script de conversão de unidade dos materiais do RN 01001 (#168) (`beee8b3`)

## [v1.1.32] — 2026-09-19

- fix(roteiro): precisão do fator de conversão e segundos nos tempos (#166) (`b2efd4a`)

## [v1.1.31] — 2026-09-18

- feat(roteiro): refugo por operação, inspeção, documentos e unidade de estoque (#164) (`a6e4a01`)

## [v1.1.30] — 2026-09-17

- feat(margem): simulação antes da venda; corrige permissão e 403 em inglês (`9cd09f1`)

### Novidades
- **Simulação de margem** (`VCUS0200`). Antes de fechar o pedido, veja o que sobra: informe preço, quantidade, custo, impostos e comissão e a tela devolve a cascata inteira — inclusive o que o prazo de pagamento custa. Informando a margem desejada, devolve também o **preço mínimo** que a atinge. A conta é a mesma do fechamento do mês, então o número que o vendedor vê é o que vai aparecer na apuração.
- **Tempo de ciclo em segundos.** A ficha de chão de fábrica traz "85 s por peça" e agora é lançado direto, sem converter à mão.
- **Família de preparação no item.** A matriz de setup já aceitava regra por família com um dos lados em branco como coringa — quarenta chapas pedem três regras, não mil e seiscentos pares. A família agora é um campo próprio, de processo, em vez da classificação comercial.
- **Parada de máquina em aberto.** A máquina parou e ainda não voltou: o planejamento passa a tratá-la como ocupada até agora, em vez de só contar paradas já encerradas.

### Correções
- **"Forbidden" em inglês virava texto solto na tela.** Falta de permissão e sessão expirada agora respondem em português dizendo qual perfil você tem e qual a operação exige — e que o perfil vale por empresa.
- **Cadastrar turno ou registrar parada exigia perfil de administrador**, enquanto cadastrar a máquina não. Quem opera a máquina é justamente quem vê ela parar; as três operações passam a aceitar o perfil de usuário. Excluir continua restrito.
- **Motivo de parada inválido** respondia erro interno do servidor.

## [v1.1.29] — 2026-09-16

- feat(máquinas)!: capacidade finita, turno noturno, paradas e consumíveis (`62bf84d`)

### Novidades
- **Capacidade finita de máquina.** O MRP deixou de propor datas no vazio: ele consulta a jornada e os turnos de cada recurso, desconta paradas, manutenção e o que outros planos já reservaram, e devolve os intervalos exatos em que a ordem vai ocupar a máquina. A necessidade original é preservada — a tela mostra início, término previsto e se a capacidade empurrou a entrega para depois do prazo.
- **Produtividade por item, máscara e máquina**, separada da jornada disponível. A mesma peça rende diferente em cada máquina, e a mesma máquina rende diferente em cada peça. Produção contínua pode continuar na janela seguinte; ciclo fechado ocupa inteiro e não atravessa turno.
- **Turno que vira o dia.** O terceiro turno passa a ser cadastrável: informe o fim menor que o início (22:00 às 06:00) e a janela é contínua até o dia seguinte. Antes o cadastro era recusado, e a saída de partir a noite em dois pedaços fazia um ciclo longo deixar de caber em qualquer um deles.
- **Paradas de máquina com hora exata** — quebra, manutenção corretiva, parada programada. O período deixa de existir como capacidade no planejamento, na análise de carga e no sequenciamento.
- **Consumíveis.** Gás de corte, eletrodo, arame. A autonomia de uma carga e o tempo de troca ficam na máquina; o consumo por hora fica no item, porque um cilindro não dura um tempo fixo — dura conforme o que está sendo cortado. O planejamento conta as trocas e soma o tempo delas à ocupação, em vez de deixar a ordem caber no turno na conta e estourar no chão.

### Correções
- **Eficiência do item valia pela metade.** Quando o item declarava a própria eficiência, ela dimensionava o tempo de ciclo mas não a capacidade da máquina — a tela dizia "eficiência 50%" e calculava a capacidade a 100%. O indicador de gargalo ficava otimista justamente nos itens que rendem menos.
- **Horário de parada dependia da configuração do servidor.** Com o banco em fuso diferente do da fábrica, uma parada das 08:00 bloqueava as 11:00 — três horas erradas do turno, sem aviso. O planejamento passa a fixar o fuso da empresa.
- **Motivo de parada inválido** respondia "erro interno do servidor" em inglês; agora explica quais motivos existem.
- Consulta de ocupação da máquina ficava mais lenta conforme o histórico de planejamento crescia.

### Documentação
- Treinamento prático (`docs/treinamento-pratico`): o Dia 1 acompanha as nove abas da tela de máquinas, com turno noturno, paradas e consumíveis sobre o caso real da Tecnofer. 70 páginas.

## [v1.1.28] — 2026-09-14

### Correções
- **A Estrutura de Produto (VENT0210) agora abre exatamente o item selecionado.** Ao buscar um código como RN-01001, uma chave numérica interna podia ser confundida com o código comercial de outro item — por exemplo, "5" — e a tela carregava o produto errado. A busca e a consulta da estrutura passam a preservar o código escolhido de ponta a ponta.

## [v1.1.27] — 2026-09-14

- fix(estrutura): tipar os erros que o handler novo transformava em 500 (`2159ed7`)

### Correções
- **"Erro interno do servidor" ao repetir a posição de um componente.** Informar uma sequência já usada na estrutura respondia erro interno; agora responde que a posição está ocupada e pede outra. A mensagem já existia e estava em português — faltava o tipo, e o classificador de erros trabalha por tipo.
- **Estrutura com muitos níveis avisava em inglês** (`max BOM depth reached`). Passa a explicar em português que o limite foi passado e que provavelmente há um componente voltando a si mesmo.
- **Máscara não cadastrada para o item** também virava erro interno na consulta da estrutura.

## [v1.1.26] — 2026-09-14

- fix(estrutura): avisos em português na VENT0210 e ciclo que explica a causa (`2eeecbf`)

### Correções
- **Estrutura de produtos: os três avisos que saíam como "erro interno do servidor".** Componente repetido, situação em branco e ciclo eram recusados pelo banco, e a tela mostrava o erro cru do PostgreSQL. Agora: componente repetido responde "este componente já está na estrutura deste item"; situação em branco assume `ATIVO`; e o item não pode mais ser componente de si mesmo.
- **O aviso de ciclo agora diz qual é o problema.** Em vez de "o componente criaria um ciclo na estrutura", a mensagem nomeia os dois itens e aponta a causa real — *"não é possível incluir X dentro de Y: o item X já contém Y na sua estrutura (direta ou indiretamente). Verifique se a estrutura existente não está invertida."* Quase sempre é isso: a estrutura foi cadastrada no sentido contrário, e a recusa era o sintoma.

## [v1.1.25] — 2026-09-13

### Correções
- **Telas que respondiam "erro interno do servidor" ao salvar agora explicam o problema.** As mensagens já existiam e estavam em português — "data do calendário inválida", "produto não encontrado", "este item já existe" —, mas nasciam sem tipo, e o classificador de erros trabalha por tipo: caíam no ramo final e viravam 500. Atingia o calendário industrial e os 17 avisos do catálogo central de erros.
- **"Não encontrado" passa a responder 404 dizendo O QUE não foi encontrado**, em vez de 500 com texto em inglês: máquina, ordem, sugestão do MRP, plano de inspeção, registro de qualidade, não conformidade, ferramenta, série de ferramenta e custo de compra/padrão.
- **Avisos do plano de corte** ("ordem de produção não encontrada", "matéria-prima não encontrada") estavam em inglês na tela.
- O aviso de parâmetro de busca inválido estava em inglês (`params invalid`).

## [v1.1.24] — 2026-09-13

### Estoque e almoxarifado
- **Endereçamento de estoque.** Saldo, saldo por lote e movimento passam a guardar o endereço; uma transferência interna é um movimento só, com origem e destino. Endereço vazio significa almoxarifado sem endereçamento, então nada muda para quem não usa.
- **Separação FEFO** (`/api/stock/separation/suggest`): ordena por validade, desconta o que já está reservado, ignora lote vencido e endereço bloqueado — contando ambos à parte — e devolve a lista na ordem da rota do galpão.
- **Onda de separação** (`/api/stock/waves`): agrupa necessidades numa caminhada só e reserva no endereço/lote dentro da mesma transação, com o saldo travado. Confirmar baixa o estoque; cancelar devolve o reservado.
- **Sugestão de guarda** (`/api/stock/putaway/suggest`): endereço fixo do item, consolidação, endereço vazio da zona ou qualquer um com espaço, respeitando capacidade e bloqueio.
- **Curva ABC apurada pelo valor consumido** (`/api/stock/abc/recalc`), gravada em `items.planning_abc_class`. A frequência da contagem cíclica passa a derivar da classe quando não há intervalo por item.
- **Contagem cíclica por endereço**: a quantidade esperada passou a filtrar o endereço.
- **Validade do lote** (`stock_lots.expires_at`), que ordena o FEFO.

### Correções
- **A tradução do código público do item não chegava ao handler em rotas montadas.** Middleware registrado em `r.Group` roda depois do roteamento, e as rotas atrás de `r.Route("/api/stock", …)` recebiam o código cru: erro com código alfanumérico e, pior, saldo e ATP zerados em silêncio com código numérico. A tradução do caminho passou a ser middleware de raiz, com teste que cobre cada rota nova.
- **A fórmula de perdas da estrutura divergia entre módulos.** O MRP respeitava o parâmetro (padrão: divisão) e a criação da ordem, o encerramento, o apontamento e o custo fixavam a multiplicação. Conta unificada em `QuantidadeComPerda`.
- **Movimento de estoque aceitava qualquer tipo**, gerando registro que não alterava o saldo. Agora é validado contra os tipos que a apuração reconhece.
- **Saída de estoque era gravada com valor zero**; passa a ser valorizada pelo custo médio.
- **Erros que viravam "erro interno do servidor"**: mensagens de regra sem tipo na previsão de venda, texto maior que a coluna (SQLSTATE 22001) e número fora de faixa (22003).
- **Campo de data recusado** em cabeçalho de estrutura e classificação fiscal: passam a aceitar `AAAA-MM-DD`, que é o que o navegador envia.
- **Cadastro de máquina não devolvia a unidade de capacidade** com o nome que a gravação espera, então reenviar o que foi lido era recusado.
- **Marcações booleanas voltavam a desligadas** numa gravação parcial (tipo de máquina, característica do configurador).
- Mensagens com nome interno de campo, sem acentuação ou em inglês foram corrigidas.

### Segurança
- **`employees` e `cost_centers` não filtravam empresa** — inclusive na alteração e na inativação por código. `cost_centers` não tinha sequer a coluna.
- A apuração de custo-padrão usava como autor o valor enviado pelo cliente; passa a usar o usuário autenticado.

## [v1.1.23] — 2026-09-11

### Correções
- **O cadastro de item deixava de gravar três campos na alteração.** Almoxarifado padrão, almoxarifado de suprimentos e consumo médio informado à mão ficavam como estavam na criação: a tela dizia "salvo" e o valor anterior continuava lá.
- **As observações da pasta Suprimentos não eram gravadas** — o campo existia na tela e não tinha onde ser guardado.
- **Os almoxarifados de transferência e de assistência técnica não aceitavam códigos com letras** (ALM-PA, ALM-EXP). O código era convertido para número e desaparecia sem erro.
- **A lista de materiais da ordem de produção ignorava a configuração do item.** A ordem de um produto configurado recebia os componentes de todas as variantes, incluía componentes fora de vigência e usava a quantidade fixa mesmo quando havia fórmula.
- **O MRP e o plano de corte também explodiam componentes fora de vigência.** A vigência passa a ser avaliada na data em que o componente será consumido e antes da escolha do material alternativo: se o principal venceu, a necessidade vai para o próximo da lista em vez de sumir.
- **Estrutura sem máscara aceitava o mesmo componente duas vezes**, e a quantidade entrava em dobro no planejamento e no custo.

### Melhorias
- **A máscara do componente configurado passa a ter três origens**: a mesma característica respondida no item pai, uma regra de equivalência ou a resposta padrão da característica. O que não se encaixa em nenhuma delas é apontado como configuração incompleta, em vez de virar silenciosamente a estrutura genérica.
- **Nova verificação de configuração da estrutura**, usada pelo botão Conferir: mostra, ainda no cadastro, qual componente tem característica sem resposta e qual é configurado sem herdar a máscara do pai.
- **Trilha de auditoria, configurador de produto, fornecedores, motivos de restrição e custo por centro de trabalho passaram a ser separados por empresa.**

## [v1.1.22] — 2026-09-10

- fix: corrige a direção da validação de ciclos ao incluir componentes na estrutura
- fix: bloqueia autorreferência direta sem impedir a primeira relação pai-filho

## [v1.1.21] — 2026-09-10

- fix: estabiliza o cadastro de estruturas com códigos de item alfanuméricos
- fix: preserva mensagens detalhadas de validação no fluxo de estrutura

## [v1.1.20] — 2026-09-10

- fix: restaura o salvamento de componentes no cadastro de estrutura de produto

## [v1.1.19] — 2026-09-10

- fix: aceita espessura decimal nas dimensões do cadastro de item

## [v1.1.18] — 2026-09-09

- fix: alteração de máquina, cadastro completo do recurso e isolamento da fila (`cd67bb8`)
- fix: traduz as mensagens de validação do cadastro de item (`ec11f4d`)

## [v1.1.17] — 2026-09-09

- fix: situação omitida em atualização deixa de inativar o cadastro (`d679960`)

## [v1.1.16] — 2026-09-07

- feat: alteração de item, resolução de código no PDM e erros tipados (`3c29707`)
- feat: executa o planejamento na janela noturna com consistência garantida (`5225c3f`)
- fix: traduz as mensagens ao usuário e destrava o cadastro por item-base (#134) (`a73f2ec`)

## [v1.1.15] — 2026-09-05

- fix: devolve status correto e mensagens em português nos erros (#132) (`ccd42ef`)

## [v1.1.14] — 2026-09-04

- fix: corrige listagem de endereços de almoxarifado (#130) (`ecaf468`)

## [v1.1.13] — 2026-09-02

- fix: preserva códigos hierárquicos de classificação por rota

## [v1.1.12] — 2026-09-02

- docs: atualiza artefato do treinamento (`a4a3d91`)
- docs: detalha corte e expedição no treinamento (`6d31a99`)
- feat: conclui ajustes operacionais enterprise (`7e8714a`)

## [v1.1.11] — 2026-08-25

- fix: gera código de reprogramação no banco (`3d09419`)

## [v1.1.10] — 2026-08-25

- fix: preenche inscrição estadual na consulta de CNPJ (`0f6a3c0`)

## [v1.1.9] — 2026-08-25

- fix: valida contratos e rotas comerciais (`ae982f7`)

## [v1.1.8] — 2026-08-21

- feat: bloqueia clientes desktop incompatíveis (`bdaae5d`)

## [v1.1.7] — 2026-08-20

- feat: entrega alertas enterprise e correcoes operacionais (`a812d32`)

## [v1.1.6] — 2026-08-13

- feat: finaliza documentos industriais e heranca fiscal (`9bce3f7`)

## [v1.1.5] — 2026-08-13

- feat: conclui ajustes operacionais enterprise (`ab522df`)
- feat: ajusta cadastros operacionais por empresa (`d7d7a53`)

## [v1.1.4] — 2026-08-09

- fix: make item base reference optional (`d417171`)

## [v1.1.3] — 2026-08-09

- feat: expand item commercial and accounting data (`92d2cbe`)

## [v1.1.2] — 2026-08-03

- fix: repair item schema on advanced databases (`9cf37b2`)

## [v1.1.1] — 2026-08-03

- fix: complete item master data contracts (`6dc7dd7`)

## [v1.1.0] — 2026-07-24

- feat: complete sales quotations (`6caf6c4`)

## [v1.0.3] — 2026-07-19

- fix: preserve backend desktop compatibility (`de11d48`)

## [v1.0.2] — 2026-07-19

- security: harden tenant authorization and production defaults (`50121d9`)
- fix(cors): use real Tauri v2 origins for the desktop client (`b8fcab4`)
- fix(update): make the update queue readable/writable by the API user (`ce532d7`)

## [v1.0.1] — 2026-07-16

- fix(cutover): pause update watcher during the direct bootstrap run (`248c8ca`)
- docs(release): document PR-based release flow, VPS provisioning and cutover (`36688b3`)

## [v1.0.0] — 2026-07-16

- fix(release): land releases via auto-merged PR and harden VPS updater (`c774024`)
- fix(release): expose release targets in the canonical makefile (`c1c9a12`)
- feat(release): add versioned backend self-update pipeline (`105c723`)
- fix: stop fiscal polling after Focus HTTP errors (`8fd9027`)
- fix: align runtime contracts used by frontend (`80166e7`)
- feat(auth): add admin-approved password changes (`faeaeeb`)
- fix: bootstrap isolated local development database (`ee68e29`)
- chore: add isolated development environment (`b869ac3`)
- fix: configure VentureERP trace resource identity (`904b446`)
- chore: standardize VentureERP runtime identity (`48339c2`)
- fix: preserve authenticated CNPJ lookup (`10bedb6`)
- chore: harden production fiscal and observability config (`a823871`)
- fix(auth): restrict user registration to admins (`379a939`)
- docs: separate production and development environments (`85b860f`)
- implementa módulo de compras (`9ff88f6`)
- feat: implement third-party manufacturing services (`a25d9ba`)
- feat: complete enterprise product sequencing (`c2ca4bb`)
- feat: complete enterprise material planning and manufacturing (`47a5739`)
- feat: add product configurator and tool production sheet, remove legacy questions system (`601171b`)
- fix commit (`bd339dc`)
- fix(loadtest): chaos report captures full output with absolute paths (`f6c08c7`)
- fix(loadtest): Docker permission, SSH sudo, chaos timing calc (`c7bb86c`)
- feat(loadtest): API load/stress/chaos testing suite (`858df07`)
- fix(observability): add OTEL_EXPORTER_OTLP_ENDPOINT to .env.example (`3d39463`)
- feat(observability): production docker-compose, sync configs, security hardening (`940fc27`)
- feat(observability): add full monitoring stack with OpenTelemetry, Prometheus, Loki, Tempo (`776d3dd`)
- feat: completa validacao das fases comerciais (`6e22115`)
- feat(comercial): add forecast promise and assistance phases (`b955298`)
- feat(comercial): add representatives and sales goals phases (`15ef5ba`)
- Implement commercial sales phases 3 and 4 (`6b6583d`)
- Expand commercial policy documentation (`5c49860`)
- Add commercial pricing and policy engines (`3246c20`)
- feat(suprimentos,bom): fecha maturidade de Compras/Suprimentos + substitutos de BOM (`90bb2cf`)
- refactor(bom): item_structures como fonte única; aposenta /api/bom, adiciona bom_headers (`d9c2458`)
- feat(mrp,bom): firmar sugestão gera OF + co-produtos/sucata e quantidade fixa na BOM (`13fd54b`)
- feat(routing): evolução enterprise+ do Roteiro de Fabricação (R1–R6) (`485a0dd`)
- fix(backend): close reported API gaps across sales, PDM, production, MRP & costs (`a536c73`)
- feat(aps-gantt): quadro de programação — reschedule, dependências e zoom semana/dia (`2754736`)
- feat(cutting-plan): enterprise-grade nesting optimisation (1D/2D/true-shape) + shop-floor add-ons (`261134d`)
- feat(shipment-manifest): Outbound Delivery dispatch + professional PDF export + demo environment (`d642541`)
- feat(cutting-plan): nesting & cutting-plan module (1D / 2D / true-shape) with firming, auto-demand and shop-floor outputs (`9491577`)
- Features: Auto-fill by CNPJ, Excel/PDF/CSV export, RBAC, and MRP suggestion confirmation (`5f4103b`)
- feat: sqlc yaml (`9d03225`)
- Feature: Readiness for Metalworking — Actual Cost of Work Order, Credit, ATP, Batch Traceability, and Average Consumption (`1550047`)
- refactor: padroniza camada de saída com response DTOs + mappers em todos os usecases (`0537e02`)
- Features: Operational readiness (go-live) + tax, shipping, and documentation gaps (`b150d94`)
- test: suíte de testes unitários e de integração + alvos de teste (`1e117fb`)
- Features: Supplier, Purchases (Order/Request/Quote) and Tax Registrations (`816d2c6`)
- feat: fix post-sqlc-generate build errors, add MRP email alerts and improve routing module (`8e25b72`)
-   feat: question/item association queries, item listing, and bug fixes (`c83c3ad`)
- feat: implement restriction/dependency maintenance module with active evaluation (`dbc0d4b`)
- feat: fix fiscal payload bugs, add pis/cofins aliquota pipeline and reorganize use cases to proper   Clean Architecture (`e890b58`)
- feat: Completing the implementation of the tax module and documentation. (`e185209`)
- feat: implement MRP, fiscal and financial modules (`97eb32b`)
- feat: full implementation of MRP. (`6bf734a`)
- feat: I implemented the logic to correctly generate planned orders. (`5dc462e`)
- feat: implementing overhead allocation logic (`bf74944`)
- feat: implementing order priority logic (`049a295`)
- feat: implement MRP calculation module with full clean architecture (`f8563fc`)
- feat: Implementation of a robust and standardized logging structure (`40a3700`)
- refactor: I created the logic and item validation in a certain unit, but the machine is configured for another, now making it impossible to use different types and/or convert the types correctly in their proper proportion to achieve creation (only if type conversion makes sense). (`ec6880b`)
- feat: Implemented the necessary logic to program and use a machine in the system. (`35be388`)
- feat: database transition to Supabase (`dc93b65`)
- feat: Implemented the logic for the calendar promise item. (`2cd8a74`)
- feat(core): I aligned repositories with interfaces, fixed the SQLC integration between modules, and added the logic to create the industrial calendar. (`4d90c3c`)
- feat: I implemented the necessary logic to create, update, delete, and list (by date, item, and code) an independent demand. (`462da0a`)
- refactor: I updated the name needed to list or the delivery reschelude (`ba7e1a4`)
- feat: implementei a lógica necessária para criar e listar o delivery reschelude (`f8e484f`)
- feat: I implemented the delivery promise logic. (`f5cb150`)
- feat: I implemented the logic for cost_center and corrected the ID to CODE allocation. (`47e3d7a`)
- feat: Implemented the tables and SQL structure necessary for the MRP to function, as well as the allocation logic. (`1a95a92`)
- Refactor(structure): Align component updates with business keys and correct domain inconsistencies (`782ba47`)
- refactor: Correcting the return value when querying a structure and renaming a column in the structure table (`64af22d`)
- feat(structure-query): Implements BOM structure query with mask propagation (`900d603`)
- refactor(structure): Optimizes direct child queries with direct DTOs and joins on items (`b688141`)
- remove IDE folders from repository (`937c547`)
- fix: Fixed the item search function (`2b0ba13`)
- fix: Business Rule (`69361d4`)
- feat: Added to the first version of creating a structure (`da7b78d`)
- fix: Corrected the logic for generating a mask. (`7c95b91`)
- fix: I corrected the SQL for creating an option for a question. Previously, it wasn't possible to create the same value for different questions. The way to check if a question exists has also been corrected. (`d3c1569`)
- fix: I redid the entire mask generation process. (`f9fe476`)
- fix: Corrected the functionality for generating a question. (`212e814`)
- fix: Updating the code to now associate questions, create questions, and generate masks based on the item and not the product, complying with the new business rule. (`4f30f12`)
- fix: correcting the route to create an employee (`1af934f`)
- feat: All the necessary logic for creating an employee in the system has been implemented. (`5cbcd7d`)
- feat: Implemented the beginning of the logic to create an employee (`c77b4b6`)
- fix: Corrected the columns and column types in the warehouse table. (`d5ff684`)
- feat: Implemented all the necessary logic to create a modifier for a group. (`0937779`)
- feat: Implemented the logic to create a company in the system. (`4018da5`)
- feat: start logic create enterprise (`193c1ab`)
- feat: finalized the creation of the necessary logic to create a group. (`d592dbf`)
- feat: starting the logic needed to register a group (`dac99d7`)
- fix: error resolved create_machine.go (`13c4e58`)
- Feature: Finalizing the implementation to create an item in the system. (`f1b11eb`)
- fix: dto request and new entity (`c4ace50`)
- finalizing the modeling of an item in the system. (`585797e`)
- feat: finishing modeling an item to be registered on the platform (`27ed9ac`)
- feat: Added new fields to the item entity. (`5a8728d`)
- feat: Starting to model an item for the ERP (`22c997a`)
- changed: I modified the entity representing a warehouse. (`4329d69`)
- feat: Development of the logic for creating a warehouse is complete. (`e452f4c`)
- feat: Almost all the logic needed to create a basic warehouse in the system has been implemented (`5fb5ddb`)
- feat: starting to improve a warehouse entity (`7fa0803`)
- fix: Changed the int64 value representing the warehouse ID to int32. (`089f657`)
- fix: add code in use case (`ce07c9b`)
- fix: Corrected the fields in the items table, the questions options table, and corrected the way data is passed in repository_sqlc. (`af59a4b`)
- fix: implemented the items table and corrected the discount field to description (`533e774`)
- feat: Implementing the logic to create items (`80878a1`)
- complement (`86255c3`)
- feat: Added logic for items, types, and beginning of critical changes (`aa0e560`)
- Refactoring: In the SQLC repository, two methods of QuestionOption were refactored. The first method, which creates a question option, previously only returned an error; now it returns the entity and the error. The second method, ExistsQuestionOptionByValue, which checks if an option exists based on its value, incorrectly returns the entity and an error. It should only return a bool and the error if it occurs. Both have been corrected. (`8230be8`)
- Feature: Implemented the methods as ExistsProductByCode. There was also a correction to the names; before they were too generic, now they are more descriptive. Added the queries for the new methods, along with all the necessary logic to make them work. There are still 2 errors that will be corrected soon to complete this task. (`9d891c7`)
- Refactor: permission validation has been added to some use cases. It's important to note that the logic for deleting X will be removed and replaced with one that simply deactivates X, thus prohibiting its use in the system until it is reactivated. And the use case for generating masks will undergo significant changes in the future! (`7099d3b`)
- Refactor: All creation use cases have been refactored. They now perform more precise checks and have minimally correct logic. It's important to note that they will undergo further changes in the near future. The next step is to do the same for delete, search, etc. use cases. (`70aef95`)
- refactor(auth): decouple authorization from JWT using AuthUser and context (`59c7dcf`)
- Feat: create valueobject (`845fc81`)
- Feat: Implemented the entity, DTO, and repository to create a component. (`a31eff3`)
- Refactor: Removed the required  and  fields for creating a product. There was no need for a product to have both fields. (`53582d4`)
- Feature: 100% implemented the logic for creating good items, with some corrections to create a good item as well. (`29f7d2f`)
- Feature: Implemented functionality to create a BOM (Good Items). Logic very similar to creating a BOM, but not yet 100%. (`6d98a75`)
- Feature: Implemented the logic to generate a BOM that will be consumed by MRP in the future. The next step is to generate the BOM_ITEMS. (`a77cdd8`)
- Feature: Implemented new tables in the database to follow the BOM and MRP rules of ERP systems, and a new unit column for products. The updated_at column was removed to ensure consistency in the ERP environment, and with the addition of BOM, mask_composition is no longer necessary. (`5e62354`)
- refactor(product): The new error and handling standard has been implemented for files that retrieve a product from the database. (`67ca786`)
- remove var in handler (`f1e459f`)
- feat(handler): A new handler pattern has been adopted using helpers and standard response and error methods. With the new changes, there have been subtle updates to the usecase, repository, and repository_sqlc. The other handlers and necessary files will undergo the same update. (`8441e20`)
- feat(question): A function has been implemented to search for a question by its name. (`cea0def`)
- refactor(request data): I defined the JSON tags in the struct to correctly serialize and deserialize the fields, and also to allow Go names to be converted to the desired JSON format. (`e87db54`)
- refactor(generate_mask): The generation of masks for products has been corrected. Previously, the mask was generated using the IDs of the responses, so I corrected it to pass the response itself instead of the IDs. (`1bc45e2`)
- feat(generate_mask): I removed the  from the request to capture via claims (I'll probably do the same for the rest of the application), created constructors, and built the handler. It's not perfect yet, but it's working; I'll fix a few things later. (`5740ebd`)
- feat(generate_mask): I just finished the use case, created getters in the value object, added the questions field to the mask entity, and started creating the handler. (`d64835b`)
- feat(generate_mask): I created the value object to validate and generate a mask, and started the process of generating a mask in the use case. (`4c4dd95`)
- wip: initial progress on associating questions with the product. Fixed the SQL tables, wrote the domain/entity and repository, started writing the value object to return the mask that will be used in the use case. (`91e86db`)
- refactor(associate_question_product): There was a change in the file names. I changed the word Create to Associate to make it clearer that we are associating a question with a product. I also rewrote the query, correcting the method names, and corrected the DTO instance in the use case. (`32caadd`)
- feat(associate_question_product): I implemented the necessary logic to associate one or more questions with a product. I needed to create a DTO in infrastructure so that, in cases where I receive a request to associate multiple questions with a product (which is very common and likely), we can perform this action. To do this, I simply created an array in the DTO passing the question ID and the position where the question should be placed. Then, in the handler, I used a  loop to iterate through the entire request for each question that was passed. The route is not yet available because I still need to implement two more cases defined in . (`419aef7`)
- feat(product_question): Implements the migrations and domain (beginning of the logic) to make questions available for a specific product. (`810ec2f`)
- update(migration): remove comments (`696666e`)
- refactor(database): Updates the generate_product_mask trigger to use the position of the questions. Replaces the old trigger and function that generated masks based on the ID to generate a deterministic mask using pq.position, respecting the defined product order, updates the mask hash with the MD5 of the ordered answers, and maintains the AFTER INSERT trigger for automatic creation in product_masks. (`ddd26cd`)
- feat(question_option): Implemented the logic to delete a question option. (`6cbef37`)
- feat(question_option): Implemented the necessary logic to generate options for each question. Now you can create a question and also create values ​​for the answer to that question. The next step is to generate the mask based on the completed questions. (`cf3bda7`)
- feat(products): Implemented functionality to search for a product using its name and generated code. It uses a composite index to speed up the process. (`41d5d41`)
- feat(middleware): A function was implemented to check each user's roles before executing any request. Because of this, I changed the routes to utilize middleware and role verification, ensuring security. (`de2c38c`)
- feat(question): implemented the function to delete a question. (`0190e1a`)
- feat(question): A question creation system has been implemented to generate a mask. A mask details the characteristics of a product, for example: PRODUCT#180#200#20. Based on this mask, we have the product name, length, depth, and height, for example. This mask is generated based on the selection of options in a question. That is, we have a question like: LENGTH, and then there will be several measurements; based on the choices, we build a mask. The next step is to create this options system. But first, I intend to fix some ERP logic problems, create functions to delete and search questions, and other points I'm still working on. (`5287453`)
- Refactoring (product): Due to changes in the database resulting from the migration, it was determined that only the user would remain as a UUID for the time being, while the other tables would change to BIGINT. Based on this change, refactoring was performed to accommodate the database. Now all products are generated with int64 to match BIGINT. (`e96dce1`)
- refactor(dto): I separated the DTOs into request and response DTOs, thus maintaining greater code clarity and making it easy to identify which DTO does what in the application. (`c04b5eb`)
- refactor(migrations): I removed all tables from the database and along with them all migrations, then I created a new migration containing everything that was there before, with the data types and relationships corrected, and I also implemented a function to automatically generate masks for a product based on the questions of a product. For example: if a product has three questions (length, height, and depth) and the mask would be: 180#50#100 (`34d6f8d`)
- feat(product): A value object was created to generate a product mask code. This code uses only the first 2 digits of the groupCode and 4 random digits, without using the current date to create the code. (`faa9b7e`)
- Refactoring (product): The value object for generating product codes was refactored to ensure successful generation of the 6-digit code GGMMDDNN (2 digits from the code group, 4 digits from the month and day, and 2 random numbers) when creating a product. (`8ce77c9`)
- feat(product): A value object has been created to generate internal product codes when they are created. (`b6c5eba`)
- Refactoring (product): The GroupCode becomes a string instead of int16 in the application, and in the database it becomes VARCHAR instead of SMALLINT. (`f29e9b4`)
- feat(product): implementing all the logic needed to delete a product. (`72ed753`)
- feat(queries): Created queries to generate and delete product masks, and also added a query to delete a product. (`373856f`)
- refactor(api): The code for creating a product was removed from the middleware for testing purposes. It will be reintegrated into the middleware soon, and the route for creating a product was also adjusted. (`c0bb5b8`)
- Feat(user): All the necessary logic for creating a user has been implemented. DTOs, use cases, repositories/repositorys_sqlc, entities, and handlers for user registration and user login have been added. The names of some files that reference the product have also been corrected, such as product/entity/entity.go -> product/entity/product_entity.go. (`ce93df0`)
- Product Feature: I implemented a DTO to load the necessary data from a request to create a product. The use case and its constructor were also implemented. I added  to the product entity, and its constructor was also implemented. I created an SQL query to insert a product into the database, along with the actual logic that uses the queries generated by SQLC, in the green layer of the clean architecture in the  file located in the  folder in . (`5de26a0`)
- Feat: Implemented the entity along with the repository and the value object related to the product in our ERP. (`9b53d0d`)
- feat(db): Create initial schema for products, masks, and components in the ERP system. Adds the initial database structure for the industrial ERP system, including products, product masks, components, component masks, composition hierarchy, and material consumption. Defines enums, keys, indexes, and relationships necessary to support product configurations, Bill of Materials (BOM), and future evolutions such as cost, inventory, and production order. (`eb3d960`)
- Features: Implements JWT base code, replaces chi with chi v5, and implements functions to generate a hash from a password and to compare the hash with the password. (`de97583`)
- Feature: We created a secure initialization with the server, along with a route called health for server status testing. We also implemented database connections, along with a test query using SQLC and the migrate package for migrations. I used a makefile to facilitate the use of the migrate package. The database was instantiated via Docker. (`0557e72`)
- Feature: The basic structure of the Clean Architecture has been created. It may undergo changes as the project progresses, as this is only a draft of the Clean Architecture. (`66a39fe`)
- first commit (`4b12737`)

- Fundação do processo de versionamento e atualização assinada.
