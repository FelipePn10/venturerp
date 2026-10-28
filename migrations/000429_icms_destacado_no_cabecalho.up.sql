-- O cabeçalho da nota de saída guardava o ICMS da OPERAÇÃO e não o destacado:
-- na venda com diferimento (CST 51) o XML autorizado leva vICMS = operação −
-- diferido (120,00 de 195,00 com 38,46%), mas a apuração do mês, a
-- contabilização e o livro somavam os 195,00 — ICMS pago a mais. Corrige as
-- notas existentes para o valor que a SEFAZ autorizou. Apurações e lançamentos
-- já gravados não são alterados por esta migração.
UPDATE fiscal_exits x
   SET valor_icms = t.destacado
  FROM (SELECT fiscal_exit_id, ROUND(SUM(valor_icms - valor_icms_diferido), 2) AS destacado
          FROM fiscal_exit_items GROUP BY fiscal_exit_id HAVING SUM(valor_icms_diferido) > 0) t
 WHERE t.fiscal_exit_id = x.id;
