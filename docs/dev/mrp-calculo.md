# MRP Calculation — Guia Completo

## O que é MRP?

MRP significa **Planejamento das Necessidades de Materiais** (em inglês: *Material Requirements Planning*).

Na prática, é o "cérebro" da fábrica. Ele responde a uma pergunta simples, porém complexa de calcular:

> **"O que eu preciso produzir ou comprar, em que quantidade, e até quando, para entregar tudo que me foi pedido?"**

Sem o MRP, o planejador faz isso manualmente — planilhas, estimativas, achismos. Com o MRP, o sistema analisa tudo automaticamente: os pedidos dos clientes, o que já tem em estoque, o que já está sendo produzido, e o que vai precisar comprar ou fabricar. O resultado são **sugestões de ordens de produção e compra** com datas e quantidades calculadas.

---

## O Problema que o MRP Resolve

Imagine que um cliente pediu 100 bicicletas para daqui a 30 dias.

Uma bicicleta tem:
- 1 quadro (fabricado internamente — leva 5 dias)
- 2 rodas (compradas de fornecedor — prazo de 10 dias)
- 1 guidão (fabricado internamente — leva 3 dias)
- Parafusos, pedais, corrente, etc.

Sem o MRP, o planejador precisa calcular manualmente: "Preciso de 200 rodas. Já tenho 50 em estoque. Logo preciso comprar 150. O prazo do fornecedor é 10 dias, então o pedido precisa sair até dia X." Isso para cada componente, de cada produto, de cada pedido. Com 500 produtos e 300 pedidos simultâneos, é impossível sem errar.

O MRP faz esse trabalho em segundos, para todos os itens ao mesmo tempo.

---

## Como o Cálculo Funciona — Passo a Passo

### Passo 1 — Coleta das Demandas

O MRP começa lendo tudo que precisa ser produzido ou entregue. Existem dois tipos de demanda:

**Demanda Independente** — o que o cliente pediu diretamente. Exemplo: "200 cadeiras modelo Executive para 15/06". Esse dado vem de pedidos de venda ou de uma previsão de vendas cadastrada manualmente.

**Demanda Dependente** — o que precisa ser fabricado *por causa* da demanda independente. Se preciso de 200 cadeiras, então preciso de 200 assentos, 800 parafusos, 200 encostos, etc. O MRP calcula isso automaticamente explodindo a Estrutura do Produto.

### Passo 2 — Lê a Estrutura do Produto (BOM)

Cada produto tem uma **estrutura** que descreve seus componentes e as quantidades necessárias. Essa estrutura pode ter vários níveis:

```
Cadeira Executive (nível 1 — produto final)
├── Assento (nível 2 — conjunto)
│   ├── Espuma 5cm (nível 3 — matéria-prima)
│   └── Tecido Couro 0,5m² (nível 3 — matéria-prima)
├── Encosto (nível 2 — conjunto)
│   └── Estrutura Metálica (nível 3 — matéria-prima)
└── Parafusos M6 x 4 unid. (nível 2 — componente)
```

O MRP percorre essa estrutura de cima para baixo, calculando a necessidade de cada nível.

### Passo 3 — Verifica o Estoque Atual (Snapshot)

Antes de calcular o que falta, o MRP tira uma "foto" do estoque no momento do cálculo. Isso se chama **snapshot de estoque**. Por que uma foto e não o estoque em tempo real? Para que o resultado seja reproduzível — se você rodar o MRP duas vezes no mesmo dia, o segundo cálculo parte do mesmo ponto, sem ser contaminado por movimentações que aconteceram entre os dois.

### Passo 4 — Calcula a Necessidade Líquida

```
Necessidade Líquida = Demanda Total − Estoque Disponível − Ordens Já Abertas
```

Se preciso de 200 assentos, tenho 30 em estoque e já tenho uma ordem de fabricação aberta por 50, então minha necessidade líquida é **120 assentos** (200 − 30 − 50 = 120).

Se a necessidade líquida for negativa (tenho mais do que preciso), o MRP não sugere nenhuma ordem.

### Passo 5 — Aplica as Regras do Item (Configured Item Rules)

Cada item pode ter regras que modificam como o MRP calcula. Por exemplo:

- **Lote mínimo**: "Esse parafuso só pode ser comprado em caixas de 500." Se a necessidade for 120, o MRP arredonda para 500.
- **Lead time**: "Esse tecido leva 15 dias para chegar." O MRP recua a data de início do pedido em 15 dias.
- **Estoque de segurança**: "Sempre manter pelo menos 100 unidades em estoque." O MRP adiciona isso à demanda.

### Passo 6 — Gera as Sugestões de Ordens

Com a necessidade líquida calculada e as regras aplicadas, o MRP gera **sugestões de ordens**:

- Para itens **fabricados internamente** → sugere uma Ordem de Produção
- Para itens **comprados de fornecedores** → sugere uma Ordem de Compra
- Para itens **de terceiros** → não gera ordem (o item não pertence à empresa)

Cada sugestão tem: o item, a quantidade, a data de necessidade, e a data de início (calculada subtraindo o lead time da data de necessidade).

### Passo 7 — Registra o Perfil do Item

Para cada item calculado, o MRP salva um **Perfil MRP**, que é um registro histórico mostrando: demanda, estoque projetado, ordens planejadas, ordens firmes. Esse perfil serve para análise posterior — você pode ver como o estoque de um item evolui semana a semana ao longo do horizonte de planejamento.

---

## O Conceito de LLC (Low-Level Code — Código de Nível Mais Baixo)

Esse é um dos conceitos mais importantes do MRP e funciona assim:

Um item pode aparecer em vários produtos e em vários níveis da estrutura. Por exemplo, o "Parafuso M6" pode estar no nível 2 de um produto e no nível 4 de outro.

O LLC é o **nível mais fundo** em que esse item aparece em qualquer estrutura. No exemplo acima, o LLC do Parafuso M6 seria 4.

Por que isso importa? Porque o MRP processa os itens em ordem crescente de LLC. Isso garante que quando o MRP for calcular o Parafuso M6, já calculou todas as ordens de todos os produtos que precisam dele — e pode somar toda a demanda de uma vez, evitando calcular o mesmo item múltiplas vezes.

Sem o LLC, o MRP poderia sugerir duas ordens de 50 parafusos separadas quando o correto seria uma única ordem de 100.

---

## Todas as Entidades Envolvidas

### O que o MRP **lê** (entradas):

---

#### Item
O cadastro central de tudo que a empresa produz, compra ou vende. Cada item tem configurações que o MRP usa diretamente:

| Campo | O que significa para o MRP |
|---|---|
| **Tipo** (FABRICADO / COMPRADO / DE_TERCEIRO) | Define que tipo de ordem o MRP vai sugerir |
| **Tipo MRP** (NORMAL_MRP / PROJETO) | NORMAL_MRP entra no cálculo automático; PROJETO é planejado separadamente |
| **LLC** | Em que nível da estrutura esse item está (calculado automaticamente) |
| **Estoque mínimo** | Adicionado à demanda como segurança |
| **Tipo de estrutura** (INDUSTRIAL / COMERCIAL) | Apenas INDUSTRIAL entra no cálculo de produção |

---

#### Demanda Independente
O que foi pedido diretamente — seja por um pedido de venda ou por uma previsão cadastrada pelo planejador. Cada registro tem:

- **Item** e **quantidade**
- **Data da demanda** — quando precisa estar pronto
- **Máscara** — identifica uma variação específica do item (ex: cor, tamanho)
- **Centro de custo** — onde será alocado o custo

Essa é a **entrada principal** do MRP. Tudo começa aqui.

---

#### Estrutura do Produto (BOM — Bill of Materials)
A "receita" de cada produto: quais componentes são necessários e em que quantidade para fazer uma unidade do produto pai.

| Campo | O que significa |
|---|---|
| **Item pai** | O produto que está sendo montado |
| **Item filho** | O componente necessário |
| **Quantidade** | Quantas unidades do filho são necessárias por unidade do pai |
| **Percentual de perda** | Se há perda no processo (ex: corte de tecido gera sobra), o MRP já aumenta a quantidade necessária |
| **Máscara pai** | Se preenchida, esse componente só é usado na variação específica do produto |

---

#### Snapshot de Estoque
Uma fotografia do estoque no momento em que o MRP é executado. Contém:

- **Item** e **armazém**
- **Quantidade disponível**
- **Quantidade reservada** (já comprometida com outras ordens)
- **Estoque de segurança** (mínimo obrigatório)

O estoque líquido disponível = Quantidade disponível − Quantidade reservada − Estoque de segurança.

---

#### Demanda de Pedido de Venda (SalesOrderDemand)
Similar à demanda independente, mas vinculada especificamente a um pedido de venda já existente no sistema. Tem controle de quantidade já entregue e status (PENDENTE / ENTREGUE / CANCELADO).

---

#### Calendário Industrial
Define quais dias são dias úteis na fábrica. O MRP usa isso para calcular datas corretamente — se a data calculada cai num domingo ou feriado, o MRP avança para o próximo dia útil. Sem isso, uma ordem poderia ter data de início em 25 de dezembro.

---

#### Calendário por Item (Item Calendar Promise)
Uma exceção por item. Por exemplo, um determinado produto pode ter uma linha de produção que só funciona de segunda a quarta. O MRP respeita esse calendário específico ao calcular as datas desse item.

---

#### Máquina e Tempo de Produção por Item
Cada item pode ser produzido em uma ou mais máquinas. Para cada combinação item+máquina, existe um cadastro com:

- **Tempo de produção** (em minutos, horas ou dias)
- **Quantidade base** (para quantos itens esse tempo se aplica — ex: "5 minutos para produzir 10 peças")
- **Tempo de setup** (tempo fixo de preparação da máquina, independente da quantidade)
- **Prioridade** (qual máquina usar primeiro se houver opção)

O MRP usa isso para calcular não apenas *o que* produzir, mas *quanto tempo* vai ocupar cada máquina — e identificar gargalos (quando a demanda exige mais do que a máquina consegue produzir no tempo disponível).

---

#### Regras Configuradas por Item (Configured Item Rules)
Regras flexíveis que personalizam o comportamento do MRP para itens específicos. Uma regra tem:

- **Tipo de tabela** (PLANNING_DATA ou PLANNER_DATA): qual conjunto de dados da configuração do item essa regra se aplica
- **Campo** (field_name): qual atributo está sendo sobrescrito (ex: "lead_time", "lote_minimo")
- **Tipo de regra** (EQUAL, DIFFERENT, RANGE): como comparar o valor
- **Valor da regra**: o valor que define o comportamento
- **Sequência**: ordem de aplicação quando há várias regras

Exemplo: "Para o item 1042, se o campo `lead_time` for EQUAL a 0, usar 15 dias." Isso permite ajustar o comportamento do MRP sem alterar o cadastro do item.

---

### O que o MRP **gera** (saídas):

---

#### Sugestão de Ordem Planejada (PlannedOrderSuggestion)
O resultado central do MRP. Para cada necessidade líquida calculada, o MRP gera uma sugestão contendo:

- **Item** e **quantidade**
- **Data de necessidade** (quando precisa estar pronto)
- **Data de início** (quando precisa começar, calculada subtraindo o lead time)
- **Tipo de ordem** (FABRICAÇÃO ou COMPRA)
- **Tipo de demanda** (INDEPENDENTE ou DEPENDENTE)
- **Item pai** que gerou essa necessidade (se for demanda dependente)
- **LLC do item** (para ordenação do processamento)

> **Co-produtos e quantidade fixa na explosão (BOM).** Componentes marcados como
> `is_coproduct` são **saídas** (co-produto/subproduto/sucata) — a explosão **não** gera
> demanda dependente para eles. Componentes `is_fixed_qty` são consumidos **uma vez por
> OF** (a fórmula de perda roda sobre base 1, sem multiplicar pela quantidade da ordem).

Essas sugestões ficam em análise. O planejador pode aceitar, rejeitar, ou modificar. Quando aceita, a sugestão se transforma em uma **Ordem Planejada** real no sistema.

> **Conversão sugestão → OF (padrão SAP "converter ordem planejada em ordem de produção").**
> As sugestões usam tipos internos em PT (`FABRICACAO`/`COMPRA`/`SERVICO`/`TECHNICAL_ASSISTANCE`)
> e são mapeadas para o enum do banco em EN (`PRODUCTION`/`PURCHASE`/`OUTSOURCING`/`TECHNICAL_ASSISTANCE`)
> por `mapMRPOrderType` ao aceitar (`FirmarSugestaoMRPUseCase`). Aceitar uma sugestão agora
> **firma a ordem em um passo** — cria a Ordem Planejada e, via `FirmPlannedOrderUseCase`,
> gera a **Ordem de Fabricação (OF)** para ordens de produção e a **requisição de serviço**
> para operações externas. (Antes, a sugestão criava só a Ordem Planejada já firme e a OF
> nunca era gerada — corrigido.) Rastro (pegging): a cadeia demanda → plano → sugestão →
> ordem planejada → OF é preservada via `plan_code`, `parent_item` e `planned_order_id`.

---

#### Perfil MRP do Item (MRPItemProfile)
O "extrato" histórico de cada item calculado. Para cada item e cada data do horizonte de planejamento, registra:

- **Demanda**: quanto foi pedido nessa data
- **Ordens planejadas**: quanto está previsto para ser produzido/comprado
- **Ordens firmes**: quanto já está confirmado
- **Estoque projetado**: quanto estará em estoque após tudo ser executado

Esse perfil é o que aparece na tela de análise de planejamento — a famosa "tabela MRP" com linhas por semana/mês.

---

#### Log de Execução (MRPCalculationLog)
Um registro de cada vez que o MRP foi executado. Contém:

- **Quando começou** e **quando terminou**
- **Status**: RUNNING (calculando), COMPLETED (concluído com sucesso), ERROR (falhou)
- **Erros encontrados**: quais itens tiveram problema e por quê
- **Total de itens processados** e **total de ordens geradas**

Isso permite auditoria: "Quem rodou o MRP ontem às 14h? Quantas ordens foram geradas? Houve algum erro?"

---

#### Agenda de Máquina (MachineSchedule)
Quando o MRP sugere uma ordem de produção que precisa de uma máquina, ele (no futuro) também vai sugerir o agendamento nessa máquina. A agenda de máquina mostra:

- **Qual máquina** e **qual data**
- **Horário de início e fim**
- **Quantidade planejada** a ser produzida
- **Sequência** na fila da máquina naquele dia
- **Status**: AGENDADO → EM_ANDAMENTO → CONCLUÍDO ou CANCELADO

Isso permite enxergar a carga de trabalho de cada máquina dia a dia.

---

## Fluxo Completo — Do Pedido à Ordem

```
CLIENTE FAZ PEDIDO
        │
        ▼
Demanda Independente é registrada
(item, quantidade, data de entrega)
        │
        ▼
PLANEJADOR ACIONA O MRP
        │
        ├─ 1. Tira snapshot do estoque atual
        │
        ├─ 2. Calcula LLC de todos os itens
        │       (garante a ordem correta de processamento)
        │
        ├─ 3. Para cada item, do nível 1 ao nível mais baixo:
        │       │
        │       ├─ Lê a demanda (independente + dependente acumulada)
        │       ├─ Subtrai o estoque disponível
        │       ├─ Subtrai ordens já abertas
        │       ├─ Aplica regras do item (lote mínimo, lead time, etc.)
        │       ├─ Calcula necessidade líquida
        │       ├─ Gera sugestão de ordem (produção ou compra)
        │       ├─ Explode a estrutura → gera demanda dependente para os filhos
        │       └─ Salva o perfil MRP do item
        │
        ├─ 4. Registra log de execução (status COMPLETED)
        │
        ▼
PLANEJADOR ANALISA AS SUGESTÕES
        │
        ├─ Aceita → Ordem Planejada é criada e agendada na máquina
        └─ Rejeita ou ajusta manualmente
```

---

## Regras de Negócio Importantes

### Quando o MRP gera ordem de fabricação?
Quando o item é do tipo **FABRICADO** e tem necessidade líquida positiva.

### Quando o MRP gera ordem de compra?
Quando o item é do tipo **COMPRADO** e tem necessidade líquida positiva.

### Quando o MRP não gera nada?
- Item do tipo **DE_TERCEIRO** (não é da empresa)
- Item com tipo MRP = **PROJETO** (planejado manualmente)
- Item com necessidade líquida ≤ 0 (estoque suficiente)
- Item com tipo de estrutura **COMERCIAL** (não entra na produção industrial)

### O que acontece quando rodo o MRP duas vezes?
Os perfis do plano são apagados e recalculados do zero. O log de execução mantém histórico das execuções anteriores. Ordens já firmadas (aprovadas pelo planejador) não são afetadas pelo novo cálculo.

### O que é "firmar" uma ordem?
Uma sugestão do MRP é apenas uma proposta. Quando o planejador "firma" (aprova), ela se torna uma Ordem Planejada real, com número, e entra na agenda de produção. A partir daí, o MRP passa a considerá-la como "ordem aberta" nos cálculos seguintes.

---

## Status das Demandas de Pedido de Venda

| Status | Significado |
|---|---|
| PENDING | Aguardando produção/compra |
| DELIVERED | Entregue ao cliente |
| CANCELLED | Cancelado, não entra mais no cálculo |

---

## Tipos de Regra Configurada

| Tipo | Significado |
|---|---|
| EQUAL | O campo deve ser exatamente igual ao valor informado |
| DIFFERENT | O campo deve ser diferente do valor informado |
| RANGE | O campo deve estar dentro de um intervalo de valores |

---

## Módulos que o MRP Depende

```
┌─────────────────────────────────────────────────────┐
│                    MRP CALCULATION                  │
│                                                     │
│  Lê de:                                             │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────┐ │
│  │  Itens   │  │ Demandas │  │ Estrutura (BOM)   │ │
│  └──────────┘  └──────────┘  └───────────────────┘ │
│  ┌──────────┐  ┌──────────┐  ┌───────────────────┐ │
│  │ Estoque  │  │Calendário│  │ Máquinas / Tempos │ │
│  └──────────┘  └──────────┘  └───────────────────┘ │
│                                                     │
│  Gera:                                              │
│  ┌──────────────────┐  ┌────────────────────────┐  │
│  │ Ordens Planejadas│  │ Perfil MRP por Item    │  │
│  └──────────────────┘  └────────────────────────┘  │
│  ┌──────────────────┐  ┌────────────────────────┐  │
│  │ Agenda Máquinas  │  │ Log de Execução        │  │
│  └──────────────────┘  └────────────────────────┘  │
└─────────────────────────────────────────────────────┘
```

---

## Endpoints da API

Todos os endpoints exigem `Authorization: Bearer <JWT>` (ADMIN ou USER).

### Execução do MRP

| Método | Rota | Descrição |
|--------|------|-----------|
| `POST` | `/api/mrp-calculation/run` | Executa o cálculo MRP para um plano — `{ "plan_code": 1 }` |

> ⚠️ **Pré-requisito — o plano precisa existir.** `plan_code` referencia
> `production_plans.code` (FK `mrp_calculation_logs_plan_code_fkey`). Crie o plano
> **antes** de rodar o MRP via `POST /api/production-plan/create` (não existe
> `planning/plans` nem `mrp/plans`). Rodar com um `plan_code` inexistente viola a FK.
| `GET` | `/api/mrp-calculation/profile/{item_code}/{plan_code}` | Perfil do item: demanda, estoque projetado, ordens |
| `POST` | `/api/mrp-calculation/configured-rules` | Cria regra configurada por item |
| `GET` | `/api/mrp-calculation/configured-rules/{item_code}` | Lista regras configuradas de um item |
| `GET` | `/api/mrp-calculation/exceptions/{plan_code}` | Lista exceções e alertas do cálculo |

### Sugestões e Ponte para Ordens Planejadas

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | `/api/mrp-calculation/suggestions/{plan_code}` | Lista todas as sugestões geradas pelo plano |
| `POST` | `/api/mrp-calculation/suggestions/{code}/firm` | **Firma uma sugestão** — converte em Ordem Planejada real |

#### `GET /api/mrp-calculation/suggestions/{plan_code}`

Lista as sugestões de ordens geradas pelo último cálculo do plano. Cada sugestão é
uma proposta pendente de análise — ainda não é uma Ordem Planejada.

**Resposta esperada (`200 OK`):**
```json
[
  {
    "code": 42,
    "plan_code": 1,
    "item_code": 2001,
    "quantity": 150.0,
    "need_date": "2026-07-15T00:00:00Z",
    "start_date": "2026-07-05T00:00:00Z",
    "order_type": "PURCHASE",
    "demand_type": "INDEPENDENT",
    "parent_item_code": null,
    "llc": 1,
    "notes": null
  }
]
```

---

#### `POST /api/mrp-calculation/suggestions/{code}/firm`

**Ponte MRP → Ordem Planejada** (a etapa de "firmar sugestão").

Converte a sugestão `{code}` em uma Ordem Planejada real (`planned_orders`),
com `is_firm = true`. Atribui um número de ordem único. Para sugestões do tipo
`PRODUCTION`, a firmação automática do `FirmPlannedOrderUseCase` também cria a
Ordem de Fabricação (`production_orders`) correspondente.

**Request:** body vazio `{}`

**Resposta esperada (`201 Created`):**
```json
{
  "suggestion_code": 42,
  "planned_code": 199,
  "order_number": 7001,
  "item_code": 2001,
  "quantity": 150.0,
  "order_type": "PURCHASE",
  "need_date": "2026-07-15T00:00:00Z",
  "status": "PLANNED",
  "is_firm": true,
  "plan_code": 1
}
```

| Campo | Significado |
|---|---|
| `suggestion_code` | Código da sugestão que deu origem à ordem |
| `planned_code` | ID da Ordem Planejada criada (chave primária) |
| `order_number` | Número sequencial único da ordem (visível no chão de fábrica) |
| `is_firm` | Sempre `true` — firmar é uma ação irreversível |
| `status` | `PLANNED` (ordem criada, aguardando liberação para produção/compra) |

---

### Ordens Planejadas

| Método | Rota | Descrição |
|--------|------|-----------|
| `POST` | `/api/planned-order/create` | Cria Ordem Planejada manualmente (sem passar pelo MRP) |
| `GET` | `/api/planned-order/list` | Lista todas as Ordens Planejadas |
| `GET` | `/api/planned-order/{code}/firm` | Firma uma Ordem Planejada já existente |

> **Corpo do `POST /api/planned-order/create`.** Campos: `item_code` (obrigatório),
> `quantity` (> 0), `order_type` (`PRODUCTION`/`PURCHASE`/`OUTSOURCING`), `need_date`,
> `demand_type` e opcionais (`mask`, `cost_center_code`, `machine_code`, ...).
> `demand_type` ∈ `SALES_ORDER` · `FORECAST` · `INDEPENDENT` · `SAFETY_STOCK` ·
> `REPLENISHMENT`; **omitido assume `INDEPENDENT`** (a coluna é NOT NULL e não tinha
> default). Valor inválido → 422. O caminho recomendado continua sendo **firmar uma
> sugestão do MRP**, não a criação manual.

---

## Fluxo Completo — Do Pedido à Ordem Firmada

```
CLIENTE FAZ PEDIDO
        │
        ▼
Pedido de Venda confirmado
→ Demanda de Pedido de Venda gerada automaticamente
        │
        ▼
PLANEJADOR ACIONA O MRP
POST /api/mrp-calculation/run { "plan_code": 1 }
        │
        ├─ 1. Snapshot de estoque
        ├─ 2. Calcula LLC
        ├─ 3. Para cada item (do LLC mais alto ao mais baixo):
        │       ├─ Lê demanda (independente + dependente acumulada)
        │       ├─ Subtrai estoque disponível e ordens firmes abertas
        │       ├─ Aplica regras do item (lote mínimo, lead time, segurança)
        │       ├─ Calcula necessidade líquida
        │       ├─ Gera sugestão (mrp_planned_suggestions)
        │       ├─ Explode a BOM → demanda dependente para os filhos
        │       └─ Salva o perfil MRP do item
        ├─ 4. Registra log (status COMPLETED)
        │
        ▼
PLANEJADOR ANALISA AS SUGESTÕES
GET /api/mrp-calculation/suggestions/{plan_code}
        │
        ├─ ACEITA → firma a sugestão
        │   POST /api/mrp-calculation/suggestions/{code}/firm
        │   → cria planned_orders (is_firm = true)
        │   → se OrderType = PRODUCTION: cria production_orders automaticamente
        │
        └─ REJEITA → descarta (sugestão permanece em mrp_planned_suggestions
                              sem virar ordem, e é apagada na próxima execução)
```

---

## Status de Suporte — Módulos Implementados

| Funcionalidade | Status |
|---|---|
| Motor de cálculo (BFS + LLC + time-phased netting) | ✅ Completo |
| Explosão de BOM multi-nível | ✅ Completo |
| Integração com Pedido de Venda → Demanda | ✅ Completo (automático ao confirmar pedido) |
| Sugestões `mrp_planned_suggestions` | ✅ Completo |
| Perfil MRP por item | ✅ Completo |
| Log de execução | ✅ Completo |
| Exceções automáticas | ✅ Completo |
| Ponte sugestão → Ordem Planejada (firmar) | ✅ Completo |
| Ordens Planejadas (`planned_orders`) | ✅ Completo |
| Firmação automática de OF para tipo PRODUCTION | ✅ Completo |
| Agendamento automático de máquinas por APS | 🚧 Parcial (agenda criada manualmente via `POST /api/machine/schedule/create`; o APS sequencia, mas não cria agendas automaticamente ainda) |
