# Ambiente de desenvolvimento isolado

O ambiente remoto de desenvolvimento executa a branch `develop` sem compartilhar
banco, JWT, credenciais ou configuração fiscal com produção.

| Recurso | Desenvolvimento | Produção |
|---|---|---|
| API | `https://dev-api.venturerp.com` | `https://api.venturerp.com` |
| Serviço | `venturerp-development.service` | `venturerp.service` |
| Porta local | `5071` | `5070` |
| PostgreSQL | container e volume próprios, porta `5433` | container e volume próprios, porta `5432` |
| Git | branch `develop` | branch `main` |
| Fiscal | Focus NFe `homologacao` | Focus NFe `producao` |
| Telemetria | `deployment.environment=development` | `deployment.environment=production` |

## Desenvolvimento local

Copie `.env.development.example` para `.env.development`, gere segredos próprios
e suba somente o banco. Caso `5433` já esteja ocupada pelo banco de testes,
defina `POSTGRES_HOST_PORT=5435` e use a mesma porta em `DATABASE_URL`:

```bash
docker compose --env-file .env.development -f docker-compose.development.yml up -d postgres-development
docker compose --env-file .env.development -f docker-compose.development.yml --profile tools run --rm migrate-development
set -a; . ./.env.development; set +a
go run ./api
```

Nunca reutilize `.env`, dumps, JWT ou credenciais de produção. O banco de
desenvolvimento deve conter somente dados fictícios.

## Deploy remoto

O checkout remoto fica em `/opt/venturerp/development`. Depois de alterações na
branch `develop`, execute testes, atualize o checkout, aplique migrations no banco
de desenvolvimento, compile `venturerp-api-development` e reinicie somente
`venturerp-development.service`.

Em nenhuma circunstância o deploy de desenvolvimento deve reiniciar
`venturerp.service` ou apontar para a porta `5432`.
