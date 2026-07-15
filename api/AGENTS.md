# AGENTS.md — Módulo `api/`

## Propósito

Este módulo é o ponto de entrada da aplicação VentureERP. Contém dois arquivos principais:

- **`main.go`**: Bootstrap da aplicação — carrega configurações via Viper (`.env`), conecta ao banco PostgreSQL via pgx, inicializa tracing OpenTelemetry e inicia o servidor HTTP.
- **`api.go`**: Monta todas as rotas HTTP (chi), instancia todos os use cases e handlers, define o struct `application` que agrupa todas as dependências.

## Como modificar

### Adicionar um novo grupo de rotas

1. No arquivo `api.go`, localize a função `NewAPI` ou o local onde as rotas são montadas.
2. Adicione um novo grupo de rotas usando o padrão chi:
   ```go
   r.Route("/novo-modulo", func(r chi.Router) {
       r.Get("/", handler.List)
       r.Post("/", handler.Create)
       r.Get("/{id}", handler.GetByID)
       r.Put("/{id}", handler.Update)
       r.Delete("/{id}", handler.Delete)
   })
   ```
3. O handler deve ser instanciado com suas dependências (use case) no struct `application`.
4. Se necessário, adicione middlewares específicos da rota (autenticação, autorização, etc.).

### Adicionar um novo handler ao `application`

1. Importe o pacote do handler e do use case.
2. Adicione os campos correspondentes ao struct `application`.
3. Instancie o use case com suas dependências (repositórios, serviços).
4. Instancie o handler passando o use case.
5. Registre as rotas na função de montagem.

### Estrutura do servidor

- Porta padrão: **5070** (configurável via `SERVER_PORT` no `.env`)
- Shutdown graceful com timeout configurável
- CORS configurado para aceitar origens definidas no `.env`

## Regras importantes

- O `main.go` deve permanecer enxuto — apenas bootstrap, sem lógica de negócio.
- O `api.go` é o único ponto de wiring de dependências. Não crie wiring em outros arquivos.
- Nunca importe pacotes de infraestrutura diretamente no `main.go` além do necessário para bootstrap (config, DB, tracing).
- Middlewares globais (logging, métricas, CORS) são aplicados no router principal antes dos grupos de rotas.
- `GET /api/version` é público para permitir a trava de compatibilidade antes do login.
- Os endpoints `/api/system/update` são exclusivos de `ADMIN` e apenas escrevem/leem a fila persistente; handlers nunca executam comandos do host.
