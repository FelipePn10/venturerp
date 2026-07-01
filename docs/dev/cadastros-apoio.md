# Cadastros de Apoio e Plataforma — Documentação técnica

Reúne os cadastros estruturais e de plataforma não cobertos pelos docs de Cliente,
Fornecedor e Item: Empresa, Funcionário, Armazém, PDM, Localização geográfica,
Classificação de itens, Configurador (perguntas/restrições), Calendário industrial e
Prioridade de ordens. Versão de negócio em
[`../apresentacao/cadastros.md`](../apresentacao/cadastros.md).

> Convenções: `Authorization: Bearer <JWT>`, papel `ADMIN`/`USER`.

---

## 1. Empresa (`/api/enterprise`)
`POST /create` — cadastro da empresa (CNPJ, inscrições, regime tributário, dados SEFAZ).
`GET /list` — lista todas · `GET /{code}` — busca por código.
Suporta matriz/filiais; vínculo de fornecedores via `/api/suppliers/enterprises`.
Código duplicado no `POST /create` retorna **409 Conflict** (não mais 500).

## 2. Funcionário (`/api/employee`)
`POST /create` · `GET /list` · `GET /{code}` · `PUT /update` · `DELETE /{code}/deactivate`.

## 3. PDM (`/api/pdm`)
Grupos/famílias e modificadores de descrição — base da descrição técnica padronizada
e da máscara do item.

| Recurso | Rotas |
|---|---|
| Grupos | `POST /create-group` · `GET /groups` · `GET /groups/{code}` · `PUT /groups/{code}` |
| Modificadores | `POST /create-modifier` · `GET /modifiers` · `GET /modifiers/{id}` · `PUT /modifiers/{id}` |

Corpo do `PUT /groups/{code}`: `{ "description": "...", "enterprise_id": 1 }`.
Corpo do `PUT /modifiers/{id}`: `{ "description": "..." }`.

## 4. Armazém (`/api/warehouse`)
`POST /create`. Tipos em enum `WarehouseType` (linha de produção/normal). Localização
física de estoque e seus tipos: enum `LocationType` (interno/externo/inspeção/rejeição/
reserva/trânsito/especial).

## 5. Localização geográfica (`/api/location`)
Base para endereços e regras fiscais (UF origem/destino → ICMS).

| Recurso | Rotas |
|---|---|
| Países | `POST /countries` · `PUT /countries` · `GET /countries` · `GET /countries/{sigla}` · `GET /countries/{sigla}/ufs` |
| UFs | `POST /ufs` · `PUT /ufs` · `GET /ufs` · `GET /ufs/{sigla}` |

## 6. Classificação de itens (`/api/items/classifications`)
Árvore de categorias por máscara de classificação.

| Recurso | Rotas |
|---|---|
| Máscaras | `POST/PUT/GET /masks`, `GET /masks/{code}`, `GET /masks/{maskID}/items` |
| Classificações | `POST /`, `PUT /`, `GET /{maskCode}/{code}`, `GET /{parentID}/children` |

## 7. Configurador

### Perguntas e opções (`/api/questions`)
`POST /create`, `GET /` (busca por nome); opções: `POST /options/create`,
`GET /options/{questionID}`; associação a itens: `POST /associate`, `GET /associate`,
`GET /associate/item/{itemCode}`.

### Restrições (`/api/restriction`) e motivos (`/api/restriction-reason`)
Regras que bloqueiam combinações inválidas, com avaliação ativa.

| Recurso | Rotas |
|---|---|
| Restrição | `POST /create`, `GET /list`, `GET /{code}`, `GET /item/{itemCode}`, `GET /customer/{customerCode}`, `POST /evaluate`, `PUT /{code}`, `PATCH /{code}/deactivate` |
| Motivo | `POST /create`, `GET /list`, `GET /{code}`, `PUT /{code}`, `DELETE /{code}` |

## 8. Calendário industrial (`/api/industrial-calendar`)
Dias úteis/não úteis da fábrica, consumidos pelo planejamento.

`POST /create` (dia) · `GET /month/{year}/{month}` · `GET /workdays/{year}/{month}`.

## 9. Prioridade de ordens (`/api/order-priority`)
Níveis de prioridade usados pelo APS no sequenciamento.

`POST /create` · `GET /list` · `GET /find/{value}`.
