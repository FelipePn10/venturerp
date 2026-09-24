-- Sexta leva do isolamento multiempresa — o dinheiro.
--
-- A leva 361 isolou os CADASTROS do financeiro (conta bancária, plano de contas,
-- centro de custo, condição de pagamento). Ficaram de fora os DOCUMENTOS, que
-- são o que de fato importa: título a pagar, título a receber, movimento de
-- caixa e apuração de imposto não tinham coluna de empresa nenhuma.
--
-- O efeito prático de deixar assim: com duas empresas na base, a segunda veria
-- — e poderia baixar — os títulos da primeira; o DRE, o aging, a curva ABC e o
-- saldo consolidado somariam as duas. É a pior classe de vazamento do sistema,
-- porque não parece erro: os números simplesmente ficam maiores.
--
-- Escapou da auditoria (`npm run audit:tenant`) porque a sonda compara
-- REGISTROS entre duas empresas e estas tabelas estão vazias no ambiente de
-- desenvolvimento. Tabela vazia passa em qualquer prova de isolamento.
--
-- Produção ainda não tem lançamento financeiro, então o backfill não tem nada a
-- decidir — a coluna nasce antes de existir dado para atribuir a alguém.

ALTER TABLE contas_pagar     ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
ALTER TABLE contas_receber   ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
ALTER TABLE fluxo_caixa      ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;
ALTER TABLE tax_assessments  ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;

UPDATE contas_pagar    SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
UPDATE contas_receber  SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
UPDATE fluxo_caixa     SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
UPDATE tax_assessments SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;

DO $$
DECLARE
    alvo   TEXT;
    chave  TEXT;
    orfaos BIGINT;
BEGIN
    FOREACH alvo IN ARRAY ARRAY['contas_pagar','contas_receber','fluxo_caixa','tax_assessments']
    LOOP
        chave := 'fk_' || alvo || '_enterprise';
        IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = chave) THEN
            EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I FOREIGN KEY (enterprise_id) REFERENCES enterprise(id)', alvo, chave);
        END IF;
        EXECUTE format('CREATE INDEX IF NOT EXISTS %I ON %I (enterprise_id)', 'idx_' || alvo || '_enterprise', alvo);
        EXECUTE format('SELECT count(*) FROM %I WHERE enterprise_id IS NULL', alvo) INTO orfaos;
        IF orfaos = 0 THEN
            EXECUTE format('ALTER TABLE %I ALTER COLUMN enterprise_id SET NOT NULL', alvo);
        END IF;
    END LOOP;
END $$;

-- A apuração é POR EMPRESA: cada uma apura o seu ICMS da mesma competência.
-- Com a chave única global, a segunda empresa a apurar sobrescreveria a
-- primeira — o `ON CONFLICT` do upsert de crédito somaria no registro alheio.
ALTER TABLE tax_assessments DROP CONSTRAINT IF EXISTS tax_assessments_imposto_competencia_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_tax_assessments_empresa_imposto_competencia
    ON tax_assessments (enterprise_id, imposto, competencia);
