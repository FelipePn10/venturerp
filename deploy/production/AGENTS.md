# AGENTS.md — Implantação de produção

Este diretório define a implantação versionada do backend. `compose.yml` executa somente a API e reutiliza o PostgreSQL privado do host. Nunca adicione o socket Docker ao contêiner nem coloque segredos nos arquivos versionados.

Releases chegam como `ghcr.io/felipepn10/panossoerp:vX.Y.Z`. O updater deve usar sempre a tag exata e manter backup, migrations da própria imagem, health-check e rollback. As unidades systemd separam a solicitação sem privilégios da execução root.

Antes de alterar estes arquivos, leia `docs/dev/releases-e-atualizacoes.md` e valide tanto sucesso quanto falha. A versão é injetada por build args/ldflags e nunca deve ser hard-coded.
