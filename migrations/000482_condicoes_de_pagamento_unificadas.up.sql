-- Uma condição de pagamento só: `payment_conditions`.
--
-- A VFIN0110 gravava em `condicoes_pagamento`, que nenhum fluxo lê: cliente,
-- fornecedor, orçamento, pedido e faturamento usam `payment_conditions`. Quem
-- cadastrou "30/60" pela VFIN0110 não via a condição no pedido. A partir daqui a
-- VFIN0110 mantém `payment_conditions`, e esta migração leva para lá as condições
-- que existiam só na tabela antiga.
--
-- Regras:
--  * a tabela antiga fica como estava (histórico); só deixa de ser usada;
--  * condição com a mesma descrição na mesma empresa não é duplicada;
--  * `parcelas` é a lista de dias ([28,56]); um item objeto vale pelo campo
--    dias/days; lista vazia ou ilegível não migra (a condição continua só na
--    tabela antiga);
--  * item objeto com `percentual` leva o percentual, se TODAS as parcelas o
--    tiverem e a soma fechar 100; senão, percentual nulo = partes iguais.
DO $$
DECLARE
    c        RECORD;
    dias     INT[];
    pcts     NUMERIC[];
    novo_id  BIGINT;
    proximo  BIGINT;
    i        INT;
BEGIN
    SELECT COALESCE(MAX(code), 0) INTO proximo FROM payment_conditions;
    FOR c IN
        SELECT cp.id, cp.nome, cp.parcelas, cp.ativo, cp.enterprise_id
          FROM condicoes_pagamento cp
         WHERE cp.enterprise_id IS NOT NULL
           AND NOT EXISTS (SELECT 1 FROM payment_conditions p
                            WHERE p.enterprise_id = cp.enterprise_id
                              AND lower(btrim(p.description)) = lower(btrim(cp.nome)))
         ORDER BY cp.enterprise_id, cp.id
    LOOP
        dias := NULL;
        pcts := NULL;
        IF jsonb_typeof(c.parcelas) = 'array' THEN
            BEGIN
                SELECT array_agg(d ORDER BY ord) INTO dias FROM (
                    SELECT ord, CASE jsonb_typeof(e)
                                  WHEN 'number' THEN (e #>> '{}')::numeric::int
                                  WHEN 'string' THEN (e #>> '{}')::numeric::int
                                  WHEN 'object' THEN COALESCE(e ->> 'dias', e ->> 'days', e ->> 'due_days')::numeric::int
                                END AS d
                      FROM jsonb_array_elements(c.parcelas) WITH ORDINALITY AS t(e, ord)
                ) x;
                SELECT array_agg(CASE WHEN jsonb_typeof(e) = 'object' THEN NULLIF(e ->> 'percentual', '')::numeric END ORDER BY ord)
                  INTO pcts FROM jsonb_array_elements(c.parcelas) WITH ORDINALITY AS t(e, ord);
            EXCEPTION WHEN others THEN
                dias := NULL;
            END;
        END IF;
        IF dias IS NULL OR cardinality(dias) = 0 OR EXISTS (SELECT 1 FROM unnest(dias) d WHERE d IS NULL OR d < 0 OR d > 32767) THEN
            RAISE NOTICE 'condição de pagamento % (%): parcelas ilegíveis, não migrada', c.id, c.nome;
            CONTINUE;
        END IF;
        IF pcts IS NULL OR EXISTS (SELECT 1 FROM unnest(pcts) v WHERE v IS NULL OR v <= 0 OR v > 100)
           OR (SELECT sum(v) FROM unnest(pcts) v) <> 100 THEN
            pcts := NULL;
        END IF;
        proximo := proximo + 1;
        INSERT INTO payment_conditions (enterprise_id, code, description, average_term, is_at_sight, is_active)
        VALUES (c.enterprise_id, proximo, left(btrim(c.nome), 150),
                (SELECT round(avg(d))::smallint FROM unnest(dias) d),
                cardinality(dias) = 1 AND dias[1] = 0, c.ativo)
        RETURNING id INTO novo_id;
        FOR i IN 1 .. cardinality(dias) LOOP
            INSERT INTO payment_condition_installments (payment_condition_id, installment_number, due_days, percentage)
            VALUES (novo_id, i, dias[i], pcts[i]);
        END LOOP;
    END LOOP;
END $$;
