# Módulo Fiscal & Financeiro

### Apresentação para o setor Financeiro e Contabilidade

---

## Sumário

1. [O que este módulo entrega](#1-o-que-este-módulo-entrega)
2. [Como a empresa é configurada](#2-como-a-empresa-é-configurada)
3. [Cálculo automático de impostos](#3-cálculo-automático-de-impostos)
4. [Vendas — Nota Fiscal de Saída (NF-e)](#4-vendas--nota-fiscal-de-saída-nf-e)
5. [Compras — Nota Fiscal de Entrada e Frete (CT-e)](#5-compras--nota-fiscal-de-entrada-e-frete-ct-e)
6. [Contas a Pagar e a Receber](#6-contas-a-pagar-e-a-receber)
7. [Fluxo de Caixa e Conciliação Bancária](#7-fluxo-de-caixa-e-conciliação-bancária)
8. [Apuração de Impostos](#8-apuração-de-impostos)
9. [Obrigações acessórias e SPED](#9-obrigações-acessórias-e-sped)
10. [Contabilidade](#10-contabilidade)
11. [Relatórios gerenciais](#11-relatórios-gerenciais)
12. [Cadastros de apoio fiscal](#12-cadastros-de-apoio-fiscal)
13. [O que o sistema ainda não faz](#13-o-que-o-sistema-ainda-não-faz)
14. [Glossário rápido](#14-glossário-rápido)

---

## 1. O que este módulo entrega

O módulo Fiscal & Financeiro cuida de **todo o ciclo do dinheiro e dos documentos fiscais da empresa**, do momento em que uma venda é faturada até a entrega das obrigações ao governo.

Na prática, ele resolve quatro grandes blocos de trabalho:

| Bloco | O que faz |
|---|---|
| **Documentos fiscais** | Emite, cancela e corrige Notas Fiscais Eletrônicas (NF-e) junto à SEFAZ; registra notas de compra e conhecimentos de frete (CT-e). |
| **Cálculo de impostos** | Calcula sozinho ICMS, IPI, PIS, COFINS, DIFAL e FCP em cada venda, seguindo a legislação. |
| **Financeiro** | Controla contas a pagar, contas a receber, fluxo de caixa, saldos bancários e conciliação do extrato. |
| **Fiscal/contábil** | Faz a apuração mensal de impostos e gera os arquivos para o SPED Fiscal, SPED Contábil (ECD) e demais obrigações. |

**Pontos importantes para a contabilidade:**

- A emissão de NF-e é feita por integração oficial e homologada com a SEFAZ (via plataforma **Focus NF-e**), tanto em ambiente de **homologação** (testes) quanto de **produção**.
- O regime tributário padrão do sistema é o **Lucro Real** (PIS/COFINS não-cumulativo), com suporte também a **Lucro Presumido** e **Simples Nacional**.
- Todos os valores monetários são tratados com precisão decimal, evitando os erros de arredondamento de planilhas.

---

## 2. Como a empresa é configurada

Antes de emitir qualquer documento, a empresa é cadastrada uma única vez. Esses dados alimentam automaticamente todas as notas e cálculos.

**O que é configurado:**

- **Identificação:** CNPJ, razão social, inscrição estadual e regime tributário (Simples, Presumido ou Real).
- **Endereço completo:** logradouro, número, bairro, município (com código IBGE), CEP e UF — obrigatórios para a SEFAZ aceitar a nota.
- **Parâmetros de ICMS:** alíquota interna do estado e percentual de diferimento parcial.
- **Integração SEFAZ:** ambiente (homologação ou produção) e credencial de emissão.
- **Regras financeiras:** percentual de juros ao mês, multa por atraso e o dia de vencimento de cada imposto (ICMS, IPI, PIS/COFINS).

> **Por que isso importa:** alterar, por exemplo, o percentual de diferimento na configuração reflete **imediatamente** no cálculo da próxima nota. Não é preciso mexer em cada documento.

---

## 3. Cálculo automático de impostos

Este é o coração do módulo. Ao registrar uma venda, o sistema **calcula todos os tributos sozinho**, identificando o cenário tributário correto a partir das características da operação (estado de origem e destino, se o cliente é contribuinte, origem da mercadoria, NCM do produto).

### 3.1 ICMS — os cenários reconhecidos automaticamente

| Situação da venda | Como o sistema trata |
|---|---|
| **Dentro do estado, para contribuinte** | Alíquota interna + **diferimento parcial** (CST 51), conforme o percentual configurado. |
| **Dentro do estado, para não-contribuinte** | Base inclui o IPI, tributação cheia (CST 00), sem diferimento. |
| **Interestadual, para contribuinte** | Alíquota da tabela interestadual, base sem IPI, sem DIFAL. |
| **Interestadual, para não-contribuinte ou pessoa física** | Base inclui IPI, calcula o **DIFAL** (partilha EC 87/2015) e o **FCP** quando o estado destino exige. |
| **Mercadoria importada** | Força a alíquota interestadual de **4%** (Resolução do Senado 13/2012), independente do destino. |

O sistema identifica sozinho se o cliente é contribuinte (pela inscrição estadual) ou não, e aplica a regra correspondente — inclusive a partilha do DIFAL entre os estados.

### 3.2 PIS e COFINS (Lucro Real)

- Alíquotas padrão: **PIS 1,65%** e **COFINS 7,6%** (regime não-cumulativo).
- Produtos com tributação específica têm suas alíquotas próprias respeitadas (a partir de uma tabela por NCM).
- Produtos **monofásicos** são reconhecidos e têm o imposto zerado no documento, como exige a lei.

### 3.3 IPI

- A alíquota é definida por NCM do produto.
- O sistema sabe que, para **contribuintes**, o IPI **não** entra na base do ICMS; já para **não-contribuintes**, ele **entra** — aplicando a regra automaticamente.

### 3.4 Tabelas tributárias mantidas pela contabilidade

Para que o cálculo acompanhe a legislação, a contabilidade mantém três tabelas, atualizáveis a qualquer momento:

- **Tabela de NCM** — alíquotas de IPI, PIS e COFINS por produto.
- **ICMS interestadual** — alíquota para cada combinação estado de origem → destino.
- **ICMS interno e FCP** — alíquota interna e o adicional de FCP de cada estado.

> Quando uma alíquota muda na lei, basta atualizar a tabela: as próximas notas já saem com o valor novo.

---

## 4. Vendas — Nota Fiscal de Saída (NF-e)

O fluxo de uma venda segue quatro etapas claras:

```
1. Criar a nota   →  2. Autorizar na SEFAZ  →  3. (se preciso) Corrigir/Cancelar
   (rascunho,          (vira "autorizada",
    calcula impostos)   gera os efeitos abaixo)
```

### 4.1 Criar a nota (rascunho)

Ao criar a nota, o sistema já **calcula automaticamente** ICMS, IPI, PIS, COFINS e DIFAL de cada item e da nota inteira. A nota fica em **rascunho**, podendo ser revisada antes do envio.

### 4.2 Autorizar na SEFAZ

Ao autorizar, a nota é enviada à SEFAZ. Quando aprovada, o sistema **encadeia automaticamente uma série de efeitos**, eliminando trabalho manual:

1. Grava a **chave de acesso** e o **protocolo** de autorização.
2. Cria automaticamente a **Conta a Receber** vinculada à nota (vencimento padrão em 30 dias).
3. **Baixa o estoque** do produto vendido.
4. **Consome as reservas** do pedido de venda.
5. Marca o **pedido de venda como Faturado**.

> Tudo isso acontece numa única operação: faturou, o financeiro e o estoque já estão atualizados.

### 4.3 Eventos da nota após autorizada

| Evento | Para que serve |
|---|---|
| **Carta de Correção (CC-e)** | Corrige dados como razão social, natureza da operação, CFOP ou descrição. **Não** corrige valores, quantidades ou impostos. |
| **Cancelamento** | Anula a nota dentro do prazo legal, com justificativa. |
| **Manifestação do Destinatário** | Registra ciência, confirmação, desconhecimento ou operação não realizada de notas recebidas. |
| **Inutilização de numeração** | Invalida junto à SEFAZ uma faixa de números que não foram usados. |

A qualquer momento é possível **consultar o status atualizado** de uma nota diretamente na SEFAZ (autorizada, rejeitada, cancelada, em processamento) e listar todas as notas e correções emitidas.

---

## 5. Compras — Nota Fiscal de Entrada e Frete (CT-e)

### 5.1 Notas de entrada (compras)

As notas de compra podem ser registradas de **três formas**, da mais manual à totalmente automática:

| Forma | Como funciona |
|---|---|
| **Lançamento manual** | A contabilidade digita os dados da nota e seus impostos. |
| **Importação do XML** | O sistema lê o arquivo XML da nota e preenche tudo sozinho. |
| **Importação pela chave de acesso** | Informando apenas os 44 dígitos da chave, o sistema busca a nota na SEFAZ, baixa, processa e já movimenta o estoque. |

Em todos os casos, ao aprovar a entrada o sistema:

- Cria automaticamente a **Conta a Pagar** ao fornecedor.
- **Atualiza o estoque** e recalcula o **custo médio ponderado** de cada item.
- **Baixa o pedido de compra** correspondente, registrando o que foi recebido (total ou parcial).
- **Vincula a nota ao fornecedor cadastrado** pelo CNPJ do emitente, habilitando regras fiscais e financeiras específicas daquele fornecedor.

Os impostos da nota de entrada (ICMS, IPI, PIS, COFINS) são **informados pelo fornecedor** e registrados como **crédito** quando aplicável — alimentando a apuração mensal.

### 5.2 Conhecimento de Transporte (CT-e)

O CT-e é registrado para **custear o frete** e vinculá-lo à nota de entrada correspondente. O frete pode ser **rateado por valor ou por peso** entre as notas.

> **Observação para a contabilidade:** o CT-e é registrado internamente para custo e apuração; ele **não** é transmitido à SEFAZ pelo sistema (apenas a NF-e de saída usa a integração de autorização).

---

## 6. Contas a Pagar e a Receber

### 6.1 Cadastros de base

Antes de operar, ficam cadastrados: **contas bancárias**, **condições de pagamento** (à vista, 30/60/90 etc.), **plano de contas** e **centros de custo**. São esses cadastros que organizam e classificam cada lançamento.

### 6.2 Contas a Pagar

- Geradas automaticamente a partir das notas de entrada, ou lançadas manualmente.
- Possuem **workflow de aprovação** (aprovar ou rejeitar com motivo).
- A **baixa (pagamento)** é uma operação única e segura: dá baixa no título, lança no fluxo de caixa e atualiza o saldo da conta bancária ao mesmo tempo.
- Suporta **pagamento parcial** (mantém o saldo restante em aberto).
- Pode ser filtrada por status, período ou fornecedor, e gera o **relatório de aging** (o que está a vencer e o que está vencido, por faixa de atraso).

### 6.3 Contas a Receber

- Criadas automaticamente ao autorizar uma NF-e de saída, ou manualmente.
- A **baixa (recebimento)** também é uma operação única: atualiza o saldo bancário, lança no fluxo de caixa e dá baixa no título.
- Suporta **recebimento parcial** (gera automaticamente um novo título com o restante).
- Também possui **aging** por faixa de vencimento.

### 6.4 Geração de boletos (CNAB 240)

O sistema gera o arquivo de **remessa CNAB 240** (padrão FEBRABAN) para envio dos boletos ao banco.

> ⚠️ O arquivo segue o **layout-padrão FEBRABAN**. Como cada banco (Itaú, Bradesco, Santander, BB, Caixa) tem particularidades, o layout deve ser **homologado com o banco** antes do uso em produção.

---

## 7. Fluxo de Caixa e Conciliação Bancária

### 7.1 Fluxo de caixa

| Visão | O que mostra |
|---|---|
| **Fluxo realizado** | Entradas e saídas já efetivadas, por período. |
| **Fluxo projetado** | Projeção futura, baseada em contas a pagar e a receber ainda em aberto. |
| **Saldo das contas** | Saldo atual de cada conta bancária. |

### 7.2 Conciliação bancária (extrato OFX)

A empresa importa o **extrato bancário em formato OFX** (aceita os formatos usados pela maioria dos bancos brasileiros) e o sistema:

- **Evita duplicidade** — transações já importadas são reconhecidas e ignoradas.
- **Concilia automaticamente** — casa cada lançamento do extrato com o fluxo de caixa quando o valor bate (tolerância de centavos) e a data é próxima.
- Ao final, informa quantas transações foram **importadas, duplicadas e conciliadas**.

---

## 8. Apuração de Impostos

### 8.1 Apuração mensal (ICMS, IPI, PIS, COFINS)

A cada competência (mês), o sistema **consolida automaticamente** os impostos das notas de saída e de entrada, calculando o **saldo a recolher ou a compensar**:

| Imposto | Saídas (débito) | Entradas (crédito) | Saldo |
|---|---|---|---|
| ICMS | total das vendas | crédito das compras | a recolher / a compensar |
| IPI | total das vendas | crédito das compras | a recolher / a compensar |
| PIS | total das vendas | crédito das compras | a recolher / a compensar |
| COFINS | total das vendas | crédito das compras | a recolher / a compensar |

### 8.2 Apuração do Simples Nacional

Para empresas no Simples, há registro da apuração mensal por **anexo** (I a VI), com receita interna/externa, folha de pagamento, receita bruta dos últimos 12 meses, alíquotas nominal e efetiva e o valor recolhido.

### 8.3 ICMS-ST — Restituição, Ressarcimento e Complementação

Módulo para registrar e gerar pedidos de **restituição de ICMS-ST** (conforme a decisão do STF RE 593.849/MG), cobrindo os registros do SPED Fiscal correspondentes (C180/C181/C185/C186/1250/1251).

### 8.4 Notas especiais e ajustes de apuração

Suporte a **notas complementares** (de base/alíquota) e **notas de ajuste**, que podem gerar automaticamente a linha de ajuste na apuração de ICMS.

---

## 9. Obrigações acessórias e SPED

| Obrigação | O que o sistema oferece |
|---|---|
| **SPED Fiscal (EFD ICMS/IPI)** | Cadastros e estruturas que alimentam os blocos C e E: códigos de ajuste de apuração de ICMS por UF (Tabelas 5.1.1, 5.2, 5.3, 5.6, 5.7), linhas de apuração (bloco E), lançamentos resumo por período/UF/CFOP e seus adicionais (C197, processos judiciais). |
| **SPED Contábil (ECD)** | Geração completa do arquivo da Escrituração Contábil Digital (Blocos 0, I, J, K, 9), pronto para transmissão via PVA. |
| **DAPI** | Cadastro de motivos de transferência usados na DAPI. |
| **IBPT / Lei da Transparência** | Importa a tabela oficial IBPT por estado e calcula a **carga tributária aproximada** por produto (Lei 12.741/2012), exigida no cupom/nota. |

---

## 10. Contabilidade

O módulo contábil cobre a escrituração completa:

- **Plano de contas** e **contas contábeis** (sintéticas e analíticas).
- **Lançamentos contábeis** por período (débito/crédito).
- **Balancete** — agrega os lançamentos do período por conta, com totais e o indicador de **partidas dobradas** (confere se o total de débitos é igual ao de créditos).
- **Demonstrativos** (DRE, Balanço Patrimonial etc.).
- **Geração do arquivo SPED ECD** para entrega ao fisco.

---

## 11. Relatórios gerenciais

O sistema entrega um conjunto amplo de relatórios para a gestão e a contabilidade:

| Relatório | O que mostra |
|---|---|
| **Livro de Entradas / Saídas** | Todas as notas do período com seus impostos. |
| **Impostos das Saídas / Entradas** | Detalhamento de tributos por CFOP (débitos) e créditos aproveitados. |
| **DRE** | Demonstração do resultado com **CMV real**, baseado no custo médio ponderado do estoque. |
| **Aging de Receber / Pagar (detalhado)** | Cada título em aberto, com dias de atraso, cliente ou fornecedor. |
| **Extrato por Fornecedor / Cliente** | Histórico de contas de um parceiro específico. |
| **Produtos Vendidos** | Quantidade, receita, custo e **margem bruta** por produto. |
| **Produtos Produzidos** | Produção do período com custo real da ordem. |
| **Histórico de Custos** | Evolução do custo médio de cada item ao longo do tempo. |
| **Ficha Técnica com Custo** | Estrutura do produto com custo de cada componente. |
| **Curva ABC de Clientes / Produtos** | Ranking por receita, classificando em A (até 80%), B (80–95%) e C (acima de 95%). |
| **Compras no Período** | Resumo por fornecedor e produto, com impostos recuperáveis. |

---

## 12. Cadastros de apoio fiscal

Para sustentar a parametrização fiscal, o sistema mantém uma série de cadastros que a contabilidade reconhece do dia a dia:

- **Naturezas de Operação (CFOP)** — com classificação por direção (entrada/saída) e finalidade.
- **Dispositivos legais** — base legal de isenções e benefícios de ICMS, IPI, PIS e COFINS.
- **Parâmetros de ICMS/IPI por NCM/item/UF** — alíquotas e CSTs por operação.
- **Redução / Substituição / Diferimento de ICMS** — parametrização avançada com **hierarquia de busca em 11 níveis** (regra preferencial, por item, por cliente, por classificação etc.), permitindo regras específicas por produto, cliente ou fornecedor.
- **Classificações fiscais de mercadorias** — NCM, CEST, alíquotas e CSTs por produto, com atributos de exportação (SISCOMEX) e descrição por idioma.
- **Tipos de Operação de Entrada** — com validação automática de UF × natureza da operação.
- **Tipos de movimento de estoque**, **tabelas de preço de venda**, **países e UFs** (as 27 UFs já vêm pré-cadastradas) e **classificações hierárquicas de itens**.

---

## 13. Funcionalidades entregues (antes limitações)

Todos os itens que antes constavam como "o que o sistema ainda não faz" **foram
implementados**. O quadro abaixo resume o que passou a existir:

| Item | O que passou a existir |
|---|---|
| **Substituição Tributária (MVA/ST)** | O motor **calcula automaticamente** o ICMS-ST quando o item informa a MVA (inclusive MVA ajustada e redução de base de ST). Apura a base de ST, a alíquota interna do destino e o ICMS-ST a recolher, ajusta o CST (10/70), soma o ST ao total da nota e envia tudo na emissão. |
| **Nota Fiscal de Serviços (NFS-e)** | Novo módulo completo (modelo ABRASF): cadastro do serviço e do tomador, **cálculo do ISS** e do valor líquido, **emissão na prefeitura**, consulta, cancelamento e listagem. |
| **CT-e — autorização na SEFAZ** | Além do registro para custo, o CT-e agora pode ser **transmitido e autorizado na SEFAZ**, com os dados de remetente, destinatário, tomador, modal e municípios; o sistema guarda a chave e o protocolo. |
| **Adiantamentos a fornecedores/clientes** | Novo controle de **adiantamentos**: registra o pagamento/recebimento antecipado (com movimento de caixa), acompanha o saldo e **aplica o adiantamento** sobre as contas a pagar/receber, quitando-as total ou parcialmente. |
| **DANFE e XML** | O sistema **disponibiliza os links** do DANFE (PDF) e do XML da nota autorizada na própria consulta de status — prontos para download, envio ao cliente e guarda. |
| **Boletos por banco (CNAB 240)** | O arquivo de remessa usa um **perfil por banco** (Itaú, Bradesco, Santander, Banco do Brasil e Caixa), ajustando carteira, espécie do título e versões de layout, além de gravar o código do banco nos totalizadores. |
| **Faturamento por carga** | A expedição monta a carga com romaneios e o faturamento gera a NF-e diretamente dessa carga, vinculando nota, romaneio e carga sem redigitação. |
| **Cupom/NFC-e rastreável** | NF-e gerada a partir de cupom fiscal, NFC-e ou CF-e guarda número do cupom, data e série/equipamento de origem para consulta e auditoria. |

> **Importante:** os documentos fiscais eletrônicos (NF-e, NFS-e, CT-e) e os boletos
> **devem sempre ser homologados** com a SEFAZ, com a prefeitura e com o banco antes do
> uso em produção — isso depende das credenciais e do cadastro de cada empresa.

---

## 14. Glossário rápido

| Termo | Significado |
|---|---|
| **NF-e** | Nota Fiscal Eletrônica (modelo 55), de venda (saída) ou compra (entrada). |
| **CT-e** | Conhecimento de Transporte Eletrônico — documento do frete. |
| **CC-e** | Carta de Correção Eletrônica — corrige dados não-fiscais de uma nota autorizada. |
| **DIFAL** | Diferencial de alíquota do ICMS em vendas interestaduais a não-contribuintes. |
| **FCP** | Fundo de Combate à Pobreza — adicional de ICMS de alguns estados. |
| **Diferimento** | Adiamento do recolhimento de parte do ICMS para uma etapa posterior. |
| **CST** | Código de Situação Tributária — define como o imposto incide em cada item. |
| **NCM** | Nomenclatura Comum do Mercosul — classificação fiscal do produto. |
| **CFOP** | Código Fiscal de Operações — identifica a natureza da operação. |
| **Custo médio ponderado** | Método de custeio do estoque que recalcula o custo a cada entrada. |
| **Aging** | Relatório que agrupa títulos por faixa de vencimento/atraso. |
| **SPED** | Sistema Público de Escrituração Digital — entrega das obrigações ao fisco. |
| **ECD** | Escrituração Contábil Digital (SPED Contábil). |
| **CMV** | Custo da Mercadoria Vendida. |
| **DRE** | Demonstração do Resultado do Exercício. |
| **SEFAZ** | Secretaria da Fazenda — órgão que autoriza as notas fiscais. |

---

> **Resumo em uma frase:** o módulo automatiza o cálculo dos tributos, a emissão dos documentos fiscais, o controle financeiro e a geração das obrigações — reduzindo o trabalho manual e o risco de erro, e mantendo o financeiro e a contabilidade sempre conciliados.
