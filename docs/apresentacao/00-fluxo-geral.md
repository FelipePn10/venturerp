# VentureERP — Como o sistema funciona de ponta a ponta

### Apresentação geral para a empresa

---

Este documento explica, em ordem e em linguagem simples, **todo o caminho que um produto percorre dentro do VentureERP** — desde o momento em que o cliente faz um pedido até a entrega da nota fiscal e a baixa no estoque. É a visão "de cima" do sistema; cada etapa tem um documento próprio com mais detalhes.

---

## Sumário

1. [A ideia em uma frase](#1-a-ideia-em-uma-frase)
2. [O fluxo completo em uma imagem](#2-o-fluxo-completo-em-uma-imagem)
3. [Etapa por etapa](#3-etapa-por-etapa)
4. [O que o sistema faz sozinho (automações)](#4-o-que-o-sistema-faz-sozinho-automações)
5. [Os "documentos" do sistema e seus status](#5-os-documentos-do-sistema-e-seus-status)
6. [Onde ver cada assunto em detalhe](#6-onde-ver-cada-assunto-em-detalhe)

---

## 1. A ideia em uma frase

> O VentureERP transforma **pedidos de venda** em **um plano de produção e compras**, executa esse plano na fábrica e no recebimento, e fecha o ciclo com a **nota fiscal**, o **estoque** e o **financeiro** — tudo conectado, sem redigitar informação.

A grande vantagem é que cada cadastro feito uma única vez (cliente, item, máquina, fornecedor, impostos) volta a ser usado automaticamente nas etapas seguintes. Quando o vendedor confirma um pedido, o sistema já sabe o que precisa ser comprado, o que precisa ser fabricado, em qual máquina, quanto tempo leva e quanto vai custar.

---

## 2. O fluxo completo em uma imagem

```
   CLIENTE                                      CADASTROS DE APOIO
  faz o pedido                            (item, estrutura, roteiro,
       │                                   máquina, fornecedor, impostos)
       ▼
 ┌──────────────┐
 │ PEDIDO DE    │  confirmado  ──►  vira "demanda" para o planejamento
 │ VENDA        │
 └──────────────┘
       │
       ▼
 ┌──────────────┐   O cérebro do planejamento. A partir da demanda ele calcula:
 │  MRP         │   • o que falta comprar      • o que precisa fabricar
 │ (planejam.)  │   • em que data              • em que quantidade
 └──────────────┘
       │
       ├───────────────► CRP / APS  (a fábrica tem capacidade? em que ordem produzir?)
       │
   ┌───┴────────────────────────────┐
   ▼                                 ▼
 COMPRAR (matéria-prima)         FABRICAR (produto)
   │                                 │
   ▼                                 ▼
 Pedido de Compra                Ordem de Produção
   │  recebe a NF do fornecedor    │  inicia → consome insumo → aponta → conclui
   ▼                                 ▼
 ENTRA no estoque ──────────────► usa na fabricação ──► PRODUTO ACABADO no estoque
                                                              │
                                                              ▼
                                              Atende o PEDIDO DE VENDA
                                                              │
                                                              ▼
                                       NOTA FISCAL DE SAÍDA + baixa de estoque
                                                + conta a receber (financeiro)
                                                              │
                                                              ▼
                                                  EXPEDIÇÃO / entrega ao cliente
```

---

## 3. Etapa por etapa

### Etapa 1 — O pedido de venda entra
O vendedor registra o pedido do cliente: quais itens, quantidades e datas de entrega. O sistema verifica o cliente, sua condição de pagamento e seu limite de crédito (pode **bloquear** automaticamente um pedido acima do limite).

### Etapa 2 — O pedido vira demanda do planejamento
Quando o pedido é **confirmado**, o sistema cria automaticamente uma "necessidade" para cada item — é o gatilho que liga a venda ao planejamento. Nada precisa ser redigitado.

### Etapa 3 — O MRP calcula o que fazer
O MRP (planejamento de necessidades de materiais) é o cérebro do sistema. Ele:
- olha tudo o que foi pedido e previsto;
- "abre" cada produto na sua receita (estrutura/BOM) para descobrir as peças e matérias-primas;
- desconta o que já existe em estoque e o que já está comprado;
- calcula **o que falta**, **quanto** e **para quando**, recuando no tempo a partir da data de entrega.

O resultado são **sugestões**: ordens de **compra** (para o que se compra) e ordens de **produção** (para o que se fabrica).

### Etapa 4 — A fábrica consegue atender? (CRP e APS)
- O **CRP** soma as horas que cada máquina vai precisar e mostra se alguma está **sobrecarregada**.
- O **APS** organiza a fila: define em que ordem e em qual máquina cada item será produzido, respeitando feriados e paradas.

Assim o planejador enxerga gargalos **antes** de prometer prazos.

### Etapa 5 — Aprovar e disparar
As sugestões passam por uma decisão humana:
- **Compras** aprova as sugestões de compra → vira **Pedido de Compra** para o fornecedor.
- **Produção/PCP** confirma ("firma") as sugestões de fabricação → vira **Ordem de Produção**.

### Etapa 6 — Recebimento da compra
Quando a matéria-prima chega, a **nota fiscal do fornecedor** é importada. O sistema reconhece o fornecedor, **dá entrada no estoque** e **baixa o pedido de compra** automaticamente.

### Etapa 7 — Fabricação
A ordem de produção é executada no chão de fábrica: **inicia**, **consome** a matéria-prima (que sai do estoque), **aponta** o tempo e a quantidade produzida e **conclui** — quando o produto acabado **entra no estoque**.

**Plano de Corte (quando há peças cortadas):** para barras, perfis e chapas/MDF, o sistema gera o **plano de corte** direto das ordens, **otimiza o aproveitamento** do material, dá a **baixa real** (em metro, m², peça ou quilo), guarda as **sobras** para reuso e entrega o **mapa de corte** (para imprimir ou mandar à máquina). É opcional — produtos sem corte seguem direto. Detalhes em [`plano-de-corte`](plano-de-corte.md).

### Etapa 8 — Faturamento e saída
Com o produto pronto, o pedido de venda é atendido: o sistema emite a **Nota Fiscal Eletrônica (NF-e)** junto à SEFAZ, calcula todos os impostos, **baixa o estoque**, gera a **conta a receber** e marca o pedido como faturado.

### Etapa 9 — Expedição
O **romaneio** organiza a **separação** (com reserva de estoque), a **conferência** (detectando divergências), a **embalagem em volumes** e o **transporte** da carga, amarra a **NF-e** e registra o **despacho** ao cliente — com documento profissional impresso. Detalhes em [`romaneio`](romaneio.md).

---

## 4. O que o sistema faz sozinho (automações)

Estas etapas acontecem **automaticamente**, sem digitação manual:

| Quando acontece | O que o sistema faz sozinho |
|---|---|
| Pedido de venda é **confirmado** | Cria a demanda para o planejamento |
| Roda o **planejamento** | Encadeia MRP → CRP → APS num clique e devolve um parecer de viabilidade |
| **Compras aprova** uma sugestão | Gera o Pedido de Compra com o fornecedor preferencial e suas condições |
| **Produção firma** uma sugestão | Gera a Ordem de Produção já com item, máquina e datas |
| **Importa a NF** do fornecedor | Dá entrada no estoque e baixa o pedido de compra |
| **Consumo** na produção | Baixa a matéria-prima do estoque |
| **Conclusão** da produção | Dá entrada do produto acabado no estoque |
| **Autoriza a NF-e de saída** | Baixa o estoque, baixa as reservas, gera a conta a receber e marca o pedido como faturado |

> Todo movimento de estoque **atualiza o saldo na mesma hora**, incluindo o custo médio — não há "estoque defasado".

---

## 5. Os "documentos" do sistema e seus status

Cada etapa gera um documento que evolui por status. Acompanhar o status é como acompanhar o pedido:

| Documento | Caminho dos status |
|---|---|
| **Pedido de Venda** | Rascunho → Confirmado → (bloqueado por crédito) → Faturado / Cancelado |
| **Ordem Planejada** (sugestão) | Planejada → Liberada (firme) → Cancelada |
| **Pedido de Compra** | Rascunho → Solicitado → Aprovado → Parcial → Recebido → Cancelado |
| **Nota de Entrada** (compra) | Pendente → Conferida → Aprovada → Baixada / Cancelada |
| **Ordem de Produção** | Aberta → Em andamento → Concluída → Encerrada / Cancelada |
| **Nota Fiscal de Saída** | Rascunho → Autorizada → Cancelada / Rejeitada |
| **Romaneio (Expedição)** | Aberto → Separado → Conferido → Despachado / Cancelado |

---

## 6. Onde ver cada assunto em detalhe

Esta pasta (`apresentacao/`) tem um documento por área, todos no mesmo estilo simples:

| Quero entender… | Documento |
|---|---|
| Como cadastrar cliente, fornecedor, item, empresa | `cadastros.md` |
| Como funcionam as máquinas e os tempos de produção | `maquinas.md` |
| Como o MRP planeja e o CRP/APS avaliam a fábrica | `mrp-planejamento.md` |
| Como uma ordem é fabricada (qualidade, manutenção) | `producao.md` |
| Como funcionam as compras (cotação, pedido) | `compras.md` |
| Pedido de venda, expedição e prazos de entrega | `vendas.md` |
| Romaneio: separação, conferência, volumes e despacho | `romaneio.md` |
| Almoxarifado e movimentações de estoque | `estoque.md` |
| Custo padrão e centros de custo | `custos.md` |
| Notas fiscais, impostos e financeiro | `fiscal-financeiro.md` |

> A versão **técnica** (para a equipe de desenvolvimento), com endpoints e regras internas, está na pasta `../dev/`.
