BEGIN;

-- ─────────────────────────────────────────────────────────────────────────────
-- Remoção do configurador legado baseado em `questions`.
--
-- Substituído integralmente pelo novo configurador (`cfg_*`, ver migrations
-- 000199–000203) e pela resolução de estrutura cfg-only. `item_masks` é MANTIDA
-- (usada pelo `cfg_*`); apenas as tabelas exclusivamente legadas são removidas.
-- ─────────────────────────────────────────────────────────────────────────────

DROP TABLE IF EXISTS item_question_answers CASCADE; -- resíduo do design antigo de respostas
DROP TABLE IF EXISTS item_mask_answers CASCADE;
DROP TABLE IF EXISTS item_questions CASCADE;
DROP TABLE IF EXISTS question_options CASCADE;
DROP TABLE IF EXISTS questions CASCADE;

COMMIT;
