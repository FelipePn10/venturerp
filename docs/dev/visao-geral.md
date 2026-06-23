# Visão Geral da API — VentureERP

> **Versão:** 1.1  
> **Data:** Junho/2026  
> **Idioma:** Português (Brasil)

---

## 1. INTRODUÇÃO

O **VentureERP** é um sistema de gestão empresarial (ERP) desenvolvido em **Go** com arquitetura **Clean Architecture** (Domain-Driven Design). Utiliza **PostgreSQL** como banco de dados relacional e o roteador HTTP **go-chi/chi v5** para exposição da API REST.

> Este documento é a **visão geral** do sistema. Áreas aprofundadas têm documentos
> dedicados (ver [`README.md`](../README.md) da pasta `docs/`): **Fiscal & Financeiro** em
> [`fiscal-financeiro.md`](fiscal-financeiro.md), **Manufatura/Compras** em
> [`manufatura-e-compras.md`](manufatura-e-compras.md), e os cadastros de **Cliente/Fornecedor**
> nos respectivos docs.

### Pilares Arquiteturais

| Camada               | Responsabilidade                                      |
|----------------------|-------------------------------------------------------|
| **Domain**           | Entidades, Value Objects, interfaces de repositório, regras de negócio puras |
| **Application**      | Casos de Uso (Use Cases), DTOs, portas de serviço     |
| **Infrastructure**   | Implementações de repositório (SQLC/PGX), serviços externos (auth) |
| **Interfaces/HTTP**  | Handlers HTTP, middlewares (JWT, logging, correlation) |

### Stack Técnica

- **Linguagem:** Go 1.25.5
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
| POST | `/api/planning/run-pipeline` | **Pipeline MRP→CRP→APS** num disparo + parecer de viabilidade (escopo `planning:run`) |
| GET | `/api/mrp-calculation/profile/{item_code}/{plan_code}` | Perfil MRP de um item |
| POST | `/api/mrp-calculation/configured-rules` | Cria regra configurada para item |
| GET | `/api/mrp-calculation/configured-rules/{item_code}` | Lista regras configuradas |
| GET | `/api/mrp-calculation/exceptions/{plan_code}` | Lista exceções do plano |
| GET | `/api/mrp-calculation/suggestions/{plan_code}` | Lista sugestões geradas pelo plano |
| POST | `/api/mrp-calculation/suggestions/{code}/firm` | **Firma sugestão** → cria Ordem Planejada real |

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
| PATCH | `/api/sales-order/{code}/status` | Altera status (status `"P"` ✅ **gera demanda independente** por item; `"F"` = Faturado, marcado ao autorizar a NF-e de saída) |
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

> Nota: o **CRUD de Cliente já existe** (`/api/customers`); o item correspondente, válido
> em 2026-05-31, foi resolvido depois e removido desta lista. As lacunas abaixo seguem
> abertas (módulo Pedido de Venda, fora do escopo do Plano de Corte).

- Pastas: Transporte, Descontos/Acréscimos, Forma de Pagamento no pedido
- Cálculo automático de impostos nos itens (campos de alíquota existem mas não são preenchidos automaticamente)
- Workflow de aprovação (análise comercial/financeira)

### 4.2 Pedido de Compra (Purchase Order)

> **Pedido de Compra completo** (capa e itens estendidos, resolução automática de
> preço/UM/IPI, sugestões do MRP, solicitações de compra, geração de pedidos e cotação)
> está em [`manufatura-e-compras.md`](manufatura-e-compras.md) **§13–§16**. Abaixo, apenas o
> núcleo do recurso.

Grupos de rotas relacionados (detalhados no `manufatura-e-compras.md`):
`/api/purchase-order/{code}/items` (item com resolução automática),
`/api/purchase-order/suggestions/*` (sugestões MRP),
`/api/purchase-requisitions/*` (solicitações + geração de pedidos),
`/api/purchase-quotations/*` (cotação).

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
| `PARTIAL` | Parcialmente recebido (✅ atualizado pela entrada de NF-e) |
| `RECEIVED` | Totalmente recebido (✅ atualizado pela entrada de NF-e) |
| `CANCELLED` | Cancelado |

> ✅ Ao importar a **NF-e de entrada** com `purchase_order_code`, as quantidades
> recebidas baixam os itens (`received_qty`) e recalculam o status da linha e do
> cabeçalho (`PARTIAL`/`RECEIVED`). Ver `fiscal-financeiro.md` §5.

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
| POST | `/api/production-order/appointment` | Registra apontamento (tempo + quantidade); com `backflush_warehouse_id` ✅ **baixa a BOM** (movimentos `OUT`) |
| POST | `/api/production-order/consumption` | Registra consumo de matéria-prima; com `warehouse_id` ✅ gera movimento **`OUT`** do insumo |
| POST | `/api/production-order/{id}/complete` | Conclui produção (IN_PROGRESS → COMPLETED); com `warehouse_id` ✅ gera movimento **`IN`** do acabado |
| POST | `/api/production-order/{id}/close` | Fecha ordem (COMPLETED → CLOSED); ✅ **apura o custo real** |
| POST | `/api/production-order/{id}/cancel` | Cancela ordem (→ CANCELLED) |
| GET | `/api/production-order/{id}/appointments` | Lista apontamentos |
| GET | `/api/production-order/{id}/consumptions` | Lista consumos |
| POST | `/api/production-order/{id}/settle-cost` | Apura/recalcula o custo real da OF (material a custo médio + conversão) |
| GET | `/api/production-order/{id}/cost` | Consulta a apuração de custo real + variâncias |
| POST | `/api/production-order/{id}/scrap-return` | Retorna sucata/retalho ao estoque como subproduto valorizado (`IN`) |

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

> ✅ Todo movimento **atualiza o saldo** (`stock_balances`) na mesma transação:
> quantidade (`IN`/`TRANSFER_IN` somam, `OUT`/`TRANSFER_OUT` subtraem), custo médio
> ponderado (recalculado nas entradas) e último custo. Tipos canônicos: `IN`, `OUT`,
> `TRANSFER_IN`, `TRANSFER_OUT`, `ADJUSTMENT`.

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/stock/movements/create` | Cria movimento de estoque (atualiza saldo) |
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
| GET | `/api/stock/balances/atp/{itemCode}` | **Disponível para promessa (ATP)** — saldo − reservas (opcional `?mask=`) |

**Rastreabilidade de Lote / Corrida:**

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/stock/lots/register` | Registra lote/corrida (heat number) + certificado de qualidade |
| GET | `/api/stock/lots/item/{itemCode}` | Saldos por lote do item |
| GET | `/api/stock/lots/genealogy/{itemCode}/{lot}` | Genealogia do lote (consumido em / produzido por + lotes de entrada) |

**Consumo Médio (ROP):**

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/stock/consumption-average/recalc` | Recalcula consumo médio mensal (item específico ou todos) |
| GET | `/api/stock/consumption-average/{itemCode}` | Consulta consumo médio do item |

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

### 5.3 Expedição / Carregamento (romaneio)

Logística de saída (separação → conferência → despacho). Detalhe em
`manufatura-e-compras.md` §19. Migration `000146`.

| Método | Path | Descrição |
|--------|------|-----------|
| POST | `/api/shipments` | Cria romaneio (vincula a pedido de venda/transportadora) |
| GET | `/api/shipments` · `/api/shipments/{code}` | Lista / detalha |
| POST | `/api/shipments/{code}/items` | Adiciona item ao romaneio |
| POST | `/api/shipments/items/confer` | Confere um item (qtd conferida) |
| POST | `/api/shipments/{code}/confer` | Marca o romaneio como conferido |
| POST | `/api/shipments/{code}/ship` | Despacha (exige todos os itens conferidos) |
| POST | `/api/shipments/{code}/cancel` | Cancela o romaneio |

Status: `OPEN` → `SEPARATED` → `CONFERRED` → `SHIPPED` (`CANCELLED`).

---

## 6. Módulo Fiscal & Financeiro

> **A documentação fiscal e financeira é única e completa em
> [fiscal-financeiro.md](fiscal-financeiro.md).** Esta seção é apenas um índice
> navegacional — campos, regras, exemplos de request/response, parâmetros e cadastros
> de apoio ficam no doc dedicado, para evitar duplicação.

### Fiscal

Motor tributário (ICMS com diferimento, DIFAL/FCP e Res. SF 13/2012; IPI; PIS/COFINS),
NF-e de saída e de entrada (com importação por chave via FocusNFE), CT-e, apuração de
impostos, SPED Contábil (ECD) e cadastros de apoio.

| Tema | fiscal-financeiro.md |
|---|---|
| Configuração fiscal (pré-requisito) | §2 |
| Motor tributário + tabelas | §3, §3.1 |
| NF-e de saída | §4 |
| NF-e de entrada (+ FocusNFE) | §5 |
| CT-e | §6 |
| Apuração de impostos / Simples Nacional | §11, §27 |
| Relatórios fiscais | §12 |
| Parâmetros e cadastros de apoio | §16–§26, §28–§31 |
| SPED Contábil (ECD) | §32 |
| Cadastro de Fornecedores (integração) | §33 |
| Classificações Fiscais | §34 |
| Tipos de Operação de Entrada | §35 |
| Manifestação do Destinatário / Inutilização | §36 |
| IBPT/SCI (carga tributária aproximada) | §37 |
| CNAB 240 (remessa de boletos) | §38 |
| Balancete contábil | §39 |

### Financeiro

Contas a pagar/receber (aprovação, baixa, aging), fluxo de caixa projetado e realizado,
saldos, conciliação bancária (OFX) e cadastros base (contas bancárias, condições/formas
de pagamento, plano de contas, centros de custo).

| Tema | fiscal-financeiro.md |
|---|---|
| Cadastros base | §7 |
| Contas a pagar | §8 |
| Contas a receber | §9 |
| Fluxo de caixa & saldos | §10 |
| Conciliação Bancária (OFX) | §13 |
| Validação CNPJ/CPF | §14 |

---

## 7. Status dos Módulos

> Atualizado em 2026-05-31. Os cadastros de Cliente e Fornecedor e todo o épico de
> Compras foram implementados — veja os docs dedicados.

### Implementado

| Área | Documentação |
|---|---|
| Itens, BOM, estrutura | `README.md` |
| Máquina (tipos, tempos) | `maquinas-e-roteiro.md` |
| MRP (cálculo, exceções, parâmetros) | `mrp-calculo.md`, `manufatura-e-compras.md` |
| Roteiro, CRP, APS, Qualidade, Manutenção, Previsão, Restrições | `manufatura-e-compras.md` |
| Pedido de Venda · Produção · Estoque | esta visão geral (§4–§5) |
| Cadastro de Cliente | `cadastros-cliente.md` |
| Cadastro de Fornecedor (+ consulta SEFAZ) | `cadastros-fornecedor.md` |
| Compras: Pedido completo, Solicitação→Geração, Cotação, Conversão UM, Tabela de Preço, Fornecedor preferencial | `manufatura-e-compras.md` §10–§16 |
| Pipeline de Planejamento (MRP→CRP→APS), Backflush, Expedição/romaneio, Idempotência/Escopos | `manufatura-e-compras.md` §17–§20 |
| Fiscal (NF-e e/s, CT-e, apuração, SPED ECD) e Financeiro (CP/CR, fluxo, OFX) | `fiscal-financeiro.md` |
| Manifestação/Inutilização, IBPT/SCI, CNAB 240, Balancete | `fiscal-financeiro.md` §36–§39 |

### Automações do fluxo (2026-06-03 · atualizado 2026-06-10)

| Automação | Onde |
|---|---|
| Pedido de venda confirmado (`P`) → demanda independente | `sales_order_uc` |
| Pedido de venda confirmado (`P`) → **checagem de limite de crédito** (bloqueia se exceder) | `sales_order_uc/credit_check.go` |
| Pedido de venda confirmado (`P`) → **reserva de estoque disponível por item (ATP)** | `sales_order_uc/order_reserve.go` |
| Firmar ordem planejada PRODUCTION → cria a OF | `planned_order_uc` |
| Consumo da OF → `OUT`; conclusão → `IN` (com lote produzido); apontamento → backflush da BOM | `production_order_uc` |
| Fechar a OF (`close`) → **apura o custo real e a variância vs. padrão** | `production_order_uc/settle_production_cost_uc.go` |
| Movimento de estoque → atualiza saldo (qtd + custo médio) **e o saldo por lote/corrida** | `repository/stock` |
| Reserva criada/liberada/consumida → atualiza `reserved_qty` do saldo | `repository/stock` |
| NF-e entrada → baixa o pedido de compra (`received_qty`/status) | `fiscal_uc` + `purchase_order` |
| NF-e saída autorizada → `OUT` + baixa de reservas + pedido Faturado (`F`) + Conta a Receber | `fiscal_uc` |

### Pendências conhecidas

| Funcionalidade | Status |
|---|---|
| ATP / reserva automática no pedido de venda | ✅ (reserva o disponível por linha ao confirmar; `GET /api/stock/balances/atp/{itemCode}`) |
| Limite de crédito do cliente no pedido | ✅ (checa exposição = contas a receber + pedidos em aberto ao confirmar) |
| Custo real da OF + variância vs. padrão | ✅ (apurado ao fechar; `GET /api/production-order/{id}/cost`) |
| Rastreabilidade de lote/corrida + genealogia | ✅ (saldo por lote + `GET /api/stock/lots/genealogy/{itemCode}/{lot}`) |
| Sucata/retalho valorizado como subproduto | ✅ (`POST /api/production-order/{id}/scrap-return`) |
| Cálculo automático de consumo médio (CM) | ✅ (`POST /api/stock/consumption-average/recalc`) |
| Conciliação de cartões / borderô | ❌ (exige layouts de adquirentes) |
| Workflow de alçada do pedido de compra | ❌ (hoje só um campo) |
| Execução do inventário cíclico | ❌ (só configuração) |
| Frente de Caixa (PDV) | ❌ |
| Testes de integração E2E do MRP | ❌ |

---

## 8. Migrations

As migrações ficam em `migrations/` (formato `NNNNNN_nome.up.sql` / `.down.sql`),
aplicadas via `make migrate_up`. A base começa em `000001` (núcleo: itens, BOM,
estrutura, MRP) e segue incremental — marcos principais: Pedidos de Venda (`000089`),
Pedido de Compra (`000092`), Estoque (`000093`), Produção (`000094`), Fiscal
(`000095`), Financeiro (`000096`), Cliente (`000115`–`000118`), Fornecedor
(`000135`–`000136`), o épico de Compras/Fiscal (`000137`–`000144`), IBPT/SCI
(`000145`), Expedição/romaneio (`000146`), auditoria (`000151`), custo real da OF
(`000152`), rastreabilidade de lote/corrida (`000153`) e consumo médio (`000154`).

> Consulte `migrations/` para a lista completa e atual; cada doc de módulo cita a
> migração correspondente.

---

## 9. Segurança e infraestrutura HTTP

### 9.1 Autenticação JWT

- **Login:** `POST /users/login` — retorna token JWT
- **Registro:** `POST /users/register` — cria usuário
- **Algoritmo:** HMAC-SHA256 com secret configurável
- **Expiração:** definida pelo servidor
- **Claims:** `UserID` (UUID), `Role` (string)

### 9.2 Middleware de Autorização

Todos os endpoints sob `/api/*` exigem:

1. **JWT Middleware:**
   - Valida header `Authorization: Bearer <token>`
   - Extrai claims e popula `AuthUser` no contexto
   - Loga tentativas inválidas com IP

2. **Role Middleware:**
   - `RequireRole("ADMIN", "USER")` na maioria das rotas

3. **Permission Middleware (escopos):**
   - `RequirePermission(scope)` com mapa papel→escopos: `ADMIN` (tudo), `USER`
     (operacional, sem `admin`), `VIEWER` (somente leitura)
   - Escopos: `planning:run`, `purchase:approve`, `fiscal:authorize`,
     `financial:manage`, `item:activate`, `admin`
   - Aplicado às rotas sensíveis novas (pipeline, fiscal manifestação/inutilização/
     IBPT, CNAB, prontidão de item)

4. **AuthService (porta):**
   - `CanCreate()` — verifica permissão de criação
   - `CanBaixarContaPagar()` / `CanBaixarContaReceber()` — permissões financeiras
   - `CanApurarImpostos()` — permissão fiscal
   - `UserID(ctx)` — extrai UUID do usuário autenticado

### 9.3 Middlewares de Infraestrutura

| Middleware | Descrição |
|-----------|-----------|
| `CorrelationMiddleware` | Gera UUID por request e injeta nos logs |
| `RealIP` (chi) | Extrai IP real do request |
| `Recoverer` (chi) | Recupera de panics |
| `Timeout(60s)` (chi) | Timeout global de 60 segundos |
| `StripSlashes` (chi) | Normaliza trailing slash |
| `RequestLoggerMiddleware` | Loga método, path, status, duração |
| `Idempotency` | Em `POST/PUT/PATCH`, reproduz a resposta original quando o header `Idempotency-Key` se repete (TTL 24h, por instância). Marca replays com `Idempotent-Replayed: true`. |

### 9.4 Health Check

`GET /health` — endpoint público (sem autenticação), retorna `{"status": "ok", "timestamp": "...", "mask": "core-api"}`.

---

