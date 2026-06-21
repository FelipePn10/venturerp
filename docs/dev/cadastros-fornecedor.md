# Cadastro de Fornecedor (Fornecedores / Transportadoras)

Este documento descreve o módulo de **Fornecedor** do ERP: entidade principal, suas
pastas (telefones, e-mails, vencimentos, contatos), cadastros de apoio, parâmetros,
regras de negócio e as integrações com o módulo fiscal e com o Pedido de Compra/MRP.

O módulo segue a Clean Architecture do projeto:

| Camada | Caminho |
| --- | --- |
| Domínio (entidades + regras) | `internal/domain/supplier/entity/entity.go` |
| Interface de repositório | `internal/domain/supplier/repository/repository.go` |
| Queries SQL (sqlc) | `internal/infrastructure/database/queries/supplier.sql` |
| Repositório (impl) | `internal/infrastructure/repository/supplier/supplier_repository_sqlc.go` |
| DTOs | `internal/application/dto/request/supplier_dto.go` |
| Casos de uso | `internal/application/usecase/supplier_uc/supplier_uc.go` |
| Handler HTTP | `internal/interfaces/http/handler/supplier_handler.go` |
| Migração | `migrations/000135_suppliers.up.sql` |

Os cadastros de **Condição de Pagamento** (`payment_conditions`), **Transportadora**
(`carriers`) e **Região/Cidade** (`regions`) são **reaproveitados** do módulo de
Cliente — o fornecedor apenas referencia esses registros, evitando duplicação.

---

## 1. Entidade Fornecedor — campo a campo

| Campo | Coluna | Observações |
| --- | --- | --- |
| Código | `code` | Gerado automaticamente (`MAX(code)+1`) se não informado. |
| Ativo | `is_active` | Default `true`. |
| Representante | `is_representative` | Checkbox. |
| Cliente | `is_customer` | Marca o fornecedor também como cliente. |
| Descrição / Razão social | `name` | Obrigatório. |
| Fantasia | `trade_name` | |
| Pessoa | `person_type` | `JURIDICA` ou `FISICA`. |
| CNPJ / CPF | `document_number` + `document_type` | `CNPJ`, `CPF`, `ESTRANGEIRO`, `ISENTO`. |
| Inscr. Estadual | `state_registration` | Ver regra de obrigatoriedade abaixo. |
| Insc. Municipal | `municipal_registration` | |
| Tipo de Fornecedor | `supplier_type_id` | FK para `supplier_types`. |
| Tipo Frete | `freight_type` | `CIF`, `DAF`, `FOB`, `SEM_FRETE`, `CONVENIO`, `RETIRA`, `CORTESIA`, `TERCEIROS`. |
| Dt. Cadastro | `register_date` | Default = data atual. |
| Código Pai | `corporate_code` | Agrupa fornecedores (matriz/estabelecimentos). |
| Obr. Vitícola | `viticola_obligation` | `NUNCA`, `AS_VEZES`, `SEMPRE`. |
| Código GLN | `gln_code` | |
| Registro M.A. | `agriculture_ministry_registration` | Formato `AA-99999-9`. |
| Contrib. ICMS | `icms_contributor` | `CONTRIBUINTE`, `NAO_CONTRIBUINTE`, `ISENTO`. |
| Microempreendedor Individual | `is_mei` | Não permitido para Pessoa Física. |
| Plataforma de Rastreio | `tracking_platform` | `SSW`, `FRETEWEB`, `ENGLOBA_SISTEMAS`, `NENHUM`. |
| Homologado | `homologated` | Default conforme parâmetro 7. |
| Última Consulta SEFAZ | `last_sefaz_query` | Snapshot da consulta cadastral. |
| Situação Fat./Rec. | `billing_receipt_status` | `LIBERADO` / `BLOQUEADO`. |
| Endereço | tabela `supplier_addresses` | CEP, logradouro, número, complemento, bairro, cidade, UF, país. |

Os campos **Telefone**, **E-mail** e **Transp.** da tela não são editados diretamente:
recebem os dados das pastas correspondentes.

### Regras de negócio (aplicadas na entidade / caso de uso)

1. **Inscrição Estadual obrigatória** — exceto quando o `supplier_types.kind` for
   `TRANSPORTADORA`, `TRANSP_REDESP` ou `REDESPACHO`.
2. **MEI** não pode ser marcado quando `person_type = FISICA`.
3. **Registro M.A.** quando informado deve obedecer ao formato `AA-99999-9`.
4. **Documento duplicado** — ao criar, se já existir fornecedor com o mesmo
   CNPJ/CPF, a API retorna **409 Conflict** indicando o código existente.

---

## 2. Pastas do fornecedor

| Pasta | Tabela | Conteúdo |
| --- | --- | --- |
| Telefones | `supplier_phones` | número + ranking |
| E-mails | `supplier_emails` | e-mail + ranking |
| Vencimentos | `supplier_due_dates` | descrição, ranking, data base, condição de pagamento, tipo de pagamento, mês subsequente, arredondamento dia útil, horários e tempo médio de descarga (Inf. Recebimento) |
| Contatos | `supplier_contacts` (+ `supplier_contact_phones` / `supplier_contact_emails`) | nome, tipo, cargo, departamento, ranking, observação, **Tag do Pedido de Compra** |
| Empresas | `supplier_enterprises` | vínculo por empresa: conta financeira, checkbox IPI, tipo de NF default, tabela de preço de compra |

### Vencimentos — cálculo das condições de pagamento

A data base segue o parâmetro **10 – Data base padrão para vencimentos**
(`EMISSAO` / `ENTRADA` / `DIGITACAO`). A condição de pagamento define o número de
parcelas e a diferença de dias. O tipo de pagamento (`SEMANAL` / `MENSAL`) e o
arredondamento (`POSTERGA`, `ANTECIPA`, `UTIL`, `FIXO`) ajustam as datas para dias
úteis. `subsequent_month = true` adiciona um mês às parcelas.

---

## 3. Cadastros de apoio

- **Tipos de Fornecedor** (`supplier_types`): código, descrição e `kind`
  (`NORMAL` / `TRANSPORTADORA` / `TRANSP_REDESP` / `REDESPACHO`). O `kind` controla a
  obrigatoriedade da Inscrição Estadual.
- **Tipos de Contato** (`supplier_contact_types`): código + descrição.

---

## 4. Parâmetros de Fornecedores (`supplier_parameters`, 1 por empresa)

| # | Coluna | Descrição |
| --- | --- | --- |
| 1 | `default_financial_account` | Conta financeira default. |
| 2 | `unique_item_code_per_supplier` | Código do item único por fornecedor. |
| 3 | `requires_financial_account` | Obriga conta financeira no cadastro. |
| 4 | `purchase_supplier_type_id` | Tipo de fornecedor para comprar. |
| 5 | `copy_obs_to_purchase_order` | Leva observação para o pedido de compra. |
| 6 | `copy_obs_to_entry_invoice` | Leva observação para a NF de entrada. |
| 7 | `homologation_default` | Indicador de homologação default. |
| 8 | `use_stock_uom` | Usa a UM de estoque no cadastro. |
| 9 | `generic_supplier_code` | Fornecedor genérico usado na NFe. |
| 10 | `default_due_base_date` | Data base padrão para vencimentos. |

---

## 5. Endpoints REST

Base: `/api/suppliers` (todas exigem papel `ADMIN` ou `USER`).

### Cadastros de apoio (`/support`)
- `POST /support/supplier-types` · `PUT /support/supplier-types` · `GET /support/supplier-types`
- `POST /support/contact-types` · `GET /support/contact-types`
- `PUT /support/parameters` · `GET /support/parameters/{enterpriseCode}`

### Fornecedor
- `POST /` — criar (gera código; 409 se documento duplicado)
- `GET /` — listar (`?only_active=false` inclui inativos; `?format=xlsx|pdf|csv` baixa como arquivo)
- `GET /{code}` — obter (com pastas hidratadas)
- `PUT /` — atualizar
- `PATCH /{code}/block` · `PATCH /{code}/unblock`
- `GET /{code}/establishments` — estabelecimentos pelo código pai

### Pastas
- `POST /addresses` · `POST /phones` · `POST /emails` · `POST /due-dates`
- `POST /contacts` · `POST /contacts/phones` · `POST /contacts/emails`

### Vínculo com empresa
- `GET /{code}/enterprises` · `POST /enterprises` · `PUT /enterprises`

### Exemplo — criação de fornecedor PJ

```json
POST /api/suppliers
{
  "name": "Aços do Sul Ltda",
  "trade_name": "Aços Sul",
  "person_type": "JURIDICA",
  "document_type": "CNPJ",
  "document_number": "12345678000190",
  "state_registration": "1234567890",
  "supplier_type_code": 1,
  "freight_type": "CIF",
  "icms_contributor": "CONTRIBUINTE",
  "created_by": "00000000-0000-0000-0000-000000000000"
}
```

---

## 6. Integrações

### Auto-fill por CNPJ (Receita Federal)

`GET /api/cnpj/{cnpj}` devolve razão social, nome fantasia, **inscrição
estadual**, endereço, CNAE, porte e Simples/MEI para pré-preencher o cadastro
antes do `POST /api/suppliers`. Diferente da consulta SEFAZ (que valida situação
fiscal), esta serve para **digitar menos** ao criar o fornecedor. Detalhes em
[`integracao-cnpj-e-exportacao.md`](integracao-cnpj-e-exportacao.md).

### Consulta Cadastral SEFAZ (via FocusNFE)

`POST /api/suppliers/{code}/sefaz-query` consulta a situação cadastral do fornecedor
junto à SEFAZ/Receita via FocusNFE (usa o token de `fiscal_config`) e grava o snapshot
no cadastro: `last_sefaz_query`, `billing_receipt_status` (`LIBERADO`/`BLOQUEADO`),
`last_sefaz_update` e `sefaz_update_user`.

- A UF é obtida do endereço padrão do fornecedor. Estados **sem** o serviço de consulta
  (AL, AP, DF, MA, PA, PI, RJ, RO, RR, SE, TO) retornam erro informativo.
- A situação é `LIBERADO` quando o retorno indica habilitado/ativo, senão `BLOQUEADO`.

### Provider de defaults (fonte única)

O caso de uso de fornecedor implementa `ports.SupplierPurchasingDefaultsProvider`,
consumido pelo Pedido de Compra e pelo Fiscal:

- `GetPurchasingDefaults(supplierCode, enterpriseCode)` → condição de pagamento
  (do cadastro ou, se ausente, o vencimento de menor ranking com condição), tipo de
  frete, contribuinte de ICMS, inscrição estadual e, por empresa, conta financeira,
  tipo de NF default, tabela de preço de compra e flag IPI.
- `FindSupplierCodeByDocument(documento)` → casa CNPJ/CPF (somente dígitos) a um
  fornecedor cadastrado.
- Endpoint: `GET /api/suppliers/{code}/purchasing-defaults?enterprise=<código>`.

### Pedido de Compra (Fase 2 — entregue)
- `purchase_orders.supplier_code` possui **FK** para `suppliers(code)`.
- `CreatePurchaseOrderUseCase` recebe o provider (opcional, nil-safe): ao criar um
  pedido com `supplier_code` e **sem** `payment_term_code`, a condição de pagamento é
  preenchida automaticamente a partir do fornecedor.

### Fiscal — NF de entrada (Fase 2 — entregue)
- Nova coluna `fiscal_entries.supplier_code` (migration 000136).
- Na importação de NF-e de compra (`ImportNFePurchaseUseCase`), o **CNPJ do emitente**
  é casado a um fornecedor cadastrado e o vínculo é gravado na entrada
  (`supplier_matched` no retorno). Habilita conta a pagar por fornecedor e o uso de
  `icms_contributor` / tipo de NF default no cálculo fiscal.
- O campo já existente `ICMSReductionSubstitution.SupplierID` passa a ter destino real
  (validação em nível de aplicação; mantém-se sem FK por convenção do módulo fiscal).

### MRP → Sugestão de compra (Fase 3 — entregue)

Uma **sugestão de compra** é uma `planned_order` do tipo `PURCHASE` ainda não firme
(`is_firm = false`, `status = PLANNED`). O PCP/Compras a aprova ou rejeita:

- **Aprovar** (`ApprovePurchaseSuggestionUseCase`): gera um `purchase_order`
  (`origin = MRP`, `status = APPROVED`, `is_firm = true`) com o fornecedor escolhido e
  um item (código/quantidade corrigida/data de necessidade da sugestão); a condição de
  pagamento vem dos defaults do fornecedor. Em seguida torna a `planned_order` firme
  (`FirmPlannedOrder` → `is_firm = true, status = RELEASED`).
- **Rejeitar** (`RejectPurchaseSuggestionUseCase`): muda o status para `CANCELLED` e
  inativa a sugestão.
- Apenas suprimentos firmes/aprovados entram no netting do MRP
  (`PlannedOrderSupplyPort.ListFirmSupplyForItems`), então sugestões não aprovadas não
  reduzem a necessidade líquida.

Endpoints (sob `/api/purchase-order`):
- `GET  /suggestions` — lista as sugestões de compra abertas.
- `POST /suggestions/{code}/approve` — corpo: `enterprise_code`, `supplier_code`,
  `unit_price`, `notes`, `created_by`. Retorna o pedido de compra gerado.
- `POST /suggestions/{code}/reject` — rejeita a sugestão.

> Observação: não há cadastro de "fornecedor preferencial por item"; o comprador
> informa o `supplier_code` na aprovação.
