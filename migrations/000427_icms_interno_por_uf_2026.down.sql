-- Volta ao seed da migração 000095: as 12 UFs com os valores antigos (onde
-- ainda estão com o valor desta migração) e sem as UFs incluídas.
WITH semente(uf, icms, fcp) AS (VALUES
    ('PR',0.195,0.02),('SP',0.18,0.02),('RJ',0.18,0.02),('MG',0.18,0.02),('RS',0.17,0.02),('SC',0.17,0.02),
    ('ES',0.17,0.02),('BA',0.18,0.02),('GO',0.17,0.02),('MT',0.17,0.02),('MS',0.17,0.02),('DF',0.18,0.02)
), novo(uf, icms, fcp) AS (VALUES
    ('PR',0.195,0),('SP',0.18,0),('RJ',0.20,0.02),('MG',0.18,0),('RS',0.17,0),('SC',0.17,0),
    ('ES',0.17,0),('BA',0.205,0),('GO',0.19,0),('MT',0.17,0),('MS',0.17,0),('DF',0.20,0)
)
UPDATE icms_internal i SET aliq_icms = s.icms, aliq_fcp = s.fcp
  FROM semente s JOIN novo n ON n.uf = s.uf
 WHERE i.uf = s.uf AND i.aliq_icms = n.icms AND i.aliq_fcp = n.fcp;
DELETE FROM icms_internal
 WHERE uf NOT IN ('PR','SP','RJ','MG','RS','SC','ES','BA','GO','MT','MS','DF');
