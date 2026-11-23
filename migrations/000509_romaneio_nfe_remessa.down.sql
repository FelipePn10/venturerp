-- Não volta a VARCHAR(20) com remessa gravada: a conversão falharia.
ALTER TABLE public.fiscal_exits ALTER COLUMN source_type TYPE VARCHAR(20);
