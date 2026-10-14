-- Volta a uma configuração só (a mais recente) e reservas sem empresa.
ALTER TABLE public.delivery_promise_params DROP CONSTRAINT IF EXISTS delivery_promise_params_enterprise_key;
DELETE FROM public.delivery_promise_params WHERE id <> (SELECT MAX(id) FROM public.delivery_promise_params);
ALTER TABLE public.delivery_promise_params DROP COLUMN IF EXISTS enterprise_id;
DROP INDEX IF EXISTS public.ix_delivery_tank_reservations_enterprise;
ALTER TABLE public.delivery_tank_reservations DROP COLUMN IF EXISTS enterprise_id;
