# VentureERP — Gestão de Máquinas e Produção

> Documento de apresentação

---

## O que o sistema faz nessa área?

O VentureERP organiza **como cada produto é fabricado, em qual máquina e quanto tempo isso leva**. Com essas informações o sistema consegue dizer automaticamente:

- Quando uma ordem de produção vai ficar pronta.
- Se a máquina tem capacidade para atender o pedido no prazo.
- Qual máquina usar quando o produto pode ser feito em mais de uma.

---

## Os três cadastros que sustentam tudo

### 1. Tipo de Máquina

Antes de cadastrar uma máquina, ela precisa ter um **tipo**. O tipo é simplesmente a categoria do equipamento — define o que aquela máquina faz dentro da fábrica.

| Tipo | Exemplos de uso |
|---|---|
| Corte | Serra fita, guilhotina, plasma |
| Dobrar | Dobradeira, prensa de dobra |
| Soldar | Solda MIG, TIG, ponto |
| Montar | Bancada de montagem |
| Pintar | Cabine de pintura, estufa |
| Torno | Torno mecânico, CNC |
| Moinho | Fresadora |
| Imprensa | Prensa hidráulica, excêntrica |
| Injeção | Injetora de plástico |

> O tipo não define tempo nem custo — ele só organiza as máquinas em categorias lógicas.

---

### 2. Máquina

Aqui é onde cada equipamento físico da fábrica é cadastrado individualmente.

**O que é configurado por máquina:**

| Campo | O que significa |
|---|---|
| **Capacidade** | Quanto a máquina consegue produzir por período (ex: 100 peças por hora) |
| **Unidade** | Em que medida ela trabalha: peças, kg, metros, m², toneladas... |
| **Período** | O intervalo dessa capacidade: por minuto, por hora ou por dia |
| **Eficiência** | O quanto da capacidade total ela realmente entrega (ex: 85%) |

**Exemplo prático:**
> A Serra Fita #1 tem capacidade de **50 peças por hora** com **eficiência de 90%** — ou seja, na prática ela entrega **45 peças por hora** considerando paradas, ajustes e variações normais de operação.

A eficiência existe porque nenhuma máquina roda 100% do tempo disponível — há micro-paradas, aquecimento, variação de operador, entre outros fatores.

---

### 3. Configuração de Tempo por Item e por Máquina

Este é o **cadastro mais importante** para o cálculo de produção. Aqui é definido, para cada produto fabricado em cada máquina, quanto tempo de produção é necessário.

**O que é configurado:**

| Campo | O que significa |
|---|---|
| **Item** | Qual produto está sendo configurado |
| **Variante (máscara)** | Qual dimensão ou configuração específica do produto |
| **Máquina** | Em qual máquina esse tempo foi medido |
| **Tempo de produção** | Quanto tempo leva um ciclo de fabricação |
| **Unidade do tempo** | Minutos, horas ou dias por ciclo |
| **Quantidade base** | Quantas peças saem por ciclo |
| **Tempo de setup** | Tempo para preparar a máquina antes de começar |
| **Prioridade** | Qual máquina usar primeiro quando há opções |

---

## O conceito de Variante (máscara)

Um mesmo produto pode ter **dimensões diferentes** — e cada dimensão pode ter um tempo de fabricação diferente. O sistema chama isso de **variante** (internamente chamada de máscara).

**Exemplo real — Mesa de escritório, item 2210:**

| Variante | Dimensões (C × L × A) | Tempo na Serra | Setup na Serra |
|---|---|---|---|
| Padrão (sem variante) | Configuração genérica | 5 min/peça | 10 min |
| 130 × 240 × 234 | Grande | 8 min/peça | 20 min |
| 60 × 80 × 75 | Pequena | 3 min/peça | 8 min |

O setup maior para a mesa grande faz sentido: a máquina precisa ser reajustada, trocar gabaritos ou carregar um programa diferente — o que leva mais tempo do que para uma peça menor.

**Regra de funcionamento:**
Quando o sistema vai calcular o tempo de produção de um pedido, ele busca a configuração exata daquela variante. Se não existir cadastro para aquela variante específica, ele usa a configuração padrão (sem variante).

---

## Como o sistema calcula o tempo de produção

Quando um pedido de produção entra no sistema, o cálculo funciona assim:

### Passo a passo com exemplo:

**Pedido:** 73 chapas dobradas, variante "250×120"
**Máquina:** Prensa Hidráulica #2

**Configuração cadastrada:**
- Tempo por ciclo: 4 minutos
- Quantidade por ciclo (lote): 10 chapas
- Setup: 15 minutos

**Cálculo:**

```
Número de ciclos = arredondar para cima (73 ÷ 10) = 8 ciclos
                   (o 8º ciclo processa só 3 chapas, mas ainda ocupa a máquina)

Tempo de fabricação = 8 ciclos × 4 minutos = 32 minutos
Tempo de setup      = 15 minutos (uma vez só, no início)

Tempo total         = 32 + 15 = 47 minutos
```

O **arredondamento para cima** no número de ciclos é fundamental: se a prensa faz 10 chapas por golpe e o pedido precisa de 73, o último golpe vai processar apenas 3 — mas a máquina foi ocupada por um ciclo completo. O sistema considera isso corretamente.

---

## Normalização de períodos

O sistema aceita tempos cadastrados em **minutos, horas ou dias** e converte tudo automaticamente na hora do cálculo.

| Cadastrado como | Sistema converte para |
|---|---|
| 2 horas por lote | 120 minutos por lote |
| 1 dia por lote | 480 minutos por lote (1 turno de 8h) |
| 30 minutos por lote | 30 minutos por lote |

Isso permite que cada produto seja cadastrado na unidade que faz mais sentido operacionalmente — sem necessidade de padronizar manualmente.

---

## Compatibilidade de unidades entre item e máquina

O sistema verifica automaticamente se a unidade de medida do produto é compatível com a unidade de capacidade da máquina — e converte quando faz sentido.

**Conversões aceitas automaticamente:**

| Unidade do produto | Unidade da máquina | Conversão |
|---|---|---|
| Quilograma (kg) | Tonelada (t) | 1 kg = 0,001 t |
| Tonelada (t) | Quilograma (kg) | 1 t = 1.000 kg |
| Milímetro (mm) | Metro (m) | 1 mm = 0,001 m |
| Metro cúbico (m³) | Litros | 1 m³ = 1.000 L |
| Peças | Unidades | equivalente |

**Incompatibilidades bloqueadas:**
O sistema impede configurações sem sentido físico — por exemplo, cadastrar um produto medido em **quilogramas** numa máquina configurada em **metros**. Isso evita erros de cadastro que poderiam gerar cálculos incorretos de prazo e capacidade.

---

## Verificação de gargalo

Ao calcular o tempo de produção, o sistema também verifica se a máquina tem **capacidade suficiente** para atender a demanda dentro do tempo calculado.

Se a velocidade exigida pelo pedido for maior do que a capacidade efetiva da máquina, o sistema sinaliza um **gargalo de produção** — indicando que pode ser necessário redistribuir a carga, fazer hora extra ou utilizar uma máquina alternativa.

---

## Prioridade entre máquinas

Um produto pode ser fabricado em mais de uma máquina. O sistema usa o campo **prioridade** para decidir automaticamente qual usar:

- Prioridade **1** → máquina preferida (normalmente a mais rápida ou de menor custo)
- Prioridade **2** → alternativa quando a primeira não está disponível
- Prioridade **3** → terceira opção, e assim por diante

Isso permite que o planejamento seja feito de forma automática sem intervenção manual a cada pedido.

---

## Resumo do fluxo

```
Tipo de Máquina
  └── agrupa as máquinas por categoria (corte, solda, pintura...)

Máquina
  └── define capacidade real + eficiência de cada equipamento

Configuração de Tempo (por item + variante + máquina)
  └── define tempo de ciclo, quantidade por ciclo e setup
  └── o sistema seleciona a configuração correta pela variante do produto
  └── se não houver variante específica, usa a configuração padrão

Cálculo de Produção (MRP)
  └── converte unidades automaticamente
  └── calcula número de ciclos (arredondando para cima)
  └── soma setup + fabricação = tempo total
  └── verifica se a máquina é gargalo
  └── retorna prazo de entrega em minutos, horas e dias
```
