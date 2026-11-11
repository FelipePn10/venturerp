-- Versão da estrutura (cabeçalho da BOM) passa a governar a produção.
--
-- Até aqui o cabeçalho era um registro solto: aceitava qualquer troca de
-- situação, permitia duas versões aprovadas ao mesmo tempo e nada usava a
-- versão aprovada. Modelo adotado (o do SAP — utilização/situação da BOM — e o
-- da revisão de estrutura do Protheus):
--   * uma versão APROVADA por item + máscara + tipo (MBOM fabricação / EBOM
--     engenharia); aprovar a nova torna a anterior OBSOLETA;
--   * a aprovação guarda a FOTO das linhas aprovadas (rastreabilidade e
--     "o que mudou desde a aprovação");
--   * item com controle de versão e sem MBOM aprovada não está pronto para
--     produzir (prontidão do item).

ALTER TABLE bom_headers
    ADD COLUMN IF NOT EXISTS approved_by  UUID,
    ADD COLUMN IF NOT EXISTS approved_at  TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS obsoleted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS notes        TEXT;

-- Duas aprovadas para o mesmo item/máscara/tipo: fica a de maior versão.
UPDATE bom_headers h
   SET status = 'OBSOLETE', obsoleted_at = NOW(), updated_at = NOW()
 WHERE h.status = 'APPROVED' AND h.is_active
   AND EXISTS (SELECT 1 FROM bom_headers o
                WHERE o.enterprise_id = h.enterprise_id AND o.item_code = h.item_code
                  AND COALESCE(o.mask, '') = COALESCE(h.mask, '') AND o.bom_type = h.bom_type
                  AND o.status = 'APPROVED' AND o.is_active AND o.version > h.version);

CREATE UNIQUE INDEX IF NOT EXISTS idx_bom_headers_uma_aprovada
    ON bom_headers (enterprise_id, item_code, COALESCE(mask, ''), bom_type)
 WHERE status = 'APPROVED' AND is_active;

CREATE TABLE IF NOT EXISTS bom_header_lines (
    id                BIGSERIAL PRIMARY KEY,
    enterprise_id     BIGINT        NOT NULL REFERENCES enterprise(id),
    bom_header_id     BIGINT        NOT NULL REFERENCES bom_headers(id) ON DELETE CASCADE,
    structure_id      BIGINT,
    child_code        BIGINT        NOT NULL,
    parent_mask       VARCHAR(500),
    sequence          INTEGER,
    quantity          NUMERIC(15,4) NOT NULL,
    unit_of_measurement VARCHAR(10),
    loss_percentage   NUMERIC(5,2),
    quantity_formula  VARCHAR(500),
    start_date        DATE,
    end_date          DATE,
    is_coproduct      BOOLEAN       NOT NULL DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_bom_header_lines_header ON bom_header_lines (enterprise_id, bom_header_id);

COMMENT ON TABLE bom_header_lines IS
  'Foto das linhas da estrutura no momento em que a versão foi aprovada. Não é editada.';

-- Vínculo novo (bom_header_id) com conferência de empresa, como toda tabela filha.
SELECT instalar_conferencia_de_empresa();
