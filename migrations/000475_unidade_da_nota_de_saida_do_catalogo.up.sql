-- A linha da nota de saída guarda a unidade do CATÁLOGO (a de venda do pedido),
-- não a fiscal: a tradução para o código fiscal (uCom, até 6 caracteres) é feita
-- na autorização. Com VARCHAR(6), vender em unidade de código longo (MICROMETRO,
-- TONELADA, ou uma cadastrada pela empresa com até 10 letras) quebrava a gravação
-- da nota. O catálogo aceita até 10.
ALTER TABLE fiscal_exit_items ALTER COLUMN unidade_comercial TYPE VARCHAR(10);
