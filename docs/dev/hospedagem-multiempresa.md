# Hospedagem multiempresa — uma VPS, várias empresas clientes

Até a v1.3.0 a VPS atendia uma empresa (Tecnofer) e um ambiente de treinamento. A
entrada da Usimac tornou isso um modelo explícito: **cada empresa cliente é um
database próprio, um role próprio do PostgreSQL e um contêiner de API próprio,
atrás de um subdomínio próprio.**

## Por que isolamento físico, e não `enterprise_id`

O schema já é multiempresa: quase toda tabela operacional tem `enterprise_id`, e o
token carrega a empresa. Ainda assim, duas empresas com CNPJ distinto **não**
compartilham base aqui. Três razões concretas:

1. **O filtro de empresa é uma promessa, não uma garantia.** O histórico do projeto
   registra seis levas de correção de isolamento; a sexta ainda mapeia contas a
   pagar/receber, fluxo de caixa e ferramentaria como globais. Uma consulta sem
   `WHERE enterprise_id` mistura dois CNPJs — e num ERP fiscal isso é um problema
   contábil, não um bug de tela.
2. **Backup e restauração viram por empresa.** Restaurar a Tecnofer para ontem não
   pode significar restaurar a Usimac para ontem.
3. **Uma credencial vazada alcança só uma empresa.** O role de cada cliente não tem
   `CONNECT` na base do outro — conferido no provisionamento, não suposto.

O ambiente de treinamento é a exceção proposital: ele empresta a autenticação da
produção (`IDENTITY_DATABASE_URL`) para que a mesma pessoa entre nos dois com a
mesma senha. Uma empresa cliente **nunca** define essa variável.

## O mapa

| Empresa | Perfil do compose | Porta | Database | Subdomínio |
|---|---|---|---|---|
| Tecnofer | (sempre ativo) | 5070 | `venturerp` | `api.venturerp.com` |
| Usimac | `usimac` | 5075 | `venturerp_usimac` | `usimac.api.venturerp.com` |
| Truckparts (grupo Tecnofer) | `truckparts` | 5076 | `venturerp_truckparts` | `truckparts.api.venturerp.com` |
| Treinamento | `training` | 5077 | `venturerp_training` | `treino.api.venturerp.com` |

Portal de instalação e catálogo de empresas: `app.venturerp.com`
(`deploy/production/nginx/app.venturerp.com.conf`).

## Como o aplicativo acha a empresa certa

O instalador é **um só**. O endereço da API não é mais fixado no build: o app lê o
domínio do e-mail digitado no login e consulta o catálogo publicado em
`https://app.venturerp.com/tenants.json`.

```
compras@usimacusinagem.com.br
        ↓ domínio usimacusinagem.com.br
   tenants.json → https://usimac.api.venturerp.com
        ↓
   POST /users/login nesse endereço
```

Detalhes que importam:

- **Domínio fora do catálogo cai em `defaultApiUrl`**, hoje a Tecnofer. É o que
  mantém funcionando quem já usava o sistema antes desta mudança, sem depender de
  o domínio da Tecnofer estar listado.
- O catálogo é validado antes de ser usado: só `https`, só host, sem caminho nem
  credencial. Um catálogo adulterado não consegue apontar o app para um servidor
  que colete senhas.
- Sem rede vale a cópia guardada no disco e, na falta dela, a lista embutida no
  app — a tela de login nunca depende do catálogo estar no ar.
- `VITE_API_URL` preenchido **vence tudo**. É assim que demonstração e treinamento
  continuam apontando para o servidor deles.
- Cada empresa tem `JWT_SECRET` próprio. Com o segredo compartilhado, um token
  emitido pela API de uma empresa seria aceito pela API da outra — a validação é
  da assinatura, e o `enterprise_id` do token viajaria junto.

Código: `src/services/tenantDirectory.ts` e `src/services/httpClient.ts` no
repositório do frontend. Testes: `npm run test:tenant-directory`.

### Grupo de empresas (Tecnofer + Truckparts)

A Truckparts (CNPJ 64.815.197/0001-08, Simples Nacional) é do mesmo dono da
Tecnofer (Lucro Real), mas é **outra empresa**: base, usuários, senhas e
`JWT_SECRET` próprios. Ser do mesmo grupo não abre nada entre as duas — só muda o
login.

No catálogo, `group` reúne as empresas do grupo e `defaultTenantId` diz de quem é
o `defaultApiUrl`:

```json
"defaultTenantId": "tecnofer",
"tenants": [
  { "id": "tecnofer",   "apiUrl": "https://api.venturerp.com",            "emailDomains": ["tecnofer.com.br"],   "group": "tecnofer" },
  { "id": "truckparts", "apiUrl": "https://truckparts.api.venturerp.com", "emailDomains": ["truckpartsir.com.br"], "group": "tecnofer" },
  { "id": "usimac",     "apiUrl": "https://usimac.api.venturerp.com",     "emailDomains": ["usimacusinagem.com.br"] }
]
```

- E-mail que cai numa empresa de grupo com **mais de uma** empresa mostra o
  seletor "Entrar em" (Tecnofer | Truckparts). Começa na última empresa escolhida
  naquela máquina **com o mesmo domínio de e-mail** (desktop ≥ 1.19.12), ou na
  dona do domínio. É por domínio porque a mesma pessoa entra na Tecnofer com
  @tecnofer.com.br e na Truckparts com @truckpartsir.com.br.
- Quem cai no padrão (domínio fora do catálogo, ex.: Gmail) cai na Tecnofer e
  também vê o seletor: o padrão sempre foi a Tecnofer.
- Usimac não tem `group`: não aparece seletor, e uma escolha gravada de outra
  empresa é ignorada. A escolha só vale **dentro** do grupo do e-mail.
- Implantação (20/11/2026): a Truckparts nasceu com os mesmos 6 usuários da
  Tecnofer (mesmo e-mail @tecnofer.com.br, mesmo papel e o hash da senha copiado).
  A pessoa digita o e-mail de sempre, escolhe "Truckparts" e usa a senha de
  sempre. Daí em diante as senhas são independentes: trocar numa empresa não
  troca na outra. Em 21/11/2026 Rulian (ADMIN) e Karin passaram, só na base da
  Truckparts, para comercial@ e financeiro@truckpartsir.com.br, com a senha
  padrão combinada. Os outros 4 seguem com @tecnofer.com.br até terem e-mail; o
  ADMIN da Truckparts troca no cadastro quando chegarem.
- A empresa nasce com **id = código** (Truckparts: 1/1, como Tecnofer e Usimac).
  O `seed-tenant` garante isso mesmo depois de um `-dry-run`, que gasta a
  sequência. Com id ≠ código, um filtro que compara `enterprise_code` com
  `tenant.ID` só falharia em produção. Em 20/11/2026 a Truckparts tinha nascido
  como id 2 e foi recriada como id 1, com backup em
  `/var/backups/venturerp/manual/pre-id1-truckparts-*.dump`.
- `defaultTenantId` só vale se o tenant tiver o mesmo `apiUrl` do padrão. Um
  catálogo incoerente não desvia o padrão para outro servidor.
- O catálogo fica guardado por 24 h. Domínio desconhecido força uma nova busca, uma
  vez por sessão, para que o primeiro usuário de uma empresa recém-publicada não
  caia no servidor padrão.
- A troca de senha da tela de login vai para a empresa escolhida, não para o
  último servidor usado na máquina.
- Para trocar de empresa: sair e entrar escolhendo a outra. Cada sessão fala com
  um servidor só.

## Entrando com um cliente novo

Ordem importa: banco antes de tráfego, tráfego antes de login.

### 1. Banco, role, migrations e configuração

```bash
# na VPS, a partir do checkout do backend
sudo ./deploy/production/provision-tenant.sh <cliente> --senha '<senha-forte>'
```

O script é idempotente e faz, em ordem: cria o role e o database; fecha o database
novo para `PUBLIC`; **blinda as bases que já existiam** concedendo `CONNECT`
nominal a todo role que hoje consegue abrir cada uma, antes de remover o de
`PUBLIC`; prova o isolamento nos dois sentidos; aplica as migrations; gera o
`.env.<cliente>` com `JWT_SECRET` e `METRICS_TOKEN` próprios; e registra o cliente
em `/etc/venturerp/update.env`.

> A blindagem preserva o status quo de propósito. Critérios mais estreitos — ser
> dono, ter `CREATE` — deixariam de fora justamente o role da aplicação, que tem
> privilégio de objeto e nenhum privilégio de banco, e trancariam um ambiente que
> estava no ar.

Conferir sem alterar nada:

```bash
sudo ./deploy/production/provision-tenant.sh <cliente> --conferir
```

### 2. Empresa e usuários

Uma base recém-migrada não tem empresa nem usuário, e `POST /users/register` exige
um token `ADMIN` — que não existe ainda. O seeder resolve esse ovo-e-galinha:

```bash
go run ./cmd/seed-tenant \
  -database-url "postgres://venturerp_<cliente>:<senha>@127.0.0.1:5432/venturerp_<cliente>?sslmode=disable" \
  -enterprise-code 1 \
  -enterprise-name "NOME ABREVIADO" \
  -user "NOME COMPLETO|email@empresa.com.br|ADMIN" \
  -fiscal-cnpj "00.000.000/0000-00" -fiscal-razao-social "RAZÃO SOCIAL LTDA" \
  -fiscal-uf SP -fiscal-logradouro "Rua" -fiscal-numero "1" -fiscal-bairro "Centro" \
  -fiscal-municipio "Cidade" -fiscal-codigo-municipio "<IBGE>" -fiscal-cep "00000-000" \
  -fiscal-regime 1 -fiscal-ie "<IE>" -fiscal-nome-fantasia "FANTASIA" \
  -fiscal-complemento "<compl.>" -fiscal-telefone "<DDD+número>"
```

`-fiscal-regime` grava o código que a VEMP0100 aceita: `1` Simples, `2` Presumido
(padrão) ou `3` Real. Por extenso (`simples_nacional`, `lucro_real`) também serve.
O texto por extenso, gravado como estava, travava o primeiro salvamento da tela.

As senhas temporárias aparecem **uma vez** na saída. A configuração fiscal nasce em
`homologacao` e sem token: emissão real só depois do certificado A1 e da
homologação do cliente. Os demais usuários podem ser criados pela própria tela,
pelo `ADMIN` semeado.

### 3. DNS, nginx e certificado

`venturerp.com` tem um curinga `*.venturerp.com` apontando para a Vercel, então o
subdomínio precisa de um registro **A específico** para trazê-lo para a VPS.

```
<cliente>.api.venturerp.com   A   <IP da VPS>
```

Depois:

```bash
sudo cp deploy/production/nginx/usimac.api.venturerp.com.conf \
        /etc/nginx/sites-available/venturerp-<cliente>   # ajuste subdomínio e porta
sudo ln -s /etc/nginx/sites-available/venturerp-<cliente> /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
sudo certbot --nginx -d <cliente>.api.venturerp.com
```

### 4. Subir a API

```bash
cd /opt/venturerp/updater
COMPOSE_PROFILES=<cliente> VENTURERP_IMAGE=<imagem-em-uso> \
  VENTURERP_API_ENV=/opt/venturerp/panossoerp/.env \
  VENTURERP_<CLIENTE>_API_ENV=/opt/venturerp/panossoerp/.env.<cliente> \
  docker compose -f compose.yml up -d api-<cliente>
curl -fsS http://127.0.0.1:<porta>/health/ready
```

Os limites de memória e CPU do `compose.yml` existem para que um cliente não
consuma a VPS inteira e derrube o outro. Ajuste ao crescer o hardware; não remova.

### 5. Catálogo e portal

No repositório do frontend, acrescente o cliente em `portal/tenants.json` e
publique:

```bash
./scripts/publish-portal.sh --somente-catalogo
```

`publish-portal.sh` roda os testes do catálogo antes de enviar: um `apiUrl` que o
app recusaria faria o cliente cair no servidor da outra empresa sem aviso.

### 6. Monitoramento

```bash
printf '%s' "<METRICS_TOKEN do .env do cliente>" \
  > observability/prometheus/tokens/<cliente>
docker compose -f observability/docker-compose.yml up -d prometheus
```

Confirme que o alvo ficou `up` em `http://127.0.0.1:9090/api/v1/targets`. O rótulo
`tenant` é o que separa carga, latência e erro por empresa nos painéis.

## Atualização da plataforma

`scripts/self-update.sh` percorre a lista de empresas: backup de cada base,
migrations em cada base, health-check de cada API, e rollback que restaura
**todas**. Um cliente entra nesse ciclo só por configuração:

```ini
VENTURERP_TENANTS=usimac                    # lista separada por espaços
TENANT_USIMAC_DATABASE_URL=postgres://...
TENANT_USIMAC_API_ENV_FILE=/opt/venturerp/panossoerp/.env.usimac
TENANT_USIMAC_HEALTH_URL=http://127.0.0.1:5075/health/ready
```

O modelo completo está em `deploy/production/update.env.example`.

Duas empresas apontando para a mesma base fazem o `self-update` abortar **antes**
do primeiro backup — senão o rollback restauraria o dump errado sobre dados de
produção.

Quem dispara a atualização é o ambiente principal. Nos contêineres das demais
empresas a fila é montada só-leitura, e o pedido responde `403` explicando isso em
vez de estourar um erro de escrita.

## Backup

`deploy/production/backup-databases.sh` **descobre** as bases em vez de nomeá-las:
toda base que não seja de sistema entra. Um script que nomeia protege quem foi
lembrado e deixa o cliente novo sem backup até o dia em que alguém precisa
restaurar. Cada base tem dump próprio, verificação com `pg_restore --list`,
`sha256` e retenção aplicada por base. Uma base que falha não interrompe as demais,
e a execução termina com código diferente de zero para que o timer do systemd
mostre o problema.

## Capacidade da VPS com duas empresas

Medido em 27/09/2026 na VPS atual (3 vCPU, 3,7 GB, 75 GB): **1,2 GB em uso, 2,3 GB
disponíveis**. Há folga, mas ela estava mal distribuída.

### O gargalo real era o pool de conexões, não memória

O padrão do pgxpool é `max(4, NumCPU)`. Com 3 vCPU isso dava **quatro conexões por
API** — para a empresa inteira — enquanto o PostgreSQL oferece `max_connections=100`
com 6 em uso. Contagem de CPU é má medida aqui: o trabalho de um ERP é espera de
I/O, não processador, e um relatório longo segurando uma das quatro conexões
enfileira as telas de todos.

Agora o teto é explícito por empresa, e a soma cabe no limite compartilhado:

| Ambiente | `DB_MAX_CONNS` |
|---|---|
| Tecnofer | 25 |
| Usimac | 15 |
| Truckparts | 10 |
| Treinamento | 8 |
| **Total** | **58 de 100** |

`DB_MIN_CONNS=2` mantém conexões quentes (a primeira tela depois de um período
ocioso não paga abertura e autenticação). `DB_MAX_CONN_IDLE_MIN=15` é menor que os
30 minutos do driver: numa máquina pequena, devolver memória de conexão ociosa mais
cedo importa mais que economizar reconexões. **Ao somar um cliente, reveja a conta.**

### A observabilidade consumia mais que todo o resto

Medição por contêiner: tempo 470 MB, alloy 120 MB, prometheus 135 MB, grafana
85 MB, loki 74 MB, otel-collector 55 MB, exporters 36 MB — cerca de **975 MB**,
mais que as APIs e o PostgreSQL somados. Sem teto, um pico de retenção do Tempo
disputaria memória com as duas empresas.

Os oito serviços passaram a ter `mem_limit` e `cpus` proporcionais ao medido, com
folga. São tetos, não reservas: o uso real segue igual, e o que muda é que nenhum
deles consegue mais derrubar a VPS.

### PostgreSQL já estava bem ajustado

`shared_buffers=512MB`, `effective_cache_size=2GB`, `work_mem=4MB`,
`statement_timeout=2min`, `idle_in_transaction_session_timeout=5min` e
`pg_stat_statements` carregado. Não mexi. Duas observações:

- `log_connections`/`log_disconnections` estão em `on`. Com pools mantendo conexões
  quentes e duas empresas, isso vira ruído de log e I/O sem valor diagnóstico.
  Considere desligar.
- Há um segundo PostgreSQL (desenvolvimento, porta 5433) e uma API de
  desenvolvimento nativa ocupando a **porta 5071**. Por isso o treino (perfil
  `training`) roda na **5077** desde a v1.19.13, e a Usimac ficou em 5075.

## Estado em produção (28/09/2026)

A Usimac está **no ar**. O que foi executado na VPS, em ordem:

| Item | Estado |
|---|---|
| Registro DNS `usimac.api.venturerp.com` → 5.78.210.160 | criado na Vercel |
| Base `venturerp_usimac` + role, migrations até 369 | criados |
| Isolamento conferido nos dois sentidos | ✓ |
| Empresa + 4 usuários ADMIN + configuração fiscal | semeados |
| nginx `venturerp-usimac` → 127.0.0.1:5075 | ativo |
| Certificado Let's Encrypt | emitido, expira 27/12/2026 |
| Contêiner `venturerp-api-usimac` | `Up (healthy)` |
| Backup diário cobrindo as DUAS bases | timer apontado para o script novo |
| Alvo `venturerp-api-usimac` no Prometheus | `up`, com rótulo `tenant` |
| Tetos de memória na observabilidade | aplicados |
| Portal `app.venturerp.com` | no ar (Vercel) |

Login real de um usuário da Usimac pela internet devolve token com
`enterprise_id=1` e vê só `USIMAC USINAGEM TATUI`. O mesmo token na API da
Tecnofer devolve **401**, e a senha da Usimac no login da Tecnofer também.

### Duas armadilhas encontradas ao executar

1. **O checkout da VPS tem migrations defasadas** (até 238), porque o atualizador
   aplica as migrations **da imagem**, não do checkout. Para provisionar um cliente
   use `MIGRATIONS_DIR=/var/lib/venturerp-update/migrations`, que é o que a última
   release deixou.
2. **O firewall libera as portas de API por interface do Docker, uma por uma.** A
   5075 não estava liberada, e o alvo do Prometheus dava *timeout* sem explicar o
   motivo. Ao subir um cliente novo:
   ```bash
   ufw allow in on docker0 to any port <porta> proto tcp comment "Observability scrape <cliente>"
   ufw allow in on br-+   to any port <porta> proto tcp comment "Observability scrape <cliente>"
   ```
   A porta continua fechada para a internet — o acesso externo passa pelo nginx.

Também: o arquivo de token do Prometheus precisa pertencer ao usuário do contêiner
(`nobody`, uid 65534), senão o scrape falha com *permission denied*.

## Pendências conhecidas

- **Falta reiniciar a API da Tecnofer** para as métricas dela passarem a ser
  coletadas. O `METRICS_TOKEN` já foi gerado e gravado nos dois lados (`.env` da
  produção e `observability/prometheus/tokens/tecnofer`), e o job do Prometheus já
  está configurado com autenticação — mas a API só lê o `.env` ao subir. Um
  reinício interrompe os usuários por alguns segundos, então fica para uma janela
  escolhida por quem opera:
  ```bash
  cd /opt/venturerp/updater && COMPOSE_PROFILES=usimac \
    VENTURERP_IMAGE=$(cat /var/lib/venturerp-update/deployed-image) \
    VENTURERP_API_ENV=/opt/venturerp/panossoerp/.env \
    VENTURERP_USIMAC_API_ENV=/opt/venturerp/panossoerp/.env.usimac \
    docker compose -f compose.yml up -d api
  ```
- O domínio de e-mail da Tecnofer ainda não está em `portal/tenants.json`. Funciona
  pelo `defaultApiUrl`, mas listar explicitamente deixa o encaminhamento óbvio e
  libera o `defaultApiUrl` para outro uso. **Falta informar qual é o domínio.**
- O script de backup que roda hoje na VPS (`/opt/venturerp/database/backup.sh`)
  não era versionado e cobre uma base só. Substituir pelo versionado.
