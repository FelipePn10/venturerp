# VentureERP — Custos

### Apresentação para Controladoria e Direção

---

Saber **quanto custa fabricar** cada produto é o que sustenta a formação de preço e a margem. O VentureERP calcula o custo a partir dos próprios cadastros — estrutura, roteiro, tempos de máquina e custo de material — sem depender de planilhas paralelas. O resultado é um **custo padrão** confiável, comparável com o custo real da produção.

---

## Sumário

1. [Custo padrão: o custo "de referência"](#1-custo-padrão-o-custo-de-referência)
2. [Do que o custo é composto](#2-do-que-o-custo-é-composto)
3. [O rollup: como o custo é montado nível a nível](#3-o-rollup-como-o-custo-é-montado-nível-a-nível)
4. [Custo de máquina (centros de trabalho)](#4-custo-de-máquina-centros-de-trabalho)
5. [Custo de compra dos itens](#5-custo-de-compra-dos-itens)
6. [Centros de custo](#6-centros-de-custo)
7. [Custos indiretos: overhead e base de alocação](#7-custos-indiretos-overhead-e-base-de-alocação)
8. [Custo padrão x custo real](#8-custo-padrão-x-custo-real)
9. [Glossário rápido](#9-glossário-rápido)

---

## 1. Custo padrão: o custo "de referência"

O **custo padrão** é o custo esperado de fabricar um item, calculado a partir dos cadastros: a receita (estrutura), as operações (roteiro) e os tempos/custos de máquina e de material. Serve de **referência** para formar preço, avaliar o estoque e comparar com o que realmente aconteceu na produção. O custo padrão de cada item pode ser **consultado** a qualquer momento.

---

## 2. Do que o custo é composto

O custo de um produto fabricado é a soma de três parcelas:

| Parcela | O que é |
|---|---|
| **Material** | A matéria-prima da estrutura (BOM), pelo custo de compra/estoque |
| **Mão de obra / máquina** | O tempo das operações do roteiro × o custo/hora do centro de trabalho |
| **Custos indiretos (overhead)** | A parcela de custos gerais da fábrica rateada ao produto |

---

## 3. O rollup: como o custo é montado nível a nível

O **rollup** é o cálculo que "sobe" o custo pela estrutura, de baixo para cima:

```
Chapa de aço (comprada)        → custo de compra
   ↓ entra em
Suporte cortado (fabricado)    → material (chapa) + máquina (corte) + overhead
   ↓ entra em
Suporte soldado (produto final)→ material (cortado + parafusos) + máquina (solda/pintura) + overhead
```

Assim, o custo de um produto final **já inclui** o custo de todos os seus intermediários, calculado de forma consistente. O rollup pode ser disparado sempre que os cadastros mudarem (novo preço de matéria-prima, novo tempo de máquina).

---

## 4. Custo de máquina (centros de trabalho)

Cada **centro de trabalho/máquina** tem um **custo por hora** cadastrado. Multiplicado pelo tempo das operações do roteiro, ele dá a parcela de transformação do produto. Os custos de centro de trabalho podem ser cadastrados e listados, mantendo o cálculo sempre atualizado com a realidade da fábrica.

---

## 5. Custo de compra dos itens

Para os itens **comprados**, o sistema mantém o **custo de compra** de referência (por item), que entra como parcela de material no rollup dos produtos fabricados. Esse custo pode ser cadastrado e consultado por item.

---

## 6. Centros de custo

O **centro de custo** agrupa **para onde os gastos vão** (um setor, uma linha, uma máquina). Ele permite enxergar **onde** o dinheiro é consumido, é a base para ratear os custos indiretos de forma justa e conecta a produção à contabilidade/financeiro. Os centros de custo podem ser criados, listados e consultados.

---

## 7. Custos indiretos: overhead e base de alocação

Nem todo custo é diretamente ligado a um produto (energia, supervisão, aluguel da fábrica). Para distribuí-los de forma justa, o sistema usa dois conceitos:

| Conceito | O que é |
|---|---|
| **Base de alocação** | O **critério** de rateio (ex.: horas de máquina, quantidade produzida, valor de material) |
| **Alocação de overhead** | A regra que **distribui os custos indiretos** aos produtos/centros usando a base escolhida |

Assim o custo final reflete o gasto real da operação, não só o material — e a margem calculada é confiável.

---

## 8. Custo padrão x custo real

O custo padrão é o **planejado**; a produção registra o **realizado** (tempo apontado, material consumido — ver `producao.md`). Comparar os dois revela **desvios** — uma operação que demorou mais, um material que rendeu menos — e aponta onde agir para proteger a margem.

> Relatórios de custo (histórico de custos, ficha técnica por item, produtos produzidos) estão no módulo Financeiro — ver `fiscal-financeiro.md`.

---

## 9. Glossário rápido

| Termo | Significado |
|---|---|
| **Custo padrão** | Custo esperado de fabricar, calculado pelos cadastros |
| **Rollup** | Cálculo que monta o custo subindo pela estrutura |
| **Centro de trabalho** | Máquina/posto com custo por hora |
| **Centro de custo** | Agrupamento de onde os gastos ocorrem |
| **Overhead** | Custos indiretos rateados aos produtos |
| **Base de alocação** | O critério usado para distribuir o overhead |
| **Desvio** | Diferença entre o custo padrão e o custo real |

## Novidades (2026-06) — Custo real da ordem de produção

Além do **custo padrão** (planejado), cada ordem de produção agora apura o **custo
real**: o material baixado é valorizado pelo custo médio do estoque e a
transformação vem das horas apontadas × o custo/hora do centro de trabalho. Ao
fechar a ordem, o sistema calcula sozinho o custo real e o **desvio** (real −
padrão) de material, mão-de-obra e overhead — então dá para saber a **margem real**
de cada peça, e não só a teórica.

> A versão técnica está em `../dev/custos.md` (§4 Custo Real), `../dev/manufatura-e-compras.md` (§4 Custo Padrão) e em `../dev/visao-geral.md`.
