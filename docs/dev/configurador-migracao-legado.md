# Configurador — Remoção do `questions` legado (CONCLUÍDA)

O configurador antigo (`questions` / `question_options` / `item_questions` /
`item_mask_answers` / `item_question_answers`) foi **totalmente removido** e substituído
pelo novo configurador `cfg_*` (ver [`configurador-produto.md`](configurador-produto.md),
migrations 000199–000203). A resolução de estrutura/BOM passou a ser **cfg-only**.

Migration de remoção: `000204_drop_legacy_questions`.

---

## O que foi removido

**Rotas / superfície HTTP**
- `POST/GET /api/questions/**` (create, find, options, associate) — **removido**.
- `POST /api/items/mask/generate` (geração de máscara legada) — **removido**.

**Código**
- Handlers: `create_question_handler`, `find_question_by_name_handler`,
  `create_question_option_handler`, `delete_question_option`, `associate_question_handler`,
  `delete_question_handler`, `generate_mask_handler` + os tipos `QuestionHandler`,
  `QuestionOptionHandler`, `AssociateByQuestionItemHandler`, `GenerateMaskHandler`.
- Use cases: `question_uc`, `question_option_uc`, `generate_mask_uc`.
- Domínio: `domain/questions`, `domain/questions_options`, `domain/associate_questions`.
- Repositórios: `repository/questions`, `repository/questions_options`,
  `repository/item_question`, `repository/generate_mask`.
- sqlc: `questions.sql.go`, `questions_options.sql.go`, `item_questions.sql.go`,
  `generate_mask.sql.go` + as queries legadas de `structure.sql.go`/`structure_query.sql.go`
  (`GetItemQuestions`, `GetMaskAnswersByItemAndValue`, `GetMaskAnswersWithNames`,
  `GetItemMaskAnswersByValue`).
- DTOs legados de request (`associate_item_questions_request`, `generate_mask_product`) e
  o handler agregado morto (`handler/new.go` reduzido).
- Importador transitório (`migrate-legacy-questions`) e o `materialize_structure_variant.go`
  (que já estava comentado/inativo).

**Banco** (migration `000204`)
```sql
DROP TABLE IF EXISTS item_question_answers CASCADE;
DROP TABLE IF EXISTS item_mask_answers    CASCADE;
DROP TABLE IF EXISTS item_questions       CASCADE;
DROP TABLE IF EXISTS question_options     CASCADE;
DROP TABLE IF EXISTS questions            CASCADE;
```
> `item_masks` foi **mantida** — é usada pelo `cfg_*` (string da máscara + hash,
> consumida por estrutura/venda/MRP).

---

## Resolução de estrutura/BOM — agora cfg-only

`StructureQueryRepositorySQLC` (repositório do resolver ativo
`structure_query/service/resolver.go`) resolve a configuração **exclusivamente** pelo
`cfg_*`, mapeando `characteristic_id → QuestionID` e `variable_id → OptionID`:

| Método | Fonte cfg |
|---|---|
| `GetItemQuestions` | `cfg_item_characteristics` (sequence → Position) |
| `GetMaskAnswersByItemAndValue` | `cfg_item_mask_answers` (respostas com variável) |
| `GetMaskAnswersWithNames` | idem, por código da característica (fórmulas) |
| `CreateMaskForItem` | grava `item_masks` + `cfg_item_mask_answers` |

A propagação de máscara pela árvore de BOM continua casando por `QuestionID` — que agora é
o `characteristic_id` (compartilhado entre pai/filho). Itens configuráveis **precisam**
estar cadastrados no `cfg_*` (`/api/configurator`).

---

## Migração de dados (deployments com dados legados)

Este repositório removeu o importador; para ambientes com dados legados reais, a migração
`questions → cfg_*` deve ser executada **antes** de aplicar a `000204` (o importador
idempotente está no histórico do git — `migrate-legacy-questions`). Passos:

1. Rodar o importador (questions→características, options→variáveis, item_questions→
   características do item, item_mask_answers→`cfg_item_mask_answers`).
2. Confirmar que todos os itens configurados têm `cfg_item_characteristics`.
3. Aplicar a migration `000204` (drop).

---

## Validação

`go build`/`go vet`/`gofmt` limpos; **suíte unitária e de integração completas verdes**
com as tabelas legadas já removidas do banco de teste (incl. BOM/co-produto, custo,
produção, fiscal). `scripts/test-e2e.sh` §45 migrado de `/api/questions` para
`/api/configurator`.
