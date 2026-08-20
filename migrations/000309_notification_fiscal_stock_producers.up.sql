BEGIN;

ALTER TABLE fiscal_exits ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES enterprise(id);
UPDATE fiscal_exits x SET enterprise_id=ue.enterprise_id FROM user_enterprises ue WHERE x.enterprise_id IS NULL AND x.created_by=ue.user_id AND (SELECT COUNT(*) FROM user_enterprises u2 WHERE u2.user_id=x.created_by)=1;
UPDATE fiscal_exits SET enterprise_id=(SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise)=1;
DO $$ BEGIN IF EXISTS(SELECT 1 FROM fiscal_exits WHERE enterprise_id IS NULL) THEN RAISE EXCEPTION 'Nao foi possivel determinar a empresa de todas as NF-e de saida'; END IF; END $$;
ALTER TABLE fiscal_exits ALTER COLUMN enterprise_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_fiscal_exits_tenant_status ON fiscal_exits(enterprise_id,status,created_at DESC);
ALTER TABLE fiscal_exits ALTER COLUMN status TYPE VARCHAR(30);

CREATE TABLE fiscal_entry_divergences (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), enterprise_id BIGINT NOT NULL REFERENCES enterprise(id) ON DELETE RESTRICT,
 fiscal_entry_id BIGINT NOT NULL REFERENCES fiscal_entries(id) ON DELETE CASCADE,
 fiscal_entry_item_id BIGINT REFERENCES fiscal_entry_items(id) ON DELETE CASCADE,
 divergence_type VARCHAR(20) NOT NULL CHECK(divergence_type IN('FISCAL','QUANTIDADE')),
 expected_value NUMERIC(20,6), actual_value NUMERIC(20,6), description VARCHAR(300) NOT NULL,
 state VARCHAR(12) NOT NULL DEFAULT 'ABERTO' CHECK(state IN('ABERTO','RESOLVIDO','IGNORADO')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), resolved_at TIMESTAMPTZ,
 UNIQUE(enterprise_id,fiscal_entry_item_id,divergence_type)
);
CREATE INDEX idx_fiscal_divergence_tenant ON fiscal_entry_divergences(enterprise_id,state,created_at DESC);

CREATE OR REPLACE FUNCTION notification_fiscal_exit_payload(p_id BIGINT) RETURNS JSONB AS $$
 SELECT jsonb_build_object('nfe_saida_id',x.id,'numero',x.numero_nf,'serie',x.serie,'chave_final',CASE WHEN x.chave_acesso IS NULL THEN NULL ELSE RIGHT(x.chave_acesso,8) END,'destinatario',x.razao_social_destinatario,'emissao',x.data_emissao,'saida',x.data_saida,'situacao',x.status,'operacao',x.natureza_operacao,'valor_produtos',x.valor_produtos,'frete',x.valor_frete,'seguro',x.valor_seguro,'desconto',x.valor_desconto,'ipi',x.valor_ipi,'icms',x.valor_icms,'pis',x.valor_pis,'cofins',x.valor_cofins,'total',x.valor_total,'itens',COALESCE((SELECT jsonb_agg(jsonb_build_object('sequencia',i.sequence,'item_codigo',i.item_code,'descricao',i.description,'ncm',i.ncm,'cfop',i.cfop,'quantidade',i.quantity,'preco_unitario',i.unit_price,'total',i.total_price) ORDER BY i.sequence) FROM fiscal_exit_items i WHERE i.fiscal_exit_id=x.id),'[]'::jsonb),'link','/fiscal/exits/'||x.id::text) FROM fiscal_exits x WHERE x.id=p_id
$$ LANGUAGE SQL STABLE;
CREATE OR REPLACE FUNCTION notification_fiscal_entry_payload(p_id BIGINT) RETURNS JSONB AS $$
 SELECT jsonb_build_object('nfe_entrada_id',x.id,'numero',x.numero_nf,'serie',x.serie,'chave_final',CASE WHEN x.chave_acesso IS NULL THEN NULL ELSE RIGHT(x.chave_acesso,8) END,'emitente',x.razao_social_emitente,'emissao',x.data_emissao,'entrada',x.data_entrada,'situacao',x.status,'operacao',x.tipo_documento,'valor_produtos',x.valor_produtos,'frete',x.valor_frete,'seguro',x.valor_seguro,'desconto',x.valor_desconto,'ipi',x.valor_ipi,'icms',x.valor_icms,'pis',x.valor_pis,'cofins',x.valor_cofins,'total',x.valor_total,'itens',COALESCE((SELECT jsonb_agg(jsonb_build_object('sequencia',i.sequence,'item_codigo',i.item_code,'codigo_fornecedor',i.supplier_item_identifier,'descricao',i.description,'um',i.uom,'ncm',i.ncm,'cfop',i.cfop,'quantidade',i.quantity,'preco_unitario',i.unit_price,'total',i.total_price) ORDER BY i.sequence) FROM fiscal_entry_items i WHERE i.fiscal_entry_id=x.id),'[]'::jsonb),'link','/fiscal/entries/'||x.id::text) FROM fiscal_entries x WHERE x.id=p_id
$$ LANGUAGE SQL STABLE;

CREATE OR REPLACE FUNCTION notification_fiscal_exit_event() RETURNS TRIGGER AS $$ DECLARE v_event_key TEXT; BEGIN
 IF TG_OP='INSERT' THEN v_event_key:='FISCAL_NFE_SAIDA_CRIADA';
 ELSIF OLD.status IS NOT DISTINCT FROM NEW.status THEN RETURN NEW;
 ELSE v_event_key:=CASE NEW.status WHEN 'AGUARDANDO_AUTORIZACAO' THEN 'FISCAL_NFE_SAIDA_AGUARDANDO_AUTORIZACAO' WHEN 'AUTHORIZED' THEN 'FISCAL_NFE_SAIDA_AUTORIZADA' WHEN 'REJECTED' THEN 'FISCAL_NFE_SAIDA_REJEITADA' WHEN 'CANCELLED' THEN 'FISCAL_NFE_SAIDA_CANCELADA' END; END IF;
 IF v_event_key IS NOT NULL THEN INSERT INTO notification_outbox(enterprise_id,event_key,event_version,aggregate_type,aggregate_internal_id,aggregate_public_id,payload,deduplication_key,originator_user_id,available_at) VALUES(NEW.enterprise_id,v_event_key,1,'NFE_SAIDA',NEW.id::text,NEW.numero_nf::text,notification_fiscal_exit_payload(NEW.id),'nfe_saida:'||NEW.id::text||':'||v_event_key,NEW.created_by,CASE WHEN TG_OP='INSERT' THEN NOW()+INTERVAL '30 seconds' ELSE NOW() END) ON CONFLICT(enterprise_id,event_key,deduplication_key) DO UPDATE SET payload=EXCLUDED.payload; END IF; RETURN NEW;
END $$ LANGUAGE plpgsql;
CREATE TRIGGER trg_notification_fiscal_exit AFTER INSERT OR UPDATE OF status ON fiscal_exits FOR EACH ROW EXECUTE FUNCTION notification_fiscal_exit_event();
CREATE OR REPLACE FUNCTION notification_fiscal_exit_item_event() RETURNS TRIGGER AS $$ BEGIN UPDATE notification_outbox SET payload=notification_fiscal_exit_payload(NEW.fiscal_exit_id) WHERE aggregate_type='NFE_SAIDA' AND aggregate_internal_id=NEW.fiscal_exit_id::text AND state='PENDENTE'; RETURN NEW; END $$ LANGUAGE plpgsql;
CREATE TRIGGER trg_notification_fiscal_exit_item AFTER INSERT OR UPDATE ON fiscal_exit_items FOR EACH ROW EXECUTE FUNCTION notification_fiscal_exit_item_event();

CREATE OR REPLACE FUNCTION notification_fiscal_entry_event() RETURNS TRIGGER AS $$ DECLARE v_event_key TEXT; BEGIN
 IF TG_OP='INSERT' THEN v_event_key:='FISCAL_NFE_ENTRADA_IMPORTADA'; ELSIF OLD.status IS NOT DISTINCT FROM NEW.status THEN RETURN NEW; ELSE v_event_key:=CASE NEW.status WHEN 'APPROVED' THEN 'FISCAL_NFE_ENTRADA_APROVADA' WHEN 'CANCELLED' THEN 'FISCAL_NFE_ENTRADA_CANCELADA' END; END IF;
 IF v_event_key IS NOT NULL THEN INSERT INTO notification_outbox(enterprise_id,event_key,event_version,aggregate_type,aggregate_internal_id,aggregate_public_id,payload,deduplication_key,originator_user_id,available_at) VALUES(NEW.enterprise_id,v_event_key,1,'NFE_ENTRADA',NEW.id::text,NEW.numero_nf::text,notification_fiscal_entry_payload(NEW.id),'nfe_entrada:'||NEW.id::text||':'||v_event_key,NEW.created_by,CASE WHEN TG_OP='INSERT' THEN NOW()+INTERVAL '30 seconds' ELSE NOW() END) ON CONFLICT(enterprise_id,event_key,deduplication_key) DO UPDATE SET payload=EXCLUDED.payload; END IF; RETURN NEW;
END $$ LANGUAGE plpgsql;
CREATE TRIGGER trg_notification_fiscal_entry AFTER INSERT OR UPDATE OF status ON fiscal_entries FOR EACH ROW EXECUTE FUNCTION notification_fiscal_entry_event();

CREATE OR REPLACE FUNCTION notification_fiscal_entry_item_event() RETURNS TRIGGER AS $$ DECLARE entry_row RECORD; po_item RECORD; BEGIN
 SELECT * INTO entry_row FROM fiscal_entries WHERE id=NEW.fiscal_entry_id;
 UPDATE notification_outbox SET payload=notification_fiscal_entry_payload(NEW.fiscal_entry_id) WHERE aggregate_type='NFE_ENTRADA' AND aggregate_internal_id=NEW.fiscal_entry_id::text AND state='PENDENTE';
 IF NEW.item_code IS NULL THEN INSERT INTO notification_outbox(enterprise_id,event_key,event_version,aggregate_type,aggregate_internal_id,aggregate_public_id,payload,deduplication_key,originator_user_id) VALUES(entry_row.enterprise_id,'FISCAL_NFE_ENTRADA_ITEM_NAO_IDENTIFICADO',1,'ITEM_NFE_ENTRADA',NEW.id::text,entry_row.numero_nf::text,jsonb_build_object('nfe_entrada_id',entry_row.id,'item_sequencia',NEW.sequence,'codigo_fornecedor',NEW.supplier_item_identifier,'descricao',NEW.description,'link','/fiscal/entries/'||entry_row.id::text),'item_nfe_entrada:'||NEW.id::text||':nao_identificado',entry_row.created_by) ON CONFLICT(enterprise_id,event_key,deduplication_key) DO NOTHING; RETURN NEW; END IF;
 IF entry_row.purchase_order_code IS NOT NULL THEN SELECT * INTO po_item FROM purchase_order_items WHERE purchase_order_code=entry_row.purchase_order_code AND item_code=NEW.item_code AND is_active ORDER BY sequence LIMIT 1;
   IF FOUND AND NEW.quantity IS DISTINCT FROM po_item.requested_qty THEN INSERT INTO fiscal_entry_divergences(enterprise_id,fiscal_entry_id,fiscal_entry_item_id,divergence_type,expected_value,actual_value,description) VALUES(entry_row.enterprise_id,entry_row.id,NEW.id,'QUANTIDADE',po_item.requested_qty,NEW.quantity,'Quantidade da NF-e diverge do pedido de compra') ON CONFLICT DO NOTHING; INSERT INTO notification_outbox(enterprise_id,event_key,event_version,aggregate_type,aggregate_internal_id,payload,deduplication_key,originator_user_id) VALUES(entry_row.enterprise_id,'FISCAL_NFE_ENTRADA_DIVERGENCIA_QUANTIDADE',1,'ITEM_NFE_ENTRADA',NEW.id::text,jsonb_build_object('nfe_entrada_id',entry_row.id,'item_codigo',NEW.item_code,'esperado',po_item.requested_qty,'recebido',NEW.quantity,'link','/fiscal/entries/'||entry_row.id::text),'item_nfe_entrada:'||NEW.id::text||':divergencia_quantidade',entry_row.created_by) ON CONFLICT(enterprise_id,event_key,deduplication_key) DO NOTHING; END IF;
   IF FOUND AND (NEW.unit_price IS DISTINCT FROM po_item.unit_price OR NULLIF(NEW.ncm,'') IS NULL) THEN INSERT INTO fiscal_entry_divergences(enterprise_id,fiscal_entry_id,fiscal_entry_item_id,divergence_type,expected_value,actual_value,description) VALUES(entry_row.enterprise_id,entry_row.id,NEW.id,'FISCAL',po_item.unit_price,NEW.unit_price,'Preço ou classificação fiscal diverge do pedido') ON CONFLICT DO NOTHING; INSERT INTO notification_outbox(enterprise_id,event_key,event_version,aggregate_type,aggregate_internal_id,payload,deduplication_key,originator_user_id) VALUES(entry_row.enterprise_id,'FISCAL_NFE_ENTRADA_DIVERGENCIA_FISCAL',1,'ITEM_NFE_ENTRADA',NEW.id::text,jsonb_build_object('nfe_entrada_id',entry_row.id,'item_codigo',NEW.item_code,'preco_esperado',po_item.unit_price,'preco_recebido',NEW.unit_price,'ncm',NEW.ncm,'link','/fiscal/entries/'||entry_row.id::text),'item_nfe_entrada:'||NEW.id::text||':divergencia_fiscal',entry_row.created_by) ON CONFLICT(enterprise_id,event_key,deduplication_key) DO NOTHING; END IF;
 END IF; RETURN NEW;
END $$ LANGUAGE plpgsql;
CREATE TRIGGER trg_notification_fiscal_entry_item AFTER INSERT OR UPDATE ON fiscal_entry_items FOR EACH ROW EXECUTE FUNCTION notification_fiscal_entry_item_event();

COMMIT;
