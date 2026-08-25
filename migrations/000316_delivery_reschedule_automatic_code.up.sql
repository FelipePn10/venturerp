CREATE SEQUENCE delivery_reschedules_code_seq;

SELECT setval(
    'delivery_reschedules_code_seq',
    GREATEST(COALESCE((SELECT MAX(code) FROM delivery_reschedules), 0), 1),
    COALESCE((SELECT MAX(code) FROM delivery_reschedules), 0) > 0
);

ALTER TABLE delivery_reschedules
    ALTER COLUMN code SET DEFAULT nextval('delivery_reschedules_code_seq');

ALTER SEQUENCE delivery_reschedules_code_seq
    OWNED BY delivery_reschedules.code;
