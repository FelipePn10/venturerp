# VentureERP — Produção (Chão de Fábrica)

### Apresentação para o PCP e a produção

---

Este módulo cuida da **execução da fábrica**: como o produto é fabricado (o roteiro), como uma ordem de produção nasce, é executada, consome material, registra tempo e entrega o produto acabado no estoque — além da **qualidade** e da **manutenção** das máquinas que apoiam tudo isso.

---

## Sumário

1. [O roteiro: o passo a passo de fabricação](#1-o-roteiro-o-passo-a-passo-de-fabricação)
2. [Lead time: quanto tempo a fabricação leva](#2-lead-time-quanto-tempo-a-fabricação-leva)
3. [A Ordem de Produção e seu ciclo de vida](#3-a-ordem-de-produção-e-seu-ciclo-de-vida)
4. [Operações da ordem (acompanhamento por etapa)](#4-operações-da-ordem-acompanhamento-por-etapa)
5. [Consumo, apontamento e backflush](#5-consumo-apontamento-e-backflush)
6. [Qualidade](#6-qualidade)
7. [Manutenção preventiva](#7-manutenção-preventiva)
8. [Glossário rápido](#8-glossário-rápido)

---

## 1. O roteiro: o passo a passo de fabricação

O **roteiro** descreve **como** um produto é feito. Ele é montado a partir de três peças:

| Elemento | O que é |
|---|---|
| **Operações** | O catálogo de tarefas possíveis (cortar, dobrar, soldar, pintar…), cada uma ligada a um centro de trabalho/máquina, com tempo de setup e tempo por peça |
| **Roteiro do item (rota)** | A sequência de operações de um produto específico |
| **Rede de precedências** | A ligação entre as operações — o que precisa terminar antes do quê (permite caminhos paralelos, não só linha reta) |

**Exemplo — Suporte Soldado:**
```
Operação 10 — Cortar    (Serra Fita)        setup 10 min · 2 min/peça
Operação 20 — Dobrar    (Prensa)            setup 15 min · 3 min/peça
Operação 30 — Soldar    (Solda MIG)         setup 20 min · 5 min/peça
Operação 40 — Pintar    (Cabine de pintura) setup  8 min · 1 min/peça
```

Operações e roteiros podem ser **criados, listados, atualizados e desativados**, e as operações de uma rota podem ser **adicionadas, alteradas e removidas** — mantendo o histórico íntegro.

> Os detalhes de máquinas, capacidade e tempos por item estão em `maquinas.md`.

---

## 2. Lead time: quanto tempo a fabricação leva

A partir do roteiro e da rede de precedências, o sistema calcula o **lead time** de fabricação — o **caminho crítico**, ou seja, a sequência de operações que determina o prazo total. Esse número alimenta:

- as **datas** que o MRP calcula (quando começar para entregar no prazo);
- a **carga** que o CRP soma por máquina;
- a **fila** que o APS sequencia.

Como o sistema considera caminhos paralelos, operações que rodam ao mesmo tempo **não somam** no prazo — o cálculo reflete a fábrica real.

---

## 3. A Ordem de Produção e seu ciclo de vida

A **Ordem de Produção (OF)** é o documento que autoriza e acompanha a fabricação de uma quantidade de um item. Ela normalmente nasce **automaticamente** quando o PCP "firma" uma sugestão do MRP — já vindo com o item, a quantidade, a máquina, o roteiro e as datas preenchidas. Também pode ser criada manualmente para casos avulsos.

**Ciclo de vida:**

| Etapa | Comando | O que acontece |
|---|---|---|
| **Aberta** | criada | Aguardando início |
| **Em andamento** | iniciar | A produção começou; passam a valer apontamentos e consumos |
| **Concluída** | concluir | Produção terminada; o produto acabado **entra no estoque** |
| **Encerrada** | fechar | Ordem fechada administrativamente |
| **Cancelada** | cancelar | Ordem cancelada (antes da conclusão) |

É possível **listar** as ordens, **consultar** uma ordem e ver seu histórico de **apontamentos** e **consumos**.

---

## 4. Operações da ordem (acompanhamento por etapa)

Além de acompanhar a ordem como um todo, o sistema permite acompanhar **operação por operação**:

- **Explodir o roteiro** na ordem — cria as operações da OF a partir do roteiro do item;
- **Listar as operações** da ordem e seu andamento;
- **Avançar a operação** — registrar a conclusão de uma etapa e liberar a próxima.

Isso dá visibilidade de **onde cada ordem está** no chão de fábrica (ex.: "já cortou e dobrou, está na solda").

---

## 5. Consumo, apontamento e backflush

Durante a fabricação, dois registros mantêm o estoque e os custos corretos:

- **Consumo de material:** ao consumir a matéria-prima, o sistema **baixa o insumo do estoque** automaticamente (movimento de saída). Você sempre sabe quanto de cada material já foi usado.
- **Apontamento de produção:** registra **quanto tempo** foi gasto e **quantas peças** foram produzidas (e refugadas, se houver). É o que permite comparar o tempo real com o planejado.

**Backflush (baixa automática):** em vez de dar baixa item a item, o sistema pode **consumir os componentes da receita (BOM) automaticamente** quando você aponta a operação — proporcional à quantidade produzida. Menos digitação, estoque sempre certo.

Ao **concluir** a ordem, o produto acabado **entra no estoque** com o custo calculado, pronto para faturar.

---

## 6. Qualidade

O módulo de qualidade garante que defeitos sejam pegos cedo, evitando retrabalho e devolução. Ele é estruturado assim:

| Recurso | O que é |
|---|---|
| **Planos de inspeção** | Definem **o que** inspecionar em cada item e **quando** (no recebimento, no processo ou no produto final) |
| **Características** | Os pontos medidos em cada plano (dimensão, dureza, acabamento…), com seus limites |
| **Registros de inspeção** | O resultado real de cada inspeção, ligado à ordem e ao item |
| **Não-conformidades (NC)** | Quando algo sai fora do padrão: abre-se uma NC, que é acompanhada até a **disposição** (o que fazer com a peça: retrabalhar, sucatear, liberar com concessão) |

É possível consultar registros e NCs **por ordem** e **por item**, e listar as **NCs em aberto** — uma visão direta dos problemas de qualidade pendentes.

---

## 7. Manutenção preventiva

Máquina parada quebra prazo. O módulo de manutenção mantém os equipamentos rodando e **conversa com o planejamento**:

| Recurso | O que é |
|---|---|
| **Planos de manutenção** | A rotina de manutenção de cada máquina (periodicidade, tarefas) |
| **Ordens de manutenção** | A execução de uma manutenção; podem ser **geradas automaticamente** a partir dos planos |
| **Avanço da ordem** | Acompanhar a manutenção: planejada → em execução → concluída |

As paradas de manutenção são **respeitadas automaticamente** pelo CRP (capacidade) e pelo APS (fila) — ou seja, o sistema **não agenda produção numa máquina que estará em manutenção**. É possível consultar as ordens por plano e por centro de trabalho.

---

## 8. Glossário rápido

| Termo | Significado |
|---|---|
| **Roteiro / rota** | A sequência de operações para fabricar um item |
| **Operação** | Uma tarefa do roteiro (cortar, soldar…) num centro de trabalho |
| **Rede de precedências** | As ligações que dizem o que vem antes do quê |
| **Centro de trabalho** | A máquina/posto onde a operação é feita |
| **Setup** | Tempo de preparação da máquina antes de produzir |
| **Lead time / caminho crítico** | O prazo total de fabricação |
| **Ordem de Produção (OF)** | O documento que autoriza e acompanha a fabricação |
| **Apontamento** | Registro de tempo e quantidade produzida |
| **Consumo** | Baixa da matéria-prima usada |
| **Backflush** | Baixa automática dos componentes ao apontar |
| **Não-conformidade (NC)** | Registro de uma peça/lote fora do padrão de qualidade |

## Novidades (2026-06)

- **Custo real da ordem:** ao fechar a OF, o sistema apura o custo real (material +
  transformação) e o desvio contra o padrão — ver `../apresentacao/custos.md`.
- **Sucata/retalho valorizado:** a sucata e o retalho (sobras de chapa/barra) podem
  **voltar ao estoque como subproduto com valor**, para revenda ou reaproveitamento.
- **Lote produzido:** ao concluir a OF informando o lote do acabado, fica registrada
  a **genealogia** (quais lotes de matéria-prima compõem o produto).

> A versão técnica (operações, endpoints, regras) está em `../dev/producao.md` (§3 Custo real, §4 Sucata, §5 Lote), `../dev/manufatura-e-compras.md` (Roteiro, Qualidade, Manutenção) e `../dev/visao-geral.md` (§5 Produção).
