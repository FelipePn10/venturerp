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
    "minimum_stock": 20,              // alerta de compra
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
    "type_mrp": "REORDER_POINT",      // ou MIN_MAX/KANBAN conforme política
    "llc": 9,                         // matéria-prima
    "reorder_point": {"tr": 7, "cm": 200, "cr": 2, "es": 50},
    "ghost": false
  },
  "supplies": {"type_of_use": "CONSUMPTION"},
  "created_by": "<uuid-do-usuario>"
}
```

Pontos importantes para a metalúrgica:

- **UM de compra ≠ UM de estoque:** a chapa é **comprada em KG** mas **estocada em UN**.
  Cadastre o **fator de conversão por item** (senão o pedido de compra não calcula a
  quantidade interna):
  `POST /api/item-conversions` → `{ "item_code": 1020, "from_uom": "KG", "to_uom": "UN", "factor": 0.005571, "created_by": "<uuid>" }`
  (1 KG = 1/179,5 chapa). Ver `manufatura-e-compras.md` §11.
- **Ponto de pedido (ROP):** `(TR × CM / CR) + ES` — o MRP usa isso quando `type_mrp = REORDER_POINT`.

### 3.1 Classificação fiscal da matéria-prima

`POST /api/fiscal-classifications` (NCM, CEST, IPI/PIS/COFINS, CST). Ex. chapa de aço:

```jsonc
{ "description": "Chapas de aço carbono", "ncm": "72085400", "ipi_rate": 0, "pis_rate": 0.0165, "cofins_rate": 0.076, "created_by": "<uuid>" }
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
{ "item_code": 50, "machine_code": 1, "production_time": 0.5, "production_time_unit": "MINUTO", "production_base_qty": 1, "setup_time": 15 }
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
