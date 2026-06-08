# VentureERP — Estoque e Almoxarifado

### Apresentação para o Almoxarifado e o PCP

---

O estoque é o **espelho físico** de todo o sistema: cada compra que entra, cada material consumido na produção e cada venda faturada **mexem no saldo automaticamente**, na hora. O objetivo é simples — o que o sistema mostra é o que existe na prateleira, com o custo correto.

---

## Sumário

1. [Armazéns e localizações](#1-armazéns-e-localizações)
2. [Saldo sempre atualizado](#2-saldo-sempre-atualizado)
3. [Movimentações: entradas e saídas](#3-movimentações-entradas-e-saídas)
4. [Tipos de movimento](#4-tipos-de-movimento)
5. [Custo médio do estoque](#5-custo-médio-do-estoque)
6. [Reservas](#6-reservas)
7. [Inventário e contagem](#7-inventário-e-contagem)
8. [Glossário rápido](#8-glossário-rápido)

---

## 1. Armazéns e localizações

O estoque é organizado em **armazéns/depósitos** (ex.: matéria-prima, produto acabado, linha de produção) e, dentro deles, em **localizações** (rua, prateleira, posição), cada uma com um tipo (interno, externo, inspeção, rejeição, reserva, trânsito, especial). Isso permite saber não só *quanto* tem, mas *onde* está e *em que situação*.

> O cadastro de armazéns e localizações está detalhado em `cadastros.md`.

---

## 2. Saldo sempre atualizado

A regra de ouro do módulo: **todo movimento atualiza o saldo na mesma operação**. Não existe "rodar fechamento" para o estoque ficar certo — entrada, consumo e venda refletem imediatamente na **quantidade disponível** e no **custo**. O saldo pode ser consultado **por item** e **por armazém**.

---

## 3. Movimentações: entradas e saídas

Cada mexida no estoque é uma **movimentação**, com origem rastreável (de qual pedido, ordem ou nota veio):

| Tipo | Quando acontece | Efeito |
|---|---|---|
| **Entrada** | Recebimento de compra (NF do fornecedor) | Aumenta o saldo |
| **Entrada** | Conclusão de uma ordem de produção | Produto acabado entra no estoque |
| **Saída** | Consumo de matéria-prima na produção | Reduz o saldo do insumo |
| **Saída** | Faturamento de venda (NF-e de saída) | Reduz o saldo do produto |
| **Ajuste** | Correção de inventário | Acerta o saldo |

Como cada movimento aponta sua origem, é possível responder "de onde veio" e "para onde foi" qualquer quantidade. As movimentações podem ser listadas **por item** e **por armazém**.

---

## 4. Tipos de movimento

O sistema mantém um cadastro de **tipos de movimento** (com sigla), que classifica cada lançamento de estoque (entrada de compra, produção, ajuste, transferência…). Esse cadastro padroniza os relatórios e garante que cada movimento tenha um significado claro e consistente.

---

## 5. Custo médio do estoque

A cada entrada, o sistema recalcula o **custo médio ponderado** do item e guarda também o **último custo**. Isso garante que o valor do estoque e o custo dos produtos vendidos estejam sempre corretos — base confiável para o preço de venda e para a contabilidade.

**Exemplo:**
> Estoque: 100 kg a R$ 10,00 (= R$ 1.000).
> Entra: 100 kg a R$ 12,00 (= R$ 1.200).
> Novo saldo: 200 kg a **R$ 11,00** de custo médio (R$ 2.200 ÷ 200).

---

## 6. Reservas

Um pedido pode **reservar** estoque, separando uma quantidade para um cliente/ordem específica antes do faturamento. Isso evita que o mesmo material seja prometido a dois pedidos. As reservas podem ser:

- **criadas** (separar a quantidade);
- **liberadas** (devolver ao disponível, se o pedido caiu);
- **consumidas** (baixadas de fato quando o material sai).

Ao faturar uma venda, a reserva é **consumida automaticamente** junto com a saída do estoque.

---

## 7. Inventário e contagem

Para conferir o estoque físico contra o sistema, há o módulo de **inventário**:

```
1. Cria o inventário (define o escopo a contar)
2. Conta os itens (registra a quantidade física encontrada)
3. Ajusta as diferenças (o sistema gera o movimento de acerto)
4. Fecha o inventário
```

É possível listar inventários, consultar um inventário, ver seus itens e registrar **contagem** e **ajuste** item a item. Assim o acerto de saldo é **controlado e auditável**, em vez de uma edição solta de quantidade.

---

## 8. Glossário rápido

| Termo | Significado |
|---|---|
| **Armazém / Depósito** | Local físico de estoque |
| **Localização** | Posição dentro do armazém |
| **Movimentação** | Cada entrada, saída ou ajuste |
| **Tipo de movimento** | A classificação (com sigla) de cada lançamento |
| **Saldo** | Quantidade disponível no momento |
| **Custo médio** | Valor médio ponderado de cada item em estoque |
| **Reserva** | Estoque separado para um pedido |
| **Inventário** | Conferência física × sistema, com contagem e ajuste |

> A versão técnica está em `../dev/visao-geral.md` (§5.2 Movimentos de Estoque).
