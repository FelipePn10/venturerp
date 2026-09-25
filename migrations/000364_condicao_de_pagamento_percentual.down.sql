ALTER TABLE payment_condition_installments
    DROP CONSTRAINT IF EXISTS payment_condition_installments_entrada_sem_prazo_check,
    DROP CONSTRAINT IF EXISTS payment_condition_installments_percentage_check;

ALTER TABLE payment_condition_installments
    DROP COLUMN IF EXISTS percentage,
    DROP COLUMN IF EXISTS base_event;

DROP TYPE IF EXISTS payment_base_event_enum;
