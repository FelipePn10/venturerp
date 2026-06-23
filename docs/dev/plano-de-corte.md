# Plano de Corte — Documentação técnica

Documento técnico do **módulo Plano de Corte** (`cutting_plan`), que otimiza o
aproveitamento de matéria-prima ao nestar peças demandadas no estoque disponível.
A versão de apresentação (linguagem de negócio) está em
[`../apresentacao/plano-de-corte.md`](../apresentacao/plano-de-corte.md).

> Convenções: `Authorization: Bearer <JWT>`, `Content-Type: application/json`.
> Todas as rotas exigem papel `ADMIN` ou `USER`.

> **Status:**
> - **Fase 1 (entregue):** corte **linear 1D** (barras, perfis, tubos) com estoque
>   heterogêneo, kerf, refile (trim) e sobra mínima. Otimizador nativo em Go, testado.
> - **Fase 2 (entregue):** **firmar (baixa real de estoque) + retalhos rastreáveis**.
>   Consumo por modo configurável (automático FIFO / manual por lote / padrão da
>   empresa), geração de retalho reaproveitável herdando corrida+certificado,
>   vínculo da baixa à OP e trilha de consumo. Migrations 000159–000160 (UoM).
> - **Fase 3 (entregue):** **corte 2D guilhotinado** (chapa / painel MDF) — nesting
>   de retângulos com veio, rotação, kerf e refile; baixa por área; retalhos 2D
>   rastreáveis. Migration 000161. Mesmo agregado/rotas, selecionado por `cut_type`.
> - **Fase 4 (entregue):** **corte true-shape** (irregular, laser/plasma) — peça com
>   contorno (polígono) + **integração híbrida**: provedor nativo de *bounding-box*
>   (funciona out-of-the-box, reusa o 2D) e **porta de provedor externo** (adapter
>   HTTP estilo DeepNest/ProNest) para o nesting irregular real. Migration 000162.
> - **Fase 5 (entregue):** **demanda automática de OP/ordens planejadas** — explode o
>   BOM das ordens, transforma cada componente cortado em peça e **agrega várias
>   ordens do mesmo material** num único plano. Sem migration (reusa o agregado).
> - **Fase de complementos (entregue):** export do mapa (SVG/DXF/PDF), programa/
>   sequenciamento de corte + agendamento na máquina, **nesting irregular nativo
>   shape-aware (raster)**, fita de borda, rateio de custo por OP, e limpeza de
>   nomenclatura. Migrations 000163–000165. Ver §3f e §9.

---

## 1. Princípio de arquitetura — estratégia plugável por tipo de corte

O otimizador é um **serviço de domínio puro** (sem persistência/HTTP), exposto por
uma única interface registrável por tipo de corte. Isso mantém o algoritmo
testável isoladamente e permite que 1D, 2D-guilhotina e true-shape coexistam.

```
internal/domain/cutting_plan/service/
  optimizer.go                  // interface CuttingOptimizer + tipos + registry por CutType
  optimizer_1d.go               // Best-Fit Decreasing linear (FASE 1)
  optimizer_2d_guillotine.go    // retângulos livres + corte guilhotina (FASE 3)
  nesting_trueshape.go          // provedor true-shape nativo (bounding-box) (FASE 4)
  uom.go                        // conversão comprimento/área → UoM de estoque
internal/infrastructure/nesting/
  http_provider.go              // adapter HTTP p/ engine externo de nesting (FASE 4)
```

```go
type CuttingOptimizer interface {
    Type() entity.CutType
    Optimize(demand []DemandPiece, stock []StockPiece, p CutParams) (*Solution, error)
}
```

Cada otimizador se auto-registra no `init()`. `service.Optimizer(cutType)` resolve a
estratégia; tipos sem implementação retornam `ErrNoOptimizer` (ex.: true-shape antes
do provider externo) — sem quebrar o resto do sistema.

---

## 2. Modelo do módulo

Um **plano corta UM item de matéria-prima** (`material_item_code`) em várias peças.
Materiais diferentes são planos diferentes. **Não há "barra padrão"**: o estoque é
heterogêneo, então cada peça de estoque carrega seu próprio comprimento.

| Entidade (`internal/domain/cutting_plan/entity`) | Papel |
|---|---|
| **CuttingPlan** | Cabeçalho: tipo de corte, origem, status, item de matéria-prima, parâmetros (kerf/trim/sobra mínima) e métricas do resultado. |
| **CuttingPlanPart** | Demanda: peça a cortar (comprimento × quantidade), opcionalmente ligada a um item/OP. |
| **CuttingStockPiece** | Estoque disponível para o plano: cada peça com seu comprimento próprio; `is_remnant` marca retalho. |
| **CuttingPattern** | Resultado: um layout de corte repetido N vezes (`repeat_count`), com aproveitamento, kerf e sobra. |
| **PatternPlacement** | Posição de cada peça ao longo da barra (`offset_mm`) — a instrução de chão-de-fábrica. |

**Status do plano:** `RASCUNHO → OTIMIZADO → FIRMADO → EM_EXECUCAO → CONCLUIDO`.
**Tipos de corte:** `LINEAR_1D`, `GUILLOTINE_2D`, `TRUE_SHAPE_2D` — **todos ativos**;
`cut_type` seleciona o otimizador no registry.
**Origem:** `MANUAL` (ativo); `ORDEM_PRODUCAO` / `ORDEM_PLANEJADA` ficam para a
demanda automática (roadmap §8).

---

## 3. Algoritmo 1D (`optimizer_1d.go`)

Heurística **Best-Fit Decreasing** com estoque heterogêneo:

1. A demanda é expandida em unidades e cortada em ordem **decrescente** de comprimento.
2. Cada peça tenta primeiro a barra **já aberta** que deixa a **menor folga** (best fit),
   empacotando peças menores na sobra de barras abertas.
3. Se não couber em nenhuma, abre-se uma nova peça de estoque, preferindo **menor
   `priority`** (retalhos antes de barras inteiras) e, no mesmo nível, o **maior
   comprimento** disponível (empacota mais por barra → menos barras).
4. **Kerf** é descontado entre peças consecutivas; **trim** é removido da cabeça de
   cada barra antes do primeiro corte. A sobra após o último corte é **retalho**,
   reaproveitável quando atinge `min_remnant_mm`.
5. Layouts idênticos (mesmo comprimento de barra + mesmo multiconjunto de cortes) são
   **agrupados** em um padrão com `repeat_count`.

Peças mais longas que qualquer estoque viram `Unplaced` (aviso ao operador).
É uma heurística rápida e aceitável no chão-de-fábrica; o contrato permite trocar por
um método exato (column generation / Gilmore-Gomory) sem alterar chamadores.

### Métricas calculadas
- **Aproveitamento** = demanda total / estoque consumido.
- **Refugo (scrap)** = estoque − demanda − **retalho reaproveitável** (sobra ≥ mínimo
  não conta como perda, pois a Fase 2 a devolve ao estoque).
- `stock_used_count`, `cut_count`, `total_demand_mm`, `total_stock_mm`.

---

## 3b. Fase 2 — firmar (baixa) + retalhos rastreáveis

**Firmar** (`POST /api/cutting-plans/{id}/release`) transforma o plano otimizado em
consumo real. Para cada peça de estoque que os padrões exigem:

1. **Resolve o modo de consumo** (override do plano → padrão da empresa):
   - `AUTOMATIC`: escolhe lotes por **FIFO** (corrida mais antiga primeiro), com
     fallback para baixa genérica ao custo médio quando não há lotes em saldo.
   - `MANUAL`: usa o **lote atribuído** na peça de estoque (obrigatório); corrida e
     certificado vêm do registro do lote.
   - A empresa define o padrão em `cutting_settings`; o plano pode sobrescrever em
     `lot_consumption_mode`. (É o "os dois, ou a empresa decide".)
2. **Baixa real:** posta `StockMovement` OUT do material (atualiza saldo + saldo por
   lote + custo médio via `stock.CreateMovement`), referenciando a **OP**
   (`production_order_code`) quando houver, senão o próprio plano.
3. **Retalho consumido:** se a peça era um retalho do inventário (`remnant_id`), o
   retalho é marcado `CONSUMED` — **sem** nova baixa (já saíra ao consumir a barra-mãe).
4. **Retalho gerado:** sobra do padrão ≥ `min_remnant_mm` vira um `StockRemnant`
   `AVAILABLE`, herdando lote/corrida/certificado da origem (rastreabilidade
   sobrevive ao recorte) e custo proporcional (`custo_barra × sobra/comprimento`).
5. **Trilha de consumo:** cada draw vira uma linha em `cutting_plan_consumptions`
   (lote ou retalho, qtd, custo, id do movimento). Plano passa a `FIRMADO`.

**Conversão de unidade de estoque (UoM):** o material pode ser estocado em metro,
m², m³, peça, kg, tonelada etc. — não só por peça. A baixa converte o comprimento
cortado (mm) para a UoM de estoque via `service.StockQtyForLength(uom, lengthMM, fator)`:

| UoM de estoque | Conversão (qtd por peça de comprimento L mm) |
|---|---|
| `UN` (peça) / vazio | **1** (cada barra/retalho é uma unidade, independe do comprimento) |
| `M`, `CM`, `MM`, `IN`, `MICROMETRO` | geométrica (ex: `M` = L/1000; `CM` = L/10) — **fator ignorado** |
| `KG`, `TONELADA`, `M2`, `M3` | `(L/1000) × fator`, onde **fator = quantidade de estoque por metro linear** (densidade kg/m, largura m²/m, seção m³/m) — **obrigatório > 0** |

A UoM de estoque é um **snapshot** copiado do item (`Warehouse.UnitOfMeasurement`) na
criação do plano (campo `stock_uom`), ou informada explicitamente; o `uom_factor`
cobre os casos de massa/área/volume. O custeio fica correto: a quantidade da baixa
sai na UoM de estoque e o custo é **por UoM** (o custo médio/lote já é por UoM), com
`total = qtd × custo`. O retalho gerado herda o custo **por UoM** (o tamanho é
implícito no comprimento). Migration 000160.

O valor do retalho fica em `stock_remnants` (registro de material recuperável) até
ser reusado.

**Reuso na otimização:** com `include_remnants=true` e `warehouse_id` no plano, o
`Optimize` semeia automaticamente os retalhos `AVAILABLE` do material como peças de
estoque (prioridade 0 → consumidos antes de barras inteiras). A semeadura é
idempotente (limpa as peças-retalho antigas a cada rodada).

> **Atomicidade:** as escritas do lado-corte (marcar retalho consumido, criar
> retalhos, gravar consumo, mudar status) são **uma transação** (`CommitRelease`);
> os movimentos de estoque são postados antes, cada um atômico. Firmar exige status
> `OTIMIZADO` e bloqueia re-firmar/re-otimizar um plano já `FIRMADO`.

## 3c. Algoritmo 2D guilhotinado (`optimizer_2d_guillotine.go`)

Nesting de retângulos em chapas, com **cortes guilhotina** (de ponta a ponta), via
heurística de **retângulos livres**:

1. As peças são posicionadas por **maior área primeiro**.
2. Cada peça ocupa o retângulo livre que **menos desperdiça área** (best-area-fit),
   testando a orientação **rotacionada só quando permitido** (`allow_rotation` e veio
   `NONE`) — peça com veio visível mantém a orientação.
3. Posicionar a peça **divide** o retângulo livre em dois filhos com um **corte
   guilhotina**, descontando o kerf, escolhendo o eixo que preserva o maior offcut.
4. Sem chapa aberta que sirva, abre-se uma nova (retalho primeiro por prioridade,
   depois a maior chapa), mantendo todo layout **compatível com seccionadora**.

**Dados 2D:** parte tem `width_mm`, `height_mm`, `grain` (NONE/LENGTH/WIDTH) e
`allow_rotation`; o estoque tem `width_mm`, `height_mm`; o padrão guarda
`stock_width_mm`/`stock_height_mm`, `used_area_mm2`, `remnant_area_mm2` e o **maior
retângulo de sobra** (`remnant_width_mm`/`remnant_height_mm`); cada posição guarda
`pos_x_mm`, `pos_y_mm`, `width_mm`, `height_mm`, `rotated` (instrução de chão-de-fábrica
+ desenho do layout). Uma linha 1D mantém `length_mm`; uma 2D usa width/height.

**Baixa 2D (firmar):** a quantidade sai por **área** via `StockQtyForArea(uom, w, h,
fator)` — `UN`→1 chapa; `M2`→área m²; `M3/KG/TONELADA`→área × fator (espessura m³/m²,
peso kg/m²). O retalho 2D gerado é o maior retângulo de sobra quando **ambos os lados**
≥ sobra mínima, herdando lote/corrida/certificado e custo por-UoM.

O mesmo `cut_type` no plano (`GUILLOTINE_2D`) seleciona o otimizador no registry; o
restante do fluxo (criar, otimizar, firmar, retalhos, settings) é idêntico ao 1D.

## 3d. Corte true-shape (irregular) — Fase 4

Para peças de **formato qualquer** (laser/plasma/router), o nesting irregular real
(no-fit-polygon) é caro de fazer bem. A entrega é **híbrida**:

- **Provedor nativo (`nesting_trueshape.go`):** envolve cada peça na sua **caixa
  envolvente** (bounding-box do polígono) e reusa o otimizador 2D guilhotinado.
  Funciona **out-of-the-box**, registrado no registry sob `TRUE_SHAPE_2D` — todo
  plano true-shape produz um resultado utilizável (rendimento menor que o irregular
  verdadeiro, mas imediato).
- **Provedor externo (`infrastructure/nesting/http_provider.go`):** quando a variável
  de ambiente **`NESTING_SERVICE_URL`** está definida, o ERP delega o nesting a um
  serviço externo (ex.: um microsserviço DeepNest/ProNest) via um **protocolo JSON
  documentado** (request: params+parts com polígono+sheets; response: sheets com
  placements x/y/width/height/`rotation_deg`+unplaced). Esse provedor implementa o
  **mesmo contrato `CuttingOptimizer`** e sobrescreve o nativo só para true-shape.

**Dados:** a peça true-shape guarda o contorno em `geometry` (JSON `[{x,y},…]`) e a
caixa envolvente em `width_mm/height_mm` (reusadas da fase 3); a posição ganha
`rotation_deg` (ângulo livre; o provedor nativo usa 0/90). Firmar trata true-shape
como **chapa** (baixa por área, retalhos retangulares), igual ao 2D. O detalhe traz
`pos_x/pos_y/width/height/rotation_deg` por peça para desenhar/exportar o mapa.

## 3e. Demanda automática de OP/MRP — Fase 5

`POST /api/cutting-plans/from-orders` gera os planos de corte a partir de ordens, sem
digitação. Para cada **Ordem de Produção** e/ou **ordem planejada (MRP)** informada:

1. **Explode o BOM** do produto da ordem (`StructureQueryRepository.GetDirectChildrenForMask`,
   respeitando a máscara da ordem).
2. **Identifica peças cortadas:** todo componente-filho com **dimensões** (`Engineering.
   Dimensions`) é uma peça a cortar; itens sem dimensão (ferragem, parafuso) são ignorados.
3. **Resolve a matéria-prima** do componente: o filho do componente marcado como
   **matéria-prima** (`Planning.LLC == 9`), senão o seu único filho, senão o
   `ItemBaseCod`. Sem material resolvível → entra em `warnings` (transparente).
4. **Geometria/quantidade:** 1D usa `Dimensions.Length`; 2D (material em `M2`/`M3`) usa
   `Length × Width` (Height = espessura). Quantidade = `qtd_da_ordem × qtd_BOM` (com
   perda), arredondada para cima (peças inteiras).
5. **Agrega por matéria-prima** *entre todas as ordens* → **um plano por material**
   (`source = ORDEM_PRODUCAO`), com cada peça carimbada com sua ordem (`source_ref`).
   O `cut_type` sai do material (1D/2D); a UoM de estoque é snapshot do item.
   `production_order_code` é preenchido só quando **uma única OP** alimentou o material
   (senão a baixa referencia o plano; a descrição lista as ordens).

O resultado é uma lista de **planos criados** (id, código, material, cut_type, nº de
peças, ordens) + `warnings`. O operador então adiciona o **estoque** (manual ou via
retalhos com `include_remnants`) e roda optimize/release normalmente.

> **Dependências:** o usecase (`DemandUseCase`) injeta os repositórios de OP, ordem
> planejada, estrutura (BOM), item e plano de corte. É a única parte do módulo que
> cruza vários domínios — orquestrada no use case, sem transação distribuída.

## 3f. Fase de complementos

Seis complementos sobre o módulo já entregue (sem novos tipos de corte):

1. **Export do mapa de corte** (`service/cutmap.go`) — desenha os padrões em **SVG**
   (visualização), **DXF** (LWPOLYLINE + TEXT, para CAM/seccionadora) e **PDF**
   (vetorial, A4), tudo dependency-free. `GET .../{id}/export?format=svg|dxf|pdf`.
2. **Programa de corte + agendamento** — `GET .../{id}/program` devolve a sequência
   ordenada de cortes por padrão (offset/posição de cada peça). `POST .../{id}/schedule`
   cria um `MachineSchedule` para a máquina do plano (`machine_code`), levando o corte
   ao calendário de máquina (CRP/APS).
3. **Nesting irregular nativo (raster, shape-aware)** (`service/nesting_raster.go`) —
   substitui o bounding-box por um nester com **grade de ocupação + bottom-left fill
   e rotações 90°**, que respeita o contorno real (peças se **intertravam** em
   concavidades). É o provedor registrado para `TRUE_SHAPE_2D` (dispatcher: usa raster
   quando há polígono; cai para bbox/guilhotina quando as peças são retângulos puros).
   A grade é limitada (≤120 células/eixo) para manter performance; o engine externo
   ainda sobrescreve para alto volume.
4. **Fita de borda (moveleiro)** — a peça 2D guarda quais lados levam fita
   (`edge_top/bottom/left/right`), o material (`band_item_code`) e o custo/m
   (`band_cost_per_m`). `BandingLengthMM()` calcula o perímetro encapado × qtd; o
   detalhe do plano traz `banding` (comprimento e custo totais). Migration 000164.
5. **Rateio de custo por OP** — ao firmar, o custo total da baixa é distribuído entre
   as ordens de origem das peças, proporcional à demanda de cada uma (comprimento 1D
   / área 2D), gravado em `cutting_plan_order_costs`. `GET .../{id}/order-costs`.
   Migration 000165.
6. **Limpeza de nomenclatura** — `total_demand_mm`/`total_stock_mm` →
   `total_demand`/`total_stock` (guardam comprimento OU área). Migration 000163.

## 4. Endpoints (`/api/cutting-plans`)

| Método | Rota | Ação |
|---|---|---|
| POST | `/api/cutting-plans` | Cria plano (pode já vir com `parts` e `stock_pieces` inline) |
| POST | `/api/cutting-plans/from-orders` | **Gera planos a partir de OP/ordens planejadas** (Fase 5) |
| GET | `/api/cutting-plans?only_open=true` | Lista planos (filtro de abertos: RASCUNHO/OTIMIZADO) |
| GET | `/api/cutting-plans/{id}` | Detalhe: cabeçalho + demanda + estoque + padrões |
| DELETE | `/api/cutting-plans/{id}` | Remove plano (cascata em peças/padrões) |
| POST | `/api/cutting-plans/{id}/optimize` | **Roda o otimizador**, grava os padrões e retorna o resultado (com `unplaced`) |
| POST | `/api/cutting-plans/{id}/release` | **Firma o plano**: baixa de estoque + retalhos + trilha de consumo (Fase 2) |
| GET | `/api/cutting-plans/{id}/export?format=svg\|dxf\|pdf` | Baixa o **mapa de corte** (vetorial) |
| GET | `/api/cutting-plans/{id}/program` | Programa ordenado de cortes |
| POST | `/api/cutting-plans/{id}/schedule` | Agenda o corte na máquina do plano |
| GET | `/api/cutting-plans/{id}/order-costs` | Rateio do custo da baixa por OP |
| POST | `/api/cutting-plans/{id}/parts` | Adiciona peça à demanda |
| DELETE | `/api/cutting-plans/{id}/parts/{partId}` | Remove peça |
| POST | `/api/cutting-plans/{id}/stock` | Adiciona peça de estoque |
| DELETE | `/api/cutting-plans/{id}/stock/{stockId}` | Remove peça de estoque |

### Exemplo — criar e otimizar
```http
POST /api/cutting-plans
{
  "material_item_code": 5001,
  "description": "Corte cantoneira 2pol",
  "kerf_mm": 3,
  "trim_mm": 0,
  "min_remnant_mm": 300,
  "stock_uom": "M",          // opcional: vazio = copia do item; aqui barra estocada em metros
  "uom_factor": 0,           // só para KG/M2/M3/TON (qtd por metro linear)
  "parts": [
    { "label": "Perna 720", "length_mm": 720, "quantity": 8 },
    { "label": "Travessa 1200", "length_mm": 1200, "quantity": 4 }
  ],
  "stock_pieces": [
    { "length_mm": 6000, "quantity": 5 },
    { "length_mm": 2300, "quantity": 1, "is_remnant": true }
  ],
  "created_by": "<uuid>"
}

POST /api/cutting-plans/{id}/optimize   → retorna patterns[] + métricas + unplaced[]
POST /api/cutting-plans/{id}/release    → baixa de estoque + retalhos; retorna resumo
```

### Endpoints auxiliares (Fase 2)
| Método | Rota | Ação |
|---|---|---|
| GET | `/api/cutting-settings` | Lê o padrão da empresa (modo de consumo, sobra mínima, depósito) |
| PUT | `/api/cutting-settings` | Atualiza o padrão (papel `ADMIN`) |
| GET | `/api/stock-remnants?item_code=&only_available=true` | Lista retalhos do material no inventário |

---

## 5. Persistência

- **Migrations:**
  - `000158_cutting_plan` — `cutting_plans`, `cutting_plan_parts`,
    `cutting_stock_pieces`, `cutting_patterns`, `cutting_pattern_placements`.
  - `000159_cutting_plan_release` (Fase 2) — `stock_remnants`,
    `cutting_plan_consumptions`, `cutting_settings` (singleton id=1); colunas novas
    em `cutting_plans` (warehouse_id, production_order_code, lot_consumption_mode,
    include_remnants, released_at) e `cutting_stock_pieces` (remnant_id, heat_number).
  - `000160_cutting_plan_uom` — colunas `stock_uom` + `uom_factor` em `cutting_plans`
    (unidade de estoque do material e fator de conversão).
  - `000161_cutting_plan_2d` (Fase 3) — colunas 2D: parts (width/height/grain/
    allow_rotation), stock_pieces (width/height), patterns (stock_width/height,
    used_area, remnant_area, remnant_width/height), placements (pos_x/pos_y/width/
    height/rotated), stock_remnants (width/height).
  - `000162_cutting_plan_trueshape` (Fase 4) — `geometry` (TEXT/JSON) em
    cutting_plan_parts e `rotation_deg` em cutting_pattern_placements.
  - `000163_cutting_plan_rename_metrics` — `total_demand_mm/total_stock_mm` →
    `total_demand/total_stock`.
  - `000164_cutting_plan_edge_banding` — fita de borda em cutting_plan_parts
    (edge_top/bottom/left/right, band_item_code, band_cost_per_m).
  - `000165_cutting_plan_order_costs` — tabela `cutting_plan_order_costs` (rateio por OP).
  - `000166_machine_schedule_nullable_order` — torna `machine_schedules.order_code`
    opcional (necessário p/ agendar plano de corte; corrige o módulo de agenda).
  - Enums como `VARCHAR + CHECK`.
- **Queries SQLC:** `internal/infrastructure/database/queries/cutting_plan.sql`.
- **Repositório:** `internal/infrastructure/repository/cutting_plan/cutting_plan_repository_sqlc.go`.
  `ReplacePatterns` apaga e regrava os padrões/posições **em uma transação** (resultado
  derivado de cada otimização).
- **Use case:** `internal/application/usecase/cutting_plan_uc/` — mapeia entidade ↔ serviço,
  roda o otimizador e atualiza as métricas do plano.
- **Handler:** `internal/interfaces/http/handler/cutting_plan_handler.go`.

---

## 6. Testes

- `go test ./internal/domain/cutting_plan/...` — **8 testes** do otimizador 1D:
  ajuste exato, kerf forçando barra extra, best-fit empacotando barra aberta,
  retalho consumido antes da barra inteira, peça sem encaixe (`unplaced`),
  agrupamento de padrões idênticos, trim deslocando offsets + sobra, e validação de
  parâmetros negativos.
- `go test ./internal/domain/cutting_plan/service/` (2D guilhotina) — **7 testes**:
  ladrilhamento exato, rotação quando permitida, sem rotação → não encaixa, veio
  proíbe rotação, abre segunda chapa, peça grande demais, retalho antes da chapa cheia.
- `go test ./internal/domain/cutting_plan/service/` (UoM) — tabelas de comprimento
  (`StockQtyForLength`) e de área (`StockQtyForArea`): peça, m, m², m³/m², kg/m² e
  erros de fator ausente.
- `go test ./internal/domain/cutting_plan/service/` (true-shape) — **3 testes**:
  bounding-box de polígono, nesting de peças em L pela bbox, e bbox grande demais →
  não encaixa.
- `go test ./internal/infrastructure/nesting/` — **2 testes** do adapter externo:
  tipo declarado e **round-trip HTTP** (request serializado + response mapeado em
  Solution com `rotation_deg`) contra um `httptest.Server`.
- `go test ./internal/application/usecase/cutting_plan_uc/` (demanda) — **2 testes** da
  geração por ordens (com fakes de OP/estrutura/item/plano): explosão de BOM,
  resolução da matéria-prima por LLC 9, agregação (2 produtos × 4 pernas = 8 peças),
  `production_order_code` e UoM snapshot; e validação de "nenhuma ordem".
- `go test ./internal/domain/cutting_plan/service/` (complementos) — raster nester
  (**intertrava 2 peças em L numa chapa** vs 2 chapas sem rotação; in-bounds; peça
  grande demais → unplaced) e export do mapa (SVG/DXF/PDF bem-formados + formato inválido).
- `go test ./internal/application/usecase/cutting_plan_uc/` (rateio) — firmar rateia o
  custo 2000:4000 → 3,33 : 6,67 entre OP-1 e OP-2.

**Teste end-to-end (HTTP):** `make test-cutting` (ou `bash scripts/test-cutting.sh`)
exercita o módulo inteiro contra a API rodando + banco de teste (**36 checagens,
36/36 verdes**): cadastros de apoio, settings (+ rejeição de modo inválido), 1D
(criar/otimizar/firmar/retalho/UoM em metros + cenário de peça sem encaixe), modo
MANUAL sem lote (rejeitado), reuso de retalho (`include_remnants`), 2D + fita de
borda, true-shape (raster), export SVG/DXF/PDF, programa, agenda na máquina, demanda
de OP (`from-orders`, com BOM real) e rateio por OP. Requer `BASE_URL` apontando para
o servidor.

> **Bugs reais encontrados ao rodar o e2e e corrigidos** (pré-existentes no módulo de
> **agenda de máquina**, quebrados para todos, não só para o corte):
> 1. `CreateSchedule` não inseria `code`, mas `machine_schedules.code` é `NOT NULL`
>    sem default → **todo agendamento falhava**. Corrigido: a query passa a auto-atribuir
>    `code = MAX+1`.
> 2. `machine_schedules.order_code` era `NOT NULL` com FK para `planned_orders(code)`
>    → impossível agendar um plano de corte (não vem de ordem planejada). Corrigido:
>    `order_code` virou **opcional** (migration 000166; `OrderCode *int64` na entidade/
>    repo; o corte agenda com `order_code` nulo, referenciando o plano via notas).
- `go test ./internal/application/usecase/cutting_plan_uc/...` — **6 testes** do
  firmar (com fakes de repo + estoque): baixa de barra inteira no modo automático,
  geração de retalho reaproveitável, reuso de retalho do inventário sem nova baixa,
  bloqueio de plano não otimizado, modo manual exigindo lote, **conversão
  comprimento → UoM (metros)** e **firmar 2D com baixa por área (m²) + retalho 2D**.

## 7. Configuração (true-shape externo)

| Variável | Efeito |
|---|---|
| `NESTING_SERVICE_URL` | Quando definida, o ERP delega o nesting **true-shape** a esse endpoint HTTP (protocolo JSON da §3d). Vazia → usa o provedor **nativo raster shape-aware** (§3f.3), com fallback bounding-box para retângulos puros. |

O serviço externo é qualquer microsserviço que implemente o protocolo (ex.: um
wrapper em torno do DeepNest). Nada mais no ERP muda: o `cut_type=TRUE_SHAPE_2D`
continua selecionando o fluxo, e o provedor externo só substitui o cálculo do layout.

## 8. Ideias futuras (fora do escopo entregue)

Tudo do roadmap anterior (export, programa/agenda, NFP nativo, fita de borda, rateio
por OP, limpeza de nomenclatura) **foi entregue na fase de complementos**. Possíveis
evoluções adicionais, não planejadas:

- **NFP geométrico exato** (no-fit-polygon contínuo) no lugar do raster, para máxima
  densidade sem depender de grade/engine externo.
- **Árvore de cortes guilhotinados explícita** no programa 2D (hoje a sequência é a
  ordem dos placements; a árvore de cortes retos pode ser derivada para a seccionadora).
- **Custo da fita de borda** somado ao custeio da OP (hoje exposto no detalhe do plano).
