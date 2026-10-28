-- Volta para os dois cadastros: as referências financeiras apontam de novo para
-- o centro financeiro de origem; os centros criados pela migração saem (se
-- ninguém do lado industrial passou a usá-los, o DELETE passa).
DELETE FROM notification_alerts WHERE event_key IN ('FINANCEIRO_CENTRO_ORCAMENTO_ESTOURADO', 'COMPRAS_REQUISICAO_AGUARDANDO_CENTRO');
DELETE FROM notification_outbox WHERE event_key IN ('FINANCEIRO_CENTRO_ORCAMENTO_ESTOURADO', 'COMPRAS_REQUISICAO_AGUARDANDO_CENTRO');
DELETE FROM enterprise_notification_subscriptions WHERE event_key IN ('FINANCEIRO_CENTRO_ORCAMENTO_ESTOURADO', 'COMPRAS_REQUISICAO_AGUARDANDO_CENTRO');
DELETE FROM notification_event_catalog WHERE event_key IN ('FINANCEIRO_CENTRO_ORCAMENTO_ESTOURADO', 'COMPRAS_REQUISICAO_AGUARDANDO_CENTRO');
DROP TABLE IF EXISTS requisicao_aprovacoes_centro;
ALTER TABLE plano_contas DROP COLUMN IF EXISTS exige_centro_custo;
DROP TABLE IF EXISTS cost_center_orcamentos;

ALTER TABLE centros_custo_legado RENAME TO centros_custo;

ALTER TABLE fiscal_entry_items DROP CONSTRAINT IF EXISTS fiscal_entry_items_centro_custo_id_fkey;
ALTER TABLE fiscal_entry_installment_allocations DROP CONSTRAINT IF EXISTS fiscal_entry_installment_allocations_centro_custo_id_fkey;
ALTER TABLE contas_pagar_rateios DROP CONSTRAINT IF EXISTS contas_pagar_rateios_centro_custo_id_fkey;

UPDATE fiscal_entry_items t SET centro_custo_id = m.antigo_id FROM centro_custo_migracao m WHERE t.centro_custo_id = m.novo_id;
UPDATE fiscal_entry_installment_allocations t SET centro_custo_id = m.antigo_id FROM centro_custo_migracao m WHERE t.centro_custo_id = m.novo_id;
UPDATE contas_pagar_rateios t SET centro_custo_id = m.antigo_id FROM centro_custo_migracao m WHERE t.centro_custo_id = m.novo_id;
UPDATE contas_pagar t SET centro_custo_id = m.antigo_id FROM centro_custo_migracao m WHERE t.centro_custo_id = m.novo_id;
UPDATE contas_receber t SET centro_custo_id = m.antigo_id FROM centro_custo_migracao m WHERE t.centro_custo_id = m.novo_id;
UPDATE fiscal_freight_allocations t SET centro_custo_id = m.antigo_id FROM centro_custo_migracao m WHERE t.centro_custo_id = m.novo_id;
UPDATE cost_overhead_rules t SET centro_custo_id = m.antigo_id FROM centro_custo_migracao m WHERE t.centro_custo_id = m.novo_id;
UPDATE accounting_account_refs t SET cost_center_id = m.antigo_id FROM centro_custo_migracao m WHERE t.cost_center_id = m.novo_id;

-- Centro financeiro criado depois da migração (sem origem) não tem para onde
-- voltar: a referência fica vazia.
UPDATE fiscal_entry_items SET centro_custo_id = NULL WHERE centro_custo_id IS NOT NULL AND centro_custo_id NOT IN (SELECT id FROM centros_custo);
UPDATE fiscal_entry_installment_allocations SET centro_custo_id = NULL WHERE centro_custo_id IS NOT NULL AND centro_custo_id NOT IN (SELECT id FROM centros_custo);
UPDATE contas_pagar_rateios SET centro_custo_id = NULL WHERE centro_custo_id IS NOT NULL AND centro_custo_id NOT IN (SELECT id FROM centros_custo);

ALTER TABLE fiscal_entry_items ADD CONSTRAINT fiscal_entry_items_centro_custo_id_fkey
    FOREIGN KEY (centro_custo_id) REFERENCES centros_custo(id);
ALTER TABLE fiscal_entry_installment_allocations ADD CONSTRAINT fiscal_entry_installment_allocations_centro_custo_id_fkey
    FOREIGN KEY (centro_custo_id) REFERENCES centros_custo(id);
ALTER TABLE contas_pagar_rateios ADD CONSTRAINT contas_pagar_rateios_centro_custo_id_fkey
    FOREIGN KEY (centro_custo_id) REFERENCES centros_custo(id);

CREATE TEMP TABLE cc_criados_na_migracao ON COMMIT DROP AS SELECT novo_id FROM centro_custo_migracao WHERE criado;
DROP TABLE IF EXISTS centro_custo_migracao;
DROP TRIGGER IF EXISTS trg_cost_center_gestao_inicial ON cost_centers;
DROP FUNCTION IF EXISTS cost_center_gestao_inicial();
DROP TABLE IF EXISTS cost_center_gestao;
DELETE FROM cost_centers WHERE id IN (SELECT novo_id FROM cc_criados_na_migracao);

SELECT instalar_conferencia_de_empresa();
