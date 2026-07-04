# Roadmap de maturidade ERP

Comparativo operacional entre o VentureERP atual e rotinas esperadas em ERPs
industriais maduros, usando como referência pública a árvore de processos do
FoccoERP (`help.foccoerp.com.br`) e práticas comuns de SAP/Oracle ERP.

## Ranking de setores críticos

| Ordem | Setor | Diagnóstico |
|---|---|---|
| — | Compras / Suprimentos | **Fechado no backend** (migrations `000182`–`000185`): recebimento por linha, inspeção estruturada com ciclo de estoque, aviso/divergências, IQF auto-calculado, alçada com bloqueio, contratos com saldo, EDI estruturado, importação com custo nacionalizado, parâmetros e homologação. Resta apenas frontend e refinos opcionais. |
| 1 | Comercial | Próximo setor. Vendas, promessa, reserva e expedição existem, mas faltam rotinas de assistência técnica completa, CRM/pós-venda, metas, EDI cliente e políticas comerciais avançadas. |
| 2 | Engenharia / Produção | É a área mais madura hoje: BOM, substitutos, MRP, CRP, APS, roteiro, produção, qualidade, manutenção e plano de corte já estão avançados. As lacunas são refinamentos industriais e integração fina com assistência/serviço. |

## Setor atual: Compras / Suprimentos

### O que aparece no FoccoERP

Na área de Suprimentos, o FoccoERP organiza processos como:

- Alçada de Valores;
- Avaliação de Fornecedores;
- Aviso de Recebimento;
- Cadastro de Fornecedores;
- Cálculo de ICMS-ST do Pedido de Compra;
- Contra Nota Produtor Rural;
- Contrato de Fornecedores;
- Cotação de Compra;
- EDI Fornecedores;
- Emissão de Etiquetas da Nota de Entrada;
- Entrada da Nota a partir do Aviso de Recebimento;
- Inspeção de Recebimento;
- Item Comercial - Recebimento;
- Nota Fiscal de Importação;
- Pedido de Compra;
- Recebimento;
- Solicitação de Compra.

### Telas/programas FoccoERP relacionados a Suprimentos

O help do FoccoERP separa os processos de Suprimentos em telas/programas com código.
Esta é a matriz de referência que vamos usar para evoluir o VentureERP.

| Processo | Tela/programa FoccoERP | Situação no VentureERP |
|---|---|---|
| Parâmetros da tabela de compra | `FUTL0125 PRC PRC` | Implementado no painel de parâmetros (`domain=PURCHASE_TABLE`) + tabela de preço de compra. |
| Parâmetros de pedidos de compra | `FUTL0125 PDC PDC` | Implementado no painel de parâmetros (`domain=PURCHASE_ORDER`). |
| Parâmetros de cotação de compra | `FUTL0125 COT COT` | Implementado no painel de parâmetros (`domain=QUOTATION`). |
| Parâmetros de solicitação de compra | `FUTL0125 SLC SLC` | Implementado no painel de parâmetros (`domain=REQUISITION`). |
| Parâmetros de aviso de recebimento | `FUTL0125 AVR AVR` | Implementado no painel de parâmetros (`domain=RECEIVING_NOTICE`). |
| Parâmetros de inspeção de recebimento | `FUTL0125 INSP INSP` | Implementado no painel de parâmetros (`domain=INSPECTION`) + inspeção já integrada ao recebimento (FINS0212). |
| Parâmetros de avaliação de fornecedor | `FUTL0125 AVF AVF` | Implementado no painel de parâmetros (`domain=SUPPLIER_EVALUATION`). |
| Parâmetros de contratos de fornecedores | `FUTL0125 CTRA CTRA` | Implementado no painel de parâmetros (`domain=CONTRACT`). |
| Parâmetros de fornecedores | `FUTL0125 FOR FOR` | Implementado no painel de parâmetros (`domain=SUPPLIER`) + cadastro rico de fornecedor. |
| Parâmetros de notas fiscais de entrada | `FUTL0125 NFE NFE` | Implementado no painel de parâmetros (`domain=NF_ENTRY`) + NF-e de entrada. |
| Cadastro de fornecedores | `FFOR0200` | Implementado. |
| Descrições de itens por fornecedor | `FFOR0201` | Implementado em fornecedor preferencial/item-fornecedor. |
| Itens por fornecedor | `FFOR0202` | Implementado. |
| Geração de itens por fornecedor | `FFOR0204` | Implementado: `POST /api/procurement/suppliers/{code}/generate-items` cria vínculos item-fornecedor a partir do histórico de compras. |
| Cadastro de cotação de compra | `FCOT0200` | Implementado. |
| Cotação de frete | `FCOT0200 FRE` | Não implementado. |
| Análise de cotação de compra | `FCOT0201` | Backend com seleção de vencedor e preços por fornecedor; a grade comparativa lado a lado é frontend. |
| Liberação para cotação | `FCOT0202` | Implementado para solicitações e ordens planejadas. |
| Consulta da cotação de compra | `CCOT0400` | Implementado via consulta da cotação. |
| Relatório de cotação de compra | `FCOT0300` | Dados disponíveis pela consulta de cotação; renderização de relatório é frontend/export. |
| Cadastro do pedido de compra | `FPDC0200` | Implementado. |
| Pedido de compra de serviço | `FPDC0200 SER` | Parcial: requisição de serviço por operação externa existe; falta tela específica. |
| Pedido de frete | `FPDC0200 FRE` | Parcial: campos de frete existem no pedido; falta fluxo próprio. |
| Geração de pedidos a partir de solicitações | `FPDC0204` | Implementado. |
| Cancelamento/atendimento de pedidos | `FPDC0205` | Backend completo (cancelamento + recebimento por linha + status parcial/recebido); console é frontend. |
| Liberação de ordens de compra planejadas | `FPLA0202` | Implementado via sugestão MRP/aprovação. |
| Liberação de ordens de compra para cotação | `FPLA0203` | Backend completo (cotação a partir de ordens planejadas); tela dedicada é frontend. |
| Consulta de pedido de compra | `CPDC0400` | Implementado. |
| Consulta itens a comprar | `CPDC0402` | Backend completo (sugestões MRP + requisições abertas + histórico de compras); tela é frontend. |
| Histórico de movimentações de compra | `CPDC0403` | Implementado: `GET /api/procurement/purchase-movements` (solicitado/recebido/cancelado/aberto, preço e datas), filtrável por fornecedor/item. |
| Relatório de pedidos de compra | `FPDC0250` | Dados disponíveis por consulta/histórico de compras; renderização é frontend/export. |
| Cadastro de solicitação de compra | `FPDC0201` | Implementado. |
| Cancelamento de solicitação de compra | `FPDC0202` | Backend completo (status/atendimento da solicitação); rotina dedicada é frontend. |
| Consulta de solicitação de compra | `CPDC0401` | Implementado via consulta de solicitação. |
| Relatório de solicitação de compra | `FPDC0251` | Dados disponíveis pela consulta de solicitação; renderização é frontend/export. |
| Cadastro do aviso de recebimento | `FAVR0200` | Implementado normalizado (`receiving_notices` + itens) com doca, agenda e status. |
| Cancelamento do aviso de recebimento | `FAVR0201` | Implementado via `PATCH .../status` para `CANCELLED`. |
| Desbloqueio do recebimento | `FAVR0204` | Implementado: `blocked` + status `BLOCKED`/`RELEASED` no aviso normalizado. |
| Histórico de divergências do recebimento | `FAVR0300` | Implementado normalizado (`receiving_divergences`) com tipo, quantidades, preço e resolução, consultável por fornecedor. |
| Manutenção de notas fiscais de entrada | `FREC0200` | Parcial: importação/criação/aprovação de entrada existem. |
| Geração de contra nota produtor rural | `FREC0201` | Não implementado. |
| Confirmação de NF de importação | `FREC0203` | Implementado normalizado (`import_processes`) com câmbio, despesas, rateio (valor/peso/qtd) e **custo nacionalizado por item**. Integração SEFAZ fica com o FocusNFE. |
| Manutenção de dados específicos da NFE | `FREC0255` | Backend com campos fiscais de entrada; manutenção fina/tela é frontend. |
| Roteiro de checklist de recebimento | `FCLR0200` | Implementado como `RECEIVING_CHECKLIST`. |
| Roteiro de inspeção de recebimento | `FINS0200` | Implementado núcleo estruturado: roteiro por item/classificação, vigência, almoxarifado de inspeção, etapas, espécie, forma de apontamento, amostra, instrumentos, norma, faixa e atributos. |
| Ordens de inspeção | `FINS0201` | Implementado núcleo: ordem por origem manual/recebimento/aviso/NF, fornecedor, item, lote, quantidade e status. |
| Apontamentos das inspeções | `FINS0202` | Implementado núcleo: resultados por sequência/amostra, valor/intervalo/atributo/status e aprovação/reprovação. |
| Análise das inspeções | `FINS0203` | Implementado e integrado ao estoque: quantidades conformes, rejeitadas, retrabalho/conserto, aprovadas com restrição, tratamento para avaliação do fornecedor e, com `move_stock`, movimento de quarentena→disponível/retrabalho/bloqueado. |
| Inspeções parciais | `FINS0207` | Implementado no backend (quantidades conforme/rejeitada/retrabalho/restrita por ordem); tela dedicada é frontend. |
| Geração de ordens de inspeção | `FINS0212` | Implementado: o recebimento físico por linha detecta roteiro ativo, recebe no almoxarifado de inspeção e abre a ordem automaticamente (origem `PURCHASE_RECEIPT`). |
| Etiquetas de inspeção | `FINS0304` | Implementado como `RECEIVING_LABEL`. |
| Avaliação de fornecedores | `FAVF0200` | Implementado com IQF auto-calculado a partir de inspeções e atrasos de entrega (`/supplier-scorecards/compute`), além do lançamento manual. |
| Abono de divergências | `FAVF0201` | Implementado como evento/status em `SUPPLIER_EVALUATION`. |
| Dados para IQF | `FAVF0202` | Implementado em `supplier_scorecard_snapshots`. |
| Indicador de homologação | `FAVF0203` | Implementado (`supplier_homologations`): status derivado do IQF por faixas (homologado/condicional/rejeitado) ou manual, com validade. |
| Checklist de avaliação | `FAVF0205` | Implementado como payload de avaliação/checklist. |
| Cadastro de alçada | `FALC0200` | Implementado normalizado (`purchase_approval_limits`) com escopo global/fornecedor/centro de custo/categoria e tetos de auto-aprovação e absoluto. |
| Desbloqueio de pedidos de compra | `FALC0201` | Implementado com bloqueio real: `.../approve` avalia a alçada e bloqueia acima do teto (`alcada_status=B`); `.../authorize` (ADMIN) libera. |
| Contratos de fornecedores | `FCON0200` | Implementado normalizado (`supplier_contracts` + `supplier_contract_items`) com vigência, moeda, índice e preço por item. |
| Cancelamento de itens do contrato | `FCON0202` | Implementado via status do contrato e consumo de saldo por item; cancelamento de linha individual fica como refino. |
| Consulta de contratos | `CCON0400` | Implementado em `/api/procurement/supplier-contracts` com saldo (`remaining_qty`) por item. |
| EDI fornecedor - parâmetros gerais | `FEDS0130` | Implementado (`supplier_edi_messages`) + parâmetros no painel (`domain=SUPPLIER`). |
| Tipos de nota por fornecedor | `FEDS0131` | Implementado: `message_type` tipado (ORDER_CONFIRMATION/SHIP_NOTICE/INVOICE/ORDER). |
| Recebimento de arquivos EDI | `FEDS0251` | Implementado: mensagem INBOUND com linhas confirmadas e payload; parser de layout VAN fica como integração futura. |
| Geração de NFE por EDI | `FEDS0252` | Parcial: mensagem estruturada pronta; a emissão fiscal automática permanece no FocusNFE. |
| Envio de arquivos a fornecedores | `FEDS0253` | Implementado: mensagem OUTBOUND com status `SENT`. |
| Divergências EDI x NFE | `FEDS0300` | Implementado: divergência QTY/PRICE/DATE detectada por linha vs. pedido, com tolerância, e contagem na mensagem. |

### O que o VentureERP já tem

| Rotina | Situação atual |
|---|---|
| Cadastro de fornecedores | Implementado, com defaults de compra e vínculo fiscal. |
| Fornecedor preferencial por item | Implementado. |
| Tabela de preço de compra | Implementado. |
| Solicitação de compra | Implementado. |
| Geração de pedidos por solicitação | Implementado, agrupando por fornecedor. |
| Cotação de compra | Implementado, com preço por fornecedor, seleção e geração de pedido. |
| Pedido de compra | Implementado com capa e itens ricos. |
| Sugestão MRP → pedido de compra | Implementado. |
| Operações de entrada | Implementado. |
| NF-e de entrada com baixa de pedido | Implementado. |
| Recebimento físico por linha do pedido | Implementado nesta task. |

### O que ainda falta ou precisa amadurecer

Todas as lacunas críticas de backend do setor foram fechadas (ver "Status de
fechamento" abaixo). Restam apenas itens de frontend (telas/relatórios dedicados) e
fluxos de baixo valor para o cliente metalúrgico/moveleiro:

| Lacuna | Situação |
|---|---|
| Cotação/pedido de frete próprios (`FCOT0200 FRE`/`FPDC0200 FRE`) | Backend do pedido já tem campos de frete; fluxo próprio pendente (baixo valor). |
| Tela de pedido de serviço (`FPDC0200 SER`) | Requisição de serviço por operação externa existe; tela dedicada é frontend. |
| Contra nota de produtor rural (`FREC0201`) | Fora de escopo (cliente metalúrgico/moveleiro não compra de produtor rural). |
| Parser EDI por layout/VAN (`FEDS0251/0252`) | Mensagem EDI estruturada e detecção de divergência prontas; parser de arquivo VAN e emissão fiscal automática ficam como integração futura. |

## Ordem de criação/melhoria em Suprimentos

1. Recebimento físico por linha de pedido, com movimento de estoque e status parcial/recebido.
2. Inspeção de recebimento (`FINS0200`, `FINS0201`, `FINS0202`, `FINS0203`, `FINS0212`) integrada ao recebimento físico, com estoque em quarentena/inspeção.
3. Divergências de recebimento e desbloqueio (`FAVR0204`, `FAVR0300`), com aceite parcial, rejeição, devolução e pendência fiscal/comercial.
4. Aviso de recebimento (`FAVR0200`, `FAVR0201`) e agenda de doca.
5. Avaliação de fornecedor / IQF (`FAVF0200` a `FAVF0205`) por entrega, qualidade e divergência.
6. Alçada de valores (`FALC0200`, `FALC0201`) para solicitação, cotação e pedido.
7. Contratos de fornecedores (`FCON0200`, `FCON0202`, `CCON0400`) e preço por contrato.
8. Etiquetas/checklist de recebimento (`FCLR0200`, `FINS0304`) e rastreabilidade por volume/lote.
9. EDI fornecedores (`FEDS0130`, `FEDS0131`, `FEDS0251`, `FEDS0252`, `FEDS0253`, `FEDS0300`) e confirmação automática.
10. Compras de importação (`FREC0203` e família `FIMP/CIMP`) com custos adicionais e nacionalização.

## Task atual

Implementados:

- recebimento físico por linha de pedido;
- registros operacionais de inspeção/quarentena, aviso/divergência, checklist,
  etiquetas, contratos, alçadas, EDI, importação e avaliação de fornecedor;
- núcleo estruturado de inspeção de recebimento com roteiro, ordem, apontamento e
  análise;
- disposição de inspeção com transferência de estoque da quarentena para depósito
  disponível/bloqueado;
- **fechamento do ciclo inspeção→estoque no caminho estruturado**: a análise
  (`.../analysis` com `move_stock`) movimenta o estoque para fora do almoxarifado de
  inspeção — conforme/restrita para disponível, retrabalho para o almoxarifado de
  retrabalho e rejeitada para bloqueado/devolução — e devolve as movimentações na
  resposta. Antes, só a disposição genérica sobre `procurement_records` movia estoque;
- snapshots de IQF por fornecedor.

Também foi identificado e corrigido um bug no abatimento legado por `item_code`: em
pedidos com o mesmo item em várias linhas, a baixa por NF-e podia aplicar a mesma
quantidade em mais de uma linha. A regra agora distribui a quantidade pelo saldo das
linhas em vez de replicar integralmente.

## Concluído na evolução de governança (migration `000184`)

- Inspeção automática a partir do recebimento físico por linha (`FINS0212`).
- IQF auto-calculado de qualidade e entrega a partir de dados reais (`FAVF0200`).
- Alçada de valores normalizada com bloqueio/autorização real do pedido
  (`FALC0200`/`FALC0201`).
- Contratos de fornecedores normalizados com itens, saldo e consumo atômico
  (`FCON0200`/`CCON0400`).
- Histórico consolidado de movimentações de compra (`CPDC0403`).

## Status de fechamento de Suprimentos (migration `000185`)

Com esta migração o setor está **funcionalmente fechado no backend** para o cliente
metalúrgico/moveleiro:

- **Aviso de recebimento + divergências (FAVR)** normalizados: `receiving_notices`
  (doca, agenda, status), itens e `receiving_divergences` (tipo, quantidades, preço,
  resolução), consultáveis por fornecedor.
- **EDI de fornecedores (FEDS)** estruturado: mensagens inbound/outbound com linhas
  confirmadas e **detecção de divergência QTY/PRICE/DATE por linha** vs. pedido.
- **Importação/nacionalização (FREC0203/FIMP)**: câmbio, despesas com/sem custo,
  rateio por valor/peso/quantidade e **custo nacionalizado por item**.
- **Parâmetros de suprimentos (FUTL0125 *)**: painel único por domínio/chave/valor
  tipado, cobrindo PRC/PDC/COT/SLC/AVR/INSP/AVF/CTRA/FOR/NFE.
- **Homologação de fornecedor (FAVF0203)**: status derivado do IQF por faixas.
- **Geração de itens por fornecedor (FFOR0204)** a partir do histórico de compras.

## Próximas melhorias a validar antes de criar (opcionais / futuras)

- Frequência/skip de inspeção por item, classificação e fornecedor.
- Consumo de contrato automático ao firmar a linha do pedido com `contract_code`
  (hoje o consumo é por endpoint explícito) e reajuste por índice.
- Alçada por nível hierárquico de aprovador (hoje o desbloqueio é por papel ADMIN).
- Parser EDI por layout de VAN e emissão fiscal automática por EDI.
- Fluxos próprios de cotação/pedido de frete
  (`FCOT0200 FRE`/`FPDC0200 FRE`) como fluxos próprios.
