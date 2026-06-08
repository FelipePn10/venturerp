# Módulos de Manufatura — Documentação

Cobre os módulos implementados neste ciclo:
**Roteiro de Fabricação · CRP · APS · Custo Padrão · Qualidade · Manutenção Preventiva · Previsão Estatística · Alertas MRP (e-mail + webhook)**

> Documentação fiscal em separado: **fiscal-financeiro.md** (mesma pasta `docs/`).
> Índice geral da documentação: **README.md** (pasta `docs/`).

---

## 1. Roteiro de Fabricação

### O que é

O roteiro descreve **como** um item é produzido: quais operações são executadas, em que sequência, em quais centros de trabalho, com quais tempos e quais dependências existem entre as etapas.

O roteiro é criado **manualmente** pelo PCP/engenharia de processo. O MRP, CRP e APS apenas o *leem* — nunca o criam nem o modificam.

### Estrutura de dados

```
operations                  ← biblioteca reutilizável de operações genéricas
  └─ manufacturing_routes   ← roteiro de um item específico
       └─ route_operations  ← instância de uma operação dentro do roteiro
            └─ route_operation_network  ← grafo de dependências entre operações
```

#### `machine_types` — centros de trabalho

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `code` | int64 | Código do centro de trabalho |
| `name` | string | Nome (ex: "Fresadora CNC 01") |
| `type` | enum | `CUT`, `BEND`, `WELD`, `ASSEMBLE`, `PAINT`, `LATHE`, `MILL`, `INJECTION`, `PRESS` |
| `requires_operator` | bool | **`true` = máquina manual** (operador humano controla); **`false` = máquina automática** |
| `setup_time` | float64 | Tempo de setup padrão em minutos |

> **`requires_operator` é o campo que distingue máquinas manuais de automáticas.** Afeta diretamente o CPM e o APS — veja as seções abaixo. O padrão é `true` (a maioria das máquinas em chão de fábrica é operada por humanos).

---

#### `operations` — biblioteca de operações

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `name` | string | Ex: "Solda TIG", "Pintura Eletrostática" |
| `origin` | enum | `INTERNA`, `EXTERNA`, `TERCEIROS` |
| `standard_time` | float64 | Tempo padrão em horas |

**O campo `origin` determina o tipo de ordem que o MRP gera:**

| Origin | Significado | Ordem gerada pelo MRP |
|--------|-------------|----------------------|
| `INTERNA` | Operação executada pelo próprio chão de fábrica | Ordem de Fabricação (OF) |
| `EXTERNA` | Operação enviada para fornecedor externo | Ordem de Serviço (OS) |
| `TERCEIROS` | Operação realizada por terceiros contratados | Ordem de Serviço (OS) |

Quando um item do tipo `FABRICACAO` possui operações com origin `EXTERNA` ou `TERCEIROS` no seu roteiro padrão, o MRP gera automaticamente **ordens de serviço adicionais** para cada uma dessas operações, além da ordem de fabricação principal.

#### `manufacturing_routes` — roteiro de um item

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `item_code` | int64 | Item ao qual o roteiro pertence |
| `mask` | string? | Máscara de item (opcional) |
| `alternative` | int16 | Número de alternativa do roteiro (padrão: 1) |
| `description` | string? | Descrição livre do roteiro |
| `is_standard` | bool | `TRUE` = roteiro usado pelo MRP/CRP; apenas um por item |

#### `route_operations` — operação dentro do roteiro

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `sequence` | int16 | Posição (ex: 10, 20, 30) |
| `operation_id` | int64 | FK para `operations` |
| `work_center_id` | int64? | Centro de trabalho (sobrescreve o padrão da operação) |
| `standard_time` | float64? | Tempo corrigido; se nulo, herda da operação |
| `setup_time` | float64? | Tempo de setup |
| `notes` | text? | Observações livres |

#### `route_operation_network` — grafo de dependências

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `predecessor_id` | int64 | Operação que deve terminar (ou estar suficientemente avançada) antes |
| `successor_id` | int64 | Operação que só pode iniciar após a predecessora |
| `overlap_pct` | float64 | Sobreposição permitida em **porcentagem (0–100)**. Ex: `20` = 20%. O valor `0` (padrão) exige que a predecessora termine 100% antes. Ver CPM abaixo. |

Operações sem sucessor simplesmente não aparecem como `predecessor_id` em nenhuma aresta. A última operação do roteiro não precisa de nenhum registro especial — ela contribui naturalmente para o cálculo de lead time.

---

### Como cadastrar um roteiro (passo a passo)

**Passo 1 — Criar operações genéricas (uma vez; ficam na biblioteca)**

```http
POST /api/routing/operations
{
  "name": "Corte a laser",
  "origin": "INTERNA",
  "standard_time": 0.5
}

POST /api/routing/operations
{
  "name": "Pintura eletrostática",
  "origin": "EXTERNA",
  "standard_time": 2.0
}
```

Operações criadas uma única vez e reutilizadas em múltiplos roteiros.

---

**Passo 2 — Criar o roteiro do item**

```http
POST /api/routing/routes
{
  "item_code": 1001,
  "description": "Roteiro Padrão – Produto X",
  "alternative": 1,
  "is_standard": true,
  "created_by": "uuid-do-usuario"
}
→ { "id": 7, "item_code": 1001, "is_standard": true, ... }
```

---

**Passo 3 — Adicionar operações ao roteiro**

```http
POST /api/routing/route-operations/7
{ "operation_id": 1, "sequence": 10, "work_center_id": 2, "standard_time": 0.5 }

POST /api/routing/route-operations/7
{ "operation_id": 3, "sequence": 20, "work_center_id": 4, "standard_time": 1.5, "setup_time": 0.25 }

POST /api/routing/route-operations/7
{ "operation_id": 5, "sequence": 30, "work_center_id": 2, "standard_time": 0.5 }
```

---

**Passo 4 — Definir a ordem de dependência entre as operações**

As dependências definem qual operação precisa terminar (ou estar suficientemente avançada) antes de outra começar.

```http
POST /api/routing/routes/7/edges
{
  "predecessor_id": 10,
  "successor_id": 20,
  "overlap_pct": 0
}
```
→ Op 20 só começa quando a op 10 terminar completamente.

```http
POST /api/routing/routes/7/edges
{
  "predecessor_id": 20,
  "successor_id": 30,
  "overlap_pct": 20
}
```
→ Op 30 pode começar quando restar apenas 20% do tempo da op 20 (`overlap_pct = 20`).
**Atenção:** esse overlap só vale se o centro de trabalho da op 20 tiver `requires_operator = false`. Se for máquina manual, o sistema ignora o overlap e exige que a op 20 termine 100% antes.

A operação 30 não precisa de nenhum registro adicional — por não aparecer como predecessora de ninguém, o sistema já sabe que ela é a última.

---

### Lead Time via CPM

O lead time de fabricação responde a uma pergunta simples: **"quanto tempo leva para fabricar este produto do zero?"** Não é a soma de todas as operações — é o tempo do **caminho mais lento**, levando em conta quais etapas podem acontecer ao mesmo tempo e quais precisam esperar.

O sistema usa o **CPM (Método do Caminho Crítico)** para esse cálculo. O MRP usa esse número para planejar quando emitir uma ordem: se o lead time é 5 dias e a entrega é dia 20, a ordem precisa ser emitida no dia 15.

---

#### O que é o caminho crítico?

Imagine um roteiro de 3 operações:

```
[Corte — 2h] ──────────────────────────────────→ [Montagem — 1h]
                                                      ↑
[Solda — 3h] ──────────────────────────────────→ ──┘
```

Corte e Solda acontecem **em paralelo** (máquinas diferentes ao mesmo tempo). Montagem só começa depois que **ambas** terminam.

- Corte termina em 2h
- Solda termina em 3h
- Montagem começa quando a **mais lenta** termina → começa em 3h, termina em 4h

**Lead time = 4h** (não 2+3+1=6h, porque corte e solda correm juntos)

O caminho mais lento (Solda → Montagem) é o **caminho crítico**. Adiantar o Corte não muda nada; adiantar a Solda encurta o lead time.

---

#### O que é `overlap_pct`?

Em algumas etapas automáticas, a próxima máquina não precisa esperar o lote **todo** terminar — ela pode começar a trabalhar assim que as **primeiras peças** chegam.

Exemplo: Corte a laser (automático) produz 100 peças. A dobradeira pode começar quando as primeiras 20% chegarem, sem esperar as outras 80%.

```
overlap_pct = 20  →  "a próxima operação pode começar quando 20% da duração
                       anterior ainda resta para terminar"
```

```
Tempo:  0h ──────────────────────────── 4h
Corte:  [==============================]   (termina em 4h)
                          [os últimos 20% = 0.8h ainda correm]
Dobra:              [==============]        (começa em 3.2h, termina em 5.2h)
```

**`overlap_pct = 0` (padrão):** a próxima operação só começa depois que a anterior **termina completamente**.

> **Máquinas manuais nunca têm overlap válido.** Um operador não abandona a fresadora no meio de uma peça para ir operar outra máquina. Quando o centro de trabalho tem `requires_operator = true`, o sistema **ignora** qualquer `overlap_pct` configurado e trata a operação como se fosse 0. Isso evita que o lead time seja subestimado.

---

#### Como o cálculo funciona, passo a passo

O sistema percorre as operações do roteiro na ordem das dependências e calcula para cada uma: **"o mais cedo que ela pode terminar"**.

**Regra 1 — Operação sem predecessora** (primeira do roteiro ou que começa em paralelo):
```
termina_cedo = duração da operação
```
Ela começa em t=0, pois não depende de nada.

**Regra 2 — Operação com predecessora(s):**
```
começa_cedo  = termina_cedo[predecessora]  −  overlap × duração[predecessora]
termina_cedo = começa_cedo + duração da operação
```
Se houver **mais de uma predecessora**, usa-se a que resultar no `começa_cedo` mais tardio — ela é o gargalo.

Quando `requires_operator = true` na predecessora, o overlap é tratado como 0:
```
começa_cedo = termina_cedo[predecessora]   (espera terminar 100%)
```

**Lead time final = maior `termina_cedo` entre todas as operações.**

---

#### Exemplo completo — roteiro de 3 etapas em série

Roteiro: Preparação (2h) → Usinagem manual (3h) → Acabamento automático (1h)

A aresta Usinagem→Acabamento tem `overlap_pct = 20` configurada. Porém, Usinagem é **manual** (`requires_operator = true`), então o overlap é ignorado.

```
Preparação:
  sem predecessora → termina_cedo = 2h

Usinagem (manual):
  predecessora = Preparação (termina em 2h)
  overlap forçado a 0 (requires_operator = true)
  começa_cedo  = 2h − 0 × 2h = 2h
  termina_cedo = 2h + 3h = 5h

Acabamento (automático, mas predecessora é manual):
  predecessora = Usinagem (termina em 5h, duração 3h)
  overlap forçado a 0 (predecessora é manual)
  começa_cedo  = 5h − 0 × 3h = 5h
  termina_cedo = 5h + 1h = 6h

Lead Time = max(2h, 5h, 6h) = 6h
```

Se a Usinagem fosse **automática** (`requires_operator = false`) e o overlap de 20 (`overlap_pct = 20` → 20 ÷ 100 = 0,20 na fórmula) valesse:

```
Acabamento:
  começa_cedo  = 5h − 0.20 × 3h = 5h − 0.6h = 4.4h
  termina_cedo = 4.4h + 1h = 5.4h

Lead Time = max(2h, 5h, 5.4h) = 5.4h   ← 36 minutos a menos
```

> Esses 36 minutos de diferença podem parecer pequenos, mas multiplicados por dezenas de ordens por dia, o erro acumula. Um lead time subestimado faz o MRP emitir ordens tarde — a ordem chega atrasada no chão de fábrica.

---

### Endpoints do módulo de roteiro

| Método | Rota | Descrição |
|--------|------|-----------|
| POST | `/api/routing/operations` | Criar operação genérica |
| GET | `/api/routing/operations` | Listar operações |
| GET | `/api/routing/operations/{id}` | Buscar operação |
| PUT | `/api/routing/operations/{id}` | Atualizar operação |
| DELETE | `/api/routing/operations/{id}` | Desativar operação |
| POST | `/api/routing/routes` | Criar roteiro |
| GET | `/api/routing/routes` | Listar roteiros por item |
| GET | `/api/routing/routes/{id}` | Buscar roteiro com operações e rede |
| PUT | `/api/routing/routes/{id}` | Atualizar roteiro |
| DELETE | `/api/routing/routes/{id}` | Desativar roteiro |
| POST | `/api/routing/route-operations/{routeId}` | Adicionar operação ao roteiro |
| PUT | `/api/routing/route-operations/{routeId}/{opId}` | Atualizar operação do roteiro |
| DELETE | `/api/routing/route-operations/{routeId}/{opId}` | Remover operação do roteiro |
| GET | `/api/routing/routes/{id}/edges` | Listar dependências da rede |
| POST | `/api/routing/routes/{id}/edges` | Criar dependência predecessor→sucessor |
| DELETE | `/api/routing/routes/{id}/edges` | Remover dependência |
| GET | `/api/routing/routes/{id}/lead-time` | Calcular lead time via CPM |

---

## 2. CRP — Capacity Requirements Planning

### O que é

O CRP responde a uma pergunta direta: **"a fábrica tem horas suficientes nos centros de trabalho para executar todas as ordens planejadas?"**

Ele soma tudo que precisa ser feito (horas de cada operação × quantidade) e compara com quanto cada centro de trabalho tem disponível no dia. Se precisar de 12h e o centro trabalha 8h, está sobrecarregado.

O CRP **não rearranja as ordens** — ele só mostra onde está o problema. Quem resolve é o PCP (adiando datas, autorizando hora extra, terceirizando) ou o APS (que redistribui automaticamente).

### Onde o CRP se encaixa no fluxo

```
MRP gera sugestões de ordens
         ↓
PCP analisa e aprova as ordens
         ↓
PCP roda o CRP  ← "essas ordens são viáveis na capacidade atual?"
         ↓
CRP mostra quais centros de trabalho estão sobrecarregados e em quais dias
         ↓
PCP decide: adiar ordens / autorizar hora extra / terceirizar
         ↓
PCP libera as ordens para o chão de fábrica
```

**O CRP é acionado manualmente pelo PCP** porque a decisão de ajustar ordens é humana — o sistema não sabe se vale a pena pagar hora extra ou se é melhor atrasar a entrega. O PCP pode rodá-lo a qualquer momento, inclusive para simular cenários ("e se eu aprovar todas essas sugestões do MRP?").

### Como o cálculo funciona

Para cada ordem com roteiro, o CRP olha cada operação e acumula:

```
horas necessárias no centro X no dia D  +=  tempo da operação  ×  quantidade da ordem
```

Depois, para cada centro de trabalho em cada dia:

```
horas disponíveis  =  capacidade nominal do centro (ex: 8h/dia)
                    − horas bloqueadas por manutenção agendada naquele dia

carga (%)  =  horas necessárias  ÷  horas disponíveis  ×  100
```

Se a carga ultrapassar 100%, o centro está sobrecarregado naquele dia.

#### Exemplo prático

3 ordens precisam passar pela fresadora no mesmo dia:

| Ordem | Quantidade | Tempo/peça | Horas necessárias |
|-------|-----------|------------|-------------------|
| OP-101 | 10 peças | 0.5h | 5h |
| OP-102 | 8 peças  | 0.5h | 4h |
| OP-103 | 6 peças  | 0.5h | 3h |

Total necessário: **12h**. Fresadora trabalha 8h/dia, com 1h de manutenção preventiva agendada → **7h disponíveis**.

```
Carga = 12h ÷ 7h × 100 = 171%  →  SOBRECARGA
```

O CRP retorna isso no relatório e o PCP sabe que precisa redistribuir essas ordens.

### Integração com Manutenção Preventiva

Quando existe uma ordem de manutenção (`PLANNED` ou `IN_PROGRESS`) para um centro de trabalho em uma data, o CRP desconta essas horas da capacidade disponível antes de calcular a carga.

```
Centro X, dia 10/06:
  Capacidade nominal:      8h
  Manutenção preventiva: − 2h
  Disponível para produção: 6h
```

Isso evita que o PCP planeje produção em horários que a máquina estará parada para manutenção.

### Endpoints

| Método | Rota | Descrição |
|--------|------|-----------|
| POST | `/api/crp/calculate` | Calcular CRP para um plano MRP |
| GET | `/api/crp/plans/{planCode}` | Listar todos os registros de capacidade do plano |
| GET | `/api/crp/plans/{planCode}/overload` | Listar apenas centros sobrecarregados |
| GET | `/api/crp/work-centers/{id}?from=&to=` | Capacidade de um centro em um período |

**`POST /api/crp/calculate`:**
```json
{ "plan_code": 42 }
```
**Resposta:**
```json
{ "plan_code": 42, "total_entries": 18, "overload_count": 2 }
```

**`GET /api/crp/plans/42/overload`:**
```json
[
  {
    "work_center_id": 3,
    "req_date": "2026-06-10",
    "required_hours": 12.5,
    "available_hours": 8.0,
    "load_pct": 156.25,
    "is_overloaded": true
  }
]
```

---

## 3. APS — Advanced Planning and Scheduling

### O que é

O APS resolve um problema que o CRP não resolve: **quando exatamente cada operação começa e termina?**

Enquanto o CRP diz "o centro X está sobrecarregado na sexta-feira", o APS diz "a OP-101 começa na fresadora às 07h00 e termina às 09h30; a OP-102 começa às 09h30 e termina às 12h00." Ele produz um **Gantt** — um calendário detalhado de produção.

### Diferença entre CRP e APS

| | CRP | APS |
|-|-----|-----|
| Pergunta | O centro tem horas suficientes? | Quando exatamente cada ordem é executada? |
| Precisão | Por dia (turno) | Por hora (minuto) |
| Quando sobra capacidade | Mostra % de carga | Distribui as ordens no tempo |
| Quando falta capacidade | Aponta sobrecarga | Empurra ordens para os próximos slots |
| Resultado | Relatório de carga | Gantt com horários |

### Como o APS pensa

O APS funciona como um **agendador de consultas médicas**: cada centro de trabalho tem uma agenda, e cada operação ocupa um slot nessa agenda. Nenhum centro pode ter dois trabalhos ao mesmo tempo.

O algoritmo percorre as ordens em ordem de prioridade (mais urgente primeiro) e, para cada operação, encontra o **primeiro slot livre** no centro de trabalho, respeitando duas restrições:

1. **O centro precisa estar livre** — se outra operação já está ocupando aquele horário, espera terminar
2. **A operação anterior da mesma ordem precisa ter terminado** — você não pode começar a pintar antes de terminar de soldar

```
┌─────────────────────────────────────────────────────┐
│  Fresadora (Centro 3)  —  Agenda do dia 10/06       │
├──────────────┬──────────────┬───────────────────────┤
│ 07:00–09:30  │ 09:30–12:00  │ 12:00–14:30           │
│   OP-101     │   OP-102     │   OP-103               │
└──────────────┴──────────────┴───────────────────────┘
```

Se a OP-104 também precisar da fresadora e não couber no dia 10/06, ela vai para o dia 11/06 — não fica pendurada no mesmo dia causando conflito.

### Máquinas manuais

Para máquinas com `requires_operator = true`, o comportamento é idêntico ao de qualquer outra máquina: **o centro só recebe uma operação por vez**. O operador termina o que está fazendo antes de começar o próximo — exatamente o que acontece na prática.

O APS já faz isso naturalmente: ele nunca aloca duas operações simultâneas no mesmo centro, seja manual ou automático.

> **Premissa atual:** cada centro de trabalho manual tem um operador dedicado. Um operador que alterna entre dois centros diferentes exigiria cadastro de operadores com capacidade própria — não implementado nesta versão.

### Ordem de prioridade

As ordens são sequenciadas por:
1. **Prioridade** (campo numérico — menor número = mais urgente)
2. **Data de necessidade** (a que precisa ser entregue primeiro sai na frente)

Essa lógica se chama **EDD (Earliest Due Date)** — minimiza atrasos priorizando quem está mais próximo do vencimento.

### Como a duração é calculada

Para cada operação, a duração alocada no Gantt é:

```
duração = setup_time + planned_time
```

O sistema respeita o **limite de horas disponíveis por dia** do centro (padrão: 8h). Se uma operação precisar de 10h, ela é dividida entre dois dias úteis automaticamente. Fins de semana são pulados.

### Endpoints

| Método | Rota | Descrição |
|--------|------|-----------|
| POST | `/api/aps/sequence` | Gerar sequenciamento de todas as ordens abertas |
| GET | `/api/aps/gantt/order/{orderID}` | Ver Gantt de uma ordem específica |
| POST | `/api/aps/gantt/work-center` | Ver Gantt de um centro em um período |

**Gantt (trecho de resposta):**
```json
[
  {
    "sequence_id": 1,
    "production_order_id": 101,
    "work_center_id": 3,
    "sequence_position": 10,
    "scheduled_start": "2026-06-10T07:00:00Z",
    "scheduled_end": "2026-06-10T09:30:00Z",
    "duration_hours": 2.5,
    "status": "SCHEDULED"
  },
  {
    "sequence_id": 2,
    "production_order_id": 101,
    "work_center_id": 5,
    "sequence_position": 20,
    "scheduled_start": "2026-06-10T09:30:00Z",
    "scheduled_end": "2026-06-10T11:30:00Z",
    "duration_hours": 2.0,
    "status": "SCHEDULED"
  }
]
```

---

## 4. Custo Padrão

### O que é

Calcula o custo de fabricação de um item considerando materiais (BOM) e mão de obra/máquina (roteiro). Suporta rollup multinível.

### Fórmula

```
custo_material  = Σ (qtd_componente × custo_padrão_componente)
custo_operação  = Σ (tempo_operação × taxa_centro_trabalho)
custo_overhead  = custo_operação × taxa_overhead (%)
custo_total     = custo_material + custo_operação + custo_overhead
```

### Endpoints

| Método | Rota | Descrição |
|--------|------|-----------|
| POST | `/api/standard-cost/calculate/{itemCode}` | Calcular custo padrão |
| GET | `/api/standard-cost/{itemCode}` | Buscar custo padrão salvo |
| GET | `/api/standard-cost/` | Listar todos os custos padrão |

---

## 5. Qualidade

### O que é

Registra pontos de inspeção ao longo do processo produtivo com laudo (aprovado/reprovado/condicional), quantidades e observações.

### Tipos de ponto de inspeção

| Tipo | Momento |
|------|---------|
| `RECEIVING` | Inspeção de recebimento (matéria-prima) |
| `IN_PROCESS` | Durante a fabricação, após uma operação |
| `FINAL` | Produto acabado, antes do estoque |

### Endpoints

| Método | Rota | Descrição |
|--------|------|-----------|
| POST | `/api/quality/inspection-points` | Criar ponto de inspeção |
| GET | `/api/quality/inspection-points` | Listar pontos |
| POST | `/api/quality/inspection-points/{id}/results` | Registrar laudo |
| GET | `/api/quality/inspection-points/{id}/results` | Buscar resultado |

---

## 6. Manutenção Preventiva

### O que é

Gerencia planos de manutenção periódica de máquinas e centros de trabalho. Gera ordens automaticamente conforme a frequência definida. As horas agendadas são descontadas da capacidade pelo CRP.

### Entidades

**Plano de Manutenção (`maintenance_plans`):**

| Campo | Descrição |
|-------|-----------|
| `machine_id` | Máquina que receberá manutenção |
| `work_center_id` | Centro de trabalho (afeta capacidade no CRP) |
| `frequency` | `DAILY`, `WEEKLY`, `MONTHLY`, `CUSTOM_DAYS` |
| `frequency_days` | Intervalo em dias |
| `estimated_hours` | Horas estimadas de parada |
| `next_scheduled_at` | Calculado automaticamente |

**Ordem de Manutenção (`maintenance_orders`):**

| Campo | Descrição |
|-------|-----------|
| `plan_id` | Plano de origem |
| `machine_id` | Máquina (copiado do plano) |
| `scheduled_date` | Data programada |
| `status` | `PLANNED` → `IN_PROGRESS` → `DONE` / `CANCELLED` |
| `actual_hours` | Preenchido ao concluir |
| `started_at` / `completed_at` | Timestamps automáticos na mudança de status |

### Ciclo de vida

```
Plano criado
    ↓ GenerateOrders (disparo manual ou periódico)
Ordem PLANNED  (idempotente: não cria duplicata para mesmo plano+data)
    ↓ AdvanceOrder {status: "IN_PROGRESS"}
Ordem IN_PROGRESS  (registra started_at)
    ↓ AdvanceOrder {status: "DONE", actual_hours: 1.5}
Ordem DONE  (registra completed_at)
```

### Endpoints

| Método | Rota | Descrição |
|--------|------|-----------|
| POST | `/api/maintenance/plans` | Criar plano |
| GET | `/api/maintenance/plans` | Listar planos (`?active=true`) |
| GET | `/api/maintenance/plans/{id}` | Buscar plano |
| GET | `/api/maintenance/machines/{machineId}/plans` | Planos de uma máquina |
| DELETE | `/api/maintenance/plans/{id}` | Desativar plano |
| POST | `/api/maintenance/orders` | Criar ordem manual |
| PUT | `/api/maintenance/orders/{id}/advance` | Avançar status / registrar horas reais |
| GET | `/api/maintenance/plans/{planId}/orders` | Ordens de um plano |
| GET | `/api/maintenance/work-centers/{wcId}/orders?from=&to=` | Ordens por período |
| POST | `/api/maintenance/orders/generate` | Gerar ordens automáticas (`{ "horizon_days": 30 }`) |

---

## 7. Previsão Estatística

### O que é

Calcula previsões de demanda futura aplicando modelos estatísticos a uma série histórica. Retorna o modelo de melhor ajuste (menor MAPE).

### Modelos disponíveis

| Modelo | Quando é melhor |
|--------|----------------|
| Holt-Winters (aditivo) | Séries com tendência + sazonalidade |
| Suavização Exponencial | Séries com tendência sem sazonalidade |
| Média Móvel (k=3) | Séries estáveis / sem padrão claro |
| Média Móvel (k=6) | Séries estáveis com mais histórico |

O sistema calcula o MAPE de cada modelo e retorna o de menor erro. O campo `model_used` indica qual foi selecionado.

### Endpoint

**`POST /api/forecast/statistical`**

```json
{
  "item_code": 1001,
  "history": [
    { "period": "2026-01", "quantity": 120.0 },
    { "period": "2026-02", "quantity": 135.0 },
    { "period": "2026-03", "quantity": 118.0 },
    { "period": "2026-04", "quantity": 142.0 },
    { "period": "2026-05", "quantity": 130.0 }
  ],
  "periods_ahead": 3
}
```

**Resposta:**
```json
{
  "item_code": 1001,
  "model_used": "exponential_smoothing",
  "mape": 4.82,
  "forecasts": [
    { "period": "2026-06", "quantity": 133.2 },
    { "period": "2026-07", "quantity": 135.8 },
    { "period": "2026-08", "quantity": 134.5 }
  ]
}
```

> A previsão é calculada em tempo real e não é persistida automaticamente. Para armazenar, use `POST /api/sales-forecast/blocks`.

---

## 8. Alertas de Exceções MRP

### O que é

Após o MRP rodar, exceções são geradas para situações que exigem atenção do PCP. Este módulo consolida e envia os alertas via **webhook** e/ou **e-mail**.

### Tipos de exceção

| Tipo | Significado |
|------|-------------|
| `LATE_ORDER` | Ordem planejada com data de necessidade no passado |
| `OVERDUE_PURCHASE` | Sugestão de compra com prazo vencido |
| `EXCESS_STOCK` | Estoque projetado acima do máximo definido |
| `OPEN_ORDER_NO_DEMAND` | Ordem aberta sem demanda correspondente |
| `CAPACITY_OVERLOAD` | Centro de trabalho sobrecarregado no período |

### Canais de notificação

| Canal | Campo no body | Requisito |
|-------|--------------|-----------|
| Webhook HTTP | `webhook_url` | URL do sistema destino |
| E-mail | `email_to` | SMTP configurado via `.env` |

Ambos os canais funcionam simultaneamente. Se o SMTP não estiver configurado, o e-mail é silenciosamente ignorado sem afetar o webhook.

### Configuração SMTP (`.env`)

```dotenv
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=seu@email.com
SMTP_PASSWORD=sua_senha_app
SMTP_FROM=erp@suaempresa.com
```

### Endpoint

**`POST /api/mrp-calculation/exceptions/notify`**

```json
{
  "plan_code": 42,
  "webhook_url": "https://chat.empresa.com/mrp-alerts",
  "email_to": ["pcp@empresa.com", "gerencia@empresa.com"]
}
```

**Resposta:**
```json
{
  "plan_code": 42,
  "generated_at": "2026-05-22T10:00:00Z",
  "total": 3,
  "by_type": { "LATE_ORDER": 2, "EXCESS_STOCK": 1 },
  "exceptions": [
    {
      "item_code": 1001,
      "message_type": "LATE_ORDER",
      "description": "Ordem planejada para 2026-05-18, já vencida"
    }
  ]
}
```

**Corpo do e-mail gerado:**
```
Relatório de Exceções MRP — Plano 42
Gerado em: 22/05/2026 10:00
Total de exceções: 3

Por tipo:
  LATE_ORDER                     2
  EXCESS_STOCK                   1

Detalhes:
  Item 1001   [LATE_ORDER              ] Ordem planejada para 2026-05-18, já vencida
  Item 1002   [LATE_ORDER              ] ...
  Item 1008   [EXCESS_STOCK            ] Estoque projetado acima do máximo
```

---

## 9. Restrições e Configurador

### O que é

Permite definir regras de negócio que controlam quais combinações de atributos de um item são válidas. Útil em configuradores de produto ou validações de cadastro.

### Operadores suportados

`==`, `!=`, `>`, `<`, `>=`, `<=`, `IN`, `NOT_IN`

### Endpoints

| Método | Rota | Descrição |
|--------|------|-----------|
| POST | `/api/restrictions` | Criar restrição |
| GET | `/api/restrictions` | Listar restrições |
| GET | `/api/restrictions/{id}` | Buscar restrição |
| POST | `/api/restrictions/{id}/evaluate` | Avaliar restrição com um contexto |
| DELETE | `/api/restrictions/{id}` | Remover restrição |

---

## 10. Fornecedores e Sugestão de Compra (MRP → Compras)

### O que é

Cadastro de fornecedores/transportadoras e o fluxo que transforma sugestões de
compra do MRP em pedidos de compra. A documentação completa do cadastro (campos,
pastas, parâmetros, regras de IE/MEI/vitícola/SEFAZ e endpoints) está em
[`cadastros-fornecedor.md`](cadastros-fornecedor.md).

### Integrações principais

- **Pedido de Compra** — `purchase_orders.supplier_code` tem FK para `suppliers`. Ao
  criar um pedido com fornecedor e sem condição de pagamento, ela é preenchida a
  partir do cadastro do fornecedor (provider `SupplierPurchasingDefaultsProvider`).
- **Fiscal (NF de entrada)** — a importação de NF-e de compra casa o CNPJ do emitente
  a um fornecedor cadastrado e grava o vínculo em `fiscal_entries.supplier_code`.
  (Ver Módulo Fiscal & Financeiro.)

### Sugestão de compra (MRP → PCP/Compras)

Uma sugestão de compra é uma `planned_order` do tipo `PURCHASE` ainda não firme
(`is_firm = false`, `status = PLANNED`). O PCP/Compras decide:

| Ação | Efeito |
| --- | --- |
| **Aprovar** | Gera `purchase_order` (`origin = MRP`, `status = APPROVED`, firme) com o fornecedor escolhido + item da sugestão; torna a `planned_order` firme (`status = RELEASED`). |
| **Rejeitar** | `status = CANCELLED` e inativa a sugestão. |

Somente suprimentos firmes/aprovados entram no netting do MRP — sugestões pendentes
não reduzem a necessidade líquida.

**Endpoints** (sob `/api/purchase-order`):
- `GET  /suggestions` — lista sugestões abertas.
- `POST /suggestions/{code}/approve` — corpo: `enterprise_code`, `supplier_code`, `unit_price`, `notes`, `created_by`.
- `POST /suggestions/{code}/reject`.

---

## 11. Conversão de UM por Item

### O que é

Cadastro de **Conversões por Item** (migration `000138`, tabela `item_unit_conversions`):
fatores de conversão entre unidades de medida de um item (ex.: `1 CX = 12 UN`),
usado quando a UM de compra difere da UM de estocagem. Atende ao requisito do Pedido
de Compra: "caso não exista fator de conversão para a UM da pasta Estoque, abrir o
Cadastro de Conversões por Item".

### Como funciona

- Cada registro define `1 from_uom = factor × to_uom` para um `item_code`.
- A resolução tenta a conversão **direta** (from→to); se ausente, usa a **inversa**
  (`1/factor`); UMs iguais retornam fator 1. Sem cadastro → erro orientando a cadastrar.
- Conversões expostas como porta `ports.UOMConverter` (`Factor`, `ConvertQuantity`,
  `ConvertUnitPrice`), que o **Pedido de Compra** consumirá para calcular UM interna,
  Qtde. Interna e Preço Interno do item.

### Endpoints (`/api/item-conversions`)

- `POST /` — cadastrar fator (`item_code`, `from_uom`, `to_uom`, `factor`); upsert por chave.
- `GET /item/{itemCode}` — listar conversões ativas do item.
- `GET /convert?item=&from=&to=&qty=` — resolver fator e quantidade convertida.
- `DELETE /{id}` — inativar uma conversão.

---

## 12. Tabela de Preço de Compra

### O que é

Cadastro de **Tabelas de Preço de Compra** (migration `000139`): tabelas
`purchase_price_tables` (cabeçalho: código, descrição, moeda, vigência) e
`purchase_price_table_items` (preço por item, com UM e quantidade mínima). O preço
pode ser **genérico** (qualquer fornecedor) ou **específico por fornecedor**
(`supplier_code`).

### Integração

- O `supplier_enterprises.purchase_price_table_id` (pasta Empresas do fornecedor)
  agora tem **FK** para `purchase_price_tables(id)` — define a tabela default do
  fornecedor.
- Exposta como porta `ports.PurchasePriceProvider` (`GetItemPrice`), que o **Pedido
  de Compra** usará para trazer o preço do item automaticamente (1º nível da
  hierarquia de UM/preço do spec: Tabela de Preço Compra). A resolução prefere a
  linha específica do fornecedor e cai para a genérica.

### Endpoints (`/api/purchase-price-tables`)

- `POST /` · `PUT /` · `GET /` · `GET /{code}` (com itens).
- `POST /items` (upsert por tabela+item+fornecedor) · `GET /{code}/items` ·
  `DELETE /items/{id}`.

---

## 13. Pedido de Compra (completo)

### O que é

Pedido de compra com capa e itens ricos (migration `000140` estende
`purchase_orders` e `purchase_order_items`). Integra os módulos 1–3: o item resolve
**preço** (Tabela de Preço de Compra), **UM interna / Qtde. interna / Preço interno**
(Conversões por Item) e **% IPI** (Classificação Fiscal) automaticamente.

### Capa (campos estendidos)

Tabela de preço, tipo de NF, conta financeira, tipo de solicitação, data da moeda;
**transporte** (tipo de frete, tipo/modo/valor do frete, transportadora);
**redespacho** (transportadora, tipo e valor); **adiantamento** (data/valor);
**importação** (incoterm, data de embarque); nº do talão e status de alçada (A/B/R/I/N).
Na criação, quando há fornecedor e os campos não são informados, **condição de
pagamento, tabela de preço, tipo de NF, conta financeira e tipo de frete** são
puxados dos defaults do fornecedor (`SupplierPurchasingDefaultsProvider`).

### Item (resolução automática)

Ao adicionar um item (`POST /api/purchase-order/{code}/items`):

1. **Preço** — se não informado e a capa tiver tabela de preço, vem de
   `PurchasePriceProvider` (preferindo preço específico do fornecedor).
2. **% IPI** — se não informado e o item tiver classificação fiscal, vem de
   `FiscalClassificationProvider.GetIPIRate`.
3. **UM interna** — se UM de compra ≠ UM de estoque, `UOMConverter` calcula
   `internal_qty` e `internal_price` (com inversa 1/fator quando necessário).

Demais campos do item: desconto, % ICMS / ICMS-ST, tolerância, datas de
entrega/prometida, tipo de operação, tipo de NF, conta contábil, centro de custo,
solicitante, contrato, cotação e tipo de utilização
(INDUSTRIALIZACAO/CONSUMO/IMOBILIZADO).

### Endpoints

- `POST /api/purchase-order/create` — capa (com defaults do fornecedor).
- `POST /api/purchase-order/{code}/items` — adicionar item (com resolução automática).
- Demais: `GET /list`, `GET /{code}`, `PUT /{code}`, `DELETE /{code}/cancel`,
  `GET /supplier/{supplierCode}`, `GET /status/{status}`, e o fluxo de sugestões
  (`/suggestions...`).

---

## 14. Fornecedor Preferencial por Item

### O que é

Cadastro que liga um **item** a **fornecedores** com ranking de preferência (migration
`000141`, tabela `item_preferred_suppliers`). Também serve como **Descrição de Itens
por Fornecedor**: guarda o código, a descrição e a UM do item no fornecedor (2º nível
da hierarquia de UM/descrição do Pedido de Compra).

### Integração

- Exposto como porta `ports.PreferredSupplierProvider` (`GetPreferredSupplier` →
  fornecedor de menor ranking), consumida pela **Geração de Pedidos a partir de
  Solicitações** para sugerir o fornecedor de cada item.

### Endpoints (`/api/item-suppliers`)

- `POST /` — vincular/atualizar (upsert por item+fornecedor): ranking, código/descrição/UM
  no fornecedor, lead time.
- `GET /item/{itemCode}` — listar fornecedores do item (por ranking).
- `DELETE /{id}` — desvincular.

---

## 15. Solicitação de Compra → Geração de Pedidos

### O que é

Solicitação de compra (migration `000142`, `purchase_requisitions` +
`purchase_requisition_items`) e o programa que **gera pedidos de compra a partir das
solicitações**, agrupando por fornecedor.

### Solicitação

Cabeçalho (código, empresa, tipo de solicitação, solicitante, emissão, status) e itens
(item, quantidade, UM, centro de custo, conta contábil, valor sugerido, data de
entrega, aplicação, tipo de utilização). O **saldo** do item = quantidade − atendida −
cancelada. O status do item evolui OPEN → PARTIAL → ATTENDED conforme o atendimento.

### Geração de Pedidos (`POST /api/purchase-requisitions/generate-orders`)

Recebe seleções `{requisition_item_id, qty_to_attend, supplier_code?}` e:

1. Resolve o **fornecedor** de cada item — informado ou o **preferencial** (módulo 14);
   itens sem fornecedor entram em `skipped`.
2. **Agrupa por fornecedor** e gera um pedido de compra por grupo (`APPROVED`, firme),
   puxando do fornecedor a condição de pagamento, tabela de preço, tipo de NF, conta
   financeira e frete (defaults do fornecedor).
3. **Preço** do item: valor sugerido da solicitação ou, se ausente, da tabela de preço.
4. **Registra o atendimento** de volta na solicitação (atendida += qtde, limitada ao
   saldo), atualizando o status do item.

Retorna os pedidos gerados e a lista de itens não atendidos (`skipped`).

### Endpoints (`/api/purchase-requisitions`)

- `POST /` (com itens) · `GET /` (`?only_open=true`) · `GET /{code}` (com itens) ·
  `POST /{code}/items` · `POST /generate-orders`.

---

## 16. Cotação de Compra

### O que é

Cotação de compra (migration `000143`): libera itens de **solicitações de compra** e
**ordens planejadas** para cotação, registra os **preços dos fornecedores**, permite
**selecionar o vencedor** e **gerar pedidos** a partir das seleções. Quatro tabelas:
`purchase_quotations`, `purchase_quotation_items`, `purchase_quotation_suppliers`,
`purchase_quotation_prices`.

### Fluxo

1. **Liberar para cotação** (`POST /`): informa `requisition_item_ids` e/ou
   `planned_order_codes` (+ fornecedores convidados). Cada item vira um item de cotação
   guardando a origem (`REQUISITION`/`PLANNED_ORDER`) para rastreio.
2. **Registrar preços** (`POST /prices`): preço/lead time/condição de pagamento por
   item × fornecedor (upsert). A cotação passa a `QUOTED`.
3. **Selecionar** (`PATCH /prices/{priceID}/select`): marca o preço vencedor do item
   (limpa os demais do mesmo item, em transação).
4. **Gerar pedidos** (`POST /{code}/generate-orders`): agrupa os preços selecionados
   por fornecedor, gera um pedido de compra por fornecedor (com defaults do fornecedor)
   e **registra o atendimento** nos itens de solicitação de origem; fecha a cotação.

### Endpoints (`/api/purchase-quotations`)

- `POST /` · `GET /` (`?only_open=true`) · `GET /{code}` (itens + preços + fornecedores)
- `POST /{code}/suppliers` · `POST /prices` · `PATCH /prices/{priceID}/select`
- `POST /{code}/generate-orders`

> Integra os módulos de Solicitação (15), Fornecedor preferencial/defaults e Pedido de
> Compra completo (13).

---

## 17. Pipeline de Planejamento (MRP → CRP → APS)

### O que é
Um único disparo que encadeia os três motores de planejamento que antes eram
chamados separadamente, devolvendo um **parecer de viabilidade consolidado**.

### Como funciona
1. **MRP** — explode a BOM e gera ordens planejadas (`generate_llc`).
2. **CRP** — soma a carga por centro de trabalho/dia e detecta sobrecarga.
3. **APS** — sequencia as ordens em capacidade finita (EDD).

O resultado traz itens/ordens do MRP, entradas e contagem de sobrecarga do CRP,
operações sequenciadas do APS e o veredito `viable` (falso quando o MRP não
concluiu ou o CRP achou sobrecarga), com `notes` explicativas.

### Endpoint
- `POST /api/planning/run-pipeline` — body `{ "plan_code": <P>, "generate_llc": true, "start_from": "2026-06-10T00:00:00Z" }`.
  Requer o escopo `planning:run` (ver §20). Implementado em
  `planning_uc.RunPlanningPipelineUseCase`. As chamadas individuais
  (`/api/mrp-calculation/run`, `/api/crp/calculate`, `/api/aps/sequence`) seguem disponíveis.

## 18. Backflush no apontamento

### O que é
Baixa automática dos componentes da estrutura (BOM) ao **apontar** produção, em
proporção à quantidade produzida.

### Como funciona
No `POST /api/production-order/appointment`, informando `backflush_warehouse_id`,
o sistema resolve a BOM do item da OF (`GetDirectChildrenForMask` quando há
máscara, senão `GetAllDirectChildren`) e gera um movimento **`OUT`** por componente:
`consumo = qtd_produzida × qtd_componente × (1 + perda%/100)` (fórmula 1). Os
movimentos atualizam o saldo (ver Estoque). Omitir `backflush_warehouse_id`
desliga o backflush para aquele apontamento. Implementado em
`production_order_uc/add_appointment_uc.go`.

## 19. Expedição / Carregamento (romaneio) — migration 000146

### O que é
Logística de saída: separação, conferência e despacho de mercadorias por
**romaneio** (shipment). Complementa — sem substituir — a baixa fiscal da NF-e de
saída (ver `fiscal-financeiro.md`).

### Ciclo de vida
`OPEN` → `SEPARATED` → `CONFERRED` → `SHIPPED` (`CANCELLED`). O despacho (`ship`)
exige **todos os itens conferidos**.

### Endpoints (`/api/shipments`)
| Ação | Endpoint |
|---|---|
| Criar romaneio | `POST /api/shipments` (`sales_order_code`, `carrier_code`, volumes, peso) |
| Listar / detalhar | `GET /api/shipments` · `GET /api/shipments/{code}` |
| Adicionar item | `POST /api/shipments/{code}/items` |
| Conferir item | `POST /api/shipments/items/confer` (`item_id`, `conferred_qty`) |
| Conferir romaneio | `POST /api/shipments/{code}/confer` |
| Despachar | `POST /api/shipments/{code}/ship` |
| Cancelar | `POST /api/shipments/{code}/cancel` |

## 20. Plataforma: Idempotência e Escopos de permissão

### Idempotência
Métodos mutantes (`POST/PUT/PATCH`) aceitam o header **`Idempotency-Key`**. Numa
repetição com a mesma chave (mesmo método+rota+usuário) dentro da janela (TTL
24 h), a resposta original é **reproduzida** (header `Idempotent-Replayed: true`),
evitando duplicidade em retries. Memória por instância (não persiste reinício).

### Escopos de permissão
Além do `RequireRole(ADMIN/USER)`, há um middleware **`RequirePermission(scope)`**
com mapa papel→escopos: `ADMIN` (tudo), `USER` (operacional, sem `admin`),
`VIEWER` (somente leitura). Escopos: `planning:run`, `purchase:approve`,
`fiscal:authorize`, `financial:manage`, `item:activate`, `admin`. Aplicado às
rotas sensíveis novas (pipeline, fiscal manifestação/inutilização/IBPT, CNAB,
prontidão de item).

## Relação entre módulos

```
Pedido de Venda  (confirmar → demanda independente automática)
      │
      ▼
    MRP ──────── BOM (estrutura do produto)
      │    └──── Roteiro (lead time via CPM, tipo de ordem por origin)
      │    └──── Estoque (saldo disponível)
      │    └──── Parâmetros (lote mínimo, estoque de segurança)
      │
      ├── Sugestões de Compra  → Pedido de Compra
      │
      └── Sugestões de Fabricação
                  │  origin INTERNA  → Ordem de Fabricação (OF)
                  │  origin EXTERNA/TERCEIROS → Ordem de Serviço (OS)
                  │
                  ▼ (PCP analisa e aprova)
            Ordens Aprovadas
                  │
          ┌───────┴────────┐
          ▼                ▼
        CRP              APS
   (carga % por       (sequenciamento
    centro/dia)        finito / Gantt)
          │
          └── Manutenção Preventiva
              (desconta horas de parada da capacidade disponível)

  (MRP→CRP→APS num disparo: POST /api/planning/run-pipeline — §17)

Ordem de Fabricação
   start → consumo (OUT) → apontamento (backflush opcional, §18) → conclusão (IN)
                                   │
                                   ▼
                         Estoque de acabados
                                   │
                                   ▼
   NF-e de saída (autorizar → OUT + baixa de reservas + pedido Faturado + Conta a Receber)
                                   │
                                   ▼
                   Expedição / romaneio (§19): separar → conferir → despachar
```
