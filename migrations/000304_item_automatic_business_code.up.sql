BEGIN;
CREATE TABLE item_business_code_sequences (
 enterprise_id BIGINT PRIMARY KEY REFERENCES enterprise(id) ON DELETE CASCADE,
 last_value BIGINT NOT NULL DEFAULT 0 CHECK(last_value>=0),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE OR REPLACE FUNCTION next_item_business_code(p_enterprise_id BIGINT) RETURNS TEXT
LANGUAGE plpgsql AS $$
DECLARE candidate BIGINT;
BEGIN
 INSERT INTO item_business_code_sequences(enterprise_id,last_value) VALUES(p_enterprise_id,0)
 ON CONFLICT(enterprise_id) DO NOTHING;
 PERFORM 1 FROM item_business_code_sequences WHERE enterprise_id=p_enterprise_id FOR UPDATE;
 LOOP
  UPDATE item_business_code_sequences SET last_value=last_value+1,updated_at=NOW()
  WHERE enterprise_id=p_enterprise_id RETURNING last_value INTO candidate;
  EXIT WHEN NOT EXISTS(SELECT 1 FROM items WHERE enterprise_id=p_enterprise_id AND business_code=candidate::text);
 END LOOP;
 RETURN candidate::text;
END $$;
COMMIT;
