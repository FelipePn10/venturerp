-- Volta a proibir turno que vira o dia. Turnos noturnos já cadastrados violariam
-- o CHECK antigo e impediriam a reversão, então são removidos antes — é a única
-- leitura possível: sob a regra antiga eles não poderiam existir.
DELETE FROM machine_calendar_intervals WHERE end_time <= start_time;

ALTER TABLE machine_calendar_intervals DROP CONSTRAINT machine_calendar_intervals_check;
ALTER TABLE machine_calendar_intervals ADD CONSTRAINT machine_calendar_intervals_check
    CHECK (end_time > start_time);
