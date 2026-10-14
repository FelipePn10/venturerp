-- Responsáveis de garantia (assistência técnica) por empresa. O cadastro não
-- tinha empresa: a lista mostrava os de todas e o atendimento aceitava o de
-- outra. Convenção do atendimento: enterprise_code.
--   * responsável usado em atendimentos de várias empresas fica com a menor; as
--     demais recebem cópia com código novo e os atendimentos delas passam para
--     a cópia;
--   * responsável sem uso vai para todas as empresas (cópias com código novo).
ALTER TABLE public.technical_assistance_warranty_responsibles ADD COLUMN IF NOT EXISTS enterprise_code BIGINT;

CREATE TEMP TABLE _uso_resp ON COMMIT DROP AS
SELECT DISTINCT warranty_responsible_code AS code, enterprise_code
  FROM public.technical_assistance_calls WHERE warranty_responsible_code IS NOT NULL;
INSERT INTO _uso_resp
SELECT w.code, e.code FROM public.technical_assistance_warranty_responsibles w CROSS JOIN public.enterprise e
 WHERE NOT EXISTS (SELECT 1 FROM _uso_resp u WHERE u.code = w.code);

UPDATE public.technical_assistance_warranty_responsibles w SET enterprise_code = d.dono
  FROM (SELECT DISTINCT ON (u.code) u.code, u.enterprise_code AS dono
          FROM _uso_resp u JOIN public.enterprise e ON e.code = u.enterprise_code
         ORDER BY u.code, e.id) d
 WHERE d.code = w.code;

CREATE TEMP TABLE _mapa_resp ON COMMIT DROP AS
SELECT u.code AS antigo, u.enterprise_code, nextval('public.technical_assistance_warranty_responsibles_code_seq') AS novo
  FROM _uso_resp u JOIN public.technical_assistance_warranty_responsibles w ON w.code = u.code
 WHERE u.enterprise_code <> w.enterprise_code
   AND EXISTS (SELECT 1 FROM public.enterprise e WHERE e.code = u.enterprise_code);

INSERT INTO public.technical_assistance_warranty_responsibles
       (code, name, employee_code, customer_code, email, phone, is_active, created_at, updated_at, created_by, enterprise_code)
SELECT m.novo, w.name, w.employee_code, w.customer_code, w.email, w.phone, w.is_active, w.created_at, NOW(), w.created_by, m.enterprise_code
  FROM _mapa_resp m JOIN public.technical_assistance_warranty_responsibles w ON w.code = m.antigo;

UPDATE public.technical_assistance_calls c SET warranty_responsible_code = m.novo
  FROM _mapa_resp m WHERE m.antigo = c.warranty_responsible_code AND m.enterprise_code = c.enterprise_code;

DELETE FROM public.technical_assistance_warranty_responsibles WHERE enterprise_code IS NULL; -- instalação sem empresa
ALTER TABLE public.technical_assistance_warranty_responsibles ALTER COLUMN enterprise_code SET NOT NULL;
CREATE INDEX IF NOT EXISTS ix_ta_warranty_responsibles_enterprise ON public.technical_assistance_warranty_responsibles (enterprise_code, name);
