BEGIN;

ALTER TABLE public.purchase_orders DROP CONSTRAINT IF EXISTS fk_purchase_orders_supplier;

DROP TABLE IF EXISTS supplier_parameters;
DROP TABLE IF EXISTS supplier_enterprises;
DROP TABLE IF EXISTS supplier_contact_emails;
DROP TABLE IF EXISTS supplier_contact_phones;
DROP TABLE IF EXISTS supplier_contacts;
DROP TABLE IF EXISTS supplier_due_dates;
DROP TABLE IF EXISTS supplier_emails;
DROP TABLE IF EXISTS supplier_phones;
DROP TABLE IF EXISTS supplier_addresses;
DROP TABLE IF EXISTS suppliers;
DROP TABLE IF EXISTS supplier_contact_types;
DROP TABLE IF EXISTS supplier_types;

COMMIT;
