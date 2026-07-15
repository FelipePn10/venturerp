# AGENTS.md — Raiz do Projeto VentureERP

Este arquivo central descreve a localização e o propósito de cada `AGENTS.md` distribuído pelos módulos do projeto. Cada módulo possui seu próprio `AGENTS.md` com instruções específicas para IAs sobre como modificar, estender e compreender aquele módulo.

## Leitura de documentação

Documentos extensos de negócio ficam em `docs/domain/`.

A IA não deve carregar diretórios completos de documentação no contexto.
Deve ler somente os documentos e seções referenciados pela tarefa atual.

Caso a tarefa não indique a seção necessária, primeiro localize-a por busca
textual e abra apenas o trecho relevante.

A documentação funcional é fonte de requisitos, mas não autoriza implementar
funcionalidades fora do escopo definido em `.ai/tasks/`.

## Índice de AGENTS.md por Módulo

| Módulo | Caminho | Conteúdo |
|--------|---------|----------|
| **api/** | `api/AGENTS.md` | Ponto de entrada da aplicação: `main.go` (bootstrap de config, DB, tracing) e `api.go` (montagem de rotas e handlers HTTP). Descreve como iniciar o servidor, adicionar novos grupos de rotas e configurar middlewares. |
| **internal/** | `internal/AGENTS.md` | Visão geral da arquitetura Clean Architecture do sistema. Explica as 5 camadas, fluxo de dependências e convenções globais. |
| **internal/domain/** | `internal/domain/AGENTS.md` | Camada de domínio: entidades, value objects, serviços de domínio e interfaces de repositório. Explica como criar um novo domínio, modelar entidades e definir contratos. |
| **internal/application/** | `internal/application/AGENTS.md` | Camada de aplicação: use cases, DTOs, ports e security. Explica como criar um novo use case, definir DTOs de request/response e orquestrar domínios. |
| **internal/infrastructure/** | `internal/infrastructure/AGENTS.md` | Camada de infraestrutura: repositórios SQLC, conexão com banco, autenticação JWT, logger, tracing, notificações, exportação, integrações externas (FocusNFe, CNPJ, CNAB). Descreve como implementar repositórios, configurar infra e adicionar integrações. |
| **internal/interfaces/** | `internal/interfaces/AGENTS.md` | Camada de adaptadores: handlers HTTP (chi), middlewares e helpers de contexto. Explica como criar endpoints REST, handlers e middlewares. |
| **internal/pkg/** | `internal/pkg/AGENTS.md` | Pacotes compartilhados: utilitários de data/hora e validação. Descreve quando e como adicionar novos pacotes utilitários. |
| **cmd/** | `cmd/AGENTS.md` | Ferramentas CLI auxiliares (ex: `cutting-samples`). Explica como criar novos comandos CLI reutilizando a lógica de domínio. |
| **migrations/** | `migrations/AGENTS.md` | Migrações de banco de dados PostgreSQL com `golang-migrate`. Explica a convenção de nomenclatura, como criar novas migrações (up/down) e como executá-las. |
| **scripts/** | `scripts/AGENTS.md` | Scripts de automação: testes E2E, backup/restore, seeding de dados demo, load testing. Descreve o padrão dos scripts de teste e como adicionar novos. |
| **docs/** | `docs/AGENTS.md` | Documentação do projeto dividida em `apresentacao/` (negócio) e `dev/` (técnica). Explica como manter a documentação atualizada ao alterar funcionalidades. |
| **docker/** | `docker/AGENTS.md` | Configurações Docker: Dockerfile multi-stage, docker-compose para dev/test/demo/observability. Explica como modificar a stack Docker e adicionar novos serviços. |
| **observability/** | `observability/AGENTS.md` | Stack de observabilidade: OpenTelemetry Collector, Grafana, Prometheus, Tempo, Loki, Alloy. Descreve como configurar tracing, métricas, logs e dashboards. |

## Como usar este arquivo

Antes de modificar qualquer parte do sistema, a IA deve:
1. Ler este `AGENTS.md` raiz para entender a estrutura geral
2. Ler o `AGENTS.md` do módulo que será alterado para entender regras e convenções específicas
3. Ler também os `AGENTS.md` dos módulos adjacentes quando a alteração cruzar camadas

## Convenções Globais

- **Linguagem:** Go 1.25+
- **Framework HTTP:** chi v5
- **Banco:** PostgreSQL 16 com pgx
- **Geração de SQL:** sqlc (arquivos `.sql` em `internal/infrastructure/database/queries/`)
- **Migrações:** golang-migrate v4
- **Autenticação:** JWT (golang-jwt/jwt/v5)
- **Módulo Go:** `github.com/FelipePn10/panossoerp`
- **Dependências:** Vendorizadas em `vendor/`
- **Porta padrão:** 5070
- **Arquitetura:** Clean Architecture (Domain → Application → Infrastructure → Interfaces)

## Protocolo obrigatório de execução

Antes de alterar código:

1. Leia este arquivo.
2. Leia todos os AGENTS.md entre a raiz e o diretório do arquivo alterado.
3. Leia a tarefa em `.ai/tasks/`.
4. Leia somente as seções funcionais referenciadas pela tarefa.
5. Localize implementações similares.
6. Execute os testes atuais do pacote.
7. Apresente um plano e os arquivos candidatos.

Durante a implementação:

1. Mantenha a alteração dentro do escopo.
2. Não faça refatorações oportunistas não solicitadas.
3. Não altere contratos compartilhados silenciosamente.
4. Não edite arquivos gerados.
5. Crie commits pequenos e coerentes quando solicitado.
6. Atualize documentação somente quando o comportamento mudar.

Após a implementação:

1. Formate os arquivos modificados.
2. Execute testes focados.
3. Execute testes de regressão proporcionais ao impacto.
4. Analise o diff final.
5. Apresente arquivos, decisões, testes, riscos e pendências.

## Regras críticas

- Toda operação deve preservar isolamento por empresa/tenant.
- Queries não podem omitir o identificador da empresa quando aplicável.
- Regras de negócio não devem residir em handlers ou repositories.
- Operações compostas devem avaliar necessidade de transação.
- Não alterar migrations já aplicadas; criar nova migration.
- Não editar código gerado pelo sqlc manualmente.
- Não modificar arquivos de produção, segredos ou infraestrutura sem escopo explícito.
- Não silenciar erros para fazer testes passarem.
- Não usar float para valores monetários ou quantidades que exijam precisão.
- Não utilizar `context.Background()` dentro de fluxos que já recebem contexto.

## Paralelização

Subagentes podem ser usados para:

- exploração;
- localização de contratos;
- análise de testes;
- revisão;
- investigação de logs;
- análise de migrations.

Não delegar escrita paralela quando dois agentes precisarem:

- editar o mesmo arquivo;
- alterar a mesma interface;
- modificar a mesma migration;
- redefinir o mesmo contrato;
- implementar partes com ordem obrigatória.

Antes de paralelizar, classifique cada subtarefa como:

- PARALLEL_SAFE;
- PARALLEL_READ_ONLY;
- SEQUENTIAL_DEPENDENCY;
- SAME_FILES_DO_NOT_PARALLELIZE.

## Entrega obrigatória

A resposta final deve incluir:

1. resumo;
2. arquivos modificados;
3. critérios de aceite;
4. decisões técnicas;
5. testes executados e resultados;
6. riscos residuais;
7. itens não executados;
8. confirmação de que não houve alterações fora do escopo.

## Releases e atualizações

- Não use `git pull` em produção. Código publicável chega como imagem imutável do GHCR.
- O único gesto de release do backend é `make release VERSION=X.Y.Z`, executado na `main` limpa.
- A versão vem da tag e é injetada no build; nunca fixe uma versão em `internal/version/version.go`.
- Tags `v*` acionam pipelines. Commits em `develop` ou `main` não atualizam clientes.
- A API não acessa Docker. `POST /api/system/update` cria uma solicitação; o systemd do host executa `scripts/self-update.sh` com `flock`.
- Preserve backup verificado, migrations, readiness e rollback. Consulte `docs/dev/releases-e-atualizacoes.md`.
