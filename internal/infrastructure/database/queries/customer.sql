-- ─── Regions ─────────────────────────────────────────────────────────────────

-- name: CreateRegion :one
INSERT INTO regions (code, description, uf, city, created_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateRegion :one
UPDATE regions
SET description = $2, uf = $3, city = $4, is_active = $5
WHERE id = $1
RETURNING *;

-- name: GetRegionByCode :one
SELECT * FROM regions WHERE code = $1;

-- name: ListRegions :many
SELECT * FROM regions
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: NextRegionCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM regions;

-- ─── Market Segments ──────────────────────────────────────────────────────────

-- name: CreateMarketSegment :one
INSERT INTO market_segments (code, description, parent_id, has_pis_cofins_retention, retention_indicator)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateMarketSegment :one
UPDATE market_segments
SET description = $2, parent_id = $3, has_pis_cofins_retention = $4,
    retention_indicator = $5, is_active = $6
WHERE id = $1
RETURNING *;

-- name: GetMarketSegmentByCode :one
SELECT * FROM market_segments WHERE code = $1;

-- name: ListMarketSegments :many
SELECT * FROM market_segments
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: NextMarketSegmentCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM market_segments;

-- ─── Customer Contact Types ───────────────────────────────────────────────────

-- name: CreateContactType :one
INSERT INTO customer_contact_types (code, description)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateContactType :one
UPDATE customer_contact_types
SET description = $2, is_active = $3
WHERE id = $1
RETURNING *;

-- name: GetContactTypeByCode :one
SELECT * FROM customer_contact_types WHERE code = $1;

-- name: ListContactTypes :many
SELECT * FROM customer_contact_types
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: NextContactTypeCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM customer_contact_types;

-- ─── Customer Types ───────────────────────────────────────────────────────────

-- name: CreateCustomerType :one
INSERT INTO customer_types (code, description, category, delivery_days)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateCustomerType :one
UPDATE customer_types
SET description = $2, category = $3, delivery_days = $4, is_active = $5
WHERE id = $1
RETURNING *;

-- name: GetCustomerTypeByCode :one
SELECT * FROM customer_types WHERE code = $1;

-- name: ListCustomerTypes :many
SELECT * FROM customer_types
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- ─── Carriers ─────────────────────────────────────────────────────────────────

-- name: CreateCarrier :one
INSERT INTO carriers (code, description, billing_type, uses_credit_limit, consider_available,
                      postpone_due_date, receipt_days, payment_days)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateCarrier :one
UPDATE carriers
SET description = $2, billing_type = $3, uses_credit_limit = $4, consider_available = $5,
    postpone_due_date = $6, receipt_days = $7, payment_days = $8, is_active = $9
WHERE id = $1
RETURNING *;

-- name: GetCarrierByCode :one
SELECT * FROM carriers WHERE code = $1;

-- name: ListCarriers :many
SELECT * FROM carriers
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: NextCarrierCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM carriers;

-- ─── Carrier Groups ───────────────────────────────────────────────────────────

-- name: CreateCarrierGroup :one
INSERT INTO carrier_groups (code, description)
VALUES ($1, $2)
RETURNING *;

-- name: GetCarrierGroupByCode :one
SELECT * FROM carrier_groups WHERE code = $1;

-- name: ListCarrierGroups :many
SELECT * FROM carrier_groups ORDER BY code;

-- name: AddCarrierToGroup :exec
INSERT INTO carrier_group_carriers (carrier_group_id, carrier_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveCarrierFromGroup :exec
DELETE FROM carrier_group_carriers
WHERE carrier_group_id = $1 AND carrier_id = $2;

-- name: NextCarrierGroupCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM carrier_groups;

-- ─── Payment Conditions ───────────────────────────────────────────────────────

-- name: CreatePaymentCondition :one
INSERT INTO payment_conditions (
    code, description, carrier_id, analysis_type, parcel_start,
    expenses, average_term, is_special, is_revenue, is_at_sight
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdatePaymentCondition :one
UPDATE payment_conditions
SET description = $2, carrier_id = $3, analysis_type = $4, parcel_start = $5,
    expenses = $6, average_term = $7, is_special = $8, is_revenue = $9,
    is_at_sight = $10, is_active = $11
WHERE id = $1
RETURNING *;

-- name: GetPaymentConditionByCode :one
SELECT * FROM payment_conditions WHERE code = $1;

-- name: ListPaymentConditions :many
SELECT * FROM payment_conditions
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: NextPaymentConditionCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM payment_conditions;

-- name: AddInstallment :one
INSERT INTO payment_condition_installments (
    payment_condition_id, installment_number, due_days, description,
    document_type, movement_type, carrier_id
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListInstallments :many
SELECT * FROM payment_condition_installments
WHERE payment_condition_id = $1 AND is_active = TRUE
ORDER BY installment_number;

-- name: DeleteInstallment :exec
UPDATE payment_condition_installments SET is_active = FALSE WHERE id = $1;

-- ─── Sales Tables ─────────────────────────────────────────────────────────────

-- name: CreateSalesTable :one
INSERT INTO sales_tables (
    code, description, validity_start, validity_end,
    tolerance_min_pct, tolerance_max_pct, price_formation, decimal_places,
    composition, table_type, base_date,
    allow_items_below_cent, icms_interestadual_por_dentro, observation
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8,
    $9, $10, $11,
    $12, $13, $14
) RETURNING *;

-- name: UpdateSalesTable :one
UPDATE sales_tables
SET description = $2, validity_start = $3, validity_end = $4,
    tolerance_min_pct = $5, tolerance_max_pct = $6, price_formation = $7,
    decimal_places = $8, is_active = $9,
    composition = $10, table_type = $11, base_date = $12,
    allow_items_below_cent = $13, icms_interestadual_por_dentro = $14,
    observation = $15
WHERE id = $1
RETURNING *;

-- name: GetSalesTableByCode :one
SELECT * FROM sales_tables WHERE code = $1;

-- name: ListSalesTables :many
SELECT * FROM sales_tables
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: NextSalesTableCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM sales_tables;

-- ─── Invoice Types ────────────────────────────────────────────────────────────

-- name: CreateInvoiceType :one
INSERT INTO invoice_types (
    code, description, type, stock_movement, icms_type,
    icms_pct, icms_reduction_pct, ipi_pct, pis_pct, cofins_pct,
    issqn_pct, ir_pct, csll_pct, inss_pct,
    generates_revenue, updates_inventory, generates_financial_title,
    considers_goals, calc_substitution_tax, calc_icms_deferral,
    calc_pis_cofins, calc_difal, requires_sales_order, lists_fiscal_books,
    model_nf, cst_icms, csosn_icms, cst_ipi, cst_pis, cst_cofins,
    baixa_pedido, gera_titulo_dev, exige_suframa,
    ir_pct_presumption, csll_pct_presumption,
    description_nf, impostos_nfe, cfop_id,
    dispositivo_legal_ipi_id, dispositivo_legal_icms_id, dispositivo_legal_icms_st_id,
    dispositivo_legal_pis_id, dispositivo_legal_cofins_id,
    hierarchy_ipi, hierarchy_icms, hierarchy_icms_st, hierarchy_pis, hierarchy_cofins,
    ipi_transfer_sales_table_id,
    lista_valor_contabil, lista_registro_saida, lista_icms_ipi, sintegra_sped_fiscal,
    calc_fomentar, excecao_fomentar, comp_ress_ret_st, calc_reducao, complemento_itens,
    busca_tipo_nf, icms_st_ult_entrada, somente_consulta_lotes, calc_imp_ibpt,
    cred_presumido_icms, ciap, vlr_agregado_base_subst, contrato_facon,
    desc_icms_licitacoes, sisdeclara,
    cod_clas_trib, cod_clas_trib_trib_reg, cod_motivo_rest_comp_icms_st, cod_beneficio_fiscal
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10,
    $11, $12, $13, $14,
    $15, $16, $17,
    $18, $19, $20,
    $21, $22, $23, $24,
    $25, $26, $27, $28, $29, $30,
    $31, $32, $33,
    $34, $35,
    $36, $37, $38,
    $39, $40, $41,
    $42, $43,
    $44, $45, $46, $47, $48,
    $49,
    $50, $51, $52, $53,
    $54, $55, $56, $57, $58,
    $59, $60, $61, $62,
    $63, $64, $65, $66,
    $67, $68,
    $69, $70, $71, $72
) RETURNING *;

-- name: UpdateInvoiceType :one
UPDATE invoice_types
SET description = $2, type = $3, stock_movement = $4, icms_type = $5,
    icms_pct = $6, icms_reduction_pct = $7, ipi_pct = $8, pis_pct = $9,
    cofins_pct = $10, issqn_pct = $11, ir_pct = $12, csll_pct = $13, inss_pct = $14,
    generates_revenue = $15, updates_inventory = $16, generates_financial_title = $17,
    considers_goals = $18, calc_substitution_tax = $19, calc_icms_deferral = $20,
    calc_pis_cofins = $21, calc_difal = $22, requires_sales_order = $23,
    lists_fiscal_books = $24, is_active = $25,
    model_nf = $26, cst_icms = $27, csosn_icms = $28, cst_ipi = $29,
    cst_pis = $30, cst_cofins = $31,
    baixa_pedido = $32, gera_titulo_dev = $33, exige_suframa = $34,
    ir_pct_presumption = $35, csll_pct_presumption = $36,
    description_nf = $37, impostos_nfe = $38, cfop_id = $39,
    dispositivo_legal_ipi_id = $40, dispositivo_legal_icms_id = $41,
    dispositivo_legal_icms_st_id = $42, dispositivo_legal_pis_id = $43,
    dispositivo_legal_cofins_id = $44,
    hierarchy_ipi = $45, hierarchy_icms = $46, hierarchy_icms_st = $47,
    hierarchy_pis = $48, hierarchy_cofins = $49,
    ipi_transfer_sales_table_id = $50,
    lista_valor_contabil = $51, lista_registro_saida = $52, lista_icms_ipi = $53,
    sintegra_sped_fiscal = $54,
    calc_fomentar = $55, excecao_fomentar = $56, comp_ress_ret_st = $57,
    calc_reducao = $58, complemento_itens = $59,
    busca_tipo_nf = $60, icms_st_ult_entrada = $61, somente_consulta_lotes = $62,
    calc_imp_ibpt = $63, cred_presumido_icms = $64, ciap = $65,
    vlr_agregado_base_subst = $66, contrato_facon = $67,
    desc_icms_licitacoes = $68, sisdeclara = $69,
    cod_clas_trib = $70, cod_clas_trib_trib_reg = $71,
    cod_motivo_rest_comp_icms_st = $72, cod_beneficio_fiscal = $73
WHERE id = $1
RETURNING *;

-- name: GetInvoiceTypeByCode :one
SELECT * FROM invoice_types WHERE code = $1;

-- name: ListInvoiceTypes :many
SELECT * FROM invoice_types
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: NextInvoiceTypeCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM invoice_types;

-- ─── Tax Types ────────────────────────────────────────────────────────────────

-- name: CreateTaxType :one
INSERT INTO tax_types (
    code, description,
    ipi_base_total_items, ipi_base_subtract_discount, ipi_base_add_freight, ipi_base_add_expenses,
    icms_base_total_items, icms_base_subtract_discount, icms_base_add_freight, icms_base_add_ipi, icms_base_add_expenses,
    pis_cofins_base_total_items, pis_cofins_base_subtract_discount, pis_cofins_base_add_freight,
    pis_cofins_base_add_insurance, pis_cofins_base_add_expenses,
    csll_base_total_items, csll_base_subtract_discount, csll_base_add_freight,
    ir_base_total_items, ir_base_subtract_discount, ir_base_add_freight,
    is_consumer
) VALUES (
    $1, $2,
    $3, $4, $5, $6,
    $7, $8, $9, $10, $11,
    $12, $13, $14,
    $15, $16,
    $17, $18, $19,
    $20, $21, $22,
    $23
) RETURNING *;

-- name: UpdateTaxType :one
UPDATE tax_types
SET description = $2,
    ipi_base_total_items = $3, ipi_base_subtract_discount = $4, ipi_base_add_freight = $5, ipi_base_add_expenses = $6,
    icms_base_total_items = $7, icms_base_subtract_discount = $8, icms_base_add_freight = $9, icms_base_add_ipi = $10, icms_base_add_expenses = $11,
    pis_cofins_base_total_items = $12, pis_cofins_base_subtract_discount = $13, pis_cofins_base_add_freight = $14,
    pis_cofins_base_add_insurance = $15, pis_cofins_base_add_expenses = $16,
    csll_base_total_items = $17, csll_base_subtract_discount = $18, csll_base_add_freight = $19,
    ir_base_total_items = $20, ir_base_subtract_discount = $21, ir_base_add_freight = $22,
    is_consumer = $23, is_active = $24
WHERE id = $1
RETURNING *;

-- name: GetTaxTypeByCode :one
SELECT * FROM tax_types WHERE code = $1;

-- name: ListTaxTypes :many
SELECT * FROM tax_types
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: NextTaxTypeCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM tax_types;

-- ─── Customers ────────────────────────────────────────────────────────────────

-- name: CreateCustomer :one
INSERT INTO customers (
    code, corporate_code, is_corporate, name, trade_name,
    document_type, document_number, state_registration, municipal_registration,
    suframa_code, suframa_expiry,
    region_id, market_segment_id, customer_type_id,
    payment_condition_id, sales_table_id, carrier_id, carrier_group_id,
    invoice_type_id, tax_type_id, payment_cond_visibility,
    credit_limit, website, created_by
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9,
    $10, $11,
    $12, $13, $14,
    $15, $16, $17, $18,
    $19, $20, $21,
    $22, $23, $24
) RETURNING *;

-- name: UpdateCustomer :one
UPDATE customers
SET name = $2, trade_name = $3, state_registration = $4, municipal_registration = $5,
    suframa_code = $6, suframa_expiry = $7,
    region_id = $8, market_segment_id = $9, customer_type_id = $10,
    payment_condition_id = $11, sales_table_id = $12, carrier_id = $13,
    carrier_group_id = $14, invoice_type_id = $15, tax_type_id = $16,
    payment_cond_visibility = $17, credit_limit = $18, website = $19,
    is_active = $20, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetCustomerByCode :one
SELECT * FROM customers WHERE code = $1;

-- name: GetCustomerByDocument :one
SELECT * FROM customers WHERE document_number = $1 LIMIT 1;

-- name: ListCustomers :many
SELECT * FROM customers
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: ListEstablishments :many
SELECT * FROM customers
WHERE corporate_code = $1 AND is_active = TRUE
ORDER BY code;

-- name: BlockCustomer :exec
UPDATE customers
SET blocked = TRUE, block_reason = $2, updated_at = NOW()
WHERE code = $1;

-- name: UnblockCustomer :exec
UPDATE customers
SET blocked = FALSE, block_reason = NULL, updated_at = NOW()
WHERE code = $1;

-- name: NextCustomerCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM customers;

-- ─── Customer Addresses ───────────────────────────────────────────────────────

-- name: AddAddress :one
INSERT INTO customer_addresses (
    customer_id, address_type, zip_code, street, number,
    complement, neighborhood, city, uf, country, is_default
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: UpdateAddress :one
UPDATE customer_addresses
SET address_type = $2, zip_code = $3, street = $4, number = $5,
    complement = $6, neighborhood = $7, city = $8, uf = $9,
    country = $10, is_default = $11
WHERE id = $1
RETURNING *;

-- name: ListAddresses :many
SELECT * FROM customer_addresses WHERE customer_id = $1 ORDER BY address_type, is_default DESC;

-- name: DeleteAddress :exec
DELETE FROM customer_addresses WHERE id = $1;

-- ─── Customer Contacts ────────────────────────────────────────────────────────

-- name: AddContact :one
INSERT INTO customer_contacts (
    customer_id, contact_type_id, name, email, phone, mobile, position, is_primary
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateContact :one
UPDATE customer_contacts
SET contact_type_id = $2, name = $3, email = $4, phone = $5,
    mobile = $6, position = $7, is_primary = $8, is_active = $9
WHERE id = $1
RETURNING *;

-- name: ListContacts :many
SELECT * FROM customer_contacts
WHERE customer_id = $1 AND is_active = TRUE
ORDER BY is_primary DESC, name;

-- name: DeleteContact :exec
UPDATE customer_contacts SET is_active = FALSE WHERE id = $1;
