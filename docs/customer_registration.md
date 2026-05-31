# Cadastro de Cliente

Este documento descreve o processo completo para cadastrar um cliente no sistema.

---

## Estrutura de Rotas

Todas as rotas do módulo de cliente vivem sob `/api/customers`:

```
/api/customers/support/*   → Cadastros de apoio (pré-requisitos)
/api/customers/*           → Cadastro e gestão de clientes
```

---

## 1. Cadastros de Apoio (`/api/customers/support/`)

Esses cadastros devem existir antes de criar um cliente.

### 1.1 Região (`/api/customers/support/regions`)

Define a área geográfica de atuação (território de vendas/representantes).
**UF e City são obrigatórios.**

```http
POST /api/customers/support/regions
{
  "description": "Sul — SC Litoral",
  "uf": "SC",
  "city": "Florianópolis",
  "created_by": "<uuid>"
}

PUT  /api/customers/support/regions        → atualiza
GET  /api/customers/support/regions        → lista
GET  /api/customers/support/regions/{code} → busca por código
```

| Campo | Descrição |
|---|---|
| `description` | Nome da região de vendas (ex: "Sul — SC Litoral") |
| `uf` | Sigla do estado (obrigatório) — define a UF principal da região |
| `city` | Cidade de referência da região (obrigatório) |
| `created_by` | UUID do usuário que está criando o registro |

---

### 1.2 Segmento de Mercado (`/api/customers/support/market-segments`)

Classifica o mercado do cliente. Suporta hierarquia pai-filho e retenção PIS/COFINS.

```http
POST /api/customers/support/market-segments
{
  "description": "Indústria Alimentícia",
  "parent_code": null,
  "has_pis_cofins_retention": false,
  "retention_indicator": null
}
```

| Campo | Descrição |
|---|---|
| `description` | Nome do segmento de mercado |
| `parent_code` | Código do segmento pai — permite hierarquia (ex: "Indústria" → "Indústria Alimentícia") |
| `has_pis_cofins_retention` | `true` indica que clientes desse segmento retêm PIS/COFINS na fonte |
| `retention_indicator` | Percentual ou código de retenção — **obrigatório** quando `has_pis_cofins_retention = true` |

---

### 1.3 Tipo de Contato (`/api/customers/support/contact-types`)

Categorias de contatos (Comprador, Gerente, Diretor, etc.).

```http
POST /api/customers/support/contact-types
{ "description": "Comprador" }
```

| Campo | Descrição |
|---|---|
| `description` | Nome do tipo de contato (ex: "Comprador", "Diretor Financeiro") |

---

### 1.4 Tipo de Cliente (`/api/customers/support/customer-types`)

Classifica o cliente e define dias de entrega padrão.

```http
POST /api/customers/support/customer-types
{
  "code": 1,
  "description": "Indústria",
  "category": "NORMAL",
  "delivery_days": 5
}
```

| Campo | Valores / Descrição |
|---|---|
| `description` | Nome do tipo de cliente |
| `category` | `NORMAL` — cliente comum; `CONSUMIDOR` — consumidor final (afeta tributação ICMS/DIFAL) |
| `delivery_days` | Prazo padrão de entrega em dias corridos para este tipo de cliente |

---

### 1.5 Portador (`/api/customers/support/carriers`)

Instituição financeira ou modalidade de cobrança usada para geração de títulos (banco, boleto, carteira, etc.).

```http
POST /api/customers/support/carriers
{
  "description": "Banco do Brasil — Boleto",
  "billing_type": "BOLETO",
  "uses_credit_limit": false,
  "consider_available": true,
  "postpone_due_date": false,
  "receipt_days": 3,
  "payment_days": 1
}
```

| Campo | Descrição |
|---|---|
| `description` | Nome do portador (ex: "Banco do Brasil — Boleto") |
| `billing_type` | `CARTEIRA` — cobrança manual/interna; `COBRANCA_ESCRITURAL` — cobrança bancária escritural; `BOLETO` — boleto bancário registrado |
| `uses_credit_limit` | `true` → ao usar este portador em um pedido de venda, o sistema verifica o limite de crédito do cliente antes de aprovar. `false` → cobrança bypassa a análise de crédito |
| `consider_available` | `true` → a análise de crédito usa o saldo disponível (limite − pedidos em aberto); `false` → usa o limite total bruto do cliente |
| `postpone_due_date` | `true` → se a data de vencimento calculada cair em fim de semana ou feriado, é adiada para o próximo dia útil (comportamento padrão de boleto bancário no Brasil) |
| `receipt_days` | Dias após o vencimento que o banco leva para confirmar o recebimento (usado na projeção de fluxo de caixa) |
| `payment_days` | Dias de compensação do pagamento (prazo de liquidação/crédito em conta) |

---

### 1.6 Grupo de Portadores (`/api/customers/support/carrier-groups`)

Agrupa portadores para facilitar a associação em condições de pagamento.

```http
POST /api/customers/support/carrier-groups
{ "description": "Portadores Principais" }

POST /api/customers/support/carrier-groups/members
{
  "carrier_group_code": 1,
  "carrier_code": 1
}
```

| Campo | Descrição |
|---|---|
| `description` | Nome do grupo de portadores |
| `carrier_group_code` | Código do grupo ao qual o portador será vinculado |
| `carrier_code` | Código do portador a ser adicionado ao grupo |

---

### 1.7 Condição de Pagamento (`/api/customers/support/payment-conditions`)

Define como o cliente paga (à vista, parcelado, boleto, etc.). Este é o cadastro de condição
de pagamento **comercial do cliente**, conectado ao pedido de venda via `payment_term_code`.

> **Diferença do módulo financeiro:** o módulo financeiro possui sua própria "condição de
> pagamento" (usada em contas a pagar/receber). As duas coexistem com propósitos distintos:
> a do cliente define o prazo comercial negociado; a financeira comanda a geração de títulos.

```http
POST /api/customers/support/payment-conditions
{
  "description": "30/60/90 dias DD",
  "carrier_code": 1,
  "analysis_type": "SEMPRE_ANALISA",
  "parcel_start": "EMISSAO",
  "expenses": 0,
  "average_term": 60,
  "is_special": false,
  "is_revenue": true,
  "is_at_sight": false
}
```

| Campo | Descrição |
|---|---|
| `description` | Nome da condição de pagamento (ex: "30/60/90 dias DD") |
| `carrier_code` | Portador padrão associado a esta condição (banco/modalidade de cobrança) |
| `analysis_type` | `SEMPRE_ANALISA` — sempre passa por análise de crédito; `BLOQUEIA_SEMPRE` — bloqueia o pedido independentemente do crédito; `LIBERA_SEM_ANALISE` — aprova automaticamente sem análise |
| `parcel_start` | Quando começa a contagem do primeiro vencimento: `EMISSAO` — a partir da data de emissão da NF; `PROXIMO_MES` — no 1º dia do mês seguinte; `PROXIMA_QUINZENA` — na próxima quinzena |
| `expenses` | Valor de despesas/taxas adicionais cobradas nesta condição (ex: tarifa bancária) |
| `average_term` | Prazo médio ponderado em dias (usado em relatórios de projeção de recebíveis) |
| `is_special` | `true` → condição negociada/excepcional, fora da política comercial padrão; restringe quem pode aplicá-la no pedido (ex: somente gerente) |
| `is_revenue` | `true` → pedidos com esta condição geram título financeiro (conta a receber) no módulo financeiro; `false` → para amostras grátis, devoluções internas ou operações sem receita |
| `is_at_sight` | `true` → pagamento à vista; o vencimento das parcelas deve ser 0 dias; flag semântico usado pelo pedido de venda para tratamento diferenciado |

Para adicionar parcelas:

```http
POST /api/customers/support/payment-conditions/installments
{
  "payment_condition_code": 1,
  "installment_number": 1,
  "due_days": 30,
  "description": "1ª Parcela",
  "document_type": "DUPLICATA",
  "movement_type": null,
  "carrier_code": 1
}
```

| Campo | Descrição |
|---|---|
| `payment_condition_code` | Código da condição de pagamento pai |
| `installment_number` | Número sequencial da parcela (1, 2, 3...) |
| `due_days` | Dias a partir do `parcel_start` para vencimento desta parcela |
| `description` | Rótulo da parcela (ex: "1ª Parcela", "Entrada") |
| `document_type` | Tipo de documento gerado: `DUPLICATA`, `CHEQUE`, `PROMISSORIA`, etc. |
| `movement_type` | Tipo de movimento financeiro (opcional — herda do portador se nulo) |
| `carrier_code` | Portador específico desta parcela (pode diferir do portador da condição pai) |

**Conexão com pedido de venda:** `sales_orders.payment_term_code` referencia `payment_conditions.code` (FK via migration 000120).

---

### 1.8 Tabela de Vendas (`/api/customers/support/sales-tables`)

Tabela de preços aplicada ao cliente. Conectada ao pedido de venda via `price_table_code`.

```http
POST /api/customers/support/sales-tables
{
  "description": "Tabela Indústria 2025",
  "validity_start": "2025-01-01T00:00:00Z",
  "validity_end": null,
  "tolerance_min_pct": 0,
  "tolerance_max_pct": 5,
  "price_formation": "INFORMADO",
  "decimal_places": 2,
  "composition": "FOB",
  "table_type": "NORMAL",
  "base_date": "PEDIDO",
  "allow_items_below_cent": false,
  "icms_interestadual_por_dentro": false,
  "observation": null
}
```

| Campo | Descrição |
|---|---|
| `description` | Nome da tabela de preços |
| `validity_start` | Data de início da vigência da tabela |
| `validity_end` | Data de fim da vigência (null = sem prazo de expiração) |
| `tolerance_min_pct` | Desconto máximo permitido (%) no pedido de venda; se o preço unitário ficar abaixo deste piso, o pedido é bloqueado |
| `tolerance_max_pct` | Acréscimo máximo permitido (%) acima do preço tabelado; valor acima trava o pedido |
| `price_formation` | Como o preço é formado: `INFORMADO` — digitado manualmente; `CUSTO_MEDIO` — calculado pelo custo médio do item; `CUSTO_STANDARD_TOTAL` / `CUSTO_STANDARD_MATERIAL` — custo padrão; `INFORMADO_SEM_ICMS` — preço sem ICMS embutido; `MAT_OPER` — material + operação (MRP); `TABELA_CUSTO` — tabela de custo específica; `TRANSFERENCIA_IPI` — base de cálculo do IPI em transferências entre empresas; `TRANSFERENCIA_UF` — valor por última entrada (transferências interestaduais) |
| `decimal_places` | Casas decimais para arredondamento dos preços (padrão: 2) |
| `composition` | Incoterm para exportação: `EXWORK` — cliente retira na fábrica; `CIF` — frete e seguro embutidos no preço; `FOB` — frete nacional até o porto/aeroporto |
| `table_type` | `NORMAL` — tabela padrão; `PROMOCIONAL` — tabela promocional (campo valor fica em vermelho no pedido) |
| `base_date` | Data usada para cálculo do preço: `PEDIDO` — data de emissão do pedido (padrão); `DATA_ATUAL` — data do sistema no momento do cálculo. Só se aplica quando `price_formation = CUSTO_MEDIO` |
| `allow_items_below_cent` | `true` → permite cadastrar itens com valor menor que R$0,01. Ao marcar, o sistema avisa sobre itens existentes com preço zero |
| `icms_interestadual_por_dentro` | `true` → inclui ICMS por dentro no valor do produto encontrado pela última entrada (só para `price_formation = TRANSFERENCIA_UF`) |
| `observation` | Observação livre sobre a tabela |

---

### 1.9 Tipo de Nota Fiscal de Saída (`/api/customers/support/invoice-types`)

Configura o comportamento fiscal das NF-e emitidas para o cliente (quais impostos calcular,
se gera receita, se atualiza estoque, etc.). Armazenado como preferência padrão no cliente;
o pedido de venda herda via `default_nf_type`.

> **Nota:** o cálculo efetivo dos impostos na NF-e (módulo fiscal) usa NCM, CFOP e alíquotas
> da tabela fiscal, não os percentuais aqui cadastrados diretamente. Este cadastro serve como
> configuração de comportamento (quais impostos calcular, se atualiza estoque, etc.).

```http
POST /api/customers/support/invoice-types
{
  "description": "Venda Normal — Indústria",
  "type": "VENDA",
  "stock_movement": "ATUALIZA",
  "icms_type": "TRIBUTADO",
  "icms_pct": 12,
  "icms_reduction_pct": 0,
  "ipi_pct": 0,
  "pis_pct": 1.65,
  "cofins_pct": 7.6,
  "issqn_pct": 0,
  "ir_pct": 0,
  "csll_pct": 0,
  "inss_pct": 0,
  "generates_revenue": true,
  "updates_inventory": true,
  "generates_financial_title": true,
  "considers_goals": true,
  "calc_substitution_tax": false,
  "calc_icms_deferral": false,
  "calc_pis_cofins": true,
  "calc_difal": false,
  "requires_sales_order": true,
  "lists_fiscal_books": true,
  "model_nf": "55",
  "cst_icms": "00",
  "csosn_icms": null,
  "cst_ipi": "50",
  "cst_pis": "01",
  "cst_cofins": "01",
  "baixa_pedido": true,
  "gera_titulo_dev": false,
  "exige_suframa": false,
  "ir_pct_presumption": 0,
  "csll_pct_presumption": 0
}
```

**Campos de comportamento geral:**

| Campo | Descrição |
|---|---|
| `description` | Nome do tipo de NF (ex: "Venda Normal — Indústria") |
| `type` | Natureza da operação: `VENDA`, `DEVOLUCAO`, `REMESSA`, `REMESSA_CONSIGNACAO`, `REMESSA_ARMAZENAGEM`, `REMESSA_BENEFICIAMENTO`, `RETORNO_BENEFICIAMENTO`, `SIMPLES_REMESSA`, `TRANSFERENCIA`, `VENDA_CONSIGNACAO`, `COMPLEMENTAR_ICM`, `COMPLEMENTAR_IPI`, `DEMONSTRACAO`, `EMPRESTIMO`, `FATURAMENTO_ANTECIPADO`, `PRESTACAO_SERVICOS`, `OUTROS` |
| `stock_movement` | `ATUALIZA` → baixa estoque; `NAO_ATUALIZA` → sem movimentação; `TRANSFERENCIA_EXTERNA` → obrigatório para remessa/consignação/armazenagem |
| `icms_type` | Situação tributária do ICMS: `TRIBUTADO`, `ISENTO`, `OUTROS` |
| `icms_pct` | Alíquota de referência do ICMS (% — o cálculo efetivo usa a tabela de Redução/Substituição por NCM/UF) |
| `icms_reduction_pct` | Percentual de redução da base de cálculo do ICMS |
| `ipi_pct` | Alíquota de referência do IPI (%) |
| `pis_pct` | Alíquota de referência do PIS (%) |
| `cofins_pct` | Alíquota de referência do COFINS (%) |
| `issqn_pct` | Alíquota do ISS/ISSQN (% — prestação de serviços) |
| `ir_pct` | Alíquota de retenção do IR na fonte (%) |
| `csll_pct` | Alíquota de retenção do CSLL na fonte (%) |
| `inss_pct` | Alíquota de retenção do INSS na fonte (%) |
| `generates_revenue` | `true` → operação gera receita (aparece no DRE e relatórios de vendas) |
| `updates_inventory` | `true` → emissão da NF atualiza saldo de estoque |
| `generates_financial_title` | `true` → emissão gera título a receber no módulo financeiro |
| `considers_goals` | `true` → venda contabilizada nas metas comerciais |
| `calc_substitution_tax` | `true` → habilita cálculo de ICMS-ST |
| `calc_icms_deferral` | `true` → habilita cálculo de diferimento do ICMS |
| `calc_pis_cofins` | `true` → habilita cálculo de PIS e COFINS na NF |
| `calc_difal` | `true` → habilita cálculo de DIFAL (venda interestadual para consumidor final) |
| `requires_sales_order` | `true` → NF só pode ser emitida via pedido de venda |
| `lists_fiscal_books` | `true` → operação aparece nos livros fiscais de saída |
| `baixa_pedido` | `true` → ao faturar, o item do pedido de venda é dado como atendido; `false` → para faturamento antecipado onde a saída física vem depois |
| `gera_titulo_dev` | `true` → devoluções com este tipo geram título no contas a pagar. Obriga `type = DEVOLUCAO` |
| `exige_suframa` | `true` → exige código SUFRAMA válido no cliente no momento da emissão (Zona Franca de Manaus) |

**Campos para FocusNFE (XML NF-e):**

| Campo | Descrição |
|---|---|
| `model_nf` | Modelo NF-e enviado ao SEFAZ via FocusNFE: `"55"` = NF-e (padrão); `"65"` = NFC-e (cupom fiscal eletrônico) |
| `cst_icms` | CST de ICMS enviado no XML (ex: `"00"` tributado integral, `"40"` isento, `"41"` não tributado, `"10"` com ST). Tabela SEFAZ |
| `csosn_icms` | CSOSN para empresas do Simples Nacional (ex: `"400"` não tributado, `"500"` ICMS ST cobrado anteriormente). Uso exclusivo do Simples — deixar null para Lucro Real/Presumido |
| `cst_ipi` | CST do IPI no XML (ex: `"50"` saída tributada, `"99"` outras saídas) |
| `cst_pis` | CST do PIS no XML (ex: `"01"` operação tributável alíquota básica, `"07"` operação isenta) |
| `cst_cofins` | CST do COFINS no XML — mesma estrutura do PIS |
| `ir_pct_presumption` | Percentual de presunção do IR (Lucro Presumido) — base de cálculo do IRPJ sobre receita bruta |
| `csll_pct_presumption` | Percentual de presunção da CSLL (Lucro Presumido) — base de cálculo da CSLL sobre receita bruta |

> **O que o FocusNFE faz automaticamente:** geração do XML, assinatura digital, transmissão ao SEFAZ, retorno da chave NF-e, geração do DANFE (PDF), cancelamento, carta de correção (CCe), contingência offline. **O que o ERP faz:** calcula os valores dos impostos, determina CFOP e CST com base neste cadastro + tabelas de Redução/Substituição de ICMS, monta o payload e envia para a API do FocusNFE.

> **Hierarquia de alíquotas:** os campos `*_pct` são alíquotas de referência. Para ICMS e IPI, o cálculo efetivo consulta primeiro o cadastro de Redução e Substituição de ICMS (por item/UF/cliente), depois a tabela fiscal (NCM), e só então recorre ao tipo de NF como fallback. Para PIS/COFINS, a hierarquia é: tipo de NF → divisão de vendas → classificação fiscal.

---

### 1.10 Tipo de Imposto (`/api/customers/support/tax-types`)

Define as regras de formação de base de cálculo para IPI, ICMS, PIS/COFINS, CSLL e IR.
Armazenado como preferência padrão no cliente; o pedido de venda herda via `tax_type_code`.

```http
POST /api/customers/support/tax-types
{
  "description": "Tributação Padrão Indústria",
  "ipi_base_total_items": true,
  "ipi_base_subtract_discount": true,
  "ipi_base_add_freight": false,
  "ipi_base_add_expenses": false,
  "icms_base_total_items": true,
  "icms_base_subtract_discount": true,
  "icms_base_add_freight": false,
  "icms_base_add_ipi": false,
  "icms_base_add_expenses": false,
  "pis_cofins_base_total_items": true,
  "pis_cofins_base_subtract_discount": true,
  "pis_cofins_base_add_freight": false,
  "pis_cofins_base_add_insurance": false,
  "pis_cofins_base_add_expenses": false,
  "csll_base_total_items": true,
  "csll_base_subtract_discount": true,
  "csll_base_add_freight": false,
  "ir_base_total_items": true,
  "ir_base_subtract_discount": true,
  "ir_base_add_freight": false,
  "is_consumer": false
}
```

Cada conjunto de flags controla a **composição da base de cálculo** de cada imposto:

| Padrão dos campos | Significado |
|---|---|
| `*_base_total_items` | Inclui o total dos itens (valor dos produtos) na base de cálculo |
| `*_base_subtract_discount` | Subtrai os descontos comerciais da base de cálculo |
| `*_base_add_freight` | Soma o valor do frete à base de cálculo |
| `*_base_add_expenses` | Soma despesas acessórias à base de cálculo |
| `icms_base_add_ipi` | Soma o valor do IPI à base de cálculo do ICMS (operações industriais) |
| `pis_cofins_base_add_insurance` | Soma o valor do seguro à base de PIS/COFINS |
| `is_consumer` | `true` → cliente é consumidor final; altera regras de ICMS e habilita DIFAL automaticamente |

---

## 2. Cadastro do Cliente (`/api/customers`)

Com todos os cadastros de apoio prontos, crie o cliente:

```http
POST /api/customers
{
  "code": 1001,
  "corporate_code": null,
  "is_corporate": false,
  "name": "Empresa Exemplo Ltda",
  "trade_name": "Exemplo",
  "document_type": "CNPJ",
  "document_number": "12.345.678/0001-90",
  "state_registration": "123456789",
  "municipal_registration": null,
  "suframa_code": null,
  "suframa_expiry": null,
  "region_code": 1,
  "market_segment_code": 1,
  "customer_type_code": 1,
  "payment_condition_code": 1,
  "sales_table_code": 1,
  "carrier_code": 1,
  "carrier_group_code": 1,
  "invoice_type_code": 1,
  "tax_type_code": 1,
  "payment_cond_visibility": "SOMENTE_VINCULADOS",
  "credit_limit": 50000,
  "website": "https://exemplo.com.br",
  "created_by": "<uuid>"
}
```

| Campo | Descrição |
|---|---|
| `code` | Código único do cliente no sistema |
| `corporate_code` | Código da matriz — preencher quando este cliente é uma filial de um cliente corporativo |
| `is_corporate` | `true` → este cliente é uma matriz (empresa mãe); habilita consulta de filiais via `/establishments` |
| `name` | Razão social (para PJ) ou nome completo (para PF) |
| `trade_name` | Nome fantasia (apelido comercial) |
| `document_type` | `CNPJ` — empresa; `CPF` — pessoa física; `ESTRANGEIRO` — cliente do exterior; `ISENTO` — isento de inscrição |
| `document_number` | Número do CNPJ ou CPF com pontuação |
| `state_registration` | Inscrição Estadual (IE) — obrigatória para contribuintes de ICMS |
| `municipal_registration` | Inscrição Municipal (IM) — obrigatória para prestadores de serviços (ISSQN) |
| `suframa_code` | Código SUFRAMA — para clientes da Zona Franca de Manaus com benefícios fiscais |
| `suframa_expiry` | Data de validade do cadastro SUFRAMA |
| `region_code` | Região de vendas do cliente (define território do representante) |
| `market_segment_code` | Segmento de mercado (classifica o tipo de negócio do cliente) |
| `customer_type_code` | Tipo de cliente (define categoria e prazo de entrega padrão) |
| `payment_condition_code` | Condição de pagamento padrão do cliente (herdada pelo pedido de venda) |
| `sales_table_code` | Tabela de preços padrão do cliente (herdada pelo pedido de venda) |
| `carrier_code` | Portador padrão do cliente para cobrança |
| `carrier_group_code` | Grupo de portadores disponíveis para seleção no pedido de venda |
| `invoice_type_code` | Tipo de NF padrão — preferência copiada para o pedido de venda na criação |
| `tax_type_code` | Tipo de imposto padrão — define composição de base de cálculo herdada pelo pedido |
| `payment_cond_visibility` | `SOMENTE_VINCULADOS` — só exibe condições vinculadas ao grupo do cliente; `VINCULADOS_E_NENHUM` — inclui opção "nenhuma condição"; `TODOS` — exibe todas as condições cadastradas |
| `credit_limit` | Limite de crédito do cliente em reais — verificado quando o portador do pedido tem `uses_credit_limit = true` |
| `website` | Site do cliente (informativo) |

---

### 2.1 Cliente Corporativo (Matriz / Filiais)

- Crie a **matriz** com `is_corporate: true`
- Crie cada **filial** com `corporate_code` igual ao `code` da matriz

```http
GET /api/customers/{corporateCode}/establishments
```

---

### 2.2 Endereços

```http
POST /api/customers/{code}/addresses
{
  "customer_code": 1001,
  "address_type": "COBRANCA",
  "zip_code": "88010-000",
  "street": "Rua Exemplo",
  "number": "100",
  "complement": "Sala 1",
  "neighborhood": "Centro",
  "city": "Florianópolis",
  "uf": "SC",
  "country": "Brasil",
  "is_default": true
}
```

| Campo | Descrição |
|---|---|
| `address_type` | `COBRANCA` — endereço de cobrança/fatura; `ENTREGA` — endereço de entrega da mercadoria; `COMERCIAL` — endereço comercial/sede; `OUTRO` — uso geral |
| `zip_code` | CEP no formato "00000-000" |
| `is_default` | `true` → endereço padrão para este tipo; apenas um por tipo pode ser padrão |

---

### 2.3 Contatos

```http
POST /api/customers/{code}/contacts
{
  "customer_code": 1001,
  "contact_type_code": 1,
  "name": "João Silva",
  "email": "joao@exemplo.com.br",
  "phone": "(48) 3333-4444",
  "mobile": "(48) 99999-0000",
  "position": "Comprador",
  "is_primary": true
}
```

| Campo | Descrição |
|---|---|
| `contact_type_code` | Tipo de contato cadastrado em `/support/contact-types` |
| `is_primary` | `true` → contato principal do cliente (usado como padrão em comunicações) |
| `position` | Cargo/função do contato na empresa |

---

## 3. Bloqueio e Desbloqueio

```http
PATCH /api/customers/{code}/block
{ "customer_code": 1001, "reason": "Inadimplência" }

PATCH /api/customers/{code}/unblock
```

| Campo | Descrição |
|---|---|
| `reason` | Motivo do bloqueio — registrado no histórico e exibido nas consultas do cliente |

---

## 4. Conexões com outros módulos

| Campo no Pedido de Venda | Referencia |
|---|---|
| `payment_term_code` | `payment_conditions.code` (FK — migration 000120) |
| `price_table_code` | `sales_tables.code` (FK — migration 000120) |
| `tax_type_code` | `tax_types.code` (FK — migration 000120) |
| `bearer_code` | `carriers.code` (FK — migration 000120) |

O campo `default_nf_type` do pedido de venda é um código de modelo NF-e (ex: "55" para NF-e, "65" para NFC-e), não uma FK para `invoice_types`. O `invoice_type_code` do cliente é uma **preferência** que deve ser copiada para o pedido ao criá-lo (lógica de negócio a implementar no front/use case de criação de pedido).

### Fluxo de verificação de crédito (a implementar no pedido de venda)

```
Pedido de venda criado
  └─ carrier.uses_credit_limit = true?
       ├─ carrier.consider_available = true  → saldo disponível = credit_limit − Σ pedidos em aberto
       └─ carrier.consider_available = false → usa credit_limit bruto
            └─ total do pedido > saldo disponível?
                 ├─ payment_condition.analysis_type = BLOQUEIA_SEMPRE → bloqueia
                 ├─ payment_condition.analysis_type = SEMPRE_ANALISA  → envia para análise
                 └─ payment_condition.analysis_type = LIBERA_SEM_ANALISE → aprova
```

---

## 5. Ordem recomendada de cadastro

```
Região → Segmento de Mercado → Tipo de Contato → Tipo de Cliente
  → Portador → Grupo de Portadores
  → Condição de Pagamento (+ parcelas)
  → Tabela de Vendas → Tipo de NF → Tipo de Imposto
  → Cliente → Endereços → Contatos
```
