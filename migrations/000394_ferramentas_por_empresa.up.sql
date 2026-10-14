-- Ferramentas por empresa.
--
-- O cadastro de ferramentas (e suas séries) não tinha empresa: a lista, a
-- leitura, a alteração, o reset de vida e as séries de uma empresa eram
-- alcançáveis pela outra. A ferramenta passa a ser da empresa (enterprise_id);
-- séries seguem a ferramenta. O código continua único no banco.
--
-- Backfill (ferramenta é ativo físico — não se replica para quem não usa):
--   * usada em roteiros/ordens de várias empresas: fica com a menor; as demais
--     recebem cópia (código novo) com as séries, e roteiros, vínculos de série
--     nas ordens e substituições delas passam para a cópia;
--   * sem uso: fica com a menor empresa.

ALTER TABLE public.tools ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
ALTER TABLE public.tools ADD COLUMN _origem_id BIGINT;
ALTER TABLE public.tool_serials ADD COLUMN _origem_id BIGINT;

CREATE TEMP TABLE _uso_ferr (tool_id BIGINT NOT NULL, enterprise_id BIGINT NOT NULL) ON COMMIT DROP;
INSERT INTO _uso_ferr
SELECT rot.tool_id, mr.enterprise_id
  FROM public.route_operation_tools rot
  JOIN public.route_operations ro ON ro.id = rot.route_operation_id
  JOIN public.manufacturing_routes mr ON mr.id = ro.route_id
 WHERE mr.enterprise_id IS NOT NULL
UNION
SELECT ts.tool_id, po.enterprise_id
  FROM public.production_order_operation_tool_serials ts
  JOIN public.production_order_operations poo ON poo.id = ts.production_order_operation_id
  JOIN public.production_orders po ON po.id = poo.production_order_id
 WHERE po.enterprise_id IS NOT NULL
UNION
SELECT s.tool_id, po.enterprise_id
  FROM public.tool_serial_substitutions s
  JOIN public.production_order_operations poo ON poo.id = s.production_order_operation_id
  JOIN public.production_orders po ON po.id = poo.production_order_id
 WHERE po.enterprise_id IS NOT NULL;

UPDATE public.tools t SET enterprise_id = d.dono
  FROM (SELECT tool_id, MIN(enterprise_id) AS dono FROM _uso_ferr GROUP BY tool_id) d
 WHERE d.tool_id = t.id;
UPDATE public.tools SET enterprise_id = (SELECT MIN(id) FROM public.enterprise) WHERE enterprise_id IS NULL;

-- Cópias com código novo.
INSERT INTO public.tools (code, name, tool_type, life_type, life_limit, life_used, cost, status, is_active,
                          created_at, updated_at, created_by, enterprise_id, _origem_id)
SELECT (SELECT COALESCE(MAX(code), 0) FROM public.tools) + ROW_NUMBER() OVER (ORDER BY t.id, u.enterprise_id),
       t.name, t.tool_type, t.life_type, t.life_limit, t.life_used, t.cost, t.status, t.is_active,
       t.created_at, NOW(), t.created_by, u.enterprise_id, t.id
  FROM (SELECT DISTINCT tool_id, enterprise_id FROM _uso_ferr) u
  JOIN public.tools t ON t.id = u.tool_id
 WHERE u.enterprise_id <> t.enterprise_id;

INSERT INTO public.tool_serials (tool_id, serial_number, status, life_used, location, notes, is_active,
                                 created_at, updated_at, created_by, _origem_id)
SELECT nt.id, s.serial_number, s.status, s.life_used, s.location, s.notes, s.is_active,
       s.created_at, NOW(), s.created_by, s.id
  FROM public.tool_serials s JOIN public.tools nt ON nt._origem_id = s.tool_id;

CREATE TEMP TABLE _mapa_ferr ON COMMIT DROP AS
SELECT _origem_id AS antigo, enterprise_id, id AS novo FROM public.tools WHERE _origem_id IS NOT NULL;
CREATE TEMP TABLE _mapa_serie ON COMMIT DROP AS
SELECT s._origem_id AS antigo, t.enterprise_id, s.id AS novo
  FROM public.tool_serials s JOIN public.tools t ON t.id = s.tool_id WHERE s._origem_id IS NOT NULL;

UPDATE public.route_operation_tools x SET tool_id = m.novo
  FROM _mapa_ferr m, public.route_operations ro, public.manufacturing_routes mr
 WHERE ro.id = x.route_operation_id AND mr.id = ro.route_id
   AND m.antigo = x.tool_id AND m.enterprise_id = mr.enterprise_id;

UPDATE public.production_order_operation_tool_serials x SET tool_serial_id = m.novo
  FROM _mapa_serie m, public.production_order_operations poo, public.production_orders po
 WHERE poo.id = x.production_order_operation_id AND po.id = poo.production_order_id
   AND m.antigo = x.tool_serial_id AND m.enterprise_id = po.enterprise_id;
UPDATE public.production_order_operation_tool_serials x SET tool_id = m.novo
  FROM _mapa_ferr m, public.production_order_operations poo, public.production_orders po
 WHERE poo.id = x.production_order_operation_id AND po.id = poo.production_order_id
   AND m.antigo = x.tool_id AND m.enterprise_id = po.enterprise_id;

UPDATE public.tool_serial_substitutions x SET tool_id = m.novo
  FROM _mapa_ferr m, public.production_order_operations poo, public.production_orders po
 WHERE poo.id = x.production_order_operation_id AND po.id = poo.production_order_id
   AND m.antigo = x.tool_id AND m.enterprise_id = po.enterprise_id;
UPDATE public.tool_serial_substitutions x SET old_serial_id = m.novo
  FROM _mapa_serie m, public.production_order_operations poo, public.production_orders po
 WHERE poo.id = x.production_order_operation_id AND po.id = poo.production_order_id
   AND m.antigo = x.old_serial_id AND m.enterprise_id = po.enterprise_id;
UPDATE public.tool_serial_substitutions x SET new_serial_id = m.novo
  FROM _mapa_serie m, public.production_order_operations poo, public.production_orders po
 WHERE poo.id = x.production_order_operation_id AND po.id = poo.production_order_id
   AND m.antigo = x.new_serial_id AND m.enterprise_id = po.enterprise_id;

ALTER TABLE public.tool_serials DROP COLUMN _origem_id;
ALTER TABLE public.tools DROP COLUMN _origem_id;
DELETE FROM public.tools WHERE enterprise_id IS NULL; -- instalação sem empresa
ALTER TABLE public.tools ALTER COLUMN enterprise_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS ix_tools_enterprise ON public.tools (enterprise_id, code);
