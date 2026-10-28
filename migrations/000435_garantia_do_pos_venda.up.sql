-- Pós-venda (Fase 3.2): garantia determinada pelo sistema.
--
-- A garantia era um número de dias digitado no item do chamado, contado da data
-- da nota de compra que a pessoa lembrasse de informar. Agora vem do perfil de
-- pós-venda do item (prazo, início e cobertura) e da base instalada (data da
-- venda e da instalação da série) — como a garantia mestre do SAP e a garantia
-- criada na venda da Oracle. Peça sem série é coberta pela NF de saída.

ALTER TABLE item_pos_venda
    ADD COLUMN garantia_meses      INTEGER     NOT NULL DEFAULT 0 CHECK (garantia_meses BETWEEN 0 AND 240),
    ADD COLUMN garantia_inicio     VARCHAR(12) NOT NULL DEFAULT 'VENDA' CHECK (garantia_inicio IN ('VENDA', 'INSTALACAO')),
    ADD COLUMN cobre_pecas         BOOLEAN     NOT NULL DEFAULT TRUE,
    ADD COLUMN cobre_mao_obra      BOOLEAN     NOT NULL DEFAULT TRUE,
    ADD COLUMN cobre_deslocamento  BOOLEAN     NOT NULL DEFAULT FALSE;

-- Garantia estendida (contrato, cortesia comercial) por série.
ALTER TABLE numeros_serie
    ADD COLUMN garantia_estendida_ate    DATE,
    ADD COLUMN garantia_estendida_motivo TEXT;

-- O que o item do chamado usou para decidir a garantia.
ALTER TABLE technical_assistance_call_items
    ADD COLUMN serie_id            BIGINT      REFERENCES numeros_serie(id),
    ADD COLUMN garantia_origem     VARCHAR(10) CHECK (garantia_origem IN ('SERIE', 'NF', 'INFORMADA')),
    ADD COLUMN garantia_inicio     DATE,
    ADD COLUMN cobre_pecas         BOOLEAN,
    ADD COLUMN cobre_mao_obra      BOOLEAN,
    ADD COLUMN cobre_deslocamento  BOOLEAN;
CREATE INDEX idx_ta_call_items_serie ON technical_assistance_call_items(serie_id) WHERE serie_id IS NOT NULL;

SELECT instalar_conferencia_de_empresa();
