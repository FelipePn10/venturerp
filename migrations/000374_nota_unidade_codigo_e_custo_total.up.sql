-- Três correções apontadas na revisão do PR 175, todas com efeito em documento
-- fiscal ou em custo gravado.
--
-- ── 1. Unidade e código do produto na linha da nota ─────────────────────────
-- `fiscal_exit_items` não tem coluna de unidade, e o autorizador manda
-- `UnidadeComercial: "UN"` FIXO. O código do produto sai de `item_code`, que é
-- BIGINT e vira "0" quando nulo.
--
-- Isso é errado em qualquer nota e é errado JÁ no primeiro cliente: a NF-e 5.956
-- da Usimac (retorno parcial, anexada ao levantamento) tem as linhas de material
-- em **KG** com os códigos **10014485** e **10014670**, que são códigos DO CLIENTE
-- e não existem como item nosso. Com a unidade fixa em UN e o código em "0", a
-- nota de retorno sai descrevendo outra mercadoria.
--
-- Nulo mantém o comportamento de hoje (UN e `item_code`), então nenhuma nota já
-- emitida muda de sentido.
ALTER TABLE fiscal_exit_items
    ADD COLUMN IF NOT EXISTS unidade_comercial VARCHAR(6),
    -- Texto, não número: o código pode ser do cliente (beneficiamento) ou conter
    -- letras. `item_code` continua existindo para o vínculo com o nosso cadastro.
    ADD COLUMN IF NOT EXISTS codigo_produto VARCHAR(60);

-- ── 1b. Vínculo da nota com a remessa de beneficiamento ────────────────────
-- O faturamento cria a nota em RASCUNHO e depois baixa o saldo. Se a criação das
-- linhas ou a baixa falhar no meio, o rascunho fica gravado — e a mensagem manda
-- repetir. Só que repetir pedia número novo e criava OUTRO rascunho: em vez de
-- concluir a nota original, a repetição produzia duplicata.
--
-- Com o vínculo, o faturamento ENCONTRA o rascunho da remessa e o retoma. Não há
-- transação possível entre os dois módulos; retomar é o que torna a repetição
-- segura de verdade, em vez de apenas anunciada como segura.
ALTER TABLE fiscal_exits
    ADD COLUMN IF NOT EXISTS customer_material_remittance_id BIGINT;

-- Uma remessa não pode ter DOIS rascunhos de beneficiamento ao mesmo tempo: é
-- exatamente a duplicata que o vínculo existe para impedir. Notas já autorizadas ou
-- canceladas ficam fora do índice, porque a mesma remessa é faturada várias vezes
-- ao longo do tempo (retorno parcial).
CREATE UNIQUE INDEX IF NOT EXISTS uq_fiscal_exits_rascunho_de_beneficiamento
    ON fiscal_exits (customer_material_remittance_id)
    WHERE customer_material_remittance_id IS NOT NULL
      AND status = 'DRAFT'
      AND is_active;

-- ── 2. O custo-padrão gravado ignorava preparação, máquina e terceiro ───────
-- `total_cost` é coluna GERADA somando material + labor + overhead. Quando a
-- apuração passou a separar a conversão em componentes (migração 000373),
-- `labor_cost` deixou de ser a conversão inteira e passou a ser só a mão de obra
-- direta — e o total GRAVADO perdeu preparação, hora-máquina e serviço de terceiro.
--
-- A resposta da apuração estava certa (é calculada em memória), mas
-- `GetStandardCost` e a precificação leem o valor gravado. O efeito é preço de
-- venda formado sobre um custo menor que o real.
--
-- Linhas antigas têm zero nas colunas novas, então o total delas não muda.
ALTER TABLE item_standard_costs DROP COLUMN IF EXISTS total_cost;
ALTER TABLE item_standard_costs
    ADD COLUMN total_cost NUMERIC(18,6)
    GENERATED ALWAYS AS (
        material_cost + setup_cost + machine_cost + labor_cost + subcontract_cost + overhead_cost
    ) STORED;

-- `own_level_cost` e `lower_level_cost` ficam DE FORA da soma de propósito: são a
-- mesma quantia vista por outro corte (o que esta etapa agrega × o que veio de
-- baixo), e somá-los dobraria o custo.

COMMENT ON COLUMN item_standard_costs.total_cost IS
    'Custo unitário cheio. material_cost já inclui o custo dos componentes de níveis inferiores; own_level_cost e lower_level_cost são um CORTE dos mesmos valores e não entram nesta soma.';
