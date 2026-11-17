-- ─── Regions ─────────────────────────────────────────────────────────────────

-- name: CreateRegion :one
INSERT INTO regions (enterprise_id, code, description, uf, city, created_by)
VALUES ($6, $1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateRegion :one
UPDATE regions
SET description = $2, uf = $3, city = $4, is_active = $5
WHERE id = $1 AND enterprise_id = $6
RETURNING *;

-- name: GetRegionByCode :one
SELECT * FROM regions WHERE code = $1 AND enterprise_id = $2;

-- name: ListRegions :many
SELECT * FROM regions
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE) AND enterprise_id = $2
ORDER BY code;

-- name: NextRegionCode :one
SELECT (COALESCE(MAX(code), 0) + 1)::bigint AS next_code FROM regions WHERE enterprise_id = $1;

-- ─── Market Segments ──────────────────────────────────────────────────────────

-- name: CreateMarketSegment :one
INSERT INTO market_segments (enterprise_id, code, description, parent_id, has_pis_cofins_retention, retention_indicator)
VALUES ($6, $1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateMarketSegment :one
UPDATE market_segments
SET description = $2, parent_id = $3, has_pis_cofins_retention = $4,
    retention_indicator = $5, is_active = $6
WHERE id = $1 AND enterprise_id = $7
RETURNING *;

-- name: GetMarketSegmentByCode :one
SELECT * FROM market_segments WHERE code = $1 AND enterprise_id = $2;

-- name: ListMarketSegments :many
SELECT * FROM market_segments
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE) AND enterprise_id = $2
ORDER BY code;

-- name: NextMarketSegmentCode :one
SELECT (COALESCE(MAX(code), 0) + 1)::bigint AS next_code FROM market_segments WHERE enterprise_id = $1;

-- ─── Customer Contact Types ───────────────────────────────────────────────────

-- name: CreateContactType :one
INSERT INTO customer_contact_types (enterprise_id, code, description)
VALUES ($3, $1, $2)
RETURNING *;

-- name: UpdateContactType :one
UPDATE customer_contact_types
SET description = $2, is_active = $3
WHERE id = $1 AND enterprise_id = $4
RETURNING *;

-- name: GetContactTypeByCode :one
SELECT * FROM customer_contact_types WHERE code = $1 AND enterprise_id = $2;

-- name: ListContactTypes :many
SELECT * FROM customer_contact_types
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE) AND enterprise_id = $2
ORDER BY code;

-- name: NextContactTypeCode :one
SELECT (COALESCE(MAX(code), 0) + 1)::bigint AS next_code FROM customer_contact_types WHERE enterprise_id = $1;

-- ─── Customer Types ───────────────────────────────────────────────────────────

-- name: CreateCustomerType :one
INSERT INTO customer_types (enterprise_id, code, description, category, delivery_days)
VALUES ($5, $1, $2, $3, $4)
RETURNING *;

-- name: UpdateCustomerType :one
UPDATE customer_types
SET description = $2, category = $3, delivery_days = $4, is_active = $5
WHERE id = $1 AND enterprise_id = $6
RETURNING *;

-- name: GetCustomerTypeByCode :one
SELECT * FROM customer_types WHERE code = $1 AND enterprise_id = $2;

-- name: ListCustomerTypes :many
SELECT * FROM customer_types
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE) AND enterprise_id = $2
ORDER BY code;

-- ─── Carriers ─────────────────────────────────────────────────────────────────

-- name: CreateCarrier :one
INSERT INTO carriers (enterprise_id, code, description)
VALUES ($3, $1, $2)
RETURNING *;

-- name: UpdateCarrier :one
UPDATE carriers
SET description = $2, is_active = $3
WHERE id = $1 AND enterprise_id = $4
RETURNING *;

-- name: GetCarrierByCode :one
SELECT * FROM carriers WHERE code = $1 AND enterprise_id = $2;

-- name: GetCarrierByID :one
SELECT * FROM carriers WHERE id = $1 AND enterprise_id = $2;

-- name: ListCarriers :many
SELECT * FROM carriers
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE) AND enterprise_id = $2
ORDER BY code;

-- name: NextCarrierCode :one
SELECT (COALESCE(MAX(code), 0) + 1)::bigint AS next_code FROM carriers WHERE enterprise_id = $1;

-- ─── Carrier Groups ───────────────────────────────────────────────────────────

-- name: CreateCarrierGroup :one
INSERT INTO carrier_groups (enterprise_id, code, description)
VALUES ($3, $1, $2)
RETURNING *;

-- name: GetCarrierGroupByCode :one
SELECT * FROM carrier_groups WHERE code = $1 AND enterprise_id = $2;

-- name: UpdateCarrierGroup :one
UPDATE carrier_groups SET description=$2 WHERE id=$1 AND enterprise_id = $3 RETURNING *;

-- name: ListCarrierGroups :many
SELECT * FROM carrier_groups WHERE enterprise_id = $1 ORDER BY code;

-- name: AddCarrierToGroup :execrows
-- Grupo e portador têm de ser da empresa.
INSERT INTO carrier_group_carriers (carrier_group_id, carrier_id)
SELECT $1, $2
 WHERE EXISTS (SELECT 1 FROM carrier_groups g WHERE g.id = $1 AND g.enterprise_id = $3)
   AND EXISTS (SELECT 1 FROM carriers c WHERE c.id = $2 AND c.enterprise_id = $3)
ON CONFLICT DO NOTHING;

-- name: RemoveCarrierFromGroup :exec
DELETE FROM carrier_group_carriers
WHERE carrier_group_id = $1 AND carrier_id = $2
  AND EXISTS (SELECT 1 FROM carrier_groups g WHERE g.id = $1 AND g.enterprise_id = $3);

-- name: NextCarrierGroupCode :one
SELECT (COALESCE(MAX(code), 0) + 1)::bigint AS next_code FROM carrier_groups WHERE enterprise_id = $1;

-- ─── Payment Conditions ───────────────────────────────────────────────────────

-- name: CreatePaymentCondition :one
-- A empresa entra na gravacao: a coluna e NOT NULL desde o isolamento do
-- financeiro, e sem ela o cadastro de condicao de pagamento parava de gravar.
INSERT INTO payment_conditions (
    enterprise_id, code, description, carrier_id, analysis_type, parcel_start,
    expenses, average_term, is_special, is_revenue, is_at_sight
) VALUES (sqlc.arg(enterprise_id), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: UpdatePaymentCondition :one
UPDATE payment_conditions
SET description = $2, carrier_id = $3, analysis_type = $4, parcel_start = $5,
    expenses = $6, average_term = $7, is_special = $8, is_revenue = $9,
    is_at_sight = $10, is_active = $11
WHERE id = $1 AND enterprise_id = sqlc.arg(enterprise_id)
RETURNING *;

-- name: GetPaymentConditionByCode :one
SELECT * FROM payment_conditions WHERE code = $1 AND enterprise_id = sqlc.arg(enterprise_id);

-- name: GetPaymentConditionByID :one
SELECT * FROM payment_conditions WHERE id = $1 AND enterprise_id = sqlc.arg(enterprise_id);

-- name: ListPaymentConditions :many
SELECT * FROM payment_conditions
WHERE enterprise_id = sqlc.arg(enterprise_id)
  AND (sqlc.arg(only_active)::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: NextPaymentConditionCode :one
SELECT (COALESCE(MAX(code), 0) + 1)::bigint AS next_code FROM payment_conditions;

-- name: AddInstallment :one
-- Upsert pela parcela: reenviar a mesma parcela ALTERA, não duplica. Antes, o
-- segundo POST batia na chave única e devolvia erro técnico, e a tela não tinha
-- como corrigir um percentual digitado errado.
INSERT INTO payment_condition_installments (
    payment_condition_id, installment_number, due_days, description,
    document_type, movement_type, carrier_id, percentage, base_event
) VALUES ($1, $2, $3, $4, $5, $6, $7, sqlc.narg(percentage), sqlc.arg(base_event))
ON CONFLICT (payment_condition_id, installment_number) DO UPDATE SET
    due_days = EXCLUDED.due_days,
    description = EXCLUDED.description,
    document_type = EXCLUDED.document_type,
    movement_type = EXCLUDED.movement_type,
    carrier_id = EXCLUDED.carrier_id,
    percentage = EXCLUDED.percentage,
    base_event = EXCLUDED.base_event,
    is_active = TRUE
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
    enterprise_id, code, description, validity_start, validity_end,
    tolerance_min_pct, tolerance_max_pct, price_formation, decimal_places,
    composition, table_type, base_date,
    allow_items_below_cent, icms_interestadual_por_dentro, observation
) VALUES (
    sqlc.arg(enterprise_id), sqlc.arg(code), sqlc.arg(description), sqlc.arg(validity_start), sqlc.arg(validity_end),
    sqlc.arg(tolerance_min_pct), sqlc.arg(tolerance_max_pct), sqlc.arg(price_formation), sqlc.arg(decimal_places),
    sqlc.arg(composition), sqlc.arg(table_type), sqlc.arg(base_date),
    sqlc.arg(allow_items_below_cent), sqlc.arg(icms_interestadual_por_dentro), sqlc.arg(observation)
) RETURNING *;

-- name: UpdateSalesTable :one
UPDATE sales_tables
SET description = $2, validity_start = $3, validity_end = $4,
    tolerance_min_pct = $5, tolerance_max_pct = $6, price_formation = $7,
    decimal_places = $8, is_active = $9,
    composition = $10, table_type = $11, base_date = $12,
    allow_items_below_cent = $13, icms_interestadual_por_dentro = $14,
    observation = $15
WHERE id = $1 AND enterprise_id = sqlc.arg(enterprise_id)
RETURNING *;

-- name: GetSalesTableByCode :one
SELECT * FROM sales_tables WHERE code = $1 AND enterprise_id = sqlc.arg(enterprise_id);

-- name: ListSalesTables :many
SELECT * FROM sales_tables
WHERE enterprise_id=sqlc.arg(enterprise_id) AND (sqlc.arg(only_active)::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: NextSalesTableCode :one
-- Enquanto sales_orders.price_table_code mantiver a FK legada para code, o
-- proximo codigo precisa permanecer globalmente unico.
SELECT (COALESCE(MAX(code), 0) + 1)::bigint AS next_code FROM sales_tables;

-- ─── Invoice Types ────────────────────────────────────────────────────────────

-- name: CreateInvoiceType :one
INSERT INTO invoice_types (enterprise_id, 
    code, description, type, stock_movement, icms_type, icms_reduction_pct,
    generates_revenue, updates_inventory, generates_financial_title, calc_substitution_tax, calc_icms_deferral,
    calc_pis_cofins, calc_difal, requires_sales_order,
    baixa_pedido, exige_suframa) VALUES (
    $17, $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11,
    $12, $13, $14,
    $15, $16) RETURNING *;

-- name: UpdateInvoiceType :one
UPDATE invoice_types
SET description = $2, type = $3, stock_movement = $4, icms_type = $5, icms_reduction_pct = $6,
    generates_revenue = $7, updates_inventory = $8, generates_financial_title = $9, calc_substitution_tax = $10, calc_icms_deferral = $11,
    calc_pis_cofins = $12, calc_difal = $13, requires_sales_order = $14, is_active = $15,
    baixa_pedido = $16, exige_suframa = $17
WHERE id = $1 AND enterprise_id = $18
RETURNING *;

-- name: GetInvoiceTypeByCode :one
SELECT * FROM invoice_types WHERE code = $1 AND enterprise_id = $2;

-- name: ListInvoiceTypes :many
SELECT * FROM invoice_types
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE) AND enterprise_id = $2
ORDER BY code;

-- name: NextInvoiceTypeCode :one
SELECT (COALESCE(MAX(code), 0) + 1)::bigint AS next_code FROM invoice_types WHERE enterprise_id = $1;

-- ─── Tax Types ────────────────────────────────────────────────────────────────

-- name: CreateTaxType :one
INSERT INTO tax_types (enterprise_id, 
    code, description,
    is_consumer
) VALUES (
    $4, $1, $2,
    $3
) RETURNING *;

-- name: UpdateTaxType :one
UPDATE tax_types
SET description = $2,
    is_consumer = $3, is_active = $4
WHERE id = $1 AND enterprise_id = $5
RETURNING *;

-- name: GetTaxTypeByCode :one
SELECT * FROM tax_types WHERE code = $1 AND enterprise_id = $2;

-- name: ListTaxTypes :many
SELECT * FROM tax_types
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE) AND enterprise_id = $2
ORDER BY code;

-- name: NextTaxTypeCode :one
SELECT (COALESCE(MAX(code), 0) + 1)::bigint AS next_code FROM tax_types WHERE enterprise_id = $1;

-- ─── Customers ────────────────────────────────────────────────────────────────

-- name: CreateCustomer :one
-- O cliente pertence à empresa que o cadastrou (convenção enterprise_id).
INSERT INTO customers (
    code, corporate_code, is_corporate, name, trade_name,
    document_type, document_number, state_registration, municipal_registration,
    suframa_code, suframa_expiry,
    region_id, market_segment_id, customer_type_id, carrier_id, tax_type_id,
    credit_limit, website, created_by, enterprise_id, finalidade_compra,
    ie_indicator, tax_regime, main_cnae, nfe_email, billing_email,
    representative_code, shipping_carrier_id, freight_type,
    requires_customer_po, requires_quality_cert, allows_partial_billing,
    receiving_notes, commercial_notes, financial_notes
) VALUES (
    sqlc.arg(code), sqlc.narg(corporate_code), sqlc.arg(is_corporate), sqlc.arg(name), sqlc.narg(trade_name),
    sqlc.arg(document_type), sqlc.arg(document_number), sqlc.narg(state_registration), sqlc.narg(municipal_registration),
    sqlc.narg(suframa_code), sqlc.narg(suframa_expiry),
    sqlc.narg(region_id), sqlc.narg(market_segment_id), sqlc.narg(customer_type_id), sqlc.narg(carrier_id), sqlc.narg(tax_type_id),
    sqlc.arg(credit_limit), sqlc.narg(website), sqlc.arg(created_by), sqlc.arg(enterprise_id), sqlc.narg(finalidade_compra)::varchar,
    sqlc.narg(ie_indicator)::varchar, sqlc.narg(tax_regime)::varchar, sqlc.narg(main_cnae)::varchar, sqlc.narg(nfe_email)::varchar, sqlc.narg(billing_email)::varchar,
    sqlc.narg(representative_code)::bigint, sqlc.narg(shipping_carrier_id)::bigint, sqlc.narg(freight_type)::varchar,
    sqlc.arg(requires_customer_po), sqlc.arg(requires_quality_cert), sqlc.arg(allows_partial_billing),
    sqlc.narg(receiving_notes)::text, sqlc.narg(commercial_notes)::text, sqlc.narg(financial_notes)::text
) RETURNING *;

-- name: UpdateCustomer :one
-- O limite de crédito não passa por aqui: só muda por SetCustomerCredit, que
-- grava a trilha da análise.
UPDATE customers
SET name = sqlc.arg(name), trade_name = sqlc.narg(trade_name),
    state_registration = sqlc.narg(state_registration), municipal_registration = sqlc.narg(municipal_registration),
    suframa_code = sqlc.narg(suframa_code), suframa_expiry = sqlc.narg(suframa_expiry),
    corporate_code = sqlc.narg(corporate_code), is_corporate = sqlc.arg(is_corporate),
    region_id = sqlc.narg(region_id), market_segment_id = sqlc.narg(market_segment_id),
    customer_type_id = sqlc.narg(customer_type_id), carrier_id = sqlc.narg(carrier_id), tax_type_id = sqlc.narg(tax_type_id),
    website = sqlc.narg(website), finalidade_compra = sqlc.narg(finalidade_compra)::varchar,
    ie_indicator = sqlc.narg(ie_indicator)::varchar, tax_regime = sqlc.narg(tax_regime)::varchar,
    main_cnae = sqlc.narg(main_cnae)::varchar, nfe_email = sqlc.narg(nfe_email)::varchar,
    billing_email = sqlc.narg(billing_email)::varchar,
    representative_code = sqlc.narg(representative_code)::bigint, shipping_carrier_id = sqlc.narg(shipping_carrier_id)::bigint,
    freight_type = sqlc.narg(freight_type)::varchar,
    requires_customer_po = sqlc.arg(requires_customer_po), requires_quality_cert = sqlc.arg(requires_quality_cert),
    allows_partial_billing = sqlc.arg(allows_partial_billing),
    receiving_notes = sqlc.narg(receiving_notes)::text, commercial_notes = sqlc.narg(commercial_notes)::text,
    financial_notes = sqlc.narg(financial_notes)::text,
    updated_at = NOW()
WHERE id = sqlc.arg(id) AND enterprise_id = sqlc.arg(enterprise_id)
RETURNING *;

-- name: SetCustomerCredit :execrows
UPDATE customers
SET credit_limit = sqlc.arg(credit_limit), credit_limit_suggested = sqlc.arg(credit_limit_suggested),
    credit_score = sqlc.arg(credit_score), credit_risk_class = sqlc.arg(credit_risk_class),
    credit_limit_expires_at = sqlc.arg(credit_limit_expires_at), credit_reviewed_at = NOW(), updated_at = NOW()
WHERE code = sqlc.arg(code) AND enterprise_id = sqlc.arg(enterprise_id);

-- name: SetCustomerRegistration :execrows
-- Situação cadastral lida na Receita. Os campos que a consulta não trouxe
-- ficam como estavam.
UPDATE customers
SET registration_status = sqlc.arg(registration_status), registration_checked_at = NOW(),
    opening_date = COALESCE(sqlc.narg(opening_date), opening_date),
    share_capital = COALESCE(sqlc.narg(share_capital), share_capital),
    company_size = COALESCE(NULLIF(sqlc.arg(company_size)::varchar, ''), company_size),
    legal_nature = COALESCE(NULLIF(sqlc.arg(legal_nature)::varchar, ''), legal_nature),
    main_cnae = COALESCE(main_cnae, NULLIF(sqlc.arg(main_cnae)::varchar, '')),
    updated_at = NOW()
WHERE code = sqlc.arg(code) AND enterprise_id = sqlc.arg(enterprise_id);

-- name: SetCustomerActive :execrows
UPDATE customers SET is_active = sqlc.arg(is_active), updated_at = NOW()
WHERE code = sqlc.arg(code) AND enterprise_id = sqlc.arg(enterprise_id);

-- name: DeleteCustomer :execrows
DELETE FROM customers WHERE code = sqlc.arg(code) AND enterprise_id = sqlc.arg(enterprise_id);

-- name: ListCustomersForRegistrationCheck :many
-- Clientes CNPJ ativos cuja situação na Receita não é conferida há `days` dias,
-- de todas as empresas (o agendador roda fora de requisição).
SELECT c.enterprise_id, e.code AS enterprise_code, c.code, c.document_number, c.blocked, c.created_by
  FROM customers c
  JOIN enterprise e ON e.id = c.enterprise_id
 WHERE c.document_type = 'CNPJ' AND c.is_active
   AND (c.registration_checked_at IS NULL
        OR c.registration_checked_at < NOW() - make_interval(days => COALESCE(
             (SELECT p.registration_check_days FROM customer_credit_policies p WHERE p.enterprise_id = c.enterprise_id), 30)))
   -- Quem falhou há menos de um dia espera a vez dos outros.
   AND (c.registration_attempt_at IS NULL OR c.registration_attempt_at < NOW() - interval '1 day')
 ORDER BY c.registration_attempt_at NULLS FIRST, c.registration_checked_at NULLS FIRST
 LIMIT sqlc.arg(max_rows);

-- name: MarkRegistrationAttempt :execrows
UPDATE customers SET registration_attempt_at = NOW()
WHERE code = sqlc.arg(code) AND enterprise_id = sqlc.arg(enterprise_id);

-- name: GetCreditPolicy :one
SELECT * FROM customer_credit_policies WHERE enterprise_id = $1;

-- name: UpsertCreditPolicy :one
INSERT INTO customer_credit_policies (
    enterprise_id, max_limit, new_customer_capital_pct, new_customer_cap, override_tolerance_pct,
    overdue_block_days, history_months, validity_months_a, validity_months_b, validity_months_c,
    validity_months_d, validity_months_e, registration_check_days, updated_by, updated_at, new_customer_revenue_pct,
    customer_revenue_pct, declared_monthly_revenue, auto_review_months, auto_increase_max_pct
) VALUES (
    sqlc.arg(enterprise_id), sqlc.arg(max_limit), sqlc.arg(new_customer_capital_pct), sqlc.arg(new_customer_cap), sqlc.arg(override_tolerance_pct),
    sqlc.arg(overdue_block_days), sqlc.arg(history_months), sqlc.arg(validity_months_a), sqlc.arg(validity_months_b), sqlc.arg(validity_months_c),
    sqlc.arg(validity_months_d), sqlc.arg(validity_months_e), sqlc.arg(registration_check_days), sqlc.arg(updated_by), NOW(), sqlc.arg(new_customer_revenue_pct),
    sqlc.arg(customer_revenue_pct), sqlc.arg(declared_monthly_revenue), sqlc.arg(auto_review_months), sqlc.arg(auto_increase_max_pct)
)
ON CONFLICT (enterprise_id) DO UPDATE SET
    max_limit = EXCLUDED.max_limit, new_customer_capital_pct = EXCLUDED.new_customer_capital_pct,
    new_customer_cap = EXCLUDED.new_customer_cap, override_tolerance_pct = EXCLUDED.override_tolerance_pct,
    overdue_block_days = EXCLUDED.overdue_block_days, history_months = EXCLUDED.history_months,
    validity_months_a = EXCLUDED.validity_months_a, validity_months_b = EXCLUDED.validity_months_b,
    validity_months_c = EXCLUDED.validity_months_c, validity_months_d = EXCLUDED.validity_months_d,
    validity_months_e = EXCLUDED.validity_months_e, registration_check_days = EXCLUDED.registration_check_days,
    updated_by = EXCLUDED.updated_by, updated_at = NOW(), new_customer_revenue_pct = EXCLUDED.new_customer_revenue_pct,
    customer_revenue_pct = EXCLUDED.customer_revenue_pct, declared_monthly_revenue = EXCLUDED.declared_monthly_revenue,
    auto_review_months = EXCLUDED.auto_review_months, auto_increase_max_pct = EXCLUDED.auto_increase_max_pct
RETURNING *;

-- name: CreateCreditReview :one
INSERT INTO customer_credit_reviews (
    enterprise_id, customer_code, origin, suggested_limit, approved_limit, previous_limit,
    score, risk_class, expires_at, deviates, reason, factors, created_by
) VALUES (
    sqlc.arg(enterprise_id), sqlc.arg(customer_code), sqlc.arg(origin), sqlc.arg(suggested_limit), sqlc.arg(approved_limit), sqlc.narg(previous_limit),
    sqlc.arg(score), sqlc.arg(risk_class), sqlc.arg(expires_at), sqlc.arg(deviates), sqlc.narg(reason), sqlc.arg(factors), sqlc.arg(created_by)
) RETURNING *;

-- name: ListCreditReviews :many
-- Só as análises do cadastro atual: a trilha de um cliente excluído fica
-- guardada, mas não aparece para outro cliente criado depois com o mesmo código.
SELECT r.*, COALESCE(u.name, '')::text AS created_by_name FROM customer_credit_reviews r
  JOIN customers c ON c.code = r.customer_code AND c.enterprise_id = r.enterprise_id
  LEFT JOIN users u ON u.id = r.created_by
 WHERE r.enterprise_id = sqlc.arg(enterprise_id) AND r.customer_code = sqlc.arg(customer_code)
   AND r.created_at >= c.created_at
 ORDER BY r.created_at DESC, r.id DESC;

-- name: GetCustomerByCode :one
SELECT * FROM customers WHERE code = $1 AND enterprise_id = $2;

-- name: GetCustomerByDocument :one
-- Compara só os dígitos: cadastro antigo pode ter o documento com pontuação.
SELECT * FROM customers
WHERE regexp_replace(document_number, '\D', '', 'g') = regexp_replace(sqlc.arg(document_number)::TEXT, '\D', '', 'g')
  AND enterprise_id = sqlc.arg(enterprise_id)
ORDER BY code LIMIT 1;

-- name: ListCustomers :many
SELECT * FROM customers
WHERE enterprise_id = sqlc.arg(enterprise_id)
  AND (sqlc.arg(only_active)::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: ListEstablishments :many
SELECT * FROM customers
WHERE corporate_code = $1 AND enterprise_id = $2 AND is_active = TRUE
ORDER BY code;

-- name: BlockCustomer :execrows
UPDATE customers
SET blocked = TRUE, block_reason = $2, updated_at = NOW()
WHERE code = $1 AND enterprise_id = $3;

-- name: UnblockCustomer :execrows
UPDATE customers
SET blocked = FALSE, block_reason = NULL, updated_at = NOW()
WHERE code = $1 AND enterprise_id = $2;

-- name: NextCustomerCode :one
-- O código continua crescendo no banco inteiro (não por empresa): um código
-- novo nunca coincide com o de outra empresa, o que evita confusão em relatório
-- e integração. A unicidade garantida pelo banco é (empresa, código).
SELECT (COALESCE(MAX(code), 0) + 1)::bigint AS next_code FROM customers;

-- ─── Customer Addresses ───────────────────────────────────────────────────────

-- name: AddAddress :one
-- Só grava se o cliente for da empresa autenticada (sem linha = não achou).
INSERT INTO customer_addresses (
    customer_id, address_type, zip_code, street, number,
    complement, neighborhood, city, uf, country, is_default
)
SELECT c.id, sqlc.arg(address_type), sqlc.arg(zip_code), sqlc.arg(street), sqlc.arg(number),
       sqlc.arg(complement), sqlc.arg(neighborhood), sqlc.arg(city), sqlc.arg(uf), sqlc.arg(country), sqlc.arg(is_default)
  FROM customers c
 WHERE c.id = sqlc.arg(customer_id) AND c.enterprise_id = sqlc.arg(enterprise_id)
RETURNING *;

-- name: UpdateAddress :one
UPDATE customer_addresses
SET address_type = $2, zip_code = $3, street = $4, number = $5,
    complement = $6, neighborhood = $7, city = $8, uf = $9,
    country = $10, is_default = $11
WHERE customer_addresses.id = $1
  AND EXISTS (SELECT 1 FROM customers c WHERE c.id = customer_addresses.customer_id AND c.enterprise_id = sqlc.arg(enterprise_id))
RETURNING *;

-- name: ListAddresses :many
SELECT a.* FROM customer_addresses a
  JOIN customers c ON c.id = a.customer_id AND c.enterprise_id = $2
 WHERE a.customer_id = $1
 ORDER BY a.address_type, a.is_default DESC;

-- name: DeleteAddress :execrows
DELETE FROM customer_addresses
 WHERE customer_addresses.id = $1
   AND EXISTS (SELECT 1 FROM customers c WHERE c.id = customer_addresses.customer_id AND c.enterprise_id = $2);

-- ─── Customer Contacts ────────────────────────────────────────────────────────

-- name: AddContact :one
INSERT INTO customer_contacts (
    customer_id, contact_type_id, name, email, phone, mobile, position, is_primary
)
SELECT c.id, sqlc.arg(contact_type_id), sqlc.arg(name), sqlc.arg(email), sqlc.arg(phone), sqlc.arg(mobile), sqlc.arg(position), sqlc.arg(is_primary)
  FROM customers c
 WHERE c.id = sqlc.arg(customer_id) AND c.enterprise_id = sqlc.arg(enterprise_id)
RETURNING *;

-- name: UpdateContact :one
UPDATE customer_contacts
SET contact_type_id = $2, name = $3, email = $4, phone = $5,
    mobile = $6, position = $7, is_primary = $8, is_active = $9
WHERE customer_contacts.id = $1
  AND EXISTS (SELECT 1 FROM customers c WHERE c.id = customer_contacts.customer_id AND c.enterprise_id = sqlc.arg(enterprise_id))
RETURNING *;

-- name: ListContacts :many
SELECT ct.* FROM customer_contacts ct
  JOIN customers c ON c.id = ct.customer_id AND c.enterprise_id = $2
 WHERE ct.customer_id = $1 AND ct.is_active = TRUE
 ORDER BY ct.is_primary DESC, ct.name;

-- name: DeleteContact :execrows
UPDATE customer_contacts SET is_active = FALSE
 WHERE customer_contacts.id = $1
   AND EXISTS (SELECT 1 FROM customers c WHERE c.id = customer_contacts.customer_id AND c.enterprise_id = $2);
