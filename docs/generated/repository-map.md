# VentureERP — Repository Map

> Arquivo gerado automaticamente.
> Não editar manualmente.

- Commit: `bd339dc`
- Gerado em: `2026-07-10T15:47:10-03:00`
- Módulo: `github.com/FelipePn10/panossoerp`

## Resumo

- Arquivos Go: 1203
- Arquivos de teste: 114
- Queries SQL: 446
- Migrations: 401

## Documentação para agentes

- `AGENTS.md`
- `api/AGENTS.md`
- `cmd/AGENTS.md`
- `docs/AGENTS.md`
- `internal/AGENTS.md`
- `internal/application/AGENTS.md`
- `internal/domain/AGENTS.md`
- `internal/infrastructure/AGENTS.md`
- `internal/interfaces/AGENTS.md`
- `internal/pkg/AGENTS.md`
- `migrations/AGENTS.md`
- `observability/AGENTS.md`
- `scripts/AGENTS.md`

## Pacotes Go

```text
github.com/FelipePn10/panossoerp/api
github.com/FelipePn10/panossoerp/cmd/cutting-samples
github.com/FelipePn10/panossoerp/internal/application/dto/request
github.com/FelipePn10/panossoerp/internal/application/dto/response
github.com/FelipePn10/panossoerp/internal/application/ports
github.com/FelipePn10/panossoerp/internal/application/security
github.com/FelipePn10/panossoerp/internal/application/usecase/accounting_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/allocation_base_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/aps_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/bom_header_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/cnpj_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/component_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/configurator_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/consumer_service_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/cost_center_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/cost_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/crp_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/customer_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/cutting_plan_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_promise_params_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_promise_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_reschedule_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/drawing_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/employee
github.com/FelipePn10/panossoerp/internal/application/usecase/enterprise_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/entry_operation_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/errors
github.com/FelipePn10/panossoerp/internal/application/usecase/financial_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_classification_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_params_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/forecast_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/group_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/ibpt_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/independent_demand_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/industrial_calendar
github.com/FelipePn10/panossoerp/internal/application/usecase/item_calendar_promise_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/item_classification_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/item_conversion_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/item_supplier_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/item_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/location_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/lot_mask_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/machine_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/maintenance_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/modifier_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_calculation_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/nfse_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/order_priority_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/overhead_allocation_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/planned_order_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/planning_params_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/planning_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/procurement_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/production_plan_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/product_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_price_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_quotation_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_requisition_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/quality_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/recurring_sales_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/representative_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/restriction_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/routing_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/sales_division_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/sales_forecast_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/sales_goal_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/sales_order_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/sales_quotation_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/shipment_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/stock_movement_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/stock_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/structure_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/supplier_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/technical_assistance_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/tool_sheet_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/tool_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/user_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/warehouse_uc
github.com/FelipePn10/panossoerp/internal/domain/accounting/ecd
github.com/FelipePn10/panossoerp/internal/domain/accounting/entity
github.com/FelipePn10/panossoerp/internal/domain/accounting/repository
github.com/FelipePn10/panossoerp/internal/domain/allocation_base/entity
github.com/FelipePn10/panossoerp/internal/domain/allocation_base/repository
github.com/FelipePn10/panossoerp/internal/domain/aps/entity
github.com/FelipePn10/panossoerp/internal/domain/aps/repository
github.com/FelipePn10/panossoerp/internal/domain/bom_header/entity
github.com/FelipePn10/panossoerp/internal/domain/bom_header/repository
github.com/FelipePn10/panossoerp/internal/domain/cnpj/entity
github.com/FelipePn10/panossoerp/internal/domain/cnpj/service
github.com/FelipePn10/panossoerp/internal/domain/component/entity
github.com/FelipePn10/panossoerp/internal/domain/component/repository
github.com/FelipePn10/panossoerp/internal/domain/component/valueobject
github.com/FelipePn10/panossoerp/internal/domain/configurator/entity
github.com/FelipePn10/panossoerp/internal/domain/consumer_service/entity
github.com/FelipePn10/panossoerp/internal/domain/consumer_service/repository
github.com/FelipePn10/panossoerp/internal/domain/cost_center/entity
github.com/FelipePn10/panossoerp/internal/domain/cost_center/repository
github.com/FelipePn10/panossoerp/internal/domain/crp/entity
github.com/FelipePn10/panossoerp/internal/domain/crp/repository
github.com/FelipePn10/panossoerp/internal/domain/customer/entity
github.com/FelipePn10/panossoerp/internal/domain/customer/repository
github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity
github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/repository
github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/service
github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/service/lp
github.com/FelipePn10/panossoerp/internal/domain/delivery_promise/entity
github.com/FelipePn10/panossoerp/internal/domain/delivery_promise_params/entity
github.com/FelipePn10/panossoerp/internal/domain/delivery_promise_params/repository
github.com/FelipePn10/panossoerp/internal/domain/delivery_promise/repository
github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/entity
github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/repository
github.com/FelipePn10/panossoerp/internal/domain/drawing/entity
github.com/FelipePn10/panossoerp/internal/domain/employee/entity
github.com/FelipePn10/panossoerp/internal/domain/employee/repository
github.com/FelipePn10/panossoerp/internal/domain/enterprise/entity
github.com/FelipePn10/panossoerp/internal/domain/enterprise/repository
github.com/FelipePn10/panossoerp/internal/domain/entry_operation/entity
github.com/FelipePn10/panossoerp/internal/domain/entry_operation/repository
github.com/FelipePn10/panossoerp/internal/domain/enums/types
github.com/FelipePn10/panossoerp/internal/domain/financial/entity
github.com/FelipePn10/panossoerp/internal/domain/financial/repository
github.com/FelipePn10/panossoerp/internal/domain/fiscal_classification/entity
github.com/FelipePn10/panossoerp/internal/domain/fiscal_classification/repository
github.com/FelipePn10/panossoerp/internal/domain/fiscal/engine
github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity
github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository
github.com/FelipePn10/panossoerp/internal/domain/fiscal/sped
github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/entity
github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/mask/service
github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/repository
github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject
github.com/FelipePn10/panossoerp/internal/domain/group/entity
github.com/FelipePn10/panossoerp/internal/domain/group/repository
github.com/FelipePn10/panossoerp/internal/domain/ibpt/entity
github.com/FelipePn10/panossoerp/internal/domain/ibpt/repository
github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity
github.com/FelipePn10/panossoerp/internal/domain/independent_demand/repository
github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/entity
github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/repository
github.com/FelipePn10/panossoerp/internal/domain/item_calendar_promise/entity
github.com/FelipePn10/panossoerp/internal/domain/item_calendar_promise/repository
github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity
github.com/FelipePn10/panossoerp/internal/domain/item_conversion/repository
github.com/FelipePn10/panossoerp/internal/domain/items/entity
github.com/FelipePn10/panossoerp/internal/domain/items/repository
github.com/FelipePn10/panossoerp/internal/domain/item_supplier/entity
github.com/FelipePn10/panossoerp/internal/domain/item_supplier/repository
github.com/FelipePn10/panossoerp/internal/domain/items/valueobject
github.com/FelipePn10/panossoerp/internal/domain/location/entity
github.com/FelipePn10/panossoerp/internal/domain/location/repository
github.com/FelipePn10/panossoerp/internal/domain/lot_mask/entity
github.com/FelipePn10/panossoerp/internal/domain/machine/entity
github.com/FelipePn10/panossoerp/internal/domain/machine/repository
github.com/FelipePn10/panossoerp/internal/domain/machine/service
github.com/FelipePn10/panossoerp/internal/domain/maintenance/entity
github.com/FelipePn10/panossoerp/internal/domain/maintenance/repository
github.com/FelipePn10/panossoerp/internal/domain/modifier/entity
github.com/FelipePn10/panossoerp/internal/domain/modifier/repository
github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity
github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/ports
github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository
github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/service
github.com/FelipePn10/panossoerp/internal/domain/nfse/entity
github.com/FelipePn10/panossoerp/internal/domain/nfse/repository
github.com/FelipePn10/panossoerp/internal/domain/order_priority/entity
github.com/FelipePn10/panossoerp/internal/domain/order_priority/repository
github.com/FelipePn10/panossoerp/internal/domain/overhead_allocation/entity
github.com/FelipePn10/panossoerp/internal/domain/overhead_allocation/repository
github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity
github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository
github.com/FelipePn10/panossoerp/internal/domain/planning_params/entity
github.com/FelipePn10/panossoerp/internal/domain/planning_params/repository
github.com/FelipePn10/panossoerp/internal/domain/procurement/entity
github.com/FelipePn10/panossoerp/internal/domain/procurement/repository
github.com/FelipePn10/panossoerp/internal/domain/product/entity
github.com/FelipePn10/panossoerp/internal/domain/production_order/entity
github.com/FelipePn10/panossoerp/internal/domain/production_order/repository
github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity
github.com/FelipePn10/panossoerp/internal/domain/production_plan/repository
github.com/FelipePn10/panossoerp/internal/domain/product/repository
github.com/FelipePn10/panossoerp/internal/domain/product/valueobject
github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity
github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository
github.com/FelipePn10/panossoerp/internal/domain/purchase_price/entity
github.com/FelipePn10/panossoerp/internal/domain/purchase_price/repository
github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/entity
github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/repository
github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/entity
github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/repository
github.com/FelipePn10/panossoerp/internal/domain/quality/entity
github.com/FelipePn10/panossoerp/internal/domain/quality/repository
github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/entity
github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/repository
github.com/FelipePn10/panossoerp/internal/domain/representative/entity
github.com/FelipePn10/panossoerp/internal/domain/representative/repository
github.com/FelipePn10/panossoerp/internal/domain/restriction/entity
github.com/FelipePn10/panossoerp/internal/domain/restriction/repository
github.com/FelipePn10/panossoerp/internal/domain/routing/entity
github.com/FelipePn10/panossoerp/internal/domain/routing/repository
github.com/FelipePn10/panossoerp/internal/domain/sales_division/entity
github.com/FelipePn10/panossoerp/internal/domain/sales_division/repository
github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/entity
github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/repository
github.com/FelipePn10/panossoerp/internal/domain/sales_goal/entity
github.com/FelipePn10/panossoerp/internal/domain/sales_goal/repository
github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity
github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository
github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity
github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/repository
github.com/FelipePn10/panossoerp/internal/domain/shipment/entity
github.com/FelipePn10/panossoerp/internal/domain/shipment/repository
github.com/FelipePn10/panossoerp/internal/domain/standard_cost/entity
github.com/FelipePn10/panossoerp/internal/domain/standard_cost/repository
github.com/FelipePn10/panossoerp/internal/domain/stock/entity
github.com/FelipePn10/panossoerp/internal/domain/stock_movement/entity
github.com/FelipePn10/panossoerp/internal/domain/stock_movement/repository
github.com/FelipePn10/panossoerp/internal/domain/stock/repository
github.com/FelipePn10/panossoerp/internal/domain/structure/entity
github.com/FelipePn10/panossoerp/internal/domain/structure/formula
github.com/FelipePn10/panossoerp/internal/domain/structure_query/repository
github.com/FelipePn10/panossoerp/internal/domain/structure_query/service
github.com/FelipePn10/panossoerp/internal/domain/structure/repository
github.com/FelipePn10/panossoerp/internal/domain/structure/valueobject
github.com/FelipePn10/panossoerp/internal/domain/supplier/entity
github.com/FelipePn10/panossoerp/internal/domain/supplier/repository
github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/entity
github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/repository
github.com/FelipePn10/panossoerp/internal/domain/tool/entity
github.com/FelipePn10/panossoerp/internal/domain/tool/repository
github.com/FelipePn10/panossoerp/internal/domain/user/entity
github.com/FelipePn10/panossoerp/internal/domain/user/repository
github.com/FelipePn10/panossoerp/internal/domain/warehouse/entity
github.com/FelipePn10/panossoerp/internal/domain/warehouse/repository
github.com/FelipePn10/panossoerp/internal/infrastructure/audit
github.com/FelipePn10/panossoerp/internal/infrastructure/auth
github.com/FelipePn10/panossoerp/internal/infrastructure/cnab
github.com/FelipePn10/panossoerp/internal/infrastructure/cnpj
github.com/FelipePn10/panossoerp/internal/infrastructure/config
github.com/FelipePn10/panossoerp/internal/infrastructure/database
github.com/FelipePn10/panossoerp/internal/infrastructure/database/nullable
github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil
github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc
github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqltypes
github.com/FelipePn10/panossoerp/internal/infrastructure/export
github.com/FelipePn10/panossoerp/internal/infrastructure/export/gantt
github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit
github.com/FelipePn10/panossoerp/internal/infrastructure/export/romaneio
github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe
github.com/FelipePn10/panossoerp/internal/infrastructure/logger
github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/employee
github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/enterprise
github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/group
github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/item
github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/modifier
github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/structure
github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/structure_query
github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/warehouse
github.com/FelipePn10/panossoerp/internal/infrastructure/nesting
github.com/FelipePn10/panossoerp/internal/infrastructure/notification
github.com/FelipePn10/panossoerp/internal/infrastructure/observability
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/accounting
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/allocation_base
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/aps
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/bom_header
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/components
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/consumer_service
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/cost_center
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/crp
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/cutting_plan
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/delivery_promise
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/delivery_promise_params
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/delivery_reschedule
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/employee
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/enterprise
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/entry_operation
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal_classification
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/group
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/ibpt
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/independent_demand
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/industrial_calendar
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_calendar_promise
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_classification
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_conversion
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_supplier
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/location
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/machine
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/maintenance
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/modifier
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_calculation
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/nfse
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/order_priority
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/overhead_allocation
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planned_order
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planning_params
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/procurement
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/product
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_plan
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_order
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_price
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_quotation
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_requisition
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/quality
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/recurring_sales
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/representative
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/restriction
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_division
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_forecast
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_goal
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_quotation
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/shipment
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/standard_cost
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock_movement
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure_query
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/supplier
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/technical_assistance
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/tool
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/user
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/warehouse
github.com/FelipePn10/panossoerp/internal/interfaces/http/context
github.com/FelipePn10/panossoerp/internal/interfaces/http/handler
github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security
github.com/FelipePn10/panossoerp/internal/interfaces/middleware
github.com/FelipePn10/panossoerp/internal/pkg/datetime
github.com/FelipePn10/panossoerp/internal/pkg/validation
```

## Dependências internas entre pacotes

```text
github.com/FelipePn10/panossoerp/api -> context, encoding/json, errors, github.com/FelipePn10/panossoerp/internal/application/usecase/accounting_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/allocation_base_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/aps_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/bom_header_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/cnpj_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/configurator_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/consumer_service_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/cost_center_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/cost_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/crp_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/customer_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/cutting_plan_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_promise_params_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_promise_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_reschedule_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/drawing_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/employee, github.com/FelipePn10/panossoerp/internal/application/usecase/enterprise_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/entry_operation_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/financial_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_classification_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_params_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/group_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/ibpt_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/independent_demand_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/industrial_calendar, github.com/FelipePn10/panossoerp/internal/application/usecase/item_calendar_promise_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/item_classification_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/item_conversion_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/item_supplier_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/item_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/location_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/lot_mask_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/machine_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/maintenance_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/modifier_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_calculation_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/nfse_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/order_priority_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/overhead_allocation_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/planned_order_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/planning_params_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/planning_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/procurement_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/production_plan_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_price_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_quotation_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_requisition_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/quality_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/recurring_sales_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/representative_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/restriction_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/routing_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/sales_division_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/sales_forecast_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/sales_goal_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/sales_order_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/sales_quotation_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/shipment_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/stock_movement_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/stock_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/structure_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/supplier_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/technical_assistance_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/tool_sheet_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/tool_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/user_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/warehouse_uc, github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/service, github.com/FelipePn10/panossoerp/internal/infrastructure/audit, github.com/FelipePn10/panossoerp/internal/infrastructure/auth, github.com/FelipePn10/panossoerp/internal/infrastructure/cnpj, github.com/FelipePn10/panossoerp/internal/infrastructure/config, github.com/FelipePn10/panossoerp/internal/infrastructure/database, github.com/FelipePn10/panossoerp/internal/infrastructure/logger, github.com/FelipePn10/panossoerp/internal/infrastructure/nesting, github.com/FelipePn10/panossoerp/internal/infrastructure/notification, github.com/FelipePn10/panossoerp/internal/infrastructure/observability, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/accounting, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/allocation_base, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/aps, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/bom_header, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/consumer_service, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/cost_center, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/crp, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/cutting_plan, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/delivery_promise, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/delivery_promise_params, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/delivery_reschedule, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/employee, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/enterprise, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/entry_operation, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal_classification, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/group, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/ibpt, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/independent_demand, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/industrial_calendar, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_calendar_promise, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_classification, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_conversion, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_supplier, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/location, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/machine, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/maintenance, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/modifier, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_calculation, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/nfse, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/order_priority, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/overhead_allocation, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planned_order, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planning_params, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/procurement, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_plan, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_order, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_price, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_quotation, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_requisition, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/quality, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/recurring_sales, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/representative, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/restriction, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_division, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_forecast, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_goal, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_quotation, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/shipment, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/standard_cost, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock_movement, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure_query, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/supplier, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/technical_assistance, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/tool, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/user, github.com/FelipePn10/panossoerp/internal/infrastructure/repository/warehouse, github.com/FelipePn10/panossoerp/internal/interfaces/http/handler, github.com/FelipePn10/panossoerp/internal/interfaces/middleware, github.com/go-chi/chi/v5, github.com/go-chi/chi/v5/middleware, go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp, net/http, os, os/signal, strings, syscall, time
github.com/FelipePn10/panossoerp/cmd/cutting-samples -> encoding/json, flag, fmt, github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity, github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/service, os, path/filepath, time
github.com/FelipePn10/panossoerp/internal/application/dto/request -> encoding/json, github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/FelipePn10/panossoerp/internal/domain/items/entity, github.com/FelipePn10/panossoerp/internal/domain/items/valueobject, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/application/dto/response -> encoding/json, github.com/FelipePn10/panossoerp/internal/domain/consumer_service/repository, github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/repository, github.com/google/uuid, github.com/shopspring/decimal, time
github.com/FelipePn10/panossoerp/internal/application/ports -> context, github.com/google/uuid
github.com/FelipePn10/panossoerp/internal/application/security -> 
github.com/FelipePn10/panossoerp/internal/application/usecase/accounting_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/accounting/ecd, github.com/FelipePn10/panossoerp/internal/domain/accounting/entity, github.com/FelipePn10/panossoerp/internal/domain/accounting/repository, sort, time
github.com/FelipePn10/panossoerp/internal/application/usecase/allocation_base_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/allocation_base/entity, github.com/FelipePn10/panossoerp/internal/domain/allocation_base/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/aps_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/aps/entity, github.com/FelipePn10/panossoerp/internal/domain/aps/repository, github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/repository, sort, strconv, strings, time
github.com/FelipePn10/panossoerp/internal/application/usecase/bom_header_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/bom_header/entity, github.com/FelipePn10/panossoerp/internal/domain/bom_header/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/cnpj_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/cnpj/entity, github.com/FelipePn10/panossoerp/internal/domain/cnpj/service, github.com/FelipePn10/panossoerp/internal/pkg/validation
github.com/FelipePn10/panossoerp/internal/application/usecase/component_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/component/entity, github.com/FelipePn10/panossoerp/internal/domain/component/repository, github.com/FelipePn10/panossoerp/internal/domain/component/valueobject
github.com/FelipePn10/panossoerp/internal/application/usecase/configurator_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/configurator/entity, github.com/FelipePn10/panossoerp/internal/domain/structure/formula, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5/pgtype, math, strconv, strings
github.com/FelipePn10/panossoerp/internal/application/usecase/consumer_service_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/consumer_service/entity, github.com/FelipePn10/panossoerp/internal/domain/consumer_service/repository, github.com/FelipePn10/panossoerp/internal/pkg/datetime, strings, time
github.com/FelipePn10/panossoerp/internal/application/usecase/cost_center_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/cost_center/entity, github.com/FelipePn10/panossoerp/internal/domain/cost_center/repository, time
github.com/FelipePn10/panossoerp/internal/application/usecase/cost_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/routing/entity, github.com/FelipePn10/panossoerp/internal/domain/standard_cost/entity, github.com/FelipePn10/panossoerp/internal/domain/standard_cost/repository, github.com/google/uuid, sort
github.com/FelipePn10/panossoerp/internal/application/usecase/crp_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/crp/entity, github.com/FelipePn10/panossoerp/internal/domain/crp/repository, github.com/FelipePn10/panossoerp/internal/domain/maintenance/repository, github.com/FelipePn10/panossoerp/internal/domain/routing/entity, time
github.com/FelipePn10/panossoerp/internal/application/usecase/customer_uc -> context, encoding/json, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/customer/entity, github.com/FelipePn10/panossoerp/internal/domain/customer/repository, strconv, time
github.com/FelipePn10/panossoerp/internal/application/usecase/cutting_plan_uc -> context, encoding/json, errors, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity, github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/repository, github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/service, github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity, github.com/FelipePn10/panossoerp/internal/domain/items/entity, github.com/FelipePn10/panossoerp/internal/domain/items/repository, github.com/FelipePn10/panossoerp/internal/domain/items/valueobject, github.com/FelipePn10/panossoerp/internal/domain/machine/entity, github.com/FelipePn10/panossoerp/internal/domain/machine/repository, github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository, github.com/FelipePn10/panossoerp/internal/domain/production_order/repository, github.com/FelipePn10/panossoerp/internal/domain/stock/entity, github.com/FelipePn10/panossoerp/internal/domain/stock/repository, github.com/FelipePn10/panossoerp/internal/domain/structure/entity, github.com/FelipePn10/panossoerp/internal/domain/structure_query/repository, math, time
github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_promise_params_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/domain/delivery_promise_params/entity, github.com/FelipePn10/panossoerp/internal/domain/delivery_promise_params/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_promise_uc -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/delivery_promise/entity, github.com/FelipePn10/panossoerp/internal/domain/delivery_promise/repository, github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/entity, github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/repository, github.com/FelipePn10/panossoerp/internal/domain/item_calendar_promise/repository, github.com/FelipePn10/panossoerp/internal/domain/items/repository, github.com/FelipePn10/panossoerp/internal/domain/items/valueobject, github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity, github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository, github.com/FelipePn10/panossoerp/internal/domain/stock/repository, math, sort, time
github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_reschedule_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/entity, github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/drawing_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/drawing/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/FelipePn10/panossoerp/internal/pkg/datetime, github.com/jackc/pgx/v5/pgtype
github.com/FelipePn10/panossoerp/internal/application/usecase/employee -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/employee/entity, github.com/FelipePn10/panossoerp/internal/domain/employee/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/enterprise_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/enterprise/entity, github.com/FelipePn10/panossoerp/internal/domain/enterprise/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/entry_operation_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/entry_operation/entity, github.com/FelipePn10/panossoerp/internal/domain/entry_operation/repository, strings
github.com/FelipePn10/panossoerp/internal/application/usecase/errors -> errors
github.com/FelipePn10/panossoerp/internal/application/usecase/financial_uc -> context, crypto/sha256, encoding/json, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/financial/entity, github.com/FelipePn10/panossoerp/internal/domain/financial/repository, github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity, github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository, github.com/shopspring/decimal, math, regexp, strconv, strings, time
github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_classification_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/fiscal_classification/entity, github.com/FelipePn10/panossoerp/internal/domain/fiscal_classification/repository, time
github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_params_uc -> context, errors, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity, github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository, regexp
github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc -> context, encoding/json, encoding/xml, errors, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/financial/entity, github.com/FelipePn10/panossoerp/internal/domain/financial/repository, github.com/FelipePn10/panossoerp/internal/domain/fiscal/engine, github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity, github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository, github.com/FelipePn10/panossoerp/internal/domain/fiscal/sped, github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository, github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity, github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository, github.com/FelipePn10/panossoerp/internal/domain/shipment/entity, github.com/FelipePn10/panossoerp/internal/domain/shipment/repository, github.com/FelipePn10/panossoerp/internal/domain/stock/entity, github.com/FelipePn10/panossoerp/internal/domain/stock/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe, github.com/google/uuid, github.com/shopspring/decimal, strconv, strings, time
github.com/FelipePn10/panossoerp/internal/application/usecase/forecast_uc -> fmt, math
github.com/FelipePn10/panossoerp/internal/application/usecase/group_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/group/entity, github.com/FelipePn10/panossoerp/internal/domain/group/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/ibpt_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/ibpt/entity, github.com/FelipePn10/panossoerp/internal/domain/ibpt/repository, strconv, strings, time
github.com/FelipePn10/panossoerp/internal/application/usecase/independent_demand_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity, github.com/FelipePn10/panossoerp/internal/domain/independent_demand/repository, time
github.com/FelipePn10/panossoerp/internal/application/usecase/industrial_calendar -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/entity, github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/repository, time
github.com/FelipePn10/panossoerp/internal/application/usecase/item_calendar_promise_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/item_calendar_promise/entity, github.com/FelipePn10/panossoerp/internal/domain/item_calendar_promise/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/item_classification_uc -> context, errors, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/items/entity, github.com/FelipePn10/panossoerp/internal/domain/items/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/item_conversion_uc -> context, errors, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity, github.com/FelipePn10/panossoerp/internal/domain/item_conversion/repository, strings
github.com/FelipePn10/panossoerp/internal/application/usecase/item_supplier_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/item_supplier/entity, github.com/FelipePn10/panossoerp/internal/domain/item_supplier/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/item_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/FelipePn10/panossoerp/internal/domain/item_conversion/repository, github.com/FelipePn10/panossoerp/internal/domain/items/entity, github.com/FelipePn10/panossoerp/internal/domain/items/repository, github.com/FelipePn10/panossoerp/internal/domain/items/valueobject, github.com/FelipePn10/panossoerp/internal/domain/routing/repository, github.com/FelipePn10/panossoerp/internal/domain/structure/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/location_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/location/entity, github.com/FelipePn10/panossoerp/internal/domain/location/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/lot_mask_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/lot_mask/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5/pgtype, time
github.com/FelipePn10/panossoerp/internal/application/usecase/machine_uc -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/FelipePn10/panossoerp/internal/domain/items/repository, github.com/FelipePn10/panossoerp/internal/domain/items/valueobject, github.com/FelipePn10/panossoerp/internal/domain/machine/entity, github.com/FelipePn10/panossoerp/internal/domain/machine/repository, github.com/FelipePn10/panossoerp/internal/domain/machine/service, github.com/FelipePn10/panossoerp/internal/pkg/datetime, time
github.com/FelipePn10/panossoerp/internal/application/usecase/maintenance_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/maintenance/entity, github.com/FelipePn10/panossoerp/internal/domain/maintenance/repository, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/application/usecase/modifier_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/modifier/entity, github.com/FelipePn10/panossoerp/internal/domain/modifier/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_calculation_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity, github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository, github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/service
github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity, github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository, github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity, github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/notification, strings, time
github.com/FelipePn10/panossoerp/internal/application/usecase/nfse_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity, github.com/FelipePn10/panossoerp/internal/domain/nfse/entity, github.com/FelipePn10/panossoerp/internal/domain/nfse/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe, github.com/shopspring/decimal, time
github.com/FelipePn10/panossoerp/internal/application/usecase/order_priority_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/order_priority/entity, github.com/FelipePn10/panossoerp/internal/domain/order_priority/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/overhead_allocation_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/overhead_allocation/entity, github.com/FelipePn10/panossoerp/internal/domain/overhead_allocation/repository, time
github.com/FelipePn10/panossoerp/internal/application/usecase/planned_order_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity, github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository, github.com/FelipePn10/panossoerp/internal/domain/production_order/entity, github.com/FelipePn10/panossoerp/internal/domain/production_order/repository, github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/entity, github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/repository, github.com/FelipePn10/panossoerp/internal/domain/routing/entity, time
github.com/FelipePn10/panossoerp/internal/application/usecase/planning_params_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/planning_params/entity, github.com/FelipePn10/panossoerp/internal/domain/planning_params/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/planning_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/usecase/aps_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/crp_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_calculation_uc
github.com/FelipePn10/panossoerp/internal/application/usecase/procurement_uc -> context, encoding/json, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/procurement/entity, github.com/FelipePn10/panossoerp/internal/domain/procurement/repository, github.com/FelipePn10/panossoerp/internal/domain/stock/entity, github.com/FelipePn10/panossoerp/internal/domain/stock/repository, github.com/google/uuid, github.com/jackc/pgx/v5, strconv, time
github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/production_order/entity, github.com/FelipePn10/panossoerp/internal/domain/production_order/repository, github.com/FelipePn10/panossoerp/internal/domain/standard_cost/repository, github.com/FelipePn10/panossoerp/internal/domain/stock/entity, github.com/FelipePn10/panossoerp/internal/domain/stock/repository, github.com/FelipePn10/panossoerp/internal/domain/structure/entity, github.com/FelipePn10/panossoerp/internal/domain/structure/repository, github.com/FelipePn10/panossoerp/internal/domain/tool/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/FelipePn10/panossoerp/internal/pkg/datetime, time
github.com/FelipePn10/panossoerp/internal/application/usecase/production_plan_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity, github.com/FelipePn10/panossoerp/internal/domain/production_plan/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/product_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/product/entity, github.com/FelipePn10/panossoerp/internal/domain/product/repository, github.com/FelipePn10/panossoerp/internal/domain/product/valueobject
github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity, github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository, github.com/FelipePn10/panossoerp/internal/domain/procurement/entity, github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity, github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository, github.com/FelipePn10/panossoerp/internal/domain/stock/entity, github.com/FelipePn10/panossoerp/internal/domain/stock/repository, time
github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_price_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/purchase_price/entity, github.com/FelipePn10/panossoerp/internal/domain/purchase_price/repository, time
github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_quotation_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository, github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity, github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository, github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/entity, github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/repository, github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/repository, time
github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_requisition_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity, github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository, github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/entity, github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/repository, time
github.com/FelipePn10/panossoerp/internal/application/usecase/quality_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/quality/entity, github.com/FelipePn10/panossoerp/internal/domain/quality/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/recurring_sales_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/entity, github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/repository, github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity, github.com/FelipePn10/panossoerp/internal/pkg/datetime, github.com/google/uuid, math, strings, time
github.com/FelipePn10/panossoerp/internal/application/usecase/representative_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/representative/entity, github.com/FelipePn10/panossoerp/internal/domain/representative/repository, github.com/FelipePn10/panossoerp/internal/pkg/datetime, strings, time
github.com/FelipePn10/panossoerp/internal/application/usecase/restriction_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/restriction/entity, github.com/FelipePn10/panossoerp/internal/domain/restriction/repository, strings
github.com/FelipePn10/panossoerp/internal/application/usecase/routing_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/routing/entity, github.com/FelipePn10/panossoerp/internal/domain/routing/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/sales_division_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/sales_division/entity, github.com/FelipePn10/panossoerp/internal/domain/sales_division/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/sales_forecast_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/application/usecase/forecast_uc, github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/repository, github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/entity, github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/repository, github.com/google/uuid, math, sort, strings, time
github.com/FelipePn10/panossoerp/internal/application/usecase/sales_goal_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/sales_goal/entity, github.com/FelipePn10/panossoerp/internal/domain/sales_goal/repository, strings, time
github.com/FelipePn10/panossoerp/internal/application/usecase/sales_order_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/customer/repository, github.com/FelipePn10/panossoerp/internal/domain/financial/entity, github.com/FelipePn10/panossoerp/internal/domain/financial/repository, github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity, github.com/FelipePn10/panossoerp/internal/domain/independent_demand/repository, github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity, github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository, github.com/FelipePn10/panossoerp/internal/domain/stock/entity, github.com/FelipePn10/panossoerp/internal/domain/stock/repository, github.com/FelipePn10/panossoerp/internal/pkg/datetime, time
github.com/FelipePn10/panossoerp/internal/application/usecase/sales_quotation_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity, github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository, github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity, github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/repository, github.com/FelipePn10/panossoerp/internal/pkg/datetime, time
github.com/FelipePn10/panossoerp/internal/application/usecase/shipment_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/shipment/entity, github.com/FelipePn10/panossoerp/internal/domain/shipment/repository, github.com/FelipePn10/panossoerp/internal/domain/stock/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/export/romaneio, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/application/usecase/stock_movement_uc -> context, errors, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/stock_movement/entity, github.com/FelipePn10/panossoerp/internal/domain/stock_movement/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/stock_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/stock/entity, github.com/FelipePn10/panossoerp/internal/domain/stock/repository, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/application/usecase/structure_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/structure/entity, github.com/FelipePn10/panossoerp/internal/domain/structure/formula, github.com/FelipePn10/panossoerp/internal/domain/structure/repository, github.com/FelipePn10/panossoerp/internal/domain/structure/valueobject, github.com/FelipePn10/panossoerp/internal/domain/structure_query/repository, github.com/FelipePn10/panossoerp/internal/domain/structure_query/service, github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/structure, github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/structure_query
github.com/FelipePn10/panossoerp/internal/application/usecase/supplier_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository, github.com/FelipePn10/panossoerp/internal/domain/supplier/entity, github.com/FelipePn10/panossoerp/internal/domain/supplier/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe, github.com/FelipePn10/panossoerp/internal/pkg/validation, strings, time
github.com/FelipePn10/panossoerp/internal/application/usecase/technical_assistance_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/production_order/entity, github.com/FelipePn10/panossoerp/internal/domain/production_order/repository, github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity, github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository, github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/entity, github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/repository, github.com/FelipePn10/panossoerp/internal/pkg/datetime, time
github.com/FelipePn10/panossoerp/internal/application/usecase/tool_sheet_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/tool/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5/pgtype
github.com/FelipePn10/panossoerp/internal/application/usecase/tool_uc -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/tool/entity, github.com/FelipePn10/panossoerp/internal/domain/tool/repository
github.com/FelipePn10/panossoerp/internal/application/usecase/user_uc -> context, errors, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/domain/user/entity, github.com/FelipePn10/panossoerp/internal/domain/user/repository, github.com/google/uuid, golang.org/x/crypto/bcrypt
github.com/FelipePn10/panossoerp/internal/application/usecase/warehouse_uc -> context, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/ports, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/warehouse/entity, github.com/FelipePn10/panossoerp/internal/domain/warehouse/repository
github.com/FelipePn10/panossoerp/internal/domain/accounting/ecd -> fmt, strings, time
github.com/FelipePn10/panossoerp/internal/domain/accounting/entity -> time
github.com/FelipePn10/panossoerp/internal/domain/accounting/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/accounting/entity, time
github.com/FelipePn10/panossoerp/internal/domain/allocation_base/entity -> github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/allocation_base/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/allocation_base/entity
github.com/FelipePn10/panossoerp/internal/domain/aps/entity -> time
github.com/FelipePn10/panossoerp/internal/domain/aps/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/aps/entity, time
github.com/FelipePn10/panossoerp/internal/domain/bom_header/entity -> errors, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/bom_header/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/bom_header/entity
github.com/FelipePn10/panossoerp/internal/domain/cnpj/entity -> 
github.com/FelipePn10/panossoerp/internal/domain/cnpj/service -> context, errors, github.com/FelipePn10/panossoerp/internal/domain/cnpj/entity
github.com/FelipePn10/panossoerp/internal/domain/component/entity -> errors, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/component/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/component/entity
github.com/FelipePn10/panossoerp/internal/domain/component/valueobject -> errors, fmt, math/rand
github.com/FelipePn10/panossoerp/internal/domain/configurator/entity -> crypto/sha256, encoding/hex, errors, github.com/google/uuid, sort, strings, time
github.com/FelipePn10/panossoerp/internal/domain/consumer_service/entity -> github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/consumer_service/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/consumer_service/entity, time
github.com/FelipePn10/panossoerp/internal/domain/cost_center/entity -> github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/cost_center/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/cost_center/entity
github.com/FelipePn10/panossoerp/internal/domain/crp/entity -> time
github.com/FelipePn10/panossoerp/internal/domain/crp/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/crp/entity, time
github.com/FelipePn10/panossoerp/internal/domain/customer/entity -> fmt, github.com/google/uuid, math, time
github.com/FelipePn10/panossoerp/internal/domain/customer/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/customer/entity
github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity -> errors, github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity
github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/service -> encoding/json, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity, github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/service/lp, github.com/FelipePn10/panossoerp/internal/domain/enums/types, math, math/rand, sort, strings, time
github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/service/lp -> math
github.com/FelipePn10/panossoerp/internal/domain/delivery_promise/entity -> github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/delivery_promise_params/entity -> github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/delivery_promise_params/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/delivery_promise_params/entity
github.com/FelipePn10/panossoerp/internal/domain/delivery_promise/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/delivery_promise/entity, time
github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/entity -> github.com/FelipePn10/panossoerp/internal/domain/items/valueobject, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/entity, github.com/FelipePn10/panossoerp/internal/domain/items/valueobject
github.com/FelipePn10/panossoerp/internal/domain/drawing/entity -> errors, github.com/google/uuid, strings, time
github.com/FelipePn10/panossoerp/internal/domain/employee/entity -> errors, github.com/google/uuid, strings, time
github.com/FelipePn10/panossoerp/internal/domain/employee/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/employee/entity
github.com/FelipePn10/panossoerp/internal/domain/enterprise/entity -> errors, github.com/google/uuid
github.com/FelipePn10/panossoerp/internal/domain/enterprise/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/enterprise/entity
github.com/FelipePn10/panossoerp/internal/domain/entry_operation/entity -> fmt, github.com/google/uuid, strings, time
github.com/FelipePn10/panossoerp/internal/domain/entry_operation/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/entry_operation/entity
github.com/FelipePn10/panossoerp/internal/domain/enums/types -> database/sql/driver, encoding/json, fmt
github.com/FelipePn10/panossoerp/internal/domain/financial/entity -> github.com/google/uuid, github.com/shopspring/decimal, time
github.com/FelipePn10/panossoerp/internal/domain/financial/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/financial/entity, github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity, github.com/google/uuid, github.com/shopspring/decimal, time
github.com/FelipePn10/panossoerp/internal/domain/fiscal_classification/entity -> fmt, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/fiscal_classification/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/fiscal_classification/entity
github.com/FelipePn10/panossoerp/internal/domain/fiscal/engine -> fmt, github.com/shopspring/decimal
github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity -> github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity, github.com/google/uuid
github.com/FelipePn10/panossoerp/internal/domain/fiscal/sped -> fmt, strings, time
github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/entity -> crypto/sha256, encoding/hex, errors, github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/mask/service -> github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject, strings
github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/entity
github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject -> crypto/sha256, encoding/hex, errors, github.com/google/uuid, sort, strings
github.com/FelipePn10/panossoerp/internal/domain/group/entity -> errors, github.com/google/uuid
github.com/FelipePn10/panossoerp/internal/domain/group/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/group/entity
github.com/FelipePn10/panossoerp/internal/domain/ibpt/entity -> time
github.com/FelipePn10/panossoerp/internal/domain/ibpt/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/ibpt/entity
github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity -> github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/independent_demand/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity, time
github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/entity -> time
github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/entity, time
github.com/FelipePn10/panossoerp/internal/domain/item_calendar_promise/entity -> time
github.com/FelipePn10/panossoerp/internal/domain/item_calendar_promise/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/item_calendar_promise/entity
github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity -> fmt, github.com/google/uuid, strings, time
github.com/FelipePn10/panossoerp/internal/domain/item_conversion/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity
github.com/FelipePn10/panossoerp/internal/domain/items/entity -> errors, github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/FelipePn10/panossoerp/internal/domain/items/valueobject, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/items/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/items/entity, github.com/FelipePn10/panossoerp/internal/domain/items/valueobject
github.com/FelipePn10/panossoerp/internal/domain/item_supplier/entity -> fmt, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/item_supplier/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/item_supplier/entity
github.com/FelipePn10/panossoerp/internal/domain/items/valueobject -> errors
github.com/FelipePn10/panossoerp/internal/domain/location/entity -> time
github.com/FelipePn10/panossoerp/internal/domain/location/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/location/entity
github.com/FelipePn10/panossoerp/internal/domain/lot_mask/entity -> errors, fmt, github.com/google/uuid, strconv, strings, time
github.com/FelipePn10/panossoerp/internal/domain/machine/entity -> github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/machine/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/machine/entity, time
github.com/FelipePn10/panossoerp/internal/domain/machine/service -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/FelipePn10/panossoerp/internal/domain/machine/entity, math, time
github.com/FelipePn10/panossoerp/internal/domain/maintenance/entity -> errors, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/maintenance/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/maintenance/entity, time
github.com/FelipePn10/panossoerp/internal/domain/modifier/entity -> errors, github.com/google/uuid
github.com/FelipePn10/panossoerp/internal/domain/modifier/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/modifier/entity
github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity -> github.com/google/uuid, strconv, strings, time
github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/ports -> context, time
github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity, github.com/FelipePn10/panossoerp/internal/domain/order_priority/entity, github.com/FelipePn10/panossoerp/internal/domain/sales_division/entity
github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/service -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity, github.com/FelipePn10/panossoerp/internal/domain/independent_demand/repository, github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/repository, github.com/FelipePn10/panossoerp/internal/domain/items/entity, github.com/FelipePn10/panossoerp/internal/domain/items/repository, github.com/FelipePn10/panossoerp/internal/domain/items/valueobject, github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity, github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/ports, github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository, github.com/FelipePn10/panossoerp/internal/domain/order_priority/entity, github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity, github.com/FelipePn10/panossoerp/internal/domain/production_plan/repository, github.com/FelipePn10/panossoerp/internal/domain/restriction/repository, github.com/FelipePn10/panossoerp/internal/domain/routing/entity, github.com/FelipePn10/panossoerp/internal/domain/routing/repository, github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/repository, github.com/FelipePn10/panossoerp/internal/domain/structure/entity, github.com/FelipePn10/panossoerp/internal/domain/structure/repository, math, sort, strconv, strings, time
github.com/FelipePn10/panossoerp/internal/domain/nfse/entity -> github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/nfse/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/nfse/entity
github.com/FelipePn10/panossoerp/internal/domain/order_priority/entity -> github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/order_priority/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/order_priority/entity
github.com/FelipePn10/panossoerp/internal/domain/overhead_allocation/entity -> github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/overhead_allocation/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/overhead_allocation/entity
github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity -> github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity, time
github.com/FelipePn10/panossoerp/internal/domain/planning_params/entity -> github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/planning_params/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/planning_params/entity, github.com/google/uuid
github.com/FelipePn10/panossoerp/internal/domain/procurement/entity -> encoding/json, github.com/google/uuid, strings, time
github.com/FelipePn10/panossoerp/internal/domain/procurement/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/procurement/entity, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/product/entity -> errors, github.com/FelipePn10/panossoerp/internal/domain/product/valueobject, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/production_order/entity -> github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/production_order/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/production_order/entity, time
github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity -> errors, github.com/google/uuid, strings, time
github.com/FelipePn10/panossoerp/internal/domain/production_plan/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity
github.com/FelipePn10/panossoerp/internal/domain/product/repository -> context, errors, github.com/FelipePn10/panossoerp/internal/domain/product/entity
github.com/FelipePn10/panossoerp/internal/domain/product/valueobject -> errors, fmt, github.com/shopspring/decimal, math/rand
github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity -> github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity
github.com/FelipePn10/panossoerp/internal/domain/purchase_price/entity -> fmt, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/purchase_price/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/purchase_price/entity
github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/entity -> fmt, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/entity
github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/entity -> fmt, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/entity
github.com/FelipePn10/panossoerp/internal/domain/quality/entity -> errors, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/quality/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/quality/entity
github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/entity -> github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/entity, time
github.com/FelipePn10/panossoerp/internal/domain/representative/entity -> time
github.com/FelipePn10/panossoerp/internal/domain/representative/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/representative/entity, time
github.com/FelipePn10/panossoerp/internal/domain/restriction/entity -> errors, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/restriction/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/restriction/entity
github.com/FelipePn10/panossoerp/internal/domain/routing/entity -> errors, github.com/google/uuid, math, sort, time
github.com/FelipePn10/panossoerp/internal/domain/routing/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/routing/entity, github.com/google/uuid
github.com/FelipePn10/panossoerp/internal/domain/sales_division/entity -> errors, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/sales_division/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/sales_division/entity
github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/entity -> errors, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/entity, time
github.com/FelipePn10/panossoerp/internal/domain/sales_goal/entity -> time
github.com/FelipePn10/panossoerp/internal/domain/sales_goal/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/sales_goal/entity, time
github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity -> fmt, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity -> github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity, time
github.com/FelipePn10/panossoerp/internal/domain/shipment/entity -> fmt, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/shipment/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/shipment/entity, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/standard_cost/entity -> github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/standard_cost/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/standard_cost/entity
github.com/FelipePn10/panossoerp/internal/domain/stock/entity -> github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/stock_movement/entity -> time
github.com/FelipePn10/panossoerp/internal/domain/stock_movement/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/stock_movement/entity
github.com/FelipePn10/panossoerp/internal/domain/stock/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/stock/entity, time
github.com/FelipePn10/panossoerp/internal/domain/structure/entity -> errors, github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/google/uuid, sort, time
github.com/FelipePn10/panossoerp/internal/domain/structure/formula -> fmt, strconv, unicode
github.com/FelipePn10/panossoerp/internal/domain/structure_query/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/mask/service, github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject, github.com/FelipePn10/panossoerp/internal/domain/structure/entity, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/structure_query/service -> context, github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/mask/service, github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject, github.com/FelipePn10/panossoerp/internal/domain/structure/entity, github.com/FelipePn10/panossoerp/internal/domain/structure_query/repository, github.com/google/uuid
github.com/FelipePn10/panossoerp/internal/domain/structure/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/mask/service, github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject, github.com/FelipePn10/panossoerp/internal/domain/structure/entity
github.com/FelipePn10/panossoerp/internal/domain/structure/valueobject -> github.com/FelipePn10/panossoerp/internal/domain/structure/entity
github.com/FelipePn10/panossoerp/internal/domain/supplier/entity -> fmt, github.com/FelipePn10/panossoerp/internal/pkg/validation, github.com/google/uuid, regexp, time
github.com/FelipePn10/panossoerp/internal/domain/supplier/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/supplier/entity
github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/entity -> github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/entity, time
github.com/FelipePn10/panossoerp/internal/domain/tool/entity -> errors, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/tool/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/tool/entity
github.com/FelipePn10/panossoerp/internal/domain/user/entity -> errors, github.com/google/uuid
github.com/FelipePn10/panossoerp/internal/domain/user/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/user/entity
github.com/FelipePn10/panossoerp/internal/domain/warehouse/entity -> errors, github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/domain/warehouse/repository -> context, github.com/FelipePn10/panossoerp/internal/domain/warehouse/entity
github.com/FelipePn10/panossoerp/internal/infrastructure/audit -> context, fmt, github.com/FelipePn10/panossoerp/internal/infrastructure/logger, github.com/jackc/pgx/v5/pgxpool, strings, sync, time
github.com/FelipePn10/panossoerp/internal/infrastructure/auth -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/application/security, github.com/FelipePn10/panossoerp/internal/interfaces/http/context, github.com/golang-jwt/jwt/v5, github.com/google/uuid, strings, time
github.com/FelipePn10/panossoerp/internal/infrastructure/cnab -> fmt, strings, time
github.com/FelipePn10/panossoerp/internal/infrastructure/cnpj -> context, encoding/json, fmt, github.com/FelipePn10/panossoerp/internal/domain/cnpj/entity, github.com/FelipePn10/panossoerp/internal/domain/cnpj/service, io, net/http, regexp, strconv, strings, time
github.com/FelipePn10/panossoerp/internal/infrastructure/config -> errors, fmt, github.com/spf13/viper, os
github.com/FelipePn10/panossoerp/internal/infrastructure/database -> context, fmt, github.com/FelipePn10/panossoerp/internal/infrastructure/config, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/exaring/otelpgx, github.com/jackc/pgx/v5/pgxpool, time
github.com/FelipePn10/panossoerp/internal/infrastructure/database/nullable -> database/sql, encoding/json, github.com/sqlc-dev/pqtype
github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil -> encoding/json, fmt, github.com/google/uuid, github.com/jackc/pgx/v5/pgtype, strconv, time
github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc -> context, database/sql/driver, fmt, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqltypes, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgconn, github.com/jackc/pgx/v5/pgtype, time
github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqltypes -> database/sql/driver, fmt
github.com/FelipePn10/panossoerp/internal/infrastructure/export -> archive/zip, encoding/csv, fmt, github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit, io, net/http, reflect, regexp, sort, strconv, strings, time
github.com/FelipePn10/panossoerp/internal/infrastructure/export/gantt -> fmt, github.com/FelipePn10/panossoerp/internal/domain/aps/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit, strconv, strings, time
github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit -> bytes, compress/zlib, errors, fmt, image/jpeg, image/png, strconv, strings, time
github.com/FelipePn10/panossoerp/internal/infrastructure/export/romaneio -> archive/zip, bytes, fmt, github.com/FelipePn10/panossoerp/internal/infrastructure/export/pdfkit, strings, time
github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe -> bytes, context, encoding/json, fmt, io, net/http, strings, time
github.com/FelipePn10/panossoerp/internal/infrastructure/logger -> context, log/slog, net/http, os, strings
github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/employee -> github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/domain/employee/entity
github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/enterprise -> github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/domain/enterprise/entity
github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/group -> github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/domain/group/entity
github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/item -> github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/domain/items/entity
github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/modifier -> github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/domain/modifier/entity
github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/structure -> github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/structure/valueobject
github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/structure_query -> github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/structure_query/service
github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/warehouse -> github.com/FelipePn10/panossoerp/internal/domain/enums/types
github.com/FelipePn10/panossoerp/internal/infrastructure/nesting -> bytes, context, encoding/json, fmt, github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity, github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/service, net/http, time
github.com/FelipePn10/panossoerp/internal/infrastructure/notification -> bytes, context, crypto/tls, encoding/json, fmt, net, net/http, net/smtp, strings, time
github.com/FelipePn10/panossoerp/internal/infrastructure/observability -> context, go.opentelemetry.io/otel, go.opentelemetry.io/otel/attribute, go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp, go.opentelemetry.io/otel/propagation, go.opentelemetry.io/otel/sdk/resource, go.opentelemetry.io/otel/sdk/trace, go.opentelemetry.io/otel/semconv/v1.26.0, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/accounting -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/accounting/entity, github.com/FelipePn10/panossoerp/internal/domain/accounting/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5/pgxpool, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/allocation_base -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/allocation_base/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/aps -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/aps/entity, github.com/FelipePn10/panossoerp/internal/domain/aps/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/bom_header -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/bom_header/entity, github.com/FelipePn10/panossoerp/internal/domain/bom_header/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/components -> context, github.com/FelipePn10/panossoerp/internal/domain/component/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/consumer_service -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/consumer_service/entity, github.com/FelipePn10/panossoerp/internal/domain/consumer_service/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgxpool, strings, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/cost_center -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/cost_center/entity, github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/crp -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/crp/entity, github.com/FelipePn10/panossoerp/internal/domain/crp/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/customer -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/customer/entity, github.com/FelipePn10/panossoerp/internal/domain/customer/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqltypes, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgtype, github.com/jackc/pgx/v5/pgxpool
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/cutting_plan -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity, github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/repository, github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5/pgtype, github.com/jackc/pgx/v5/pgxpool, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/delivery_promise -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/delivery_promise/entity, github.com/google/uuid, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgxpool, strings, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/delivery_promise_params -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/delivery_promise_params/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/delivery_reschedule -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/entity, github.com/FelipePn10/panossoerp/internal/domain/items/valueobject, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/employee -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/employee/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/enterprise -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/domain/enterprise/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5/pgconn
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/entry_operation -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/entry_operation/entity, github.com/FelipePn10/panossoerp/internal/domain/entry_operation/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5/pgxpool
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/financial -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/financial/entity, github.com/FelipePn10/panossoerp/internal/domain/financial/repository, github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity, github.com/google/uuid, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgxpool, github.com/shopspring/decimal, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal_classification -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/fiscal_classification/entity, github.com/FelipePn10/panossoerp/internal/domain/fiscal_classification/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5/pgxpool
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/fiscal -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity, github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqltypes, github.com/google/uuid, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgxpool, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/group -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/group/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/ibpt -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/ibpt/entity, github.com/FelipePn10/panossoerp/internal/domain/ibpt/repository, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgxpool
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/independent_demand -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/industrial_calendar -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/industrial_calendar/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_calendar_promise -> context, database/sql, fmt, github.com/FelipePn10/panossoerp/internal/domain/item_calendar_promise/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_classification -> context, github.com/FelipePn10/panossoerp/internal/domain/items/entity, github.com/FelipePn10/panossoerp/internal/domain/items/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item -> context, encoding/json, fmt, github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/FelipePn10/panossoerp/internal/domain/items/entity, github.com/FelipePn10/panossoerp/internal/domain/items/valueobject, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_conversion -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity, github.com/FelipePn10/panossoerp/internal/domain/item_conversion/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5/pgxpool
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item_supplier -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/item_supplier/entity, github.com/FelipePn10/panossoerp/internal/domain/item_supplier/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5/pgxpool
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/location -> context, github.com/FelipePn10/panossoerp/internal/domain/location/entity, github.com/FelipePn10/panossoerp/internal/domain/location/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/machine -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/FelipePn10/panossoerp/internal/domain/machine/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgtype, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/maintenance -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/maintenance/entity, github.com/FelipePn10/panossoerp/internal/domain/maintenance/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5/pgtype, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/modifier -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/modifier/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/mrp_calculation -> context, encoding/json, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity, github.com/FelipePn10/panossoerp/internal/domain/order_priority/entity, github.com/FelipePn10/panossoerp/internal/domain/sales_division/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/nfse -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/nfse/entity, github.com/FelipePn10/panossoerp/internal/domain/nfse/repository, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgxpool
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/order_priority -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/order_priority/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/overhead_allocation -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/overhead_allocation/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planned_order -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/ports, github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgtype, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/planning_params -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/planning_params/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/google/uuid, github.com/jackc/pgx/v5
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/procurement -> context, encoding/json, fmt, github.com/FelipePn10/panossoerp/internal/domain/procurement/entity, github.com/FelipePn10/panossoerp/internal/domain/procurement/repository, github.com/google/uuid, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgxpool, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/product -> context, errors, github.com/FelipePn10/panossoerp/internal/domain/product/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgtype
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_order -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/production_order/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/google/uuid, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgxpool, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/production_plan -> context, encoding/json, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_order -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity, github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgtype, github.com/jackc/pgx/v5/pgxpool, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_price -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/purchase_price/entity, github.com/FelipePn10/panossoerp/internal/domain/purchase_price/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5/pgxpool
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_quotation -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/entity, github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5/pgxpool
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/purchase_requisition -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/entity, github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5/pgxpool
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/quality -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/quality/entity, github.com/FelipePn10/panossoerp/internal/domain/quality/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/google/uuid, github.com/jackc/pgx/v5/pgtype
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/recurring_sales -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/entity, github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgxpool, strings
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/representative -> context, database/sql, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/representative/entity, github.com/FelipePn10/panossoerp/internal/domain/representative/repository, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgxpool, strings, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/restriction -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/restriction/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgtype
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/routing/entity, github.com/FelipePn10/panossoerp/internal/domain/routing/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqltypes, github.com/google/uuid, github.com/jackc/pgx/v5/pgtype
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_division -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/sales_division/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_forecast -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5, strings, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_goal -> context, database/sql, fmt, github.com/FelipePn10/panossoerp/internal/domain/sales_goal/entity, github.com/FelipePn10/panossoerp/internal/domain/sales_goal/repository, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgxpool, strings
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity, github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/google/uuid, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgtype, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_quotation -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity, github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgtype, github.com/jackc/pgx/v5/pgxpool, strings, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/shipment -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/usecase/shipment_uc, github.com/FelipePn10/panossoerp/internal/domain/customer/repository, github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository, github.com/FelipePn10/panossoerp/internal/domain/items/repository, github.com/FelipePn10/panossoerp/internal/domain/items/valueobject, github.com/FelipePn10/panossoerp/internal/domain/production_order/entity, github.com/FelipePn10/panossoerp/internal/domain/production_order/repository, github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity, github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository, github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity, github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository, github.com/FelipePn10/panossoerp/internal/domain/shipment/entity, github.com/FelipePn10/panossoerp/internal/domain/shipment/repository, github.com/FelipePn10/panossoerp/internal/domain/supplier/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/export/romaneio, github.com/google/uuid, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgxpool, strings
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/standard_cost -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/standard_cost/entity, github.com/FelipePn10/panossoerp/internal/domain/standard_cost/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/stock/entity, github.com/FelipePn10/panossoerp/internal/domain/stock/repository, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgxpool, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock_movement -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/stock_movement/entity, github.com/FelipePn10/panossoerp/internal/domain/stock_movement/repository, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgxpool
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure -> context, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/mask/service, github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject, github.com/FelipePn10/panossoerp/internal/domain/structure/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5/pgtype
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/structure_query -> context, crypto/sha256, encoding/hex, fmt, github.com/FelipePn10/panossoerp/internal/domain/enums/types, github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/mask/service, github.com/FelipePn10/panossoerp/internal/domain/generate_mask_for_item/valueobject, github.com/FelipePn10/panossoerp/internal/domain/structure/entity, github.com/FelipePn10/panossoerp/internal/domain/structure/formula, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/google/uuid, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/supplier -> context, errors, fmt, github.com/FelipePn10/panossoerp/internal/domain/supplier/entity, github.com/FelipePn10/panossoerp/internal/domain/supplier/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/jackc/pgx/v5/pgconn, github.com/jackc/pgx/v5/pgxpool, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/technical_assistance -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/entity, github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgxpool, strings, time
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/tool -> context, fmt, github.com/FelipePn10/panossoerp/internal/domain/tool/entity, github.com/FelipePn10/panossoerp/internal/domain/tool/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/user -> context, github.com/FelipePn10/panossoerp/internal/domain/user/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc
github.com/FelipePn10/panossoerp/internal/infrastructure/repository/warehouse -> context, github.com/FelipePn10/panossoerp/internal/domain/warehouse/entity, github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil, github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc, github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/warehouse, strconv
github.com/FelipePn10/panossoerp/internal/interfaces/http/context -> 
github.com/FelipePn10/panossoerp/internal/interfaces/http/handler -> bytes, context, encoding/json, errors, fmt, github.com/FelipePn10/panossoerp/internal/application/dto/request, github.com/FelipePn10/panossoerp/internal/application/dto/response, github.com/FelipePn10/panossoerp/internal/application/security, github.com/FelipePn10/panossoerp/internal/application/usecase/accounting_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/allocation_base_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/aps_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/bom_header_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/cnpj_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/configurator_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/consumer_service_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/cost_center_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/cost_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/crp_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/customer_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/cutting_plan_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_promise_params_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_promise_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/delivery_reschedule_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/drawing_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/employee, github.com/FelipePn10/panossoerp/internal/application/usecase/enterprise_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/entry_operation_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/application/usecase/financial_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_classification_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_params_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/fiscal_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/forecast_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/group_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/ibpt_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/independent_demand_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/industrial_calendar, github.com/FelipePn10/panossoerp/internal/application/usecase/item_calendar_promise_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/item_classification_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/item_conversion_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/item_supplier_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/item_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/location_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/lot_mask_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/machine_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/maintenance_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/modifier_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_calculation_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/nfse_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/order_priority_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/overhead_allocation_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/planned_order_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/planning_params_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/planning_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/procurement_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/product_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/production_order_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/production_plan_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_price_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_quotation_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_requisition_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/quality_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/recurring_sales_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/representative_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/restriction_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/routing_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/sales_division_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/sales_forecast_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/sales_goal_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/sales_order_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/sales_quotation_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/shipment_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/stock_movement_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/stock_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/structure_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/supplier_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/technical_assistance_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/tool_sheet_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/tool_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/user_uc, github.com/FelipePn10/panossoerp/internal/application/usecase/warehouse_uc, github.com/FelipePn10/panossoerp/internal/domain/accounting/ecd, github.com/FelipePn10/panossoerp/internal/domain/accounting/entity, github.com/FelipePn10/panossoerp/internal/domain/cnpj/service, github.com/FelipePn10/panossoerp/internal/domain/consumer_service/entity, github.com/FelipePn10/panossoerp/internal/domain/consumer_service/repository, github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity, github.com/FelipePn10/panossoerp/internal/domain/fiscal/sped, github.com/FelipePn10/panossoerp/internal/domain/items/valueobject, github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/entity, github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/repository, github.com/FelipePn10/panossoerp/internal/domain/representative/repository, github.com/FelipePn10/panossoerp/internal/domain/restriction/entity, github.com/FelipePn10/panossoerp/internal/domain/sales_goal/repository, github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity, github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository, github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity, github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/repository, github.com/FelipePn10/panossoerp/internal/domain/shipment/entity, github.com/FelipePn10/panossoerp/internal/domain/shipment/repository, github.com/FelipePn10/panossoerp/internal/domain/stock_movement/entity, github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/entity, github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/repository, github.com/FelipePn10/panossoerp/internal/infrastructure/audit, github.com/FelipePn10/panossoerp/internal/infrastructure/auth, github.com/FelipePn10/panossoerp/internal/infrastructure/cnab, github.com/FelipePn10/panossoerp/internal/infrastructure/export, github.com/FelipePn10/panossoerp/internal/infrastructure/export/gantt, github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/enterprise, github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/group, github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/item, github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/modifier, github.com/FelipePn10/panossoerp/internal/interfaces/http/context, github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security, github.com/FelipePn10/panossoerp/internal/pkg/datetime, github.com/go-chi/chi/v5, github.com/google/uuid, image/jpeg, image/png, net/http, strconv, strings, time
github.com/FelipePn10/panossoerp/internal/interfaces/http/handler/security -> encoding/json, errors, github.com/FelipePn10/panossoerp/internal/application/usecase/errors, github.com/FelipePn10/panossoerp/internal/infrastructure/logger, github.com/jackc/pgx/v5/pgconn, net/http
github.com/FelipePn10/panossoerp/internal/interfaces/middleware -> bytes, context, encoding/json, fmt, github.com/FelipePn10/panossoerp/internal/application/security, github.com/FelipePn10/panossoerp/internal/infrastructure/audit, github.com/FelipePn10/panossoerp/internal/infrastructure/auth, github.com/FelipePn10/panossoerp/internal/infrastructure/logger, github.com/FelipePn10/panossoerp/internal/interfaces/http/context, github.com/go-chi/chi/v5, github.com/go-chi/chi/v5/middleware, github.com/golang-jwt/jwt/v5, github.com/google/uuid, log/slog, net, net/http, sort, strconv, strings, sync, time
github.com/FelipePn10/panossoerp/internal/pkg/datetime -> strings, time
github.com/FelipePn10/panossoerp/internal/pkg/validation -> regexp, strconv, strings
```

## Arquivos Go por diretório

### `./api`
- `api.go`
- `main.go`

### `./cmd/cutting-samples`
- `main.go`

### `./internal/application/dto/request`
- `allocation_base.go`
- `aps_request.go`
- `associate_by_question_from_item_request_dto.go`
- `bom_header_request.go`
- `configurator_request.go`
- `consumer_service_request.go`
- `cost_center_dto_request.go`
- `create_calendar_day_dto.go`
- `create_component_request_dto.go`
- `create_delivery_reschedule_dto_request.go`
- `create_employee_dto_request.go`
- `create_enterprise_request.go`
- `create_group_request.go`
- `create_independent_demand_dto_request.go`
- `create_item.go`
- `create_modifier_request_dto.go`
- `create_product_request_dto.go`
- `create_purchase_order_dto.go`
- `create_question.go`
- `create_question_option_dto.go`
- `create_warehouse.go`
- `crp_request.go`
- `customer_dto.go`
- `cutting_plan_request.go`
- `delivery_promise_dto_request.go`
- `delivery_promise_params_dto_request.go`
- `drawing_request.go`
- `entry_operation_dto.go`
- `financial_dto.go`
- `find_item_by_code_dto_request.go`
- `fiscal_classification_dto.go`
- `fiscal_dto.go`
- `fiscal_params_dto.go`
- `generate_mask_item_request_dto.go`
- `get_all_direct_children_request_dto.go`
- `item_calendar_promise_dto_request.go`
- `item_conversion_dto.go`
- `item_struct_request.go`
- `item_supplier_dto.go`
- `login_user_request_dto.go`
- `lot_mask_request.go`
- `machine.go`
- `mrp_calculation.go`
- `nfse_dto.go`
- `order_priority.go`
- `overhead_allocation_dto_request.go`
- `planning_param_dto_request.go`
- `planning_request.go`
- `procurement_closeout_dto.go`
- `procurement_maturity_dto.go`
- `production_order_dto.go`
- `production_order_operations_request.go`
- `production_plan_dto_request.go`
- `purchase_price_dto.go`
- `purchase_quotation_dto.go`
- `purchase_receipt_dto.go`
- `purchase_requisition_dto.go`
- `quality_request.go`
- `receiving_inspection_dto.go`
- `register_user_request_dto.go`
- `representative_dto.go`
- `resolve_structure_query_request.go`
- `restriction_dto_request.go`
- `restriction_reason_dto_request.go`
- `routing_request.go`
- `sales_division_dto_request.go`
- `sales_forecast_dto_request.go`
- `sales_goal_dto.go`
- `sales_order_dto.go`
- `sales_quotation_dto.go`
- `standard_cost_request.go`
- `stock_dto.go`
- `supplier_dto.go`
- `technical_assistance_request.go`
- `tool_request.go`
- `tool_sheet_request.go`
- `update_purchase_order_dto.go`
- `update_schedule_dto.go`

### `./internal/application/dto/response`
- `allocation_base_response.go`
- `aps_response.go`
- `bom_header_response.go`
- `calendar_response.go`
- `cnpj_response.go`
- `component_response.go`
- `configurator_response.go`
- `consult_structure_response.go`
- `consumer_service_response.go`
- `cost_center_response.go`
- `crp_response.go`
- `customer_response.go`
- `cutting_plan_response.go`
- `delivery_promise_params_response.go`
- `delivery_promise_response.go`
- `delivery_reschedule_response.go`
- `drawing_response.go`
- `employee_response.go`
- `enterprise_response.go`
- `entry_operation_response.go`
- `financial_response.go`
- `fiscal_classification_response.go`
- `fiscal_params_extra_response.go`
- `fiscal_params_response.go`
- `fiscal_response.go`
- `generate_mask_response.go`
- `group_response.go`
- `ibpt_response.go`
- `independent_demand_response.go`
- `item_conversion_response.go`
- `item_response.go`
- `item_structure_response.go`
- `item_supplier_response.go`
- `lot_mask_response.go`
- `machine_response.go`
- `maintenance_response.go`
- `modifier_response.go`
- `mrp_calculation_response.go`
- `nfse_response.go`
- `order_priority_response.go`
- `overhead_allocation_response.go`
- `planned_order_response.go`
- `planning_params_response.go`
- `planning_response.go`
- `procurement_closeout_response.go`
- `procurement_maturity_response.go`
- `production_order_operations_response.go`
- `production_order_response.go`
- `production_plan_response.go`
- `product_response.go`
- `purchase_order_response.go`
- `purchase_price_response.go`
- `purchase_quotation_response.go`
- `purchase_requisition_response.go`
- `quality_response.go`
- `question_option_response.go`
- `question_response.go`
- `receiving_inspection_response.go`
- `representative_response.go`
- `resolve_structure_query_response.go`
- `restriction_response.go`
- `routing_response.go`
- `sales_division_response.go`
- `sales_forecast_response.go`
- `sales_goal_response.go`
- `sales_order_response.go`
- `sales_quotation_response.go`
- `shipment_response.go`
- `standard_cost_response.go`
- `stock_response.go`
- `supplier_response.go`
- `technical_assistance_response.go`
- `tool_response.go`
- `tool_sheet_response.go`
- `warehouse_response.go`

### `./internal/application/ports`
- `auth_service.go`
- `fiscal_classification.go`
- `preferred_supplier.go`
- `purchase_price.go`
- `supplier_defaults.go`
- `uom_converter.go`

### `./internal/application/security`
- `auth_user.go`

### `./internal/application/usecase/accounting_uc`
- `accounting_uc.go`
- `balancete_uc.go`

### `./internal/application/usecase/allocation_base_uc`
- `create_usecase.go`
- `list_usecase.go`
- `response_mapper.go`

### `./internal/application/usecase/aps_uc`
- `aps_scheduling_test.go`
- `aps_uc.go`
- `gantt_month_uc.go`
- `gantt_month_uc_test.go`
- `reschedule_uc.go`
- `reschedule_uc_test.go`

### `./internal/application/usecase/bom_header_uc`
- `bom_header_uc.go`

### `./internal/application/usecase/cnpj_uc`
- `lookup_cnpj_uc.go`

### `./internal/application/usecase/component_uc`
- `create_component.go`
- `response_mapper.go`

### `./internal/application/usecase/configurator_uc`
- `cartesian_uc.go`
- `characteristic_uc.go`
- `configurator_integration_test.go`
- `configurator_uc.go`
- `description_uc.go`
- `formula_uc.go`
- `item_characteristic_uc.go`
- `mappers.go`
- `mask_uc.go`
- `rules_uc.go`

### `./internal/application/usecase/consumer_service_uc`
- `consumer_service_uc.go`
- `consumer_service_uc_test.go`
- `mapper.go`

### `./internal/application/usecase/cost_center_uc`
- `cost_center_create.go`
- `cost_center_get.go`
- `cost_center_list.go`
- `response_mapper.go`

### `./internal/application/usecase/cost_uc`
- `coproduct_cost_integration_test.go`
- `cost_integration_test.go`
- `cost_rollup_helpers_test.go`
- `cost_rollup_uc.go`

### `./internal/application/usecase/crp_uc`
- `crp_uc.go`
- `crp_uc_test.go`

### `./internal/application/usecase/customer_uc`
- `customer_uc.go`

### `./internal/application/usecase/cutting_plan_uc`
- `cutting_plan_uc.go`
- `demand_uc.go`
- `demand_uc_test.go`
- `export_uc.go`
- `mapper.go`
- `release_uc.go`
- `release_uc_test.go`

### `./internal/application/usecase/delivery_promise_params_uc`
- `manage.go`
- `response_mapper.go`

### `./internal/application/usecase/delivery_promise_uc`
- `delivery_promise_uc.go`
- `delivery_promise_uc_test.go`

### `./internal/application/usecase/delivery_reschedule_uc`
- `create_usecase.go`
- `list.go`
- `response_mapper.go`

### `./internal/application/usecase/drawing_uc`
- `drawing_integration_test.go`
- `drawing_uc.go`

### `./internal/application/usecase/employee`
- `create_employee.go`
- `deactivate_employee.go`
- `get_employee.go`
- `list_employees.go`
- `response_mapper.go`
- `update_employee.go`

### `./internal/application/usecase/enterprise_uc`
- `create_enterprise.go`
- `get_enterprise.go`
- `response_mapper.go`

### `./internal/application/usecase/entry_operation_uc`
- `entry_operation_uc.go`
- `response_mapper.go`

### `./internal/application/usecase/errors`
- `erros.go`
- `typed.go`

### `./internal/application/usecase/financial_uc`
- `adiantamento_uc.go`
- `approve_conta_pagar_uc.go`
- `apurar_impostos_uc.go`
- `baixa_adiantamento_uc_test.go`
- `baixar_conta_pagar_uc.go`
- `baixar_conta_receber_uc.go`
- `cancel_conta_pagar_uc.go`
- `cancel_conta_receber_uc.go`
- `create_centro_custo_uc.go`
- `create_condicao_pagamento_uc.go`
- `create_conta_bancaria_uc.go`
- `create_conta_pagar_uc.go`
- `create_conta_receber_uc.go`
- `create_plano_contas_uc.go`
- `crud_uc_test.go`
- `get_aging_pagar_uc.go`
- `get_aging_receber_uc.go`
- `get_conta_pagar_uc.go`
- `get_conta_receber_uc.go`
- `get_fluxo_caixa_uc.go`
- `get_fluxo_projetado_uc.go`
- `get_saldo_contas_uc.go`
- `get_tax_assessment_uc.go`
- `importar_ofx_uc.go`
- `importar_ofx_uc_test.go`
- `list_centros_custo_uc.go`
- `list_condicoes_pagamento_uc.go`
- `list_contas_bancarias_uc.go`
- `list_contas_pagar_uc.go`
- `list_contas_receber_uc.go`
- `list_plano_contas_uc.go`
- `reports_uc.go`
- `response_mapper.go`

### `./internal/application/usecase/fiscal_classification_uc`
- `fiscal_classification_uc.go`
- `response_mapper.go`

### `./internal/application/usecase/fiscal_params_uc`
- `cfop_uc.go`
- `icms_apuracao_uc.go`
- `icms_apuracao_uc_test.go`
- `icms_reduction_uc.go`
- `legal_device_uc.go`
- `response_mapper.go`
- `tax_param_uc.go`

### `./internal/application/usecase/fiscal_uc`
- `approve_fiscal_entry_uc.go`
- `authorize_cte_uc.go`
- `authorize_fiscal_exit_uc.go`
- `authorize_fiscal_exit_uc_test.go`
- `cancel_fiscal_exit_uc.go`
- `consultar_nfe_uc.go`
- `create_cte_uc.go`
- `create_fiscal_entry_uc.go`
- `create_fiscal_exit_from_load_uc.go`
- `create_fiscal_exit_from_load_uc_test.go`
- `create_fiscal_exit_uc.go`
- `emitir_cce_uc.go`
- `get_cte_uc.go`
- `get_danfe_uc.go`
- `get_fiscal_config_uc.go`
- `get_fiscal_entry_uc.go`
- `get_fiscal_exit_uc.go`
- `import_nfe_purchase_uc.go`
- `list_cte_uc.go`
- `list_fiscal_entries_uc.go`
- `list_fiscal_exits_uc.go`
- `manage_ncm_uc.go`
- `manifestacao_uc.go`
- `response_mapper.go`
- `sped_uc.go`
- `update_branding_uc.go`
- `update_fiscal_config_uc.go`
- `upload_nfe_entry_uc.go`

### `./internal/application/usecase/forecast_uc`
- `statistical_forecast_uc.go`

### `./internal/application/usecase/group_uc`
- `create_group.go`
- `manage_group.go`
- `response_mapper.go`

### `./internal/application/usecase/ibpt_uc`
- `ibpt_uc.go`
- `response_mapper.go`

### `./internal/application/usecase/independent_demand_uc`
- `create_independent_demand.go`
- `delete_independent_demand.go`
- `get_by_code_independent_demand.go`
- `list_by_date_independent_demand.go`
- `list_by_item_independent_demand.go`
- `list_independent_demand.go`
- `response_mapper.go`
- `update_independent_demand.go`

### `./internal/application/usecase/industrial_calendar`
- `manage_industrial_calendar.go`
- `response_mapper.go`

### `./internal/application/usecase/item_calendar_promise_uc`
- `manage.go`
- `response_mapper.go`

### `./internal/application/usecase/item_classification_uc`
- `item_classification_uc.go`
- `response_mapper.go`

### `./internal/application/usecase/item_conversion_uc`
- `item_conversion_uc.go`
- `item_conversion_uc_test.go`
- `response_mapper.go`

### `./internal/application/usecase/item_supplier_uc`
- `item_supplier_uc.go`
- `response_mapper.go`

### `./internal/application/usecase/item_uc`
- `create_item.go`
- `find_item_by_code.go`
- `list_items.go`
- `list_items_with_masks.go`
- `response_mapper.go`
- `validate_item_activation.go`

### `./internal/application/usecase/location_uc`
- `location_uc.go`
- `response_mapper.go`

### `./internal/application/usecase/lot_mask_uc`
- `lot_mask_integration_test.go`
- `lot_mask_uc.go`

### `./internal/application/usecase/machine_uc`
- `calculate_production_time.go`
- `create_item_machine_time.go`
- `create_machine.go`
- `create_type.go`
- `delete_machine.go`
- `delete_machine_type.go`
- `get_item_machime_time.go`
- `get_machine.go`
- `get_machine_type.go`
- `list_by_machine.go`
- `list_by_type.go`
- `list_item_machine_time.go`
- `list_machine.go`
- `list_types.go`
- `response_mapper.go`
- `schedule.go`
- `update_machine.go`
- `update_machine_type.go`

### `./internal/application/usecase/maintenance_uc`
- `maintenance_uc.go`
- `response_mapper.go`

### `./internal/application/usecase/modifier_uc`
- `create_modifier.go`
- `manage_modifier.go`
- `response_mapper.go`

### `./internal/application/usecase/mrp_calculation_uc`
- `calculate.go`
- `configured_rules.go`
- `get_profile.go`
- `list_exceptions.go`
- `response_mapper.go`

### `./internal/application/usecase/mrp_uc`
- `firmar_sugestao_uc.go`
- `firmar_sugestao_uc_test.go`
- `notify_exceptions_uc.go`

### `./internal/application/usecase/nfse_uc`
- `nfse_uc.go`
- `nfse_uc_test.go`
- `response_mapper.go`

### `./internal/application/usecase/order_priority_uc`
- `create_order_priority.go`
- `find.go`
- `list.go`
- `response_mapper.go`

### `./internal/application/usecase/overhead_allocation_uc`
- `create_overhead_allocation_uc.go`
- `list_overhead_allocation.go`
- `response_mapper.go`

### `./internal/application/usecase/planned_order_uc`
- `create_planned_order_uc.go`
- `firm_planned_order_uc.go`
- `firm_planned_order_uc_test.go`
- `list_planned_order.go`
- `response_mapper.go`
- `service_requisition_integration_test.go`

### `./internal/application/usecase/planning_params_uc`
- `get_param.go`
- `list_params.go`
- `response_mapper.go`
- `update_param.go`

### `./internal/application/usecase/planning_uc`
- `run_pipeline_uc.go`

### `./internal/application/usecase/procurement_uc`
- `procurement_closeout_uc.go`
- `procurement_governance_uc.go`
- `procurement_uc.go`
- `procurement_uc_test.go`

### `./internal/application/usecase/production_order_uc`
- `add_appointment_uc.go`
- `add_consumption_uc.go`
- `cancel_production_order_uc.go`
- `close_production_order_uc.go`
- `complete_production_order_uc.go`
- `coproduct_receipt_integration_test.go`
- `create_production_order_uc.go`
- `get_production_order_uc.go`
- `list_production_orders_uc.go`
- `order_operations_uc.go`
- `response_mapper.go`
- `return_scrap_uc.go`
- `settle_production_cost_uc.go`
- `start_production_order_uc.go`
- `stock_settlement_uc_test.go`
- `tool_life_hook_integration_test.go`

### `./internal/application/usecase/production_plan_uc`
- `create_plan.go`
- `delete_plan.go`
- `get_plan.go`
- `list_plans.go`
- `response_mapper.go`
- `update_plan.go`

### `./internal/application/usecase/product_uc`
- `create_product.go`
- `delete_product.go`
- `response_mapper.go`
- `search_byid_product.go`

### `./internal/application/usecase/purchase_order_uc`
- `add_purchase_order_item_uc.go`
- `approve_purchase_order_uc.go`
- `cancel_purchase_order_uc.go`
- `create_purchase_order_uc.go`
- `get_purchase_order_uc.go`
- `purchase_suggestion_test.go`
- `purchase_suggestion_uc.go`
- `receive_purchase_order_uc.go`
- `receive_purchase_order_uc_test.go`
- `response_mapper.go`
- `update_purchase_order_uc.go`

### `./internal/application/usecase/purchase_price_uc`
- `purchase_price_uc.go`
- `response_mapper.go`

### `./internal/application/usecase/purchase_quotation_uc`
- `generate_orders_from_quotation_uc.go`
- `purchase_quotation_uc.go`
- `purchase_quotation_uc_test.go`
- `response_mapper.go`

### `./internal/application/usecase/purchase_requisition_uc`
- `generate_integration_test.go`
- `generate_purchase_orders_uc.go`
- `purchase_requisition_uc.go`
- `response_mapper.go`

### `./internal/application/usecase/quality_uc`
- `quality_uc.go`

### `./internal/application/usecase/recurring_sales_uc`
- `mapper.go`
- `order_generation.go`
- `recurring_sales_uc.go`
- `recurring_sales_uc_test.go`

### `./internal/application/usecase/representative_uc`
- `mapper.go`
- `representative_uc.go`
- `representative_uc_test.go`

### `./internal/application/usecase/restriction_uc`
- `create_restriction.go`
- `create_restriction_reason.go`
- `deactivate_restriction.go`
- `delete_restriction_reason.go`
- `evaluate_combination.go`
- `evaluate_combination_test.go`
- `evaluate_restrictions.go`
- `get_by_customer.go`
- `get_by_item.go`
- `get_restriction.go`
- `get_restriction_reason.go`
- `list_restriction_reasons.go`
- `list_restrictions.go`
- `response_mapper.go`
- `update_restriction.go`
- `update_restriction_reason.go`

### `./internal/application/usecase/routing_uc`
- `lead_time_uc.go`
- `operation_uc.go`
- `route_uc.go`
- `route_uc_test.go`

### `./internal/application/usecase/sales_division_uc`
- `create_sales_division.go`
- `delete_sales_division.go`
- `get_sales_division.go`
- `list_sales_divisions.go`
- `response_mapper.go`
- `update_sales_division.go`

### `./internal/application/usecase/sales_forecast_uc`
- `create_appropriation.go`
- `create_block.go`
- `create_forecast.go`
- `generate_forecast.go`
- `generate_forecast_test.go`
- `get_forecast_by_item.go`
- `helpers.go`
- `list_appropriations.go`
- `list_blocks.go`
- `list_forecasts.go`
- `response_mapper.go`
- `set_default_appropriation.go`

### `./internal/application/usecase/sales_goal_uc`
- `mapper.go`
- `sales_goal_uc.go`
- `sales_goal_uc_test.go`

### `./internal/application/usecase/sales_order_uc`
- `change_status_demand_uc_test.go`
- `create_sales_order_uc.go`
- `credit_check.go`
- `get_sales_order_uc.go`
- `manage_sales_order_uc.go`
- `order_reserve.go`
- `response_mapper.go`
- `sales_order_item_uc.go`
- `update_sales_order_uc.go`

### `./internal/application/usecase/sales_quotation_uc`
- `convert_uc.go`
- `items_uc.go`
- `items_uc_test.go`
- `mapper.go`
- `sales_quotation_uc.go`

### `./internal/application/usecase/shipment_uc`
- `auto_fill_uc.go`
- `auto_fill_uc_test.go`
- `export_uc.go`
- `load_uc.go`
- `load_uc_test.go`
- `response_mapper.go`
- `shipment_uc.go`
- `shipment_uc_test.go`

### `./internal/application/usecase/stock_movement_uc`
- `response_mapper.go`
- `stock_movement_uc.go`

### `./internal/application/usecase/stock_uc`
- `adjust_inventory_uc.go`
- `close_inventory_uc.go`
- `consume_reservation_uc.go`
- `consumption_average_uc.go`
- `count_inventory_item_uc.go`
- `create_inventory_uc.go`
- `create_stock_movement_uc.go`
- `create_stock_movement_uc_test.go`
- `get_inventory_uc.go`
- `get_stock_balance_uc.go`
- `list_inventories_uc.go`
- `list_stock_movements_uc.go`
- `lot_uc.go`
- `release_reservation_uc.go`
- `reserve_stock_uc.go`
- `response_mapper.go`

### `./internal/application/usecase/structure_uc`
- `consult_structure_usecase.go`
- `create_structure.go`
- `get_all_direct_children.go`
- `get_structure_tree.go`
- `resolve_structure_query_usecase.go`
- `uptade_structure.go`
- `where_used_usecase.go`

### `./internal/application/usecase/supplier_uc`
- `purchasing_defaults.go`
- `response_mapper.go`
- `sefaz_query_uc.go`
- `supplier_uc.go`

### `./internal/application/usecase/technical_assistance_uc`
- `mapper.go`
- `technical_assistance_uc.go`
- `technical_assistance_uc_test.go`

### `./internal/application/usecase/tool_sheet_uc`
- `tool_sheet_uc.go`
- `tool_sheet_uc_test.go`

### `./internal/application/usecase/tool_uc`
- `tool_uc.go`

### `./internal/application/usecase/user_uc`
- `login_user.go`
- `register_user.go`

### `./internal/application/usecase/warehouse_uc`
- `create_warehouse.go`
- `get_warehouse.go`
- `list_warehouses.go`
- `response_mapper.go`

### `./internal/domain/accounting/ecd`
- `ecd_generator.go`
- `ecd_types.go`

### `./internal/domain/accounting/entity`
- `entity.go`

### `./internal/domain/accounting/repository`
- `repository.go`

### `./internal/domain/allocation_base/entity`
- `allocation_base.go`

### `./internal/domain/allocation_base/repository`
- `allocation_base_repository.go`

### `./internal/domain/aps/entity`
- `entity.go`

### `./internal/domain/aps/repository`
- `repository.go`

### `./internal/domain/bom_header/entity`
- `entity.go`

### `./internal/domain/bom_header/repository`
- `repository.go`

### `./internal/domain/cnpj/entity`
- `company.go`

### `./internal/domain/cnpj/service`
- `provider.go`

### `./internal/domain/component/entity`
- `component_entity.go`
- `new.go`

### `./internal/domain/component/repository`
- `component_repository.go`

### `./internal/domain/component/valueobject`
- `generate_code_component.go`

### `./internal/domain/configurator/entity`
- `entity.go`
- `entity_test.go`

### `./internal/domain/consumer_service/entity`
- `consumer_service.go`

### `./internal/domain/consumer_service/repository`
- `repository.go`

### `./internal/domain/cost_center/entity`
- `cost_center_entity.go`

### `./internal/domain/cost_center/repository`
- `cost_center_repository.go`

### `./internal/domain/crp/entity`
- `entity.go`

### `./internal/domain/crp/repository`
- `repository.go`

### `./internal/domain/customer/entity`
- `commercial_policy_test.go`
- `entity.go`
- `pricing_test.go`

### `./internal/domain/customer/repository`
- `repository.go`

### `./internal/domain/cutting_plan/entity`
- `cutting_plan.go`
- `remnant.go`

### `./internal/domain/cutting_plan/repository`
- `cutting_plan_repository.go`

### `./internal/domain/cutting_plan/service`
- `cutmap.go`
- `cutmap_test.go`
- `geometry.go`
- `geometry_test.go`
- `guillotine_cuts.go`
- `guillotine_cuts_test.go`
- `guillotine_knapsack.go`
- `guillotine_knapsack_test.go`
- `knapsack.go`
- `knapsack_test.go`

### `./internal/domain/cutting_plan/service/lp`
- `simplex.go`
- `simplex_test.go`

### `./internal/domain/cutting_plan/service`
- `nesting_freerot_test.go`
- `nesting_meta.go`
- `nesting_meta_test.go`
- `nesting_raster.go`
- `nesting_raster_test.go`
- `nesting_trueshape.go`
- `nesting_trueshape_test.go`
- `optimizer_1d_cg.go`
- `optimizer_1d_cg_test.go`
- `optimizer_1d.go`
- `optimizer_1d_test.go`
- `optimizer_2d_cg.go`
- `optimizer_2d_cg_test.go`
- `optimizer_2d_guillotine.go`
- `optimizer_2d_guillotine_test.go`
- `optimizer_cg_round.go`
- `optimizer.go`
- `uom.go`
- `uom_test.go`

### `./internal/domain/delivery_promise/entity`
- `delivery_promise_entity.go`

### `./internal/domain/delivery_promise_params/entity`
- `delivery_promise_params_entity.go`

### `./internal/domain/delivery_promise_params/repository`
- `delivery_promise_params_repository.go`

### `./internal/domain/delivery_promise/repository`
- `delivery_promise_repository.go`

### `./internal/domain/delivery_reschedule/entity`
- `delivery_reschedule_entity.go`

### `./internal/domain/delivery_reschedule/repository`
- `delivery_reschedule_repository.go`

### `./internal/domain/drawing/entity`
- `entity.go`

### `./internal/domain/employee/entity`
- `employee_entity.go`
- `new.go`

### `./internal/domain/employee/repository`
- `employee_repository.go`

### `./internal/domain/enterprise/entity`
- `enterprise_entity.go`
- `new.go`

### `./internal/domain/enterprise/repository`
- `enterprise_repository.go`

### `./internal/domain/entry_operation/entity`
- `entity.go`
- `entity_test.go`

### `./internal/domain/entry_operation/repository`
- `repository.go`

### `./internal/domain/enums/types`
- `CostCenterEnum.go`
- `DemandType.go`
- `ItemHealth.go`
- `ItemMRPType.go`
- `ItemSituation.go`
- `ItemStatus.go`
- `ItemStruct.go`
- `ItemType.go`
- `ItemTypeOfUse.go`
- `ItemUnitOfMeasurement.go`
- `LocationType.go`
- `MachineType.go`
- `OrderType.go`
- `WarehouseType.go`

### `./internal/domain/financial/entity`
- `adiantamento_entity.go`
- `centro_custo_entity.go`
- `condicao_pagamento_entity.go`
- `conta_bancaria_entity.go`
- `conta_pagar_entity.go`
- `conta_receber_entity.go`
- `fluxo_caixa_entity.go`
- `plano_contas_entity.go`
- `tax_assessment_entity.go`

### `./internal/domain/financial/repository`
- `financial_repository.go`

### `./internal/domain/fiscal_classification/entity`
- `entity.go`

### `./internal/domain/fiscal_classification/repository`
- `repository.go`

### `./internal/domain/fiscal/engine`
- `tax_engine.go`
- `tax_engine_test.go`

### `./internal/domain/fiscal/entity`
- `cte_entity.go`
- `fiscal_config_entity.go`
- `fiscal_entry_entity.go`
- `fiscal_exit_entity.go`
- `fiscal_params_entity.go`
- `ncm_tax_entity.go`
- `tax_scenario_entity.go`

### `./internal/domain/fiscal/repository`
- `fiscal_params_repository.go`
- `fiscal_repository.go`

### `./internal/domain/fiscal/sped`
- `efd_generator.go`
- `efd_generator_test.go`
- `efd_types.go`

### `./internal/domain/generate_mask_for_item/entity`
- `generate_mask_for_item.go`
- `new.go`

### `./internal/domain/generate_mask_for_item/mask/service`
- `mask_propagation_service.go`

### `./internal/domain/generate_mask_for_item/repository`
- `generate_mask_for_product.go`

### `./internal/domain/generate_mask_for_item/valueobject`
- `generate_mask_for_product.go`

### `./internal/domain/group/entity`
- `group_entity.go`
- `new.go`

### `./internal/domain/group/repository`
- `group_repository.go`

### `./internal/domain/ibpt/entity`
- `entity.go`

### `./internal/domain/ibpt/repository`
- `repository.go`

### `./internal/domain/independent_demand/entity`
- `independent_demand_entity.go`

### `./internal/domain/independent_demand/repository`
- `independent_demand_repository.go`

### `./internal/domain/industrial_calendar/entity`
- `industrial_calendar_entity.go`

### `./internal/domain/industrial_calendar/repository`
- `industrial_calendar_repository.go`

### `./internal/domain/item_calendar_promise/entity`
- `item_calendar_promise_entity.go`

### `./internal/domain/item_calendar_promise/repository`
- `item_calendar_promise_repository.go`

### `./internal/domain/item_conversion/entity`
- `entity.go`

### `./internal/domain/item_conversion/repository`
- `repository.go`

### `./internal/domain/items/entity`
- `classification_entity.go`
- `item_entity.go`
- `new.go`

### `./internal/domain/items/repository`
- `classification_repository.go`
- `item_repository.go`

### `./internal/domain/item_supplier/entity`
- `entity.go`

### `./internal/domain/item_supplier/repository`
- `repository.go`

### `./internal/domain/items/valueobject`
- `valueobj.go`
- `valueobj_test.go`

### `./internal/domain/location/entity`
- `entity.go`

### `./internal/domain/location/repository`
- `repository.go`

### `./internal/domain/lot_mask/entity`
- `entity.go`
- `entity_test.go`

### `./internal/domain/machine/entity`
- `machine_entity.go`

### `./internal/domain/machine/repository`
- `machine_repository.go`

### `./internal/domain/machine/service`
- `machine_service.go`
- `production_time.go`
- `production_time_test.go`
- `unit_conversion.go`
- `unit_conversion_test.go`

### `./internal/domain/maintenance/entity`
- `entity.go`

### `./internal/domain/maintenance/repository`
- `repository.go`

### `./internal/domain/modifier/entity`
- `modifier_entity.go`
- `new.go`

### `./internal/domain/modifier/repository`
- `modifier_repository.go`

### `./internal/domain/mrp_calculation/entity`
- `mrp_calculation_entity.go`
- `planning_params.go`

### `./internal/domain/mrp_calculation/ports`
- `supply_port.go`

### `./internal/domain/mrp_calculation/repository`
- `mrp_calculation_repository.go`

### `./internal/domain/mrp_calculation/service`
- `mrp_helpers_extra_test.go`
- `mrp_helpers_test.go`
- `mrp_service.go`
- `mrp_service_impl.go`

### `./internal/domain/nfse/entity`
- `nfse_entity.go`

### `./internal/domain/nfse/repository`
- `nfse_repository.go`

### `./internal/domain/order_priority/entity`
- `order_priority.go`

### `./internal/domain/order_priority/repository`
- `order_priority_repository.go`

### `./internal/domain/overhead_allocation/entity`
- `overhead_allocation_entity.go`

### `./internal/domain/overhead_allocation/repository`
- `overhead_allocation_repository.go`

### `./internal/domain/planned_order/entity`
- `planned_order_entity.go`

### `./internal/domain/planned_order/repository`
- `planned_order_repository.go`

### `./internal/domain/planning_params/entity`
- `planning_param_entity.go`

### `./internal/domain/planning_params/repository`
- `planning_param_repository.go`

### `./internal/domain/procurement/entity`
- `closeout.go`
- `closeout_test.go`
- `entity.go`
- `entity_test.go`

### `./internal/domain/procurement/repository`
- `repository.go`

### `./internal/domain/product/entity`
- `new.go`
- `product_entity.go`

### `./internal/domain/production_order/entity`
- `cost.go`
- `cost_test.go`
- `production_order_entity.go`

### `./internal/domain/production_order/repository`
- `production_order_repository.go`

### `./internal/domain/production_plan/entity`
- `new.go`
- `production_plan_entity.go`

### `./internal/domain/production_plan/repository`
- `production_plan_repository.go`

### `./internal/domain/product/repository`
- `errros.go`
- `product_repository.go`

### `./internal/domain/product/valueobject`
- `generate_code_product.go`
- `generate_code_product_mask.go`
- `quantity.go`

### `./internal/domain/purchase_order/entity`
- `purchase_order_entity.go`

### `./internal/domain/purchase_order/repository`
- `purchase_order_repository.go`

### `./internal/domain/purchase_price/entity`
- `entity.go`

### `./internal/domain/purchase_price/repository`
- `repository.go`

### `./internal/domain/purchase_quotation/entity`
- `entity.go`

### `./internal/domain/purchase_quotation/repository`
- `repository.go`

### `./internal/domain/purchase_requisition/entity`
- `entity.go`
- `entity_test.go`

### `./internal/domain/purchase_requisition/repository`
- `repository.go`

### `./internal/domain/quality/entity`
- `entity.go`

### `./internal/domain/quality/repository`
- `repository.go`

### `./internal/domain/recurring_sales/entity`
- `recurring_sales.go`

### `./internal/domain/recurring_sales/repository`
- `repository.go`

### `./internal/domain/representative/entity`
- `representative.go`

### `./internal/domain/representative/repository`
- `repository.go`

### `./internal/domain/restriction/entity`
- `new.go`
- `restriction_entity.go`
- `restriction_reason_entity.go`

### `./internal/domain/restriction/repository`
- `restriction_reason_repository.go`
- `restriction_repository.go`

### `./internal/domain/routing/entity`
- `critical_path.go`
- `critical_path_test.go`
- `entity.go`
- `operation_time.go`
- `operation_time_test.go`
- `route_effectivity_test.go`

### `./internal/domain/routing/repository`
- `repository.go`

### `./internal/domain/sales_division/entity`
- `new.go`
- `sales_division_entity.go`

### `./internal/domain/sales_division/repository`
- `sales_division_repository.go`

### `./internal/domain/sales_forecast/entity`
- `new.go`
- `sales_forecast_entity.go`

### `./internal/domain/sales_forecast/repository`
- `sales_forecast_repository.go`

### `./internal/domain/sales_goal/entity`
- `sales_goal.go`

### `./internal/domain/sales_goal/repository`
- `repository.go`

### `./internal/domain/sales_order/entity`
- `credit.go`
- `credit_test.go`
- `sales_order_entity.go`

### `./internal/domain/sales_order/repository`
- `sales_order_repository.go`

### `./internal/domain/sales_quotation/entity`
- `sales_quotation.go`

### `./internal/domain/sales_quotation/repository`
- `repository.go`

### `./internal/domain/shipment/entity`
- `entity.go`
- `entity_test.go`

### `./internal/domain/shipment/repository`
- `repository.go`

### `./internal/domain/standard_cost/entity`
- `entity.go`

### `./internal/domain/standard_cost/repository`
- `repository.go`

### `./internal/domain/stock/entity`
- `consumption_average.go`
- `costing.go`
- `costing_test.go`
- `lot.go`
- `stock_entity.go`
- `stock_entity_test.go`

### `./internal/domain/stock_movement/entity`
- `entity.go`

### `./internal/domain/stock_movement/repository`
- `repository.go`

### `./internal/domain/stock/repository`
- `stock_repository.go`

### `./internal/domain/structure/entity`
- `new.go`
- `structure_entity.go`
- `substitute.go`
- `substitute_test.go`

### `./internal/domain/structure/formula`
- `evaluator.go`

### `./internal/domain/structure_query/repository`
- `structure_query_repository.go`

### `./internal/domain/structure_query/service`
- `new.go`
- `resolver.go`

### `./internal/domain/structure/repository`
- `item_structure_repository.go`

### `./internal/domain/structure/valueobject`
- `structure_node_obj.go`

### `./internal/domain/supplier/entity`
- `entity.go`
- `entity_test.go`

### `./internal/domain/supplier/repository`
- `repository.go`

### `./internal/domain/technical_assistance/entity`
- `technical_assistance.go`

### `./internal/domain/technical_assistance/repository`
- `repository.go`

### `./internal/domain/tool/entity`
- `entity.go`
- `entity_test.go`

### `./internal/domain/tool/repository`
- `repository.go`

### `./internal/domain/user/entity`
- `new.go`
- `user_entity.go`

### `./internal/domain/user/repository`
- `user_repository.go`

### `./internal/domain/warehouse/entity`
- `new.go`
- `warehouse_entity.go`

### `./internal/domain/warehouse/repository`
- `warehouse_repository.go`

### `./internal/infrastructure/audit`
- `audit.go`
- `reader.go`

### `./internal/infrastructure/auth`
- `auth_service.go`
- `jwt.go`

### `./internal/infrastructure/cnab`
- `cnab240.go`
- `cnab240_test.go`

### `./internal/infrastructure/cnpj`
- `brasilapi.go`
- `client.go`
- `cnpja.go`
- `cnpj_test.go`

### `./internal/infrastructure/config`
- `config.go`

### `./internal/infrastructure/database`
- `db.go`

### `./internal/infrastructure/database/nullable`
- `nullable.go`

### `./internal/infrastructure/database/pgutil`
- `bool.go`
- `int.go`
- `json.go`
- `numeric.go`
- `scan.go`
- `slices.go`
- `text.go`
- `time.go`
- `uuid.go`

### `./internal/infrastructure/database/sqlc`
- `accounting.sql.go`
- `allocation.sql.go`
- `aps_gantt.sql.go`
- `aps.sql.go`
- `bom_header.sql.go`
- `component.sql.go`
- `configurator_descriptions.sql.go`
- `configurator_rules.sql.go`
- `configurator.sql.go`
- `copyfrom.go`
- `cost_center.sql.go`
- `crp.sql.go`
- `customer.sql.go`
- `cutting_plan.sql.go`
- `db.go`
- `delivery_promise_params.sql.go`
- `delivery_reschedule.sql.go`
- `drawings.sql.go`
- `employee.sql.go`
- `enterprise.sql.go`
- `entry_operation.sql.go`
- `fiscal_classification.sql.go`
- `fiscal_params.sql.go`
- `group.sql.go`
- `independent_demand.sql.go`
- `industrial_calendar_ext.go`
- `industrial_calendar.sql.go`
- `item_calendar_promise.sql.go`
- `item_conversion.sql.go`
- `item_mask.sql.go`
- `item.sql.go`
- `item_supplier.sql.go`
- `lot_masks.sql.go`
- `machine.sql.go`
- `maintenance.sql.go`
- `models.go`
- `modifier.sql.go`
- `mrp_bulk.go`
- `mrp_calculation.sql.go`
- `mrp_exception_messages.go`
- `order_priority.sql.go`
- `overhead_allocation.sql.go`
- `planned_order.sql.go`
- `planning_params.sql.go`
- `production_order_operations.sql.go`
- `production_plan.sql.go`
- `product_mask.sql.go`
- `product.sql.go`
- `purchase_price.sql.go`
- `purchase_quotation.sql.go`
- `purchase_requisition.sql.go`
- `quality.sql.go`
- `restriction_reason.sql.go`
- `restriction.sql.go`
- `routing.sql.go`
- `sales_division.sql.go`
- `sales_forecast.sql.go`
- `sales_order.sql.go`
- `standard_cost.sql.go`
- `structure_bom.go`
- `structure_cfg.sql.go`
- `structure_query.sql.go`
- `structure.sql.go`
- `supplier.sql.go`
- `tool_serials.sql.go`
- `tool.sql.go`
- `users.sql.go`
- `warehouse.sql.go`

### `./internal/infrastructure/database/sqltypes`
- `customer_enums.go`
- `fiscal_params_enums.go`
- `routing_enums.go`

### `./internal/infrastructure/export`
- `csv.go`
- `docx.go`
- `docx_test.go`
- `export_test.go`

### `./internal/infrastructure/export/gantt`
- `gantt.go`
- `gantt_test.go`
- `pdf.go`
- `svg.go`

### `./internal/infrastructure/export`
- `http.go`
- `pdf.go`

### `./internal/infrastructure/export/pdfkit`
- `doc.go`
- `image.go`
- `metrics.go`
- `report.go`
- `report_test.go`
- `winansi.go`

### `./internal/infrastructure/export`
- `reflect.go`

### `./internal/infrastructure/export/romaneio`
- `romaneio_data.go`
- `romaneio_pdf.go`
- `romaneio_test.go`
- `romaneio_xlsx.go`

### `./internal/infrastructure/export`
- `table.go`
- `xlsx.go`

### `./internal/infrastructure/focusnfe`
- `client.go`

### `./internal/infrastructure/logger`
- `context.go`
- `logger.go`
- `sanitize.go`

### `./internal/infrastructure/mapper/employee`
- `employee_mapper.go`

### `./internal/infrastructure/mapper/enterprise`
- `enterprise_mapper.go`

### `./internal/infrastructure/mapper/group`
- `group_mapper.go`

### `./internal/infrastructure/mapper/item`
- `item_mapper.go`

### `./internal/infrastructure/mapper/modifier`
- `modifier_mapper.go`

### `./internal/infrastructure/mapper/structure_query`
- `structure_query_mapper.go`

### `./internal/infrastructure/mapper/structure`
- `structure_mapper.go`

### `./internal/infrastructure/mapper/warehouse`
- `warehouse_mapper.go`

### `./internal/infrastructure/nesting`
- `http_provider.go`
- `http_provider_test.go`

### `./internal/infrastructure/notification`
- `smtp_email_service.go`
- `webhook.go`

### `./internal/infrastructure/observability`
- `tracing.go`

### `./internal/infrastructure/repository/accounting`
- `accounting_repository.go`

### `./internal/infrastructure/repository/allocation_base`
- `allocation_base_repository_sqlc.go`
- `new.go`

### `./internal/infrastructure/repository/aps`
- `aps_repository_sqlc.go`

### `./internal/infrastructure/repository/bom_header`
- `bom_header_integration_test.go`
- `repository_sqlc.go`

### `./internal/infrastructure/repository/components`
- `new.go`
- `repository_sqlc.go`

### `./internal/infrastructure/repository/consumer_service`
- `repository.go`

### `./internal/infrastructure/repository/cost_center`
- `cost_center_repository_sqlc.go`
- `new.go`

### `./internal/infrastructure/repository/crp`
- `crp_repository_sqlc.go`

### `./internal/infrastructure/repository/customer`
- `customer_repository_sqlc.go`

### `./internal/infrastructure/repository/cutting_plan`
- `cutting_plan_repository_sqlc.go`

### `./internal/infrastructure/repository/delivery_promise_params`
- `delivery_promise_params_repository_sqlc.go`
- `new.go`

### `./internal/infrastructure/repository/delivery_promise`
- `repository.go`

### `./internal/infrastructure/repository/delivery_reschedule`
- `delivery_reschedule_repository_sqlc.go`
- `new.go`

### `./internal/infrastructure/repository/employee`
- `new.go`
- `repository_sqlc.go`

### `./internal/infrastructure/repository/enterprise`
- `new.go`
- `repository_sqlc.go`

### `./internal/infrastructure/repository/entry_operation`
- `entry_operation_integration_test.go`
- `entry_operation_repository_sqlc.go`

### `./internal/infrastructure/repository/financial`
- `adiantamento_integration_test.go`
- `adiantamento_repository_pg.go`
- `financial_repository_pg.go`
- `new.go`

### `./internal/infrastructure/repository/fiscal_classification`
- `fiscal_classification_repository_sqlc.go`

### `./internal/infrastructure/repository/fiscal`
- `fiscal_integration_test.go`
- `fiscal_params_repository.go`
- `fiscal_repository_pg.go`

### `./internal/infrastructure/repository/group`
- `new.go`
- `repository_sqlc.go`

### `./internal/infrastructure/repository/ibpt`
- `ibpt_repository_pg.go`

### `./internal/infrastructure/repository/independent_demand`
- `independent_demand_repository_sqlc.go`
- `new.go`

### `./internal/infrastructure/repository/industrial_calendar`
- `industrial_calendar_repository_sqlc.go`
- `new.go`

### `./internal/infrastructure/repository/item_calendar_promise`
- `item_calendar_promise_repository_sqlc.go`
- `new.go`

### `./internal/infrastructure/repository/item_classification`
- `item_classification_repository.go`

### `./internal/infrastructure/repository/item_conversion`
- `item_conversion_integration_test.go`
- `item_conversion_repository_sqlc.go`

### `./internal/infrastructure/repository/item`
- `new.go`
- `repository_sqlc.go`

### `./internal/infrastructure/repository/item_supplier`
- `item_supplier_repository_sqlc.go`

### `./internal/infrastructure/repository/location`
- `location_repository.go`

### `./internal/infrastructure/repository/machine`
- `machine_repository_sqlc.go`
- `new.go`

### `./internal/infrastructure/repository/maintenance`
- `maintenance_repository_sqlc.go`

### `./internal/infrastructure/repository/modifier`
- `new.go`
- `repository_sqlc.go`

### `./internal/infrastructure/repository/mrp_calculation`
- `mrp_calculation_repository_sqlc.go`
- `mrp_enhanced_repo.go`
- `new.go`

### `./internal/infrastructure/repository/nfse`
- `nfse_integration_test.go`
- `nfse_repository_pg.go`

### `./internal/infrastructure/repository/order_priority`
- `new.go`
- `order_priority_repository_sqlc.go`

### `./internal/infrastructure/repository/overhead_allocation`
- `new.go`
- `overhead_allocation_repository_sqlc.go`

### `./internal/infrastructure/repository/planned_order`
- `mrp_supply_adapter.go`
- `planned_order_repository_sqlc.go`

### `./internal/infrastructure/repository/planning_params`
- `new.go`
- `repository_sqlc.go`

### `./internal/infrastructure/repository/procurement`
- `repository_closeout.go`
- `repository.go`

### `./internal/infrastructure/repository/production_order`
- `production_order_repository.go`

### `./internal/infrastructure/repository/production_plan`
- `new.go`
- `repository_sqlc.go`

### `./internal/infrastructure/repository/product`
- `new.go`
- `repository_sqlc.go`

### `./internal/infrastructure/repository/purchase_order`
- `new.go`
- `purchase_order_repository_sqlc.go`

### `./internal/infrastructure/repository/purchase_price`
- `purchase_price_integration_test.go`
- `purchase_price_repository_sqlc.go`

### `./internal/infrastructure/repository/purchase_quotation`
- `purchase_quotation_repository_sqlc.go`

### `./internal/infrastructure/repository/purchase_requisition`
- `purchase_requisition_integration_test.go`
- `purchase_requisition_repository_sqlc.go`

### `./internal/infrastructure/repository/quality`
- `quality_repository_sqlc.go`

### `./internal/infrastructure/repository/recurring_sales`
- `repository.go`

### `./internal/infrastructure/repository/representative`
- `repository.go`

### `./internal/infrastructure/repository/restriction`
- `new.go`
- `reason_repository_sqlc.go`
- `repository_sqlc.go`

### `./internal/infrastructure/repository/routing`
- `effectivity_integration_test.go`
- `resources_integration_test.go`
- `routing_integration_test.go`
- `routing_repository_sqlc.go`

### `./internal/infrastructure/repository/sales_division`
- `new.go`
- `repository_sqlc.go`

### `./internal/infrastructure/repository/sales_forecast`
- `new.go`
- `repository_sqlc.go`

### `./internal/infrastructure/repository/sales_goal`
- `repository.go`

### `./internal/infrastructure/repository/sales_order`
- `new.go`
- `sales_order_repository_sqlc.go`

### `./internal/infrastructure/repository/sales_quotation`
- `repository.go`

### `./internal/infrastructure/repository/shipment`
- `adapters.go`
- `adapters_test.go`
- `romaneio_enricher.go`
- `shipment_repository_pg.go`

### `./internal/infrastructure/repository/standard_cost`
- `standard_cost_repository_sqlc.go`

### `./internal/infrastructure/repository/stock_movement`
- `repository.go`

### `./internal/infrastructure/repository/stock`
- `stock_repository_pg.go`

### `./internal/infrastructure/repository/structure`
- `item_structure_presenter.go`
- `item_structure_repository_sqlc.go`
- `new.go`

### `./internal/infrastructure/repository/structure_query`
- `item_structure_repository_query_sqlc.go`
- `new.go`

### `./internal/infrastructure/repository/supplier`
- `supplier_integration_test.go`
- `supplier_repository_sqlc.go`

### `./internal/infrastructure/repository/technical_assistance`
- `repository.go`

### `./internal/infrastructure/repository/tool`
- `tool_integration_test.go`
- `tool_repository_sqlc.go`

### `./internal/infrastructure/repository/user`
- `new.go`
- `repository_sqlc.go`

### `./internal/infrastructure/repository/warehouse`
- `new.go`
- `repository_sqlc.go`

### `./internal/infrastructure/testutil`
- `db.go`

### `./internal/interfaces/http/context`
- `context.go`

### `./internal/interfaces/http/handler`
- `accounting_handler.go`
- `adiantamento_handler.go`
- `allocation_base_handler.go`
- `aps_handler.go`
- `audit_handler.go`
- `bom_header_handler.go`
- `cnab_handler.go`
- `cnpj_handler.go`
- `configurator_handler.go`
- `consumer_service_handler.go`
- `cost_center_handler.go`
- `create_delivery_reschedule_handler.go`
- `create_employee_handler.go`
- `create_enterprise_handler.go`
- `create_group_handler.go`
- `create_item_handler.go`
- `create_modifier_handler.go`
- `create_product_handler.go`
- `create_warehouse_handler.go`
- `crp_handler.go`
- `cte_authorize_handler.go`
- `customer_handler.go`
- `cutting_plan_handler.go`
- `delete_product_handler.go`
- `delivery_promise_handler.go`
- `delivery_promise_params.go`
- `drawing_handler.go`
- `entry_operation_handler.go`
- `financial_handler.go`
- `find_item_by_code_handler.go`
- `fiscal_branding_handler.go`
- `fiscal_classification_handler.go`
- `fiscal_handler.go`
- `fiscal_manifest_handler.go`
- `fiscal_params_handler.go`
- `forecast_handler.go`
- `handler.go`
- `ibpt_handler.go`
- `icms_apuracao_handler.go`
- `icms_reduction_handler.go`
- `import_nfe_handler.go`
- `independent_demand_handler.go`
- `industrial_calendar.go`
- `item_activation_handler.go`
- `item_calendar_promise_handler.go`
- `item_classification_handler.go`
- `item_conversion_handler.go`
- `item_structure_handler.go`
- `item_supplier_handler.go`
- `list_items_handler.go`
- `location_handler.go`
- `login_user.go`
- `lot_mask_handler.go`
- `machine_handler.go`
- `maintenance_handler.go`
- `manage_group_handler.go`
- `manage_modifier_handler.go`
- `mrp_calculation_handler.go`
- `mrp_exceptions_handler.go`
- `new.go`
- `nfse_handler.go`
- `order_priority_handler.go`
- `overhead_allocation_handler.go`
- `planned_order_handler.go`
- `planning_handler.go`
- `planning_params_handler.go`
- `procurement_handler.go`
- `production_order_handler.go`
- `production_plan_handler.go`
- `purchase_order_handler.go`
- `purchase_order_item_handler.go`
- `purchase_price_handler.go`
- `purchase_quotation_handler.go`
- `purchase_requisition_handler.go`
- `purchase_suggestion_handler.go`
- `quality_handler.go`
- `recurring_sales_handler.go`
- `register_user.go`
- `report_export_handler.go`
- `representative_handler.go`
- `resolve_structure_query_handler.go`
- `restriction_handler.go`
- `restriction_reason_handler.go`
- `routing_handler.go`
- `sales_division_handler.go`
- `sales_forecast_handler.go`
- `sales_goal_handler.go`
- `sales_order_handler.go`
- `sales_quotation_handler.go`

### `./internal/interfaces/http/handler/security`
- `base_handler.go`
- `response.go`
- `response_helpers.go`
- `usecase_error.go`

### `./internal/interfaces/http/handler`
- `shipment_handler.go`
- `sped_handler.go`
- `standard_cost_handler.go`
- `stock_handler.go`
- `stock_movement_handler.go`
- `supplier_handler.go`
- `supplier_sefaz_handler.go`
- `technical_assistance_handler.go`
- `tool_handler.go`
- `tool_sheet_handler.go`

### `./internal/interfaces/middleware`
- `audit.go`
- `audit_test.go`
- `correlation.go`
- `cors.go`
- `hardening_test.go`
- `idempotency.go`
- `jwt.go`
- `metrics.go`
- `permissions.go`
- `permissions_test.go`
- `ratelimit.go`
- `request_logger.go`
- `security_headers.go`

### `./internal/pkg/datetime`
- `datetime.go`

### `./internal/pkg/validation`
- `document.go`
- `document_test.go`

### `./scripts`
- `fix_sqlc_output.go`

## Tipos, interfaces e funções exportadas

```text
```

## Rotas HTTP

```text
```

## Interfaces de repositório

```text
```

## Queries SQLC

```text
```

## Tabelas e tipos criados por migrations

```text
```

## Arquivos de teste

- `internal/application/usecase/aps_uc/aps_scheduling_test.go`
- `internal/application/usecase/aps_uc/gantt_month_uc_test.go`
- `internal/application/usecase/aps_uc/reschedule_uc_test.go`
- `internal/application/usecase/configurator_uc/configurator_integration_test.go`
- `internal/application/usecase/consumer_service_uc/consumer_service_uc_test.go`
- `internal/application/usecase/cost_uc/coproduct_cost_integration_test.go`
- `internal/application/usecase/cost_uc/cost_integration_test.go`
- `internal/application/usecase/cost_uc/cost_rollup_helpers_test.go`
- `internal/application/usecase/crp_uc/crp_uc_test.go`
- `internal/application/usecase/cutting_plan_uc/demand_uc_test.go`
- `internal/application/usecase/cutting_plan_uc/release_uc_test.go`
- `internal/application/usecase/delivery_promise_uc/delivery_promise_uc_test.go`
- `internal/application/usecase/drawing_uc/drawing_integration_test.go`
- `internal/application/usecase/financial_uc/baixa_adiantamento_uc_test.go`
- `internal/application/usecase/financial_uc/crud_uc_test.go`
- `internal/application/usecase/financial_uc/importar_ofx_uc_test.go`
- `internal/application/usecase/fiscal_params_uc/icms_apuracao_uc_test.go`
- `internal/application/usecase/fiscal_uc/authorize_fiscal_exit_uc_test.go`
- `internal/application/usecase/fiscal_uc/create_fiscal_exit_from_load_uc_test.go`
- `internal/application/usecase/item_conversion_uc/item_conversion_uc_test.go`
- `internal/application/usecase/lot_mask_uc/lot_mask_integration_test.go`
- `internal/application/usecase/mrp_uc/firmar_sugestao_uc_test.go`
- `internal/application/usecase/nfse_uc/nfse_uc_test.go`
- `internal/application/usecase/planned_order_uc/firm_planned_order_uc_test.go`
- `internal/application/usecase/planned_order_uc/service_requisition_integration_test.go`
- `internal/application/usecase/procurement_uc/procurement_uc_test.go`
- `internal/application/usecase/production_order_uc/coproduct_receipt_integration_test.go`
- `internal/application/usecase/production_order_uc/stock_settlement_uc_test.go`
- `internal/application/usecase/production_order_uc/tool_life_hook_integration_test.go`
- `internal/application/usecase/purchase_order_uc/purchase_suggestion_test.go`
- `internal/application/usecase/purchase_order_uc/receive_purchase_order_uc_test.go`
- `internal/application/usecase/purchase_quotation_uc/purchase_quotation_uc_test.go`
- `internal/application/usecase/purchase_requisition_uc/generate_integration_test.go`
- `internal/application/usecase/recurring_sales_uc/recurring_sales_uc_test.go`
- `internal/application/usecase/representative_uc/representative_uc_test.go`
- `internal/application/usecase/restriction_uc/evaluate_combination_test.go`
- `internal/application/usecase/routing_uc/route_uc_test.go`
- `internal/application/usecase/sales_forecast_uc/generate_forecast_test.go`
- `internal/application/usecase/sales_goal_uc/sales_goal_uc_test.go`
- `internal/application/usecase/sales_order_uc/change_status_demand_uc_test.go`
- `internal/application/usecase/sales_quotation_uc/items_uc_test.go`
- `internal/application/usecase/shipment_uc/auto_fill_uc_test.go`
- `internal/application/usecase/shipment_uc/load_uc_test.go`
- `internal/application/usecase/shipment_uc/shipment_uc_test.go`
- `internal/application/usecase/stock_uc/create_stock_movement_uc_test.go`
- `internal/application/usecase/technical_assistance_uc/technical_assistance_uc_test.go`
- `internal/application/usecase/tool_sheet_uc/tool_sheet_uc_test.go`
- `internal/domain/configurator/entity/entity_test.go`
- `internal/domain/customer/entity/commercial_policy_test.go`
- `internal/domain/customer/entity/pricing_test.go`
- `internal/domain/cutting_plan/service/cutmap_test.go`
- `internal/domain/cutting_plan/service/geometry_test.go`
- `internal/domain/cutting_plan/service/guillotine_cuts_test.go`
- `internal/domain/cutting_plan/service/guillotine_knapsack_test.go`
- `internal/domain/cutting_plan/service/knapsack_test.go`
- `internal/domain/cutting_plan/service/lp/simplex_test.go`
- `internal/domain/cutting_plan/service/nesting_freerot_test.go`
- `internal/domain/cutting_plan/service/nesting_meta_test.go`
- `internal/domain/cutting_plan/service/nesting_raster_test.go`
- `internal/domain/cutting_plan/service/nesting_trueshape_test.go`
- `internal/domain/cutting_plan/service/optimizer_1d_cg_test.go`
- `internal/domain/cutting_plan/service/optimizer_1d_test.go`
- `internal/domain/cutting_plan/service/optimizer_2d_cg_test.go`
- `internal/domain/cutting_plan/service/optimizer_2d_guillotine_test.go`
- `internal/domain/cutting_plan/service/uom_test.go`
- `internal/domain/entry_operation/entity/entity_test.go`
- `internal/domain/fiscal/engine/tax_engine_test.go`
- `internal/domain/fiscal/sped/efd_generator_test.go`
- `internal/domain/items/valueobject/valueobj_test.go`
- `internal/domain/lot_mask/entity/entity_test.go`
- `internal/domain/machine/service/production_time_test.go`
- `internal/domain/machine/service/unit_conversion_test.go`
- `internal/domain/mrp_calculation/service/mrp_helpers_extra_test.go`
- `internal/domain/mrp_calculation/service/mrp_helpers_test.go`
- `internal/domain/procurement/entity/closeout_test.go`
- `internal/domain/procurement/entity/entity_test.go`
- `internal/domain/production_order/entity/cost_test.go`
- `internal/domain/purchase_requisition/entity/entity_test.go`
- `internal/domain/routing/entity/critical_path_test.go`
- `internal/domain/routing/entity/operation_time_test.go`
- `internal/domain/routing/entity/route_effectivity_test.go`
- `internal/domain/sales_order/entity/credit_test.go`
- `internal/domain/shipment/entity/entity_test.go`
- `internal/domain/stock/entity/costing_test.go`
- `internal/domain/stock/entity/stock_entity_test.go`
- `internal/domain/structure/entity/substitute_test.go`
- `internal/domain/supplier/entity/entity_test.go`
- `internal/domain/tool/entity/entity_test.go`
- `internal/infrastructure/cnab/cnab240_test.go`
- `internal/infrastructure/cnpj/cnpj_test.go`
- `internal/infrastructure/export/docx_test.go`
- `internal/infrastructure/export/export_test.go`
- `internal/infrastructure/export/gantt/gantt_test.go`
- `internal/infrastructure/export/pdfkit/report_test.go`
- `internal/infrastructure/export/romaneio/romaneio_test.go`
- `internal/infrastructure/nesting/http_provider_test.go`
- `internal/infrastructure/repository/bom_header/bom_header_integration_test.go`
- `internal/infrastructure/repository/entry_operation/entry_operation_integration_test.go`
- `internal/infrastructure/repository/financial/adiantamento_integration_test.go`
- `internal/infrastructure/repository/fiscal/fiscal_integration_test.go`
- `internal/infrastructure/repository/item_conversion/item_conversion_integration_test.go`
- `internal/infrastructure/repository/nfse/nfse_integration_test.go`
- `internal/infrastructure/repository/purchase_price/purchase_price_integration_test.go`
- `internal/infrastructure/repository/purchase_requisition/purchase_requisition_integration_test.go`
- `internal/infrastructure/repository/routing/effectivity_integration_test.go`
- `internal/infrastructure/repository/routing/resources_integration_test.go`
- `internal/infrastructure/repository/routing/routing_integration_test.go`
- `internal/infrastructure/repository/shipment/adapters_test.go`
- `internal/infrastructure/repository/supplier/supplier_integration_test.go`
- `internal/infrastructure/repository/tool/tool_integration_test.go`
- `internal/interfaces/middleware/audit_test.go`
- `internal/interfaces/middleware/hardening_test.go`
- `internal/interfaces/middleware/permissions_test.go`
- `internal/pkg/validation/document_test.go`

## Comandos disponíveis no Makefile

```text
backup
build
ci
create_migration
cutting-samples
demo-bootstrap
demo-down
demo-logs
demo-migrate
demo-reset
demo-seed
demo-up
docker-build
down
fmt-check
logs
migrate_down
migrate_force
migrate_up
.PHONY
print_db
reset
restore
run
sqlc
test
test-bom-mrp
test-cover
test-cutting
test-gantt
test-integration
test-procurement-governance
test-purchase-receiving
test-romaneio
up
up-backup
vet
```
