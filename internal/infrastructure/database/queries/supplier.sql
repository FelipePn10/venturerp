-- ─── Supplier Types ───────────────────────────────────────────────────────────

-- name: CreateSupplierType :one
INSERT INTO supplier_types (code, description, kind)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateSupplierType :one
UPDATE supplier_types
SET description = $2, kind = $3, is_active = $4
WHERE code = $1
RETURNING *;

-- name: GetSupplierTypeByCode :one
SELECT * FROM supplier_types WHERE code = $1;

-- name: ListSupplierTypes :many
SELECT * FROM supplier_types
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: NextSupplierTypeCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM supplier_types;

-- ─── Supplier Contact Types ─────────────────────────────────────────────────────

-- name: CreateSupplierContactType :one
INSERT INTO supplier_contact_types (code, description)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateSupplierContactType :one
UPDATE supplier_contact_types
SET description = $2, is_active = $3
WHERE code = $1
RETURNING *;

-- name: GetSupplierContactTypeByCode :one
SELECT * FROM supplier_contact_types WHERE code = $1;

-- name: ListSupplierContactTypes :many
SELECT * FROM supplier_contact_types
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: NextSupplierContactTypeCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM supplier_contact_types;

-- ─── Suppliers ────────────────────────────────────────────────────────────────

-- name: CreateSupplier :one
INSERT INTO suppliers (
    code, corporate_code, is_active, is_representative, is_customer,
    name, trade_name, person_type, document_type, document_number,
    state_registration, municipal_registration, supplier_type_id,
    payment_condition_id, carrier_id, region_id, freight_type, register_date,
    viticola_obligation, gln_code, agriculture_ministry_registration,
    icms_contributor, is_mei, tracking_platform, homologated, created_by
)
VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10,
    $11, $12, $13,
    $14, $15, $16, $17, $18,
    $19, $20, $21,
    $22, $23, $24, $25, $26
)
RETURNING *;

-- name: UpdateSupplier :one
UPDATE suppliers
SET corporate_code = $2, is_active = $3, is_representative = $4, is_customer = $5,
    name = $6, trade_name = $7, person_type = $8, document_type = $9, document_number = $10,
    state_registration = $11, municipal_registration = $12, supplier_type_id = $13,
    payment_condition_id = $14, carrier_id = $15, region_id = $16, freight_type = $17,
    viticola_obligation = $18, gln_code = $19, agriculture_ministry_registration = $20,
    icms_contributor = $21, is_mei = $22, tracking_platform = $23, homologated = $24,
    updated_at = NOW()
WHERE code = $1
RETURNING *;

-- name: GetSupplierByCode :one
SELECT * FROM suppliers WHERE code = $1;

-- name: GetSupplierByDocument :one
SELECT * FROM suppliers WHERE document_number = $1 LIMIT 1;

-- name: ListSuppliers :many
SELECT * FROM suppliers
WHERE ($1::BOOLEAN = FALSE OR is_active = TRUE)
ORDER BY code;

-- name: ListSupplierEstablishments :many
SELECT * FROM suppliers WHERE corporate_code = $1 ORDER BY code;

-- name: BlockSupplier :exec
UPDATE suppliers SET blocked = TRUE, block_reason = $2, updated_at = NOW() WHERE code = $1;

-- name: UnblockSupplier :exec
UPDATE suppliers SET blocked = FALSE, block_reason = NULL, updated_at = NOW() WHERE code = $1;

-- name: NextSupplierCode :one
SELECT COALESCE(MAX(code), 0) + 1 AS next_code FROM suppliers;

-- name: PropagateStateRegistration :exec
-- Spec: ao alterar a IE, atualiza outros cadastros com o mesmo CNPJ/CPF.
UPDATE suppliers
SET state_registration = $2, updated_at = NOW()
WHERE document_number = $1 AND code <> $3;

-- name: DeleteSupplier :exec
-- Hard delete. Fails with FK violation when purchase orders reference the
-- supplier (the application surfaces a friendly message); inactivate instead.
DELETE FROM suppliers WHERE code = $1;

-- name: UpdateSupplierSefaz :exec
UPDATE suppliers
SET last_sefaz_query = $2, billing_receipt_status = $3,
    last_sefaz_update = $4, sefaz_update_user = $5, updated_at = NOW()
WHERE code = $1;

-- ─── Supplier Addresses ─────────────────────────────────────────────────────────

-- name: CreateSupplierAddress :one
INSERT INTO supplier_addresses (
    supplier_id, address_type, zip_code, street, number, complement,
    neighborhood, city, uf, country, is_default
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: UpdateSupplierAddress :one
UPDATE supplier_addresses
SET address_type = $2, zip_code = $3, street = $4, number = $5, complement = $6,
    neighborhood = $7, city = $8, uf = $9, country = $10, is_default = $11
WHERE id = $1
RETURNING *;

-- name: ListSupplierAddresses :many
SELECT * FROM supplier_addresses WHERE supplier_id = $1 ORDER BY id;

-- name: DeleteSupplierAddress :exec
DELETE FROM supplier_addresses WHERE id = $1;

-- ─── Supplier Phones ──────────────────────────────────────────────────────────

-- name: CreateSupplierPhone :one
INSERT INTO supplier_phones (supplier_id, number, ranking)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListSupplierPhones :many
SELECT * FROM supplier_phones WHERE supplier_id = $1 ORDER BY ranking, id;

-- name: DeleteSupplierPhone :exec
DELETE FROM supplier_phones WHERE id = $1;

-- ─── Supplier Emails ──────────────────────────────────────────────────────────

-- name: CreateSupplierEmail :one
INSERT INTO supplier_emails (supplier_id, email, ranking)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListSupplierEmails :many
SELECT * FROM supplier_emails WHERE supplier_id = $1 ORDER BY ranking, id;

-- name: DeleteSupplierEmail :exec
DELETE FROM supplier_emails WHERE id = $1;

-- ─── Supplier Due Dates ─────────────────────────────────────────────────────────

-- name: CreateSupplierDueDate :one
INSERT INTO supplier_due_dates (
    supplier_id, description, ranking, base_date, payment_condition_id,
    payment_type, subsequent_month, rounding, receipt_start_time,
    receipt_end_time, avg_unload_minutes
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: ListSupplierDueDates :many
SELECT * FROM supplier_due_dates WHERE supplier_id = $1 ORDER BY ranking, id;

-- name: DeleteSupplierDueDate :exec
DELETE FROM supplier_due_dates WHERE id = $1;

-- ─── Supplier Contacts ──────────────────────────────────────────────────────────

-- name: CreateSupplierContact :one
INSERT INTO supplier_contacts (
    supplier_id, contact_type_id, name, position, department, ranking,
    observation, purchase_order_tag
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListSupplierContacts :many
SELECT * FROM supplier_contacts WHERE supplier_id = $1 ORDER BY ranking, id;

-- name: DeleteSupplierContact :exec
DELETE FROM supplier_contacts WHERE id = $1;

-- name: CreateSupplierContactPhone :one
INSERT INTO supplier_contact_phones (contact_id, value, ranking)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListSupplierContactPhones :many
SELECT * FROM supplier_contact_phones WHERE contact_id = $1 ORDER BY ranking, id;

-- name: CreateSupplierContactEmail :one
INSERT INTO supplier_contact_emails (contact_id, value, ranking)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListSupplierContactEmails :many
SELECT * FROM supplier_contact_emails WHERE contact_id = $1 ORDER BY ranking, id;

-- ─── Supplier Enterprises ─────────────────────────────────────────────────────

-- name: CreateSupplierEnterprise :one
INSERT INTO supplier_enterprises (
    supplier_id, enterprise_code, financial_account, applies_ipi,
    default_invoice_type_id, purchase_price_table_id
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateSupplierEnterprise :one
UPDATE supplier_enterprises
SET financial_account = $2, applies_ipi = $3, default_invoice_type_id = $4,
    purchase_price_table_id = $5, is_active = $6
WHERE id = $1
RETURNING *;

-- name: ListSupplierEnterprises :many
SELECT * FROM supplier_enterprises WHERE supplier_id = $1 ORDER BY enterprise_code;

-- name: DeleteSupplierEnterprise :exec
DELETE FROM supplier_enterprises WHERE id = $1;

-- ─── Supplier Parameters ────────────────────────────────────────────────────────

-- name: GetSupplierParameters :one
SELECT * FROM supplier_parameters WHERE enterprise_code = $1;

-- name: UpsertSupplierParameters :one
INSERT INTO supplier_parameters (
    enterprise_code, default_financial_account, unique_item_code_per_supplier,
    requires_financial_account, purchase_supplier_type_id, copy_obs_to_purchase_order,
    copy_obs_to_entry_invoice, homologation_default, use_stock_uom,
    generic_supplier_code, default_due_base_date
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (enterprise_code) DO UPDATE SET
    default_financial_account = EXCLUDED.default_financial_account,
    unique_item_code_per_supplier = EXCLUDED.unique_item_code_per_supplier,
    requires_financial_account = EXCLUDED.requires_financial_account,
    purchase_supplier_type_id = EXCLUDED.purchase_supplier_type_id,
    copy_obs_to_purchase_order = EXCLUDED.copy_obs_to_purchase_order,
    copy_obs_to_entry_invoice = EXCLUDED.copy_obs_to_entry_invoice,
    homologation_default = EXCLUDED.homologation_default,
    use_stock_uom = EXCLUDED.use_stock_uom,
    generic_supplier_code = EXCLUDED.generic_supplier_code,
    default_due_base_date = EXCLUDED.default_due_base_date,
    updated_at = NOW()
RETURNING *;
