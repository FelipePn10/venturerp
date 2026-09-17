DROP INDEX IF EXISTS idx_items_setup_family;
ALTER TABLE items DROP COLUMN IF EXISTS setup_family;

-- O valor SEGUNDO permanece no enum. Remover um rótulo exige recriar o tipo e
-- reescrever todas as colunas que o usam, e qualquer linha já gravada em
-- segundos ficaria órfã. Um rótulo a mais é inofensivo; a reversão seria o
-- contrário disso.
