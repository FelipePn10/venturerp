CREATE TABLE IF NOT EXISTS consumer_service_consumer_enterprises (
    consumer_code BIGINT NOT NULL REFERENCES consumer_service_consumers(code) ON DELETE CASCADE,
    enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (consumer_code, enterprise_id)
);

INSERT INTO consumer_service_consumer_enterprises (consumer_code, enterprise_id)
SELECT DISTINCT call.consumer_code, enterprise.id
FROM consumer_service_calls call
JOIN enterprise ON enterprise.code = call.enterprise_code
ON CONFLICT DO NOTHING;

INSERT INTO consumer_service_consumer_enterprises (consumer_code, enterprise_id)
SELECT consumer.code, enterprise.id
FROM consumer_service_consumers consumer
CROSS JOIN enterprise
WHERE (SELECT COUNT(*) FROM enterprise) = 1
ON CONFLICT DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_consumer_service_consumer_enterprise
    ON consumer_service_consumer_enterprises (enterprise_id, consumer_code);

ALTER TABLE consumer_service_customer_contacts
    ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);

UPDATE consumer_service_customer_contacts contact
SET enterprise_id = candidate.enterprise_id
FROM (
    SELECT call.customer_code, MIN(enterprise.id) AS enterprise_id
    FROM consumer_service_calls call
    JOIN enterprise ON enterprise.code = call.enterprise_code
    WHERE call.customer_code IS NOT NULL
    GROUP BY call.customer_code
    HAVING COUNT(DISTINCT enterprise.id) = 1
) candidate
WHERE candidate.customer_code = contact.customer_code
  AND contact.enterprise_id IS NULL;

UPDATE consumer_service_customer_contacts
SET enterprise_id = (SELECT MIN(id) FROM enterprise)
WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise) = 1;

CREATE INDEX IF NOT EXISTS idx_consumer_service_customer_contacts_tenant
    ON consumer_service_customer_contacts (enterprise_id, customer_code, opened_at);
