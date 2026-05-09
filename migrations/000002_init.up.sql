CREATE TABLE IF NOT EXISTS item_question_answers (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    item_id BIGINT NOT NULL REFERENCES items(id),
    question_id BIGINT NOT NULL REFERENCES questions(id),
    answer TEXT NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (item_id, question_id)
);

CREATE OR REPLACE FUNCTION generate_item_mask()
RETURNS TRIGGER AS $$
DECLARE
    concatenated_mask TEXT;
BEGIN
    SELECT string_agg(answer, '#' ORDER BY id)
    INTO concatenated_mask
    FROM item_question_answers
    WHERE item_id = NEW.item_id;

    INSERT INTO item_masks (
        item_id,
        mask,
        mask_hash,
        business_id,
        created_by,
        created_at
    )
    VALUES (
        NEW.item_id,
        concatenated_mask,
        substr(md5(concatenated_mask), 1, 8),
        'default_business',
        NEW.created_by,
        now()
    );

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_generate_item_mask ON item_question_answers;

CREATE TRIGGER trigger_generate_item_mask
AFTER INSERT ON item_question_answers
FOR EACH ROW
EXECUTE FUNCTION generate_item_mask();
