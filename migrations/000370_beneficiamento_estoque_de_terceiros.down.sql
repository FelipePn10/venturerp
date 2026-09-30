-- Desfaz o beneficiamento / estoque de terceiros.
--
-- Remove apenas o que a 000370 criou. Nenhuma tabela preexistente foi alterada
-- por ela, então não há nada a restaurar fora deste escopo.
--
-- Ordem inversa da criação: o razão referencia os itens, que referenciam a
-- remessa. Os tipos saem por último, quando ninguém mais os usa.

DROP TABLE IF EXISTS customer_material_movements;
DROP TABLE IF EXISTS customer_material_items;
DROP TABLE IF EXISTS customer_material_remittances;

DROP TYPE IF EXISTS customer_material_scrap_destination_enum;
DROP TYPE IF EXISTS customer_material_movement_type_enum;
DROP TYPE IF EXISTS customer_material_remittance_status_enum;
