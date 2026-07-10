# Ficha de Produção da Ferramenta — Documentação técnica

A **Ficha de Produção da Ferramenta** define qual **série (instância física)** de cada
ferramenta será utilizada na execução de cada operação de uma ordem de produção (OF).
É o elo entre o cadastro de ferramentas (com vida útil) e o chão de fábrica: ao concluir
a operação, a vida útil é debitada tanto no **mestre da ferramenta** quanto na **série**
efetivamente vinculada, garantindo rastreabilidade por peça física.

Versão de negócio em [`../apresentacao/producao.md`](../apresentacao/producao.md).
Ferramentas e roteiro em [`maquinas-e-roteiro.md`](maquinas-e-roteiro.md); operações da
OF em [`producao.md`](producao.md) §2.

> Convenções: `Authorization: Bearer <JWT>`, papel `ADMIN`/`USER`. `created_by`/`assigned_by`
> vêm do JWT (não precisam ser enviados no corpo).

---

## 1. Modelo de dados (migration `000198_tool_serials`)

| Tabela | Papel |
|---|---|
| `tool_serials` | Instâncias físicas (nº de série) de um mestre de ferramenta. Vida útil (`life_used`) e status próprios por série. `UNIQUE (tool_id, serial_number)`. |
| `production_order_operation_tool_serials` | Vínculo ativo série ↔ (operação da OF, ferramenta). `UNIQUE (production_order_operation_id, tool_id)` — 1 série por par. Reatribuir o mesmo par **atualiza** a linha (UPSERT). |
| `tool_serial_substitutions` | Trilha de auditoria de cada substituição (série antiga → nova, motivo, quem, quando). |

`status` da série ∈ `ATIVA` · `MANUTENCAO` · `INATIVA` · `BAIXADA`. Apenas séries
`ATIVA` e ativas (`is_active`) podem ser vinculadas a uma operação.

---

## 2. Cadastro de Ferramentas (mestre) — `/api/routing/tools`

O **mestre de ferramentas** (matrizes, gabaritos, dispositivos, ferramentas de corte)
guarda a vida útil consumida a cada produção. Migration `000176_tools`.

| Método | Rota | Ação |
|---|---|---|
| POST | `/api/routing/tools` | Cria a ferramenta (código gerado automaticamente) |
| GET | `/api/routing/tools?only_active=true` | Lista ferramentas |
| GET | `/api/routing/tools/replacement` | Ferramentas que atingiram o limite de vida útil |
| GET/PUT/DELETE | `/api/routing/tools/{id}` | Consultar / atualizar / inativar |
| POST | `/api/routing/tools/{id}/reset-life` | Zerar a vida útil após a troca física |

Campo a campo (corpo de criação):

| Campo | Descrição |
|---|---|
| `code` | **Gerado automaticamente** (não enviar) — próximo código sequencial |
| `name` | Nome/descrição da ferramenta (obrigatório) |
| `tool_type` | Tipo livre (default `FERRAMENTA`; ex.: `MATRIZ`, `GABARITO`) |
| `life_type` | Unidade da vida útil: `GOLPES` · `HORAS` · `PECAS` (default `PECAS`) |
| `life_limit` | Limite de vida útil; `0` = sem controle de vida |
| `cost` | Custo da ferramenta |
| `status` | `ATIVA` · `MANUTENCAO` · `INATIVA` (default `ATIVA`) |

```json
{ "name": "Matriz Corte 200", "tool_type": "MATRIZ", "life_type": "GOLPES", "life_limit": 100000, "cost": 5000 }
```

A ferramenta é vinculada às operações do **roteiro** (`route_operation_tools`, ver
[`maquinas-e-roteiro.md`](maquinas-e-roteiro.md)); a vida útil é debitada no apontamento
da operação (ver §6).

---

## 3. Cadastro de séries (mestre da ferramenta)

Sob `/api/routing/tools` (onde vive o mestre de ferramentas):

| Método | Rota | Ação |
|---|---|---|
| POST | `/{id}/serials` | Cadastra uma série da ferramenta `{id}` |
| GET | `/{id}/serials?only_active=true` | Lista séries da ferramenta |
| GET | `/serials/{serialId}` | Detalha uma série |
| PUT | `/serials/{serialId}` | Atualiza série (nº, status, localização, notas) |
| DELETE | `/serials/{serialId}` | Inativa a série (`is_active=false`, status `INATIVA`) |

Corpo de criação:
```json
{ "serial_number": "SN-A", "status": "ATIVA", "location": "Almox A", "notes": "" }
```

---

## 4. Ficha de produção (`/api/tool-production-sheet`)

| Método | Rota | Ação |
|---|---|---|
| GET | `/orders?q=<busca>` | Lista de valores das ordens elegíveis (**exclui tipo OFC**). `q` filtra por nº da OF, código ou descrição do item. |
| GET | `/{orderId}` | Ficha completa: cabeçalho + operações + ferramentas + séries. Também serve ao botão **Atualiza**. |
| POST | `/assign` | Vincula uma série a uma operação/ferramenta |
| POST | `/substitute` | Substitui a série já vinculada (registra auditoria) |
| GET | `/substitutions?operation_id=&tool_id=` | Histórico de substituições |

### Cabeçalho (campo a campo)

| Campo | Origem |
|---|---|
| Empresa | `enterprise` acessada (`enterprise_id`/`enterprise_name`) |
| Ordem | `production_orders.order_number` (`order_id` = PK) |
| Tipo | tipo da ordem planejada mapeado: `PRODUCTION`/manual → **OF**; `OUTSOURCING` → **OFC** (excluído da LOV); demais → valor bruto. `type_raw` traz o valor original |
| Dt. Início / Dt. Fim | `start_date` / `end_date` |
| Quantidade | `planned_qty` |
| Cód. Item | `item_code` (+ `item_name` = descrição técnica do PDM) |
| Configurado | `mask` (configuração do item) |

### Bloco de operações

Cada operação lista suas ferramentas (herdadas do roteiro via `route_operation_tools`).
Uma linha por par (operação, ferramenta):

| Campo do prompt | Campo JSON |
|---|---|
| Seq. | `sequence` |
| Operação (código) | `operation_code` |
| Operação (descrição) | `operation_name` / `operation_description` |
| Recurso (código) | `resource_code` (centro de trabalho) |
| Recurso (descrição) | `resource_name` |
| Ferramenta | `tool_code` + `tool_name` |
| Número de Série | `assigned_serial_*` (selecionada) e `available_serials[]` (LOV) |
| Substituir | habilitado quando `can_substitute = true` (só há função se já existe série vinculada) |

---

## 5. Vincular / Substituir

**Assign** (`POST /assign`):
```json
{ "operation_id": 123, "tool_id": 45, "serial_id": 9 }
```
Valida: operação existe, série existe, série pertence à ferramenta e está disponível
(`ATIVA` + ativa). Faz UPSERT — reatribuir o mesmo par troca a série sem duplicar.

**Substitute** (`POST /substitute`):
```json
{ "operation_id": 123, "tool_id": 45, "new_serial_id": 10, "reason": "desgaste" }
```
Exige que já exista uma série vinculada (senão `422` — coerente com o botão Substituir
sem função quando não há série). Troca a série e grava um registro em
`tool_serial_substitutions`. A nova série deve ser diferente da atual.

Ambos retornam a **linha atualizada** (`SheetOperationToolResponse`) para o front
refrescar só a operação afetada, sem recarregar a ficha inteira.

---

## 6. Integração com o apontamento (vida útil)

Ao concluir uma operação (`POST /api/production-order/operations/advance` com
`status=DONE`), o hook `consumeToolLife` (`order_operations_uc.go`) debita a vida útil
de cada ferramenta do roteiro **e**, quando há série vinculada àquela operação/ferramenta,
debita o mesmo montante em `tool_serials.life_used`. Montante = peças produzidas
(`GOLPES`/`PECAS`) ou horas reais (`HORAS`). Isso mantém o desgaste por instância física
sincronizado com o mestre e alimenta a lista de troca (`GET /api/routing/tools/replacement`).

---

## 7. Camadas / arquivos

| Camada | Arquivo |
|---|---|
| Migration | `migrations/000198_tool_serials.{up,down}.sql` |
| SQL (hand-written sqlc) | `internal/infrastructure/database/sqlc/tool_serials.sql.go` |
| Entidade + repo | `internal/domain/tool/entity/entity.go` (`ToolSerial`), `.../tool/repository/repository.go`, `internal/infrastructure/repository/tool/tool_repository_sqlc.go` |
| Use cases | `internal/application/usecase/tool_uc/tool_uc.go` (CRUD de série), `internal/application/usecase/tool_sheet_uc/tool_sheet_uc.go` (ficha) |
| DTOs | `internal/application/dto/request/tool_sheet_request.go`, `.../response/tool_sheet_response.go` |
| Handlers | `internal/interfaces/http/handler/tool_handler.go` (séries), `.../tool_sheet_handler.go` (ficha) |
| Rotas | `api/api.go` (`/api/routing/tools/**/serials`, `/api/tool-production-sheet`) |
