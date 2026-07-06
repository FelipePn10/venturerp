# Observabilidade VentureERP — Stack Completa

Esta stack fornece uma visao completa de metricas, logs e traces distribuidos do
VENTURE ERP usando Grafana como centralizador.

---

## Arquitetura da Stack

```
┌─────────────────────────┐
│      VentureERP API     │──── OTLP (traces) ─────┐
│    (Go/CHI + otelhttp)  │                         │
│    :5070 /metrics       │─── Prometheus scrape ───┤
│    stderr → Docker logs │                         │
└─────────────────────────┘                         │
                                                    ▼
┌─────────────────────────┐    ┌──────────────────────────────┐
│   Postgres Exporter     │    │  OpenTelemetry Collector      │
│   :9187                 │    │  (recebe OTLP, exporta para   │
│   Postgres metrics      │    │   Tempo/Prometheus)           │
└───────┬─────────────────┘    └──────┬───────┬───────┬───────┘
        │                             │       │       │
        │ ┌───────────────────────────┘       │       │
        │ │     (scrape every 15s)            │       │
        ▼ ▼                                   ▼       ▼
┌───────────────┐    ┌───────────────┐    ┌────────────────┐
│  Prometheus   │    │    Loki       │    │     Tempo      │
│  metricas TS  │    │   logs TS     │    │  traces TS     │
│  :9090        │    │  :3100        │    │  :3200         │
└───────┬───────┘    └───────┬───────┘    └────────┬───────┘
        │                    │                     │
        └────────────────────┼─────────────────────┘
                             ▼
                    ┌──────────────────┐
                    │     Grafana      │
                    │    :3000         │
                    │  Dashboards      │
                    │  Explore/Logs    │
                    │  Traces          │
                    └──────────────────┘

┌───────────────┐
│ Grafana Alloy │──logs Docker─► Loki
│ :12345        │
└───────────────┘
```

---

## Fluxo de Dados

### 1. Metricas (Prometheus)
**O que coleta:** Contadores, histogramas, gauges — metricas numericas em series temporais.

**Como chega:**
- A API gera metricas via middleware proprio em `/metrics` (formato texto Prometheus).
- O postgres-exporter expoe metricas do banco em `/metrics` (formato texto Prometheus).
- O OTel Collector tambem expoe suas metricas internas em `:8889/metrics`.
- O Prometheus faz `scrape` de todos esses `/metrics` a cada 15s e armazena em series temporais com retencao de 7 dias.

**Metricas coletadas pela API:**
| Metrica | Tipo | Descricao |
|---------|------|-----------|
| `http_requests_total{method, route, status}` | Counter | Total de requisicoes HTTP |
| `http_request_duration_seconds_bucket{method, route, le}` | Histogram | Distribuicao de latencia (11 buckets) |
| `http_requests_in_flight` | Gauge | Requisicoes em andamento no momento |
| `app_uptime_seconds` | Gauge | Tempo desde o inicio da aplicacao |
| `pg_up` (via postgres-exporter) | Gauge | Saude do banco (1=ok, 0=down) |

**Por que Prometheus (e nao VictoriaMetrics):** Para este monolito local, Prometheus e mais
simples de operar e ja e nativamente compreendido pelo Grafana. VictoriaMetrics faria
sentido com alto volume de metricas, longa retencao (>30d) ou multi-cluster.

### 2. Logs (Loki + Grafana Alloy)
**O que coleta:** Logs estruturados (JSON) da aplicacao e containers.

**Como chega:**
- A API escreve logs JSON no stdout/stderr (via `log/slog`).
- O Grafana Alloy (substituto do Promtail, que ficou EOL em marco/2026) le os
  logs dos containers Docker via socket (`/var/run/docker.sock`), parseia campos JSON (`level`,
  `msg`, `request_id`, `method`, `path`, `status`, `latency_ms`), e os envia para
  o Loki.
- O Loki armazena os logs indexados por labels (compose_service, level, method,
  status) com retencao de 7 dias.

**Campos extraidos dos logs (via config.alloy):**
| Campo | Origem | Uso |
|-------|--------|-----|
| `compose_service` | Label Docker | Filtrar logs por servico (`api`) |
| `level` | JSON log | Filtrar por severidade (debug, info, warn, error) |
| `msg` | JSON log | Mensagem do log |
| `request_id` | JSON log | Rastrear uma requisicao especifica |
| `method` | JSON log | Metodo HTTP da requisicao |
| `path` | JSON log | Rota da requisicao |
| `status` | JSON log | Codigo de status HTTP |
| `latency_ms` | JSON log | Latencia da requisicao em milissegundos |

**Por que Loki + Alloy (e nao FluentBit):** Loki integra-se nativamente com o
Grafana para correlacionar logs com traces (via `trace_id`). Alloy e o substituto
oficial do Promtail e ja faz o trabalho de coleta de logs Docker sem precisar de
outro agente. FluentBit seria viavel se precisassemos enviar logs para
multiplos destinos simultaneos.

### 3. Distributed Tracing (OTel Collector + Tempo)
**O que coleta:** Spans — registros de cada etapa de uma requisicao, com
parentesco entre spans e contexto propagado via headers HTTP.

**Como chega:**
- O codigo Go instrumenta cada requisicao HTTP com OpenTelemetry
  (`otelhttp.NewMiddleware` no router CHI).
- Os spans sao exportados via OTLP HTTP para o OpenTelemetry Collector
  (`otel-collector:4318`).
- O OTel Collector processa (batch, memory) e encaminha os traces para o Tempo
  (armazenamento local, retencao 7 dias).

**Dados em cada span:**
| Atributo | Descricao |
|----------|-----------|
| `service.name` | Nome do servico (`panossoerp-api`) |
| `deployment.environment` | Ambiente (`local`, `production`) |
| `http.method` | Metodo HTTP |
| `http.route` | Rota da requisicao |
| `http.status_code` | Codigo de resposta |
| `http.duration_ms` | Duracao do span |
| `trace_id` | ID unico do trace (correlaciona com logs) |
| `span_id` | ID unico do span |

**Por que OTel Collector + Tempo (e nao Jaeger):**
- OTel Collector e vendor-neutral e processa/metricas/logs/traces no mesmo
  pipeline, seguindo o padrao CNCF.
- Tempo integra profundamente com Grafana (TraceQL, service graphs,
  Trace-to-logs, Trace-to-metrics) e nao requer componentes extras como
  Cassandra/Elasticsearch (usa object storage local).
- Jaeger e mais voltado para debug isolado de tracing; Tempo e desenhado para
  observabilidade como um todo.

---

## Integracao OpenTelemetry no Codigo

### Dependencias adicionadas

```bash
go get go.opentelemetry.io/otel@latest
go get go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp@latest
go get go.opentelemetry.io/otel/sdk@latest
go get go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp@latest
go mod vendor
```

> **Nota:** O pacote `go.opentelemetry.io/contrib/instrumentation/github.com/go-chi/chi/otelchi`
  nao esta disponivel como modulo Go independente no proxy. Em seu lugar,
  usamos `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`, que
  funciona com qualquer `http.Handler` (incluindo CHI, ja que implementa a
  interface padrao). A API e exatamente a mesma: `otelhttp.NewMiddleware("service-name")`.

### Arquivos criados/modificados

| Arquivo | O que faz |
|---------|-----------|
| `internal/infrastructure/observability/tracing.go` | Funcao `InitTracing` — inicializa o TracerProvider com exporter OTLP HTTP, configura resource (service.name, namespace, environment) e propagation (TraceContext + Baggage). Retorna funcao de shutdown para drenar spans pendentes. |
| `api/main.go` | Inicializa tracing apos carregar config e antes de criar o router. O shutdown do tracing e garantido via `defer`. |
| `api/api.go` | Adiciona `otelhttp.NewMiddleware("panossoerp-api")` como primeiro middleware do router CHI. |

### Variaveis de ambiente (injetadas pelo compose de observabilidade)

```env
OTEL_SERVICE_NAME=panossoerp-api
OTEL_RESOURCE_ATTRIBUTES=deployment.environment=local,service.namespace=panossoerp
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
OTEL_TRACES_EXPORTER=otlp
OTEL_METRICS_EXPORTER=none
OTEL_LOGS_EXPORTER=none
```

> O SDK OTel le automaticamente essas variaveis. Nao e necessario configura-las
> manualmente no codigo Go — o `otlptracehttp.New(ctx)` ja as respeita.

---

## Como Subir o Ambiente

### Pre-requisitos

- Docker e Docker Compose instalados.
- Arquivo `.env` com `JWT_SECRET` definido.
- Portas `5070`, `3000`, `9090`, `3100`, `3200` livres.

### Subir tudo

```bash
docker compose -f docker-compose.yml -f docker-compose.observability.yml up -d --build
```

Isso sobe: Postgres, migracoes, API, Grafana, Prometheus, Loki, Tempo, OTel
Collector, Alloy e Postgres Exporter.

### Verificar saude

```bash
curl http://localhost:5070/health/live
curl http://localhost:5070/health/ready
curl http://localhost:5070/metrics
```

### Acessar

| Servico | URL | Credenciais |
|---------|-----|-------------|
| Grafana | <http://localhost:3000> | `admin` / `admin` |
| Prometheus | <http://localhost:9090> | — |
| Loki | <http://localhost:3100> | — |
| Tempo | <http://localhost:3200> | — |
| API | <http://localhost:5070> | JWT via `/users/login` |

---

## Como Usar os Dados

### No Grafana — Dashboard Pre-configurado

Abra `VentureERP` > `VentureERP - Overview`. O dashboard tem 7 paineis:

1. **Throughput API** — Requisicoes por segundo. Serve para identificar picos de
   carga e dimensionar recursos.
2. **Taxa de Erro 5xx** — Indice de falhas. Alerta precoce de bugs em producao.
3. **Latencia HTTP p95 por Rota** — 95% das requisicoes rodam abaixo deste
   valor. Rotas acima de 500ms merecem investigacao (queries lentas, N+1).
4. **Requisicoes por Rota e Status** — Volume de trafego segmentado.
5. **Postgres Up** — Monitor de saude do banco.
6. **Requisicoes em Andamento** — Conexoes simultaneas. Se crescer
   monotonicamente, ha vazamento de conexoes ou lentidao no banco.
7. **Logs da API** — Stream de logs JSON da aplicacao para correlacionar com
   metricas.

### Consultas Uteis

#### Prometheus (PromQL)

```promql
# Throughput por rota
sum(rate(http_requests_total[5m])) by (route)

# Top 5 rotas com mais erros 5xx
topk(5, sum(rate(http_requests_total{status=~"5.."}[5m])) by (route))

# Latencia p95 por rota
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, route))

# Saude do Postgres
pg_up

# Conexoes ativas no pool do Postgres
pg_stat_database_numbackends
```

#### Loki (LogQL)

```logql
# Todos os logs da API
{compose_service="api"}

# Apenas erros
{compose_service="api", level="error"}

# Requisicoes lentas (>500ms)
{compose_service="api"} | json | latency_ms > 500

# Logs de uma requisicao especifica
{compose_service="api"} |= "seu-request-id-aqui"
```

#### Tempo (TraceQL)

No Grafana, va em Explore → Tempo → Query type: Search:

```
{ service.name = "panossoerp-api" && status = error }
{ duration > 500ms && http.route =~ "/api/items.*" }
```

### Correlacionando Logs, Metricas e Traces

O Grafana faz isso automaticamente quando os datasources estao linkados:

1. Abra um trace no Tempo (Explore → Tempo).
2. Ao lado dos spans, o Grafana mostra logs do Loki com o mesmo `trace_id`
   (configurado em `datasources.yml`: `derivedFields` mapeia `trace_id` do Tempo
   para `trace_id` nos logs do Loki).
3. Voce pode pular de um pico de latencia no dashboard de metricas direto para
   os traces que geraram aquele pico (ex.: clicar no grafico de latencia p95 →
   "View traces").

---

## Operacao Local

### Parar tudo

```bash
docker compose -f docker-compose.yml -f docker-compose.observability.yml down
```

### Remover dados de observabilidade

```bash
docker compose -f docker-compose.yml -f docker-compose.observability.yml down -v
```

> Os volumes `grafana_data`, `prometheus_data`, `loki_data` e `tempo_data` sao locais e temporarios.

---

## Troubleshooting

### API nao inicia / Postgres unhealthy

O erro `container panossoerp-postgres-1 is unhealthy` geralmente ocorre quando:

1. As portas ja estao em uso. Verifique: `docker ps -a | grep postgres`.
2. O volume `pgdata` tem dados corrompidos de execucao anterior. Remova-o com:
   `docker compose -f docker-compose.yml down -v` e suba novamente.
3. O `.env` nao tem as credenciais corretas ou `POSTGRES_USER`/`POSTGRES_PASSWORD`
   padrao.

### "go: inconsistent vendoring"

Execute na ordem:
```bash
go mod tidy
go mod vendor
```

### Curl retorna "Failed to connect to localhost port 5070"

A API nao esta rodando. Verifique com `docker ps -a` se o container `panossoerp-api-1`
esta `Up`. Se estiver `Exited`, veja os logs: `docker logs panossoerp-api-1`.
