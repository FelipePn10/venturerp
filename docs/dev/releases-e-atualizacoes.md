# Releases e atualizações do VentureERP

## Princípio operacional

Commits não são releases. `develop` recebe trabalho em andamento, `main` representa produção e somente uma tag SemVer criada pelo comando oficial dispara distribuição. A VPS não executa `git pull`: recebe uma imagem imutável. O desktop instala apenas artefatos aceitos pela assinatura do updater Tauri.

## Criar uma versão do backend

Pré-requisitos: `main`, worktree limpo, Docker ativo e acesso ao `origin`.

1. Integre e valide as mudanças em `develop`.
2. Revise a integração para `main` e rode a regressão.
3. Execute `make release-check VERSION=1.4.0`; ele testa Go e a imagem sem criar commit/tag.
4. Revise migrations, compatibilidade mínima do desktop e rollback.
5. Execute `make release VERSION=1.4.0`.
6. O comando atualiza `CHANGELOG.md`, cria commit e tag anotada e envia `main`/tag atomicamente.
7. Acompanhe **Release backend**: testes, imagem `ghcr.io/felipepn10/panossoerp:v1.4.0`, `latest`, SBOM/proveniência e GitHub Release.

Nunca mova uma tag publicada. Corrija com nova versão, como `1.4.1`.

## Contrato de compatibilidade

`GET /api/version` é público e retorna `{"version":"1.4.0","min_client":"1.4.0"}`. Os valores são injetados no binário; builds locais retornam `dev`. O desktop bloqueia uma versão inferior a `min_client` antes do login.

## Preparar o agente na VPS

Faça uma única vez em janela de manutenção:

1. Instale Docker Engine/Compose, `jq`, `curl` e `flock`.
2. Crie `/opt/venturerp/updater`, `/var/lib/venturerp-update` e `/var/backups/venturerp/releases`.
3. Instale `scripts/self-update.sh` como `/opt/venturerp/updater/self-update.sh`, root, modo `0750`.
4. Instale `deploy/production/compose.yml` no mesmo diretório, modo `0644`.
5. Crie `/etc/venturerp/update.env` a partir do exemplo, modo `0600`, com container/usuário/nome/URL reais do banco.
6. Dê ao UID `10001` escrita na fila; ela é a única ponte API-host.
7. Instale as unidades em `/etc/systemd/system`, rode `systemctl daemon-reload` e `systemctl enable --now venturerp-update.path`.
8. Confirme o `.path` ativo e que a API não monta `/var/run/docker.sock`.

## Atualizar pelo painel

1. Apenas `ADMIN` vê “Atualização disponível”.
2. O clique chama `POST /api/system/update`; a API grava `request.json`/`active.lock`. Repetição retorna HTTP 409.
3. O `.path` chama o agente root. O painel acompanha `GET /api/system/update/status`.
4. O agente cria `pg_dump` custom, valida com `pg_restore --list`, baixa a tag exata e extrai as migrations da imagem.
5. Ele para a API, migra, inicia a imagem e consulta `/health/ready`.
6. Em sucesso, registra a imagem e retém backup por 30 dias. Em falha, para a versão nova, restaura o banco e inicia a imagem anterior ou o serviço legado.

Estados: `idle`, `queued`, `running`, `succeeded`, `failed`, `rolled_back`. Nunca remova `active.lock` com o serviço ativo.

## Diagnóstico e recuperação

Consulte `journalctl -u venturerp-update.service`, `/var/lib/venturerp-update/status.json` e `docker logs venturerp-api`. Se o host reiniciar, valide banco/lock e reexecute a unidade; a solicitação permanece em disco. Após uma release defeituosa, publique uma versão corrigida; não mova tags.

## Segurança

A API roda sem privilégios, sem socket Docker e sem credenciais do registry. A configuração root é `0600`. Versões aceitam SemVer estrito e conteúdo da fila nunca é avaliado como shell. Mantenha branch protection e aprovação do ambiente de produção.
