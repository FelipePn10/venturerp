-- Contas do fechamento de estoque: produção em processo e provisão de obsoletos.
ALTER TABLE accounting_posting_params
    ADD COLUMN contabilizar_producao boolean NOT NULL DEFAULT FALSE,
    ADD COLUMN producao_processo_account_id bigint REFERENCES accounting_accounts(id),
    ADD COLUMN custos_absorvidos_account_id bigint REFERENCES accounting_accounts(id),
    ADD COLUMN provisao_obsoletos_account_id bigint REFERENCES accounting_accounts(id),
    ADD COLUMN despesa_provisao_account_id bigint REFERENCES accounting_accounts(id);

-- Saldo da provisão depois do fechamento: o mês seguinte lança só a variação.
ALTER TABLE stock_closings ADD COLUMN obsolete_provision numeric(18,2) NOT NULL DEFAULT 0;

-- O que o fechamento levou à contabilidade por ordem. A soma dos fechamentos em
-- vigor é o saldo da ordem em produção em processo.
CREATE TABLE stock_closing_production (
    closing_id          bigint        NOT NULL REFERENCES stock_closings(id) ON DELETE CASCADE,
    production_order_id bigint        NOT NULL,
    consumed_value      numeric(18,2) NOT NULL DEFAULT 0,
    delivered_value     numeric(18,2) NOT NULL DEFAULT 0,
    absorbed_value      numeric(18,2) NOT NULL DEFAULT 0,
    variance_value      numeric(18,2) NOT NULL DEFAULT 0,
    PRIMARY KEY (closing_id, production_order_id)
);
CREATE INDEX idx_stock_closing_production_ordem ON stock_closing_production (production_order_id);
