-- O configurador de produto inteiro nasceu sem dono. As quinze tabelas cfg_*
-- não tinham coluna de empresa e nenhuma das 69 funções de acesso filtrava:
-- `GET /api/configurator/sets` devolvia os conjuntos, variáveis e
-- características de todas as empresas para qualquer usuário autenticado.
--
-- A coluna é obrigatória. Não existe registro de configurador sem empresa —
-- ao contrário da auditoria, aqui não há evento pré-sessão.

ALTER TABLE cfg_sets ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
UPDATE cfg_sets SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE cfg_sets ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE cfg_sets DROP CONSTRAINT IF EXISTS fk_cfg_sets_enterprise;
ALTER TABLE cfg_sets ADD CONSTRAINT fk_cfg_sets_enterprise
    FOREIGN KEY (enterprise_id) REFERENCES enterprise(id);
CREATE INDEX IF NOT EXISTS idx_cfg_sets_enterprise ON cfg_sets (enterprise_id);

ALTER TABLE cfg_variables ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
UPDATE cfg_variables SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE cfg_variables ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE cfg_variables DROP CONSTRAINT IF EXISTS fk_cfg_variables_enterprise;
ALTER TABLE cfg_variables ADD CONSTRAINT fk_cfg_variables_enterprise
    FOREIGN KEY (enterprise_id) REFERENCES enterprise(id);
CREATE INDEX IF NOT EXISTS idx_cfg_variables_enterprise ON cfg_variables (enterprise_id);

ALTER TABLE cfg_variable_languages ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
UPDATE cfg_variable_languages SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE cfg_variable_languages ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE cfg_variable_languages DROP CONSTRAINT IF EXISTS fk_cfg_variable_languages_enterprise;
ALTER TABLE cfg_variable_languages ADD CONSTRAINT fk_cfg_variable_languages_enterprise
    FOREIGN KEY (enterprise_id) REFERENCES enterprise(id);
CREATE INDEX IF NOT EXISTS idx_cfg_variable_languages_enterprise ON cfg_variable_languages (enterprise_id);

ALTER TABLE cfg_characteristics ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
UPDATE cfg_characteristics SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE cfg_characteristics ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE cfg_characteristics DROP CONSTRAINT IF EXISTS fk_cfg_characteristics_enterprise;
ALTER TABLE cfg_characteristics ADD CONSTRAINT fk_cfg_characteristics_enterprise
    FOREIGN KEY (enterprise_id) REFERENCES enterprise(id);
CREATE INDEX IF NOT EXISTS idx_cfg_characteristics_enterprise ON cfg_characteristics (enterprise_id);

ALTER TABLE cfg_characteristic_languages ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
UPDATE cfg_characteristic_languages SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE cfg_characteristic_languages ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE cfg_characteristic_languages DROP CONSTRAINT IF EXISTS fk_cfg_characteristic_languages_enterprise;
ALTER TABLE cfg_characteristic_languages ADD CONSTRAINT fk_cfg_characteristic_languages_enterprise
    FOREIGN KEY (enterprise_id) REFERENCES enterprise(id);
CREATE INDEX IF NOT EXISTS idx_cfg_characteristic_languages_enterprise ON cfg_characteristic_languages (enterprise_id);

ALTER TABLE cfg_characteristic_receiving_items ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
UPDATE cfg_characteristic_receiving_items SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE cfg_characteristic_receiving_items ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE cfg_characteristic_receiving_items DROP CONSTRAINT IF EXISTS fk_cfg_characteristic_receiving_items_enterprise;
ALTER TABLE cfg_characteristic_receiving_items ADD CONSTRAINT fk_cfg_characteristic_receiving_items_enterprise
    FOREIGN KEY (enterprise_id) REFERENCES enterprise(id);
CREATE INDEX IF NOT EXISTS idx_cfg_characteristic_receiving_items_enterprise ON cfg_characteristic_receiving_items (enterprise_id);

ALTER TABLE cfg_item_characteristics ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
UPDATE cfg_item_characteristics SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE cfg_item_characteristics ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE cfg_item_characteristics DROP CONSTRAINT IF EXISTS fk_cfg_item_characteristics_enterprise;
ALTER TABLE cfg_item_characteristics ADD CONSTRAINT fk_cfg_item_characteristics_enterprise
    FOREIGN KEY (enterprise_id) REFERENCES enterprise(id);
CREATE INDEX IF NOT EXISTS idx_cfg_item_characteristics_enterprise ON cfg_item_characteristics (enterprise_id);

ALTER TABLE cfg_item_char_default_answers ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
UPDATE cfg_item_char_default_answers SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE cfg_item_char_default_answers ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE cfg_item_char_default_answers DROP CONSTRAINT IF EXISTS fk_cfg_item_char_default_answers_enterprise;
ALTER TABLE cfg_item_char_default_answers ADD CONSTRAINT fk_cfg_item_char_default_answers_enterprise
    FOREIGN KEY (enterprise_id) REFERENCES enterprise(id);
CREATE INDEX IF NOT EXISTS idx_cfg_item_char_default_answers_enterprise ON cfg_item_char_default_answers (enterprise_id);

ALTER TABLE cfg_item_mask_answers ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
UPDATE cfg_item_mask_answers SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE cfg_item_mask_answers ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE cfg_item_mask_answers DROP CONSTRAINT IF EXISTS fk_cfg_item_mask_answers_enterprise;
ALTER TABLE cfg_item_mask_answers ADD CONSTRAINT fk_cfg_item_mask_answers_enterprise
    FOREIGN KEY (enterprise_id) REFERENCES enterprise(id);
CREATE INDEX IF NOT EXISTS idx_cfg_item_mask_answers_enterprise ON cfg_item_mask_answers (enterprise_id);

ALTER TABLE cfg_description_types ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
UPDATE cfg_description_types SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE cfg_description_types ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE cfg_description_types DROP CONSTRAINT IF EXISTS fk_cfg_description_types_enterprise;
ALTER TABLE cfg_description_types ADD CONSTRAINT fk_cfg_description_types_enterprise
    FOREIGN KEY (enterprise_id) REFERENCES enterprise(id);
CREATE INDEX IF NOT EXISTS idx_cfg_description_types_enterprise ON cfg_description_types (enterprise_id);

ALTER TABLE cfg_item_descriptions ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
UPDATE cfg_item_descriptions SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE cfg_item_descriptions ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE cfg_item_descriptions DROP CONSTRAINT IF EXISTS fk_cfg_item_descriptions_enterprise;
ALTER TABLE cfg_item_descriptions ADD CONSTRAINT fk_cfg_item_descriptions_enterprise
    FOREIGN KEY (enterprise_id) REFERENCES enterprise(id);
CREATE INDEX IF NOT EXISTS idx_cfg_item_descriptions_enterprise ON cfg_item_descriptions (enterprise_id);

ALTER TABLE cfg_item_description_lines ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
UPDATE cfg_item_description_lines SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE cfg_item_description_lines ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE cfg_item_description_lines DROP CONSTRAINT IF EXISTS fk_cfg_item_description_lines_enterprise;
ALTER TABLE cfg_item_description_lines ADD CONSTRAINT fk_cfg_item_description_lines_enterprise
    FOREIGN KEY (enterprise_id) REFERENCES enterprise(id);
CREATE INDEX IF NOT EXISTS idx_cfg_item_description_lines_enterprise ON cfg_item_description_lines (enterprise_id);

ALTER TABLE cfg_item_rules ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
UPDATE cfg_item_rules SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE cfg_item_rules ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE cfg_item_rules DROP CONSTRAINT IF EXISTS fk_cfg_item_rules_enterprise;
ALTER TABLE cfg_item_rules ADD CONSTRAINT fk_cfg_item_rules_enterprise
    FOREIGN KEY (enterprise_id) REFERENCES enterprise(id);
CREATE INDEX IF NOT EXISTS idx_cfg_item_rules_enterprise ON cfg_item_rules (enterprise_id);

ALTER TABLE cfg_item_rule_conditions ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
UPDATE cfg_item_rule_conditions SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE cfg_item_rule_conditions ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE cfg_item_rule_conditions DROP CONSTRAINT IF EXISTS fk_cfg_item_rule_conditions_enterprise;
ALTER TABLE cfg_item_rule_conditions ADD CONSTRAINT fk_cfg_item_rule_conditions_enterprise
    FOREIGN KEY (enterprise_id) REFERENCES enterprise(id);
CREATE INDEX IF NOT EXISTS idx_cfg_item_rule_conditions_enterprise ON cfg_item_rule_conditions (enterprise_id);

ALTER TABLE cfg_equivalent_rules ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
UPDATE cfg_equivalent_rules SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
ALTER TABLE cfg_equivalent_rules ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE cfg_equivalent_rules DROP CONSTRAINT IF EXISTS fk_cfg_equivalent_rules_enterprise;
ALTER TABLE cfg_equivalent_rules ADD CONSTRAINT fk_cfg_equivalent_rules_enterprise
    FOREIGN KEY (enterprise_id) REFERENCES enterprise(id);
CREATE INDEX IF NOT EXISTS idx_cfg_equivalent_rules_enterprise ON cfg_equivalent_rules (enterprise_id);
