-- Alíquota interna modal de ICMS das 27 UFs, vigente em 2026, e o FCP
-- (Fundo de Combate à Pobreza) que incide sobre QUALQUER mercadoria. A tabela
-- só tinha 12 UFs, várias desatualizadas (RJ 18%, BA 18%, GO 17%, DF 18%), e
-- FCP de 2% em todas — mas o FCP só é generalizado no RJ (2%) e em SE (1%); nas
-- demais UFs ele incide sobre listas de produtos e vai na regra de ICMS/IPI do
-- produto (acréscimo do tipo Fundo de Combate à Pobreza), não nesta tabela.
--
-- Valores: NCM Brasil (tabela ICMS 2026, com a data de cada alteração) e, para
-- AL, a Lei 9.776/2025 (19% → 20,5% a partir de 01/04/2026).
--
-- Só é atualizada a linha que ainda tem o valor semeado pela migração 000095:
-- quem já corrigiu a tabela pela tela de tabelas tributárias não é sobrescrito.
-- As UFs que faltavam são incluídas.
WITH novo(uf, icms, fcp) AS (VALUES
    ('AC',0.19,0),('AL',0.205,0),('AM',0.20,0),('AP',0.18,0),('BA',0.205,0),('CE',0.20,0),('DF',0.20,0),
    ('ES',0.17,0),('GO',0.19,0),('MA',0.23,0),('MG',0.18,0),('MS',0.17,0),('MT',0.17,0),('PA',0.19,0),
    ('PB',0.20,0),('PE',0.205,0),('PI',0.225,0),('PR',0.195,0),('RJ',0.20,0.02),('RN',0.20,0),('RO',0.195,0),
    ('RR',0.20,0),('RS',0.17,0),('SC',0.17,0),('SE',0.19,0.01),('SP',0.18,0),('TO',0.20,0)
), semente(uf, icms, fcp) AS (VALUES
    ('PR',0.195,0.02),('SP',0.18,0.02),('RJ',0.18,0.02),('MG',0.18,0.02),('RS',0.17,0.02),('SC',0.17,0.02),
    ('ES',0.17,0.02),('BA',0.18,0.02),('GO',0.17,0.02),('MT',0.17,0.02),('MS',0.17,0.02),('DF',0.18,0.02)
), atualizados AS (
    UPDATE icms_internal i SET aliq_icms = n.icms, aliq_fcp = n.fcp, is_active = true
      FROM novo n JOIN semente s ON s.uf = n.uf
     WHERE i.uf = n.uf AND i.aliq_icms = s.icms AND i.aliq_fcp = s.fcp
    RETURNING i.uf
)
INSERT INTO icms_internal (uf, aliq_icms, aliq_fcp, is_active)
SELECT n.uf, n.icms, n.fcp, true FROM novo n
 WHERE NOT EXISTS (SELECT 1 FROM icms_internal i WHERE i.uf = n.uf)
ON CONFLICT (uf) DO NOTHING;
