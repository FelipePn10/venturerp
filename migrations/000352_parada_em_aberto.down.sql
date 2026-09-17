-- Paradas ainda abertas são encerradas no instante da reversão: sob a regra
-- antiga elas não poderiam existir sem fim, e apagá-las perderia o registro de
-- uma parada que de fato aconteceu.
UPDATE machine_downtimes SET ends_at = GREATEST(NOW(), starts_at + INTERVAL '1 minute')
 WHERE ends_at IS NULL;

DROP INDEX IF EXISTS uq_machine_downtimes_aberta;

ALTER TABLE machine_downtimes DROP CONSTRAINT machine_downtimes_check;
ALTER TABLE machine_downtimes ADD CONSTRAINT machine_downtimes_check
    CHECK (ends_at > starts_at);

ALTER TABLE machine_downtimes ALTER COLUMN ends_at SET NOT NULL;
