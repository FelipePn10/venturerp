BEGIN;

-- O cabeçalho de estrutura (VBOM0100) nasceu sem coluna de empresa: qualquer
-- tenant conseguia abrir/alterar o cabeçalho de outro pelo id. Passa a ser
-- multiempresa como o resto da engenharia.
ALTER TABLE bom_headers ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);

-- Herda a empresa do item quando possível; o resto cai na primeira empresa.
UPDATE bom_headers h
	   SET enterprise_id = candidate.enterprise_id
	  FROM (
	       SELECT code, MIN(enterprise_id) AS enterprise_id
	         FROM items
	        GROUP BY code
	       HAVING COUNT(DISTINCT enterprise_id) = 1
	  ) candidate
	 WHERE candidate.code = h.item_code AND h.enterprise_id IS NULL;
UPDATE bom_headers SET enterprise_id = (SELECT MIN(id) FROM enterprise)
 WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise) = 1;

ALTER TABLE bom_headers ALTER COLUMN enterprise_id SET NOT NULL;

-- A unicidade de versão passa a valer por empresa.
DROP INDEX IF EXISTS idx_bom_headers_unique;
CREATE UNIQUE INDEX IF NOT EXISTS idx_bom_headers_unique
    ON bom_headers(enterprise_id, item_code, COALESCE(mask, ''), version);
CREATE INDEX IF NOT EXISTS idx_bom_headers_tenant_item
    ON bom_headers(enterprise_id, item_code) WHERE is_active;

COMMIT;
