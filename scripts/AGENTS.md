# AGENTS.md — Módulo `scripts/`

## Propósito

Scripts de automação para desenvolvimento, teste, backup e operações do VentureERP. Scripts shell (bash) que automatizam tarefas repetitivas e servem como documentação executável de processos.

## Estrutura

```
scripts/
├── backup.sh              # Backup do banco de dados
├── restore.sh             # Restore do banco de dados
├── backup-loop.sh         # Loop de backup contínuo
├── seed-demo.sql          # Dados de seed para ambiente demo
├── fix_sqlc_output.go     # Correções pós-geração do sqlc
├── test-e2e.sh            # Teste end-to-end completo
├── test-cutting.sh        # Teste do módulo de plano de corte
├── test-gantt.sh          # Teste do módulo APS/Gantt
├── test-romaneio.sh       # Teste do módulo de romaneio/expedição
├── test-bom-mrp.sh        # Teste de BOM + MRP
├── test-purchase-receiving.sh  # Teste de recebimento de compras
├── test-procurement-governance.sh  # Teste de governança de compras
├── test-routing.sh        # Teste do módulo de roteiro
├── test-comercial-*.sh    # Testes dos módulos comerciais (15 scripts)
└── loadtest/              # Scripts de teste de carga
```

## Como modificar

### Adicionar um novo script de teste

1. Crie `scripts/test-<modulo>.sh`.
2. Siga o padrão dos scripts existentes:
   ```bash
   #!/usr/bin/env bash
   set -euo pipefail

   # Configurações
   BASE_URL="${BASE_URL:-http://localhost:5070}"
   TOKEN=""

   # Funções auxiliares
   login() { ... }
   api_call() { ... }

   # Testes
   echo "=== Teste: Criar recurso ==="
   # ...

   echo "=== Todos os testes passaram ==="
   ```
3. Torne o script executável: `chmod +x scripts/test-<modulo>.sh`.
4. Documente no início do script quais variáveis de ambiente são necessárias.

### Scripts de backup/restore

- `backup.sh`: Usa `pg_dump` para criar backups no diretório `backups/`.
- `restore.sh`: Usa `pg_restore` para restaurar um backup.
- Requerem variáveis de ambiente: `DATABASE_URL` ou parâmetros de conexão.

### Dados de seed

- `seed-demo.sql`: Inserts SQL puros para o ambiente de demonstração.
- Use IDs UUIDs fixos para referências consistentes entre tabelas.
- Deve ser idempotente (usar `ON CONFLICT DO NOTHING` ou verificar existência).

## Regras importantes

- **Scripts podem conter dados sensíveis** (tokens, CNPJs, credenciais). Por isso, a maioria está no `.gitignore`. Apenas `test-comercial-*.sh` e `loadtest/` são versionados.
- **Use `set -euo pipefail`** no início de todo script bash.
- **Use variáveis de ambiente** para configuração, com valores default razoáveis.
- **Scripts de teste** devem ser independentes e poder rodar em qualquer ordem.
- **Documente dependências** no início do script (ex: `# Requer: jq, curl, psql`).
- `fix_sqlc_output.go`: Execute com `go run scripts/fix_sqlc_output.go` após `sqlc generate`. Corrige issues conhecidas na saída do sqlc.
- Scripts de teste assumem que a API está rodando localmente em `localhost:5070`.

## Versionamento e atualização

- Os alvos `release`/`release-check` moram no `makefile` **minúsculo** (canônico).
  Não crie um `Makefile` maiúsculo: em hosts case-sensitive o GNU make carrega o
  minúsculo e o maiúsculo é ignorado (e os dois colidem em FS case-insensitive).
- `release.sh` só roda na `main` limpa, valida SemVer, testes e imagem, atualiza o
  CHANGELOG e — como `main` é protegida — publica via **PR auto-mesclado com admin**
  (branch `release/vX.Y.Z`), taggeando o commit resultante de `main`. Requer `gh`
  autenticado. Não faz mais push direto/atômico para `main`.
- `self-update.sh` é instalado no host e executado como root pelo systemd; nunca o
  chame de dentro do contêiner da API. Ele deriva `PGPASSWORD` do `DATABASE_URL`
  para os `docker exec` de `pg_dump`/`pg_restore`/`psql` (o Postgres exige senha
  mesmo em conexões locais).
- `deploy/production/provision-updater.sh` instala o updater no host de forma
  idempotente; `deploy/production/bootstrap-cutover.sh` faz o primeiro cutover
  binário-nativo → container reusando o `self-update.sh`.
- A configuração privilegiada fica em `/etc/venturerp/update.env` com modo `0600`, nunca no Git.
- Preserve `flock`, tag de imagem exata, backup verificado, readiness e rollback ao alterar o updater.
