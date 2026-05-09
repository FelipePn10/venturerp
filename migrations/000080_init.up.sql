CREATE OR REPLACE FUNCTION subtract_workdays(base_date DATE, days_to_sub INT)
RETURNS DATE
LANGUAGE plpgsql
AS $$
DECLARE
    result    DATE := base_date;
    remaining INT  := days_to_sub;
BEGIN
    IF days_to_sub <= 0 THEN
        RETURN base_date;
    END IF;

    WHILE remaining > 0 LOOP
        result := result - 1;

        IF EXISTS (
            SELECT 1 FROM industrial_calendar
            WHERE year  = EXTRACT(YEAR  FROM result)::INT
              AND month = EXTRACT(MONTH FROM result)::INT
              AND day   = EXTRACT(DAY   FROM result)::INT
              AND is_workday = TRUE
        ) THEN
            remaining := remaining - 1;

        -- Day not in calendar: fall back to Mon–Fri weekday rule
        ELSIF NOT EXISTS (
            SELECT 1 FROM industrial_calendar
            WHERE year  = EXTRACT(YEAR  FROM result)::INT
              AND month = EXTRACT(MONTH FROM result)::INT
              AND day   = EXTRACT(DAY   FROM result)::INT
        ) THEN
            IF EXTRACT(DOW FROM result) NOT IN (0, 6) THEN
                remaining := remaining - 1;
            END IF;
        END IF;
    END LOOP;

    RETURN result;
END;
$$;
