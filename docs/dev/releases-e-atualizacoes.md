# Releases e atualizações do VentureERP

## Princípio operacional

Commits não são releases. `develop` recebe trabalho em andamento, `main` representa produção e somente uma tag SemVer criada pelo comando oficial dispara distribuição. A VPS não executa `git pull`: recebe uma imagem imutável. O desktop instala apenas artefatos aceitos pela assinatura do updater Tauri.

## Criar uma versão do backend

Pré-requisitos: estar em `main`, worktree limpo, Docker ativo, `gh` autenticado
(`gh auth status`) e acesso ao `origin`. Os alvos de release moram no `makefile`
minúsculo (canônico); não crie um `Makefile` maiúsculo — em hosts case-sensitive
o GNU make carrega o minúsculo e o maiúsculo seria ignorado.

1. Integre e valide as mudanças em `develop` e leve para `main` (via PR).
2. Rode a regressão e revise migrations, compatibilidade mínima do desktop e rollback.
3. Execute `make release-check VERSION=1.4.0`; ele testa Go e constrói a imagem sem criar commit/tag.
4. Execute `make release VERSION=1.4.0`.
5. O comando atualiza `CHANGELOG.md` e cria o commit de release. Como `main` é
   **branch protegida**, ele não faz push direto: empurra o commit numa branch
   efêmera `release/vX.Y.Z`, abre e **auto-mescla o PR com privilégio de admin**
   e então publica a tag anotada no commit resultante de `main`. A tag — não o
   commit — dispara o pipeline.
6. Acompanhe **Release backend**: testes, imagem `ghcr.io/felipepn10/panossoerp:v1.4.0`, `latest`, SBOM/proveniência e GitHub Release.

Nunca mova uma tag publicada. Corrija com nova versão, como `1.4.1`.

## Contrato de compatibilidade

`GET /api/version` é público e retorna `{"version":"1.4.0","min_client":"1.4.0"}`. Os valores são injetados no binário; builds locais retornam `dev`. O desktop bloqueia uma versão inferior a `min_client` antes do login.

## Preparar o agente na VPS (provisionamento roteirizado)

Pré-requisitos no host: Docker Engine/Compose, `jq`, `curl`, `flock` e o
container do Postgres de produção em execução. Envie `deploy/production/*` e
`scripts/self-update.sh` preservando a estrutura de diretórios e rode, como root:

```bash
sudo ./deploy/production/provision-updater.sh
```

O script é idempotente e:

- instala `self-update.sh` e `bootstrap-cutover.sh` em `/opt/venturerp/updater`;
- instala `compose.yml` no mesmo diretório;
- gera `/etc/venturerp/update.env` (modo `0600`) derivando usuário/base/URL do
  `.env` de produção — a senha do Postgres é derivada do `DATABASE_URL` em tempo
  de execução, então `pg_dump`/`pg_restore`/`psql` via `docker exec` recebem
  `PGPASSWORD` corretamente;
- instala as unidades systemd e roda `systemctl enable --now venturerp-update.path`.

### Primeiro cutover: binário nativo → container

Se a produção ainda roda como binário nativo (`venturerp.service`), o primeiro
cutover para a imagem versionada usa o mesmo `self-update.sh`. Depois que a
imagem `ghcr.io/felipepn10/panossoerp:vX.Y.Z` existir:

```bash
sudo /opt/venturerp/updater/bootstrap-cutover.sh 1.0.0
```

Ele faz backup do banco, baixa a imagem, para o serviço nativo, aplica migrations,
sobe o container, valida `/health/ready` e `/api/version` e **desabilita** o
serviço nativo legado (para não disputar a porta 5070 no reboot). A partir daí,
toda atualização sai pelo botão do painel admin.

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
