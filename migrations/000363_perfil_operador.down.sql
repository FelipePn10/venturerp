-- A restrição antiga só volta se ninguém estiver usando os perfis novos;
-- forçar apagaria o acesso de quem já opera com eles.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM users WHERE role NOT IN ('ADMIN','USER'))
   AND NOT EXISTS (SELECT 1 FROM user_enterprises WHERE role NOT IN ('ADMIN','USER')) THEN
        ALTER TABLE users            DROP CONSTRAINT IF EXISTS users_role_check;
        ALTER TABLE user_enterprises DROP CONSTRAINT IF EXISTS user_enterprises_role_check;
        ALTER TABLE users            ADD CONSTRAINT users_role_check CHECK (role IN ('ADMIN','USER'));
        ALTER TABLE user_enterprises ADD CONSTRAINT user_enterprises_role_check CHECK (role IN ('ADMIN','USER'));
    END IF;
END $$;
