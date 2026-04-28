ALTER TABLE public.allocation_base_items
DROP CONSTRAINT IF EXISTS allocation_base_items_allocation_base_id_fkey;

ALTER TABLE public.allocation_base_items
DROP CONSTRAINT IF EXISTS allocation_base_items_cost_center_id_fkey;

DROP INDEX IF EXISTS public.idx_allocation_base_items_base;

ALTER TABLE public.allocation_base_items
    RENAME COLUMN allocation_base_id TO allocation_base_code;

ALTER TABLE public.allocation_base_items
    RENAME COLUMN cost_center_id TO cost_center_code;

CREATE INDEX idx_allocation_base_items_base_code
    ON public.allocation_base_items (allocation_base_code);

DROP INDEX IF EXISTS public.idx_allocation_base_items_base_code;

ALTER TABLE public.allocation_base_items
    RENAME COLUMN allocation_base_code TO allocation_base_id;

ALTER TABLE public.allocation_base_items
    RENAME COLUMN cost_center_code TO cost_center_id;

CREATE INDEX idx_allocation_base_items_base
    ON public.allocation_base_items (allocation_base_id);

ALTER TABLE public.allocation_base_items
    ADD CONSTRAINT allocation_base_items_allocation_base_id_fkey
        FOREIGN KEY (allocation_base_id)
            REFERENCES public.allocation_bases(id)
            ON DELETE CASCADE;

ALTER TABLE public.allocation_base_items
    ADD CONSTRAINT allocation_base_items_cost_center_id_fkey
        FOREIGN KEY (cost_center_id)
            REFERENCES public.cost_centers(id);