BEGIN;

DROP FUNCTION IF EXISTS subtract_workdays(DATE, INT, BIGINT);

ALTER TABLE industrial_calendar
    DROP CONSTRAINT IF EXISTS fk_industrial_calendar_enterprise,
    DROP CONSTRAINT IF EXISTS chk_industrial_calendar_source,
    DROP CONSTRAINT IF EXISTS uq_industrial_calendar_enterprise_date;
DROP INDEX IF EXISTS idx_industrial_calendar_tenant_month;
DROP INDEX IF EXISTS idx_industrial_calendar_tenant_workday;

ALTER TABLE industrial_calendar DROP COLUMN IF EXISTS source, DROP COLUMN IF EXISTS enterprise_id;
ALTER TABLE industrial_calendar ADD CONSTRAINT industrial_calendar_year_month_day_key UNIQUE(year, month, day);
CREATE INDEX idx_industrial_calendar_year_month ON industrial_calendar(year, month);
CREATE INDEX idx_industrial_calendar_workday ON industrial_calendar(is_workday);

COMMIT;
