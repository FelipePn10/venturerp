-- Volta ao que existia antes: só a origem PR, com os 12% que estavam gravados
-- para DF, ES, GO, MS e MT.
DELETE FROM icms_interstate WHERE origin_uf <> 'PR';
UPDATE icms_interstate SET aliq_icms = 0.12
 WHERE origin_uf = 'PR' AND destination_uf IN ('DF','ES','GO','MS','MT');
