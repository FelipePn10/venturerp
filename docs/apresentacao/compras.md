# VentureERP — Compras

### Apresentação para o setor de Compras e Suprimentos

---

O módulo de Compras transforma uma **necessidade** (gerada pelo planejamento ou por uma requisição interna) em um **pedido ao fornecedor**, passando — quando convém — por uma **cotação** para garantir o melhor preço. Tudo conectado ao estoque, ao fiscal e ao financeiro, com o mínimo de digitação.

---

## Sumário

1. [Como uma compra nasce](#1-como-uma-compra-nasce)
2. [Sugestões de compra do MRP](#2-sugestões-de-compra-do-mrp)
3. [Solicitação de compra](#3-solicitação-de-compra)
4. [Cotação (comparar fornecedores)](#4-cotação-comparar-fornecedores)
5. [Pedido de compra](#5-pedido-de-compra)
6. [Tabela de preço e fornecedor preferencial](#6-tabela-de-preço-e-fornecedor-preferencial)
7. [Operações de entrada (natureza fiscal da compra)](#7-operações-de-entrada-natureza-fiscal-da-compra)
8. [Recebimento e fechamento do ciclo](#8-recebimento-e-fechamento-do-ciclo)
9. [Glossário rápido](#9-glossário-rápido)

---

## 1. Como uma compra nasce

Existem três caminhos para iniciar uma compra, e todos terminam num **Pedido de Compra**:

| Origem | Quando acontece |
|---|---|
| **Sugestão do MRP** | O planejamento detectou que falta material e sugeriu comprar |
| **Solicitação de compra** | Alguém da empresa pediu um item (consumo, manutenção, etc.) |
| **Pedido direto** | Compra avulsa, lançada manualmente |

---

## 2. Sugestões de compra do MRP

As compras sugeridas pelo planejamento aparecem numa **lista de sugestões**. Para cada uma, o comprador decide:

- **Aprovar** → o sistema **gera o Pedido de Compra** já com o **fornecedor preferencial** do item (ou o escolhido) e seus padrões (condição de pagamento, tabela de preço, tipo de nota e frete), e marca a sugestão como atendida.
- **Rejeitar** → descarta a sugestão.

> É o elo automático entre o "falta material" do MRP e a ordem de compra — sem redigitar.

---

## 3. Solicitação de compra

A solicitação é o **pedido interno** de quem precisa do material. Cada solicitação reúne seus itens, e várias podem ser tratadas juntas: com um comando, o sistema **gera os pedidos de compra automaticamente, agrupando os itens por fornecedor** — evitando vários pedidos pequenos para o mesmo fornecedor.

A solicitação acompanha seu atendimento (**aberta → parcial → atendida**), de modo que sempre se sabe o que já virou pedido e o que ainda está pendente.

---

## 4. Cotação (comparar fornecedores)

Quando o preço precisa ser negociado, a **cotação** organiza a comparação de forma auditável:

```
1. Abre a cotação com os itens desejados
2. Adiciona os fornecedores que vão cotar
3. Registra os preços de cada fornecedor por item
4. Seleciona o melhor preço (por item)
5. Gera o(s) pedido(s) de compra a partir da seleção
```

Assim fica claro e registrado **por que** cada fornecedor foi escolhido — a decisão de compra deixa de morar em e-mails e planilhas.

---

## 5. Pedido de compra

O **Pedido de Compra** é o documento oficial enviado ao fornecedor. Ele reúne:

- fornecedor, condição de pagamento e tipo de frete;
- **itens** (que podem ser adicionados ao pedido), com quantidades e preços;
- datas de entrega.

Ao montar o pedido, o sistema **resolve automaticamente** dados do item (preço pela tabela, unidade, classificação fiscal), reduzindo digitação e erro.

**Ciclo de vida:**

| Status | Significado |
|---|---|
| **Rascunho** | Em preparação |
| **Solicitado** | Submetido para aprovação |
| **Aprovado** | Liberado e enviado ao fornecedor |
| **Parcial** | Parte das mercadorias já foi recebida |
| **Recebido** | Tudo recebido |
| **Cancelado** | Pedido cancelado |

É possível **listar** pedidos, consultar um pedido, **atualizar**, **cancelar** e filtrar **por fornecedor** ou **por status**.

---

## 6. Tabela de preço e fornecedor preferencial

Dois cadastros fazem as compras quase se preencherem sozinhas:

- **Tabela de preço de compra:** o preço de referência de cada matéria-prima (por fornecedor/tabela) — usado para valorizar o pedido e para o MRP. As tabelas e seus itens podem ser mantidos livremente.
- **Fornecedor preferencial por item:** cada matéria-prima indica seu fornecedor preferido (e alternativos). As sugestões de compra do MRP já saem com esse fornecedor.

---

## 7. Operações de entrada (natureza fiscal da compra)

Toda compra que entra precisa de uma **natureza/operação de entrada** correta (que define CFOP, tributação e o destino do material). O cadastro de **operações de entrada** padroniza isso e suporta **grupos de estados (UF)** — porque uma compra **dentro do estado** e **de fora do estado** podem ter tratamento fiscal diferente. O sistema ainda permite **validar** uma operação de entrada antes de usá-la, evitando erro fiscal no recebimento.

---

## 8. Recebimento e fechamento do ciclo

Quando a mercadoria chega com a **nota fiscal do fornecedor**, a nota é importada e o sistema, automaticamente:

1. **Reconhece o fornecedor** pelo CNPJ da nota;
2. **Dá entrada no estoque** de cada item (atualizando saldo e custo médio);
3. **Baixa o pedido de compra**, atualizando o status para *parcial* ou *recebido*;
4. Gera os **créditos fiscais** e a base da **conta a pagar** no financeiro.

> O ciclo de compra se fecha sozinho: do "falta material" do MRP até o material no estoque e a obrigação no financeiro, sem redigitar.
> O detalhe fiscal do recebimento (NF de entrada, créditos, escrituração) está em `fiscal-financeiro.md`.

---

## 9. Glossário rápido

| Termo | Significado |
|---|---|
| **Sugestão de compra** | Recomendação de compra gerada pelo MRP |
| **Solicitação de compra** | Pedido interno de material |
| **Cotação** | Comparação de preços entre fornecedores |
| **Pedido de compra** | Documento oficial enviado ao fornecedor |
| **Fornecedor preferencial** | O fornecedor padrão de cada item |
| **Operação de entrada** | A natureza fiscal aplicada na compra (CFOP, tributação) |
| **Recebimento** | Entrada da mercadoria via nota fiscal do fornecedor |

> A versão técnica está em `../dev/manufatura-e-compras.md` (§10–§16 Compras) e `../dev/cadastros-fornecedor.md`.
