# Vendas e Expedição — Documentação técnica

Cobre Pedido de Venda, Divisão de Vendas, Promessa de Entrega, Reprogramação de
Entrega e Expedição (romaneio). A versão de negócio está em
[`../apresentacao/vendas.md`](../apresentacao/vendas.md). Detalhe aprofundado do
Pedido de Venda também em [`visao-geral.md`](visao-geral.md) §4.

> Convenções: `Authorization: Bearer <JWT>`, `Content-Type: application/json`.
> Salvo indicação, todas as rotas exigem papel `ADMIN` ou `USER`.

---

## 1. Pedido de Venda (`/api/sales-order`)

O módulo de Pedido de Venda controla o compromisso comercial firmado com o
cliente. Ele recebe dados comerciais, financeiros, fiscais, logísticos e de
planejamento, e é o ponto de partida para crédito, reserva de estoque, MRP,
expedição e faturamento.

Use o pedido quando a proposta já foi aceita ou quando uma venda precisa entrar no
fluxo operacional. Antes da confirmação, o pedido pode ficar em rascunho, análise
comercial/financeira, bloqueio ou conferência. Depois da confirmação, a alteração
para status `P` alimenta demanda de planejamento e reserva ATP; depois do
faturamento, o pedido passa para `F`.

### Onde É Usado

- Comercial: registro da venda, negociação final, análise de bloqueios,
  acompanhamento da carteira e cancelamento/atendimento.
- Financeiro: análise de crédito, liberação financeira, condição de pagamento,
  portador e exposição do cliente.
- Logística/expedição: conferência do pedido, datas de entrega, transportadora,
  frete, volume, peso, lote e integração com romaneio.
- Fiscal/faturamento: tipo de nota, NFC-e, dados do consumidor, impostos
  informativos do item e emissão de NF-e/NFC-e no fluxo fiscal.
- Planejamento: confirmação do pedido gera demanda independente por item e pode
  acionar MRP/APS.

### Conceitos Principais

- **Capa do pedido:** empresa, número sequencial, status, origem, cliente,
  representante, divisão comercial, datas, condição de pagamento, tabela de
  preço, comissão, dados fiscais, transportadora, frete, volumes, projeto e
  observações.
- **Itens:** produto, máscara, quantidade solicitada, quantidade atendida,
  quantidade cancelada, saldo, preço, desconto, impostos informativos, data de
  entrega, lote, entrega com cupom, pagamento no caixa, pesos e situação.
- **Análise comercial/financeira:** estados independentes que indicam se o pedido
  ainda precisa passar por revisão, se foi aprovado ou rejeitado.
- **Liberação:** controle operacional que diferencia pedido liberado, bloqueado
  ou liberado manualmente por alçada.
- **Conferência:** controle logístico para marcar o pedido como pendente,
  conferido ou divergente antes do faturamento/expedição.
- **Histórico:** análise, liberação, bloqueio, cancelamento, atendimento,
  conferência e motivo de atraso geram eventos em `sales_order_events`.

| Método | Rota | Ação |
|---|---|---|
| POST | `/create` | Cria a capa do pedido |
| GET | `/list` | Lista pedidos |
| GET | `/search` | Consulta carteira com filtros avançados |
| GET | `/report` | Relatório gerencial da carteira |
| GET | `/{code}` | Consulta por código |
| PUT | `/{code}` | Atualiza a capa |
| DELETE | `/{code}/cancel` | Cancela o pedido com motivo/complemento |
| POST | `/{code}/analyze` | Registra análise comercial ou financeira |
| POST | `/{code}/release` | Libera, libera manualmente ou bloqueia |
| POST | `/{code}/attend` | Registra atendimento manual |
| POST | `/{code}/conference` | Atualiza conferência logística |
| POST | `/{code}/delay-reason` | Registra motivo e ação para atraso |
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

**Análise comercial/financeira:** `NOT_ANALYZED`, `APPROVED`, `REJECTED`.

**Liberação:** `BLOCKED`, `MANUAL_RELEASED`, `RELEASED`.

**Conferência:** `PENDING`, `CONFERRED`, `DIVERGENT`.

**Datas.** `emission_date`/`delivery_date` (capa) e `delivery_date` (item) aceitam
`YYYY-MM-DD` ou ISO-8601 com hora; `emission_date` omitido assume **hoje** (não mais
`0001-01-01`). `enterprise_code` é obrigatório no `POST /create` (422 se ausente).

### Consulta E Relatório

`GET /api/sales-order/search` aceita filtros por `customer_code`,
`representative_code`, `payment_term_code`, `status`,
`commercial_analysis_status`, `financial_analysis_status`, `release_status`,
`conference_status`, `is_blocked`, `emission_from`, `emission_to`,
`delivery_from` e `delivery_to`.

`GET /api/sales-order/report` consolida total de pedidos, valor bruto, valor
líquido, pedidos abertos, confirmados, faturados, cancelados, bloqueados,
pendências de análise comercial/financeira, pendências de conferência e pedidos em
atraso. A consulta de atraso considera pedido com `delivery_date` menor que a data
atual e status diferente de `F`/`CANCELLED`.

### Regras Operacionais

O cancelamento exige motivo e mantém o pedido consultável para preservar histórico.
O atendimento manual registra motivo e data de atendimento e marca o pedido como
faturado/atendido no fluxo operacional. A conferência marca o resultado logístico;
quando houver divergência, o motivo deve ser informado pelo operador. Pedidos em
atraso podem receber motivo e ação planejada para acompanhamento da carteira.

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

### Testes

Teste automatizado focado: `scripts/test-comercial-pedido-venda.sh`. Com
`BASE_URL` e `TOKEN`, o script também executa smoke HTTP de criação, item,
consulta avançada, relatório, análise, liberação, conferência, motivo de atraso e
cancelamento.

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

## 3. Orçamentos (`/api/sales-quotation`)

O módulo de orçamentos registra a negociação comercial antes da emissão de um
pedido de venda. Ele deve ser usado quando a empresa precisa formalizar uma
proposta, simular condições comerciais, manter histórico de propostas perdidas ou
atendidas e converter somente propostas aprovadas em pedidos reais.

O orçamento não substitui o pedido de venda. Ele é uma etapa anterior: guarda a
intenção comercial, valores negociados, itens, datas, frete, descontos,
acréscimos, retenções, probabilidade de fechamento e bloqueios comerciais. Quando
o cliente aprova a proposta, a conversão cria um pedido de venda e o fluxo passa a
ser controlado pelo módulo de pedidos, onde entram crédito, reserva, ATP/MRP,
expedição e faturamento.

### Onde É Usado

- Equipe comercial: cadastro de propostas, renegociação, consulta de carteira e
  acompanhamento de oportunidades.
- Gestão comercial: análise de valor em aberto, valor ponderado por
  probabilidade, propostas canceladas, atendidas e expiradas.
- Backoffice: validação de condições comerciais, frete, comissão, ordem de compra
  e observações do cliente antes de virar pedido.
- Integração com pedidos: conversão do saldo aberto dos itens para pedido de
  venda, preservando rastreabilidade pelo campo `converted_sales_order_code`.

### Conceitos Principais

- **Capa do orçamento:** dados gerais da proposta, empresa, cliente, status,
  validade, ordem de compra, datas, representante, divisão comercial, tabela de
  preço, condição de pagamento, moeda, comissão, liberação comercial e totais.
- **Itens:** produtos negociados, quantidade solicitada, preço unitário,
  descontos, impostos informativos, data de entrega e saldo ainda não atendido.
- **Transporte:** transportadora, tipo de frete, verificação de frete, valor de
  frete, redespacho e seguro.
- **Valores comerciais:** desconto, acréscimo, retenções, total bruto, total
  líquido e valor ponderado pela probabilidade de fechamento.
- **Histórico operacional:** cancelamento, descancelamento, atendimento e
  conversão registram eventos para manter rastreabilidade da decisão comercial.
- **Anexos:** a estrutura de banco já prevê documentos vinculados ao orçamento,
  com limite de 10 MB por arquivo; os endpoints de upload/download ainda não
  foram expostos nesta fase.

### Rotas

| Método | Rota | Ação |
|---|---|---|
| POST | `/api/sales-quotation/create` | Cria a capa do orçamento |
| GET | `/api/sales-quotation/list` | Lista orçamentos, filtrável por `customer_code`, `status`, `from`, `to`, `purchase_order_number`, `freight_type` |
| GET | `/api/sales-quotation/report` | Consolida totais, status, retenções e valor ponderado por probabilidade |
| GET | `/api/sales-quotation/{code}` | Consulta orçamento com itens |
| PUT | `/api/sales-quotation/{code}` | Atualiza capa, validade, condições, transporte e valores comerciais |
| DELETE | `/api/sales-quotation/{code}/cancel` | Cancela o orçamento com motivo e complemento |
| POST | `/api/sales-quotation/{code}/uncancel` | Descancela orçamento mantendo histórico |
| POST | `/api/sales-quotation/{code}/attend` | Registra atendimento manual do orçamento com motivo/data |
| PATCH | `/api/sales-quotation/{code}/status` | Altera status |
| POST | `/api/sales-quotation/{code}/convert-to-order` | Converte saldo aberto para pedido de venda |
| POST | `/api/sales-quotation/items/create` | Inclui item |
| GET | `/api/sales-quotation/items/{code}` | Lista itens do orçamento |
| PUT | `/api/sales-quotation/items/{itemCode}` | Atualiza item, atendimento e cancelamento parcial |
| DELETE | `/api/sales-quotation/items/{itemCode}/cancel` | Cancela item |

### Ciclo De Vida

| Status | Uso |
|---|---|
| `R` | Rascunho em montagem, ainda sem compromisso comercial |
| `P` | Registro originado de canal externo ou venda prévia |
| `A` | Pedido em análise comercial/financeira |
| `OA` | Orçamento em análise comercial/financeira |
| `F` | Pedido confirmado no ERP |
| `OF` | Orçamento confirmado no ERP e pronto para negociação/conversão |
| `CANCELLED` | Proposta encerrada por perda, desistência ou erro operacional |
| `ATTENDED` | Orçamento atendido manualmente ou convertido em pedido |
| `EXPIRED` | Orçamento vencido por validade expirada |

**Tipos de orçamento:** `API_TERCEIROS`, `CONSULTA`, `FOCCOPORTAL`, `IMPORTADO`,
`NEGOCIACAO`, `VENDA`.

**Liberação:** `BLOCKED`, `MANUAL_RELEASED`, `RELEASED`.

**Status do item:** `OPEN`, `PARTIAL`, `DELIVERED`, `CANCELLED`.

### Regras De Negócio

O orçamento guarda validade (`valid_until`), data de digitação (`digit_date`),
ordem de compra, tipo, liberação comercial, probabilidade de fechamento
(`probability_pct`), comissão, NFC-e, endereço do consumidor, transportadora, tipo
de frete, verificação de frete, redespacho, seguro, descontos, acréscimos,
retenções, autorização de entrega, observações e vínculo com o pedido convertido
(`converted_sales_order_code`). Itens guardam quantidade solicitada, atendida e
cancelada, permitindo saldo aberto antes da conversão.

O cancelamento exige motivo e pode receber complemento. O registro permanece
consultável para preservar histórico comercial. O descancelamento reabre a
proposta e registra o motivo da reversão. O atendimento manual encerra a proposta
sem gerar pedido, útil quando a decisão comercial precisa ser registrada mesmo sem
conversão automática.

Na conversão, o sistema cria um pedido de venda com a numeração de pedido existente
e copia somente o saldo aberto dos itens ativos. Orçamentos cancelados, expirados,
atendidos, de tipo `CONSULTA`, bloqueados comercialmente ou já convertidos são
bloqueados. A confirmação operacional do pedido continua no fluxo de Pedido de
Venda, onde entram crédito, reserva/ATP e MRP.

### NFC-e No Orçamento

O campo `is_nfce` indica que a proposta deve ser tratada como venda destinada a
cupom fiscal eletrônico quando for convertida para pedido. Nesta fase, ele foi
implementado como atributo comercial/fiscal do orçamento:

- existe na tabela `sales_quotations`;
- entra nos DTOs de criação, atualização e resposta;
- é retornado nas consultas;
- é copiado para o pedido de venda no momento da conversão.

Esta fase não emite NFC-e e não autoriza documento fiscal. A emissão continua no
módulo fiscal/faturamento. Também não foi implementada nesta fase uma regra
automática de cálculo fiscal específica para NFC-e dentro do orçamento; o campo
prepara a intenção fiscal para o pedido/faturamento consumir depois.

### Relatórios E Consultas

A listagem permite localizar propostas por cliente, status, período de emissão,
ordem de compra e tipo de frete. O relatório consolida quantidade de orçamentos,
total bruto, total líquido, propostas abertas, atendidas, canceladas, expiradas,
retenções e valor ponderado pela probabilidade de fechamento.

### Testes

Teste automatizado: `scripts/test-comercial-orcamentos.sh`. Com `BASE_URL` e
`TOKEN`, o script também faz smoke HTTP de criação, inclusão de item, consulta,
relatório, cancelamento, descancelamento e atendimento.

---

## 4. Precificação (`/api/customers/sales-tables`)

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

Política comercial é o motor que transforma preço de tabela em condição comercial
negociada. Ela responde perguntas que não pertencem ao cadastro simples de preço:
quanto desconto pode ser concedido, quando aplicar acréscimo, qual frete comercial
deve entrar na venda, qual comissão futura será provisionada e quando a negociação
precisa de aprovação.

A política é usada em simulações, orçamentos, pedidos, análises comerciais,
comissionamento e relatórios gerenciais. Nesta fase o motor está disponível como
API independente; as próximas fases devem chamá-lo ao gravar orçamento/pedido para
persistir os efeitos calculados na transação.

#### Quando usar cada tipo

| Tipo | Uso principal | Exemplo |
|---|---|---|
| `DISCOUNT` | Reduzir o valor vendido por volume, cliente, segmento, campanha ou item | 8% para cliente estratégico comprando linha premium acima de 10 unidades |
| `SURCHARGE` | Acrescentar valor por condição comercial mais onerosa | 3% para venda com prazo longo ou lote especial |
| `FREIGHT` | Compor frete comercial sem depender só do fiscal/logístico | R$ 250 fixos para entrega em região remota |
| `COMMISSION` | Calcular comissão futura do representante/equipe | 5% sobre valor líquido da venda |

#### Estrutura da política

A política possui uma capa e linhas. A capa define identidade, abrangência,
prioridade e comportamento geral. As linhas representam as faixas aplicáveis dentro
da política.

Campos de capa mais importantes:

| Campo | Para que serve |
|---|---|
| `kind` | Define se a política é desconto, acréscimo, frete ou comissão |
| `choice_type` | Define se a condição é informativa, escolhível ou opcional na negociação |
| `priority` / `sequence` | Ordena a aplicação; menor prioridade vem antes e sequência desempata |
| `stackable` | Permite ou impede acumular políticas do mesmo tipo |
| `requires_approval` | Sinaliza que a venda precisa passar por liberação comercial |
| `allow_manual_change` | Indica se o usuário pode alterar o valor sugerido |
| `allow_higher_values` | Permite negociar valor maior que o calculado |
| `used_in_commission` | Permite usar a política na base de cálculo da comissão |
| `applies_to_items` | Indica que a regra deve ser avaliada também em nível de item |
| `subtract_commission_base` | Indica que o valor da política reduz a base de comissão |
| `commission_discount_mode` | Controla se o desconto de comissão é real ou nominal |
| `data_types_json` | Lista até seis dimensões comerciais que formam a chave da regra |
| `rule_json` | Guarda critérios estruturados complementares para automações |

As dimensões de `data_types_json` tornam a política combinatória sem criar uma
tabela nova para cada variação. Exemplos de dimensões: cliente, tipo de cliente,
segmento, região, tabela de venda, condição de pagamento, transportadora, item,
máscara, linha de produto e classificação.

#### Linhas e faixas

As linhas (`/{code}/lines`) são a parte efetiva do cálculo. Elas permitem criar
faixas por combinação comercial, vigência própria e limites de valor. Uma política
de desconto pode, por exemplo, ter três linhas para o mesmo cliente e item:

| Linha | Condição | Resultado |
|---|---|---|
| 1 | até R$ 10.000 | 3% |
| 2 | de R$ 10.000 a R$ 50.000 | 5% |
| 3 | acima de R$ 50.000 | 8% com aprovação |

No estado atual do motor, quando uma política possui linhas, a avaliação usa a
primeira linha ativa e vigente retornada por ordem de linha/sequência. Se a
política não possuir linhas, o cálculo usa o valor definido na capa como fallback.
Esse fallback evita bloquear operações simples e mantém compatibilidade com
políticas de baixa complexidade.

#### Itens e classificações específicas

O cadastro `/{code}/specific-items` controla exceções por item, máscara, linha de
produto ou classificação. Ele existe para resolver casos em que a política de capa
não deve valer integralmente para uma família de produtos.

Flags disponíveis:

| Flag | Efeito operacional |
|---|---|
| `block_discount` | Bloqueia desconto de capa para o item/classificação |
| `block_surcharge` | Bloqueia acréscimo de capa para o item/classificação |
| `ignore_item_policies` | Ignora políticas específicas do item |
| `block_manual_change` | Impede alteração manual da condição calculada |

Exemplo: uma campanha concede 10% para todo o segmento industrial, mas itens de
linha premium só podem receber preço de tabela. Registre a política de desconto na
capa e vincule a classificação premium com `block_discount=true` e
`block_manual_change=true`.

#### Ordem de aplicação

1. O chamador informa contexto: valor bruto, quantidade, cliente, tabela, condição,
   transportadora, item e demais atributos disponíveis.
2. O repositório busca políticas ativas por prioridade/sequência.
3. O domínio valida vigência, faixa de valor, faixa de quantidade e filtros.
4. Para cada política compatível, o motor calcula valor percentual ou fixo.
5. Descontos reduzem o líquido; acréscimos e fretes aumentam; comissões são
   provisionadas sem alterar o líquido.
6. Se a política não for acumulável, novas políticas do mesmo tipo são ignoradas
   após a primeira aplicação.
7. A resposta informa totais por tipo, valor líquido, necessidade de aprovação e a
   lista de efeitos aplicados.

#### Exemplos de configuração

**Desconto por volume**

- `kind=DISCOUNT`
- `choice_type=INFORMATION`
- `stackable=true`
- `data_types_json=["CUSTOMER","ITEM","PAYMENT_TERM"]`
- linhas por faixa de valor/quantidade

Uso: simular preço final em orçamento e pedido, evidenciando desconto aplicado e
se a negociação precisa de aprovação.

**Acréscimo por prazo**

- `kind=SURCHARGE`
- filtro por condição de pagamento
- `applies_on_net_value=true`
- linha percentual para prazo longo

Uso: compensar custo financeiro de venda parcelada ou condição especial.

**Frete comercial**

- `kind=FREIGHT`
- `calc_type=VALUE`
- filtro por região/transportadora

Uso: compor preço vendido com custo de entrega negociado antes da expedição.

**Comissão futura**

- `kind=COMMISSION`
- `used_in_commission=true`
- `commission_discount_mode=REAL`

Uso: prever comissão do representante e permitir relatórios de comissão futura.

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

#### Integração com outros fluxos

- **Precificação**: preço de tabela e política de formação definem preço base; a
  política comercial calcula a negociação sobre esse preço.
- **Orçamento**: deve chamar `/evaluate` para mostrar preço líquido, frete,
  descontos/acréscimos e necessidade de aprovação antes da conversão em pedido.
- **Pedido de venda**: deve reavaliar a política na gravação e na liberação
  comercial, evitando que uma condição expirada seja usada.
- **Representantes/metas**: comissão calculada alimenta comissão futura e análise
  de rentabilidade por representante.
- **Faturamento/expedição**: frete comercial calculado pode orientar o frete
  faturado e a composição final da nota, respeitando regras fiscais.

#### Validações e cuidados

- Percentuais, valores e faixas não aceitam números negativos.
- `max_gross_value` e `max_quantity` iguais a zero significam "sem limite máximo".
- `rule_json`, `data_types_json` e `variables_json` precisam ser JSON válido.
- Política inativa ou fora da vigência não é aplicada.
- Linha fora da vigência é ignorada.
- Política não acumulável bloqueia novas políticas do mesmo `kind` após aplicada.
- O motor retorna `requires_approval=true` quando qualquer política aplicada exigir
  aprovação; a decisão de bloquear avanço fica no fluxo chamador.

Persistência: migration `000187_commercial_policies` cria
`commercial_policies`, `commercial_policy_lines` e
`commercial_policy_specific_items`. O cadastro de itens/classificações específicos
permite bloquear política de desconto de capa, acréscimo de capa, políticas do
nível do item e alteração manual por item ou classificação.

Teste automatizado: `scripts/test-comercial-politicas.sh` cobre o motor de domínio
e, com `BASE_URL` definido, faz smoke HTTP de cadastro, vínculo específico,
avaliação e listagem.

---

## 4. Representantes

O módulo de representantes centraliza o cadastro de vendedores externos,
vendedores internos, gerentes comerciais e prepostos que participam da venda. Ele
existe para evitar que o representante seja tratado apenas como texto livre no
pedido: cada venda passa a apontar para um cadastro com documento, território,
empresa de atuação, comissão, dados de contato e histórico comercial.

### Rotas principais (`/api/representatives`)

| Método | Rota | Ação |
|---|---|---|
| POST | `/create` | Cria representante |
| GET | `/list` | Lista representantes com filtros |
| GET | `/{code}` | Consulta cadastro completo com pastas |
| PUT | `/{code}` | Atualiza dados principais |
| PATCH | `/{code}/block` | Bloqueia representante com motivo |
| PATCH | `/{code}/unblock` | Remove bloqueio |
| GET | `/report` | Relatório cadastral por representante, UF, região e status |
| GET | `/follow-up` | Ficha de acompanhamento comercial |

### Tipos de representantes

| Método | Rota | Ação |
|---|---|---|
| POST | `/types/` | Cria tipo |
| GET | `/types/` | Lista tipos |
| GET | `/types/{code}` | Consulta tipo |
| PUT | `/types/{code}` | Atualiza tipo |

O tipo possui `description`, `is_free` e `ignores_direct_billing`. O campo
`is_free` indica se o representante fica disponível para clientes sem restrição
de carteira. `ignores_direct_billing` permite separar operações de faturamento
direto em análises comerciais.

### Pastas do cadastro

| Rota | Uso |
|---|---|
| `/enterprises` | Empresas de atuação, comissão padrão, percentual e situação ativa/inativa |
| `/accounting` | Contas, centros de custo e histórico para comissões geradas ou estornadas |
| `/regions` | Regiões e microrregiões de atendimento |
| `/segments` | Segmentos de mercado por representante ou microrregião |
| `/sales-plans` | Planos comerciais usados pela equipe de venda |
| `/interests` | Classificações de itens de interesse do representante |
| `/phones` | Telefones com DDI, DDD, tipo e ranking |
| `/emails` | E-mails com ranking |
| `/correspondence-addresses` | Endereço de correspondência |
| `/contacts` | Contatos/prepostos do representante |

Ao cadastrar telefone ou e-mail, o sistema atualiza automaticamente o contato
principal do representante pelo menor ranking. Ao informar logradouro e número,
o endereço completo é montado quando não vier preenchido.

### Relatório cadastral

`GET /api/representatives/report` aceita:

- `codes=1,2,3`
- `description=texto`
- `type_code=10`
- `state=RS`
- `region_code=5`
- `active_status=ACTIVE|INACTIVE|ALL`
- `sort_by=CODE|NAME|STATE|REGION`
- `with_accounts=true`

O relatório retorna identificação, tipo, UF, cidade, contatos principais,
regiões, situação ativa/inativa, comissão da empresa e, quando solicitado,
contas contábeis de comissão gerada.

### Ficha de acompanhamento

`GET /api/representatives/follow-up` consolida a evolução comercial do
representante a partir de orçamentos e pedidos. Filtros:

- `representative_codes=1,2`
- `customer_codes=100,200`
- `from=2026-01-01`
- `to=2026-12-31`

O retorno mostra quantidade de clientes atendidos, orçamentos, pedidos, valor
orçado, valor vendido, ticket médio, base de comissão, comissão calculada,
últimas datas e detalhamento por cliente. Isso permite acompanhar carteira,
atividade comercial e geração futura de comissão sem planilhas paralelas.

### Persistência e validações

Migration `000190_sales_representatives` cria as tabelas
`representative_types`, `representatives` e as tabelas das pastas do cadastro.
O cadastro exige nome e documento, rejeita quantidade negativa de dispositivos e
normaliza UF em maiúsculas. O representante pode ser vinculado a um cliente e/ou
fornecedor existente sem duplicar esses cadastros.

Teste automatizado: `scripts/test-comercial-representantes.sh` cobre a camada Go
e, com `BASE_URL`/`TOKEN`, executa smoke HTTP de tipo, cadastro, pastas,
relatório e ficha de acompanhamento.

---

## 5. Metas de Vendas (`/api/sales-goals`)

O módulo de metas controla objetivos comerciais por período, representante,
grupo comercial e cliente. Ele permite definir metas por valor ou quantidade,
acompanhar realizado contra previsto e registrar saldos excedentes para o
período seguinte.

Use metas quando a gestão precisa acompanhar carteira e remuneração variável com
base em venda ou faturamento. A base `SALES` calcula realizado por pedidos de
venda dentro do período; a base `INVOICING` fica registrada para metas que serão
fechadas por faturamento conforme a integração fiscal evoluir.

### Onde É Usado

- Comercial: definição de metas mensais, semanais ou customizadas.
- Representantes: acompanhamento de desempenho por carteira e região.
- Gestão de vendas: relatório previsto x realizado, percentual de atingimento e
  bônus.
- Políticas comerciais: base para descontos, premiações e comissões futuras.
- Planejamento: comparação entre meta, previsão de vendas e pedidos efetivos.

### Rotas

| Método | Rota | Ação |
|---|---|---|
| POST | `/create` | Cria meta por representante e período |
| GET | `/list` | Lista metas por representante, período e base |
| GET | `/report` | Relatório previsto x realizado |
| GET | `/{code}` | Consulta meta com itens |
| PUT | `/{code}` | Atualiza meta |
| POST | `/periods/` | Cria período mensal, semanal ou customizado |
| GET | `/periods/` | Lista períodos |
| POST | `/items` | Adiciona meta por item, classificação ou grupo |
| POST | `/group-targets` | Cria/atualiza meta por grupo comercial |
| POST | `/group-customers` | Vincula cliente à meta do grupo |
| POST | `/balances` | Registra saldo excedente de meta |

### Conceitos

- **Período:** janela da meta, com tipo `MONTH`, `WEEK` ou `CUSTOM`, data inicial
  e data final.
- **Meta por representante:** cabeçalho por representante, período e base de
  análise (`SALES` ou `INVOICING`), com percentual de premiação.
- **Itens da meta:** cada linha deve apontar exatamente um alvo: item,
  classificação de item ou grupo de item. A linha pode ter quantidade, valor,
  unidade de venda e bônus.
- **Meta por grupo comercial:** define meta mínima, provável e ideal, cada uma
  com percentual de bônus.
- **Clientes do grupo:** detalha metas mínima, provável e ideal por cliente,
  opcionalmente vinculadas ao representante responsável.
- **Saldos:** registram excedentes quando a realização supera a meta ideal e
  podem ser considerados no período subsequente.

### Relatório

`GET /api/sales-goals/report` aceita:

- `representative_code`
- `customer_code`
- `region_code`
- `microregion_code`
- `period_code`
- `from=YYYY-MM-DD`
- `to=YYYY-MM-DD`
- `analysis_base=SALES|INVOICING`
- `layout`
- `break_by`
- `include_missed_items=true`

O retorno mostra escopo da meta, representante ou grupo comercial, período, base
de análise, valor/quantidade prevista, valor/quantidade realizada, saldo,
percentual de atingimento, bônus e situação (`OPEN`, `ACHIEVED` ou `NO_TARGET`).

### Persistência e validações

Migration `000191_sales_goals` cria `sales_goal_periods`, `sales_goals`,
`sales_goal_items`, `sales_goal_group_targets`, `sales_goal_group_customers` e
`sales_goal_balances`. As validações impedem período invertido, percentuais
negativos e linhas de meta com mais de um alvo informado. O relatório respeita
filtros de representante, cliente, região, microrregião e período.

Teste automatizado: `scripts/test-comercial-metas.sh` cobre a camada Go,
validação estática de migração/rotas e, com `BASE_URL`/`TOKEN`, executa smoke HTTP
de período, meta, item, grupo, cliente, saldo e relatório.

---

## 6. Promessa de Entrega

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

## 7. Reprogramação de Entrega (`/api/delivery-reschedule`)

Histórico de remarcações de data vinculado ao pedido (data original × nova × motivo).

| Método | Rota | Ação |
|---|---|---|
| POST | `/create` | Registra a reprogramação |
| GET | `/list/{sales_order_code}` | Lista as reprogramações do pedido |

---

## 8. Expedição / Romaneio (`/api/shipments`) — migration 000146

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
