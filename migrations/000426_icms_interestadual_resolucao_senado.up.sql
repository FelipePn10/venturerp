-- Alíquota interestadual de ICMS pela Resolução do Senado Federal 22/1989:
-- saída do Sul e do Sudeste (exceto o Espírito Santo) para o Norte, o
-- Nordeste, o Centro-Oeste e o Espírito Santo é 7%; as demais, 12%. (Os 4% do
-- importado, Resolução SF 13/2012, são decididos pela origem da mercadoria no
-- cálculo, não por esta tabela.)
--
-- A tabela só tinha a origem PR — empresa de outra UF caía sempre nos 12% — e,
-- de PR, trazia 12% para DF, ES, GO, MS e MT, onde a resolução manda 7%.
-- A alíquota interestadual é fixada pelo Senado: o valor legal substitui o que
-- estiver gravado.
WITH ufs(uf, regiao) AS (VALUES
    ('AC','N'),('AM','N'),('AP','N'),('PA','N'),('RO','N'),('RR','N'),('TO','N'),
    ('AL','NE'),('BA','NE'),('CE','NE'),('MA','NE'),('PB','NE'),('PE','NE'),('PI','NE'),('RN','NE'),('SE','NE'),
    ('DF','CO'),('GO','CO'),('MS','CO'),('MT','CO'),
    ('ES','SE'),('MG','SE'),('RJ','SE'),('SP','SE'),
    ('PR','S'),('RS','S'),('SC','S')
)
INSERT INTO icms_interstate (origin_uf, destination_uf, aliq_icms, description, is_active)
SELECT o.uf, d.uf,
       CASE WHEN o.regiao IN ('S','SE') AND o.uf <> 'ES' AND (d.regiao IN ('N','NE','CO') OR d.uf = 'ES')
            THEN 0.07 ELSE 0.12 END,
       'Resolução SF 22/1989', true
  FROM ufs o CROSS JOIN ufs d
 WHERE o.uf <> d.uf
ON CONFLICT (origin_uf, destination_uf) DO UPDATE
   SET aliq_icms = EXCLUDED.aliq_icms, is_active = true,
       description = COALESCE(NULLIF(icms_interstate.description, ''), EXCLUDED.description);
