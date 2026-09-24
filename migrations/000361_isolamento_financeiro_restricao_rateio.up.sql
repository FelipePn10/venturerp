-- Quinta leva do isolamento multiempresa, encontrada por
-- scripts/audit-tenant-isolation.mjs (repositório do front) em 13/09/2026 e
-- aplicada agora.
--
-- Sete rotas entregavam o dado de outra empresa:
--
--   /api/financial/contas-bancarias/list   /api/financial/condicoes-pagamento/list
--   /api/financial/saldo-contas            /api/allocations/list
--   /api/financial/plano-contas/list       /api/restriction/list
--   /api/financial/centros-custo/list
--
-- Nenhuma das tabelas tinha coluna de empresa: as consultas eram literalmente
-- `FROM contas_bancarias WHERE is_active = true`. E não era só leitura —
-- `contas_bancarias` é alterada por id em quatro pontos (ajuste de saldo
-- inicial, baixa a pagar, baixa a receber e estorno), então uma empresa
-- movimentaria o saldo da conta bancária da outra. Mesma classe do
-- `UpdateEmployee` corrigido na 343.
--
-- Escapou das levas 340 e 343 por nomes parecidos apontando para tabelas
-- diferentes: aquelas isolaram `restriction_reasons` (os motivos) e
-- `cost_centers`; a tela do financeiro lê `restrictions` (a restrição em si) e
-- `centros_custo` (cadastro paralelo legado). Ao auditar, confira o nome da
-- TABELA na consulta, não o nome de negócio.
--
-- Produção tem uma empresa só, então nada vazou. A coluna nasce agora
-- justamente porque ainda dá para dizer a quem pertence o que já existe.

ALTER TABLE contas_bancarias    ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
ALTER TABLE condicoes_pagamento ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
ALTER TABLE payment_conditions  ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
ALTER TABLE plano_contas        ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
ALTER TABLE centros_custo       ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
ALTER TABLE restrictions        ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
ALTER TABLE allocation_bases    ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;

-- O que já existe pertence à empresa mais antiga da base — em base nova, a
-- única. Mesmo critério das migrações 340 e 343.
UPDATE contas_bancarias    SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
UPDATE condicoes_pagamento SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
UPDATE payment_conditions  SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
UPDATE plano_contas        SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
UPDATE centros_custo       SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
UPDATE restrictions        SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
UPDATE allocation_bases    SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;

-- O NOT NULL vale sempre que for possível aplicá-lo; numa instalação nova as
-- tabelas estão vazias e ele entra sem esforço. Só pulamos no caso patológico
-- (linha órfã sem empresa para herdar), onde forçar abortaria a migração
-- inteira e deixaria o banco pior do que estava.
DO $$
DECLARE
    alvo   TEXT;
    chave  TEXT;
    orfaos BIGINT;
BEGIN
    FOREACH alvo IN ARRAY ARRAY['contas_bancarias','condicoes_pagamento','payment_conditions',
                                'plano_contas','centros_custo','restrictions','allocation_bases']
    LOOP
        chave := 'fk_' || alvo || '_enterprise';
        IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = chave) THEN
            EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I FOREIGN KEY (enterprise_id) REFERENCES enterprise(id)',
                           alvo, chave);
        END IF;
        EXECUTE format('CREATE INDEX IF NOT EXISTS %I ON %I (enterprise_id)',
                       'idx_' || alvo || '_enterprise', alvo);
        EXECUTE format('SELECT count(*) FROM %I WHERE enterprise_id IS NULL', alvo) INTO orfaos;
        IF orfaos = 0 THEN
            EXECUTE format('ALTER TABLE %I ALTER COLUMN enterprise_id SET NOT NULL', alvo);
        END IF;
    END LOOP;
END $$;

-- Código de negócio passa a ser único POR EMPRESA nas duas tabelas que ninguém
-- referencia por código. Sem isso, a segunda empresa não consegue criar o seu
-- centro de custo "1" nem a conta contábil "1.1.01" — e um plano de contas que
-- não pode repetir código entre empresas não é um plano de contas.
ALTER TABLE plano_contas  DROP CONSTRAINT IF EXISTS plano_contas_codigo_key;
ALTER TABLE centros_custo DROP CONSTRAINT IF EXISTS centros_custo_codigo_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_plano_contas_empresa_codigo  ON plano_contas  (enterprise_id, codigo);
CREATE UNIQUE INDEX IF NOT EXISTS uq_centros_custo_empresa_codigo ON centros_custo (enterprise_id, codigo);

-- `payment_conditions.code` e `allocation_bases.code` continuam únicos na base
-- toda: são alvo de chave estrangeira (`sales_orders.payment_term_code` e
-- `allocation_base_items.allocation_base_code`). Trocar por chave composta
-- exigiria reescrever o pedido de venda e o rateio. O efeito é só a numeração
-- ser contínua entre empresas — mesmo acordo já feito com `suppliers.code`.
--
-- As filhas seguem o pai por join, sem coluna própria: `allocation_base_items`,
-- `restriction_determinants`, `restriction_dominants`,
-- `payment_condition_installments` e `extrato_bancario`.
