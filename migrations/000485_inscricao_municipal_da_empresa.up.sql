-- Inscrição municipal da empresa.
--
-- O cadastro da empresa já recebia "inscricao_municipal" e a descartava, e a
-- NFS-e saía sem a inscrição do prestador — exigida pela maioria das
-- prefeituras (padrão ABRASF), que recusam a nota sem ela.
ALTER TABLE public.fiscal_configs
    ADD COLUMN IF NOT EXISTS inscricao_municipal VARCHAR(20);
