-- ═══════════════════════════════════════════════════════════════════════════
-- Conversões de unidade dos materiais do RN 01001
--
-- CONFIRA OS PESOS ANTES DE RODAR. Eles vêm do Anexo A.1 do Treinamento Dia 1
-- e valem para chapa 1200 × 3000. Se a chapa que a fábrica compra tem outra
-- dimensão, o peso muda e este script grava o número errado.
--
-- O que este script faz: cadastra a ponte entre a unidade em que a engenharia
-- mede o consumo (kg de chapa, mm de tubo) e a unidade em que o almoxarifado
-- conta o material (chapas, barras).
--
-- O que ele NÃO faz: não altera nenhuma linha de estrutura. As seis linhas
-- existentes continuam com o fator congelado em 1, exatamente como hoje. Para
-- cada uma adotar a conversão, é preciso REABRIR e REGRAVAR a linha na
-- VENT0210 — a decisão de mudar o que o planejamento consome é de quem desenha
-- o produto, não de um script.
--
-- Rodar como:
--   docker exec -i -e PGPASSWORD=... venturerp-postgres \
--     psql -U <user> -d <db> < docs/dev/conversoes-rn01001.sql
-- ═══════════════════════════════════════════════════════════════════════════

BEGIN;

INSERT INTO item_unit_conversions (item_code, mask, from_uom, to_uom, factor, created_by)
SELECT v.item, '', v.de, v.para, v.fator, (SELECT id FROM users ORDER BY created_at LIMIT 1)
FROM (VALUES
    -- Tubo: o comprimento da barra está no próprio nome do item.
    (18::bigint, 'UN', 'MM', 6000.0),
    -- Chapas: peso da chapa inteira 1200 × 3000 (Anexo A.1).
    ( 3::bigint, 'UN', 'KG',  226.08),   -- 7,94 mm (5/16")
    ( 4::bigint, 'UN', 'KG',  268.47),   -- 9,5 mm (3/8")
    ( 2::bigint, 'UN', 'KG',   84.78)    -- 3,00 mm
) AS v(item, de, para, fator)
ON CONFLICT (item_code, mask, from_uom, to_uom)
DO UPDATE SET factor = EXCLUDED.factor, is_active = TRUE;

-- Conferência: o que ficou cadastrado, com o nome do material.
SELECT i.name AS material,
       '1 ' || c.from_uom || ' = ' || c.factor || ' ' || c.to_uom AS equivalencia
FROM item_unit_conversions c
JOIN items i ON i.code = c.item_code
WHERE c.item_code IN (2, 3, 4, 18) AND c.is_active
ORDER BY i.name;

-- Quais linhas de estrutura passam a poder ser regravadas com a conversão.
SELECT COALESCE(p.business_code, s.parent_code::text) AS peca,
       i.name AS material,
       s.quantity || ' ' || s.unit_of_measurement AS escrito_pela_engenharia,
       round(s.quantity * (SELECT 1 / c.factor FROM item_unit_conversions c
                           WHERE c.item_code = i.code AND c.from_uom = i.warehouse_unit_of_measurement::text
                             AND c.to_uom = s.unit_of_measurement::text AND c.is_active
                           LIMIT 1), 6) || ' ' || i.warehouse_unit_of_measurement AS passara_a_valer
FROM item_structures s
JOIN items i ON i.code = s.child_code
LEFT JOIN items p ON p.code = s.parent_code
WHERE s.is_active AND s.unit_of_measurement::text <> i.warehouse_unit_of_measurement::text
ORDER BY 1;

COMMIT;
