DROP TABLE stock_cnpj_transfer_lines;
DROP TABLE stock_cnpj_transfers;
DROP TABLE stock_consignment_settlements;
DROP TABLE stock_consignments;
ALTER TABLE stock_parameters DROP COLUMN consignment_warehouse_id, DROP COLUMN consignment_days;
SELECT instalar_conferencia_de_empresa();
