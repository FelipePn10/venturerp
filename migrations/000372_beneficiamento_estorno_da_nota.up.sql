-- Estorno das baixas quando a NF-e de retorno é cancelada.
--
-- As regras do beneficiamento da Usimac (documento 6, item "Cancelamento,
-- rejeição, devolução ou correção da nota fiscal") dizem: "o sistema deverá
-- estornar as movimentações relacionadas". Sem isso, cancelar a nota de retorno
-- deixa o saldo do cliente baixado — o sistema afirma que o material voltou
-- quando ele continua no pátio. É a pior classe de erro deste módulo, porque o
-- saldo passa a contradizer a realidade física de material que não é da empresa.
--
-- Por que marcar em vez de apagar o movimento: o razão é a conferência que se
-- apresenta ao cliente, e "saiu e voltou" é informação, não ruído. Apagar a linha
-- deixaria o razão contando uma história que não aconteceu. A quantidade estornada
-- volta para o saldo pelo decremento da coluna de consumo da linha; o movimento
-- fica visível, marcado, com quem estornou e por quê.
--
-- Não há coluna nova na tabela de itens: o saldo é coluna gerada
-- (qty_received - qty_returned - qty_leftover - qty_scrapped), então devolver
-- saldo é decrementar a mesma coluna que a baixa incrementou.

BEGIN;

ALTER TABLE customer_material_movements
    ADD COLUMN IF NOT EXISTS reversed_at      TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reversed_by      UUID REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS reversal_reason  TEXT;

-- Estorno sem motivo é estorno que ninguém audita — mesma disciplina do ajuste.
ALTER TABLE customer_material_movements
    DROP CONSTRAINT IF EXISTS customer_material_movements_estorno_completo;
ALTER TABLE customer_material_movements
    ADD CONSTRAINT customer_material_movements_estorno_completo
    CHECK (reversed_at IS NULL
           OR (reversed_by IS NOT NULL AND reversal_reason IS NOT NULL));

-- A consulta é "o que esta nota ainda sustenta": os movimentos vivos dela.
CREATE INDEX IF NOT EXISTS idx_customer_material_movements_nota_viva
    ON customer_material_movements (enterprise_id, fiscal_exit_id)
    WHERE fiscal_exit_id IS NOT NULL AND reversed_at IS NULL;

COMMIT;
