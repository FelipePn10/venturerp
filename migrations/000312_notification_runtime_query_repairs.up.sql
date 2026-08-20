BEGIN;

CREATE OR REPLACE FUNCTION notification_sync_configured_item(p_item_code BIGINT) RETURNS VOID AS $$
DECLARE item_row RECORD; has_active_question BOOLEAN; next_cycle INTEGER;
BEGIN
    SELECT id,enterprise_id,business_code,pdm_description_technique,engineering_item_base_code,nature,created_by INTO item_row FROM items WHERE code=p_item_code;
    IF NOT FOUND OR item_row.nature<>1 THEN RETURN; END IF;
    SELECT EXISTS(SELECT 1 FROM cfg_item_characteristics ic JOIN cfg_characteristics c ON c.id=ic.characteristic_id WHERE ic.item_code=p_item_code AND c.is_active) INTO has_active_question;
    IF has_active_question THEN
        UPDATE notification_alerts SET state='RESOLVIDO',resolved_at=NOW(),resolution_reason='Primeira característica ativa associada' WHERE enterprise_id=item_row.enterprise_id AND event_key='CADASTRO_ITEM_CONFIGURADO_SEM_PERGUNTAS' AND aggregate_internal_id=item_row.id::text AND state='ABERTO';
        UPDATE notification_outbox SET state='CANCELADO',processed_at=NOW(),lease_owner=NULL,lease_until=NULL WHERE enterprise_id=item_row.enterprise_id AND event_key='CADASTRO_ITEM_CONFIGURADO_SEM_PERGUNTAS' AND aggregate_internal_id=item_row.id::text AND state IN ('PENDENTE','FALHOU');
        RETURN;
    END IF;
    IF EXISTS(SELECT 1 FROM notification_alerts WHERE enterprise_id=item_row.enterprise_id AND event_key='CADASTRO_ITEM_CONFIGURADO_SEM_PERGUNTAS' AND aggregate_internal_id=item_row.id::text AND state='ABERTO') THEN RETURN; END IF;
    SELECT COALESCE(MAX(cycle),0)+1 INTO next_cycle FROM notification_alerts WHERE enterprise_id=item_row.enterprise_id AND event_key='CADASTRO_ITEM_CONFIGURADO_SEM_PERGUNTAS' AND aggregate_internal_id=item_row.id::text;
    INSERT INTO notification_outbox(enterprise_id,event_key,event_version,aggregate_type,aggregate_internal_id,aggregate_public_id,payload,deduplication_key,originator_user_id)
    VALUES(item_row.enterprise_id,'CADASTRO_ITEM_CONFIGURADO_SEM_PERGUNTAS',1,'ITEM',item_row.id::text,item_row.business_code,jsonb_build_object('codigo',item_row.business_code,'descricao',item_row.pdm_description_technique,'item_base',item_row.engineering_item_base_code,'criador_usuario_id',item_row.created_by,'data',NOW(),'link','/items/'||item_row.business_code),'item:'||item_row.id::text||':ciclo:'||next_cycle::text,item_row.created_by)
    ON CONFLICT(enterprise_id,event_key,deduplication_key) DO NOTHING;
END;
$$ LANGUAGE plpgsql;

COMMIT;
