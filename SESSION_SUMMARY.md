# Session Summary

## Current Session: Comercial Fase 6 — Meta de Vendas

### What Was Implemented

- Consulted the official FoccoERP help pages before implementation:
  - `Processos/Comercial/metas-de-vendas/`
  - `FMET0100` - Cadastro de Períodos
  - `FMET0200` - Cadastro de Metas
  - `FMET0201` - Cadastro de Metas por Grupo Comercial
  - `FMET0202` - Cadastro de Saldos de Metas
  - `FMET0300` - Relatório de Metas
- Implemented Comercial phase 6 for sales goals:
  - periods by month, week or custom interval;
  - representative goals by period and analysis base;
  - goal lines by exactly one target: item, item classification or item group;
  - commercial group goals with minimum, probable and ideal values and bonus percentages;
  - customer-level goals under commercial group goals;
  - goal balances/excess values for carry-over;
  - report for planned vs realized with filters by representative, customer, region,
    microregion, period and analysis base.

### Delivered In Code

- Added migration `000191_sales_goals`:
  - `sales_goal_periods`
  - `sales_goals`
  - `sales_goal_items`
  - `sales_goal_group_targets`
  - `sales_goal_group_customers`
  - `sales_goal_balances`
- Added domain/repository/application/HTTP stack:
  - `internal/domain/sales_goal/...`
  - `internal/application/usecase/sales_goal_uc/...`
  - `internal/infrastructure/repository/sales_goal/repository.go`
  - `internal/interfaces/http/handler/sales_goal_handler.go`
- Added API routes under `/api/sales-goals`:
  - create/list/get/update goals
  - create/list periods
  - add goal items
  - upsert commercial group targets
  - add group customers
  - add balances
  - report planned vs realized
- Added focused validation script:
  - `scripts/test-comercial-metas.sh`

### Documentation

- Updated `docs/dev/vendas.md` with the sales goals module purpose, usage,
  concepts, routes, report, persistence and validation rules.
- Updated `docs/apresentacao/vendas.md` with business-facing explanation of
  sales goals and glossary entry.
- Updated `docs/dev/API_REQUEST_BODIES.txt` with sales goals payload examples.
- Product/dev docs intentionally do not mention external systems or screen codes.

### Validation Run

- `scripts/test-comercial-metas.sh`
- `env GOCACHE=/tmp/panossoerp-go-build go test ./...`

All validations passed. HTTP smoke in the script was skipped because
`BASE_URL`/`TOKEN` were not set.

### Notes / Follow-ups

- `SALES` realization is calculated from sales orders by representative/customer
  and period. `INVOICING` is modeled and available for cadastro/report filters,
  but deeper fiscal realization should be wired when the faturamento phase
  expands the nota fiscal by load and invoice-origin routines.
- The report currently returns representative and commercial-group rows. Customer
  details are stored and used for group filtering/realization; a later UI/report
  pass can add a dedicated customer-breakdown layout if needed.

## Current Session: Comercial Fase 5 — Representantes

### What Was Implemented

- Reworked Comercial phase 5 after user clarified the implementation must follow
  the FoccoERP help behavior and only be considered complete after consulting the
  source pages.
- Consulted the exact public help pages used for the phase:
  - `FREP0200` - Cadastro de Representantes
  - `FREP0101` - Cadastro de Tipos de Representantes
  - `FREP0251` - Relatório de Representantes
  - `FREP0253` - Ficha de Acompanhamento de Representantes
- Implemented the representative module with the same functional coverage and a
  more API-friendly structure:
  - representative types with `is_free` and `ignores_direct_billing`;
  - full representative cadastro with customer/supplier links, type, category,
    register date, CORE, document, address, active/block status and device count;
  - cadastro tabs for enterprises/commission, accounting integration, regions,
    segments, sales plans, item interests, phones, emails, correspondence address
    and contacts/prepostos;
  - report filters by representative, description, type, UF, region, active
    status, ordering and optional accounting accounts;
  - commercial follow-up by representative and customers using quotations and
    sales orders.

### Delivered In Code

- Added migration `000190_sales_representatives`:
  - `representative_types`
  - `representatives`
  - `representative_enterprises`
  - `representative_accounting`
  - `representative_regions`
  - `representative_segments`
  - `representative_sales_plans`
  - `representative_interests`
  - `representative_phones`
  - `representative_emails`
  - `representative_correspondence_addresses`
  - `representative_contacts`
- Added domain/repository/application/HTTP stack:
  - `internal/domain/representative/...`
  - `internal/application/usecase/representative_uc/...`
  - `internal/infrastructure/repository/representative/repository.go`
  - `internal/interfaces/http/handler/representative_handler.go`
- Added API routes under `/api/representatives`:
  - create/list/get/update/block/unblock representatives
  - create/list/get/update representative types
  - add enterprises, accounting, regions, segments, sales plans, interests,
    phones, emails, correspondence addresses and contacts
  - report and follow-up endpoints
- Added focused test/smoke script:
  - `scripts/test-comercial-representantes.sh`

### Documentation

- Updated `docs/dev/vendas.md` with the representatives module purpose, routes,
  cadastro tabs, report, follow-up, persistence and validations.
- Updated `docs/apresentacao/vendas.md` with business-facing representative
  workflow and glossary entry.
- Updated `docs/dev/API_REQUEST_BODIES.txt` with representative payload examples.
- Product/dev docs intentionally do not mention external systems or screen codes.

### Validation Run

- `scripts/test-comercial-representantes.sh`
- `env GOCACHE=/tmp/panossoerp-go-build go test ./...`
- `git diff --check`

All validations passed. HTTP smoke in the script was skipped because
`BASE_URL`/`TOKEN` were not set.

### Notes / Follow-ups

- The follow-up endpoint reads existing `sales_quotations` and `sales_orders`.
  It will become more valuable as frontends consistently send
  `representative_code` on quotations and orders.
- A future phase can wire representative commission settlement into financeiro
  once payable commission generation is prioritized.

## Current Session: Comercial Fase 4 — Pedido de Venda

### What Was Implemented

- Started and completed Comercial phase 4 focused on Pedido de Venda maturity.
- Consulted the exact public help pages for the phase-4 sales-order routines before implementation:
  - `FPDV0200_PDV`
  - `FPDV0202`
  - `FPDV0203_COM`
  - `FPDV0203_FIN`
  - `FPDV0205_PDV`
  - `FPDV0210`
  - `CPDV0411`
- Documentation in `docs/dev` and `docs/apresentacao` remains product-only and does not mention external systems or screen codes.

### Delivered In Code

- Added migration `000189_sales_order_phase4`:
  - analysis statuses: `commercial_analysis_status`, `financial_analysis_status`
  - release status: `release_status`
  - conference status: `conference_status`
  - cancel/attend/delay fields
  - `sales_order_events` audit/history table
- Fixed an existing gap in sales-order cadastro:
  - DTOs already exposed advanced commercial/fiscal/logistic fields, but create/update did not persist all of them and mapper did not return all of them.
  - Updated SQLC source query and regenerated SQLC so creation/update now persist representative order number, NFC-e flag, consumer address, carrier, freight, insurance, volume, weights, discount/surcharge and project fields.
- Added advanced sales-order portfolio query:
  - `GET /api/sales-order/search`
  - filters: customer, representative, payment term, status, commercial/financial analysis, release, conference, blocked flag, emission period and delivery period.
- Added sales-order report:
  - `GET /api/sales-order/report`
  - totals: order count, gross/net value, open/confirmed/invoiced/cancelled/blocked counts, pending commercial analysis, pending financial analysis, pending conference and delayed orders.
- Added operational routines:
  - `POST /api/sales-order/{code}/analyze`
  - `POST /api/sales-order/{code}/release`
  - `POST /api/sales-order/{code}/attend`
  - `POST /api/sales-order/{code}/conference`
  - `POST /api/sales-order/{code}/delay-reason`
  - `DELETE /api/sales-order/{code}/cancel` now accepts reason/complement and keeps the order active for history instead of hiding it by `is_active=false`.
- Added script `scripts/test-comercial-pedido-venda.sh`.

### Documentation

- Expanded `docs/dev/vendas.md` Pedido de Venda section with:
  - purpose, where used, concepts, routes, status models, consultation/reporting, operational rules and tests.
- Expanded `docs/apresentacao/vendas.md` Pedido de Venda section with business-facing explanation.
- Verified product docs with:
  - `rg -n "Focco|FoccoERP|FPDV|CPDV" docs/dev docs/apresentacao`
  - no matches.

### Validation Run

- `scripts/test-comercial-pedido-venda.sh`
- `env GOCACHE=/tmp/panossoerp-go-build go test ./...`
- `git diff --check`

All validations passed. HTTP smoke in the script was skipped because `BASE_URL`/`TOKEN` were not set.

## Current Session: Comercial Fase 3 — Orçamentos

### What Was Implemented

- Reworked the Comercial roadmap phase 3 for sales quotations/orçamentos after the user corrected that the implementation must be based on the exact FoccoERP help pages, not only on the routine names.
- FoccoERP help pages used as source:
  - `https://help.foccoerp.com.br/Programas/FoccoERP/Comercial/Or%C3%A7amento/FPDV0200_ORC/`
  - `https://help.foccoerp.com.br/Programas/FoccoERP/Comercial/Or%C3%A7amento/FPDV0205_ORC/`
  - `https://help.foccoerp.com.br/Programas/FoccoERP/Comercial/Or%C3%A7amento/Consulta/CPDV0410_ORC/`
  - `https://help.foccoerp.com.br/Programas/FoccoERP/Comercial/Or%C3%A7amento/Relat%C3%B3rios/FPDV0206_ORC/`
- Implemented the FoccoERP-aligned routines:
  - `FPDV0200 ORC` - Cadastro de Orçamentos
  - `FPDV0205 ORC` - Cancelamento/Atendimento de Orçamentos
  - `CPDV0410 ORC` - Consulta de Orçamentos
  - `FPDV0206 ORC` - Relatório de Orçamentos
  - conversion to `FPDV0200 PDV`
- Added migration `000188_sales_quotation` with:
  - `sales_quotation_sequences`
  - `sales_quotations`
  - `sales_quotation_items`
  - `sales_quotation_events`
  - `sales_quotation_attachments`
- Added quotation lifecycle statuses:
  - `R`, `P`, `A`, `OA`, `F`, `OF`, `CANCELLED`, `ATTENDED`, `EXPIRED`
- Added quotation types aligned to the Focco field `Tipo`:
  - `API_TERCEIROS`, `CONSULTA`, `FOCCOPORTAL`, `IMPORTADO`, `NEGOCIACAO`, `VENDA`
- Added quotation release states:
  - `BLOCKED`, `MANUAL_RELEASED`, `RELEASED`
- Added Focco-aligned header/transport/commercial fields including purchase order number, digit date, NFC-e flag, commission percent, consumer address fields, commercial block reason, carrier, freight type, freight verification, redelivery freight, retained tax, delivery authorization, cancellation complement and attendance reason/date.
- Added quotation item statuses:
  - `OPEN`, `PARTIAL`, `DELIVERED`, `CANCELLED`
- Added domain/repository/application/HTTP stack:
  - `internal/domain/sales_quotation/...`
  - `internal/application/usecase/sales_quotation_uc/...`
  - `internal/infrastructure/repository/sales_quotation/repository.go`
  - `internal/interfaces/http/handler/sales_quotation_handler.go`
- Added API routes under `/api/sales-quotation`:
  - create/list/get/update quotation
  - cancel quotation with reason/complement
  - uncancel quotation with reason/complement
  - attend quotation with reason/complement/date
  - change status
  - report summary
  - convert quotation to sales order
  - create/list/update/cancel items
- Conversion behavior:
  - blocks cancelled, expired, attended, consultation-type, commercially blocked and already converted quotations;
  - creates a sales order using the existing sales order numbering/repository;
  - copies only active open item balance to the order;
  - copies existing compatible commercial/fiscal fields such as commission, NFC-e flag, consumer address, carrier and freight type;
  - marks the quotation as `ATTENDED` and stores `converted_sales_order_code`.
- Added focused test and validation script:
  - `internal/application/usecase/sales_quotation_uc/items_uc_test.go`
  - `scripts/test-comercial-orcamentos.sh`
- Updated documentation:
  - `docs/dev/vendas.md`
  - `docs/apresentacao/vendas.md`

### Architectural Decisions

- Quotations are stored in dedicated tables instead of overloading `sales_orders` statuses `OA`/`OF`, because quotation needs its own validity, probability, cancellation reason, partial attendance and conversion traceability.
- Conversion to order reuses the existing `sales_order` repository so downstream credit, ATP/reservation, MRP and invoicing remain in the sales order flow.
- The quotation module uses direct `pgxpool` SQL in its repository to keep this phase isolated from generated SQLC churn.
- Pricing and commercial policy from phases 1 and 2 remain as independent engines; this phase stores negotiated values and leaves deeper automatic policy application for a later integration pass.

### Validation Run

- `scripts/test-comercial-orcamentos.sh`
- `env GOCACHE=/tmp/panossoerp-go-build go test ./...`
- `git diff --check`

All validations passed. HTTP smoke in the script was skipped because `BASE_URL`/`TOKEN` were not set.

### Documentation Correction

- User clarified that product/developer documentation must explain purpose, usage, where the feature is used and why it exists, not only list routes.
- User also clarified that documentation must describe only this ERP and must not reference FoccoERP or other systems.
- Updated `docs/dev/vendas.md`:
  - removed FoccoERP/help URL/screen-code references from the Orçamentos section;
  - expanded the module explanation with purpose, business usage, concepts, lifecycle, rules, NFC-e behavior, reports/queries and tests.
- Updated `docs/apresentacao/vendas.md`:
  - removed external-system wording from status descriptions;
  - expanded the business explanation of orçamentos and the NFC-e indicator.
- Verified with `rg -n "Focco|FoccoERP|FPDV|CPDV" docs/dev docs/apresentacao`: no matches.
- Re-ran `scripts/test-comercial-orcamentos.sh` and `git diff --check`: both passed.

### Notes / Follow-ups

- Integrate quotation item creation with `/api/customers/sales-tables/pricing` and `/commercial-policies/evaluate` so price/policy effects can be applied automatically when the frontend creates quote lines.
- Add Postgres integration tests for quotation conversion once a migrated test database is available.
- The working tree contains a generated `internal/infrastructure/database/sqlc/models.go` diff for commercial policy/pricing model structs from previous phases; review before commit if the repository expects generated SQLC files to stay synchronized.

## What Was Implemented

- Added BOM substitute/alternative component support on the real BOM model (`item_structures`):
  - `substitute_group`
  - `substitute_priority`
- Added migration `000181_structure_substitutes`.
- Propagated the new BOM fields through:
  - request DTOs
  - response DTOs
  - SQLC queries/generated structs
  - structure repositories
  - structure mappers/presenters
- Centralized substitute selection in `internal/domain/structure/entity/substitute.go`.
- Updated MRP BOM explosion to:
  - ignore co-products as demand inputs
  - respect fixed quantity components
  - select only the primary component from substitute groups
  - avoid assigning LLC through co-products
- Updated standard cost rollup to:
  - cost only the primary substitute in a group
  - respect BOM `mask` when resolving child components
  - keep co-product/fixed-quantity logic aligned with MRP
- Updated production backflush to:
  - ignore co-products
  - respect fixed quantity components
  - consume only the primary substitute
- Updated production completion co-product receipt to use primary substitute selection.
- Updated cutting-plan demand generation to:
  - ignore co-products
  - use only primary substitute components
- Added focused BOM/MRP validation:
  - `scripts/test-bom-mrp.sh`
  - `make test-bom-mrp`
- Fixed quality targets:
  - `make fmt-check` now ignores `vendor/`
  - Go cache/module paths now use `/tmp` defaults for sandbox/CI compatibility
  - formatted previously non-gofmt project files
- Updated documentation for BOM enterprise+ behavior across MRP, cost, production, cutting, item cadastro, and API examples.

## Architectural Decisions

- `item_structures` remains the single source of truth for BOM lines.
- `bom_headers` remains the BOM header/version/status/type layer.
- The older `boms`/`bom_items` model stays retired and was not extended.
- Substitute groups are resolved deterministically:
  - lower `substitute_priority` wins
  - ties use lower `sequence`
  - final tie uses lower `child_code`
- Planning/cost/automatic execution use the primary substitute only.
- Secondary substitutes remain modeled and visible for operational/manual substitution, not duplicated demand.
- Substitute selection is centralized in the structure domain so MRP, production, and cutting share the same interpretation.
- Standard cost rollup resolves BOM by `mask` to avoid costing unrelated variant-specific components.

## Pending Tasks

- Decide whether to add operational UI/API for explicitly selecting a secondary substitute during production consumption when the primary is unavailable.
- Add integration tests against Postgres for substitute groups in `item_structures` once a migrated test DB is available.
- Decide whether substitute availability should eventually be stock-aware in MRP, rather than always priority-first.
- Review whether co-product substitute groups are meaningful or should be rejected by validation.
- Consider adding database constraints for invalid substitute values beyond defaults/checks if business wants stricter enforcement.

## Important Files Modified

- `migrations/000181_structure_substitutes.up.sql`
- `migrations/000181_structure_substitutes.down.sql`
- `internal/domain/structure/entity/substitute.go`
- `internal/domain/structure/entity/substitute_test.go`
- `internal/domain/mrp_calculation/service/mrp_service_impl.go`
- `internal/domain/mrp_calculation/service/mrp_helpers_test.go`
- `internal/application/usecase/cost_uc/cost_rollup_uc.go`
- `internal/application/usecase/cost_uc/cost_rollup_helpers_test.go`
- `internal/application/usecase/production_order_uc/add_appointment_uc.go`
- `internal/application/usecase/production_order_uc/complete_production_order_uc.go`
- `internal/application/usecase/cutting_plan_uc/demand_uc.go`
- `internal/infrastructure/database/queries/structure.sql`
- `internal/infrastructure/database/sqlc/structure.sql.go`
- `internal/infrastructure/database/sqlc/models.go`
- `internal/infrastructure/database/sqlc/structure_bom.go`
- `internal/infrastructure/repository/structure/item_structure_repository_sqlc.go`
- `internal/infrastructure/repository/standard_cost/standard_cost_repository_sqlc.go`
- `scripts/test-bom-mrp.sh`
- `makefile`
- `docs/dev/mrp-calculo.md`
- `docs/dev/custos.md`
- `docs/dev/cadastros-item.md`
- `docs/dev/manufatura-e-compras.md`
- `docs/dev/plano-de-corte.md`
- `docs/dev/producao.md`
- `docs/dev/API_REQUEST_BODIES.txt`
- `docs/apresentacao/cadastros.md`
- `docs/README.md`

## Known Issues

- `make ci` passes, but total coverage remains low at `14.2%`.
- Integration tests were not run in this session because they require a migrated Postgres test database.
- `make test` and `make ci` require permission to run tests that open local sockets via `httptest`.
- Build/test cache paths were redirected to `/tmp` to avoid read-only home cache issues.
- A build artifact can be produced at `bin/erp` by `make build`; it is not tracked by git.

## Validation Run

- `make fmt-check`
- `git diff --check`
- `make build`
- `make vet`
- `make test-bom-mrp`
- `make test`
- `make ci`

All listed validation commands passed by the end of the session.

## Next Steps

1. Apply migration `000181_structure_substitutes` in a migrated development/test database.
2. Run `make test-integration` with `TEST_DATABASE_URL` configured.
3. Add API/UI workflow for choosing a non-primary substitute during production if needed.
4. Consider stock-aware substitute selection in future MRP enhancement.
5. Review generated API responses in the frontend to expose substitute group/priority cleanly in BOM maintenance screens.

---

## Subsequent Session: ERP Maturity Roadmap and Purchase Receiving

### What Was Implemented

- Compared the current VentureERP module coverage against the public FoccoERP help tree for critical sectors:
  - Compras / Suprimentos
  - Comercial
  - Engenharia / Produção
- Ranked the current criticality:
  1. Compras / Suprimentos
  2. Comercial
  3. Engenharia / Produção
- Selected Compras / Suprimentos as the first sector to mature.
- Created a roadmap document:
  - `docs/dev/maturidade-erp-roadmap.md`
- Documented FoccoERP Suprimentos processes and program/screen codes, including:
  - `FUTL0125 PRC PRC` - Parâmetros da Tabela de Compra
  - `FUTL0125 PDC PDC` - Parâmetros de Pedidos de Compra
  - `FUTL0125 COT COT` - Parâmetros da Cotação de Compra
  - `FUTL0125 SLC SLC` - Parâmetros de Solicitação de Compra
  - `FUTL0125 AVR AVR` - Parâmetros do Aviso do Recebimento
  - `FUTL0125 INSP INSP` - Parâmetros de Inspeção de Recebimento
  - `FUTL0125 AVF AVF` - Parâmetros da Avaliação de Fornecedor
  - `FFOR0200`, `FFOR0201`, `FFOR0202`, `FFOR0204` - Fornecedores e itens por fornecedor
  - `FCOT0200`, `FCOT0201`, `FCOT0202`, `CCOT0400`, `FCOT0300` - Cotação de compra
  - `FPDC0200`, `FPDC0204`, `FPDC0205`, `CPDC0400`, `CPDC0402`, `CPDC0403`, `FPDC0250` - Pedido de compra
  - `FPDC0201`, `FPDC0202`, `CPDC0401`, `FPDC0251` - Solicitação de compra
  - `FAVR0200`, `FAVR0201`, `FAVR0204`, `FAVR0300` - Aviso/divergências de recebimento
  - `FREC0200`, `FREC0201`, `FREC0203`, `FREC0255` - Recebimento/NF-e de entrada
  - `FCLR0200` - Checklist de recebimento
  - `FINS0200`, `FINS0201`, `FINS0202`, `FINS0203`, `FINS0207`, `FINS0212`, `FINS0304` - Inspeção de recebimento
  - `FAVF0200` to `FAVF0205` - Avaliação de fornecedores/IQF
  - `FALC0200`, `FALC0201` - Alçada de valores
  - `FCON0200`, `FCON0202`, `CCON0400` - Contratos de fornecedores
  - `FEDS0130`, `FEDS0131`, `FEDS0251`, `FEDS0252`, `FEDS0253`, `FEDS0300` - EDI fornecedores
- Added operational purchase receiving by purchase order line:
  - `POST /api/purchase-order/{code}/receipts`
  - request DTO: `internal/application/dto/request/purchase_receipt_dto.go`
  - use case: `internal/application/usecase/purchase_order_uc/receive_purchase_order_uc.go`
  - response DTOs in `purchase_order_response.go`
  - handler method in `purchase_order_handler.go`
  - API wiring in `api/api.go`
- Added repository support for exact purchase order line receipts:
  - `RegisterItemReceipts`
- Kept the previous NF-e import path compatible through `RegisterReceipts`.
- Fixed a bug in legacy receipt registration by `item_code`:
  - Before: if the same item appeared in multiple purchase order lines, the same received quantity could be applied to more than one line.
  - Now: the quantity is distributed by each line's remaining balance.
- Added tolerance handling in physical receipt:
  - `tolerance_pct`
  - `cancelled_tolerance_qty`
- Added focused tests:
  - `internal/application/usecase/purchase_order_uc/receive_purchase_order_uc_test.go`
- Added focused validation script:
  - `scripts/test-purchase-receiving.sh`
- Added make target:
  - `make test-purchase-receiving`
- Updated documentation:
  - `docs/README.md`
  - `docs/dev/00-fluxo-geral.md`
  - `docs/dev/API_REQUEST_BODIES.txt`
  - `docs/dev/manufatura-e-compras.md`
  - `docs/dev/maturidade-erp-roadmap.md`
  - `docs/apresentacao/00-fluxo-geral.md`
  - `docs/apresentacao/compras.md`

### Architectural Decisions

- Operational receiving should be done by `purchase_order_item_code`, not by `item_code`.
- NF-e import can continue to use `item_code`, but the repository must distribute quantities safely by remaining balance.
- Receiving posts stock movement `IN` with reference type `PURCHASE_ORDER`.
- Purchase receiving updates both stock and purchase order line/header status.
- The next Suprimentos routine should be inspection/quarantine before vendor evaluation, because it is the operational gate that feeds later IQF and divergence metrics.

### Current Suprimentos Coverage Against FoccoERP Screens

- Created / implemented equivalents:
  - `FFOR0200`, `FFOR0201`, `FFOR0202` - supplier and item-supplier registration
  - `FCOT0200` - purchase quotation creation
  - `FCOT0201` - partial quotation analysis / winner selection
  - `FCOT0202` - release requisitions/planned orders to quotation
  - `CCOT0400` - quotation consultation
  - `FPDC0200` - purchase order creation
  - `FPDC0204` - generate purchase orders from requisitions
  - `FPDC0205` - partial cancellation/attendance behavior through cancel/receipt flows
  - `FPLA0202` - release planned purchase orders through MRP suggestion approval
  - `FPLA0203` - partial support through quotation from planned orders
  - `CPDC0400` - purchase order consultation
  - `CPDC0402` - partial support through purchase suggestions/open requisitions
  - `FPDC0201` - purchase requisition creation
  - `CPDC0401` - purchase requisition consultation
  - `FREC0200` - partial support through fiscal entry import/create/approve
  - physical receipt equivalent for the operational receiving cycle, exposed as `/api/purchase-order/{code}/receipts`
- Next to create:
  1. `FINS0200`, `FINS0201`, `FINS0202`, `FINS0203`, `FINS0212` - receiving inspection integrated with quarantine stock.
  2. `FAVR0200`, `FAVR0201`, `FAVR0204`, `FAVR0300` - receiving notice and divergence handling.
  3. `FAVF0200` to `FAVF0205` - supplier evaluation / IQF.
  4. `FALC0200`, `FALC0201` - purchase approval limits.
  5. `FCON0200`, `FCON0202`, `CCON0400` - supplier contracts.
  6. `FCLR0200`, `FINS0304` - receiving checklist and labels.
  7. `FEDS0130`, `FEDS0131`, `FEDS0251`, `FEDS0252`, `FEDS0253`, `FEDS0300` - supplier EDI.
  8. `FREC0203` and import flows - import purchase/nationalization.

### Important Files Modified In This Session

- `.gitignore`
- `SESSION_SUMMARY.md`
- `api/api.go`
- `makefile`
- `docs/README.md`
- `docs/dev/00-fluxo-geral.md`
- `docs/dev/API_REQUEST_BODIES.txt`
- `docs/dev/manufatura-e-compras.md`
- `docs/dev/maturidade-erp-roadmap.md`
- `docs/apresentacao/00-fluxo-geral.md`
- `docs/apresentacao/compras.md`

---

## Subsequent Session: Suprimentos Operational Maturity

### What Was Implemented

- Added migration `000182_procurement_maturity` with:
  - `procurement_records` for operational procurement routines;
  - `procurement_inspection_dispositions` for receiving inspection outcomes;
  - `supplier_scorecard_snapshots` for supplier IQF/history.
- Added a new procurement domain/use case/repository/handler stack:
  - `internal/domain/procurement/...`
  - `internal/application/usecase/procurement_uc/procurement_uc.go`
  - `internal/infrastructure/repository/procurement/repository.go`
  - `internal/interfaces/http/handler/procurement_handler.go`
- Added REST endpoints under `/api/procurement`:
  - create/list/get operational records;
  - update record status;
  - dispose receiving inspections with approved/rejected quantities;
  - create/list supplier scorecard snapshots.
- Covered the missing Suprimentos blocks identified from the FoccoERP/SAP/Oracle-style comparison as VentureERP business concepts:
  - receiving inspection/quarantine;
  - receiving notice and divergences;
  - supplier evaluation/IQF;
  - approval limits;
  - supplier contracts;
  - receiving checklist and labels;
  - supplier EDI records;
  - import/nationalization process tracking.
- Kept Focco screen codes out of backend names, routes, variables, folders and tables.
- Integrated receiving inspection disposition with stock:
  - approved quantity can transfer from quarantine warehouse to available warehouse;
  - rejected quantity can transfer to a blocked/quarantine/return warehouse.
- Added initial IQF weighting:
  - quality 40%;
  - delivery 30%;
  - commercial 20%;
  - service 10%.

### Architectural Decisions

- `procurement_records` is an operational workflow ledger, not a copy of any third-party ERP screen model.
- The first version uses typed record categories plus flexible `payload` JSONB so the frontend can support factory-specific checklists, EDI layouts, contract clauses and divergence metadata without a migration for every field.
- Stock impact is intentionally limited to inspection disposition; merely creating an inspection/divergence record does not move stock.
- Approval limits and contracts are recorded now, but automatic blocking/unblocking of purchase orders remains a future rule engine step.
- EDI and import are tracked operationally now; parser/layout processing and fiscal document generation remain future integrations.

### Important Files Modified In This Session

- `api/api.go`
- `migrations/000182_procurement_maturity.up.sql`
- `migrations/000182_procurement_maturity.down.sql`
- `internal/application/dto/request/procurement_maturity_dto.go`
- `internal/application/dto/response/procurement_maturity_response.go`
- `internal/application/usecase/procurement_uc/procurement_uc.go`
- `internal/domain/procurement/entity/entity.go`
- `internal/domain/procurement/repository/repository.go`
- `internal/infrastructure/repository/procurement/repository.go`
- `internal/interfaces/http/handler/procurement_handler.go`
- `docs/README.md`
- `docs/dev/00-fluxo-geral.md`
- `docs/dev/API_REQUEST_BODIES.txt`
- `docs/dev/manufatura-e-compras.md`
- `docs/dev/maturidade-erp-roadmap.md`
- `docs/apresentacao/compras.md`
- `SESSION_SUMMARY.md`

### Validation Run

- Initial focused `go test` failed because the default Go build cache in the home directory is read-only in the sandbox.
- Re-ran with project-approved temp cache:
  - `env GOCACHE=/tmp/panossoerp-go-build go test ./...`
- Full test suite passed.

### Follow-up Backlog From New Suprimentos Sweep

Items to validate with the business before implementation:

1. Automatic generation of receiving inspection from item/supplier/entry-operation criticality.
2. Purchase order blocking by approval limit before supplier release.
3. Supplier contracts with normalized item lines, consumed contracted balance and price adjustment index.
4. Real EDI parser/layout engine, inbound/outbound queues and automatic fiscal document creation.
5. Import purchase costing with DI/DUIMP, exchange rate, expenses, taxes and landed cost per item.
6. Freight quotation/order flow integrated with purchase order delivery and inbound logistics.
7. Consolidated purchase movement history/reporting for buyer and supplier performance.
8. Rural producer counter-invoice flow if the factory buys from rural producers.

---

## Subsequent Session: FoccoERP URL Pattern and Structured Receiving Inspection

### External Documentation Consulted

- Confirmed the correct FoccoERP help URL pattern:
  - process pages: `/Processos/{Area}/{processo}/`
  - program pages: `/Programas/FoccoERP/{Area}/{Modulo}/{CODIGO}/?h={codigo}`
- Consulted:
  - `Processos/Suprimentos/inspecao-de-recebimento/`
  - `FINS0200` receiving inspection route
  - `FINS0201` inspection order maintenance
  - `FINS0202` inspection result entry
  - `FINS0203` inspection analysis
  - `FINS0212` manual inspection order generation
- Confirmed that the previous generic `procurement_records` approach was useful as a ledger, but below the functional depth of FoccoERP for receiving inspection.

### Ranking Reconfirmed

1. Compras / Suprimentos remains the most deficient critical sector.
   - Reason: post-purchase routines are still the largest maturity gap: receiving inspection, skip/frequency, supplier divergences, contracts, EDI, imports and IQF.
2. Comercial is second.
   - Reason: sales order/forecast/shipping exist, but assistance, CRM, policies, goals and customer EDI are incomplete.
3. Engenharia / Produção is third.
   - Reason: BOM/MRP/CRP/APS/routing/production/quality are already advanced; remaining work is refinement and service integration.

### What Was Implemented

- Added migration `000183_receiving_inspection_core` with structured receiving inspection tables:
  - `receiving_inspection_routes`
  - `receiving_inspection_route_steps`
  - `receiving_inspection_step_attributes`
  - `receiving_inspection_orders`
  - `receiving_inspection_results`
  - `receiving_inspection_analyses`
- Added typed support for:
  - route basis by item or supply classification;
  - route validity;
  - inspection warehouse;
  - handling/storage instructions;
  - step kind: value, attribute or structure;
  - appointment mode: all measurements, single interval, multiple interval, status only;
  - mandatory appointment;
  - label emission flag;
  - instrument group, sample, norm and reference;
  - nominal/min/max values and approved/rejected attributes;
  - inspection order source: purchase receipt, receiving notice, fiscal entry or manual;
  - inspection order statuses;
  - analysis treatment that can affect supplier score.
- Added endpoints:
  - `POST /api/procurement/receiving-inspection-routes`
  - `GET /api/procurement/receiving-inspection-routes/{id}`
  - `POST /api/procurement/receiving-inspection-orders`
  - `GET /api/procurement/receiving-inspection-orders?status=...`
  - `POST /api/procurement/receiving-inspection-orders/{id}/results`
  - `POST /api/procurement/receiving-inspection-orders/{id}/analysis`
- Route selection now tries:
  1. item + exact/blank mask;
  2. supply classification, from most specific prefix to most generic.
- Updated documentation:
  - `docs/dev/manufatura-e-compras.md`
  - `docs/dev/API_REQUEST_BODIES.txt`
  - `docs/dev/maturidade-erp-roadmap.md`
  - `docs/apresentacao/compras.md`
  - `SESSION_SUMMARY.md`

### Architectural Decisions

- Keep the generic procurement ledger from migration `000182`, but use normalized tables for the core inspection route/order/result/analysis lifecycle.
- Do not use FoccoERP screen codes in backend names, routes, tables or variables.
- Model FoccoERP concepts in VentureERP language:
  - roteiro -> receiving inspection route;
  - ordem -> receiving inspection order;
  - apontamento -> result;
  - análise -> analysis.
- The current implementation stores quality decisions; automatic stock movement on structured analysis is a next step because existing generic disposition already handles stock transfer and should be unified carefully.

### Validation Run

- `env GOCACHE=/tmp/panossoerp-go-build go test ./internal/application/usecase/procurement_uc ./internal/infrastructure/repository/procurement ./internal/interfaces/http/handler ./api`
- Focused compile/tests passed.

### Next Suprimentos Backlog

1. Connect structured inspection analysis to stock quarantine/rejection/available transfers, replacing or wrapping the older generic disposition endpoint.
2. Generate inspection orders automatically from purchase receipt, receiving notice or fiscal entry according to parameters.
3. Add inspection frequency/skip by item/classification/supplier and history of classification changes.
4. Add occurrence types and supplier-facing occurrence emails that feed IQF.
5. Add inspection labels/reports from structured route/order data.
- `internal/application/dto/request/purchase_receipt_dto.go`
- `internal/application/dto/response/purchase_order_response.go`
- `internal/application/usecase/purchase_order_uc/receive_purchase_order_uc.go`
- `internal/application/usecase/purchase_order_uc/receive_purchase_order_uc_test.go`
- `internal/domain/purchase_order/repository/purchase_order_repository.go`
- `internal/infrastructure/repository/purchase_order/purchase_order_repository_sqlc.go`
- `internal/interfaces/http/handler/purchase_order_handler.go`
- `scripts/test-purchase-receiving.sh`

### Validation Run

- `env GOCACHE=/tmp/panossoerp-go-build go test ./internal/application/usecase/purchase_order_uc ./internal/infrastructure/repository/purchase_order ./internal/interfaces/http/handler`
- `make test-purchase-receiving`
- `make test`
- `git diff --check`

All listed validation commands passed.

### Next Step

Wait for explicit user command before starting the next implementation.

Recommended next implementation:

1. Receiving inspection integrated with stock quarantine:
   - inspection route/plan (`FINS0200`)
   - inspection orders (`FINS0201`)
   - inspection appointments/results (`FINS0202`, `FINS0203`)
   - automatic generation from purchase receipt (`FINS0212`)
   - stock release/rejection from inspection warehouse

---

## Subsequent Session: Audit of Suprimentos Maturity + Inspection→Stock Loop Closure

### Goal

Audit everything built for the eight Suprimentos maturity blocks (inspection,
receiving notice/divergence, supplier evaluation/IQF, approval limits, contracts,
checklist/labels, EDI, import) against FoccoERP/SAP-style depth, verify it is correct
and compiles, then close the highest-value functional gap and refresh the roadmap.

### Audit result (what exists and at what depth)

- **Structured (normalized tables, real lifecycle):**
  - Receiving inspection core — migration `000183`: routes/steps/attributes, orders,
    results, analysis. Route resolution by item+mask then classification prefix. Good
    enterprise depth.
  - Purchase receipt by PO line — tolerance handling, stock `IN`, PO line/header
    status. Legacy `item_code` distribution bug already fixed in a prior session.
  - Supplier scorecard snapshots — migration `000182`, weighted IQF (40/30/20/10).
- **Ledger depth only (`procurement_records` typed rows + JSONB payload):**
  - Receiving notice + divergences (`FAVR`), approval limits (`FALC`), supplier
    contracts (`FCON`), checklist/labels (`FCLR`/`FINS0304`), EDI (`FEDS`),
    import/nationalization (`FREC0203`/`FIMP`). These are recorded and status-tracked
    but not yet enforced/normalized. This is the correct honest state — usable as an
    operational ledger, not yet a rule engine.
- **Verified:** `go build`, `go vet`, `go test ./...` all green with the pre-existing
  uncommitted work; no regressions.

### What was implemented this session

- **Closed the inspection→stock loop on the structured path** (the roadmap's own #1
  next step). Previously `AnalyzeReceivingInspectionOrder` only recorded quantities
  and status; only the older generic `DisposeInspection` (over `procurement_records`)
  moved stock. Now the structured analysis can move stock too:
  - `AnalyzeReceivingInspectionOrderDTO` gained `move_stock`,
    `destination_warehouse_id`, `rejection_warehouse_id`, `rework_warehouse_id`,
    `restricted_warehouse_id`.
  - With `move_stock: true`, analyzed quantities leave the order's inspection
    warehouse by `TRANSFER_OUT`/`TRANSFER_IN` (reference `RECEIVING_INSPECTION_ANALYSIS`):
    conform + restricted → available destination (restricted may use its own
    warehouse), rework → rework warehouse, rejected → blocked/return warehouse.
  - Validations: requires stock-movement permission; destination required when
    approving; total analyzed cannot exceed order quantity; zero/missing/self legs
    skipped. Movements are returned in `ReceivingInspectionAnalysisResponse.movements`.
  - Behavior is backward compatible: without `move_stock` the endpoint records only.
- **Refactor:** extracted the leg-planning into a pure function
  `planInspectionStockLegs` and reused `transferStock` (the old `transfer(rec,...)`
  now delegates to it).
- **Unit test:** `procurement_uc_test.go` covers leg routing, restricted→destination
  fallback, and skipping zero/missing/self-transfer legs. No DB/mocks needed.

### Files modified

- `internal/application/dto/request/receiving_inspection_dto.go`
- `internal/application/dto/response/receiving_inspection_response.go`
- `internal/application/usecase/procurement_uc/procurement_uc.go`
- `internal/application/usecase/procurement_uc/procurement_uc_test.go` (new)
- `docs/dev/API_REQUEST_BODIES.txt`
- `docs/dev/manufatura-e-compras.md`
- `docs/dev/maturidade-erp-roadmap.md`
- `docs/apresentacao/compras.md`
- `SESSION_SUMMARY.md`

### Validation Run

- `gofmt -l` clean on changed files
- `env GOCACHE=/tmp/panossoerp-go-build GOFLAGS=-mod=vendor go build ./...`
- `env GOCACHE=/tmp/panossoerp-go-build GOFLAGS=-mod=vendor go vet ./...`
- `env GOCACHE=/tmp/panossoerp-go-build GOFLAGS=-mod=vendor go test ./...` (all green;
  integration packages still need a migrated Postgres and were not run here)

### Prioritized backlog to raise the ledger areas to structured depth

Ranked by factory impact; validate scope with the business before building:

1. **Auto-generate inspection order from purchase receipt (`FINS0212`)** — when a PO
   line is received and an active route matches, receive into the inspection
   warehouse and open the order automatically (today it is manual via API).
2. **IQF auto-computation** — compute the scorecard for a supplier+period from real
   receipts/inspection analyses/divergences instead of manual entry.
3. **Approval limits enforcement (`FALC0201`)** — normalized limit rules + block PO
   approval above limit without the required approver; unblock endpoint.
4. **Supplier contracts normalized (`FCON`)** — contract header + item lines with
   negotiated price, contracted balance, consumption against PO, index readjustment.
5. **Receiving notice normalized (`FAVR`)** — dock schedule + divergence types
   feeding IQF, entry-of-invoice-from-notice.
6. **Real EDI engine (`FEDS`)** — layout parser, inbound/outbound queues, auto NF-e.
7. **Import landed cost (`FREC0203`/`FIMP`)** — DI/DUIMP, expenses, FX, per-item
   nationalized cost.

---

## Subsequent Session: Suprimentos Governance — FINS0212, IQF auto, FALC, FCON, CPDC0403

### Goal

Implement the prioritized backlog to raise the ledger-depth Suprimentos areas to
structured/enforced depth, with tests, docs and a fresh FoccoERP comparison.

### What was implemented (migration `000184_procurement_governance`)

1. **FINS0212 — auto-inspection on receipt.** `ReceivePurchaseOrderUseCase` gained an
   optional `ReceivingInspectionGate` port (implemented by the procurement use case).
   When a received PO line matches an active inspection route, the material is
   received into the route's inspection warehouse (not the requested one) and a
   `PURCHASE_RECEIPT`-sourced inspection order is opened automatically. Response adds
   `inspection_orders` and per-line `under_inspection`. No route ⇒ prior behaviour.
2. **IQF auto-computation.** `POST /api/procurement/supplier-scorecards/compute`
   derives quality (from inspection orders) and delivery (from late PO lines) for a
   supplier/period; commercial/service stay manual (default 100); overall keeps
   40/30/20/10. `persist:true` also stores it. New repo aggregate
   `AggregateSupplierPerformance`.
3. **FALC — approval limits with real enforcement.** New table
   `purchase_approval_limits` (scope GLOBAL/SUPPLIER/COST_CENTER/CATEGORY,
   `auto_approve_max`, `block_above`). New `ApprovePurchaseOrderUseCase` +
   `ApprovalPolicy` port (implemented by procurement): `POST /api/purchase-order/
   {code}/approve` evaluates the most specific rule and approves / blocks
   (`alcada_status=B`) / hard-rejects (`R`); `POST .../authorize` (ADMIN) releases a
   blocked order. Uses the pre-existing `alcada_status` field. No rule ⇒ auto-approve.
4. **FCON — normalized supplier contracts.** New tables `supplier_contracts` +
   `supplier_contract_items` (contracted/consumed qty, price, min order). CRUD +
   status + atomic `.../consume` (rejects over-balance, requires ACTIVE) under
   `/api/procurement/supplier-contracts`. `remaining_qty` exposed per line.
5. **CPDC0403 — consolidated purchase movement history.**
   `GET /api/procurement/purchase-movements` (requested/received/cancelled/open,
   price, dates; filter by supplier/item).

### Architecture / decisions

- Cross-package coupling kept clean via consumer-defined ports in `purchase_order_uc`
  (`ReceivingInspectionGate`, `ApprovalPolicy`), implemented by `procurement_uc`. The
  purchase order package does not import the procurement use case; it only knows the
  interfaces (procurement domain entity used for the decision type). No import cycle.
- Approval authority is currently expressed by route role (`authorize` gated to
  ADMIN). Per-approver hierarchical levels deferred.
- Contract consumption is an explicit endpoint for now; auto-consume on PO line
  commitment with `contract_code` is the documented next wiring step.
- Governance tables use raw pgx (consistent with the existing procurement repo), so
  no sqlc regeneration was needed.

### Tests

- Unit (no DB): `planInspectionStockLegs`, `ratioScore`, `overallIQF`
  (`procurement_uc_test.go`); `SupplierContractItem.RemainingQty`
  (`domain/procurement/entity/entity_test.go`).
- Script + make target: `scripts/test-procurement-governance.sh` /
  `make test-procurement-governance` — runs the focused unit tests and, when
  `BASE_URL` points at a running server, an HTTP smoke of the new endpoints
  (approval-limit create/list, contract create/consume incl. over-balance 422,
  scorecard compute, purchase-movements).

### Files (new/changed)

- migrations `000184_procurement_governance.{up,down}.sql`
- `internal/domain/procurement/entity/entity.go` (+ `entity_test.go`)
- `internal/domain/procurement/repository/repository.go`
- `internal/infrastructure/repository/procurement/repository.go`
- `internal/application/usecase/procurement_uc/procurement_governance_uc.go` (new)
- `internal/application/usecase/procurement_uc/procurement_uc_test.go`
- `internal/application/usecase/purchase_order_uc/approve_purchase_order_uc.go` (new)
- `internal/application/usecase/purchase_order_uc/receive_purchase_order_uc.go`
- `internal/application/dto/request/procurement_maturity_dto.go`
- `internal/application/dto/response/procurement_maturity_response.go`
- `internal/application/dto/response/purchase_order_response.go`
- `internal/interfaces/http/handler/procurement_handler.go`
- `internal/interfaces/http/handler/purchase_order_handler.go`
- `api/api.go`, `makefile`, `scripts/test-procurement-governance.sh`
- docs: `maturidade-erp-roadmap.md`, `manufatura-e-compras.md`,
  `API_REQUEST_BODIES.txt`, `apresentacao/compras.md`

### Validation

- `gofmt -l` clean; `go build ./...`; `go vet ./...`; `go test ./...` all green.
- Migration not applied here (needs Postgres); SQL validated against real
  purchase_orders/purchase_order_items columns (incl. `promised_date` from mig 000140).

### Still open (validate scope before building)

- Real EDI engine (`FEDS`): layout parser, inbound/outbound queues, auto NF-e.
- Import landed cost (`FREC0203`/`FIMP`): DI/DUIMP, expenses, FX, per-item cost.
- Rural producer counter-invoice (`FREC0201`); freight quotation/order
  (`FCOT0200 FRE`/`FPDC0200 FRE`); service PO screen (`FPDC0200 SER`).
- Parameter panels (`FUTL0125 *`); inspection frequency/skip; report screens
  (`FCOT0300`/`FPDC0250`/`FPDC0251`).

---

## Subsequent Session: Suprimentos 100% close-out (migration 000185)

### Goal

Close the remaining backend gaps in Compras/Suprimentos for a metalworking/furniture
factory (rural flows out of scope) so the sector is functionally complete and the
next sector (Comercial) can begin.

### Implemented (migration 000185_suprimentos_closeout)

1. **FAVR — Receiving notice + divergences (normalized).** `receiving_notices`
   (+items) = dock schedule/conference before the invoice (SCHEDULED→ARRIVED→
   IN_CONFERENCE→RELEASED/BLOCKED/CANCELLED, `blocked` flag); `receiving_divergences`
   = formal shortage/excess/damage/wrong-item/price/document/late with resolution
   (accept/return/waive/supplier-debit), queryable by supplier — feeds IQF.
2. **FEDS — Supplier EDI structured.** `supplier_edi_messages`(+lines), inbound/
   outbound typed messages; per-line divergence detection (QTY/PRICE/DATE) vs. the PO
   reference values with tolerance (`entity.DetectEDILineDivergence`), message status
   PROCESSED/WITH_DIVERGENCE/SENT.
3. **FREC0203/FIMP — Import landed cost.** `import_processes`(+items,+expenses):
   currency/FX/incoterm/DI-DUIMP ref; cost-composing expenses apportioned by
   VALUE/WEIGHT/QUANTITY into per-item `landed_unit_cost` via pure
   `entity.ComputeLandedCosts`; `/recompute` and `/status` (nationalize).
4. **FUTL0125 — Procurement parameters panel.** `procurement_parameters` typed
   key/value per domain (PURCHASE_TABLE/PURCHASE_ORDER/QUOTATION/REQUISITION/
   RECEIVING_NOTICE/INSPECTION/SUPPLIER_EVALUATION/CONTRACT/SUPPLIER/NF_ENTRY), UPSERT,
   ADMIN write.
5. **FAVF0203 — Supplier homologation.** `supplier_homologations`: status derived from
   period IQF by thresholds (`entity.HomologationStatusForIQF`, default 80/60) or set
   manually; keeps IQF, category, validity.
6. **FFOR0204 — Generate item-supplier links** from purchase history in one
   INSERT…SELECT…ON CONFLICT DO NOTHING.

### Audit of the user-provided matrix

Confirmed accurate. Reclassified the "Parcial — falta tela/relatório/console/UI" rows
as **frontend** (backend data already exposed): FPDC0205, FPDC0202, FPLA0203,
CPDC0402, FCOT0201, FINS0207, FREC0255, and the report screens FCOT0300/FPDC0250/
FPDC0251. Out of scope for this client: FREC0201 (rural counter-invoice). Remaining
low-value/optional: own freight quotation/order flows, EDI VAN-layout parser + auto
NF-e emission. Sector is now **functionally closed in the backend**.

### Architecture / decisions

- All new tables/repos use raw pgx in the procurement package (consistent), no sqlc
  regen. Bug-prone logic (landed-cost apportionment, EDI divergence, homologation
  rule) extracted as pure domain functions and unit-tested.
- EDI stays decoupled from the purchase order repo: the caller passes PO reference
  values per line (the integration that parses the EDI already has the PO).

### Tests

- Unit (no DB): `internal/domain/procurement/entity/closeout_test.go` —
  `ComputeLandedCosts` (by-value + equal-split), `DetectEDILineDivergence`,
  `HomologationStatusForIQF`. Writing these caught a bad test expectation (price 0.2
  beyond 0.1 tolerance is correctly a divergence).
- `scripts/test-procurement-governance.sh` extended: HTTP smoke now also covers
  receiving-notice/divergence, EDI (asserts WITH_DIVERGENCE), import (asserts landed
  20/60), parameters, homologation, generate-items. `make test-procurement-governance`.

### Files (new/changed)

- migrations `000185_suprimentos_closeout.{up,down}.sql`
- `internal/domain/procurement/entity/closeout.go` (+ `closeout_test.go`)
- `internal/domain/procurement/repository/repository.go`
- `internal/infrastructure/repository/procurement/repository_closeout.go` (new)
- `internal/application/usecase/procurement_uc/procurement_closeout_uc.go` (new)
- `internal/application/dto/request/procurement_closeout_dto.go` (new)
- `internal/application/dto/response/procurement_closeout_response.go` (new)
- `internal/interfaces/http/handler/procurement_handler.go`
- `api/api.go`, `scripts/test-procurement-governance.sh`
- docs: `maturidade-erp-roadmap.md` (ranking now marks Suprimentos closed; Comercial
  is next), `manufatura-e-compras.md`, `API_REQUEST_BODIES.txt`, `apresentacao/compras.md`

### Validation

- `gofmt -l` clean; `go build ./...`; `go vet ./...`; `go test ./...` all green.
- Migrations 000184/000185 not applied here (need Postgres); SQL written against real
  columns. Run `make migrate_up` then `BASE_URL=... make test-procurement-governance`
  for the HTTP smoke.

### Next sector

Comercial (assistência técnica, CRM/pós-venda, metas, EDI cliente, políticas
comerciais). Await user go-ahead.
## 2026-07-04 — Comercial fase 1: Precificacao

Branch de trabalho criada a partir de `feature/routing-enterprise`:
`feature/comercial-fase-1-precificacao` (o nome pedido tinha typo
`routing-enmterprise`; a branch existente era `routing-enterprise`).

### Escopo entregue — fechamento completo, refeito com help Focco

Fase 1 do roadmap Comercial concluida no backend após releitura das páginas reais do
help Focco. Referências usadas:

- `FCST0205` — Formação do Preço de Venda:
  `https://help.foccoerp.com.br/Programas/FoccoERP/Custos/Forma%C3%A7%C3%A3o%20do%20Pre%C3%A7o%20de%20Venda/FCST0205/?h=fcst0205`
- `FCST0262 PREC` — Precificação de Produtos:
  `https://help.foccoerp.com.br/Programas/FoccoERP/Comercial/Precifica%C3%A7%C3%A3o/FCST0262_PREC/?h=fcst0262`
- `FPRV0200` — Cadastro da Tabela de Vendas:
  `https://help.foccoerp.com.br/Programas/FoccoERP/Cadastros%20Auxiliares/Comercial/Pedido%20de%20Venda/Tabela%20de%20Venda/FPRV0200/?h=fprv0200`
- `FPRV0201` — Cadastro de Preços da Tabela de Vendas:
  `https://help.foccoerp.com.br/Programas/FoccoERP/Cadastros%20Auxiliares/Comercial/Pedido%20de%20Venda/Tabela%20de%20Venda/FPRV0201/?h=fprv0201`
- `FPPV0200 FPDV/PREC` — Cadastro de Políticas de Formação de Preço de Venda:
  `https://help.foccoerp.com.br/Programas/FoccoERP/Comercial/Pol%C3%ADtica%20Formata%C3%A7%C3%A3o%20do%20Pre%C3%A7o%20de%20Venda/FPPV0200/?h=fppv0200`

Observação: o prompt citava `FPDV0200` para política, mas o help Focco localiza o
cadastro de políticas como `FPPV0200` com escopos `FPPV` e `PREC`.

A primeira entrega estava parcial; a correção refez a base com os comportamentos do
Focco: fórmula `FCST0205`, tolerâncias/regras de tabela `FPRV0200`, validações de
preço manual `FPRV0201`, prioridade/sequência/vigência/margens/incidências da
política `FPPV0200`, fontes de custo, geração/reprecificação e histórico.

1. **Tabela de vendas**: consulta por codigo e update por codigo em
   `/api/customers/sales-tables/{tableCode}`.
2. **Precos da tabela**: as rotas `/{tableCode}/prices` agora trabalham por codigo
   de tabela, resolvendo internamente o ID. Corrigido o contrato antigo que exigia
   `sales_table_id` no body mesmo quando a rota ja estava dentro da tabela.
3. **Precificacao de produto**: novo `POST /api/customers/sales-tables/pricing`
   resolve preco unitario e total bruto por tabela/item/quantidade, validando tabela
   ativa/vigente, preco nao bloqueado e situacao diferente de `INATIVO`.
4. **Formacao do preco de venda**: novo
   `POST /api/customers/sales-tables/price-formation`, calculando preco sugerido a
   partir de custo base, markup e/ou margem + cargas comerciais (impostos, frete,
   comissao, desconto, despesas), respeitando casas decimais da tabela quando
   informada.
5. **Politica de formacao de preco**: migration `000186_comercial_pricing_closeout`
   criou `sales_price_policies` e endpoints `/api/customers/sales-price-policies`.
   A politica guarda fonte de custo (`INFORMED`, `STANDARD_TOTAL`,
   `STANDARD_MATERIAL`, `PURCHASE`, `STOCK_AVG`, `STOCK_LAST`),
   prioridade/sequencia, escopo `FPPV`/`PREC`, tipos, margem minima/maxima/ideal,
   incidencias JSON, percentuais, vigencia e tabela default.
6. **Geracao/reprecificacao da tabela**:
   `POST /api/customers/sales-tables/generate-prices` busca custo no ERP, calcula
   preco pela politica, faz upsert em `sales_table_prices` e registra
   `sales_table_price_history`.
7. **Historico**:
   `GET /api/customers/sales-tables/{tableCode}/price-history?item_code=...`.

### Arquivos principais

- `migrations/000186_comercial_pricing_closeout.{up,down}.sql`
- `api/api.go`
- `internal/domain/customer/entity/entity.go`
- `internal/domain/customer/entity/pricing_test.go`
- `internal/domain/customer/repository/repository.go`
- `internal/application/dto/request/customer_dto.go`
- `internal/application/dto/response/customer_response.go`
- `internal/application/usecase/customer_uc/customer_uc.go`
- `internal/infrastructure/repository/customer/customer_repository_sqlc.go`
- `internal/interfaces/http/handler/customer_handler.go`
- `scripts/test-comercial-pricing.sh`
- Docs: `docs/dev/vendas.md`, `docs/apresentacao/vendas.md`,
  `docs/dev/API_REQUEST_BODIES.txt`

### Ajuste de documentação pós-fase 1

O usuário pediu que a documentação não faça menção a telas/códigos/sistemas de
referência externos. A documentação deve refletir exclusivamente o VentureERP: como
o sistema funciona, age, calcula, valida e expõe APIs.

Alterações feitas:

- Removidos `docs/dev/comercial-roadmap.md` e `docs/dev/maturidade-erp-roadmap.md`.
- Removidas referências a roadmaps comparativos do `docs/README.md`.
- Sanitizados `docs/dev/vendas.md`, `docs/apresentacao/plano-de-corte.md`,
  `docs/dev/plano-de-corte.md`, `docs/dev/custos.md` e
  `docs/dev/manufatura-e-compras.md` para não citar Focco/códigos de tela.
- `rg` em `docs/` não retorna mais `Focco`, códigos de telas comerciais usados como
  referência, nem links para os roadmaps removidos.

Regra para próximas fases: usar fontes externas apenas como referência de pesquisa e
implementação. Documentação final deve ser produto puro do VentureERP, sem mencionar
as telas/códigos externos.

### Validacao

- `env GOCACHE=/tmp/panossoerp-go-build go test ./...` passou.
- `./scripts/test-comercial-pricing.sh` passou em modo unitario; smoke HTTP e
  pulado quando `BASE_URL` nao esta definido.
- Primeira tentativa de `go test` sem `GOCACHE` falhou por sandbox/cache Go em
  `~/.cache/go-build` somente leitura; usar `/tmp/panossoerp-go-build`.

### Proximo passo

Aguardar comando do usuario para iniciar a fase 2: Politica Comercial
(desconto/acrescimo/frete/comissao/regras).

## 2026-07-04 — Comercial fase 2: Politica Comercial

Fase 2 refeita apos crítica correta do usuário: a primeira versão tinha sido
marcada como concluída sem validar as páginas do help. Depois disso as páginas
corretas foram localizadas pelo índice de Programas > Comercial > Política
Comercial e a implementação foi ajustada para seguir o modelo real, melhorando-o
no VentureERP. Documentação final de produto continua sem citar telas/códigos
externos; este resumo registra as fontes para agentes.

Referências usadas:

- `FPDV0108` — Cadastro da Política Comercial de Descontos:
  `https://help.foccoerp.com.br/Programas/FoccoERP/Comercial/Pol%C3%ADtica%20Comercial/FPDV0108/`
- `FPDV0109` — Cadastro da Política Comercial de Acréscimo:
  `https://help.foccoerp.com.br/Programas/FoccoERP/Comercial/Pol%C3%ADtica%20Comercial/FPDV0109/`
- `FPDV0110` — Cadastro da Política Comercial de Comissões:
  `https://help.foccoerp.com.br/Programas/FoccoERP/Comercial/Pol%C3%ADtica%20Comercial/FPDV0110/`
- `FPDV0111` — Cadastro da Política Comercial de Fretes:
  `https://help.foccoerp.com.br/Programas/FoccoERP/Comercial/Pol%C3%ADtica%20Comercial/FPDV0111/`
- `FPDV0115` — Cadastro de Regras (Configurador de Produto):
  `https://help.foccoerp.com.br/Programas/FoccoERP/Comercial/Pol%C3%ADtica%20Comercial/FPDV0115/`
- `FPDV0117` — Cadastro de Itens/Classificações com Políticas Específicas:
  `https://help.foccoerp.com.br/Programas/FoccoERP/Comercial/Pol%C3%ADtica%20Comercial/FPDV0117/`
- `FPDV0250` — Relatório da Política Comercial de Descontos/Acréscimos/Comissões:
  `https://help.foccoerp.com.br/Programas/FoccoERP/Comercial/Pol%C3%ADtica%20Comercial/Relat%C3%B3rios/FPDV0250/`

### Escopo entregue

1. **Motor de política comercial** em `internal/domain/customer/entity`:
   - tipos `DISCOUNT`, `SURCHARGE`, `FREIGHT`, `COMMISSION`;
   - cálculo `PERCENT` ou `VALUE`;
   - capa da política com prioridade, sequência, validade e tipo de escolha
     (`INFORMATION`, `CHOICE`, `OPTIONAL`);
   - até seis dimensões comerciais combináveis em `data_types_json`;
   - flags de alteração manual, valores maiores, uso em comissão, aplicação por
     item e abatimento da base de comissão;
   - linhas/faixas efetivas de política com número da linha, sequência, vigência
     própria, variáveis, tipo percentual/valor, valor mínimo e máximo;
   - filtros por cliente, tipo, segmento, região, tabela, condição, transportadora,
     item, máscara, linha e classificação;
   - avaliação retorna desconto, acréscimo, frete, comissão, valor líquido e efeitos.
2. **Persistência**:
   - migration `000187_commercial_policies` cria `commercial_policies` e
     `commercial_policy_lines` e `commercial_policy_specific_items`.
3. **API** em `/api/customers/support/commercial-policies`:
   - CRUD/listagem/exportação;
   - `POST /evaluate` para simular/apurar políticas;
   - `/{code}/lines` para linhas/faixas de política;
   - `/{code}/specific-items` para vínculos por item/linha/classificação.
4. **Itens/classificações específicas**:
   - validade por item/classificação;
   - flags equivalentes a D/A/D-A/M: bloqueio de desconto, bloqueio de acréscimo,
     ignorar políticas do item e bloquear alteração manual.
5. **Relatório operacional**:
   - `GET /api/customers/support/commercial-policies?kind=...`
   - a listagem usa o mecanismo de exportação já existente via `format`.

### Arquivos principais

- `migrations/000187_commercial_policies.{up,down}.sql`
- `api/api.go`
- `internal/domain/customer/entity/entity.go`
- `internal/domain/customer/entity/commercial_policy_test.go`
- `internal/domain/customer/repository/repository.go`
- `internal/application/dto/request/customer_dto.go`
- `internal/application/dto/response/customer_response.go`
- `internal/application/usecase/customer_uc/customer_uc.go`
- `internal/infrastructure/repository/customer/customer_repository_sqlc.go`
- `internal/interfaces/http/handler/customer_handler.go`
- `scripts/test-comercial-politicas.sh`
- Docs: `docs/dev/vendas.md`, `docs/apresentacao/vendas.md`,
  `docs/dev/API_REQUEST_BODIES.txt`

### Validação

- `env GOCACHE=/tmp/panossoerp-go-build go test ./...` passou.
- `./scripts/test-comercial-politicas.sh` passou em modo unitário; smoke HTTP foi
  pulado porque `BASE_URL` não estava definido.

### Complemento de documentação pós-feedback

O usuário observou corretamente que documentação enterprise não pode apenas listar
rotas. `docs/dev/vendas.md` foi expandido para explicar propósito da política
comercial, quando usar cada tipo, estrutura de capa/linhas, itens/classificações
específicas, ordem de aplicação, exemplos de configuração, integração com
precificação/orçamento/pedido/representantes/faturamento e validações/cuidados.

`docs/apresentacao/vendas.md` foi expandido com visão de negócio: para que serve,
como funciona na venda, exemplos práticos e benefício operacional.

### Próximo passo

Aguardar comando do usuário para iniciar a fase 3: Orçamento.
