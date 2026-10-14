-- Quem ligou a execução automática do planejamento.
--
-- O ciclo noturno roda sem ninguém logado, mas o MRP grava as ordens planejadas
-- em nome de um usuário (created_by) e só enxerga os dados da empresa da
-- sessão. Como na sincronização das NF-e recebidas, a execução usa a sessão de
-- quem salvou a parametrização — sem isto não há em nome de quem rodar.
ALTER TABLE planning_auto_run_settings
    ADD COLUMN IF NOT EXISTS updated_by uuid REFERENCES users(id);

COMMENT ON COLUMN planning_auto_run_settings.updated_by IS
    'Usuário que salvou a parametrização; a execução automática roda em nome dele.';
