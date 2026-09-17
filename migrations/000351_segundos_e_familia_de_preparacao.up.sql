-- Duas lacunas que apareceram usando a tela.
--
-- 1) SEGUNDO como unidade de tempo
--
-- A ficha de chão de fábrica traz tempo de ciclo em segundos — "85 s por peça" —
-- e o cadastro só aceitava MINUTO, HORA e DIA. Converter à mão introduz erro de
-- arredondamento logo na entrada do dado que governa toda a capacidade. O
-- material de treinamento já ensinava a lançar em segundos; era o cadastro que
-- não acompanhava.

ALTER TYPE capacity_period_enum ADD VALUE IF NOT EXISTS 'SEGUNDO';

-- 2) Família de preparação no item
--
-- A matriz de setup já aceitava regra por família, com um dos lados em branco
-- como coringa — o que evita a explosão combinatória: quarenta chapas não
-- precisam de mil e seiscentas linhas, precisam de três regras entre famílias.
--
-- O problema é que a família vinha de `commercial_classification_code`, que é
-- um agrupamento COMERCIAL. Para preparação, o que agrupa é o processo: numa
-- máquina de corte, a espessura e o material. Misturar os dois faria a
-- classificação que o comercial usa decidir tempo de máquina.
--
-- A coluna nasce vazia. Item sem família continua casando apenas por regra de
-- item ou por coringa, exatamente como antes — nada muda para quem não usar.

ALTER TABLE items ADD COLUMN setup_family VARCHAR(60);

COMMENT ON COLUMN items.setup_family IS
    'Família de PREPARAÇÃO: agrupa itens que custam o mesmo para trocar na máquina (ex.: CHAPA-3MM). Independe da classificação comercial.';

-- A matriz é consultada por centro de trabalho e resolvida em memória; o índice
-- serve às telas que listam itens de uma família para conferir a cobertura.
CREATE INDEX idx_items_setup_family ON items (enterprise_id, setup_family)
    WHERE setup_family IS NOT NULL;
