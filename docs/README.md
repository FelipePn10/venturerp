# Documentação — PanossoERP

Índice de toda a documentação do ERP. Cada área tem **um** documento de referência;
não há duplicação entre eles.

## Mapa da documentação

| Documento | Escopo |
|---|---|
| [`API_OVERVIEW.md`](API_OVERVIEW.md) | **Visão geral da API e do sistema** — introdução, arquitetura, estrutura do projeto, MRP, Pedidos (Venda/Compra), Produção, Estoque, segurança/infra HTTP, status dos módulos e migrations. Aponta para os docs dedicados nas áreas aprofundadas. |
| [`FISCAL_FINANCEIRO.md`](FISCAL_FINANCEIRO.md) | **Único documento de Fiscal & Financeiro.** Motor tributário, NF-e (saída/entrada + FocusNFE), CT-e, apuração, SPED ECD, contas a pagar/receber, fluxo de caixa, OFX, e todos os cadastros de apoio fiscais (§16–§35). |
| [`../DOCUMENTATION.md`](../DOCUMENTATION.md) | **Módulos de Manufatura e Compras.** Roteiro, CRP, APS, Custo Padrão, Qualidade, Manutenção, Previsão, Restrições; e o épico de Compras (§10–§16): Fornecedor↔MRP, Conversão de UM, Tabela de Preço, Pedido de Compra completo, Fornecedor preferencial, Solicitação→Geração, Cotação. |
| [`customer_registration.md`](customer_registration.md) | **Cadastro de Cliente** — campos, pastas, regras e endpoints. |
| [`supplier_registration.md`](supplier_registration.md) | **Cadastro de Fornecedor** — campos, pastas, parâmetros, regras (IE/MEI/SEFAZ), defaults e integrações (compra/fiscal/MRP). |
| [`mrp_calculation.md`](mrp_calculation.md) | **MRP — detalhe do cálculo** (explosão LLC, netting, exceções). |
| [`MAQUINA.md`](MAQUINA.md) | **Cadastro de Máquina** — tipos, máquinas e tempos por item. |
| [`API_REQUEST_BODIES.txt`](API_REQUEST_BODIES.txt) | Coletânea de exemplos de corpo de request (JSON) por módulo. Para módulos mais novos, os exemplos estão no doc do próprio módulo. |
| [`../README.md`](../README.md) | README do projeto — stack, arquitetura, setup e módulos de domínio. |

## Convenções

- **Autenticação:** todos os endpoints `/api/*` exigem `Authorization: Bearer <JWT>`
  (token em `POST /users/login`); `Content-Type: application/json`.
- **Migrations:** ficam em `migrations/` (`make migrate_up`). Cada doc de módulo cita a
  migração correspondente.
- **Geração de queries:** SQLC (`make sqlc`); ver `project_sqlc_conventions` para
  gotchas (nullable BIGINT → `*int64`, enums via `VARCHAR + CHECK`, etc.).

## Testes

Duas camadas, separadas por **build tag** para que a suíte unitária seja rápida e
não dependa de banco:

| Tipo | Como rodar | O que cobre |
|---|---|---|
| **Unitários** | `make test` (ou `go test ./...`) | Value objects, domain services (máquina), engines (MRP/CPM, APS, CRP, Fiscal), regras de entidade e use cases com *fakes*. Sem banco. |
| **Cobertura** | `make test-cover` | Gera `coverage.out` e imprime o total. |
| **Integração** | `make test-integration` | Repositórios e fluxos ponta-a-ponta contra um **Postgres migrado**. Compilados só com `-tags=integration` e **pulados** se `TEST_DATABASE_URL` não estiver setado. |

- Os testes de integração ficam em `*_integration_test.go` com `//go:build integration`
  e usam `internal/infrastructure/testutil` (conexão, `UniqueCode()` em faixa alta
  `9_000_000_000+` e limpeza via `defer`).
- `make test-integration` usa `TEST_DATABASE_URL` (ou `DATABASE_URL` do `.env`). **O
  banco precisa estar migrado** (`make migrate_up`); cada teste cria e remove os
  próprios dados.
- Cobrem, entre outros: resolução de preço (específico×genérico), recomputo de status
  de atendimento da solicitação, `UFInGroup`, sequence/transação do pedido e o E2E
  **Solicitação → Geração de Pedidos**.

## Onde está cada assunto?

- **Fiscal ou financeiro?** → sempre `FISCAL_FINANCEIRO.md` (fonte única).
- **Compras (pedido, solicitação, cotação, preço, conversão UM)?** → `DOCUMENTATION.md` §10–§16.
- **Cliente / Fornecedor?** → docs dedicados de cadastro.
- **MRP / Produção / Estoque / Pedido de Venda?** → `API_OVERVIEW.md` (+ `mrp_calculation.md` para o cálculo).
- **Manufatura (roteiro, CRP, APS, qualidade, manutenção)?** → `DOCUMENTATION.md`.
