# PanossoERP

ERP industrial desenvolvido em Go com Clean Architecture, PostgreSQL (Supabase) e geração de queries via SQLC.

## Stack

| Camada | Tecnologia |
|---|---|
| Linguagem | Go 1.25.5 |
| Router HTTP | chi v5 |
| Driver de banco | pgx v5 |
| Geração de queries | SQLC v1.31.1 |
| Banco de dados | PostgreSQL (Supabase) |
| Autenticação | JWT (golang-jwt v5) |
| Configuração | Viper |

## Arquitetura

```
internal/
├── domain/            # Entidades, repositórios (interfaces) e regras de negócio
├── application/       # Use cases e DTOs
├── infrastructure/    # Implementações concretas (SQLC, auth, repositórios)
└── interfaces/        # Handlers HTTP
```

A dependência flui em uma única direção: `interfaces → application → domain ← infrastructure`.

## Módulos de domínio

| Módulo | Descrição |
|---|---|
| `items` | Cadastro de itens/produtos |
| `bom` / `bom_items` | Bill of Materials (estrutura de produto) |
| `structure` | Estrutura de itens com hierarquia pai/filho |
| `machine` | Tipos de máquina, máquinas e configuração de tempo por item |
| `mrp_calculation` | Cálculo MRP — explosão de demanda, sugestões de ordens e exceções |
| `industrial_calendar` | Calendário industrial com dias úteis |
| `item_calendar_promise` | Promessa de entrega via calendário |
| `independent_demand` | Demanda independente |
| `delivery_promise_params` | Parâmetros de promessa de entrega |
| `delivery_reschedule` | Reprogramação de entrega |
| `allocation_base` | Base de alocação |
| `cost_center` | Centro de custo |
| `employee` | Funcionários |
| `enterprise` | Empresa |
| `warehouse` | Armazém |
| `user` | Usuários e autenticação |
| `questions` / `questions_options` | Perguntas configuráveis para itens |
| `modifier` | Modificadores de produto |
| `component` | Componentes |

## API — Endpoints

Todos os endpoints abaixo exigem autenticação JWT. As roles aceitas são `ADMIN` e `USER`.

### Autenticação
```
POST /users/register
POST /users/login
```

### Itens
```
POST   /api/items/create
GET    /api/items/search/{code}
POST   /api/items/mask/generate
POST   /api/items/structure/create
PUT    /api/items/structure/update
GET    /api/items/structure/resolve/{itemCode}
```

### Máquinas
```
POST   /api/machine/create
GET    /api/machine/list
GET    /api/machine/{code}
POST   /api/machine/types/create
GET    /api/machine/types/list
GET    /api/machine/types/{code}
POST   /api/machine/item-time/configure
GET    /api/machine/item-time/{item_code}/{machine_code}
```

### MRP
```
POST   /api/mrp-calculation/run
GET    /api/mrp-calculation/profile/{item_code}/{plan_code}
POST   /api/mrp-calculation/configured-rules
GET    /api/mrp-calculation/configured-rules/{item_code}
```

### Outros módulos
```
POST   /api/allocations/create
GET    /api/allocations/list

POST   /api/cost-center/create
GET    /api/cost-center/list
GET    /api/cost-center/{costCenterCode}

GET    /api/delivery-promise-params/
PUT    /api/delivery-promise-params/update

POST   /api/delivery-reschedule/create
GET    /api/delivery-reschedule/list/{sales_order_code}

POST   /api/independent-demand/create
PUT    /api/independent-demand/update/{code}
DELETE /api/independent-demand/delete/{code}
GET    /api/independent-demand/list
GET    /api/independent-demand/list-from-date/{date}
GET    /api/independent-demand/list-by-item/{itemCode}
GET    /api/independent-demand/get-by-code/{code}

POST   /api/industrial-calendar/create
GET    /api/industrial-calendar/month/{year}/{month}
GET    /api/industrial-calendar/workdays/{year}/{month}
```

## Módulo de Máquinas e Produção

Gerencia como cada produto é fabricado, em qual máquina e quanto tempo leva.

**Cadastros:**
- **Tipo de Máquina** — categoriza equipamentos (corte, solda, pintura, torno etc.)
- **Máquina** — define capacidade, unidade, período e eficiência de cada equipamento
- **Configuração de Tempo por Item/Máquina** — tempo de ciclo, quantidade por lote, setup e prioridade por variante de produto

**Cálculo:**
```
ciclos = ceil(quantidade_pedido / quantidade_por_lote)
tempo_fabricacao = ciclos × tempo_por_ciclo
tempo_total = tempo_fabricacao + tempo_setup
```

O sistema converte automaticamente unidades entre item e máquina (kg↔t, mm↔m, m³↔L) e sinaliza gargalos de capacidade.

## Módulo MRP (Material Requirements Planning)

Implementa a explosão de demanda multi-nível (Low-Level Code) para geração de ordens de compra e produção.

**Fluxo do cálculo (`POST /api/mrp-calculation/run`):**
1. Carrega demandas independentes e pedidos de venda ativos
2. Explode a estrutura de produto (BOM) nível a nível via LLC
3. Considera estoque disponível, ordens firmes e parâmetros de reposição
4. Calcula datas de início via calendário industrial e lead time de máquina
5. Persiste:
   - `mrp_item_profiles` — perfil de necessidades por item/período
   - `mrp_planned_suggestions` — sugestões de ordens planejadas
   - `mrp_exception_messages` — mensagens de exceção (atrasos, gargalos, falta de configuração)
6. Registra log de execução em `mrp_calculation_logs` com status, contadores e erros

**Regras configuráveis por item (`configured_item_rules`):**
Permite parametrizar, por item, comportamentos de arredondamento de lote, lote mínimo, múltiplo de lote e outras regras de planejamento diretamente via API.

## Como executar

```bash
# Configurar variáveis de ambiente (DATABASE_URL, JWT_SECRET, etc.)
cp .env.example .env

# Rodar migrações
make migrate-up

# Gerar código SQLC (se necessário)
sqlc generate

# Iniciar o servidor
go run api/*.go
```

## Estrutura de arquivos relevantes

```
api/                    # Ponto de entrada e wiring de dependências
internal/
  domain/mrp_calculation/
    entity/             # MRPItemProfile, PlannedOrderSuggestion, MRPExceptionMessage
    repository/         # Interface MRPCalculationRepository
    service/            # Lógica MRP (explosão LLC, cálculo de datas)
  application/usecase/mrp_calculation_uc/
    calculate.go        # RunMRPCalculationUseCase
    get_profile.go      # GetItemProfileUseCase
    configured_rules.go # ManageConfiguredItemRulesUseCase
  infrastructure/database/sqlc/
    mrp_calculation.sql.go   # Queries geradas pelo SQLC
    mrp_exception_messages.go # Queries de exceções (manual, tipo em models.go)
    mrp_bulk.go              # ListLatestStockSnapshots (manual)
  migrations/
    000076_init.up.sql  # Tabelas MRP (mrp_calculation_logs, mrp_planned_suggestions, etc.)
    000077_init.up.sql  # configured_item_rules
```
