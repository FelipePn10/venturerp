-- Informações complementares (infCpl) da NF-e de saída: o texto do usuário e
-- o que a legislação manda constar — no Simples Nacional, a identificação de
-- optante e o crédito de ICMS (Resolução CGSN 140/2018, art. 59; LC 123, art.
-- 23); nos benefícios, a base legal (dispositivo legal) da regra de ICMS/IPI.
-- A nota não levava infCpl nenhum.
ALTER TABLE fiscal_exits ADD COLUMN IF NOT EXISTS informacoes_complementares text;
