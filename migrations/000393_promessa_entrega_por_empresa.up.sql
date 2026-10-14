-- Promessa de entrega por empresa.
--   * reservas de tanque: não tinham empresa; a promessa de uma empresa
--     enxergava a ocupação dos tanques da outra, e uma cancelava a reserva da
--     outra pelo código. Backfill pela empresa do item;
--   * parâmetros da promessa: uma linha só para o banco inteiro. Cada empresa
--     passa a ter os seus (cópia da configuração vigente).
ALTER TABLE public.delivery_tank_reservations ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
UPDATE public.delivery_tank_reservations r SET enterprise_id = i.enterprise_id
  FROM public.items i WHERE i.code = r.item_code AND r.enterprise_id IS NULL;
UPDATE public.delivery_tank_reservations SET enterprise_id = (SELECT MIN(id) FROM public.enterprise) WHERE enterprise_id IS NULL;
DELETE FROM public.delivery_tank_reservations WHERE enterprise_id IS NULL; -- instalação sem empresa
ALTER TABLE public.delivery_tank_reservations ALTER COLUMN enterprise_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS ix_delivery_tank_reservations_enterprise
    ON public.delivery_tank_reservations (enterprise_id, status, allocation_date);

ALTER TABLE public.delivery_promise_params ADD COLUMN IF NOT EXISTS enterprise_id BIGINT REFERENCES public.enterprise(id);
CREATE TEMP TABLE _vigente ON COMMIT DROP AS
SELECT * FROM public.delivery_promise_params ORDER BY id DESC LIMIT 1;
DELETE FROM public.delivery_promise_params;
INSERT INTO public.delivery_promise_params
       (use_delivery_promise, blocked_orders_in_promise, default_order_sort, show_order_values, blocked_export_in_promise,
        break_tank_occupation, recalculate_after_release, reprogram_loaded_orders, allow_delivery_date_change,
        created_at, updated_at, updated_by, enterprise_id)
SELECT v.use_delivery_promise, v.blocked_orders_in_promise, v.default_order_sort, v.show_order_values, v.blocked_export_in_promise,
       v.break_tank_occupation, v.recalculate_after_release, v.reprogram_loaded_orders, v.allow_delivery_date_change,
       v.created_at, v.updated_at, v.updated_by, e.id
  FROM _vigente v CROSS JOIN public.enterprise e;
ALTER TABLE public.delivery_promise_params ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE public.delivery_promise_params ADD CONSTRAINT delivery_promise_params_enterprise_key UNIQUE (enterprise_id);
