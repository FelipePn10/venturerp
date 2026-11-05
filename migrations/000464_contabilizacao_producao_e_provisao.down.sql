DROP TABLE stock_closing_production;
ALTER TABLE stock_closings DROP COLUMN obsolete_provision;
ALTER TABLE accounting_posting_params
    DROP COLUMN contabilizar_producao,
    DROP COLUMN producao_processo_account_id,
    DROP COLUMN custos_absorvidos_account_id,
    DROP COLUMN provisao_obsoletos_account_id,
    DROP COLUMN despesa_provisao_account_id;
