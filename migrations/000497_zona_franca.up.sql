-- Venda para a Zona Franca de Manaus / áreas de livre comércio (SUFRAMA).
-- A regra de ICMS/IPI marcada uses_icms_zona_franca desonera o ICMS (CST 40,
-- motivo 7) e zera PIS/COFINS quando o destinatário tem inscrição SUFRAMA
-- válida. O IPI segue o CST da regra, agora com o código de enquadramento
-- legal (cEnq), que antes saía sempre 999.
ALTER TABLE icms_ipi_tax_params ADD COLUMN ipi_codigo_enquadramento VARCHAR(3);

ALTER TABLE fiscal_exit_items
    ADD COLUMN icms_valor_desonerado   NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN icms_motivo_desoneracao VARCHAR(2),
    ADD COLUMN ipi_codigo_enquadramento VARCHAR(3);
