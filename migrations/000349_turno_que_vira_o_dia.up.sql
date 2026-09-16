-- Turno que atravessa a meia-noite.
--
-- O CHECK antigo (end_time > start_time) tornava o terceiro turno impossível de
-- cadastrar: 22:00–06:00 era simplesmente recusado pelo banco. O contorno seria
-- quebrar em 22:00–23:59 e 00:00–06:00, mas aí a janela deixa de ser contínua e
-- um ciclo fechado de três horas não cabe em nenhuma das duas metades — a
-- máquina fica "sem capacidade" numa madrugada em que ela está produzindo.
--
-- A partir daqui, end_time MENOR OU IGUAL a start_time significa "termina no dia
-- seguinte". A janela vira segunda 22:00 → terça 06:00, contínua. Continua
-- proibido o intervalo degenerado (início igual ao fim), que não seria turno
-- nenhum.
--
-- A convenção vale nos TRÊS lugares que constroem janela a partir do calendário
-- — MRP (machine_scheduling.go), APS (sequencing_selection.go) e CRP
-- (machine_capacity/capacity.go). Corrigir um só faria o turno noturno existir
-- para o planejamento e não para a análise de carga, que é pior que a limitação
-- anterior, porque a divergência é silenciosa.
--
-- O turno pertence ao dia em que COMEÇA: um turno de segunda 22:00 conta como
-- capacidade de segunda, ainda que seis das suas oito horas caiam na terça.

ALTER TABLE machine_calendar_intervals DROP CONSTRAINT machine_calendar_intervals_check;
ALTER TABLE machine_calendar_intervals ADD CONSTRAINT machine_calendar_intervals_check
    CHECK (end_time <> start_time);

COMMENT ON COLUMN machine_calendar_intervals.end_time IS
    'Fim do turno. Menor ou igual ao início significa que termina no dia seguinte (turno noturno).';
