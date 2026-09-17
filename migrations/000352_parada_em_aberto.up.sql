-- Parada em aberto, para o operador registrar no momento em que acontece.
--
-- Parada de chão de fábrica raramente é planejada com hora de início e fim: a
-- máquina para, alguém resolve, a máquina volta. Muitas duram menos de cinco
-- minutos e acontecem várias vezes por turno. Exigir que o operador digite as
-- duas pontas garante que ele não registre — e o que não é registrado não entra
-- na capacidade, então o planejamento segue achando que a máquina produziu o
-- turno inteiro.
--
-- Com `ends_at` nulo a parada fica ABERTA: começou e ainda não terminou. O
-- planejamento trata o intervalo aberto como ocupado até agora — `COALESCE(
-- ends_at, NOW())` — o que é a leitura honesta de uma máquina que está parada
-- neste instante.

ALTER TABLE machine_downtimes ALTER COLUMN ends_at DROP NOT NULL;

ALTER TABLE machine_downtimes DROP CONSTRAINT machine_downtimes_check;
ALTER TABLE machine_downtimes ADD CONSTRAINT machine_downtimes_check
    CHECK (ends_at IS NULL OR ends_at > starts_at);

-- Uma máquina não pode estar parada duas vezes ao mesmo tempo. Sem isto, tocar
-- duas vezes no botão abriria duas paradas e a segunda nunca seria encerrada —
-- bloqueando a capacidade do recurso para sempre.
CREATE UNIQUE INDEX uq_machine_downtimes_aberta
    ON machine_downtimes (machine_id)
    WHERE ends_at IS NULL;

COMMENT ON COLUMN machine_downtimes.ends_at IS
    'Fim da parada. Nulo = parada em aberto, acontecendo agora; o planejamento a considera ocupada até o instante atual.';
