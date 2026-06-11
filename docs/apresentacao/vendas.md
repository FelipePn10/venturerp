# VentureERP — Vendas e Expedição

### Apresentação para o setor Comercial e Logística

---

O módulo de Vendas é a **porta de entrada** do fluxo do ERP: é o pedido do cliente que dispara todo o planejamento, a produção e o faturamento. A Expedição é a **porta de saída**: organiza a separação, a conferência e o despacho ao cliente. Entre as duas pontas, o sistema ajuda a **prometer e cumprir prazos**.

---

## Sumário

1. [Pedido de venda](#1-pedido-de-venda)
2. [Itens do pedido](#2-itens-do-pedido)
3. [Crédito e bloqueio](#3-crédito-e-bloqueio)
4. [Do pedido ao planejamento](#4-do-pedido-ao-planejamento)
5. [Divisão de vendas](#5-divisão-de-vendas)
6. [Promessa de entrega (prazos confiáveis)](#6-promessa-de-entrega-prazos-confiáveis)
7. [Reprogramação de entrega](#7-reprogramação-de-entrega)
8. [Faturamento](#8-faturamento)
9. [Expedição / Romaneio](#9-expedição--romaneio)
10. [Glossário rápido](#10-glossário-rápido)

---

## 1. Pedido de venda

O pedido de venda registra o que o cliente comprou. No **cabeçalho** ficam: cliente, condição de pagamento, vendedor/divisão, datas e o **plano de produção** ao qual o pedido pertence (o elo com o planejamento).

Os preços vêm da **tabela de vendas** do cliente, e o tipo de nota e os impostos já ficam definidos pelo cadastro — o pedido sai consistente desde o início.

**Ciclo de vida:**

| Status | Significado |
|---|---|
| **Rascunho (R)** | Em montagem, ainda não confirmado |
| **Confirmado / Pedido (P)** | Pedido firmado — vira demanda para o planejamento |
| **Bloqueado** | Travado (por crédito ou manualmente) |
| **Faturado (F)** | Nota fiscal emitida |
| **Cancelado** | Pedido cancelado |

O sistema permite **criar, listar, consultar, atualizar, cancelar** e **mudar o status** do pedido, além de listar **por cliente** e **por status** — uma visão direta da carteira.

---

## 2. Itens do pedido

Cada linha do pedido é um **item**: produto, quantidade e **data de entrega** própria (um pedido pode ter entregas em datas diferentes). Os itens podem ser **adicionados, listados, atualizados e cancelados** individualmente, sem mexer no restante do pedido.

---

## 3. Crédito e bloqueio

O cliente tem um **limite de crédito**. Um pedido pode ser **bloqueado** (automaticamente ao ultrapassar o limite, ou manualmente por decisão comercial/financeira) e depois **desbloqueado**. Enquanto bloqueado, o pedido **não avança** — protegendo a empresa de vender para quem está inadimplente ou no limite.

---

## 4. Do pedido ao planejamento

Quando o pedido é **confirmado**, o sistema cria automaticamente a **demanda** de cada item para o planejamento (MRP). Esse é o elo que liga a venda à fábrica: a partir daí o sistema sabe o que precisa comprar e produzir para atender aquele cliente no prazo.

> Reconfirmar o mesmo pedido **não duplica** a demanda — o sistema é seguro contra repetição.

---

## 5. Divisão de vendas

A **divisão de vendas** organiza a área comercial (equipes, regiões ou unidades de negócio). Cada pedido pode ser associado a uma divisão, o que permite **medir resultado por equipe/segmento** e aplicar regras comerciais específicas. As divisões podem ser criadas, listadas, consultadas, atualizadas e excluídas.

---

## 6. Promessa de entrega (prazos confiáveis)

O sistema ajuda a **prometer prazos realistas**, em vez de "chutar" uma data:

| Recurso | O que faz |
|---|---|
| **Parâmetros de promessa de entrega** | Regras gerais de como a data prometida é calculada |
| **Calendário de promessa por item** | Disponibilidade (capacidade de entrega) por item/variante, dia a dia — o que pode ser prometido em cada data |

Com isso, a data de entrega informada ao cliente considera a **disponibilidade real** (estoque + capacidade), reduzindo atrasos e promessas impossíveis.

---

## 7. Reprogramação de entrega

Quando uma data precisa mudar (atraso de matéria-prima, mudança de prioridade), o sistema registra a **reprogramação de entrega** vinculada ao pedido. Assim fica o **histórico** das remarcações (data original × nova data × motivo), com transparência para o comercial e para o cliente. É possível listar as reprogramações de cada pedido.

---

## 8. Faturamento

Com o produto disponível, o pedido é faturado. Ao **autorizar a Nota Fiscal de Saída (NF-e)**, o sistema executa em cadeia, automaticamente:

- emite a NF-e junto à SEFAZ, com **todos os impostos calculados**;
- **baixa o estoque** dos produtos;
- **baixa as reservas** do pedido;
- gera a **conta a receber** no financeiro;
- marca o pedido como **faturado**.

> Um único comando fecha venda, fiscal, estoque e financeiro de forma coerente. Detalhes fiscais em `fiscal-financeiro.md`.

---

## 9. Expedição / Romaneio

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

## 10. Glossário rápido

| Termo | Significado |
|---|---|
| **Pedido de venda** | O documento da compra do cliente |
| **Demanda** | A necessidade que o pedido confirmado gera para o planejamento |
| **Divisão de vendas** | Agrupamento comercial (equipe/região/unidade) |
| **Reserva** | Estoque separado para um pedido |
| **Promessa de entrega** | Data de entrega calculada com base em estoque e capacidade |
| **Reprogramação** | Remarcação registrada de uma data de entrega |
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
