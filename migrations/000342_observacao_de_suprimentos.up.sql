-- A pasta Suprimentos do cadastro de item tinha um campo de observação na tela
-- sem coluna no banco: o texto digitado sumia no salvar, sem erro. Comercial e
-- Contábil já tinham a sua; Suprimentos passa a ter também.
ALTER TABLE items ADD COLUMN IF NOT EXISTS supplies_notes TEXT;
