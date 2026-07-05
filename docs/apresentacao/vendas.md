# VentureERP — Vendas e Expedição

### Apresentação para o setor Comercial e Logística

---

O módulo de Vendas é a **porta de entrada** do fluxo do ERP: é o pedido do cliente que dispara todo o planejamento, a produção e o faturamento. A Expedição é a **porta de saída**: organiza a separação, a conferência e o despacho ao cliente. Entre as duas pontas, o sistema ajuda a **prometer e cumprir prazos**.

---

## Sumário

1. [Pedido de venda](#1-pedido-de-venda)
2. [Itens do pedido](#2-itens-do-pedido)
3. [Orçamentos](#3-orçamentos)
4. [Precificação](#4-precificação)
5. [Crédito e bloqueio](#5-crédito-e-bloqueio)
6. [Do pedido ao planejamento](#6-do-pedido-ao-planejamento)
7. [Divisão de vendas](#7-divisão-de-vendas)
8. [Promessa de entrega (prazos confiáveis)](#8-promessa-de-entrega-prazos-confiáveis)
9. [Reprogramação de entrega](#9-reprogramação-de-entrega)
10. [Representantes](#10-representantes)
11. [Metas de vendas](#11-metas-de-vendas)
12. [Previsão de vendas](#12-previsão-de-vendas)
13. [Assistencia Tecnica](#13-assistencia-tecnica)
14. [Faturamento](#14-faturamento)
15. [Expedição / Romaneio](#15-expedição--romaneio)
16. [Glossário rápido](#16-glossário-rápido)

---

## 1. Pedido de venda

O pedido de venda registra o que o cliente comprou e coloca a venda dentro do
fluxo operacional. No **cabeçalho** ficam cliente, condição de pagamento,
representante/divisão, datas, dados fiscais, transportadora, frete, volumes,
projeto e o **plano de produção** ao qual o pedido pertence.

Os preços vêm da **tabela de vendas** do cliente, e o tipo de nota e os impostos já ficam definidos pelo cadastro — o pedido sai consistente desde o início.

Antes de seguir para produção, expedição ou faturamento, o pedido pode passar por
análise comercial, análise financeira, liberação de bloqueios e conferência
logística. Essas etapas registram histórico de decisão e permitem consultar a
carteira por pendência, bloqueio, atraso, conferência e status.

**Ciclo de vida:**

| Status | Significado |
|---|---|
| **Rascunho (R)** | Em montagem, ainda não confirmado |
| **Confirmado / Pedido (P)** | Pedido firmado — vira demanda para o planejamento |
| **Bloqueado** | Travado (por crédito ou manualmente) |
| **Faturado (F)** | Nota fiscal emitida |
| **Cancelado** | Pedido cancelado |

O sistema permite **criar, listar, consultar, atualizar, cancelar**, **analisar**,
**liberar**, **conferir**, registrar **motivo de atraso** e **mudar o status** do
pedido. A gestão acompanha totais da carteira, pedidos bloqueados, pendências de
análise, pendências de conferência e pedidos em atraso.

---

## 2. Itens do pedido

Cada linha do pedido é um **item**: produto, quantidade e **data de entrega** própria (um pedido pode ter entregas em datas diferentes). Os itens podem ser **adicionados, listados, atualizados e cancelados** individualmente, sem mexer no restante do pedido.

---

## 3. Orçamentos

O orçamento registra uma negociação antes de virar pedido de venda. Ele serve para
formalizar propostas, controlar condições negociadas e preservar o histórico de
oportunidades comerciais que ainda não viraram pedido.

Cada orçamento guarda cliente, validade, tabela de preço, condição de pagamento,
representante, itens, quantidades, descontos, acréscimos, frete, redespacho,
seguro, retenções, ordem de compra, comissão, endereço do consumidor, liberação
comercial, valor total e probabilidade de fechamento.

O comercial usa a rotina para consultar a carteira de propostas, acompanhar
status, cancelar oportunidades perdidas, descancelar quando houver reversão,
registrar atendimento manual e gerar uma visão gerencial com total bruto, total
líquido, retenções, propostas abertas, atendidas, canceladas, expiradas e valor
ponderado pela chance de fechamento.

Quando o cliente aprova, o orçamento é convertido para pedido de venda. A conversão
copia apenas o saldo aberto dos itens e cria um pedido real; a partir daí entram as
regras de crédito, reserva de estoque, MRP e faturamento do fluxo de pedido.

O orçamento também possui o indicador de NFC-e. Esse campo não emite cupom fiscal
por si só; ele apenas leva para o pedido de venda a intenção de que aquela venda
seja tratada como cupom fiscal eletrônico no fluxo fiscal/faturamento.

**Ciclo de vida do orçamento:**

| Status | Significado |
|---|---|
| **R** | Rascunho |
| **P** | Pedido originado em canal web/lojas |
| **A** | Pedido em análise |
| **OA** | Orçamento em análise |
| **F** | Pedido confirmado no ERP |
| **OF** | Orçamento confirmado no ERP |
| **ATTENDED** | Orçamento atendido ou convertido em pedido |
| **EXPIRED** | Perdeu validade |
| **CANCELLED** | Encerrado por perda/desistência |

---

## 4. Precificação

A área comercial mantém **tabelas de venda** com vigência, tipo, composição,
tolerância mínima/máxima e casas decimais. Cada tabela possui preços por item, com
situação ativa/promocional/inativa, fórmula, unidade de medida, opção de bloqueio e
controle de preço abaixo de um centavo.

Além de consultar o preço vigente de um item, o sistema calcula **preço sugerido** a
partir de custo base, markup, margem desejada, impostos, frete, comissão, descontos
e outras despesas.

Também há **políticas de formação de preço**: elas guardam prioridade, sequência,
vigência, fonte de custo (custo-padrão, custo de compra, custo médio/último do
estoque ou custo informado), margem mínima/máxima/ideal, incidências comerciais e
percentuais. Com isso o comercial consegue simular preço, reprecificar itens da
tabela em lote e manter histórico do preço antigo, novo preço, custo usado e
política aplicada.

---

## 5. Crédito e bloqueio

O cliente tem um **limite de crédito**. Um pedido pode ser **bloqueado** (automaticamente ao ultrapassar o limite, ou manualmente por decisão comercial/financeira) e depois **desbloqueado**. Enquanto bloqueado, o pedido **não avança** — protegendo a empresa de vender para quem está inadimplente ou no limite.

---

## 6. Do pedido ao planejamento

Quando o pedido é **confirmado**, o sistema cria automaticamente a **demanda** de cada item para o planejamento (MRP). Esse é o elo que liga a venda à fábrica: a partir daí o sistema sabe o que precisa comprar e produzir para atender aquele cliente no prazo.

> Reconfirmar o mesmo pedido **não duplica** a demanda — o sistema é seguro contra repetição.

---

## 7. Divisão de vendas

A **divisão de vendas** organiza a área comercial (equipes, regiões ou unidades de negócio). Cada pedido pode ser associado a uma divisão, o que permite **medir resultado por equipe/segmento** e aplicar regras comerciais específicas. As divisões podem ser criadas, listadas, consultadas, atualizadas e excluídas.

---

## 8. Promessa de entrega (prazos confiáveis)

O sistema ajuda a **prometer prazos realistas**, em vez de "chutar" uma data:

| Recurso | O que faz |
|---|---|
| **Parâmetros de promessa de entrega** | Regras gerais de como a data prometida é calculada |
| **Calendário de promessa por item** | Define dias úteis ou bloqueados por item/variante |
| **Ocupação por tanque/setor** | Mostra quanto da capacidade diária já está comprometida |
| **Reserva comercial** | Guarda capacidade por alguns dias enquanto a venda é negociada |
| **Reprogramação em lote** | Ajusta datas de pedidos/itens sem alterar datas firmes |

Com isso, a data de entrega informada ao cliente considera **estoque disponível,
capacidade produtiva e calendário operacional**. A venda consegue reservar
capacidade antes do pedido definitivo, e a fábrica enxerga a ocupação prevista
por dia. Datas firmes são protegidas para evitar mudanças acidentais.

---

## 9. Reprogramação de entrega

Quando uma data precisa mudar (atraso de matéria-prima, mudança de prioridade), o sistema registra a **reprogramação de entrega** vinculada ao pedido. Assim fica o **histórico** das remarcações (data original × nova data × motivo), com transparência para o comercial e para o cliente. É possível listar as reprogramações de cada pedido.

---

## 9.1. Política comercial

Política comercial é o conjunto de regras que controla a negociação depois que o
preço de tabela já existe. Ela evita que desconto, acréscimo, frete e comissão
sejam decididos de forma informal ou diferente a cada venda.

O sistema permite configurar regras por cliente, segmento, região, tabela de
vendas, condição de pagamento, transportadora, item, linha de produto e
classificação. Cada regra tem validade, prioridade, sequência e faixa de valor ou
quantidade. Assim a empresa consegue aplicar campanhas, acordos comerciais,
condições especiais e comissões sem depender de planilhas paralelas.

### Para que serve

| Necessidade | Como a política resolve |
|---|---|
| Controlar descontos | Define percentuais/valores permitidos por cliente, item, volume ou campanha |
| Aplicar acréscimos | Compensa prazo longo, venda especial, lote pequeno ou condição onerosa |
| Compor frete comercial | Inclui frete negociado antes da expedição/faturamento |
| Prever comissões | Calcula comissão futura por regra comercial |
| Exigir aprovação | Marca negociações que precisam de liberação comercial |
| Evitar exceções indevidas | Bloqueia desconto/acréscimo ou alteração manual por item/classificação |

### Como funciona na venda

1. O vendedor informa cliente, item, quantidade, tabela e condição comercial.
2. O sistema consulta as políticas ativas e vigentes.
3. As regras compatíveis são aplicadas por prioridade e sequência.
4. O resultado mostra valor líquido, descontos, acréscimos, frete e comissão.
5. Se alguma regra exigir aprovação, a venda fica sinalizada para liberação.

Na simulação de uma venda, o sistema retorna:

- desconto total;
- acréscimo total;
- frete comercial;
- comissão futura;
- valor líquido;
- indicação de aprovação obrigatória quando alguma política exigir.

Regras acumuláveis podem somar efeitos. Regras não acumuláveis travam novas regras
do mesmo tipo depois da primeira aplicação, mantendo previsibilidade na negociação.

### Exemplos práticos

**Campanha por volume:** cliente que compra acima de 50 unidades de uma linha recebe
8% de desconto, desde que a condição de pagamento esteja dentro do prazo padrão.

**Acréscimo financeiro:** venda com prazo longo recebe 3% de acréscimo para cobrir
custo financeiro.

**Frete negociado:** entregas em uma região específica incluem valor fixo de frete
comercial, visível antes do faturamento.

**Comissão futura:** representantes têm comissão calculada automaticamente pela
regra aplicável, permitindo previsão antes da emissão da nota.

### Benefício operacional

A política comercial padroniza a negociação, reduz exceções manuais, melhora a
rastreabilidade de aprovações e entrega para gestão uma visão clara de quanto a
empresa concedeu de desconto, quanto adicionou de acréscimo/frete e quanto será
provisionado de comissão.

---

## 10. Representantes

Representantes são a estrutura comercial que conecta clientes, territórios,
pedidos, orçamentos e comissões. O cadastro mantém em um único lugar os dados do
representante, seus prepostos, telefones, e-mails, regiões atendidas, segmentos
de mercado, planos de venda, empresas de atuação e parâmetros de comissão.

### Para que serve

| Necessidade | Como o sistema resolve |
|---|---|
| Organizar carteira | Vincula representantes a clientes, regiões e segmentos |
| Controlar atuação | Separa tipos como externo, interno, gerente ou preposto |
| Dar suporte à venda | Leva o representante para orçamentos e pedidos |
| Calcular comissão | Mantém comissão por empresa e acompanha valor vendido |
| Acompanhar desempenho | Mostra orçamentos, pedidos, clientes atendidos e ticket médio |
| Evitar cadastros incompletos | Centraliza documento, endereço, contatos e situação ativa/inativa |

### Acompanhamento comercial

A ficha de acompanhamento mostra a evolução do representante por cliente,
combinando propostas e pedidos. A gestão consegue ver quanto foi orçado, quanto
virou pedido, qual é o ticket médio, a base de comissão, a comissão futura e a
última movimentação comercial.

### Benefício operacional

O módulo reduz dependência de planilhas de representantes, melhora a análise de
carteira e cria uma base consistente para metas, comissões, políticas comerciais
e relatórios de vendas. Como cada pedido e orçamento pode apontar para um
representante cadastrado, a empresa ganha rastreabilidade desde a negociação até
o faturamento.

---

## 11. Metas de Vendas

Metas de vendas transformam objetivos comerciais em acompanhamento operacional.
A empresa define períodos, metas por representante, metas por grupo comercial e
metas específicas por cliente, item, classificação ou grupo de itens.

### Para que serve

| Necessidade | Como o sistema resolve |
|---|---|
| Definir objetivos | Cria períodos mensais, semanais ou customizados |
| Medir desempenho | Compara previsto x realizado por venda ou faturamento |
| Gerir carteira | Filtra por representante, cliente, região e microrregião |
| Premiar resultados | Controla bônus por meta mínima, provável e ideal |
| Aproveitar excedentes | Registra saldo quando a meta ideal é superada |
| Reduzir planilhas | Centraliza metas, clientes, grupos e relatório no ERP |

### Acompanhamento

O relatório de metas mostra valor previsto, quantidade prevista, realizado,
saldo, percentual de atingimento, bônus e situação da meta. A gestão consegue
avaliar se a carteira está aberta, atingida ou sem alvo definido e agir durante o
período, não apenas depois do fechamento.

### Benefício operacional

Com metas integradas a representantes e pedidos, o comercial ganha uma leitura
contínua de desempenho. Isso cria base para premiações, campanhas, comissões,
políticas comerciais e planejamento de vendas.

---

## 12. Previsão de Vendas

Previsão de vendas antecipa a demanda futura antes de existir pedido confirmado.
Ela permite que o comercial informe uma quantidade mensal por item e o sistema
distribui automaticamente por semanas conforme o calendário industrial. Também é
possível gerar a previsão usando a média do histórico de pedidos ou faturamento,
com índice de projeção para crescimento ou redução.

### Para que serve

| Necessidade | Como o sistema resolve |
|---|---|
| Antecipar demanda | Cadastra previsão mensal por item e distribui por semanas |
| Planejar produção e compras | Alimenta o MRP com demanda prevista |
| Usar histórico | Gera previsões pela média de pedidos ou faturamento |
| Projetar crescimento | Aplica percentual de projeção sobre a média histórica |
| Evitar retrabalho | Atualiza previsões existentes quando permitido |
| Controlar calendário | Bloqueia períodos que não devem receber nova previsão |
| Comparar gestão | Ajuda a confrontar metas, previsões e pedidos reais |

### Como funciona

O usuário pode cadastrar a previsão mensalmente ou gerar períodos futuros a partir
do histórico real do ERP. Na geração automática, o sistema considera apenas pedidos
liberados e sem bloqueio, ou notas fiscais autorizadas, calcula a média do período,
aplica a projeção desejada e grava a previsão por semana. Previsões bloqueadas ou
duplicadas são informadas no resultado para auditoria.

### Benefício operacional

A empresa passa a planejar antes da venda formal acontecer. Isso reduz compras e
produção reativas, melhora análise de capacidade e cria uma ponte entre objetivo
comercial, carteira real e planejamento industrial.

---

## 13. Assistencia Tecnica

Assistencia Tecnica centraliza chamados de garantia, defeitos e devolucoes em um
fluxo rastreavel. O atendimento deixa de depender de controles paralelos: cada
chamado registra cliente, consumidor, item, motivo do defeito, garantia, notas
vinculadas, diagnostico, solucao e os documentos gerados.

### Para que serve

| Necessidade | Como o sistema resolve |
|---|---|
| Controlar garantia | Calcula se o item esta em garantia pela data da nota e prazo informado |
| Padronizar defeitos | Usa grupos e motivos de defeito com regras comerciais |
| Exigir evidencias | Bloqueia atendimento quando falta nota de devolucao/remessa obrigatoria |
| Gerar continuidade operacional | Cria pedido de venda ou ordem de producao de assistencia quando necessario |
| Medir atendimento | Relatorio mostra pendentes, atendidos, fechados, cancelados e tempo medio |

### Como funciona

O atendente abre o chamado, informa os itens afetados e seleciona o motivo do
defeito. O sistema aplica as regras desse motivo: pode exigir complemento,
determinar se gera receita, solicitar nota de retorno e indicar se precisa gerar
pedido ou ordem de assistencia. Antes de atender ou fechar, o ERP valida se tudo
que foi exigido foi cumprido.

### Beneficio operacional

A assistencia passa a conversar com vendas, producao, estoque e fiscal sem perder
o historico do cliente. Isso reduz atendimentos encerrados sem documento, melhora
a analise de defeitos recorrentes e cria base para custos de garantia.

---

## 14. Faturamento

Com o produto disponível, o pedido é faturado. Ao **autorizar a Nota Fiscal de Saída (NF-e)**, o sistema executa em cadeia, automaticamente:

- emite a NF-e junto à SEFAZ, com **todos os impostos calculados**;
- **baixa o estoque** dos produtos;
- **baixa as reservas** do pedido;
- gera a **conta a receber** no financeiro;
- marca o pedido como **faturado**.

> Um único comando fecha venda, fiscal, estoque e financeiro de forma coerente. Detalhes fiscais em `fiscal-financeiro.md`.

---

## 15. Expedição / Romaneio

A expedição organiza a **saída física** da mercadoria por meio do **romaneio** (lista de carregamento):

```
Aberto      → cria o romaneio e adiciona os itens
Separado    → mercadoria separada no estoque
Conferido   → cada item é conferido (item a item e o romaneio todo)
Despachado  → carga liberada para o transporte (só após tudo conferido)
Cancelado   → romaneio cancelado
```

A regra de **só despachar com tudo conferido** evita envio errado ou incompleto ao cliente. É possível criar, listar e consultar romaneios, adicionar e conferir itens, conferir o romaneio inteiro e despachar.

---

## 16. Glossário rápido

| Termo | Significado |
|---|---|
| **Pedido de venda** | O documento da compra do cliente |
| **Orçamento** | Proposta comercial anterior ao pedido |
| **Tabela de vendas** | Cadastro de preços comerciais por item |
| **Política comercial** | Regra de desconto, acréscimo, frete ou comissão aplicada à venda |
| **Representante** | Pessoa ou equipe comercial responsável pela carteira, orçamento, pedido e comissão |
| **Meta de vendas** | Objetivo comercial por período, representante, grupo, cliente, item ou classificação |
| **Previsão de vendas** | Demanda futura estimada por item, variante e semana para planejamento |
| **Demanda** | A necessidade que o pedido confirmado gera para o planejamento |
| **Divisão de vendas** | Agrupamento comercial (equipe/região/unidade) |
| **Reserva** | Estoque separado para um pedido |
| **Promessa de entrega** | Data de entrega calculada com base em estoque e capacidade |
| **Reprogramação** | Remarcação registrada de uma data de entrega |
| **Assistencia Tecnica** | Fluxo de chamados, garantia, defeitos e documentos de atendimento |
| **Romaneio** | Lista de carregamento usada na expedição |

## Novidades (2026-06)

- **Limite de crédito automático:** ao confirmar um pedido, o sistema soma o que o
  cliente já deve (contas a receber em aberto) com os pedidos ainda não faturados.
  Se passar do limite do cliente (ou se ele estiver bloqueado), o pedido é
  **bloqueado automaticamente** — ninguém precisa lembrar de conferir crédito.
- **Reserva automática (promessa firme):** ao confirmar, o sistema **separa o
  estoque disponível** de cada item para o pedido. A consulta de **disponível para
  promessa** mostra, por item, quanto ainda pode ser vendido (saldo menos reservas).

> A versão técnica está em `../dev/visao-geral.md` (§4 Pedidos, §5.3 Expedição) e `../dev/00-fluxo-geral.md`.
