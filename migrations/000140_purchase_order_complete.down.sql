BEGIN;

ALTER TABLE public.purchase_order_items
    DROP COLUMN IF EXISTS purchase_uom,
    DROP COLUMN IF EXISTS internal_uom,
    DROP COLUMN IF EXISTS internal_qty,
    DROP COLUMN IF EXISTS internal_price,
    DROP COLUMN IF EXISTS promised_date,
    DROP COLUMN IF EXISTS tolerance_pct,
    DROP COLUMN IF EXISTS cancelled_tolerance_qty,
    DROP COLUMN IF EXISTS icms_st_pct,
    DROP COLUMN IF EXISTS operation_type_code,
    DROP COLUMN IF EXISTS invoice_type_code,
    DROP COLUMN IF EXISTS accounting_account,
    DROP COLUMN IF EXISTS cost_center_code,
    DROP COLUMN IF EXISTS requester_employee_code,
    DROP COLUMN IF EXISTS contract_code,
    DROP COLUMN IF EXISTS quotation_code,
    DROP COLUMN IF EXISTS utilization_type,
    DROP COLUMN IF EXISTS fiscal_classification_code;

ALTER TABLE public.purchase_orders
    DROP COLUMN IF EXISTS price_table_code,
    DROP COLUMN IF EXISTS invoice_type_code,
    DROP COLUMN IF EXISTS financial_account,
    DROP COLUMN IF EXISTS request_type_code,
    DROP COLUMN IF EXISTS currency_date,
    DROP COLUMN IF EXISTS freight_type,
    DROP COLUMN IF EXISTS freight_value_type,
    DROP COLUMN IF EXISTS freight_value_mode,
    DROP COLUMN IF EXISTS freight_value,
    DROP COLUMN IF EXISTS carrier_code,
    DROP COLUMN IF EXISTS redispatch_carrier_code,
    DROP COLUMN IF EXISTS redispatch_freight_type,
    DROP COLUMN IF EXISTS redispatch_freight_value,
    DROP COLUMN IF EXISTS advance_date,
    DROP COLUMN IF EXISTS advance_value,
    DROP COLUMN IF EXISTS incoterm_code,
    DROP COLUMN IF EXISTS shipment_date,
    DROP COLUMN IF EXISTS talao_number,
    DROP COLUMN IF EXISTS alcada_status;

COMMIT;
