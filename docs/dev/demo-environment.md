# Ambiente de Apresentação (Demo) — PanossoERP

Guia para o time de **front-end**: como subir, conectar e consumir o banco de
demonstração já populado com **~1 ano de operação fictícia** de uma metalúrgica
(catálogo, clientes, fornecedores, estoque, pedidos, produção, notas fiscais e
financeiro). Serve tanto para **demonstrações a clientes** quanto como
**ambiente de teste manual** do front.

> TL;DR: `make demo-bootstrap` → API em `http://localhost:5072` →
> login `admin@panossoerp.demo` / `Demo@12345`.

---

## 1. Topologia de ambientes

O projeto tem **4 ambientes isolados**, cada um com seu banco:

| Ambiente        | Banco                         | Porta DB | Porta API | Para quê                          |
|-----------------|-------------------------------|----------|-----------|-----------------------------------|
| **dev**         | Supabase (nuvem)              | 5432\*   | 5070      | Desenvolvimento (`.env`)          |
| **test**        | Docker `postgres-test`        | 5433     | 5071      | Testes automatizados (e2e/integração) |
| **demo** ← novo | Docker `postgres-demo`        | **5434** | **5072**  | **Apresentação + teste do front** |
| **prod**        | Docker `postgres`             | 5432     | 5070      | Produção single-node              |

\* O dev aponta para o Supabase remoto definido em `.env` (não é Docker local).

O ambiente **demo** é totalmente descartável e **nunca toca produção**. Arquivos:

- `docker-compose.demo.yml` — postgres-demo + migrate-demo + api-demo.
- `.env.demo` — variáveis montadas em `/app/.env` no container da API.
- `scripts/seed-demo.sql` — o seed (idempotente) com os dados de apresentação.
- Alvos `make demo-*` (ver abaixo).

---

## 2. Como subir

Pré-requisitos: Docker + Docker Compose. (Não precisa de Go, `psql` nem `migrate`
instalados no host — tudo roda em container.)

```bash
# Sobe postgres-demo + roda migrations + builda/sobe a API, e popula o banco:
make demo-bootstrap

# (equivalente manual)
make demo-up      # docker compose -f docker-compose.demo.yml up -d --build
make demo-seed    # popula via scripts/seed-demo.sql
```

Verifique:

```bash
curl http://localhost:5072/health          # {"status":"ok",...}
make demo-logs                             # logs da API demo
```

Outros alvos:

| Comando            | Efeito                                                        |
|--------------------|--------------------------------------------------------------|
| `make demo-up`     | Sobe a stack (constrói imagem se preciso).                    |
| `make demo-seed`   | (Re)popula o banco. **Idempotente**: limpa e recria os dados. |
| `make demo-down`   | Para os containers, **preserva** os dados (volume).          |
| `make demo-reset`  | Para e **apaga o volume** (zera o banco). Depois rode `demo-up` + `demo-seed`. |
| `make demo-migrate`| Reaplica migrations (após `git pull` com migrations novas).  |
| `make demo-logs`   | Segue os logs da API demo.                                    |

Para "resetar a demo do zero" antes de uma apresentação:

```bash
make demo-reset && make demo-bootstrap
```

---

## 3. Conexão

### API (o que o front consome)

- **Base URL:** `http://localhost:5072`
- **CORS:** liberado (`*`).
- **Rate limit:** alto (300 rps), à vontade para a demo.
- **Auth:** JWT Bearer (ver seção 4).

### Banco (acesso direto, se precisar)

```
postgres://panossoerp_demo:panossoerp_demo_pass@localhost:5434/panossoerpdatabase_demo?sslmode=disable
```

```bash
docker exec -it -e PGPASSWORD=panossoerp_demo_pass panossoerp-postgres-demo \
  psql -U panossoerp_demo -d panossoerpdatabase_demo
```

---

## 4. Autenticação

Usuário admin já criado pelo seed:

| Campo  | Valor                    |
|--------|--------------------------|
| e-mail | `admin@panossoerp.demo`  |
| senha  | `Demo@12345`             |
| role   | `ADMIN`                  |

Fluxo:

```bash
# 1) Login → retorna { "token": "<JWT>" }
curl -s -X POST http://localhost:5072/users/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@panossoerp.demo","password":"Demo@12345"}'

# 2) Use o token em TODAS as rotas /api/*:
curl http://localhost:5072/api/items/ -H "Authorization: Bearer <JWT>"
```

Para criar mais usuários: `POST /users/register` (`{name,email,password}`) — novos
usuários nascem com role `USER`. As rotas `/api/*` aceitam `ADMIN` e `USER`; as de
financeiro exigem login válido (o admin do seed já cobre tudo).

---

## 5. O que tem no banco (volume e período)

Período coberto: **2025-07-01 → data atual** (≈12 meses), com distribuição mensal
realista de pedidos e faturamento.

| Domínio                        | Qtde aprox. | Observação                                            |
|--------------------------------|-------------|-------------------------------------------------------|
| Itens (produtos/MP/serviços)   | 145         | 50 acabados (10001+), 80 matérias-primas (20001+), 15 serviços (30001+) |
| Clientes                       | 50          | códigos 1..50, com CNPJ e condição de pagamento       |
| Fornecedores                   | 35          | códigos 1..35, homologados                            |
| Saldos de estoque              | 130         | MP no depósito 1, acabados no depósito 2              |
| Movimentos de estoque          | ~3.000      | entradas (compra/produção) e saídas (venda)           |
| Estruturas de produto (BOM)    | ~150        | cada acabado consome 2–4 matérias-primas              |
| Roteiros de fabricação         | 50          | 1 por produto acabado                                 |
| **Pedidos de venda**           | **1.500**   | itens: ~3.750. Status realista (ver 5.1)              |
| **Pedidos de compra**          | **700**     | itens: ~1.400                                         |
| **Ordens de fabricação**       | **600**     | concluídas / em andamento / abertas                   |
| **Notas fiscais de saída**     | ~760        | geradas dos pedidos faturados (+ itens)               |
| **Contas a receber**           | ~1.550      | parceladas; recebidas / pendentes / vencidas          |
| **Contas a pagar**             | ~550        | das compras recebidas; pagas / pendentes / vencidas   |

Cadastros de apoio também populados: empresa, 4 depósitos, 4 condições de
pagamento, 3 contas bancárias, 6 centros de custo, 5 classificações fiscais (NCM),
10 funcionários, 6 tipos de máquina + 8 máquinas, 6 operações.

### 5.1 Distribuição de status (útil para filtros/telas)

- **Pedido de venda** (`status`): `F` Faturado (≈50%), `P` Pedido, `A` Análise,
  `R` Rascunho, `OF` Orçamento. Itens: `DELIVERED` / `PARTIAL` / `OPEN`.
- **Pedido de compra**: `RECEIVED` (≈40%), `APPROVED`, `PARTIAL`, `REQUESTED`,
  `DRAFT`, `CANCELLED`.
- **Ordem de fabricação**: `COMPLETED` (maioria), `CLOSED`, `IN_PROGRESS`, `OPEN`.
- **Contas a receber/pagar**: `RECEBIDO`/`PAGO`, `PENDENTE`, `VENCIDO`.
- **NF de saída**: `AUTHORIZED`.

---

## 6. Endpoints principais (verificados nesta demo)

Todas exigem `Authorization: Bearer <JWT>`. Lista completa em `api/api.go`.

### Cadastros
```
GET /api/items/                       # lista de itens (envelope { "data": [...] })
GET /api/items/search/{code}          # item por código
GET /api/customers/                   # clientes
GET /api/suppliers/                   # fornecedores
GET /api/warehouse/list               # depósitos
GET /api/machine/list                 # máquinas
GET /api/employee/list                # funcionários
GET /api/items/structure/{code}/children   # estrutura (BOM) de um item
```

### Comercial / Compras / Produção
```
GET /api/sales-order/list
GET /api/sales-order/{code}
GET /api/sales-order/status/{status}      # ex.: /status/F
GET /api/sales-order/customer/{code}
GET /api/sales-order/items/{code}         # itens de um pedido
GET /api/purchase-order/list
GET /api/purchase-order/status/{status}   # ex.: /status/RECEIVED
GET /api/production-order/list
GET /api/production-order/{id}
GET /api/production-order/{id}/cost
```

### Financeiro (telas de fluxo + relatórios)
```
GET /api/financial/contas-receber/list
GET /api/financial/contas-receber/aging
GET /api/financial/contas-pagar/list
GET /api/financial/contas-pagar/aging
GET /api/financial/fluxo-caixa
GET /api/financial/saldo-contas
GET /api/financial/relatorios/dre?inicio=2025-07-01&fim=2026-06-30
GET /api/financial/contas-bancarias/list
GET /api/financial/condicoes-pagamento/list
GET /api/financial/centros-custo/list
```

### Demais módulos disponíveis (grupos de rota em `api/api.go`)
`/api/accounting`, `/api/aps`, `/api/bom`, `/api/cnpj`, `/api/cost-center`,
`/api/crp`, `/api/cutting-plans`, `/api/fiscal`, `/api/fiscal-classifications`,
`/api/maintenance`, `/api/mrp-calculation`, `/api/planned-order`,
`/api/purchase-quotations`, `/api/purchase-requisitions`, `/api/quality`,
`/api/reports`, `/api/routing`, `/api/sales-forecast`, `/api/shipments`,
`/api/stock`, `/api/standard-cost`, `/api/warehouse` … (ver arquivo de rotas).

> Os módulos transacionais centrais (vendas, compras, produção, fiscal de saída,
> financeiro, estoque) estão **densamente populados**. Módulos auxiliares
> (qualidade, manutenção, contabilidade, MRP, planos de corte) têm a estrutura
> pronta mas **poucos ou nenhum** registro de exemplo — dá para criar via API
> durante a demo, ou pedir para ampliar o seed.

---

## 7. Convenções e detalhes que ajudam no front

- **Envelope de resposta:** parte dos endpoints retorna `{"data":[...]}` (ex.:
  `/api/items/`, `/api/warehouse/list`) e parte retorna o array direto (ex.:
  `/api/customers/`, `/api/sales-order/list`). Trate ambos.
- **Casing dos campos:** a maioria é `snake_case`; alguns endpoints financeiros
  retornam `PascalCase` (ex.: `contas-receber/list` → `ID`, `NumeroDocumento`,
  `ClienteID`). Confira por endpoint.
- **Item sem campo "name":** a descrição do item vem de
  `pdm.description_technique` (padrão PDM). Use esse campo como rótulo.
- **Códigos vs IDs:** pedidos de venda/compra são referenciados por `code`
  (ex.: `/sales-order/{code}`); itens, clientes e fornecedores têm `code` == `id`
  no seed (facilita correlação).
- **`/api/stock`** não é uma rota de listagem (use as rotas de saldo/depósito do
  módulo de estoque conforme `api/api.go`).
- **Datas:** transações têm `created_at`/`emission_date` retroagidos ao longo dos
  12 meses — bom para gráficos de série temporal e relatórios por período.

---

## 8. Re-seed / reprodutibilidade

O seed é **determinístico** (`setseed`) e **idempotente**: cada `make demo-seed`
faz `TRUNCATE ... RESTART IDENTITY` das tabelas populadas e recria tudo do zero.
Para ampliar volume ou adicionar exemplos em módulos auxiliares, edite
`scripts/seed-demo.sql` (está comentado por seção) e rode `make demo-seed`.
