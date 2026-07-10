BEGIN;

-- Irreversível: o configurador legado (`questions`) foi removido e substituído
-- pelo modelo `cfg_*`. Não há restauração dos dados/estrutura legados — os dados
-- devem ser (re)cadastrados no novo configurador (/api/configurator).
-- (no-op)

COMMIT;
