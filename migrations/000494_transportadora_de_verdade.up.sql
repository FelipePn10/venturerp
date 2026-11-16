-- carrier_code de romaneio, carga, pedido de compra e aviso de recebimento
-- sempre gravou o PORTADOR de cobrança (a tela buscava portadores, e o
-- romaneio exportado lia como portador), embora a tela chamasse o campo de
-- "Transportadora". A transportadora de verdade é o cadastro shipping_carriers.
-- carrier_code fica como está: portador e transportadora são cadastros
-- diferentes e o dado antigo não tem como ser convertido.
ALTER TABLE shipments        ADD COLUMN shipping_carrier_id BIGINT REFERENCES shipping_carriers(id);
ALTER TABLE shipment_loads   ADD COLUMN shipping_carrier_id BIGINT REFERENCES shipping_carriers(id);
ALTER TABLE purchase_orders  ADD COLUMN shipping_carrier_id BIGINT REFERENCES shipping_carriers(id),
                             ADD COLUMN redispatch_shipping_carrier_id BIGINT REFERENCES shipping_carriers(id);
ALTER TABLE receiving_notices ADD COLUMN shipping_carrier_id BIGINT REFERENCES shipping_carriers(id);

-- A trava do banco: vínculo novo só aceita transportadora da mesma empresa.
SELECT instalar_conferencia_de_empresa();
