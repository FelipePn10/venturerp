# Go-Live — implantação, operação e backup

Guia operacional para colocar o VentureERP em produção numa **instância única**
(servidor on-prem de uma metalúrgica pequena). Cobre build, deploy com Docker,
migrations, healthchecks, métricas, segurança e backup/restore.

> O **front-end é um app desktop separado** que consome esta API. Este documento
> trata apenas do back-end (API + banco + operação).

---

## 1. Pré-requisitos

- Docker + Docker Compose **ou** Go 1.25 + PostgreSQL 16 no host.
- Um arquivo `.env` (copie de [`.env.example`](../../.env.example)) com, no mínimo:
  - `JWT_SECRET` — string longa e aleatória (obrigatório; o compose recusa subir sem ele).
  - `POSTGRES_PASSWORD` — senha do banco.
  - `CORS_ALLOWED_ORIGINS` — origens do app desktop em produção (ver §6).

Em produção, use [`.env.production.example`](../../.env.production.example)
como base. O `.env.example` permanece deliberadamente voltado ao ambiente de
desenvolvimento. Os dois ambientes nunca devem compartilhar banco, senhas ou
`JWT_SECRET`.

---

## 2. Deploy com Docker (recomendado)

```bash
cp .env.example .env      # edite os segredos
make up                   # postgres → migrations → api  (build automático)
# ou, com backup automático agendado:
make up-backup
```

O `docker compose` orquestra, **nesta ordem**:

1. `postgres` — sobe e fica *healthy* (`pg_isready`).
2. `migrate` — aplica todas as migrations pendentes e encerra (one-shot).
3. `api` — só inicia depois do passo 2 concluir com sucesso.
4. `backup` *(perfil `backup`)* — sidecar que faz `pg_dump` periódico.

Comandos úteis: `make logs` (tail da API), `make down` (derruba a stack),
`docker compose run --rm migrate` (reaplica migrations manualmente).

---

## 3. Deploy no host (sem Docker)

```bash
make build                # gera ./bin/erp (binário estático)
make migrate_up           # aplica migrations (precisa do CLI golang-migrate)
./bin/erp                 # lê .env / variáveis de ambiente
```

---

## 4. Migrations

- Fonte única em `migrations/` (`NNNNNN_*.up.sql` / `.down.sql`).
- Em container: serviço `migrate` (imagem `migrate/migrate`).
- No host: `make migrate_up`, `make migrate_down`, `make migrate_force`.

O binário e as migrations implantadas devem vir do mesmo commit. O fluxo de
produção é `develop` → pull request → `main`; o servidor executa somente
`main`. Faça backup antes de migrar, aplique as migrations e só então reinicie
a API. Branches de tarefa nascem de `develop` e são removidas após o merge.

---

## 5. Saúde e observabilidade

| Endpoint | Uso |
|---|---|
| `GET /health` e `GET /health/live` | **Liveness** — processo de pé (sem checar dependências). |
| `GET /health/ready` | **Readiness** — pinga o banco; retorna 503 se o banco estiver fora. Use no balanceador/orquestrador. |
| `GET /metrics` | Métricas Prometheus (RED): `http_requests_total`, `http_request_duration_seconds`, `http_requests_in_flight`, `app_uptime_seconds`. |

O container já tem `HEALTHCHECK` apontando para `/health/live`.

**Proteger `/metrics`:** defina `METRICS_TOKEN`; o scraper deve enviar
`Authorization: Bearer <token>`. Exemplo de scrape do Prometheus:

```yaml
scrape_configs:
  - job_name: venture-erp
    authorization: { credentials: "SEU_METRICS_TOKEN" }
    static_configs:
      - targets: ["erp-host:5070"]
```

Para desligar o endpoint: `METRICS_ENABLED=false`.

---

## 6. Segurança (checklist de produção)

- [ ] `JWT_SECRET` forte e único por ambiente.
- [ ] `CORS_ALLOWED_ORIGINS` explícito (ex.: `app://.,https://erp.suaempresa.com.br`).
      Vazio só é permissivo em `ENV=development`; em produção, **liste as origens**.
- [ ] `ENV=production`.
- [ ] PostgreSQL ligado somente a `127.0.0.1` quando banco e API compartilham
      o host; clientes desktop acessam apenas `https://api.venturerp.com`.
- [ ] Banco com senha forte e, idealmente, `sslmode=require`.
- [ ] Rate limit ativo: `RATE_LIMIT_RPS`/`RATE_LIMIT_BURST` (global) e
      `AUTH_RATE_LIMIT_RPM`/`AUTH_RATE_LIMIT_BURST` (anti brute-force no login).
- [ ] `MAX_BODY_BYTES` adequado (padrão 10 MiB cobre importação de XML de NF-e).
- [ ] `METRICS_TOKEN` definido se `/metrics` ficar exposto.

Já aplicado pela API em todas as requisições: headers de segurança
(`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`), limite de
corpo, CORS, rate limiting e **graceful shutdown** (drena requisições em voo ao
receber `SIGTERM`/`SIGINT`, dentro de `SHUTDOWN_TIMEOUT_SEC`).

O primeiro administrador deve ser provisionado por uma rotina operacional
controlada. Depois do bootstrap, `POST /users/register` exige JWT com papel
`ADMIN`; o endpoint não pode ser exposto como cadastro público em produção.

---

## 7. Backup e restore

Backup **lógico** com `pg_dump` formato custom (`-Fc`): comprimido e restaurável
de forma seletiva/paralela.

### Backup
```bash
make backup                      # one-off → ./backups/<db>-<timestamp>.dump
# automático (sidecar): make up-backup
#   BACKUP_INTERVAL_SECONDS (padrão 86400 = diário)
#   BACKUP_RETENTION_DAYS   (padrão 14; remove dumps mais antigos)
```
Cada backup é **verificado** (`pg_restore --list`) logo após ser gerado.

### Restore
```bash
make restore FILE=./backups/panossoerpdatabase-20260608-020000.dump
```

> ⚠️ **Restore é destrutivo** (`--clean --if-exists`): recria objetos. Teste o
> restore num banco descartável **periodicamente** — backup que nunca foi
> restaurado é esperança, não backup.

### Recomendações
- Replique `./backups` para fora do servidor (outro disco / nuvem / NAS).
- Combine com snapshot do volume `pgdata` se quiser PITR no futuro (WAL archiving).
- Registre no calendário um teste de restore mensal.

---

## 8. Trilha de auditoria (migration 000151)

Toda requisição **autenticada e mutante** (`POST/PUT/PATCH/DELETE` sob `/api/*`) é
registrada em `audit_log` — *quem* (user_id/role do JWT), *o quê* (método + rota +
path), *quando* e *resultado* (status, latência, IP, request_id). A captura é no
middleware HTTP e a gravação é **assíncrona** (worker em background com buffer):
nunca bloqueia nem derruba a requisição de negócio; se o buffer encher, o evento é
descartado com um warning. Leituras (`GET`) não são auditadas.

Consulta (somente **ADMIN**):

```
GET /api/audit-log?user_id=&route=&from=&to=&limit=&offset=
```
`from`/`to` em RFC3339; `limit` padrão 100, máx 500; ordenação por mais recente.

> Retenção: a tabela cresce de forma append-only. Defina uma política conforme a
> necessidade legal/fiscal (ex.: arquivar/expurgar registros com mais de N meses).

---

## 9. CI

`make ci` roda o que o pipeline valida em cada push: `fmt-check` (gofmt),
`vet`, `build` e `test-cover`. No Jenkins (`Jenkinsfile`) isso roda no estágio
**Build & Test** (imagem `golang:1.25`), e o **Qodana** segue na `main`.

---

## 10. Cobertura de Testes

A estratégia é **cobertura de valor, não de linha**. A lógica de negócio crítica
está 90–100% coberta nos pacotes onde mora; os use cases CRUD de delegação (único
`if !auth { return err }; return repo.X()`) não têm cobertura de linha mas têm
cobertura de contrato (integração).

| Pacote | Cobertura aprox. | O que está coberto |
|--------|-----------------|-------------------|
| `fiscal/engine` | ~92% | Todos os cenários ICMS (interna/interestadual/importada/ST/DIFAL), PIS/COFINS, IPI |
| `fiscal/sped` | 100% | Generate EFD: blocos 0/C/E/H/9, contagens de fechamento |
| `stock/entity` | 100% | `ApplyMovementCosting` (média ponderada) |
| `cnab` | ~96% | Geração de remessa 240 por banco |
| `mrp_uc` | coberto | `FirmarSugestaoMRPUseCase`: 5 cenários |
| `planned_order_uc` | coberto | `FirmPlannedOrderUseCase`: 5 cenários (incluindo OF automática) |
| `routing_uc` | coberto | Create, Update, ListByItem: 6 cenários |
| `purchase_quotation_uc` | coberto | Create, Get, List, entity.Balance(): 8 cenários |
| `nfse_uc` | coberto | Create (5 validações + ISS), Get, List: 7 cenários |
| `fiscal_uc` | ~11% linha / >90% fluxo | `authorize_fiscal_exit` (buildFocusItems, settleStockAndOrder), `get_danfe` |
| `financial_uc` | ~31% linha / >90% fluxo | Baixar CP/CR (juros/multa), adiantamentos, approve/cancel |
| `production_order_uc` | ~19% linha / 100% fluxo crítico | AddConsumption (OUT), Complete (IN com fallback) |
| `sales_order_uc` | ~21% linha / 100% fluxo crítico | `ChangeStatus` com demanda, idempotência, precedência |

**Executar:** `make test` (unitários) · `make test-integration` (requer `TEST_DATABASE_URL`).
