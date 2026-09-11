-- Terceira leva do isolamento multiempresa, encontrada exercitando as rotas com
-- duas empresas (scripts/audit-tenant-isolation.mjs, no repositório do front).
--
-- Cada bloco abaixo era uma rota entregando dado de outra empresa:
--   /api/suppliers/{code}                  → cadastro, contatos e telefones
--   /api/suppliers/{code}/purchasing-defaults
--   /api/restriction-reason/{code} e /list → motivos de restrição
--   /api/standard-cost/work-center-costs   → custo/hora por centro
--
-- As tabelas-filhas do fornecedor (endereços, telefones, e-mails, contatos,
-- vencimentos) não ganham coluna própria: o dono delas é o fornecedor, e
-- duplicar a empresa em cada uma só criaria a chance de divergirem do pai. As
-- consultas dessas tabelas passam a exigir que o fornecedor seja da empresa.
--
-- O cadastro de fornecedor merece uma palavra. O schema já tinha
-- `supplier_enterprises`, que é o padrão do SAP (LFA1 dados gerais × LFB1 dados
-- por empresa) — mas estava vazio, e `suppliers` não tinha dono. Numa base
-- multiempresa isso significa que o cadastro, o CNPJ e os contatos comerciais
-- de uma empresa apareciam para a outra. `suppliers` passa a ter dono; a tabela
-- por empresa continua existindo para o que é de fato específico de cada uma
-- (conta contábil, tabela de preço, tratamento de IPI).

ALTER TABLE suppliers                ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
ALTER TABLE restriction_reasons      ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
ALTER TABLE work_center_costs        ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;

-- Backfill: o que já existe pertence à empresa mais antiga da base. Em base
-- nova essa é a única empresa; em base com mais de uma, não há como adivinhar
-- em retrospecto — e é por isso que a coluna nasce agora, antes de existir uma
-- segunda empresa em produção.
UPDATE suppliers               SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
UPDATE restriction_reasons     SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
UPDATE work_center_costs       SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;

ALTER TABLE suppliers                ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE restriction_reasons      ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE work_center_costs        ALTER COLUMN enterprise_id SET NOT NULL;

ALTER TABLE suppliers               ADD CONSTRAINT fk_suppliers_enterprise               FOREIGN KEY (enterprise_id) REFERENCES enterprise(id);
ALTER TABLE restriction_reasons     ADD CONSTRAINT fk_restriction_reasons_enterprise     FOREIGN KEY (enterprise_id) REFERENCES enterprise(id);
ALTER TABLE work_center_costs       ADD CONSTRAINT fk_work_center_costs_enterprise       FOREIGN KEY (enterprise_id) REFERENCES enterprise(id);

CREATE INDEX IF NOT EXISTS idx_suppliers_enterprise               ON suppliers (enterprise_id);
CREATE INDEX IF NOT EXISTS idx_restriction_reasons_enterprise     ON restriction_reasons (enterprise_id);
CREATE INDEX IF NOT EXISTS idx_work_center_costs_enterprise       ON work_center_costs (enterprise_id);

-- Unicidade passa a ser por empresa onde é possível. Global, ela vaza de outro
-- jeito: a empresa B não consegue cadastrar o código 1 porque a empresa A já o
-- tem — e a mensagem de conflito confirma a existência do registro alheio.
ALTER TABLE restriction_reasons DROP CONSTRAINT IF EXISTS restriction_reasons_code_key;
ALTER TABLE work_center_costs   DROP CONSTRAINT IF EXISTS work_center_costs_work_center_id_key;

ALTER TABLE restriction_reasons ADD CONSTRAINT restriction_reasons_enterprise_code_key UNIQUE (enterprise_id, code);
ALTER TABLE work_center_costs   ADD CONSTRAINT work_center_costs_enterprise_wc_key     UNIQUE (enterprise_id, work_center_id);

-- `suppliers.code` continua único globalmente, de propósito:
-- `purchase_orders.supplier_code` é uma chave estrangeira para ele, e trocá-la
-- por chave composta exigiria reescrever o pedido de compra inteiro — risco
-- desproporcional ao ganho. A consequência é apenas cosmética: a numeração de
-- fornecedor é contínua entre as empresas (a segunda empresa começa no 4, não
-- no 1). O isolamento de leitura, que é o que importa, vem do enterprise_id.
