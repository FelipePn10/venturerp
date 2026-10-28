UPDATE notification_event_catalog SET producer_status='FUTURO', producer_description='Produtor ainda não conectado ao processo operacional.'
 WHERE event_key IN ('CADASTRO_ITEM_SEM_CLASSIFICACAO_FISCAL','CADASTRO_ITEM_COMPRADO_SEM_FORNECEDOR','CADASTRO_ITEM_FABRICADO_SEM_ESTRUTURA',
                     'CADASTRO_ITEM_FABRICADO_SEM_ROTEIRO','CADASTRO_ITEM_INATIVO_COM_DEMANDA_ABERTA');
DROP TRIGGER IF EXISTS trg_item_utilizavel ON sales_order_items;
DROP TRIGGER IF EXISTS trg_item_utilizavel ON sales_quotation_items;
DROP TRIGGER IF EXISTS trg_item_utilizavel ON purchase_order_items;
DROP TRIGGER IF EXISTS trg_item_utilizavel ON purchase_requisition_items;
DROP TRIGGER IF EXISTS trg_item_utilizavel ON production_orders;
DROP TRIGGER IF EXISTS trg_item_utilizavel ON fiscal_exit_items;
DROP FUNCTION IF EXISTS item_conferir_utilizavel();
DROP TRIGGER IF EXISTS trg_items_historico ON items;
DROP FUNCTION IF EXISTS item_registrar_historico();
DROP FUNCTION IF EXISTS item_ator();
DROP TRIGGER IF EXISTS trg_item_historico_imutavel ON item_historico;
DROP TABLE IF EXISTS item_historico;
DROP FUNCTION IF EXISTS item_historico_imutavel();
DROP TABLE IF EXISTS item_regras_campo;
DROP TABLE IF EXISTS item_visoes_liberadas;
DROP TABLE IF EXISTS item_governanca;
DROP TABLE IF EXISTS item_governanca_parametros;
SELECT instalar_conferencia_de_empresa();
