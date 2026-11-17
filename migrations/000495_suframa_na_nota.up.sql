-- Inscrição SUFRAMA do destinatário (ISUF da NF-e), guardada como foto do
-- momento da nota, igual à IE: o documento fiscal não muda com o cadastro.
ALTER TABLE fiscal_exits ADD COLUMN suframa_destinatario VARCHAR(20);
