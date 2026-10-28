DROP TABLE IF EXISTS parametros_comerciais;
DELETE FROM aprovacao_politicas WHERE tipo = 'PEDIDO_POLITICA_COMERCIAL';
ALTER TABLE aprovacao_politicas DROP CONSTRAINT IF EXISTS aprovacao_politicas_tipo_check;
ALTER TABLE aprovacao_politicas ADD CONSTRAINT aprovacao_politicas_tipo_check
    CHECK (tipo IN ('LIBERACAO_CREDITO', 'ORCAMENTO_POLITICA_COMERCIAL'));
SELECT instalar_conferencia_de_empresa();
