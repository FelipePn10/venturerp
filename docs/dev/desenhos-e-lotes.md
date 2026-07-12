# Cadastro de Desenhos + Máscara de Lotes/Séries — Documentação técnica (Configurador Fase 3)

Dois cadastros do Configurador de Produto: **Desenhos** (com revisões) e **Máscara de
Lotes/Séries** (geração automática de código de lote). Ambos destravam tipos de
característica do configurador — `DESENHO` e `SEQUENCIAL`.

Migrations: `000200_drawings_lot_masks`, `000226_drawings_tenant_scope` e
`000227_drawing_maintenance_and_replication`. Versão de negócio em
[`../apresentacao/configurador.md`](../apresentacao/configurador.md).

> Convenções: `Authorization: Bearer <JWT>`, papel `ADMIN`/`USER`. `created_by` vem do JWT.

---

## 1. Cadastro de Desenhos (`/api/drawings`)

Um **desenho** (cabeçalho: código+dígito+formato) evolui por **revisões**. O código de
replicação é `Desenho(20 primeiras posições) + Dígito + Formato + Revisão` (campo
`composite_code` na resposta da revisão).

| Método | Rota | Ação |
|---|---|---|
| POST/GET | `/api/drawings` · `/api/drawings/{id}` | CRUD do desenho (+`only_active`, `q`) |
| PUT/DELETE | `/api/drawings/{id}` | Atualizar / inativar |
| POST/GET | `/api/drawings/{id}/revisions` | Adicionar / listar revisões |
| PUT/DELETE | `/api/drawings/revisions/{revId}` | Atualizar / remover revisão |
| POST | `/api/drawings/revisions/{revId}/distributions` | Distribuição (Botão Distribuição) |
| DELETE | `/api/drawings/distributions/{distId}` | Remover distribuição |
| POST/GET | `/api/drawings/{id}/characteristics` | Vincular / listar características do configurador |
| DELETE | `/api/drawings/characteristics/{charLinkId}` | Remover vínculo |
| PUT | `/api/drawings/item-code` | Manter código do item/configuração |
| GET | `/api/drawings/item-code/{itemCode}?mask=...` | Consultar código de engenharia |
| GET/PUT | `/api/drawings/manufacturing-parameters` | Consultar/alterar parâmetro 8 |

Campos do desenho: `code`, `digit`, `format`, `model`, `item_code`, `description`, `uom`,
`weight`, `material_spec` (E.M.), `creation_date`. Da revisão: `revision`, `start_date`,
`end_date` (vigência), `material_spec`, `reason` (Motivo), `approved_by`/`approval_date`
(Aprovação), `is_current`. Marcar `is_current` numa revisão desmarca as demais.

Tabelas: `drawings`, `drawing_revisions`, `drawing_revision_distributions`,
`drawing_characteristics` (liga a `cfg_characteristics`/`cfg_variables` com operador).

Todas as operações validam `drawings.enterprise_id` contra a empresa do JWT,
inclusive revisões, distribuições e características. Desenhos legados sem
empresa inferível permanecem em quarentena.

### Código de engenharia e parâmetro 8

`PUT /api/drawings/item-code` recebe `item_code`, `drawing_code` e `mask`
opcional. Item simples exige máscara vazia. Se o item possuir configurações em
`item_masks`, uma máscara existente é obrigatória e o código é gravado somente
para aquela configuração em `item_engineering_drawings`.

O parâmetro 8 desta rotina fica em `manufacturing_item_parameters`, sem colisão
com o parâmetro 8 do planejamento. Quando habilitado, cadastrar ou alterar uma
revisão corrente replica o código composto somente onde o código ainda é
exatamente o da revisão anterior. A primeira revisão nunca replica e deve ser
informada manualmente. A troca da revisão corrente e a replicação são atômicas.

---

## 2. Cadastro de Máscara de Lotes/Séries (`/api/lot-masks`)

Uma **máscara** é um template de geração de código de lote, resolvido por contexto
(cliente/item/classificação/aplicação) e composto por **partes ordenadas**.

| Método | Rota | Ação |
|---|---|---|
| POST/GET | `/api/lot-masks` · `/api/lot-masks/{id}` | CRUD da máscara (+`only_active`) |
| PUT/DELETE | `/api/lot-masks/{id}` | Atualizar / inativar |
| POST | `/api/lot-masks/{id}/parts` | Adicionar partição |
| PUT/DELETE | `/api/lot-masks/parts/{partId}` | Atualizar / remover partição |
| POST | `/api/lot-masks/generate` | Gerar um código de lote |

**Tipos de partição** (`part_type`): `CARACTER` (texto fixo, ajustado ao `size` —
completa com espaços à direita ou trunca), `DATA` (data atual formatada por `date_format`
com tokens `DD/MM/YYYY/YY/HH/MI/SS`), `SEQ_NUMERICA` (sequência numérica incrementada,
preenchida com zeros à esquerda até `size`), `SEQ_CARACTER` (sequência alfabética
A→B→…→Z→AA). Máximo de 20 caracteres no código gerado.

**Estado da sequência:** cada partição de sequência guarda `current_value` (último valor
gerado) e `last_year`. `zero_on_year_change` reinicia a sequência no valor inicial (`value`)
quando o ano vira. A primeira geração usa o `value` (valor inicial); as seguintes
incrementam.

**Geração** (`POST /generate`):
```json
{ "lot_mask_id": 5 }                                  // explícito
{ "application": "SUPRIMENTOS", "item_code": 4444 }   // por contexto
```
Resolve a máscara mais específica ativa (cliente+item > item > cliente > classificação >
aplicação), monta o código pelas partes em ordem de `sequence`, persiste o novo estado das
sequências e retorna `{ lot_mask_id, code }`.

Tabelas: `lot_masks`, `lot_mask_parts` (com `current_value`/`last_year`).

---

## 3. Camadas / arquivos

| Camada | Arquivo |
|---|---|
| Migration | `migrations/000200`, `000226` e `000227` |
| Domínio | `internal/domain/drawing/entity/` · `internal/domain/lot_mask/entity/` (gerador puro `Generate`) |
| SQL (hand-written) | `internal/infrastructure/database/sqlc/drawings.sql.go` · `lot_masks.sql.go` |
| Use cases | `internal/application/usecase/drawing_uc/` · `lot_mask_uc/` |
| DTOs | `internal/application/dto/request|response/{drawing,lot_mask}_*.go` |
| Handlers | `internal/interfaces/http/handler/{drawing,lot_mask}_handler.go` |
| Rotas | `api/api.go` (`/api/drawings`, `/api/lot-masks`) |
| Testes | `lot_mask/entity/entity_test.go` (unit gerador) + integração (`drawing_uc`, `lot_mask_uc`, tag `integration`) |
