BEGIN;
-- Um slot de agenda de máquina nem sempre vem de uma ordem planejada (ex.: plano
-- de corte, parada de manutenção). order_code passa a ser opcional.
ALTER TABLE public.machine_schedules ALTER COLUMN order_code DROP NOT NULL;
COMMIT;
