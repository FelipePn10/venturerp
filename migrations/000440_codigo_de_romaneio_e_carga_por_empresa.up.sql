-- O número do romaneio e o da carga são sequenciais POR EMPRESA
-- (shipment_sequences / shipment_load_sequences, chave = empresa), mas a
-- unicidade era GLOBAL: a primeira empresa criava o romaneio 1 e a segunda,
-- ao criar o seu romaneio 1, recebia "já existe um registro com os dados
-- informados". A unicidade passa a ser por empresa.

ALTER TABLE shipment_dispatch_boxes DROP CONSTRAINT IF EXISTS shipment_dispatch_boxes_current_load_fkey;

ALTER TABLE shipments DROP CONSTRAINT IF EXISTS shipments_code_key;
ALTER TABLE shipments ADD CONSTRAINT shipments_enterprise_code_key UNIQUE (enterprise_id, code);

ALTER TABLE shipment_loads DROP CONSTRAINT IF EXISTS shipment_loads_code_key;
ALTER TABLE shipment_loads ADD CONSTRAINT shipment_loads_enterprise_code_key UNIQUE (enterprise_id, code);

-- A caixa aponta a carga da mesma empresa.
ALTER TABLE shipment_dispatch_boxes ADD CONSTRAINT shipment_dispatch_boxes_current_load_fkey
    FOREIGN KEY (enterprise_id, current_load) REFERENCES shipment_loads (enterprise_id, code) ON DELETE SET NULL (current_load);

SELECT instalar_conferencia_de_empresa();
