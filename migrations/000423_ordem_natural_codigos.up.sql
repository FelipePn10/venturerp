-- Ordem natural de códigos hierárquicos ("2.1.9" antes de "2.1.10"). Cada
-- segmento numérico vira texto com zeros à esquerda; o array compara segmento a
-- segmento, e o pai (prefixo mais curto) vem antes dos filhos. Segmento não
-- numérico fica como está.
CREATE OR REPLACE FUNCTION ordem_natural(codigo text) RETURNS text[]
LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT COALESCE(array_agg(CASE WHEN s ~ '^\d+$' THEN lpad(s, 12, '0') ELSE s END ORDER BY o), '{}')
      FROM unnest(string_to_array(COALESCE(codigo, ''), '.')) WITH ORDINALITY AS u(s, o)
$$;
