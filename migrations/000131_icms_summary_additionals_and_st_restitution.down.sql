BEGIN;
DROP TABLE IF EXISTS icms_st_restitutions;
DROP TYPE IF EXISTS icms_st_restitution_type_enum;
ALTER TABLE icms_summary_entry_notes
    DROP COLUMN IF EXISTS adjustment_value,
    DROP COLUMN IF EXISTS aliquota,
    DROP COLUMN IF EXISTS calc_base,
    DROP COLUMN IF EXISTS other_value,
    DROP COLUMN IF EXISTS note_type,
    DROP COLUMN IF EXISTS motivo_id,
    DROP COLUMN IF EXISTS visto_date,
    DROP COLUMN IF EXISTS c190_obs_code,
    DROP COLUMN IF EXISTS obs_code_c190;
DROP TABLE IF EXISTS icms_summary_entry_additionals;
DROP TYPE IF EXISTS arrecadacao_indicator_enum;
COMMIT;
