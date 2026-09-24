-- Perfil OPERATOR — o posto de trabalho.
--
-- O escopo `production:report` foi criado para separar o apontamento de chão de
-- fábrica da permissão de PLANEJAMENTO (`CanCreatePlannedOrder`), que até então
-- era o gate de iniciar etapa, apontar produção, consumo, sucata e leitura de
-- código de barras. Quem opera uma máquina precisava da permissão de criar
-- ordem planejada — e ganhava junto tudo o que ela abre.
--
-- Faltava a metade de baixo: `users.role` e `user_enterprises.role` tinham um
-- CHECK que só aceitava USER e ADMIN, então o perfil novo não podia sequer ser
-- gravado. A permissão existia e o usuário que a usaria, não.
--
-- VIEWER entra junto pelo mesmo motivo: o mapa de permissões da aplicação já o
-- descrevia (somente leitura) e o banco também o recusava.

ALTER TABLE users             DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE user_enterprises  DROP CONSTRAINT IF EXISTS user_enterprises_role_check;

ALTER TABLE users
    ADD CONSTRAINT users_role_check
    CHECK (role IN ('ADMIN', 'USER', 'OPERATOR', 'VIEWER'));

ALTER TABLE user_enterprises
    ADD CONSTRAINT user_enterprises_role_check
    CHECK (role IN ('ADMIN', 'USER', 'OPERATOR', 'VIEWER'));
