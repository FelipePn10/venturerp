-- A origem REMESSA_INDUSTRIALIZACAO tem 24 caracteres: com VARCHAR(20) a NF-e de
-- remessa para industrialização nunca era gravada.
ALTER TABLE public.fiscal_exits ALTER COLUMN source_type TYPE VARCHAR(40);
