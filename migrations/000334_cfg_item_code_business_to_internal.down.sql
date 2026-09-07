-- Correção de dados sem volta: reapontar para o código de negócio devolveria as
-- linhas ao estado órfão. A migração é idempotente, então repetir a subida é
-- seguro.
SELECT 1;
