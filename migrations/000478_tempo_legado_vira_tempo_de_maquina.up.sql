-- O "tempo padrão" legado vira tempo de máquina (decisão do dono do produto,
-- 09/11/2026).
--
-- A tela antiga gravava o "tempo de máquina" da ETAPA em
-- route_operations.standard_time, que o modelo de tempo nunca leu: o número
-- ficava no cadastro e não entrava em OP, prazo, capacidade nem custo. O mesmo
-- valia para operações de biblioteca cadastradas antes do modelo de tempo, com
-- só o legado preenchido (para essas o cálculo já usava o legado como
-- alternativa; aqui ele passa a ser o valor gravado).
--
-- O valor é copiado como está, na unidade em que foi digitado (a da etapa ou,
-- sem ela, a da operação). Só se preenche o que está vazio: tempo de máquina
-- já informado nunca é sobrescrito.
UPDATE route_operations
SET run_time = standard_time, updated_at = NOW()
WHERE run_time IS NULL AND standard_time > 0;

UPDATE operations
SET run_time = standard_time, updated_at = NOW()
WHERE COALESCE(run_time, 0) = 0 AND standard_time > 0;
