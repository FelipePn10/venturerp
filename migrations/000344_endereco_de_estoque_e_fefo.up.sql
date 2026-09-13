-- Endereçamento de estoque e base para FEFO.
--
-- Até aqui o sistema sabia QUANTO havia de um item num almoxarifado, mas não
-- ONDE. `manufacturing_warehouse_addresses` já cadastrava os endereços, porém
-- nem `stock_balances` nem `stock_movements` os referenciavam — então não havia
-- saldo por endereço, separação, transferência interna nem contagem por
-- endereço. O Focco tem saldo por endereço (FEST0332) e o SAP desce até o bin;
-- era a maior lacuna do módulo.
--
-- O endereço entra na CHAVE do saldo: o mesmo item, no mesmo almoxarifado, em
-- dois endereços, são duas linhas. Endereço vazio ('') significa "almoxarifado
-- sem endereçamento", que é como toda a base existente fica — nenhuma linha
-- muda de valor com esta migração.

ALTER TABLE stock_balances     ADD COLUMN IF NOT EXISTS address VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE stock_lot_balances ADD COLUMN IF NOT EXISTS address VARCHAR(100) NOT NULL DEFAULT '';

-- Origem e destino: uma transferência entre endereços é UM movimento, com o
-- endereço de saída e o de entrada. Guardar dois movimentos espelhados
-- duplicaria a quantidade nos relatórios de giro.
ALTER TABLE stock_movements ADD COLUMN IF NOT EXISTS address    VARCHAR(100);
ALTER TABLE stock_movements ADD COLUMN IF NOT EXISTS address_to VARCHAR(100);

-- Validade por lote: é o que ordena o FEFO. Sem ela só havia FIFO por data de
-- recebimento, e só dentro do plano de corte.
ALTER TABLE stock_lots ADD COLUMN IF NOT EXISTS expires_at DATE;

-- A unicidade do saldo era (empresa, item, máscara, almoxarifado) — sem
-- endereço. Ela precisa sair: com endereçamento, o mesmo item no mesmo
-- almoxarifado em dois endereços são duas linhas legítimas, e o índice antigo
-- recusava a segunda com violação de unicidade.
DROP INDEX IF EXISTS uq_stock_balances_tenant_item_warehouse;

-- Consolida eventuais duplicatas antes de criar o índice novo.
WITH agrupado AS (
    SELECT enterprise_id, item_code, mask, warehouse_id, address,
           MIN(id) AS manter,
           SUM(quantity) AS soma_qtd,
           SUM(total_cost) AS soma_valor,
           SUM(reserved_qty) AS soma_reservado
      FROM stock_balances
     GROUP BY enterprise_id, item_code, mask, warehouse_id, address
    HAVING COUNT(*) > 1
)
UPDATE stock_balances b
   SET quantity = a.soma_qtd,
       total_cost = a.soma_valor,
       reserved_qty = a.soma_reservado,
       avg_cost = CASE WHEN a.soma_qtd > 0 THEN a.soma_valor / a.soma_qtd ELSE b.avg_cost END
  FROM agrupado a
 WHERE b.id = a.manter;

DELETE FROM stock_balances b
 USING (
    SELECT MIN(id) AS manter, enterprise_id, item_code, mask, warehouse_id, address
      FROM stock_balances
     GROUP BY enterprise_id, item_code, mask, warehouse_id, address
    HAVING COUNT(*) > 1
 ) a
 WHERE b.enterprise_id = a.enterprise_id AND b.item_code = a.item_code
   AND b.mask = a.mask AND b.warehouse_id = a.warehouse_id AND b.address = a.address
   AND b.id <> a.manter;

CREATE UNIQUE INDEX IF NOT EXISTS uq_stock_balances_endereco
    ON stock_balances (enterprise_id, item_code, mask, warehouse_id, address);

DROP INDEX IF EXISTS uq_stock_lot_balances_tenant;
CREATE UNIQUE INDEX IF NOT EXISTS uq_stock_lot_balances_endereco
    ON stock_lot_balances (enterprise_id, item_code, mask, warehouse_id, lot, address)
 WHERE enterprise_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_stock_balances_endereco
    ON stock_balances (enterprise_id, warehouse_id, address);
-- Ordem do FEFO: vence antes sai antes; sem validade, cai para o recebimento.
CREATE INDEX IF NOT EXISTS idx_stock_lots_fefo
    ON stock_lots (enterprise_id, item_code, mask, expires_at, received_at);
