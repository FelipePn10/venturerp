-- Volta a 6 caracteres traduzindo pelo catálogo da empresa; o que ainda não couber é cortado.
UPDATE fiscal_exit_items xi SET unidade_comercial = u.fiscal_code
  FROM fiscal_exits x, units_of_measure u
 WHERE x.id = xi.fiscal_exit_id AND u.enterprise_id = x.enterprise_id AND u.code = xi.unidade_comercial
   AND length(xi.unidade_comercial) > 6;
ALTER TABLE fiscal_exit_items ALTER COLUMN unidade_comercial TYPE VARCHAR(6) USING left(unidade_comercial, 6);
