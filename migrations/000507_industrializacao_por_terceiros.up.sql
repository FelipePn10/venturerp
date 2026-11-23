-- Industrialização por terceiros (bug I / melhoria 4, .ai/tasks/rodada-validacao-bugs-e-melhorias.md).
--
-- O estoque "em poder de terceiros" é um almoxarifado por fornecedor, local
-- TERCEIRO: a remessa da OS de terceiro transfere a peça para ele e o retorno
-- traz de volta. Saldo, custo e kardex são os do estoque; o Bloco H separa a
-- posse (indicador 1, participante = fornecedor do almoxarifado).

ALTER TYPE warehouse_location ADD VALUE IF NOT EXISTS 'TERCEIRO';

-- O fornecedor que tem a posse (só no almoxarifado TERCEIRO). O código do
-- fornecedor é único no banco; a empresa é a do almoxarifado.
ALTER TABLE warehouse ADD COLUMN IF NOT EXISTS supplier_code BIGINT;
CREATE UNIQUE INDEX IF NOT EXISTS uq_warehouse_terceiro_por_fornecedor
    ON warehouse (enterprise_id, supplier_code) WHERE supplier_code IS NOT NULL;
COMMENT ON COLUMN warehouse.supplier_code IS
    'Fornecedor com a posse do material (almoxarifado TERCEIRO, industrialização por encomenda).';

-- O movimento da OS que mexeu no estoque e a NF-e de remessa que o acoberta.
ALTER TABLE third_party_service_movements
    ADD COLUMN IF NOT EXISTS moved_stock BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS fiscal_exit_id BIGINT REFERENCES fiscal_exits(id);
CREATE INDEX IF NOT EXISTS idx_tps_movements_fiscal_exit ON third_party_service_movements (fiscal_exit_id) WHERE fiscal_exit_id IS NOT NULL;

SELECT instalar_conferencia_de_empresa();
