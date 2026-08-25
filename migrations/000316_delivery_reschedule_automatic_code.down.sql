ALTER TABLE delivery_reschedules
    ALTER COLUMN code DROP DEFAULT;

DROP SEQUENCE delivery_reschedules_code_seq;
