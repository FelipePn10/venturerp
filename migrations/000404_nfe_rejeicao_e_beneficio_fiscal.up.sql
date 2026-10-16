-- Motivo da rejeição da NF-e e código de benefício fiscal (cBenef) por item.
--
-- Rejeitada, a nota só dizia "rejeitada": o motivo da SEFAZ (ex.: 930 — CST com
-- benefício fiscal sem o código) sumia, e quem ia corrigir não sabia o quê.
-- O cBenef é exigido pela SEFAZ do PR, RS, RJ, SC, GO e DF para os CSTs de ICMS
-- com benefício (diferimento, isenção, redução…): sem ele a nota é rejeitada.
ALTER TABLE fiscal_exits ADD COLUMN IF NOT EXISTS motivo_rejeicao TEXT;
ALTER TABLE fiscal_exit_items ADD COLUMN IF NOT EXISTS codigo_beneficio_fiscal VARCHAR(10);
