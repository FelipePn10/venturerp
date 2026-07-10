# Configurador de Produto — Documentação técnica (Fase 1)

O Configurador de Produto permite descrever itens configuráveis por **características**
(perguntas) respondidas por **variáveis** (respostas) agrupadas em **conjuntos**, e a
partir das respostas montar a **máscara** do item configurado. Esta é a **Fase 1**
(fundação): Conjuntos/Variáveis, Características (com tipos), Características do Item e a
geração de máscara. As demais telas (Restrições/Dependências — refino, Desenhos, Regras
de Variáveis Equivalentes, Regras de Itens Configurados, Tipos de Descrição, Máscara de
Lotes/Séries) vêm em fases seguintes.

> Arquitetura: camada **nova e rica** (`cfg_*`), paralela ao antigo `questions`, com
> **ponte** para `item_masks` — a tabela que estrutura/venda/MRP já consomem. Nada do
> fluxo legado foi quebrado.

Versão de negócio em [`../apresentacao/configurador.md`](../apresentacao/configurador.md).
Restrições/Dependências já existentes: `/api/restriction` (ver `restriction` domain).

> Convenções: `Authorization: Bearer <JWT>`, papel `ADMIN`/`USER`. `created_by` vem do JWT.

---

## 1. Modelo de dados (migration `000199_configurator_core`)

| Tabela | Papel |
|---|---|
| `cfg_sets` | Conjunto (Manutenção de Conjuntos) — agrupa variáveis. |
| `cfg_variables` | Variável (Manutenção de Variáveis): código, descrição, **composição de máscara**, ativo, especial, inclui-desc, dados-esp, marketing. `UNIQUE(set_id, code)`. |
| `cfg_variable_languages` | Idiomas da variável (tradução + país). |
| `cfg_characteristics` | Característica (Manutenção de Características): código único, descrição (pergunta), **tipo**, conjunto, variável default, máscara (visualização), especial, afeta-preço, controla-metas, tipo-recebimento, e campos por tipo. |
| `cfg_characteristic_languages` | Idiomas da característica (descrição/máscara por idioma). |
| `cfg_item_characteristics` | Característica do Item: sequência (10 em 10), resposta default, **característica pai**, flags Esp./Des./Carga, fórmula. `UNIQUE(item_code, sequence)`. |
| `cfg_item_char_default_answers` | Respostas Default múltiplas (ESCOLHA_MULT). |
| `cfg_item_mask_answers` | Respostas ricas de uma máscara gerada (rastreabilidade sobre `item_masks`). |

**Tipos de característica** (`char_type`): `ESCOLHA`, `ESCOLHA_MULT`, `FORMULA`,
`DESENHO`, `INF_CARACTER`, `INF_NUMERICA`, `OPCAO`, `CAMPO`, `SEQUENCIAL`.
**Tipo Recebimento**: `NENHUM`, `RECEBIMENTO`, `VINCULO`, `RECEBIMENTO_VINCULO`.
**Campo (CAMPO)**: `ITEM_CODE`, `CUSTOMER_CODE`, `ORDER_CODE`, `SEQUENTIAL`.

---

## 2. Endpoints (`/api/configurator`)

### Conjuntos e Variáveis

| Método | Rota | Ação |
|---|---|---|
| POST/GET | `/sets` · `/sets/{id}` | CRUD de conjunto (+`only_active`) |
| PUT/DELETE | `/sets/{id}` | Atualizar / inativar |
| POST/GET | `/sets/{id}/variables` | Criar / listar variáveis do conjunto |
| GET/PUT/DELETE | `/variables/{varId}` | Detalhar / atualizar / inativar variável |
| POST | `/variables/{varId}/languages` | Upsert idioma da variável |
| DELETE | `/variables/languages/{langId}` | Remover idioma |

### Características

| Método | Rota | Ação |
|---|---|---|
| POST/GET | `/characteristics` | Criar / listar (+`only_active`) |
| GET/PUT/DELETE | `/characteristics/{id}` | Detalhar / atualizar / inativar |
| POST | `/characteristics/{id}/languages` | Upsert idioma |
| DELETE | `/characteristics/languages/{langId}` | Remover idioma |
| GET | `/characteristics/{id}/items` | **Itens Vinculados** — itens que usam a característica |

Validações por tipo: `ESCOLHA`/`ESCOLHA_MULT` exigem conjunto; `OPCAO` recebe rótulos
sim/não (default `SIM`/`NAO`); `INF_NUMERICA` valida `num_max ≥ num_min`; `CAMPO` exige
`field_source`; a variável default deve pertencer ao conjunto da característica.

### Características do Item

| Método | Rota | Ação |
|---|---|---|
| POST/GET | `/items/{itemCode}/characteristics` | Vincular / listar (ordenado por seq.) |
| PUT/DELETE | `/item-characteristics/{id}` | Atualizar / remover |

**Guarda:** a **sequência** só pode mudar e a característica só pode ser removida
enquanto o item **não tiver máscara gerada** (`item_masks`) **nem fórmula no cadastro de
estruturas** (`item_structures.loss_formula`). As demais informações (flags, default, pai,
fórmula) podem ser alteradas a qualquer momento. A resposta default do item **sobrepõe** a
da característica. A **fórmula é obrigatória** ao vincular uma característica do tipo
`FORMULA` ao item (o cálculo depende de dados do item).

### Geração de máscara

`POST /api/configurator/generate-mask`
```json
{
  "item_code": 4444,
  "persist": true,
  "answers": [
    { "characteristic_id": 10, "variable_id": 7 },
    { "characteristic_id": 11, "value": "50" }
  ]
}
```
Monta a máscara ordenando as respostas pela **sequência** da característica do item e
juntando com `#` (idêntico ao value object legado — mesmo hash sha256 de 8 dígitos).
Resolução por tipo:

- `ESCOLHA` → composição de máscara da variável escolhida (ou a default do item);
- `ESCOLHA_MULT` → composições das variáveis selecionadas juntadas por `+` (ou as respostas default);
- `INF_CARACTER` → valor livre (obrigatório quando `is_required`);
- `INF_NUMERICA` → valida faixa e múltiplo;
- `OPCAO` → rótulo sim/não;
- `DESENHO`/`CAMPO`/`SEQUENCIAL`/`FORMULA` → valor informado.

Com `persist: true`, grava em `item_masks` (visível a estrutura/venda/MRP) e as respostas
ricas em `cfg_item_mask_answers`.

### Geração em lote — produto cartesiano (Fase 2)

`POST /api/configurator/generate-masks` — gera **todas as combinações válidas** das
características do tipo `ESCOLHA` do item (Geração de Máscara para Itens Configurados).
```json
{
  "item_code": 4444,
  "restrict": [ { "characteristic_id": 3, "variable_ids": [7, 8] } ],
  "customer_code": null,
  "division_id": null,
  "persist": true
}
```
- É **obrigatório** restringir ao menos uma característica (`restrict`) para reduzir o
  volume; o produto é limitado a 20.000 combinações.
- Faz o produto cartesiano das variáveis ativas de cada característica `ESCOLHA` (fixadas
  em `restrict` ou todas as ativas do conjunto).
- Aplica **Restrições/Dependências** reaproveitando o engine existente
  (`/api/restriction`) como oráculo: cada combinação vira `characteristic_id → código da
  variável` e é validada por `EvaluateCombination`. A restrição de maior peso cujos
  **dominantes** batem age como dependência; a combinação é válida se todos os
  **determinantes** forem satisfeitos (`INVALID` proíbe; `EQUAL`/`DIFFERENT`/`BELONGS`/
  `NOT_BELONGS`/`GREATER`/`LESS`). Retorna total, válidas, persistidas e a lista de máscaras.

> **Ponte de restrições:** as tabelas `restriction_dominants/determinants` usam
> `question_id` como `BIGINT` sem FK — no configurador novo, `question_id` = id da
> característica e `answer_value` = **código da variável**. Assim a Manutenção de
> Restrições/Dependências (`/api/restriction`) já serve o configurador sem duplicação.

### Tipos de Descrição + Descrição de Itens Configurados (Fase 4)

Permite descrever a máscara de um item configurado de formas diferentes por destino
(programa/relatório/LOV). Migration `000201`.

**Tipos de Descrição** (`/api/configurator/description-types`) — CRUD de `code`,
`description`, `kind` (PROGRAMA/RELATORIO/LOV/GERAL).

**Descrição de Itens Configurados** (`/api/configurator/…`):

| Método | Rota | Ação |
|---|---|---|
| POST | `/item-descriptions` | Cria (idempotente) a descrição de um item p/ um tipo e **carrega uma linha por característica do item** |
| GET | `/items/{itemCode}/descriptions` · `/item-descriptions/{id}` | Listar por item / detalhar |
| PUT | `/item-descriptions/{id}/lines` | Atualiza a grade (Ord., Carac., Masc., Tipo Desc., Txt, Queb. Lin.) |
| POST | `/item-descriptions/{id}/reload` | Recarrega a grade das características atuais do item |
| POST | `/item-descriptions/{id}/render` | **Botão V**: renderiza a descrição para um conjunto de respostas |
| DELETE | `/item-descriptions/{id}` | Remove |

Cada linha da grade (`cfg_item_description_lines`) configura, por característica: `Ord.`
(order_index), `Carac.` (mostra a resposta), `Masc.` (mostra o rótulo), `Tipo Desc.`
(`DESCRICAO` = descrição da característica · `COMP_MASCARA` = campo máscara), `Txt` (texto
após o rótulo) e `Queb. Lin.` (quebra de linha). O **render** percorre as linhas em ordem
e monta cada segmento como `rótulo + texto + resposta`, separando por espaço ou quebra de
linha. Ex.: `Cor da tampa: AZUL`.

### Regras de Variáveis Equivalentes + Regras de Itens Configurados (Fase 5)

Migration `000202`. Dois motores de regra que derivam configuração/campos das respostas.

**Regras de Variáveis Equivalentes** (`/api/configurator/equivalent-rules`) — mapeiam a
configuração do item **pai** para a do item **filho** na estrutura. Uma regra tem uma
condição no pai (característica + operador + variável) e um alvo no filho (característica +
operador + variável + fórmula opcional).

| Método | Rota | Ação |
|---|---|---|
| POST/PUT/DELETE | `/equivalent-rules` · `/equivalent-rules/{id}` | CRUD (delete = inativa) |
| GET | `/parents/{parentItemCode}/equivalent-rules` · `/equivalent-rules/{id}` | Listar por pai / detalhar |
| POST | `/equivalent-rules/apply` | Dada a config do pai, retorna as respostas equivalentes do filho |

**Regras de Itens Configurados** (`/api/configurator/item-rules`) — quando a configuração
de um item satisfaz **todas** as condições (característica + operador + variável, em AND),
define o valor de um **campo** de uma **pasta** (tabela) do Cadastro de Item.

| Método | Rota | Ação |
|---|---|---|
| POST/PUT/DELETE | `/item-rules` · `/item-rules/{id}` | CRUD (cabeçalho + condições) |
| GET | `/items/{itemCode}/rules` · `/item-rules/{id}` | Listar por item / detalhar |
| POST | `/item-rules/evaluate` | Dada a config, retorna as atribuições `{tabela, campo, conteúdo}` das regras que dispararam |

Ex. (Focco): quando a característica *Opção* do item = *Sim*, na pasta *Dados da
Engenharia* o campo *Percentual de Perda* = 66%. As respostas são canonizadas pelo
**código da variável** (mesma convenção das restrições), e os operadores
(`EQUAL/DIFFERENT/GREATER/LESS/BELONGS/NOT_BELONGS`) reusam `entity.MatchOperator`.

---

## 3. Camadas / arquivos

| Camada | Arquivo |
|---|---|
| Migration | `migrations/000199_configurator_core.{up,down}.sql` |
| Domínio | `internal/domain/configurator/entity/entity.go` (entidades, tipos, `BuildMask`) |
| SQL (hand-written) | `internal/infrastructure/database/sqlc/configurator.sql.go` |
| Use cases | `internal/application/usecase/configurator_uc/` (sets/variables, characteristic, item_characteristic, mask, cartesian, description, **rules**) |
| Oráculo de restrições | `internal/application/usecase/restriction_uc/evaluate_combination.go` (`EvaluateCombination`) |
| Regras (SQL) | `internal/infrastructure/database/sqlc/configurator_rules.sql.go` · migration `000202` |
| DTOs | `internal/application/dto/request|response/configurator_*.go` |
| Handler | `internal/interfaces/http/handler/configurator_handler.go` |
| Rotas | `api/api.go` (`/api/configurator`) |
| Testes | `entity/entity_test.go` + `restriction_uc/evaluate_combination_test.go` (unit) + `configurator_uc/configurator_integration_test.go` (E2E flow + cartesiano, tag `integration`) |

---

## 4. Próximas fases (backlog)

- ✅ **Fase 1** — Conjuntos/Variáveis, Características, Características do Item, geração de máscara.
- ✅ **Fase 2** — Restrições/Dependências (via ponte para `/api/restriction`) + **Geração de
  Máscara para Itens Configurados** (produto cartesiano filtrado por restrições).
- ✅ **Fase 3** — **Cadastro de Desenhos** (com revisões) e **Cadastro de Máscara de
  Lotes/Séries** (ver [`desenhos-e-lotes.md`](desenhos-e-lotes.md)).
- ✅ **Fase 4** — **Tipos de Descrição** + **Descrição de Itens Configurados** (render da máscara).
- ✅ **Fase 5** — **Regras de Variáveis Equivalentes** (pai→filho) + **Regras de Itens
  Configurados** (config → campo da pasta do item).

> **Núcleo do Configurador de Produto do Focco: COMPLETO.** Todos os 11 programas do
> spec estão cobertos (Conjuntos/Variáveis, Características, Características do Item, Tipos
> de Descrição, Descrição de Itens Configurados, Restrições/Dependências, Geração de
> Máscara, Regras de Variáveis Equivalentes, Regras de Itens Configurados, Desenhos,
> Máscara de Lotes/Séries).

### `questions` legado — REMOVIDO

O configurador antigo (`questions` / `question_options` / `item_questions` /
`item_mask_answers`) foi **totalmente removido** (rotas, código, tabelas — migration
`000204`) e substituído pelo `cfg_*`. A resolução de estrutura/BOM é **cfg-only**. Detalhes
e passos de migração de dados legados em
[`configurador-migracao-legado.md`](configurador-migracao-legado.md). `item_masks` foi
mantida (usada pelo `cfg_*`).

### Botão Itens (Tipo Recebimento) — implementado

Para características com `receiving_type` ∈ `RECEBIMENTO`/`VINCULO`/`RECEBIMENTO_VINCULO`
(empresa encarroçadora, Parâmetro 44), a sub-tabela `cfg_characteristic_receiving_items`
informa quais **respostas** (variáveis), **itens** ou **classificações** são usadas por
tipo de recebimento (migration `000203`):

| Método | Rota | Ação |
|---|---|---|
| POST/GET | `/characteristics/{id}/receiving-items` | Adicionar / listar |
| DELETE | `/characteristics/receiving-items/{recvId}` | Remover |

### Limitações conhecidas (nicho — documentadas, não implementadas)

1. **Replicação do desenho (Parâmetro 8)**: a alteração de revisão não é replicada
   automaticamente para a pasta Engenharia do Cadastro de Item. O `composite_code` da
   revisão corrente é exposto para a integração manual. *(Não há tabela de parâmetros do
   sistema para gatilhar o Param 8, nem campo de destino inequívoco na pasta Engenharia —
   requer decisão de produto.)*
2. **7 parâmetros de Formação do Lote por empresa** (Suprimentos): a máscara de lote usa a
   sequência do próprio cadastro; os parâmetros por empresa não sobrepõem o cadastro.
   *(Depende de um backbone de parâmetros por empresa/módulo inexistente hoje.)*

### Avaliação de fórmula (Botão F) — implementado

Reusa `internal/domain/structure/formula/evaluator.go`. As fórmulas referenciam outras
características pelo **código normalizado** (maiúsculas; não-alfanumérico → `_`; ex.:
`COR LAM EXT` → `COR_LAM_EXT`). O valor numérico de cada resposta vem da composição de
máscara/valor (via `ParseOptionValue`).

- **Característica FORMULA** na geração de máscara: a resposta é **calculada** a partir das
  características de **sequências anteriores** (as variáveis vão sendo acumuladas na ordem).
  Ex.: `AREA = LARGURA*2`, com `LARGURA=5`, gera a máscara `5#10`.
- **Regra de Item** com fórmula (Botão F): o **conteúdo** da atribuição é o resultado da
  fórmula. Ex.: `peso = QTD*10`, com `QTD=5`, atribui `50`.

Pendente (não-núcleo):

1. Fechar as limitações de nicho acima conforme demanda das fábricas.

Concluído: remoção total do `questions` legado (código + tabelas, migration `000204`);
resolução de estrutura/BOM cfg-only. Ver
[`configurador-migracao-legado.md`](configurador-migracao-legado.md).
