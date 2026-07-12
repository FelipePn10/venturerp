# VentureERP — MRP e Planejamento

### Apresentação para o PCP e a direção

---

O planejamento é o que diferencia um ERP de um simples sistema de notas. A partir dos pedidos e das previsões, o VentureERP responde a três perguntas que normalmente consomem horas de planilha:

> **O que** comprar e fabricar? · **Quanto**? · **Para quando**?

E vai além: verifica se a fábrica **tem capacidade** para cumprir os prazos e organiza a **fila de produção**.

---

## Sumário

1. [O problema que o MRP resolve](#1-o-problema-que-o-mrp-resolve)
2. [De onde vem a demanda](#2-de-onde-vem-a-demanda)
3. [O plano de produção](#3-o-plano-de-produção)
4. [Parâmetros de planejamento e regras por item](#4-parâmetros-de-planejamento-e-regras-por-item)
5. [Como o MRP pensa, passo a passo](#5-como-o-mrp-pensa-passo-a-passo)
6. [O resultado: ordens planejadas, perfil e exceções](#6-o-resultado-ordens-planejadas-perfil-e-exceções)
7. [Modos de planejamento por item](#7-modos-de-planejamento-por-item)
8. [Prioridade das ordens](#8-prioridade-das-ordens)
9. [Calendário industrial](#9-calendário-industrial)
10. [CRP — a fábrica aguenta?](#10-crp--a-fábrica-aguenta)
11. [APS — em que ordem produzir?](#11-aps--em-que-ordem-produzir)
12. [Tudo num clique: o pipeline de planejamento](#12-tudo-num-clique-o-pipeline-de-planejamento)
13. [Glossário rápido](#13-glossário-rápido)

---

## 1. O problema que o MRP resolve

Sem planejamento, a empresa vive entre dois extremos: ou **falta material** e o pedido atrasa, ou **sobra estoque** parado consumindo dinheiro. O MRP encontra o equilíbrio calculando, item por item, exatamente o que é preciso e quando — considerando o que **já existe** e o que **já está a caminho**.

---

## 2. De onde vem a demanda

O MRP enxerga a necessidade a partir de três fontes, que podem conviver:

| Fonte | O que é | Como entra |
|---|---|---|
| **Pedido de venda** | Pedidos confirmados | Gerada **automaticamente** ao confirmar o pedido |
| **Demanda independente** | Necessidade avulsa lançada à mão (item, quantidade, data) | Cadastro manual, quando preciso |
| **Previsão de venda** | Estimativa do que será vendido, para antecipar compra/produção | Cadastro de previsão |

### Previsão de venda

A previsão é trabalhada com bastante recurso:

- **Previsão por item e ano**, consultável por item;
- **Blocos de previsão** — agrupamentos para organizar e comparar cenários;
- **Apropriação** — como a previsão é distribuída/consumida ao longo do tempo, com uma apropriação **padrão**;
- **Previsão estatística** — o sistema **projeta a demanda futura** a partir do histórico, usando modelos estatísticos (ver `producao.md`/glossário), poupando o "chute" manual.

> Conforme a venda real vai entrando, ela "consome" a previsão, evitando planejar em dobro.

---

## 3. O plano de produção

O MRP sempre roda sobre um **plano de produção**. O plano define:

- o **escopo** (quais itens entram);
- a **origem da demanda** (considerar pedidos, previsões, a partir de qual data);
- os **parâmetros** daquela rodada.

O plano pode combinar os métodos MRP, mínimos/máximos, ponto de reposição, MPS e
Kanban. Também pode considerar todas as demandas independentes, ignorá-las ou
considerá-las somente a partir de uma data. Quando um item de pedido específico
é informado, ele substitui os filtros por classificação, evitando uma seleção
ambígua.

É possível **criar, listar, consultar, atualizar e excluir** planos. Trabalhar com planos permite simular cenários ("e se eu antecipar este lote?") sem afetar a operação real.

Cada plano e cada resultado do cálculo pertencem à empresa selecionada no
acesso. Usuários autorizados em mais de uma empresa escolhem a empresa ao entrar,
e dados de planejamento de uma empresa não aparecem em outra.

O plano também pode relacionar outras empresas do grupo para o processo
inter-fábrica. Para cada empresa relacionada, o PCP decide se as ordens poderão
ter liberação automática ou se precisarão de análise manual.

Somente pedidos marcados como inter-fábrica e originados dessas empresas entram
no cálculo. As necessidades permanecem identificadas como DIF e as ordens OIF
ou OCI guardam a empresa de origem e a indicação de liberação automática.

Ao executar o plano, o PCP informa o número inicial das ordens sugeridas. Essa
numeração acompanha a sugestão até sua conversão em ordem planejada e ordem de
fabricação. Também pode solicitar a gravação do LLC calculado no cadastro dos
itens para consultas e cálculos posteriores.

Pedidos originados da assistência técnica podem gerar ordens OAT quando os
controles de planejamento e estoque de terceiros estiverem habilitados. O MRP
mantém o almoxarifado de assistência informado no item do pedido e não mistura
em uma única ordem demandas destinadas a almoxarifados diferentes.

---

## 4. Parâmetros de planejamento e regras por item

O comportamento do MRP é ajustável em dois níveis:

### Parâmetros gerais de planejamento
Um conjunto de **parâmetros de planejamento** (numerados) controla regras globais do cálculo — como tratamento de horizontes, agrupamentos e arredondamentos. Podem ser **listados e atualizados**, adaptando o motor à realidade da empresa.

### Regras configuradas por item
Cada item pode ter **regras próprias** que o MRP respeita ao gerar as ordens, por exemplo:

- **lote mínimo** e **múltiplo de compra/produção** (não adianta comprar 3 se o fornecedor vende de 10 em 10);
- **estoque de segurança**;
- **lead time** (prazo de produção ou de compra).

---

## 5. Como o MRP pensa, passo a passo

Para manter estoque, demandas e sugestões consistentes, o VentureERP executa
**um cálculo de planejamento por vez**. Se já houver uma rodada em andamento,
uma nova solicitação é recusada até a conclusão da atual.

Quando o planejamento roda, o sistema executa esta sequência:

1. **Coleta as demandas** — pedidos confirmados, demandas independentes e previsões do plano.
2. **Abre a receita do produto (BOM)** — descobre todos os componentes e matérias-primas, **nível a nível** (usando o LLC do item).
3. **Olha o estoque atual** — quanto já existe de cada item (uma "foto" do saldo).
4. **Calcula a necessidade líquida** — `o que é preciso − estoque − o que já está comprado/em produção (suprimento firme)`. Sugestões ainda não contam como suprimento.
5. **Aplica as regras do item** — lote mínimo, múltiplos, estoque de segurança.
6. **Calcula as datas** — recua no tempo a partir da data de entrega, usando o **tempo de produção** (caminho crítico do roteiro) ou o **prazo do fornecedor**.
7. **Gera as sugestões** e registra o **perfil** do item por período.

**Exemplo simples:**
> Cliente pede **100 suportes** para o dia 30.
> Estoque: 20 prontos → faltam **80**.
> Cada suporte usa 0,8 kg de chapa → precisa de **64 kg**.
> Estoque de chapa: 40 kg → comprar **24 kg**.
> O fornecedor entrega em 5 dias e a fabricação leva 3 dias → o pedido de compra precisa sair **hoje** para não atrasar.

Tudo isso o sistema faz em segundos, para milhares de itens.

---

## 6. O resultado: sugestões, ordens planejadas, perfil e exceções

O MRP **não compra nem produz sozinho** — ele **sugere**, e uma pessoa decide. Ele entrega:

| Saída | O que é |
|---|---|
| **Sugestões de ordens** | Propostas automáticas de **compra** e de **produção**, com quantidade e data — ainda não são ordens reais |
| **Ordens planejadas** | Sugestões aceitas e confirmadas pelo planejador; tornam-se ordens reais no sistema |
| **Perfil do item** | A "linha do tempo" de cada item: necessidades, estoque projetado e suprimentos por período |
| **Mensagens de exceção** | Avisos quando algo não fecha: atraso inevitável, item sem cadastro completo, gargalo de máquina |

As exceções podem ser **notificadas por e-mail/alerta**, para que o planejador aja antes de o problema virar atraso de entrega.

### O passo de "firmar" a sugestão

O MRP gera **sugestões** — propostas que ficam em análise. O planejador as revisa
e pode **firmar** cada uma individualmente. "Firmar" é o ato de converter uma
sugestão em uma **Ordem Planejada real**, com número de ordem, que passa a contar
como suprimento firme para os próximos cálculos MRP.

```
Sugestão MRP (proposta) ──firmar──▶ Ordem Planejada ──────────▶ Ordem de Produção
                                    (número de ordem)             (se tipo = Fabricação)
                                          │
                                          └─────────────────────▶ Pedido de Compra
                                                                  (se tipo = Compra)
```

- **Compra:** a sugestão firmada vira uma Ordem Planejada de compra, que o comprador converte em Pedido de Compra (ver `compras.md`).
- **Produção:** firmar uma sugestão de fabricação cria automaticamente a Ordem de Fabricação (ver `producao.md`).
- **Rejeitar:** basta não firmar — a sugestão será substituída na próxima execução do MRP.

### Liberar, firmar e replanejar

Ordens planejadas podem ser liberadas para execução ou firmadas quando a data não
pode mais mudar. Uma ordem apenas liberada pode voltar ao planejamento enquanto
não houver apontamentos ou consumos. A tela/API permite aplicar a ação a várias
ordens; itens Kanban respeitam o parâmetro 25 para evitar ordens duplicadas. Nas
relações entre fábricas marcadas para liberação automática, o cálculo já converte
e libera somente as sugestões elegíveis, sem duplicá-las em reexecuções.

### Entregar a produção

A OF liberada pode receber entregas parciais ou uma entrega final. O produto
acabado entra no almoxarifado da ordem, com lote rastreável; componentes marcados
para baixa automática geram a requisição planejada REP. Excedentes são
identificados como EPE quando a empresa habilita esse tratamento. Uma ordem de
serviço não pode ser finalizada com quantidade zero enquanto houver compra de
serviço pendente.

### Consultar e explicar o planejamento

O planejador pode comparar a fotografia do último cálculo com a posição atual,
filtrar um período e consultar totais de demanda, ordens e estoque. Na ordem de
fabricação, uma visão consolidada reúne entregas, apontamentos, consumos, lotes e
movimentos, permitindo rastrear da necessidade calculada até a execução.

---

## 7. Modos de planejamento por item

Nem todo item precisa do cálculo completo. Cada item pode ser planejado pelo método que faz mais sentido:

| Modo | Quando usar |
|---|---|
| **MRP** | Itens com receita e demanda calculável (o padrão) |
| **Mín/Máx** | Repor entre um nível mínimo e máximo |
| **Ponto de pedido** | Comprar quando o saldo cai abaixo de um limite |
| **Kanban** | Reposição puxada por consumo |
| **MPS / plano-mestre** | Itens-chave planejados manualmente |

---

## 8. Prioridade das ordens

Quando há mais demanda do que capacidade, **o que vem primeiro?** O cadastro de **prioridade de ordens** define os níveis de prioridade usados pelo sequenciamento (APS) para decidir a ordem de produção — garantindo que os pedidos mais importantes/urgentes sejam atendidos antes.

---

## 9. Calendário industrial

O **calendário industrial** registra os **dias úteis e não úteis** da fábrica (feriados, paradas, finais de semana). O planejamento usa esse calendário para calcular prazos realistas — ele **não conta um feriado como dia produtivo**. É possível cadastrar os dias e consultar o mês e os dias úteis de cada período.

---

## 10. CRP — a fábrica aguenta?

O **CRP** (planejamento de capacidade) pega todas as ordens sugeridas e **soma as horas necessárias em cada máquina/centro de trabalho, dia a dia**. Compara com a capacidade disponível (descontando paradas de manutenção) e mostra onde há **sobrecarga** (mais de 100% da capacidade). É possível listar a carga por plano e ver especificamente os centros **sobrecarregados**.

> É o alerta que evita prometer um prazo que a fábrica não tem como cumprir.

---

## 11. APS — em que ordem produzir?

Enquanto o CRP responde "cabe?", o **APS** responde "**em que ordem e em qual máquina**?". Ele monta a sequência de produção (capacidade finita) respeitando:

- a **prioridade** das ordens (quem entrega antes vem primeiro);
- a **capacidade real** de cada máquina (uma de cada vez);
- **finais de semana, feriados e paradas** de manutenção.

O resultado pode ser visto como um **gráfico de Gantt** — a "agenda" de cada máquina ou de cada ordem.

---

## 12. Tudo num clique: o pipeline de planejamento

Em vez de rodar MRP, CRP e APS separadamente, o sistema oferece o **pipeline**: um único disparo que encadeia os três e devolve um **parecer de viabilidade consolidado** — o que precisa ser comprado/produzido, onde há sobrecarga, a sequência sugerida e um veredito final: **é viável ou não** atender no prazo.

---

## 13. Glossário rápido

| Termo | Significado |
|---|---|
| **MRP** | Planejamento de necessidades de materiais — o que/quanto/quando comprar e produzir |
| **BOM / Estrutura** | A "receita" do produto: seus componentes |
| **LLC** | Nível de planejamento do item; define a ordem de explosão da estrutura |
| **Necessidade líquida** | O que falta de verdade, já descontado estoque e suprimentos firmes |
| **Suprimento firme** | Ordens já aprovadas/firmadas e itens em trânsito |
| **Lead time** | Tempo total para produzir ou receber um item |
| **Perfil do item** | A linha do tempo de necessidades/estoque/suprimentos |
| **CRP** | Verificação de capacidade das máquinas |
| **APS** | Sequenciamento — a fila/agenda de produção |
| **Firmar** | Confirmar uma sugestão, transformando-a em ordem real |

> A versão técnica detalhada está em `../dev/mrp-calculo.md`, `../dev/visao-geral.md` (§3) e `../dev/manufatura-e-compras.md` (CRP/APS/pipeline).

## 14. Da sugestão à entrega na fábrica

O fluxo cobre a criação e manutenção do plano, cálculo exclusivo, firmação e
liberação de ordens, criação manual de OF com demandas, substituição de
componentes, seleção de lotes, baixa automática, entrega parcial/final e
destinação de refugos. Entregas e estoque são atualizados juntos: uma falha não
deixa a OF parcialmente concluída.

Para operações terceirizadas, a OF permanece ligada à solicitação e ao pedido
de compra do serviço. Assim, o sistema impede encerrar uma produção enquanto o
serviço necessário ainda estiver pendente.

Os relatórios de perfil, disponibilidade, necessidades agrupadas, explosão e
ponto de reposição ajudam o planejador a responder o que falta, quando faltará e
qual quantidade comprar ou fabricar.
