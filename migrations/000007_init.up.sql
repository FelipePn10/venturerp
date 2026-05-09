
DROP FUNCTION IF EXISTS generate_product_mask();

CREATE TABLE IF NOT EXISTS item_questions (
    item_id  BIGINT NOT NULL REFERENCES items(id),
    question_id BIGINT NOT NULL REFERENCES questions(id),
    position    INT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (item_id, question_id),
    UNIQUE (item_id, position)
);
