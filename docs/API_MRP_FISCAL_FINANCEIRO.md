# Documentação da API — Veture ERP

> **Versão:** 1.0  
> **Data:** Maio/2026  
> **Idioma:** Português (Brasil)

---

## 1. INTRODUÇÃO

O **Veture ERP** é um sistema de gestão empresarial (ERP) desenvolvido em **Go** com arquitetura **Clean Architecture** (Domain-Driven Design). Utiliza **PostgreSQL** como banco de dados relacional e o roteador HTTP **go-chi/chi v5** para exposição da API REST.

### Pilares Arquiteturais

| Camada               | Responsabilidade                                      |
|----------------------|-------------------------------------------------------|
| **Domain**           | Entidades, Value Objects, interfaces de repositório, regras de negócio puras |
| **Application**      | Casos de Uso (Use Cases), DTOs, portas de serviço     |
| **Infrastructure**   | Implementações de repositório (SQLC/PGX), serviços externos (auth) |
| **Interfaces/HTTP**  | Handlers HTTP, middlewares (JWT, logging, correlation) |

### Stack Técnica

- **Linguagem:** Go 1.22+
- **Banco de Dados:** PostgreSQL (via `pgx` e `sqlc`)
- **Roteador:** `go-chi/chi/v5`
- **Autenticação:** JWT (HMAC-SHA256) com controle de perfil (ADMIN/USER)
- **Migrações:** `golang-migrate`
- **Aritmética Financeira:** `shopspring/decimal`

---

## 2. ESTRUTURA DO PROJETO

```

├── api/                    # Entrypoint HTTP (main.go, api.go)
│   ├── api.go              # Montagem de todas as rotas (mount)
│   └── main.go             # Bootstrap da aplicação
├── internal/
│   ├── application/
│   │   ├── dto/request/    # Data Transfer Objects de entrada
│   │   ├── ports/          # Portas de serviço (AuthService)
│   │   ├── security/       # Modelo AuthUser
│   │   └── usecase/        # Casos de uso organizados por domínio
│   ├── domain/
│   │   ├── <modulo>/       # Cada módulo contém:
│   │   │   ├── entity/     # Entidades e value objects
│   │   │   ├── repository/ # Interface do repositório (porta)
│   │   │   └── service/    # Serviços de domínio
│   │   └── enums/types/    # Enumerações compartilhadas
│   ├── infrastructure/
│   │   ├── auth/           # Implementação JWT
│   │   ├── config/         # Configuração (env vars)
│   │   ├── database/       # Conexão com PostgreSQL
│   │   ├── logger/         # Logger estruturado (slog)
│   │   └── repository/     # Implementações concretas de repositórios
│   └── interfaces/
│       ├── http/
│       │   ├── context/    # Chaves de contexto HTTP
│       │   └── handler/    # Handlers HTTP por módulo
│       └── middleware/      # JWT, Correlation, RequestLogger
├── migrations/             # Migrações SQL (000001 a 000096)
├── docs/                   # Documentação
└── vendor/                 # Dependências vendored
```

---

## 3. MÓDULO MRP (Material Requirements Planning)

O módulo MRP é o coração do planejamento de materiais. Suporta 5 modos de planejamento e implementa cálculo de necessidade líquida com *time-phased netting*, explosão de estrutura (BOM), geração de LLC, exceções automáticas e integração com máquinas.

### 3.1 Rotas da API

#### MRP Calculation

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/mrp-calculation/run` | Executa o cálculo MRP para um plano |
| GET | `/api/mrp-calculation/profile/{item_code}/{plan_code}` | Perfil MRP de um item |
| POST | `/api/mrp-calculation/configured-rules` | Cria regra configurada para item |
| GET | `/api/mrp-calculation/configured-rules/{item_code}` | Lista regras configuradas |
| GET | `/api/mrp-calculation/exceptions/{plan_code}` | Lista exceções do plano |

**POST /api/mrp-calculation/run**

- **Entrada:** `{ "plan_code": int64 }`
- **Saída:** `MRPCalculationLog` com status (`COMPLETED` / `COMPLETED_WITH_ERRORS` / `ERROR`), total de itens processados, total de ordens geradas, erros por chave.

**GET /api/mrp-calculation/profile/{item_code}/{plan_code}**

- **Saída:** `MRPItemProfile` com demanda, ordens planejadas, ordens firmes, estoque projetado, LLC, data de necessidade.

#### Planned Order

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/planned-order/create` | Cria ordem planejada manual |
| GET | `/api/planned-order/list` | Lista ordens planejadas |
| GET | `/api/planned-order/{code}/firm` | Firma uma ordem planejada (converte em produção/compra) |

#### Production Plan

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/production-plan/create` | Cria plano de produção |
| GET | `/api/production-plan/list` | Lista planos |
| GET | `/api/production-plan/{code}` | Busca plano por código |
| PUT | `/api/production-plan/update` | Atualiza plano |
| DELETE | `/api/production-plan/{code}` | Remove plano |

#### Independent Demand

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/independent-demand/create` | Cria demanda independente |
| PUT | `/api/independent-demand/update/{code}` | Atualiza demanda |
| DELETE | `/api/independent-demand/delete/{code}` | Remove demanda |
| GET | `/api/independent-demand/list-from-date/{date}` | Lista demandas a partir de data |
| GET | `/api/independent-demand/list-by-item/{itemCode}` | Lista demandas por item |
| GET | `/api/independent-demand/list` | Lista todas demandas |
| GET | `/api/independent-demand/get-by-code/{code}` | Busca demanda por código |

#### Sales Forecast

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/sales-forecast/create` | Cria previsão de vendas semanal |
| GET | `/api/sales-forecast/list/{year}` | Lista previsões por ano |
| GET | `/api/sales-forecast/item/{itemCode}` | Previsão por item |
| POST | `/api/sales-forecast/blocks/create` | Cria bloco de congelamento |
| GET | `/api/sales-forecast/blocks/list` | Lista blocos |
| POST | `/api/sales-forecast/appropriation/create` | Cria tabela de apropriação |
| GET | `/api/sales-forecast/appropriation/list` | Lista tabelas de apropriação |
| POST | `/api/sales-forecast/appropriation/set-default` | Define tabela padrão |

#### Machine

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/machine/create` | Cria máquina |
| GET | `/api/machine/list` | Lista máquinas |
| GET | `/api/machine/{code}` | Busca máquina por código |
| POST | `/api/machine/types/create` | Cria tipo de máquina |
| GET | `/api/machine/types/list` | Lista tipos |
| GET | `/api/machine/types/{code}` | Busca tipo por código |
| POST | `/api/machine/time/create` | Associa item × máquina × tempo |
| GET | `/api/machine/time/list` | Lista associações |
| POST | `/api/machine/time/{code}` | Busca tempo de item por máquina |
| POST | `/api/machine/time/production/calculate` | Calcula tempo de produção |
| POST | `/api/machine/schedule/create` | Agenda máquina |
| GET | `/api/machine/schedule/list` | Lista agendamentos |
| POST | `/api/machine/schedule/{code}` | Busca agendamento |

#### Items & Structure

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/items/create` | Cria item |
| GET | `/api/items/search/{code}` | Busca item por código |
| POST | `/api/items/mask/generate` | Gera máscara de item (PDM) |
| POST | `/api/items/structure/create` | Cria componente na estrutura |
| PUT | `/api/items/structure/update` | Atualiza componente |
| GET | `/api/items/structure/resolve/{itemCode}` | Resolve árvore de estrutura completa |

#### Demais Rotas MRP

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/order-priority/create` | Cria regra de prioridade |
| GET | `/api/order-priority/list` | Lista prioridades |
| GET | `/api/order-priority/find/{value}` | Busca prioridade por valor |
| POST | `/api/restriction/create` | Cria restrição de estrutura |
| GET | `/api/restriction/list` | Lista restrições |
| GET | `/api/restriction/{code}` | Busca restrição por código |
| GET | `/api/restriction/item/{itemCode}` | Restrições por item |
| PUT | `/api/restriction/{code}` | Atualiza restrição |
| PATCH | `/api/restriction/{code}/deactivate` | Desativa restrição |
| POST | `/api/sales-division/create` | Cria divisão de vendas |
| GET | `/api/sales-division/list` | Lista divisões |
| GET | `/api/sales-division/{code}` | Busca por código |
| PUT | `/api/sales-division/{code}` | Atualiza divisão |
| DELETE | `/api/sales-division/{code}` | Remove divisão |
| GET | `/api/planning-params/list` | Lista parâmetros de planejamento |
| GET | `/api/planning-params/{number}` | Busca parâmetro por número |
| PUT | `/api/planning-params/update` | Atualiza parâmetro |
| POST | `/api/industrial-calendar/create` | Cria dia no calendário industrial |
| GET | `/api/industrial-calendar/month/{year}/{month}` | Lista mês |
| GET | `/api/industrial-calendar/workdays/{year}/{month}` | Dias úteis do mês |
| GET | `/api/delivery-promise-params/` | Busca parâmetros de promessa de entrega |
| PUT | `/api/delivery-promise-params/update` | Atualiza parâmetros |
| POST | `/api/delivery-reschedule/create` | Cria reagendamento de entrega |
| GET | `/api/delivery-reschedule/list/{sales_order_code}` | Lista reagendamentos por pedido |
| POST | `/api/item-calendar-promise/create` | Upsert dia no calendário de promessa |
| GET | `/api/item-calendar-promise/{item_code}/{mask}/{year}/{month}` | Lista mês |
| GET | `/api/item-calendar-promise/{item_code}/{mask}/{year}/{month}/workdays` | Dias úteis |
| GET | `/api/item-calendar-promise/{item_code}/{mask}/{year}/{month}/{day}` | Busca dia |
| DELETE | `/api/item-calendar-promise/{item_code}/{mask}/{year}/{month}/{day}` | Remove dia |

### 3.2 Modos de Planejamento

O plano de produção (`production_plans`) define quais modos de planejamento são executados via campo `planning_types` (array de strings).

#### 3.2.1 MRP Clássico (`MRP`)

- **Algoritmo:** BFS (Breadth-First Search) nível por nível usando LLC (Low-Level Code)
- **Fluxo:**
  1. Carrega demandas independentes + previsões de vendas
  2. Gera demanda de estoque de segurança (param 4)
  3. Carrega BOM completa em uma única CTE recursiva
  4. Calcula LLC para cada item
  5. Para cada nível (0 até maxLLC):
     - Agrega demandas por item+máscara
     - Calcula necessidade líquida (*time-phased netting*)
     - Desconta estoque disponível + suprimento firme com data ≤ data de necessidade
     - Aplica lote mínimo e lead time configurados
     - Gera sugestões de ordem planejada (FABRICAÇÃO ou COMPRA)
     - Explode BOM para níveis inferiores
  6. Pós-processamento: integração com máquinas, prioridades automáticas, exceções

- **Tipos de Demanda:**
  - `INDEPENDENT` — demanda independente cadastrada
  - `DEPENDENT` — demanda derivada da explosão de BOM
  - `FORECAST` — previsão de vendas
  - `SAFETY_STOCK` — estoque de segurança

- **Tipos de Ordem Gerada:**
  - `FABRICACAO` — item tipo FABRICADO (engineering type = 0)
  - `COMPRA` — item tipo COMPRADO (engineering type = 1)
  - Itens DE_TERCEIRO (type = 2) ou PROJETO (typeMRP != 0) não geram ordens
  - `TECHNICAL_ASSISTANCE` — quando param 17 ativo e divisão de vendas é assistência técnica

- **Fórmulas de Perda na Estrutura (param 20):**
  - **Fórmula 1:** Qtde = QtdPai × QtdComponente × (1 + %Perda/100)
  - **Fórmula 2:** Qtde = QtdPai × QtdComponente / (1 − %Perda/100)
  - **Fórmula 3:** Qtde = QtdPai × QtdComponente (ignora perda)

#### 3.2.2 Mínimo e Máximo (`MIN_MAX`)

- Para cada item com estoque de segurança (LMI > 0):
  - **LMI** = safety_stock do snapshot de estoque
  - **LMA** = maximum_stock (do item_planning_extras) ou LMI × 3
  - **QTDE** = quantity − reserved_qty (estoque disponível)
  - **QTDP** = soma de todas as ordens firmes de compra
  - Se `QTDE + QTDP ≤ LMI` → gera ordem de COMPRA com `QTC = LMA − (QTDE + QTDP)`
- Demanda tipo: `MIN_MAX`

#### 3.2.3 Ponto de Reposição (`REORDER_POINT`)

- Para cada item com `reorder_point` configurado:
  - Calcula PR (ponto de reposição) via `reorderPoint.Calculate()`
  - Se `QTDE ≤ PR` → gera ordem de COMPRA com `QTC = PR × 2` (lote econômico)
- Demanda tipo: `REORDER_POINT`

#### 3.2.4 Kanban (`KANBAN`)

- Itera cartões kanban (`kanban_cards`) agrupados por item
- Para cada cartão:
  - Se `QTDE ≤ card.ReorderPoint` → gera ordem de COMPRA
  - `QTC = quantity_per_card × card_count`
- Demanda tipo: `KANBAN`

#### 3.2.5 MPS — Master Production Schedule (`MPS`)

- Carrega entradas da tabela `mps_schedule` para o plano
- Itens não firmados são convertidos em demandas e processados via MRP clássico
- Itens firmados são tratados como suprimento firme

### 3.3 Parâmetros de Planejamento (21 parâmetros)

Os parâmetros são armazenados na tabela `planning_params` com chave numérica. O MRP os interpreta via `TypedPlanningParams`:

| # | Chave | Campo | Descrição | Default |
|---|-------|-------|-----------|---------|
| 1 | `AGRUPA_DEMANDA_ESTOQUE` | `AgrupaDemandaEstoque` | Agrupa demanda de estoque de segurança com outras demandas do mesmo item | `true` |
| 2 | `COD_FORNECEDOR_INTERFACE` | `CodFornecedorInterface` | Código do fornecedor para interface externa | vazio |
| 3 | `COD_CLIENTE_INTERFACE` | `CodClienteInterface` | Código do cliente para interface externa | vazio |
| 4 | `GERAR_DEMANDA_SEGURANCA_TODOS` | `GerarDemandaSegurancaTodos` | Se true, gera demanda de segurança para todos os itens; se false, só para itens com movimentação | `true` |
| 5 | `OBRIGATORIEDADE_REFUGO` | `ObrigatoriedadeRefugo` | Exige informe de refugo nos apontamentos | `false` |
| 6 | `DATA_NECESSIDADE_ESTOQUE_FUTURO` | `DataNecessidadeEstoqueFuturo` | Data de necessidade do estoque de segurança no futuro (today + leadtime + 1) ou passado | `true` |
| 7 | `GERAR_PRIORIDADES_ORDENS` | `GerarPrioridadesOrdens` | Habilita geração automática de prioridades nas ordens planejadas | `true` |
| 8 | `DIAS_PRIORIDADES` | `DiasPrioridades` | Janela em dias para atribuição de prioridade (ordens com start_date dentro dessa janela) | `5` |
| 10 | `ITENS_FANTASMAS_GRAVAR` | `ItensFantasmasGravar` | Se true, grava ordens para itens fantasmas; se false, apenas explode BOM | `false` |
| 11 | `DESCONSIDERA_SEMANAS_PASSADAS` | `DesconsideraSemanasPassadas` | Ignora semanas já passadas nas previsões | `true` |
| 12 | `CONSIDERA_DATAS_TANQUES` | `ConsideraDatasTanques` | Utiliza `use_tank_date` do item para data de necessidade | `false` |
| 13 | `VERIFICA_SITUACAO_PEDIDO_PROJETO` | `VerificaSituacaoPedidoProjeto` | Verifica situação de pedidos de projeto | `false` |
| 14 | `UTILIZA_CALCULO_MPS` | `UtilizaCalculoMPS` | Habilita integração com MPS | `false` |
| 15 | `PROPORCAO_ENTREGA` | `ProporcaoEntrega` | Proporção de entrega (string) | vazio |
| 16 | `VALIDA_RESTRICOES_ESTRUTURA` | `ValidaRestricoesEstrutura` | Valida restrições de estrutura durante explosão BOM | `true` |
| 17 | `TRATA_ASSISTENCIA_TECNICA` | `TrataAssistenciaTecnica` | Gera ordens de assistência técnica conforme divisão de vendas | `false` |
| 18 | `PORCENTAGEM_PROPORCAO_VALORIZACAO` | `PorcentagemProporcaoValorizacao` | Percentual para rateio de valorização | `0` |
| 19 | `DEFAULT_POSICAO` | `DefaultPosicao` | Posição padrão de estoque | vazio |
| 20 | `FORMULA_PERDAS_ESTRUTURA` | `FormulaPerdasEstrutura` | Fórmula de perdas: 1=(1+%), 2=(1/(1−%)), 3=ignora | `2` |
| 24 | `NUMERACAO_ORDENS` | `NumeracaoOrdens` | Modo de numeração de ordens: AUTO ou MANUAL | `AUTO` |
| 45 | `OBRIGAR_CONTROLE_ESTOQUE_TERCEIROS` | `ObrigarControleEstoqueTerceiros` | Exige controle de estoque para itens de terceiros | `false` |

### 3.4 Entidades MRP

#### Item (`items`)

Campos principais da pasta Planejamento:

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `TypeMRP` | enum (NORMAL_MRP, PROJETO) | Tipo de MRP; PROJETO não gera ordens automáticas |
| `LLC` | int | Low-Level Code (1–9); 1=produto final, 9=matéria-prima |
| `ReorderPoint` | value object | Ponto de reposição (lead time, consumo médio mensal, estoque segurança) |
| `Ghost` | bool | Item fantasma: não gera ordem, apenas explode BOM |
| `TankCode` | *int | Setor/tanque de produção |
| `TypeItem` | enum (FABRICADO, COMPRADO, DE_TERCEIRO) | FABRICADO gera ordem produção, COMPRADO gera compra, DE_TERCEIRO não gera ordem |
| `TypeStructItem` | enum (INDUSTRIAL, COMERCIAL) | Tipo estrutural |

#### Estrutura de Produto (`items_structure` / BOM)

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `ParentCode` | int64 | Código do item pai |
| `ChildCode` | int64 | Código do item filho (componente) |
| `ParentMask` | *string | nil = componente genérico (todas configs); não-nil = específico |
| `Quantity` | float64 | Quantidade do componente por unidade do pai |
| `LossPercentage` | float64 | Percentual de perda (0–100) |
| `Inherit` | bool | Herda configuração |
| `Sequence` | int | Ordem de exibição |
| `IsActive` | bool | Componente ativo |

**Regras de Negócio:**
- `ParentMask == nil` → componente genérico, aplica-se a TODAS as configurações
- `ParentMask != nil` → componente específico para aquela máscara
- Um item não pode ser componente de si mesmo
- A adição não pode criar ciclo na árvore

#### Plano de Produção (`production_plans`)

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `Code` | int64 | Código do plano |
| `Name` | string | Nome |
| `IndependentDemands` | enum (NO, FROM_DATE, ALL) | Filtro de demandas independentes |
| `GroupSameDateOrders` | bool | Agrupa ordens de mesma data |
| `PlanningTypes` | []string | Modos: MRP, MIN_MAX, REORDER_POINT, MPS, KANBAN |
| `Classification` | *string | Classificação de itens |
| `ClassItemCodes` | *string | Códigos de itens delimitados por vírgula |
| `OrderItemCode` | *int64 | Ordenação de itens |
| `IsActive` | bool | Plano ativo |

#### Demanda Independente (`independent_demands`)

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `CodeDemand` | int64 | Código único |
| `ItemCode` | int64 | Item |
| `Mask` | *string | Máscara configurável |
| `Quantity` | float64 | Quantidade demandada |
| `DemandDate` | time | Data da necessidade |
| `CostCenterCode` | *int64 | Centro de custo |

#### Previsão de Vendas (`sales_forecasts`)

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `ItemCode` | int64 | Item |
| `Mask` | *string | Máscara |
| `Week` | int | Semana do ano (ISO 8601) |
| `Year` | int | Ano |
| `Quantity` | float64 | Quantidade prevista |

#### Ordem Planejada (`planned_orders`)

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `Code` | int64 | Código |
| `OrderNumber` | int64 | Número da ordem |
| `ItemCode` | int64 | Item |
| `Quantity` | float64 | Quantidade |
| `OrderType` | enum (PRODUCTION, PURCHASE, OUTSOURCING, TECHNICAL_ASSISTANCE) | Tipo de ordem |
| `Status` | enum (PLANNED, RELEASED, IN_PROGRESS, FINISHED, CANCELLED) | Status |
| `DemandType` | enum (SALES_ORDER, FORECAST, INDEPENDENT, SAFETY_STOCK, REPLENISHMENT) | Origem da demanda |
| `NeedDate` | time | Data de necessidade |
| `StartDate` | *time | Data de início |
| `LLC` | int | Low-Level Code |
| `IsFirm` | bool | Ordem firmada |
| `Priority` | *string | Prioridade calculada |
| `MachineCode` | *int64 | Máquina alocada |
| `ProductionTime` | float64 | Tempo de produção |

#### Perfil MRP (`mrp_item_profiles`)

Resultado por item após cálculo:

| Campo | Descrição |
|-------|-----------|
| `Demand` | Demanda bruta |
| `OrdersPlanned` | Ordens planejadas (necessidade líquida) |
| `OrdersFirm` | Ordens firmes |
| `StockProjected` | Estoque projetado após demanda |
| `LLC` | Low-Level Code |
| `NeedDate` | Data de necessidade mais próxima |

#### Exceções MRP (`mrp_exception_messages`)

Geradas automaticamente quando ordens firmes divergem da necessidade:

| Tipo | Descrição | Ação do Planejador |
|------|-----------|--------------------|
| `EXPEDITE` | Ordem chega ≤ 5 dias após necessidade | Acelerar urgentemente |
| `RESCHEDULE_IN` | Ordem chega > 5 dias após necessidade | Antecipar ordem |
| `RESCHEDULE_OUT` | Ordem chega > 30 dias antes da necessidade | Atrasar para liberar capital |
| `CANCEL` | Ordem firme sem demanda no plano | Cancelar ordem |
| `EXCESS_PROJECTED` | Suprimento firme excede necessidade líquida | Excesso de estoque projetado |

#### Regras Configuradas por Item (`configured_item_rules`)

| Campo | Descrição |
|-------|-----------|
| `TableType` | Tipo de tabela (ex: `ITEM`) |
| `FieldName` | Nome do campo (ex: `lead_time`, `lote_minimo`) |
| `RuleType` | Tipo: `EQUAL`, `DIFFERENT`, `RANGE` |
| `RuleValue` | Valor da regra |
| `Sequence` | Ordem de aplicação |

---

## 4. MÓDULO DE PEDIDOS

### 4.1 Pedido de Venda (Sales Order)

#### Rotas

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/sales-order/create` | Cria pedido de venda |
| GET | `/api/sales-order/list` | Lista pedidos |
| GET | `/api/sales-order/{code}` | Busca por código |
| PUT | `/api/sales-order/{code}` | Atualiza pedido |
| DELETE | `/api/sales-order/{code}/cancel` | Cancela pedido |
| PATCH | `/api/sales-order/{code}/block` | Bloqueia pedido |
| PATCH | `/api/sales-order/{code}/unblock` | Desbloqueia pedido |
| PATCH | `/api/sales-order/{code}/status` | Altera status |
| GET | `/api/sales-order/customer/{customerCode}` | Lista por cliente |
| GET | `/api/sales-order/status/{status}` | Lista por status |
| POST | `/api/sales-order/items/create` | Adiciona item ao pedido |
| GET | `/api/sales-order/items/{code}` | Lista itens do pedido |
| PUT | `/api/sales-order/items/{itemCode}` | Atualiza item |
| DELETE | `/api/sales-order/items/{itemCode}/cancel` | Cancela item |

#### Status do Pedido

| Código | Status | Descrição |
|--------|--------|-----------|
| `R` | Rascunho | Pedido em elaboração |
| `P` | Pedido | Pedido confirmado |
| `A` | Pedido em Análise | Aguardando análise |
| `OA` | Orçamento em Análise | Orçamento sob revisão |
| `OF` | Orçamento | Orçamento aprovado |

#### Origem do Pedido

| Origem | Descrição |
|--------|-----------|
| `NORMAL` | Pedido normal de venda |
| `DEPENDENT` | Pedido dependente (matriz/filial) |
| `ASSISTANCE` | Assistência técnica |
| `RESERVE` | Reserva |
| `COPY` | Cópia de outro pedido |

#### Status do Item

| Status | Descrição |
|--------|-----------|
| `OPEN` | Em aberto |
| `PARTIAL` | Parcialmente atendido |
| `DELIVERED` | Entregue |
| `CANCELLED` | Cancelado |

#### Campos do Pedido

**Cabeçalho:** código, número, empresa, status, origem, data emissão, data entrega, data entrega firme, cliente, endereços (cobrança/entrega), representante, plano MRP, divisão de vendas, percentual comissão, tipo tributário, indicador de presença, canal de venda, tipo NF padrão, tabela de preço, moeda, condição pagamento, dias adicionais, portador, totais (peso bruto/líquido, valor bruto/líquido, valor sem ST, valor com IPI e ST), observações, bloqueado, firme.

**Item:** sequência, item, máscara, tipo NF, unidade venda, depósito, tabela preço, qtd solicitada/atendida/cancelada/saldo, preço unitário, totais (bruto, líquido, com IPI, IPI, ST), percentuais (IPI, ICMS, PIS, COFINS, ST, desconto), peso, data entrega, status.

#### Funcionalidades Implementadas

- CRUD completo de pedidos com controle de sequência por empresa (`sales_order_sequences`)
- Bloqueio/desbloqueio com motivo
- Troca de status
- Filtro por cliente e por status
- Itens com controle de quantidade (solicitada → atendida → cancelada), saldo calculado
- Cancelamento individual de itens

#### Pendente (não implementado)

- Pastas: Transporte, Descontos/Acréscimos, Forma de Pagamento no pedido
- Cálculo automático de impostos nos itens (campos de alíquota existem mas não são preenchidos automaticamente)
- Integração com cadastro de clientes (campo `customer_code` existe mas não há CRUD de cliente)
- Workflow de aprovação (análise comercial/financeira)

### 4.2 Pedido de Compra (Purchase Order)

#### Rotas

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/purchase-order/create` | Cria pedido de compra |
| GET | `/api/purchase-order/list` | Lista pedidos |
| GET | `/api/purchase-order/{code}` | Busca por código |
| PUT | `/api/purchase-order/{code}` | Atualiza pedido |
| DELETE | `/api/purchase-order/{code}/cancel` | Cancela pedido |
| GET | `/api/purchase-order/supplier/{supplierCode}` | Lista por fornecedor |
| GET | `/api/purchase-order/status/{status}` | Lista por status |

#### Status

| Status | Descrição |
|--------|-----------|
| `DRAFT` | Rascunho |
| `REQUESTED` | Solicitado |
| `APPROVED` | Aprovado |
| `PARTIAL` | Parcialmente recebido |
| `RECEIVED` | Totalmente recebido |
| `CANCELLED` | Cancelado |

#### Origem

| Origem | Descrição |
|--------|-----------|
| `NORMAL` | Pedido normal |
| `MRP` | Gerado pelo MRP |
| `MANUAL` | Entrada manual |
| `INTERFABRICA` | Entre fábricas |

#### Campos

**Cabeçalho:** código, número, empresa, status, origem, data emissão, data entrega, fornecedor, condição pagamento, moeda, endereço entrega, totais (bruto, líquido, desconto), observações, firme.

**Item:** sequência, item, máscara, qtd solicitada/recebida/cancelada, preço unitário, total, desconto, IPI, ICMS, status, data entrega.

#### Controle de Sequência

Similar ao pedido de venda, utiliza tabela `purchase_order_sequences` por empresa.

---

## 5. MÓDULO DE PRODUÇÃO

### 5.1 Ordem de Produção (Production Order)

#### Rotas

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/production-order/create` | Cria ordem de produção |
| GET | `/api/production-order/list` | Lista ordens |
| GET | `/api/production-order/{id}` | Busca por ID |
| POST | `/api/production-order/{id}/start` | Inicia produção (OPEN → IN_PROGRESS) |
| POST | `/api/production-order/appointment` | Registra apontamento (tempo + quantidade) |
| POST | `/api/production-order/consumption` | Registra consumo de matéria-prima |
| POST | `/api/production-order/{id}/complete` | Conclui produção (IN_PROGRESS → COMPLETED) |
| POST | `/api/production-order/{id}/close` | Fecha ordem (COMPLETED → CLOSED) |
| POST | `/api/production-order/{id}/cancel` | Cancela ordem (→ CANCELLED) |
| GET | `/api/production-order/{id}/appointments` | Lista apontamentos |
| GET | `/api/production-order/{id}/consumptions` | Lista consumos |

#### Ciclo de Vida

```
OPEN → IN_PROGRESS → COMPLETED → CLOSED
  ↓         ↓            ↓
  └─────────┴────────────┴──→ CANCELLED
```

| Transição | Endpoint | De | Para |
|-----------|----------|----|------|
| Criar | `/create` | — | `OPEN` |
| Iniciar | `/{id}/start` | `OPEN` | `IN_PROGRESS` |
| Concluir | `/{id}/complete` | `IN_PROGRESS` | `COMPLETED` |
| Fechar | `/{id}/close` | `COMPLETED` | `CLOSED` |
| Cancelar | `/{id}/cancel` | qualquer | `CANCELLED` |

#### Campos da Ordem

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `OrderNumber` | int64 | Número da ordem |
| `PlannedOrderID` | *int64 | Origem: ordem planejada do MRP |
| `ItemCode` | int64 | Item a ser produzido |
| `Mask` | string | Máscara configurável |
| `PlannedQty` | float64 | Quantidade planejada |
| `ProducedQty` | float64 | Quantidade produzida |
| `ScrappedQty` | float64 | Quantidade refugada |
| `MachineID` | *int64 | Máquina alocada |
| `CostCenterID` | *int64 | Centro de custo |
| `EmployeeID` | *int64 | Funcionário responsável |
| `Priority` | *string | Prioridade |

#### Apontamento de Produção (`production_appointments`)

| Campo | Descrição |
|-------|-----------|
| `ProductionOrderID` | Ordem vinculada |
| `MachineID` | Máquina utilizada |
| `EmployeeID` | Operador |
| `AppointmentDate` | Data do apontamento |
| `StartTime / EndTime` | Horário |
| `ProducedQty` | Quantidade produzida no período |
| `ScrappedQty / ScrapReason` | Refugo e motivo |

#### Consumo de Material (`production_consumptions`)

| Campo | Descrição |
|-------|-----------|
| `ProductionOrderID` | Ordem vinculada |
| `AppointmentID` | Apontamento vinculado (opcional) |
| `ItemCode` | Item consumido (matéria-prima) |
| `ConsumedQty` | Quantidade consumida |
| `WarehouseID` | Depósito de saída |
| `Lot` | Lote |

### 5.2 Movimentos de Estoque (Stock)

#### Rotas

**Movimentos:**

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/stock/movements/create` | Cria movimento de estoque |
| GET | `/api/stock/movements/list` | Lista movimentos |
| GET | `/api/stock/movements/item/{itemCode}` | Movimentos por item |
| GET | `/api/stock/movements/warehouse/{warehouseId}` | Movimentos por depósito |

**Saldos:**

| Método | Path | Descrição |
|--------|------|-----------|
| GET | `/api/stock/balances/get` | Busca saldo (query params) |
| GET | `/api/stock/balances/list` | Lista todos saldos |
| GET | `/api/stock/balances/warehouse/{warehouseId}` | Saldos por depósito |
| GET | `/api/stock/balances/item/{itemCode}` | Saldos por item |

**Reservas:**

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/stock/reservations/create` | Cria reserva de estoque |
| PATCH | `/api/stock/reservations/{id}/release` | Libera reserva |
| PATCH | `/api/stock/reservations/{id}/consume` | Consome reserva |

**Inventário Físico:**

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/stock/inventories/create` | Cria inventário |
| GET | `/api/stock/inventories/list` | Lista inventários |
| GET | `/api/stock/inventories/{id}` | Busca inventário |
| POST | `/api/stock/inventories/{id}/close` | Fecha inventário |
| POST | `/api/stock/inventories/count` | Registra contagem |
| POST | `/api/stock/inventories/adjust` | Ajusta divergência |
| GET | `/api/stock/inventories/{id}/items` | Lista itens do inventário |

#### Tipos de Movimento

| Tipo | Descrição |
|------|-----------|
| `IN` | Entrada |
| `OUT` | Saída |
| `TRANSFER_IN` | Transferência (entrada) |
| `TRANSFER_OUT` | Transferência (saída) |
| `ADJUSTMENT` | Ajuste |
| `RESERVATION` | Reserva |
| `UNRESERVATION` | Liberação de reserva |

#### Referências de Movimento

| Tipo | Descrição |
|------|-----------|
| `PURCHASE_ORDER` | Pedido de compra |
| `PRODUCTION_ORDER` | Ordem de produção |
| `SALES_ORDER` | Pedido de venda |
| `INVENTORY` | Inventário |
| `MANUAL` | Manual |

#### Status de Reserva

| Status | Descrição |
|--------|-----------|
| `ACTIVE` | Ativa |
| `CONSUMED` | Consumida |
| `CANCELLED` | Cancelada |

#### Status de Inventário

| Status | Descrição |
|--------|-----------|
| `OPEN` | Aberto |
| `IN_PROGRESS` | Em andamento |
| `COUNTED` | Contado |
| `ADJUSTED` | Ajustado |
| `CLOSED` | Fechado |

#### Saldo de Estoque (`stock_balances`)

| Campo | Descrição |
|-------|-----------|
| `Quantity` | Quantidade em estoque |
| `ReservedQty` | Quantidade reservada |
| `AvailableQty` | Quantidade disponível (`quantity - reserved_qty`, coluna gerada) |
| `MinimumStock` | Estoque mínimo |
| `MaximumStock` | Estoque máximo |
| `SafetyStock` | Estoque de segurança |
| `AvgCost` | Custo médio |
| `LastCost` | Último custo |
| `TotalCost` | Custo total |

---

## 6. MÓDULO FISCAL

### 6.1 Motor de Cálculo de Impostos (`tax_engine.go`)

O motor implementa três cenários tributários com base na UF de origem/destino e tipo de destinatário:

#### Cenário 1: Interestadual

- **ICMS:** alíquota interestadual conforme tabela `icms_interstate` (origem_uf × destino_uf)
  - Mercadoria origem 1 ou 2 (importada): alíquota fixa 4%
  - Demais: 12% (Sul/Sudeste) ou 7% (Norte/Nordeste/Centro-Oeste)
- **DIFAL:** calculado para não contribuintes e pessoas físicas: `(alíquota interna destino − alíquota interestadual) × base`
- **Base ICMS:** (valor unitário × quantidade) + IPI + frete rateado − desconto rateado
- **CST ICMS:** `00`
- **Alíquotas Padrão:** PIS 1,65%, COFINS 7,6%

#### Cenário 2: Interna PR — Contribuinte

- **ICMS:** 19,5% (padrão PR)
- **Diferimento:** 38,46% do ICMS (diferido)
- **CST ICMS:** `51` (diferimento)
- **Base ICMS:** valor total + IPI + frete − desconto

#### Cenário 3: Interna PR — Não Contribuinte

- **ICMS:** 19,5%
- **Sem diferimento**
- **CST ICMS:** `00`

#### Fórmulas por Imposto

**IPI:**
```
Base IPI = ValorUnitario × Quantidade
Valor IPI = Base IPI × AliquotaIPI (da tabela NCM)
```

**PIS/COFINS:**
```
Base PIS/COFINS = (ValorUnitario × Quantidade) + FreteRateado − DescontoRateado
Valor PIS = Base × 1,65% (ou alíquota NCM)
Valor COFINS = Base × 7,6% (ou alíquota NCM)
```

**Rateio de Frete e Desconto:**
```
Proporção = (ValorUnitario × Quantidade) / TotalGeralItens
FreteRateado = FreteTotal × Proporção
DescontoRateado = DescontoTotal × Proporção
```

### 6.2 Documentos Fiscais de Entrada

#### Rotas

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/fiscal/entries/create` | Cria entrada fiscal manual |
| POST | `/api/fiscal/entries/upload-nfe` | Upload de XML NF-e de entrada |
| POST | `/api/fiscal/entries/{code}/approve` | Aprova entrada (PENDING → APPROVED) |
| GET | `/api/fiscal/entries/list` | Lista entradas |
| GET | `/api/fiscal/entries/{code}` | Busca entrada por código |

#### Status da Entrada

| Status | Descrição |
|--------|-----------|
| `PENDING` | Pendente de conferência |
| `CONFERRED` | Conferida |
| `APPROVED` | Aprovada (gera créditos) |
| `WRITTEN_OFF` | Baixada |
| `CANCELLED` | Cancelada |

#### Campos da Entrada Fiscal

**Cabeçalho:** chave acesso (44 dígitos), número NF, série, modelo (55=NF-e, 57=CT-e), data emissão, data entrada, CNPJ/razão social/IE/UF emitente, valores (produtos, frete, seguro, desconto, IPI, ICMS, PIS, COFINS, total), tipo documento (NFE, CTE), pedido de compra vinculado, CT-e vinculado, XML path.

**Item:** sequência, item, NCM, CFOP, quantidade, preço unitário, bases e valores (ICMS, IPI, PIS, COFINS), CSTs, flags de crédito (ICMS, IPI, PIS, COFINS).

#### Regras de Aprovação

- Ao aprovar, os créditos fiscais são contabilizados para apuração
- Itens com `gera_credito_icms/ipi/pis/cofins = true` geram crédito correspondente

### 6.3 Documentos Fiscais de Saída

#### Rotas

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/fiscal/exits/create` | Cria NF-e de saída |
| POST | `/api/fiscal/exits/{code}/authorize` | Autoriza NF-e (DRAFT → AUTHORIZED) |
| POST | `/api/fiscal/exits/{code}/cancel` | Cancela NF-e (→ CANCELLED) |
| GET | `/api/fiscal/exits/list` | Lista saídas |
| GET | `/api/fiscal/exits/{code}` | Busca saída por código |

#### Status da Saída

| Status | Descrição |
|--------|-----------|
| `DRAFT` | Rascunho |
| `AUTHORIZED` | Autorizada (NF-e emitida) |
| `CANCELLED` | Cancelada |
| `REJECTED` | Rejeitada pela SEFAZ |

#### Campos da Saída Fiscal

**Cabeçalho:** chave acesso, número NF, série, data emissão, data saída, CNPJ/razão social/IE/UF destinatário, CFOP, natureza operação, valores (produtos, frete, seguro, desconto, IPI, ICMS, PIS, COFINS, total), pedido de venda vinculado, protocolo autorização, XML/DANFE path, referência Focus NFe.

**Item:** similar à entrada, acrescido de `valor_icms_diferido` e `origem_mercadoria` (0=nacional, 1=estrangeira importação direta, 2=estrangeira mercado interno, etc.).

#### Integração com Focus NFe

- Token e ambiente configurados em `fiscal_configs`
- `focus_ref` armazena referência da nota na API Focus
- Autorização e cancelamento delegados à API Focus NFe

### 6.4 Configurações Fiscais

#### Rotas

| Método | Path | Descrição |
|--------|------|-----------|
| GET | `/api/fiscal/config` | Busca configuração fiscal |
| PUT | `/api/fiscal/config` | Atualiza configuração |

#### Campos (`fiscal_configs`)

| Campo | Descrição | Default |
|-------|-----------|---------|
| `cnpj_empresa` | CNPJ da empresa | — |
| `razao_social` | Razão social | — |
| `ie_empresa` | Inscrição Estadual | — |
| `regime_tributario` | `lucro_real`, `lucro_presumido`, `simples_nacional` | `lucro_real` |
| `uf_empresa` | UF da empresa | `PR` |
| `icms_interno_aliquota` | Alíquota ICMS interna | 19,5% |
| `icms_diferimento_percentual` | Percentual de diferimento | 38,46% |
| `focus_nfe_token` | Token API Focus NFe | — |
| `focus_nfe_ambiente` | `homologacao` ou `producao` | `homologacao` |
| `juros_mes` | Juros ao mês para atraso | 1% |
| `multa_atraso` | Multa por atraso | 2% |
| `vencimento_icms_dia` | Dia vencimento ICMS | 20 |
| `vencimento_ipi_dia` | Dia vencimento IPI | 25 |
| `vencimento_pis_cofins_dia` | Dia vencimento PIS/COFINS | 25 |

### 6.5 Tabelas Fiscais

#### NCM (`ncm_tax_table`)

| Campo | Descrição |
|-------|-----------|
| `ncm` | Código NCM (8 dígitos) |
| `aliq_ipi` | Alíquota de IPI |
| `aliq_pis` | Alíquota de PIS (default 1,65%) |
| `aliq_cofins` | Alíquota de COFINS (default 7,6%) |
| `cst_pis` / `cst_cofins` / `cst_ipi` | Códigos de Situação Tributária |

**Seed incluso:** ~89 NCMs com alíquotas IPI variando de 0% a 15% (ex: 8528.52.00 com 15% IPI).

#### Cenários Tributários (`tax_scenarios`)

| Cenário | UF Destino | Tipo | ICMS | Diferimento | CST | DIFAL |
|---------|-----------|------|------|-------------|-----|-------|
| INTERESTADUAL_SUL_SUDESTE | — | contribuinte | 12% | — | 00 | não |
| INTERESTADUAL_NORTE_NORDESTE | — | contribuinte | 7% | — | 00 | não |
| INTERESTADUAL_IMPORTADA | — | contribuinte | 4% | — | 00 | não |
| INTERNA_PR_CONTRIBUINTE | PR | contribuinte | 19,5% | 38,46% | 51 | não |
| INTERNA_PR_NAO_CONTRIBUINTE | PR | nao_contribuinte | 19,5% | 0% | 00 | não |

#### ICMS Interestadual (`icms_interstate`)

Seed com 26 rotas partindo de PR (12% Sul/Sudeste, 7% demais).

#### ICMS Interno por UF (`icms_internal`)

Seed com 12 UFs (PR 19,5%, SP/RJ/MG/DF 18%, RS/SC/ES/GO/MT/MS 17%).

---

## 7. MÓDULO FINANCEIRO

### 7.1 Contas a Pagar

#### Rotas

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/financial/contas-pagar/create` | Cria conta a pagar |
| GET | `/api/financial/contas-pagar/list` | Lista contas a pagar |
| GET | `/api/financial/contas-pagar/{id}` | Busca por ID |
| POST | `/api/financial/contas-pagar/{id}/approve` | Aprova conta |
| POST | `/api/financial/contas-pagar/{id}/baixar` | Baixa conta (pagamento) |
| POST | `/api/financial/contas-pagar/{id}/cancel` | Cancela conta |
| GET | `/api/financial/contas-pagar/aging` | Relatório de aging |

#### Status

| Status | Descrição |
|--------|-----------|
| `PENDENTE` | Aguardando pagamento |
| `APROVADO` | Aprovada para pagamento |
| `PAGO` | Paga/baixada |
| `VENCIDO` | Vencida |
| `CANCELADO` | Cancelada |

#### Status de Aprovação

| Status | Descrição |
|--------|-----------|
| `PENDENTE` | Aguardando aprovação |
| `APROVADO` | Aprovada |
| `REJEITADO` | Rejeitada (com motivo) |

#### Campos

| Campo | Descrição |
|-------|-----------|
| `NumeroDocumento` | Número do documento |
| `TipoDocumento` | Tipo (OUTROS, TAX, etc.) |
| `FornecedorID` | Fornecedor vinculado |
| `FiscalEntryID` | Entrada fiscal vinculada |
| `PurchaseOrderID` | Pedido de compra vinculado |
| `DataLancamento` / `DataEmissao` / `DataVencimento` / `DataPagamento` | Datas |
| `ValorBruto` | Valor original (decimal, > 0) |
| `Desconto` / `Juros` / `Multa` / `ValorPago` | Valores |
| `ParcelaNumero` / `ParcelaTotal` / `ParcelaPaiID` | Controle de parcelas |
| `ContaBancariaID` / `FormaPagamento` | Meio de pagamento |
| `PlanoContasID` / `CentroCustoID` | Classificação contábil |
| `AdiantamentoID` / `ValorAdiantamentoAbatido` | Controle de adiantamentos |

#### Regras de Negócio — Baixa

1. Conta deve estar `PENDENTE` ou `APROVADO`
2. Se data de pagamento > data de vencimento:
   - **Juros:** `(valor original − já pago) × (juros_mês) × (dias_atraso / 30)`
   - **Multa:** `(valor original − já pago) × multa_atraso`
   - Taxas vêm do `fiscal_configs` (default: 1% a.m., 2% multa)
3. **Pagamento parcial:**
   - Baixa o valor parcial na conta original
   - Cria nova conta a pagar com o saldo remanescente
   - Nova conta tem número `original/P` e status `APROVADO`
4. Cria registro no **Fluxo de Caixa** (tipo `SAIDA`)
5. Atualiza saldo da conta bancária

### 7.2 Contas a Receber

#### Rotas

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/financial/contas-receber/create` | Cria conta a receber |
| GET | `/api/financial/contas-receber/list` | Lista contas a receber |
| GET | `/api/financial/contas-receber/{id}` | Busca por ID |
| POST | `/api/financial/contas-receber/{id}/baixar` | Baixa conta (recebimento) |
| POST | `/api/financial/contas-receber/{id}/cancel` | Cancela conta |
| GET | `/api/financial/contas-receber/aging` | Relatório de aging |

#### Status

| Status | Descrição |
|--------|-----------|
| `PENDENTE` | Aguardando recebimento |
| `APROVADO` | Aprovada |
| `RECEBIDO` | Recebida/baixada |
| `VENCIDO` | Vencida |
| `CANCELADO` | Cancelada |

#### Campos

Similar a Contas a Pagar, acrescido de:

| Campo | Descrição |
|-------|-----------|
| `ClienteID` | Cliente vinculado |
| `FiscalExitID` | Saída fiscal vinculada |
| `SalesOrderID` | Pedido de venda vinculado |
| `NossoNumero` | Nosso número (boleto) |
| `LinhaDigitavel` | Linha digitável (boleto) |
| `CodigoBarras` | Código de barras |
| `ChavePixGerada` | Chave PIX gerada |
| `EmProtesto` | Flag de protesto |

### 7.3 Fluxo de Caixa

#### Rotas

| Método | Path | Descrição |
|--------|------|-----------|
| GET | `/api/financial/fluxo-caixa` | Fluxo de caixa realizado |
| GET | `/api/financial/fluxo-projetado` | Fluxo de caixa projetado |
| GET | `/api/financial/saldo-contas` | Saldo atual das contas bancárias |

#### Tipos de Movimento

| Tipo | Descrição |
|------|-----------|
| `ENTRADA` | Recebimento |
| `SAIDA` | Pagamento |
| `TRANSFERENCIA` | Transferência entre contas |

#### Campos do Fluxo

| Campo | Descrição |
|-------|-----------|
| `Data` | Data do movimento |
| `Tipo` | ENTRADA, SAIDA, TRANSFERENCIA |
| `Valor` | Valor (decimal) |
| `ContaBancariaID` | Conta origem |
| `ContaBancariaDestinoID` | Conta destino (transferência) |
| `ContasPagarID` / `ContasReceberID` | Vinculação |
| `Conciliado` | Flag de conciliação |
| `ExtratoHash` | Hash do registro OFX |

### 7.4 Apuração de Impostos

#### Rotas

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/financial/apuracao-impostos` | Executa apuração |
| GET | `/api/financial/apuracao-impostos/{competencia}` | Busca apuração |

#### Regras de Apuração

1. Para cada imposto (ICMS, IPI, PIS, COFINS):
   - Soma débitos das saídas fiscais no período
   - Soma créditos das entradas fiscais no período
   - `Saldo = Débitos − Créditos`
2. Se **saldo > 0** (a pagar):
   - Cria **Conta a Pagar** automática com vencimento conforme config fiscal:
     - ICMS: `vencimento_icms_dia` (default dia 20)
     - IPI: `vencimento_ipi_dia` (default dia 25)
     - PIS/COFINS: `vencimento_pis_cofins_dia` (default dia 25)
   - Vencimento no mês seguinte à competência
3. Se **saldo < 0** (a compensar):
   - Registra saldo credor para compensação futura
4. Status: `APURAR` → `APURADO` → `PAGO`

#### Campos da Apuração

| Campo | Descrição |
|-------|-----------|
| `Imposto` | ICMS, IPI, PIS, COFINS |
| `Competencia` | Formato `MM/YYYY` |
| `Debitos` | Total de débitos |
| `Creditos` | Total de créditos |
| `SaldoDevedor` | Valor a pagar |
| `SaldoCredor` | Valor a compensar |
| `CpID` | Conta a Pagar vinculada |

### 7.5 Cadastros Financeiros

#### Contas Bancárias

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/financial/contas-bancarias/create` | Cria conta bancária |
| GET | `/api/financial/contas-bancarias/list` | Lista contas |

**Campos:** Banco, Agência, Conta, Dígito, Descrição, Titular, Saldo Inicial, Chave PIX, Tipo Chave PIX.

#### Condições de Pagamento

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/financial/condicoes-pagamento/create` | Cria condição |
| GET | `/api/financial/condicoes-pagamento/list` | Lista condições |

**Campos:** Nome, Parcelas (JSONB com definição de cada parcela), Ativo.

#### Plano de Contas

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/financial/plano-contas/create` | Cria conta contábil |
| GET | `/api/financial/plano-contas/list` | Lista plano de contas |

**Campos:** Código (hierárquico), Descrição, Tipo, Natureza, ParentCode, Nível.

#### Centros de Custo

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/financial/centros-custo/create` | Cria centro de custo |
| GET | `/api/financial/centros-custo/list` | Lista centros de custo |

**Campos:** Código, Descrição, Tipo (ADMINISTRATIVO, PRODUCAO, COMERCIAL, etc.).

#### Formas de Pagamento

Tabela `formas_pagamento` existe no banco (código, descrição, ativo) mas **não possui rotas HTTP expostas**.

---

## 8. FUNCIONALIDADES PENDENTES

Funcionalidades solicitadas na especificação original mas **não implementadas**:

### 8.1 Cadastros Básicos

| Funcionalidade | Status |
|---------------|--------|
| CRUD de Cliente/Fornecedor | ❌ Não implementado — campos `customer_code` e `supplier_code` existem mas apontam para tabelas inexistentes |
| Cadastro de Séries NF | ❌ Não implementado |
| Cadastro de Veículos | ❌ Não implementado |
| Cadastro de Transportadoras | ❌ Não implementado |

### 8.2 Pedido de Venda — Pastas

| Pasta | Status |
|-------|--------|
| Transporte (transportadora, veículo, frete) | ❌ Campos não existem na entidade |
| Descontos/Acréscimos | ❌ Percentual de desconto existe no item, mas não há tela/pasta dedicada |
| Forma de Pagamento | ❌ Campo `payment_term_code` existe, mas sem CRUD de formas de pagamento associadas |

### 8.3 Produção e Logística

| Funcionalidade | Status |
|---------------|--------|
| Frente de Caixa (PDV) | ❌ Não implementado |
| Controle de Carregamento | ❌ Não implementado |
| Geração de Pedidos de Assistência Técnica | ❌ Não implementado (apenas flag e tipo de origem) |
| Desmembra Pedidos | ❌ Não implementado |

### 8.4 Fiscal

| Funcionalidade | Status |
|---------------|--------|
| Importação IBPT/SCI (tabelas de alíquotas) | ❌ Não implementado — seeds manuais de NCM no migration 000095 |
| Manifestação do Destinatário | ❌ Não implementado |
| Carta de Correção Eletrônica (CC-e) | ❌ Não implementado |
| Inutilização de Numeração | ❌ Não implementado |

### 8.5 Financeiro

| Funcionalidade | Status |
|---------------|--------|
| Cobrança Escritural / Boletos (CNAB) | ❌ Apenas campos `nosso_numero`, `linha_digitavel`, `codigo_barras` existem na entidade |
| Conciliação Bancária (OFX) | ❌ Campo `extrato_hash` e flag `conciliado` existem, mas sem importação |
| Conciliação de Cartões | ❌ Não implementado |
| Geração de Borderô | ❌ Não implementado |
| Controle de Adiantamentos | ❌ Campos existem na entidade mas sem lógica de negócio dedicada |

### 8.6 Relatórios

| Código | Nome | Status |
|--------|------|--------|
| R01 | Relatório de Vendas | ❌ |
| R02 | Relatório de Compras | ❌ |
| R03 | Relatório de Produção | ❌ |
| R04 | Relatório de Estoque | ❌ |
| R05 | Relatório Fiscal (Entradas) | ❌ |
| R06 | Relatório Fiscal (Saídas) | ❌ |
| R07 | Relatório Financeiro (Contas a Pagar) | ❌ |
| R08 | Relatório Financeiro (Contas a Receber) | ❌ |
| R09 | Fluxo de Caixa Realizado | ❌ |
| R10 | DRE Gerencial | ❌ |
| R11 | Balancete | ❌ |
| R12 | Apuração de Impostos | ❌ |
| R13 | Curva ABC de Itens | ❌ |
| R14 | MRP — Necessidades Líquidas | ❌ |
| R15 | MRP — Ordens Planejadas | ❌ |
| R16 | MRP — Exceções | ❌ |
| R17 | Aging Contas a Pagar | ✅ (endpoint GET) |
| R18 | Aging Contas a Receber | ✅ (endpoint GET) |
| R19 | Inventário Físico | ❌ |

### 8.7 Próximos Passos Sugeridos (Ordem de Prioridade)

1. **CRUD de Cliente/Fornecedor** — base para todos os módulos
2. **Integração completa Focus NFe** — validação de schemas XML, retry em rejeições
3. **Geração de Boletos (CNAB 240/400)** — com registro em banco
4. **Importação OFX** — conciliação bancária automática
5. **Relatórios** (R01, R04, R10, R12, R14) — começar pelos mais essenciais
6. **Pastas do Pedido de Venda** — Transporte, Descontos, Forma Pagamento
7. **Importação IBPT** — carga completa de NCMs e alíquotas
8. **Frente de Caixa** — integração PDV
9. **Controle de Carregamento** — logística de expedição
10. **Desmembra Pedidos** — split de pedidos de venda

---

## 9. MIGRATIONS

Lista completa das migrações do banco de dados (000001 a 000096):

| Migration | Nome | Descrição |
|-----------|------|-----------|
| 000001 | init | Tabelas base: `users` (UUID PK), `items`, `questions`, `item_masks`, enum `component_type` |
| 000002 | init | `item_question_answers`, função `generate_item_mask()` |
| 000003 | init | `question_options` |
| 000004 | init | Índice `idx_item_name_code` em items |
| 000005 | init | (vazia) |
| 000006 | init | Atualização função `generate_item_mask()` com position |
| 000007 | init | Tabela `item_questions` (associação item-pergunta com posição) |
| 000008 | init | (comentada — migração revertida) |
| 000009 | init | Componentes e estruturas base |
| 000010 | init | Extensões de estrutura |
| 000011–000015 | init | Ajustes incrementais de schema |
| 000016–000020 | init | Tabelas de suporte (almoxarifado, grupos, etc.) |
| 000021–000030 | init | Refinamentos de itens, estruturas e PDM |
| 000031–000040 | init | Extensões de planejamento, parâmetros |
| 000041–000050 | init | Calendário industrial, promessas de entrega, máquinas |
| 000051–000058 | init | Restrições, prioridades, centros de custo |
| 000059–000065 | init | Divisões de vendas, previsões, alocações |
| 000066–000072 | init | Demandas independentes, planos de produção, funcionários |
| 000073–000080 | init | Ordens planejadas, regras configuradas, snapshots de estoque |
| 000081–000088 | init | Perfis MRP, logs de cálculo, exceções, agendamento de máquinas |
| 000089 | init | **Pedidos de Venda:** `sales_orders`, `sales_order_items`, `sales_order_sequences` |
| 000090 | add_mrp_fields | Modos MIN_MAX, Kanban, MPS: `kanban_cards`, `mps_schedule`, colunas `maximum_stock`, `safety_time_days`, `coverage_days` |
| 000091 | item_planning_fields | `item_planning_extras` (safety_time, coverage, grouping_key, is_critical, maximum_stock, use_tank_date) |
| 000092 | purchase_order | **Pedidos de Compra:** `purchase_orders`, `purchase_order_items`, `purchase_order_sequences` |
| 000093 | stock_management | **Estoque:** `stock_movements`, `stock_reservations`, `stock_balances`, `physical_inventories`, `physical_inventory_items` |
| 000094 | production_order | **Ordens de Produção:** `production_orders`, `production_appointments`, `production_consumptions` |
| 000095 | fiscal_foundation | **Fiscal:** `ncm_tax_table`, `tax_scenarios`, `icms_interstate`, `icms_internal`, `fiscal_entries`, `fiscal_entry_items`, `fiscal_exits`, `fiscal_exit_items`, `fiscal_configs` — com seeds de NCMs, cenários e alíquotas |
| 000096 | financial | **Financeiro:** `contas_bancarias`, `condicoes_pagamento`, `formas_pagamento`, `plano_contas`, `centros_custo`, `contas_pagar`, `contas_receber`, `fluxo_caixa`, `tax_assessments` |

---

## 10. SEGURANÇA

### 10.1 Autenticação JWT

- **Login:** `POST /users/login` — retorna token JWT
- **Registro:** `POST /users/register` — cria usuário
- **Algoritmo:** HMAC-SHA256 com secret configurável
- **Expiração:** definida pelo servidor
- **Claims:** `UserID` (UUID), `Role` (string)

### 10.2 Middleware de Autorização

Todos os endpoints sob `/api/*` exigem:

1. **JWT Middleware:**
   - Valida header `Authorization: Bearer <token>`
   - Extrai claims e popula `AuthUser` no contexto
   - Loga tentativas inválidas com IP

2. **Role Middleware:**
   - `RequireRole("ADMIN", "USER")` em todas as rotas
   - Ambos perfis têm acesso total atualmente
   - Middleware preparado para granularidade futura

3. **AuthService (porta):**
   - `CanCreate()` — verifica permissão de criação
   - `CanBaixarContaPagar()` / `CanBaixarContaReceber()` — permissões financeiras
   - `CanApurarImpostos()` — permissão fiscal
   - `UserID(ctx)` — extrai UUID do usuário autenticado

### 10.3 Middlewares de Infraestrutura

| Middleware | Descrição |
|-----------|-----------|
| `CorrelationMiddleware` | Gera UUID por request e injeta nos logs |
| `RealIP` (chi) | Extrai IP real do request |
| `Recoverer` (chi) | Recupera de panics |
| `Timeout(60s)` (chi) | Timeout global de 60 segundos |
| `StripSlashes` (chi) | Normaliza trailing slash |
| `RequestLoggerMiddleware` | Loga método, path, status, duração |

### 10.4 Health Check

`GET /health` — endpoint público (sem autenticação), retorna `{"status": "ok", "timestamp": "...", "mask": "core-api"}`.

---

