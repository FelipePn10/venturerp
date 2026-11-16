DROP TABLE IF EXISTS customer_credit_reviews;
DROP TABLE IF EXISTS customer_credit_policies;
DROP INDEX IF EXISTS ix_customers_cnpj_root;

ALTER TABLE customers
    DROP CONSTRAINT IF EXISTS customers_shipping_carrier_id_fkey,
    DROP COLUMN IF EXISTS ie_indicator,
    DROP COLUMN IF EXISTS tax_regime,
    DROP COLUMN IF EXISTS main_cnae,
    DROP COLUMN IF EXISTS nfe_email,
    DROP COLUMN IF EXISTS billing_email,
    DROP COLUMN IF EXISTS representative_code,
    DROP COLUMN IF EXISTS shipping_carrier_id,
    DROP COLUMN IF EXISTS freight_type,
    DROP COLUMN IF EXISTS requires_customer_po,
    DROP COLUMN IF EXISTS requires_quality_cert,
    DROP COLUMN IF EXISTS allows_partial_billing,
    DROP COLUMN IF EXISTS receiving_notes,
    DROP COLUMN IF EXISTS commercial_notes,
    DROP COLUMN IF EXISTS financial_notes,
    DROP COLUMN IF EXISTS registration_status,
    DROP COLUMN IF EXISTS registration_checked_at,
    DROP COLUMN IF EXISTS registration_attempt_at,
    DROP COLUMN IF EXISTS opening_date,
    DROP COLUMN IF EXISTS share_capital,
    DROP COLUMN IF EXISTS company_size,
    DROP COLUMN IF EXISTS legal_nature,
    DROP COLUMN IF EXISTS credit_score,
    DROP COLUMN IF EXISTS credit_risk_class,
    DROP COLUMN IF EXISTS credit_limit_suggested,
    DROP COLUMN IF EXISTS credit_limit_expires_at,
    DROP COLUMN IF EXISTS credit_reviewed_at;

-- As FKs voltam compostas com a empresa, como a 000395 deixou.
ALTER TABLE customers
    ADD COLUMN payment_condition_id    BIGINT,
    ADD COLUMN sales_table_id          BIGINT,
    ADD COLUMN invoice_type_id         BIGINT,
    ADD COLUMN carrier_group_id        BIGINT,
    ADD COLUMN payment_cond_visibility payment_condition_visibility_enum NOT NULL DEFAULT 'TODOS',
    ADD CONSTRAINT customers_payment_condition_id_empresa_fkey
        FOREIGN KEY (enterprise_id, payment_condition_id) REFERENCES payment_conditions (enterprise_id, id),
    ADD CONSTRAINT customers_sales_table_id_empresa_fkey
        FOREIGN KEY (enterprise_id, sales_table_id) REFERENCES sales_tables (enterprise_id, id),
    ADD CONSTRAINT customers_invoice_type_id_empresa_fkey
        FOREIGN KEY (enterprise_id, invoice_type_id) REFERENCES invoice_types (enterprise_id, id),
    ADD CONSTRAINT customers_carrier_group_id_empresa_fkey
        FOREIGN KEY (enterprise_id, carrier_group_id) REFERENCES carrier_groups (enterprise_id, id);

-- A conferência de empresa do gatilho ainda cita as colunas que saíram
-- (representative_code...): reinstala antes de o UPDATE dispará-la.
SELECT instalar_conferencia_de_empresa();

UPDATE customers c
   SET payment_condition_id = r.payment_condition_id,
       sales_table_id = r.sales_table_id,
       invoice_type_id = r.invoice_type_id,
       carrier_group_id = r.carrier_group_id,
       payment_cond_visibility = COALESCE(r.payment_cond_visibility, 'TODOS')
  FROM customers_campos_retirados r
 WHERE r.customer_id = c.id;

DROP TABLE IF EXISTS customers_campos_retirados;

SELECT instalar_conferencia_de_empresa();
