-- Volta o cabeçalho ao ICMS da operação nas notas com diferimento.
UPDATE fiscal_exits x
   SET valor_icms = t.operacao
  FROM (SELECT fiscal_exit_id, ROUND(SUM(valor_icms), 2) AS operacao
          FROM fiscal_exit_items GROUP BY fiscal_exit_id HAVING SUM(valor_icms_diferido) > 0) t
 WHERE t.fiscal_exit_id = x.id;
