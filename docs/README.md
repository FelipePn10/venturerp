# Documentação — VentureERP

Toda a documentação do ERP, organizada em **duas versões** de cada área:

- 📘 **`apresentacao/`** — linguagem de negócio, para apresentar à empresa/cliente
  (o que cada módulo entrega, exemplos práticos, glossário). Sem detalhes técnicos.
- 🛠️ **`dev/`** — referência técnica para a equipe de desenvolvimento (endpoints,
  entidades, regras internas, migrations).

> Comece por [`apresentacao/00-fluxo-geral.md`](apresentacao/00-fluxo-geral.md)
> para a visão de ponta a ponta, ou por [`dev/00-fluxo-geral.md`](dev/00-fluxo-geral.md)
> para a versão técnica com endpoints e status de automação.

---

## Mapa por área

| Área | 📘 Apresentação (empresa) | 🛠️ Dev (técnico) |
|---|---|---|
| **Fluxo geral (ponta a ponta)** | [`00-fluxo-geral`](apresentacao/00-fluxo-geral.md) | [`00-fluxo-geral`](dev/00-fluxo-geral.md) · [`visao-geral`](dev/visao-geral.md) |
| **Cadastros** (cliente, fornecedor, item, empresa, estrutura, configurador) | [`cadastros`](apresentacao/cadastros.md) · [`configurador`](apresentacao/configurador.md) | [`cadastros-cliente`](dev/cadastros-cliente.md) · [`cadastros-fornecedor`](dev/cadastros-fornecedor.md) · [`cadastros-item`](dev/cadastros-item.md) · [`cadastros-apoio`](dev/cadastros-apoio.md) · [`configurador-produto`](dev/configurador-produto.md) · [`configurador-migracao-legado`](dev/configurador-migracao-legado.md) · [`desenhos-e-lotes`](dev/desenhos-e-lotes.md) |
| **Máquinas e Roteiro** | [`maquinas`](apresentacao/maquinas.md) | [`maquinas-e-roteiro`](dev/maquinas-e-roteiro.md) |
| **MRP e Planejamento** (MRP, CRP, APS, previsão, calendário, prioridade) | [`mrp-planejamento`](apresentacao/mrp-planejamento.md) | [`mrp-calculo`](dev/mrp-calculo.md) · [`manufatura-e-compras`](dev/manufatura-e-compras.md) · [`sequenciamento`](dev/sequenciamento-producao.md) · [`aceite enterprise`](dev/mrp-aceite-enterprise.md) |
| **Produção** (OF, operações, qualidade, manutenção, terceiros) | [`producao`](apresentacao/producao.md) · [`serviços de terceiros`](apresentacao/servicos-de-terceiros.md) | [`producao`](dev/producao.md) · [`manufatura-e-compras`](dev/manufatura-e-compras.md) · [`serviços de terceiros`](dev/servicos-de-terceiros.md) · [`ficha-producao-ferramenta`](dev/ficha-producao-ferramenta.md) |
| **Compras** (solicitação, cotação, pedido, recebimento, inspeção, contratos, EDI e IQF) | [`compras`](apresentacao/compras.md) | [`manufatura-e-compras`](dev/manufatura-e-compras.md) (§10–§16) · [`matriz de requisitos`](dev/compras-requisitos-checklist.md) |
| **Vendas e Expedição** | [`vendas`](apresentacao/vendas.md) | [`vendas`](dev/vendas.md) |
| **Romaneio / Expedição** (separação, conferência, volumes, transporte, NF-e) | [`romaneio`](apresentacao/romaneio.md) | [`romaneio`](dev/romaneio.md) |
| **Plano de Corte** (otimização 1D de barras/perfis/tubos + 2D de chapas/MDF) | [`plano-de-corte`](apresentacao/plano-de-corte.md) | [`plano-de-corte`](dev/plano-de-corte.md) |
| **Estoque** | [`estoque`](apresentacao/estoque.md) | [`estoque`](dev/estoque.md) |
| **Custos** (custo padrão, centro de custo, overhead) | [`custos`](apresentacao/custos.md) | [`custos`](dev/custos.md) |
| **Fiscal & Financeiro** | [`fiscal-financeiro`](apresentacao/fiscal-financeiro.md) | [`fiscal-financeiro`](dev/fiscal-financeiro.md) |
| **Contabilidade, NFS-e, Operações de entrada** | [`fiscal-financeiro`](apresentacao/fiscal-financeiro.md) (§10) | [`contabilidade-e-fiscal-complementos`](dev/contabilidade-e-fiscal-complementos.md) |
| **Integrações & Relatórios** (busca por CNPJ, exportação Excel/PDF/CSV) | [`cadastros`](apresentacao/cadastros.md) (§Busca por CNPJ / Exportação) | [`integracao-cnpj-e-exportacao`](dev/integracao-cnpj-e-exportacao.md) |

**Referência de apoio (dev):** [`API_REQUEST_BODIES.txt`](dev/API_REQUEST_BODIES.txt) — exemplos de corpo de request (JSON) por módulo.

---

## Convenções (dev)

- **Autenticação:** todos os endpoints `/api/*` exigem `Authorization: Bearer <JWT>`
  (token em `POST /users/login`); `Content-Type: application/json`. Papéis `ADMIN`/`USER`
  e permissões específicas (`PermFiscalAuthorize`, `PermFinancialManage`,
  `PermPlanningRun`, etc.) em rotas sensíveis.
- **Idempotência:** requisições mutáveis aceitam `Idempotency-Key` (retentativas seguras).
- **Migrations:** ficam em `migrations/` (`make migrate_up`). Cada doc de área cita a
  migração correspondente.
- **Geração de queries:** SQLC (`make sqlc`); ver `project_sqlc_conventions` para
  gotchas (nullable BIGINT → `*int64`, enums via `VARCHAR + CHECK`, etc.).

## Testes (dev)

Duas camadas, separadas por **build tag** para que a suíte unitária seja rápida e
não dependa de banco:

| Tipo | Como rodar | O que cobre |
|---|---|---|
| **Unitários** | `make test` (ou `go test ./...`) | Value objects, domain services (máquina), engines (MRP/CPM, APS, CRP, Fiscal), regras de entidade e use cases com *fakes*. Sem banco. |
| **BOM/MRP focado** | `make test-bom-mrp` | Validação rápida de BOM enterprise+ (`is_coproduct`, `is_fixed_qty`, substitutos) em MRP, custo, produção e plano de corte. Script: `scripts/test-bom-mrp.sh`. |
| **Cobertura** | `make test-cover` | Gera `coverage.out` e imprime o total. |
| **Integração** | `make test-integration` | Repositórios e fluxos ponta-a-ponta contra um **Postgres migrado**. Compilados só com `-tags=integration` e **pulados** se `TEST_DATABASE_URL` não estiver setado. |
| **E2E HTTP (corte)** | `make test-cutting` | Fluxo completo do **Plano de Corte** (1D/2D/true-shape, firmar, demanda de OP, export, agenda, rateio) via HTTP contra a API rodando. Define `BASE_URL`. Script: `scripts/test-cutting.sh`. |

---

## Onde está cada assunto?

- **Visão de negócio de qualquer módulo?** → `apresentacao/<área>.md`.
- **Endpoints, entidades, regras internas?** → `dev/<área>.md`.
- **Fluxo completo do produto (venda → MRP → produção → fiscal)?** → `*/00-fluxo-geral.md`.
- **Fiscal ou financeiro?** → `*/fiscal-financeiro.md` (+ `dev/contabilidade-e-fiscal-complementos.md`).
- **Oportunidades de melhoria do sistema** → [`../MELHORIAS.txt`](../MELHORIAS.txt).
