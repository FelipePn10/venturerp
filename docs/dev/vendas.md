# Vendas e Expedição — Documentação técnica

Cobre Pedido de Venda, Divisão de Vendas, Promessa de Entrega, Reprogramação de
Entrega e Expedição (romaneio). A versão de negócio está em
[`../apresentacao/vendas.md`](../apresentacao/vendas.md). Detalhe aprofundado do
Pedido de Venda também em [`visao-geral.md`](visao-geral.md) §4.

> Convenções: `Authorization: Bearer <JWT>`, `Content-Type: application/json`.
> Salvo indicação, todas as rotas exigem papel `ADMIN` ou `USER`.

---

## 1. Pedido de Venda (`/api/sales-order`)

| Método | Rota | Ação |
|---|---|---|
| POST | `/create` | Cria a capa do pedido |
| GET | `/list` | Lista pedidos |
| GET | `/{code}` | Consulta por código |
| PUT | `/{code}` | Atualiza a capa |
| DELETE | `/{code}/cancel` | Cancela o pedido |
| PATCH | `/{code}/block` | Bloqueia (crédito/manual) |
| PATCH | `/{code}/unblock` | Desbloqueia |
| PATCH | `/{code}/status` | Muda o status |
| GET | `/customer/{customerCode}` | Lista por cliente |
| GET | `/status/{status}` | Lista por status |

### Itens (`/api/sales-order/items`)

| Método | Rota | Ação |
|---|---|---|
| POST | `/create` | Adiciona item (item, qtd, data de entrega) |
| GET | `/{code}` | Lista itens do pedido |
| PUT | `/{itemCode}` | Atualiza item |
| DELETE | `/{itemCode}/cancel` | Cancela item |

**Status do pedido:** `R` (rascunho) → `P` (pedido/confirmado) → `F` (faturado);
`CANCELLED`; estado **bloqueado** ortogonal (crédito/manual). A coluna `status`
comporta os 20 caracteres de `CANCELLED` (migration `000170`); antes era
`VARCHAR(5)` e o cancelamento estourava com *value too long*.

**Datas.** `emission_date`/`delivery_date` (capa) e `delivery_date` (item) aceitam
`YYYY-MM-DD` ou ISO-8601 com hora; `emission_date` omitido assume **hoje** (não mais
`0001-01-01`). `enterprise_code` é obrigatório no `POST /create` (422 se ausente).

> ✅ **Automação:** mudar o status para `P` cria, por item, uma **demanda
> independente** (item, qtd, data) de forma **idempotente** — código derivado da linha
> (`código_pedido × 100000 + sequência`). Ver `sales_order_uc/manage_sales_order_uc.go`
> e [`00-fluxo-geral.md`](00-fluxo-geral.md).
>
> ✅ **Automação (crédito):** confirmar (`P`) roda a **checagem de limite de
> crédito** (exposição = contas a receber em aberto + outros pedidos em aberto).
> Excedeu o limite (ou cliente bloqueado) → pedido **bloqueado** automaticamente,
> sem gerar demanda nem reserva. Ver `sales_order_uc/credit_check.go`.
>
> ✅ **Automação (ATP/reserva):** aprovado no crédito, cada linha **reserva o
> estoque disponível** no depósito da linha (limitado ao disponível). ATP em
> `GET /api/stock/balances/atp/{itemCode}`. Ver `sales_order_uc/order_reserve.go`.
>
> ✅ **Automação (faturamento):** a autorização da NF-e de saída posta `OUT` por item,
> consome reservas do pedido e marca o pedido como `F`. Ver
> `fiscal_uc/authorize_fiscal_exit_uc.go` e [`fiscal-financeiro.md`](fiscal-financeiro.md).

---

## 2. Divisão de Vendas (`/api/sales-division`)

Organização comercial (equipe/região/unidade) associável ao pedido para análise de
resultado e regras comerciais.

| Método | Rota | Ação |
|---|---|---|
| POST | `/create` · GET `/list` · GET `/{code}` · PUT `/{code}` · DELETE `/{code}` | CRUD completo |

**Campos de análise (enum `sales_division_analysis_enum`).** Tanto `commercial_analysis`
quanto `financial_analysis` só aceitam os valores abaixo (case-sensitive). São
opcionais — quando omitidos ou vazios assumem `FREE` (default da coluna). Um valor
inválido retorna **422** com a lista de valores aceitos (não mais 500):

| Valor | Significado |
|---|---|
| `FREE` | Livre — sem análise/bloqueio |
| `BLOCK_ALWAYS` | Bloqueia sempre |
| `ALWAYS_ANALYZE` | Sempre passa por análise |

Exemplo de corpo do `POST /create`:

```json
{
  "code": 10,
  "description": "Divisão Interna",
  "commercial_analysis": "ALWAYS_ANALYZE",
  "financial_analysis": "FREE",
  "consider_mrp": true
}
```

---

## 3. Precificação (`/api/customers/sales-tables`)

O módulo de precificação mantém tabelas comerciais de venda, preços por item,
políticas de formação de preço, cálculo de preço sugerido e histórico de
reprecificação. A implementação usa os cadastros comerciais abaixo:

- `sales_tables`: cabeçalho da tabela de vendas, com vigência, formação de preço,
  tolerância, composição, tipo e casas decimais.
- `sales_table_prices`: preço por item dentro da tabela, com UME/UMC, situação,
  bloqueio, fórmula e observação.
- `sales_price_policies`: política persistente de formação de preço, com
  prioridade/sequência, escopo operacional, tipos de regra, fonte de custo,
  margem mínima/máxima/ideal, incidências em JSON, vigência e tabela padrão.
- `sales_table_price_history`: histórico de alteração/reprecificação de preços.

| Método | Rota | Ação |
|---|---|---|
| POST | `/api/customers/sales-tables` | Cria tabela de vendas |
| GET | `/api/customers/sales-tables` | Lista tabelas |
| GET | `/api/customers/sales-tables/{tableCode}` | Consulta tabela por código |
| PUT | `/api/customers/sales-tables/{tableCode}` | Atualiza tabela por código |
| POST | `/api/customers/sales-tables/{tableCode}/prices` | Inclui preço na tabela |
| GET | `/api/customers/sales-tables/{tableCode}/prices` | Lista preços da tabela |
| GET | `/api/customers/sales-tables/{tableCode}/prices/{itemCode}` | Consulta preço do item |
| PUT | `/api/customers/sales-tables/prices` | Atualiza preço por ID |
| DELETE | `/api/customers/sales-tables/prices/{id}` | Remove preço |
| POST | `/api/customers/sales-tables/pricing` | Resolve preço de venda por tabela/item |
| POST | `/api/customers/sales-tables/price-formation` | Calcula preço sugerido por custo/markup/margem |
| POST | `/api/customers/sales-tables/generate-prices` | Reprecifica itens da tabela por política |
| GET | `/api/customers/sales-tables/{tableCode}/price-history` | Histórico da tabela, filtrável por `item_code` |
| POST | `/api/customers/sales-price-policies` | Cria política de formação de preço |
| GET | `/api/customers/sales-price-policies` | Lista políticas |
| GET | `/api/customers/sales-price-policies/{code}` | Consulta política |
| PUT | `/api/customers/sales-price-policies/{code}` | Atualiza política |

`POST /pricing` valida tabela ativa/vigente, preço não bloqueado e situação
diferente de `INATIVO`; retorna preço unitário, quantidade e total bruto.
`POST /price-formation` calcula preço sugerido por:

```text
preco = custo_independente / (1 - ((percentual_despesas_venda + percentual_lucro) / 100))
```

No ERP, `percentual_lucro` é `margin_pct` ou `ideal_margin_pct` da política, e
`percentual_despesas_venda` é a soma de `expenses_pct`, `taxes_pct`,
`freight_pct`, `commission_pct` e `discount_pct`. As casas decimais da tabela são
respeitadas quando informadas.

Na manutenção manual de preços, tabelas com formação `CUSTO_MEDIO`,
`CUSTO_STANDARD_TOTAL` ou `CUSTO_STANDARD_MATERIAL` não aceitam preço digitado.
Também é bloqueado preço menor que `0.01` quando a tabela não permite itens abaixo
de um centavo.

`POST /generate-prices` usa a política para buscar o custo do item e gravar/upsertar
o preço na tabela, mantendo histórico. A política usa `priority`/`sequence` para
ordenar regras comerciais: menor prioridade tem precedência e sequências da mesma
prioridade permitem organizar incidências acumuláveis. Nesta fase, `incidences_json`
guarda as incidências estruturadas para evolução da fase 2. Fontes de custo aceitas:

| Fonte | Origem |
|---|---|
| `INFORMED` | custo informado no cálculo manual |
| `STANDARD_TOTAL` | `item_standard_costs.total_cost` |
| `STANDARD_MATERIAL` | `item_standard_costs.material_cost` |
| `PURCHASE` | `item_purchase_costs.unit_cost` |
| `STOCK_AVG` | `stock_balances.avg_cost` |
| `STOCK_LAST` | `stock_balances.last_cost` |

Teste automatizado: `scripts/test-comercial-pricing.sh` roda os testes unitários e,
com `BASE_URL` definido, faz smoke HTTP de criação de tabela, preço, política,
resolução e formação.

### Política comercial (`/api/customers/support/commercial-policies`)

O motor de política comercial centraliza descontos, acréscimos, fretes e comissões
em uma única estrutura de regras. Cada política possui:

- `kind`: `DISCOUNT`, `SURCHARGE`, `FREIGHT` ou `COMMISSION`;
- `choice_type`: `INFORMATION`, `CHOICE` ou `OPTIONAL`;
- `calc_type`: `PERCENT` ou `VALUE`;
- valor percentual/fixo, limites máximos, faixas de valor bruto e quantidade;
- prioridade e sequência para definir ordem de aplicação;
- indicador de acumulação (`stackable`), possibilidade de edição manual,
  autorização para valores maiores, uso na base de comissão, aplicação por item e
  necessidade de aprovação;
- `data_types_json` com até seis dimensões comerciais combináveis por política
  (cliente, item, classificação, tabela, condição, prazo, representante, UF etc.);
- filtros por cliente, tipo de cliente, segmento, região, tabela de vendas,
  condição de pagamento, transportadora, item, máscara, linha de produto e
  classificação;
- `rule_json` para regras estruturadas adicionais usadas por configuradores e
  automações comerciais.

As linhas da política (`/{code}/lines`) representam as faixas/regras efetivas:
número da linha, sequência, vigência própria, variáveis da combinação, tipo
percentual/valor, valor mínimo e valor máximo. Quando uma política possui linhas,
o motor usa a primeira linha válida; sem linhas, usa o valor do cabeçalho como
fallback operacional.

Endpoints:

| Método | Rota | Ação |
|---|---|---|
| POST | `/` | Cria política comercial |
| GET | `/` | Lista políticas; aceita `kind` e `only_active`; aceita exportação |
| GET | `/{code}` | Consulta política |
| PUT | `/{code}` | Atualiza política |
| POST | `/evaluate` | Simula/apura políticas aplicáveis para um contexto de venda |
| POST | `/{code}/lines` | Cria linha/faixa de regra da política |
| GET | `/{code}/lines` | Lista linhas/faixas da política |
| POST | `/{code}/specific-items` | Vincula exceção por item/linha/classificação |
| GET | `/{code}/specific-items` | Lista vínculos específicos |

A avaliação recebe valor bruto, quantidade e os atributos comerciais do contexto.
O resultado retorna totais separados (`discount_value`, `surcharge_value`,
`freight_value`, `commission_value`), valor líquido, flag de aprovação e a lista
das políticas aplicadas. Políticas não acumuláveis impedem novas regras do mesmo
tipo depois da primeira aplicação efetiva.

Persistência: migration `000187_commercial_policies` cria
`commercial_policies`, `commercial_policy_lines` e
`commercial_policy_specific_items`. O cadastro de itens/classificações específicos
permite bloquear política de desconto de capa, acréscimo de capa, políticas do
nível do item e alteração manual por item ou classificação.

Teste automatizado: `scripts/test-comercial-politicas.sh` cobre o motor de domínio
e, com `BASE_URL` definido, faz smoke HTTP de cadastro, vínculo específico,
avaliação e listagem.

---

## 4. Promessa de Entrega

Cálculo de data prometida com base em disponibilidade (estoque + capacidade).

### Parâmetros (`/api/delivery-promise-params`)
| Método | Rota | Ação |
|---|---|---|
| GET | `/` | Lê os parâmetros |
| PUT | `/update` | Atualiza os parâmetros |

> ℹ️ Se os parâmetros ainda não foram configurados, o `GET` retorna **404
> `not configured`** — é o estado "ainda não configurado", não um erro. Configure
> com o `PUT /update` antes de usar o cálculo de promessa.

### Calendário de promessa por item (`/api/item-calendar-promise`)
Disponibilidade (ATP) por item/variante, dia a dia.

| Método | Rota | Ação |
|---|---|---|
| POST | `/create` | Upsert de um dia |
| GET | `/{item_code}/{mask}/{year}/{month}` | Lista o mês |
| GET | `/{item_code}/{mask}/{year}/{month}/workdays` | Dias úteis |
| GET | `/{item_code}/{mask}/{year}/{month}/{day}` | Consulta um dia |
| DELETE | `/{item_code}/{mask}/{year}/{month}/{day}` | Remove um dia |

---

## 5. Reprogramação de Entrega (`/api/delivery-reschedule`)

Histórico de remarcações de data vinculado ao pedido (data original × nova × motivo).

| Método | Rota | Ação |
|---|---|---|
| POST | `/create` | Registra a reprogramação |
| GET | `/list/{sales_order_code}` | Lista as reprogramações do pedido |

---

## 6. Expedição / Romaneio (`/api/shipments`) — migration 000146

| Método | Rota | Ação |
|---|---|---|
| POST | `/` | Cria romaneio |
| GET | `/` | Lista |
| GET | `/{code}` | Consulta |
| POST | `/{code}/items` | Adiciona item |
| POST | `/items/confer` | Confere um item |
| POST | `/{code}/confer` | Confere o romaneio |
| POST | `/{code}/ship` | Despacha (exige tudo conferido) |
| POST | `/{code}/cancel` | Cancela |

**Status:** `OPEN` → `SEPARATED` → `CONFERRED` → `SHIPPED` (`CANCELLED`).
