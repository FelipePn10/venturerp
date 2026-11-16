-- NF-e: responsável pelo frete, transportadora e local de entrega.
-- Transportadora e local de entrega são guardados como foto do momento da nota:
-- o documento fiscal não muda se o cadastro mudar depois.
ALTER TABLE fiscal_exits
    ADD COLUMN tipo_frete          VARCHAR(10),
    ADD COLUMN shipping_carrier_id BIGINT,
    ADD COLUMN transportador       JSONB,
    ADD COLUMN local_entrega       JSONB,
    ADD CONSTRAINT fiscal_exits_tipo_frete_chk
        CHECK (tipo_frete IS NULL OR tipo_frete IN ('CIF', 'FOB', 'TERCEIROS', 'SEM_FRETE'));
