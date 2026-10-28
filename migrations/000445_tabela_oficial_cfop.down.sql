-- Os códigos carregados ficam (podem estar em notas); só saem as colunas.
ALTER TABLE cfops
    DROP COLUMN IF EXISTS ind_exc_ibscbs,
    DROP COLUMN IF EXISTS ind_comb,
    DROP COLUMN IF EXISTS ind_remes,
    DROP COLUMN IF EXISTS ind_anula,
    DROP COLUMN IF EXISTS ind_retor,
    DROP COLUMN IF EXISTS ind_devol,
    DROP COLUMN IF EXISTS ind_transp,
    DROP COLUMN IF EXISTS ind_comunica,
    DROP COLUMN IF EXISTS ind_nfe,
    DROP COLUMN IF EXISTS fim_vigencia,
    DROP COLUMN IF EXISTS inicio_vigencia,
    DROP COLUMN IF EXISTS oficial;

SELECT instalar_conferencia_de_empresa();
