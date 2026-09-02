# Evidências — ciclo comercial enterprise

Data da validação: 2026-08-31.

## Barreiras automatizadas

- `bash scripts/test-comercial-todas-fases.sh`: aprovado. A suíte executou as
  fases comerciais focadas e `go test ./...`.
- `make ci`: aprovado após limpeza do cache Go descartável em `/tmp`; inclui
  `fmt-check`, `go vet ./...`, build da API e `go test -coverprofile=coverage.out ./...`.
- Cobertura global observada na execução: 15,9% das instruções.
- Migrações `000317` a `000324`: histórico completo aplicado do zero no banco
  temporário da tarefa. A migration `000324` também teve rollback e reaplicação
  aprovados na base `venturerp_task_commercial_test_20260830`.
- Integrações autenticadas e multiempresa focadas: configuração fiscal e marca
  corporativa, RMA, razão de comissão, recorrência, orçamento, representante,
  cliente, assistência técnica e isolamento de precificação aprovados.
- Suíte HTTP autenticada executada contra API e PostgreSQL temporários em
  `http://127.0.0.1:5080`: todas as fases comerciais e o smoke de recorrência
  concluíram com código de saída zero.

## Limitação conhecida da automação

- A última busca documental de `scripts/test-comercial-todas-fases.sh` inclui
  `SESSION_SUMMARY.md`, que não existe neste worktree. O `rg` retorna código 2 e
  essa etapa não distingue arquivo ausente de ausência de correspondências. As
  referências a SAP e Oracle em `docs/dev/decisoes-enterprise-ciclo-comercial.md`
  são intencionais e registram a pesquisa exigida pela tarefa; não representam
  contratos legados copiados para o VentureERP.

## Cenários comprovados

- Código de item textual e compatibilidade temporária com JSON numérico.
- Resolução de preço/tabela por linha e validações de domínio.
- Divisão de venda com isolamento por empresa e conflito de vínculo.
- Pedido e orçamento com ator do JWT, filtros determinísticos e histórico.
- Detalhe de representante com coleções estáveis e validação de vínculos.
- Prévia de reprogramação e persistência transacional do planejamento.
- Anexos do SAC persistidos no servidor, sem caminho local confiado ao cliente.
- RMA vinculado ao chamado, itens, transições e eventos imutáveis.
- Configuração fiscal e cabeçalho corporativo obtidos pelo tenant autenticado.
- Comissão provisionada e estornada uma única vez pelo evento de competência.
- Comissão conciliada e paga com transição, referência, ator e evento idempotente.
- Cancelamento recorrente atômico com data efetiva, política futura e auditoria.
- Geração recorrente idempotente por competência e chave de idempotência.
- Evidência de RMA enviada, pesquisada e baixada com isolamento entre empresas.
- Pedido e orçamento recebem `item_code` textual e resolvem a chave interna no
  tenant, mantendo compatibilidade numérica temporária.
- Catálogo plano de classificações e filtro de carteira por workflow.
- Exportação rejeita configuração empresarial incompleta e não produz
  documento com cabeçalho silenciosamente vazio.

## Ambiente da evidência

Os smokes HTTP foram executados em uma base PostgreSQL isolada, criada a partir
do histórico integral de migrations e populada somente com dados descartáveis da
tarefa. Naquela primeira validação nenhuma API ou banco persistente foi alterado.
O banco temporário foi mantido para permitir reprodução das evidências; a API
temporária foi encerrada após a validação. A atualização operacional posterior
está registrada na seção seguinte.

## Atualização dos ambientes locais

Em 2026-08-31, após autorização operacional explícita, a imagem construída a
partir de `fix/ajustes-operacionais` foi instalada sem release/tag/commit nos
ambientes locais `5070`, demo (`5072`) e training (`5073`). Antes das alterações
foram gerados e verificados backups lógicos dos três bancos em `/tmp`.

- `5070`: migration inicial `232`; foram reparados retrofits históricos ausentes
  das migrations `199`, `206`, `214`, `216`, `220` e `231`, e aplicadas as
  migrations pendentes até `324`. O banco terminou em `324|false`.
- Demo e training: migration inicial `319`; migrations `320` a `324` aplicadas,
  terminando em `324|false`.
- Os três containers ficaram `healthy` e responderam `database=up` em readiness,
  usando o mesmo image ID local.
- Demo e training passaram `scripts/test-comercial-todas-fases.sh` autenticado.
- O `5070` passou smoke autenticado focal com o item real `50001`; a suíte
  completa pressupõe o dataset demo e falha corretamente para o item inexistente
  `10001`, portanto não foi usado seed artificial nesse banco.
- As contas técnicas criadas para smoke em `5070` e demo foram desativadas ao
  final. Nenhum token ou segredo foi registrado nos documentos ou logs de
  evidência.
