-- Cópia do XML autorizado das notas emitidas. A Focus não garante por quanto
-- tempo guarda os arquivos e o emitente tem de guardá-los por 5 anos; a cópia
-- também deixa o usuário ver e copiar o XML sem depender da emissora.
ALTER TABLE public.fiscal_exits ADD COLUMN IF NOT EXISTS xml_content TEXT;
ALTER TABLE public.nfse ADD COLUMN IF NOT EXISTS xml_content TEXT;
