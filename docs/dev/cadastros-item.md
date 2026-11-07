# Criação de Item — Guia prático (indústria metalúrgica)

Este guia mostra, passo a passo e com exemplos reais de **metalúrgica**, como cadastrar
itens no ERP — da matéria-prima ao produto final — e tudo o que precisa estar
configurado para o item participar do MRP, da produção e das compras.

> Pré-requisitos: estar autenticado (`POST /users/login` → `Authorization: Bearer <JWT>`).
> Toda request é `application/json`.

---

## 1. Conceitos antes de cadastrar

### 1.1 Natureza do item (`Item.Nature`)

| Natureza | Uso na metalúrgica |
|---|---|
| `ItemBase` (2) | **Item base/raiz** que serve de molde para variações (ex.: "Chapa de Aço Carbono"). |
| `ItemGeneric` (0) | Item comum, sem configuração por pergunta. |
| `ItemConfigured` (1) | Item **configurável** por atributos/perguntas (gera **máscara**), ex.: chapa com espessura/dimensão variável. |

Regra do domínio: se `Nature != ItemBase`, é obrigatório informar `Engineering.ItemBaseCod`
(o item base de origem).

### 1.2 PDM — como nasce a descrição técnica

A descrição não é digitada livremente: é **composta** por Grupo + Modificador +
Atributos (PDM), exatamente como na indústria:

```
Grupo: CHAPAS  ·  Modificador: Chapa Aço Carbono  ·  Atributos: {Liga: 1020, Espessura: 6,35mm, Dim: 3000x1200}
→ DescriçãoTécnica: "Chapa Aço Carbono 1020 6,35mm 3000x1200"
```

Cadastre antes (uma vez): **Grupo** (`/api/group`), **Modificador** (`/api/modifier`)
e, para itens configurados, as **Perguntas** que viram a máscara.

### 1.3 Nível de planejamento (LLC) — a espinha dorsal do MRP

`Planning.LLC` (Low-Level Code) define a posição do item na estrutura e a ordem em que
o MRP o processa:

| LLC | O que é na metalúrgica |
|---|---|
| **1** | Produto final vendável (ex.: "Suporte Soldado SS-100"). |
| **2–8** | Conjuntos/subconjuntos e peças intermediárias (ex.: "Suporte Cortado", "Conjunto Soldado"). |
| **9** | Matérias-primas (ex.: "Chapa de Aço", "Eletrodo MIG"). |

> Para o MRP **gerar ordem de compra de matéria-prima**, o item precisa estar `LLC 9`,
> `TipoMRP` que gere ordens e situação **ACTIVE**.

---

## 2. Exemplo de ponta a ponta

Vamos cadastrar a estrutura de um **suporte metálico soldado**:

```
SS-100  Suporte Soldado            (produto final, LLC 1)
 └── SC-050  Suporte Cortado        (intermediário, LLC 2)  — corte na guilhotina
      └── CHP-1020  Chapa Aço 1020  (matéria-prima, LLC 9)  — comprada
 └── EL-MIG  Eletrodo MIG ER70S-6   (matéria-prima, LLC 9)  — consumível de solda
```

---

## 3. Passo 1 — Matéria-prima (Chapa de Aço)

`POST /api/items/create`

Campos por **pasta** (refletem `internal/domain/items/entity/item_entity.go`):

```jsonc
{
  "code": 1020,                       // se omitido, o sistema sugere o próximo
  "name": "Chapa de aço carbono 6,35 mm",
  "nature": 2,                        // ItemBase (chapa-mãe) — ou 0/genérico
  "pdm": {
    "group_code": 10,                 // Grupo CHAPAS
    "modifier_code": 5,               // Modificador "Chapa Aço Carbono"
    "attributes": [
      {"name": "Liga", "value": "1020"},
      {"name": "Espessura", "value": "6,35mm"},
      {"name": "Dimensao", "value": "3000x1200"}
    ]
  },
  "warehouse": {                      // Pasta Almoxarifado
    "warehouse_code": 1,
    "unit_of_measurement": "UN",      // UM de ESTOQUE (chapa inteira)
    "automatic_low": false,
    "minimum_stock": 20,              // alerta de compra; aceita fração (12.5 KG)
    "cyclical_count_config": {"days_interval": 30}
  },
  "engineering": {                    // Pasta Engenharia
    "weight": {"gross": 179.5, "net": 179.5, "unit": "KG"}, // peso por chapa
    "dimensions": {"length": 3000, "width": 1200, "height": 6},
    "type": "RAW_MATERIAL",
    "type_struct": "SIMPLE",
    "oem": false
  },
  "planning": {                       // Pasta Planejamento
    "type_mrp": "NORMAL_MRP",
    "llc": 9,                         // matéria-prima
    "reorder_point": {"tr": 7, "cm": 200, "cr": 2, "es": 50},
    "ghost": false,
    "abc_class": "A",
    "minimum_lot": 10,
    "multiple_lot": 5,
    "safety_stock": 2,                // lote, múltiplo e segurança aceitam fração
                                      // (6 casas, na UM de estoque): barra de 6,35 M
                                      // → "multiple_lot": 6.35. O reorder_point
                                      // continua inteiro (TR/CM/CR/ES).
    "critical": false,
    "exclusive": false,
    "active": true
  },
  "supplies": {
    "type_of_use": "CONSUMO",
    "purchase_uom": "KG",
    "warehouse_code": 1,
    "receiving_checklist": true,
    "harvest": false
  },
  "commercial": {
    "description": "CHAPA DE AÇO CARBONO",
    "sale_type": "VENDA",
    "sale_multiple": 5,
    "warranty_days": 365,
    "issue_loading_labels": true
  },
  "accounting": {
    "sale_fiscal_classification_code": "72085400",
    "purchase_fiscal_classification_code": "72085400",
    "origin": 0,
    "sale_ipi_type": "PERCENTUAL",
    "sale_ipi_rate": 5,
    "sale_unit_of_measurement": "UN",
    "purchase_unit_of_measurement": "KG",
    "cest": "0100100",
    "calculate_pis_cofins": true
  },
  "created_by": "<uuid-do-usuario>"
}
```

### Cadastro ampliado (migração 000468)

O comparativo com outros ERPs está em `comparativo-cadastro-item.md`. Campos novos, nos
corpos de criar e de alterar. Neles, ausente preserva e `null` limpa (`request.Opcional`).

- Raiz: `long_description`, `general_notes`, `revision`, `previous_code`,
  `manufacturer_name`, `manufacturer_part_number`.
- `warehouse`: `shelf_life_days`, `min_remaining_life_days`, `storage_conditions`,
  `hazardous`, `un_number`, `hazard_class`, `accepts_fractional_quantity`.
- `engineering`: `volume`.
- `planning`: `planner_employee_code`, `buyer_employee_code`, `purchase_lead_time_days`,
  `manufacturing_lead_time_days`, `notes`, `material_issue_timing`,
  `production_reporting_type`, `maximum_stock`, `coverage`, `grouping_key`,
  `safety_time_days`, `maximum_lot`. Os três primeiros complementos ficam em
  `item_planning_extras`; os dois últimos em `politicas_planejamento`, onde o MRP lê.
- `supplies`: `under_delivery_tolerance_pct`, `over_delivery_tolerance_pct`,
  `receipt_routing` (`DIRETO`, `PADRAO`, `INSPECAO`), `classification_code`.
- `accounting`: `gtin`, `gtin_tax`, `efd_item_type`, `tax_unit`, `tax_unit_factor`.

Migração 000469: `is_scrap`; `engineering`: `requires_drawing`, `cutting_grain`
(`NONE`, `LENGTH`, `WIDTH`), `cutting_no_rotation`; `supplies`: `requires_certificate`;
`commercial`: `exclusive_customer_code`, `volumes_per_unit`. `Item.ConferirVenda` e
`Item.ForaDoMRP` concentram as regras. A auditoria de cada indicador está no comparativo.

Grupo e modificador têm `is_active`, e o código do grupo é único por empresa.

Migração 000470: `description_en` (vai na nota com CFOP 7xxx); `warehouse`: `secondary_uom` e
`secondary_factor`; `accounting`: `ex_tipi`, `relevant_scale`, `manufacturer_cnpj`, `fci_number`.
`item_gtins` guarda os níveis de embalagem do item, usados na NF-e, no leitor e na entrada de nota. Migração 000471:

- `parent_uom` e `parent_quantity` dizem do que o nível é feito: o palete tem 40 CX, a caixa tem 10 UN. `package_quantity` é sempre o total de unidades do item, calculado ao gravar.
- O GTIN é opcional no nível que informa o que contém: o palete costuma não ter código de barras.
- Ciclo e nível inexistente são recusados, e também o que não cabe na coluna: unidade acima de 20 caracteres, descrição acima de 120, total a partir de 1 trilhão.
- A linha da própria unidade do item tem quantidade 1.

`item_atributos_livres` guarda o que o PDM não padroniza. O nome fica em maiúsculas, com no máximo 30 atributos por item. Os valores entram na busca de item das telas, por `GET /api/items/governanca/busca/resumo`.

O saldo em duas unidades traz `details`, com quantidade e segunda quantidade por almoxarifado e lote.

Regras da NF-e conferidas na homologação da SEFAZ-PR (06/11/2026):

- `cProd` é o código comercial do item, nunca a chave interna. A linha que traz um código próprio (retorno de beneficiamento) mantém o dela.
- O CEST vem do item ou, sem ele, da classificação fiscal de venda.
- `indEscala` e `CNPJFab` pertencem ao grupo do CEST: sem CEST ficam fora do XML, e a prévia fiscal avisa.
- Escala não relevante exige o CNPJ do fabricante (rejeição 879); o cadastro recusa.
- Linha em embalagem: a quantidade tributável é quantidade × `package_quantity` × `tax_unit_factor`. O `cEAN` é o da caixa e o `cEANTrib`, o da unidade.
- `cEAN` e `cEANTrib` andam em par: só o `cEANTrib` rejeita com 886. Faltando um dos dois, ambos saem `SEM GTIN`. É o caso do palete sem código e da caixa com GTIN de item sem GTIN na unidade (autorizadas em 07/11/2026).
- Observado na homologação: só GTIN de prefixo 789 ou 790 é mantido; os demais saem `SEM GTIN` e a nota é autorizada. A prévia fiscal do item avisa.
Revisão com início futuro: a OF já nasce na revisão vigente na data, e um job de hora em hora leva essa revisão a `items.revision`.

Regras que o recebimento e o despacho aplicam:

- A tolerância de excesso e a de falta olham o total da linha, somando os lotes do mesmo recebimento.
- A entrada por NF-e segue o roteiro do item: `DIRETO` não passa pela inspeção, e `INSPECAO` sem rota ativa recusa a aprovação. A nota não traz corrida e certificado, então o sistema só avisa.
- A baixa da embalagem usa a quantidade conferida. Sem saldo, o despacho segue, baixa o que há e registra a falta no histórico do romaneio.
- A linha de venda sem data só assume o prazo estimado do item quando o pedido também não tem data.

`item_revisoes` guarda a vigência: a nova fecha a anterior na véspera, e a OF grava a revisão com
que nasceu (`item_revision`). Na segunda unidade, `stock_movements.secondary_quantity` recebe o
valor real ou, sem ele, quantidade × fator; o estorno repete o original.

A política "com aviso" chega à tela pelo cabeçalho `X-Item-Aviso` em POST e PUT dos documentos.

Rotas: `GET|PUT /api/items/governanca/itens/{item}/gtins`, `GET|PUT .../atributos-livres`, `GET|POST .../revisoes`, `GET .../saldo`,
`GET /api/pdm/uso`.

Lote, endereço e série seguem nos endpoints próprios (`/api/stock/item-control/{item}` e
`/api/pos-venda/perfis/{item}`), por causa das regras deles. A tela grava depois do item.

Quem consome: NF-e (`anexarIdentificacao`), 0200 da EFD (`cadastroEFD`), entrada de nota
(`ResolveByEAN`), lote (`registrarLoteDaEntrada`), recebimento (`RegrasDeRecebimento`),
MRP (`prazoDoCadastro`), busca de item e romaneio.

Rotas novas: `POST /api/items/{code}/impacto`, `GET /api/items/governanca/politicas`,
`PUT /api/items/governanca/politicas`, `GET /api/items/governanca/itens/{item}/prontidao`,
`GET /api/items/governanca/itens/{item}/previa-fiscal`, `GET /api/pdm/modelo`,
`GET|PUT /api/pdm/groups/{code}/modifiers`, `PUT /api/pdm/modifiers/{id}/abbreviation`,
`POST /api/pdm/atributos`, `PUT|DELETE /api/pdm/atributos/{id}`, `POST /api/pdm/descricao`.

⚠️ Rota nova em `/api/items/{code}/...` que receba o código comercial precisa entrar em
`nativeItemBusinessCodePath`. Sem isso o middleware troca o código pela chave interna
antes do roteamento e o caso de uso abre outro item.

As pastas `commercial` e `accounting` são opcionais. `PUT /api/items/{code}` aceita
as mesmas pastas de forma parcial: somente propriedades enviadas são alteradas;
valores `false` e `0` enviados explicitamente são preservados. `sale_type` aceita
`VENDA|REVENDA` e os tipos de IPI aceitam `PERCENTUAL|VALOR`.

Pontos importantes para a metalúrgica:

- **UM de compra ≠ UM de estoque:** a chapa é **comprada em KG** mas **estocada em UN**.
  Cadastre o **fator de conversão por item** (senão o pedido de compra não calcula a
  quantidade interna):
  `POST /api/item-conversions` → `{ "item_code": 1020, "from_uom": "KG", "to_uom": "UN", "factor": 0.005571, "created_by": "<uuid>" }`
  (1 KG = 1/179,5 chapa). Ver `manufatura-e-compras.md` §11.
- **Ponto de pedido (ROP):** `(TR × CM / CR) + ES`. O cadastro já guarda os parâmetros;
  uma política selecionável específica ainda depende do despacho correspondente no motor MRP.

### 3.1 Classificação fiscal da matéria-prima

`POST /api/fiscal-classifications` (NCM, CEST, IPI/PIS/COFINS, CST). Ex. chapa de aço:

```jsonc
{ "item_code": 1020, "description": "Chapas de aço carbono", "ncm": "72085400", "ipi_rate": 0, "pis_rate": 0.0165, "cofins_rate": 0.076, "created_by": "<uuid>" }
```

Ver `fiscal-financeiro.md` §34. O **%IPI** daqui é puxado automaticamente para o item do
pedido de compra.

### 3.2 Fornecedor preferencial da chapa

`POST /api/item-suppliers` (também guarda código/descrição/UM do item no fornecedor):

```jsonc
{ "item_code": 1020, "supplier_code": 500, "ranking": 1, "uom": "KG", "lead_time_days": 7, "created_by": "<uuid>" }
```

Isso faz a **Geração de Pedidos** e a **Cotação** sugerirem esse fornecedor
automaticamente (`manufatura-e-compras.md` §14–§16).

---

## 4. Passo 2 — Item intermediário (Suporte Cortado)

Mesmo `POST /api/items/create`, agora `nature = 0` (genérico), `llc = 2`,
`type_mrp` que **gere ordem de produção** (item fabricado), `type_struct` de conjunto:

```jsonc
{
  "code": 50,
  "nature": 0,
  "pdm": {"group_code": 20, "modifier_code": 9, "attributes": [{"name": "Peca", "value": "Suporte"}]},
  "warehouse": {"warehouse_code": 1, "unit_of_measurement": "UN", "minimum_stock": 0},
  "engineering": {"weight": {"gross": 1.2, "net": 1.2, "unit": "KG"}, "type": "MANUFACTURED", "type_struct": "ASSEMBLY", "oem": false, "item_base_cod": null},
  "planning": {"type_mrp": "MRP", "llc": 2, "ghost": false},
  "supplies": {"type_of_use": "INDUSTRIALIZATION"},
  "created_by": "<uuid>"
}
```

> `Ghost = true` (item fantasma) é útil para fases que **não estocam** (passam direto
> ao pai). Use quando a peça cortada não é estocada entre o corte e a solda.

---

## 5. Passo 3 — Produto final (Suporte Soldado SS-100)

`POST /api/items/create` com `llc = 1`, `type = FINISHED`/produto final:

```jsonc
{
  "code": 100,
  "nature": 0,
  "pdm": {"group_code": 30, "modifier_code": 12, "attributes": [{"name": "Modelo", "value": "SS-100"}]},
  "warehouse": {"warehouse_code": 2, "unit_of_measurement": "UN", "minimum_stock": 5},
  "engineering": {"weight": {"gross": 2.6, "net": 2.5, "unit": "KG"}, "type": "FINISHED", "type_struct": "ASSEMBLY", "oem": false},
  "planning": {"type_mrp": "MRP", "llc": 1, "ghost": false},
  "supplies": {"type_of_use": "INDUSTRIALIZATION"},
  "created_by": "<uuid>"
}
```

---

## 6. Passo 4 — Estrutura (BOM)

A estrutura liga pai → filho com quantidade e **percentual de perda** (importante em
corte/estampagem de chapa). `POST /api/items/structure/create`:

```jsonc
// SS-100 consome 1 Suporte Cortado + 0,15 kg de eletrodo
{ "parent_code": 100, "child_code": 50, "quantity": 1, "loss_percentage": 0, "unit_of_measurement": "UN", "sequence": 1 }
{ "parent_code": 100, "child_code": 9001, "quantity": 0.15, "loss_percentage": 5, "unit_of_measurement": "KG", "sequence": 2 }

// SC-050 consome 1 chapa por peça, com 8% de perda de corte
{ "parent_code": 50, "child_code": 1020, "quantity": 1, "loss_percentage": 8, "unit_of_measurement": "UN", "sequence": 1 }
```

- **Fórmula de perdas** (aplicada pelo MRP na explosão):
  - Fórmula 1 (padrão): `qtd = qtdPai × qtdComponente × (1 + %perda/100)`
  - Fórmula 2: `qtd = ... / (1 − %perda/100)`
- **Campos enterprise+ da linha da BOM:**
  - `is_coproduct`: saída da produção (co-produto/subproduto/sucata retornável), não
    gera demanda de consumo.
  - `is_fixed_qty`: quantidade por OF/lote, não por unidade produzida.
  - `substitute_group`: componentes do mesmo pai com o mesmo grupo (> 0) são
    alternativos/substitutos.
  - `substitute_priority`: menor valor é o componente primário. MRP, custo, backflush
    e plano de corte usam o primário; os demais ficam disponíveis como alternativa
    operacional quando o primário faltar.
- Consultar a árvore: `GET /api/items/structure/resolve/{itemCode}`; onde-é-usado:
  `GET /api/items/structure/where-used/{itemCode}`.

---

## 7. Passo 5 — Roteiro e tempos de máquina

Para o **APS/CRP** e o lead time, associe operações/máquinas.

### 7.1 Tipos de máquina e máquinas
- `POST /api/machine/types/create` (ex.: "Guilhotina", "Solda MIG").
- `POST /api/machine/create` (capacidade, eficiência, período — ver `maquinas-e-roteiro.md`).

### 7.2 Tempo por item × máquina
`POST /api/machine/time/create` — quanto a peça leva em cada máquina (setup + ciclo +
quantidade-base por ciclo):

```jsonc
{ "item_code": "TEA452-0", "machine_code": 1, "production_time": 0.5, "production_time_unit": "MINUTO", "production_base_qty": 1, "setup_time": 15 }
```

O serviço de cálculo (`machine/service`) considera **conversão de UM**, ciclos
(`ceil(demanda / base)`), setup e detecta **gargalo** de capacidade.
`POST /api/machine/time/production/calculate` simula o tempo total.

### 7.3 Roteiro de fabricação (operações encadeadas)
Cadastre o roteiro (operações, predecessores, overlap) — o MRP usa o **caminho crítico
(CPM)** do roteiro para o lead time, e o **APS** sequencia em capacidade finita. Ver
`manufatura-e-compras.md` (Roteiro/APS/CRP).

---

## 8. Checklist — "o item está pronto para o MRP?"

- [ ] Item `ACTIVE` (situação) e com `LLC` correto (1 produto, 9 matéria-prima).
- [ ] `TipoMRP` definido (MRP/MIN_MAX/KANBAN/REORDER_POINT/MPS).
- [ ] **Estrutura (BOM)** cadastrada para itens fabricados.
- [ ] **Conversão de UM** cadastrada quando compra ≠ estoque.
- [ ] **Classificação fiscal** vinculada (impostos do pedido).
- [ ] **Fornecedor preferencial** para matérias-primas compradas.
- [ ] **Tempos de máquina/roteiro** para itens fabricados (APS/CRP).
- [ ] Parâmetros de planejamento (lote mínimo, estoque de segurança) — `/api/planning-params`.

> ✅ **automático (validação de prontidão):** `GET /api/items/{code}/activation-readiness`
> roda este checklist para você e devolve `{ready, issues, warnings}`:
> - item **fabricado** → exige **estrutura (BOM)** e **roteiro**;
> - item **comprado** → exige **fornecedor preferencial** e alerta se faltar
>   **conversão de UM** (necessária quando compra ≠ estoque).
>
> Implementado em `item_uc.ValidateItemActivationUseCase`. Observação: o modelo de
> item ainda **não tem um ciclo ACTIVE/INACTIVE** (o campo `situation` é
> LINHA/PROMOCAO), então este endpoint **valida a prontidão** sem alterar estado —
> use-o como gate antes de colocar o item em operação.

> 💡 **integração com o fluxo:** uma vez cadastrado e pronto, o item participa do fluxo
> **automatizado** descrito em [`00-fluxo-geral.md`](00-fluxo-geral.md):
> confirmar o pedido de venda gera a **demanda do MRP** automaticamente; firmar a
> ordem planejada de produção gera a **OF**; e consumo/conclusão da OF geram os
> **movimentos de estoque** (`OUT`/`IN`) que atualizam o saldo.

---

## 9. Endpoints citados (resumo)

| Recurso | Endpoint |
|---|---|
| Criar item | `POST /api/items/create` |
| Validar prontidão p/ o fluxo | `GET /api/items/{code}/activation-readiness` |
| Listar itens (+máscaras) | `GET /api/items/` · `GET /api/items/with-masks` |
| Estrutura/BOM | `POST /api/items/structure/create` · `GET /resolve/{itemCode}` · `GET /where-used/{itemCode}` |
| Conversão de UM | `POST /api/item-conversions` |
| Classificação fiscal | `POST /api/fiscal-classifications` |
| Fornecedor preferencial | `POST /api/item-suppliers` |
| Máquina / tempos | `POST /api/machine/...` · `POST /api/machine/time/create` |

> O **fluxo completo** desse produto (pedido de venda → MRP → APS → ordens → estoque)
> está em [`00-fluxo-geral.md`](00-fluxo-geral.md).

## 10. Governança do cadastro (`/api/items/governanca`) — migration 000439

Fase 4. Domínio `internal/domain/items/governanca` (status, transições,
catálogo de campos por visão, completude, similaridade); caso de uso
`item_governanca_uc`; repositório `repository/itemgovernanca`. Tela VENT0250.

- **Status** (`item_governanca`, por empresa): `EM_CADASTRO`, `LIBERADO`,
  `BLOQUEADO`, `DESCONTINUADO` (com `substituto_code`, que precisa estar
  liberado). Sem linha = `LIBERADO` (itens antigos não mudam). Transições:
  `EM_CADASTRO → BLOQUEADO|DESCONTINUADO` (a liberação é pelas visões),
  `LIBERADO → BLOQUEADO|DESCONTINUADO`, `BLOQUEADO → LIBERADO|DESCONTINUADO`,
  `DESCONTINUADO → LIBERADO`, sempre com motivo. A mudança leva o status que a
  tela viu (`de`) e recusa com conflito se ele mudou.
- **Item novo:** com `item_governanca_parametros.exige_liberacao`, o
  `CreateItemUseCase` chama `AoCriarItem` e o item nasce `EM_CADASTRO`.
- **Visões** (`item_visoes_liberadas`): ENGENHARIA, FISCAL, SUPRIMENTOS,
  PLANEJAMENTO, COMERCIAL, CUSTOS. Liberar exige a visão sem pendência e
  `Aprovar` no módulo da área (`ControleDeAcesso.PodeNoModulo`; usuário sem
  perfil restrito segue o papel). Com a última visão exigida
  (`parametros.visoes`), `EM_CADASTRO → LIBERADO` na mesma transação.
- **Completude:** `governanca.Catalogo` define o campo, a visão, os tipos de
  item (`engineering_type`: 0 fabricado, 1 comprado, 2 de terceiro, 3 serviço)
  e se é obrigatório por padrão; a empresa ajusta em `item_regras_campo`. A
  ficha (`to_jsonb(items)` + estrutura, roteiro, fornecedor, custo padrão,
  conversão da UM de compra) é lida pelo repositório.
- **Bloqueio de uso:** gatilho `item_conferir_utilizavel` (BEFORE INSERT e
  UPDATE OF item_code) em `sales_order_items`, `sales_quotation_items`,
  `purchase_order_items`, `purchase_requisition_items`, `production_orders`,
  `fiscal_exit_items`: `RAISE ... ERRCODE 'P0001'` com a mensagem ao usuário.
  `security.RespondUseCaseError` trata `P0001` como 422 `REGRA_DE_NEGOCIO`.
- **Histórico** (`item_historico`, imutável): gatilho `trg_items_historico`
  grava a criação e cada coluna alterada (fora as calculadas: ABC, LLC,
  política de contagem), com o autor de `venture.item_actor`. O repositório de
  item com pool (`ComPool`) faz a alteração numa transação com `set_config`.
  Liberações e status também entram no histórico.
- **Duplicidade** (`GET /duplicados?nome=&ncm=&excluir=`): candidatos por
  palavra em comum ou mesmo NCM (SQL), nota por `governanca.Classificar`
  (sobreposição de palavras ou distância de edição, sem acento; ≥ 0,6). Sem
  extensão `pg_trgm`.
- **Alertas:** produtores `CADASTRO_ITEM_SEM_CLASSIFICACAO_FISCAL`,
  `_COMPRADO_SEM_FORNECEDOR`, `_FABRICADO_SEM_ESTRUTURA`,
  `_FABRICADO_SEM_ROTEIRO` (item ativo e liberado) e
  `_INATIVO_COM_DEMANDA_ABERTA` (inativo/bloqueado/descontinuado com pedido ou
  OF aberta) em `ScheduleOperationalAlerts`, com resolução automática.
- **Rotas:** `GET|PUT /parametros`, `GET /regras`, `PUT /regras/{campo}`
  (`{"obrigatorio": bool|null}`), `GET /pendentes`, `GET /duplicados`,
  `GET /itens/{item}`, `POST /itens/{item}/liberacoes|estorno|status`,
  `GET /itens/{item}/historico|uso`. `governanca` é caminho estático no
  middleware do código de negócio; o caso de uso resolve o código. Painel,
  pendentes e liberação/estorno são rotas livres (a área decide); status
  exige `Aprovar` no módulo Engenharia.

Teste: `item_governanca_uc/governanca_integration_test.go`.

### Campos fora da tela (VENT0200)

As colunas continuam no banco e na API; só deixaram de ser editadas na tela, e um
item que já as tem não perde o valor ao ser gravado (o campo ausente no payload fica
como está).

| Campo | Por quê |
|---|---|
| `commercial_volume_conversion_factor` | Repetia **volumes por unidade**. O romaneio ainda o usa quando volumes por unidade está vazio. |
| `commercial_packaging_item_code` | A embalagem do produto vai na estrutura (VENT0210). O despacho ainda baixa este item quando preenchido. |
| `accounting_purchase_unit_of_measurement` | Repetia a unidade de compra de `supplies_purchase_uom`, que é a usada. |
| `accounting_sale_unit_of_measurement` | Não saiu: foi para a aba Comercial como **unidade de venda**. |
| `planner_employee_code`, `planning_buyer_employee_code` | Apontam para uma pessoa: com demissão ou troca de setor o cadastro fica errado. Saíram também do catálogo de obrigatórios. |
| `item_planning_extras.coverage`, `grouping_key` | Lidos pelo MRP e nunca usados no cálculo. |
| `accounting_inventory_group_code` | Repetia o grupo do PDM e nada o lia. Saiu do catálogo de obrigatórios. |
| `accounting_classification_code`, `accounting_input_code`, `commercial_classification_code`, `supplies_classification_code` | Nada no sistema os lê. |

### Busca por outros códigos do item

`GET /api/items/governanca/busca/resumo` devolve, por item, o código anterior, a
referência do fabricante, o código do item em cada fornecedor
(`item_preferred_suppliers.supplier_item_code`), os GTINs (comercial, tributável e
das embalagens) e os atributos livres. Todo seletor de item do front (`loadItems`,
`loadBaseItems`, a lupa dos campos legados e as buscas das grades de OF, demanda e
política de planejamento) filtra também por esse texto.

Em campo que recebe **código digitado**, o sistema não troca o apelido pelo item:
o código anterior de um item pode ser o atual de outro, e dois fornecedores podem
usar o mesmo código. A recusa diz de quem é o código ("ANT-9981 é o código
anterior do item TEA453-3"), no middleware e em `itemresolution.Resolve`.

### Contagem cíclica

O item com intervalo próprio é contado a cada N dias da última contagem aprovada
(ou da ativação). Sem intervalo próprio vale a classe ABC (`stock_abc_count_policy`,
padrão A 90, B 180, C 365), a partir do início da política ou do cadastro do item,
o que vier depois. Item inativo não entra pela regra ABC. Programar e cancelar usam
a mesma regra (`politicaDeContagem`): antes, o item só com classe ABC nunca era
programado.

Os alertas de contagem vencida e de estoque abaixo do mínimo só geram evento com o
alerta ligado na empresa (VNOT0100). O evento de alerta desligado era criado e
cancelado, e ocupava a chave de deduplicação: depois de ligado, o alerta não vinha.

### Código digitado aponta para um item só

Além de a recusa dizer de quem é o código, o cadastro impede o conflito na origem:
o código anterior e a referência do fabricante não podem ser o código atual de
outro item, e um item novo não pode nascer com o código anterior, a referência ou
o código de fornecedor de outro (`conferirConflitoDeCodigos`). O código de
fornecedor também não pode ser o código atual de outro item nosso
(`ConferirCodigoDoFornecedor`, em `apelidos_pg.go`): o upsert de
`item_suppliers` (VSUP0130) e o vínculo aprendido na entrada de NF-e
(`upsertVinculo`) recusam com o item que já usa o código. Na busca, a opção achada por um apelido mostra qual
("Achado pelo código do fornecedor X") e o apelido digitado não vira "código
informado manualmente".

### Marcações que se contradizem

`conferirMarcadores` recusa: sucata com item base, configurado, protótipo,
ferramenta ou item de processo; item de processo com base, configurado ou
ferramenta; fantasma que não é Fabricado, que é crítico ou que tem contagem
cíclica; embalagem de exportação e diferenciada juntas; serviço perigoso ou com
contagem. O controle de lote/endereço recusa fantasma e serviço; ligar a série torna o
item de quantidade inteira (o padrão do cadastro é aceitar fração). A tela desabilita a opção e diz o motivo.

### Cuidados de armazenagem e segurança

`GET /api/items/governanca/cuidados` lista os itens perigosos ou com condição de
armazenagem. O aviso aparece na OF (resumo e materiais), na entrada de NF-e, na
requisição, na separação, no recebimento e nas etiquetas do coletor (VEST0150) e
na expedição. O romaneio (PDF e XLSX) usa o código comercial (`RomaneioItem.Codigo`)
e, no PDF, a seção "CUIDADOS NO TRANSPORTE E NA ARMAZENAGEM" depois dos itens
(`RomaneioItem.Cuidados`), sem truncar.

### Tanques / linhas de capacidade

Migração 000472: `delivery_capacity_lines` (código, descrição, capacidade diária,
unidade). Cadastro em VPME0102; o item escolhe a linha e o cadastro recusa linha
inexistente ou desativada. A promessa de entrega usa a capacidade de cada tanque;
a informada na consulta só vale para tanque sem cadastro. A migração criou uma
linha para cada tanque que os itens já usavam (capacidade 1, a ajustar).

### Ponto de pedido por dia trabalhado

`reorder_point` com `modo: "DIA_TRABALHADO"`: CM é o consumo por dia trabalhado
(0 = apurar), CR = 1 e TR o prazo em dias úteis. A reposição calcula consumo por
dia × dias que a empresa trabalha no prazo (calendário industrial; sem registro,
segunda a sexta, como `subtract_workdays`) + segurança. Sem CM, o consumo é o dos
últimos 90 dias ÷ dias trabalhados nesses 90 dias. O MRP usa o mesmo número:
`MRPServiceImpl.PontosDePedido` (`StockRepositorySQLC.PontoDePedidoDoItem`) é
consultado para o item em modo por dia trabalhado; o relatório antigo segue com
TR × CM ÷ 1 + ES, a estimativa de segunda a sexta.

### Impostos não ficam no item

O motor da nota tira o IPI do NCM (`ncm_tax_table`, VFIS0110), o ICMS da UF e da
regra de ICMS/IPI (VFIS0320) e o PIS/COFINS do regime e das exceções do NCM. As
alíquotas e o "calcula PIS/COFINS" do item não entravam no cálculo e saíram da
tela; as colunas continuam. A unidade da entrada manual vem da unidade de compra
(`supplies_purchase_uom`); a antiga, da aba Contábil, é só reserva.

### Dependências entre campos (VENT0200)

- Tipo do item: campos de OF (prazo de fabricação, baixa de material, apontamento,
  exige desenho) só para Fabricado; prazo de compra só para Comprado; Serviço
  desliga segunda unidade, estoque mínimo/máximo, baixa automática, lote, endereço,
  série, contagem e produto perigoso. Revenda exige Comprado (o backend recusa).
- Validade e vida útil mínima só com controle de lote; regra FEFO/FIFO idem. O
  backend impõe: `UpdateFolders` recusa validade sem lote
  (`conferirValidadeComLote`); a criação com validade liga `controls_lot` na mesma
  transação; `SalvarControleDoItem` recusa desligar o lote de item com validade.
- Escala relevante e CNPJ do fabricante só com CEST; escala "Não" exige o CNPJ.
- Produto perigoso exige condições de armazenagem; contagem própria exige o
  intervalo; o mínimo de venda é múltiplo do múltiplo (o backend recusa);
  embalagem é tipo 02 na EFD (o backend recusa).
- Estado "Fantasma" e a marca Fantasma andam juntos na tela (não no backend: o
  item 302 da Tecnofer é comprado com estado Fantasma).
- Marcar número de série desmarca "aceita quantidade fracionada".

### Prévia fiscal e fiscal efetivo

A prévia fiscal (aba Prontidão) mostra os impostos pela simulação, o mesmo motor
da nota, em `GET /api/items/governanca/itens/{item}/impostos?uf=XX` (módulo do
item, sem exigir o fiscal): venda a pessoa jurídica com IE, para
industrialização, dentro e fora do estado. Sem IE o motor trataria o destino como
consumidor final (IPI na base do ICMS). As alíquotas da classificação fiscal
(VFIS0350) não entram no cálculo e a prévia antes mostrava PIS/COFINS 0%. O
`fiscal_effective` do item não anuncia mais IPI, ICMS e "calcula PIS/COFINS" do
item como sobrescritos: continua só CEST, origem e unidade.

### Classificação fiscal e IPI

`FiscalClassificationUseCase.GetIPIRate` lê primeiro o IPI do NCM da
classificação na `ncm_tax_table` (`TabelaNCM`, `AliquotaIPIDoNCM`) e só cai no
`ipi_rate` da classificação quando o NCM não está na tabela. A linha do pedido de
compra sem IPI e sem classificação usa a classificação do item
(`ClassificacaoFiscalDoItem`: a de compra, senão a de venda, ativa e da empresa).

### Setores e responsáveis por grupo (migração 000473)

Reaproveita os setores da central de notificações (`enterprise_departments` e
`enterprise_department_users`, migração 000308; só usuários da empresa) e cria
`pdm_group_responsibles` (grupo PDM → setor de planejamento e setor de compras). Rotas `/api/setores`
(cadastro, pessoas, responsáveis do grupo) e
`GET /api/items/governanca/itens/{item}/responsaveis`. A cobertura do MRP mostra o
setor de planejamento do grupo no lugar do planejador do item, quando há.
