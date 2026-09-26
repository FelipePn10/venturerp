-- Condição de pagamento com PERCENTUAL por parcela e evento de vencimento.
--
-- Até aqui uma condição só sabia dizer "parcela 1 em 28 dias, parcela 2 em 56
-- dias" — todas com o mesmo peso e todas contadas da emissão. A condição que a
-- indústria mais usa não cabe nisso:
--
--     30% de ENTRADA, 20% na ENTREGA e o restante em 28/56 dias
--
-- Faltavam as duas metades do problema: QUANTO cada parcela leva (o percentual)
-- e A PARTIR DE QUANDO ela conta (o evento). Sem percentual, o sistema só sabia
-- dividir em partes iguais; sem evento, tudo contava da emissão, então "na
-- entrega" virava um número de dias chutado na hora do pedido.
--
-- É o modelo dos ERPs grandes: no SAP são as linhas de "instalment payment
-- terms" (OBB9), cada uma com o seu percentual e o seu prazo; no Protheus é a
-- condição de tipo 9; no FoccoERP, a parcela percentual com evento-base.
--
-- `percentage` nulo mantém o comportamento antigo (divisão em partes iguais),
-- então nenhuma condição já cadastrada muda de significado.

CREATE TYPE payment_base_event_enum AS ENUM ('EMISSAO', 'ENTRADA', 'ENTREGA', 'FATURAMENTO');

ALTER TABLE payment_condition_installments
    ADD COLUMN IF NOT EXISTS percentage NUMERIC(9,4),
    ADD COLUMN IF NOT EXISTS base_event payment_base_event_enum NOT NULL DEFAULT 'EMISSAO';

COMMENT ON COLUMN payment_condition_installments.percentage IS
  'Quanto do total esta parcela leva, em %. Nulo em todas as parcelas = divisão em partes iguais (comportamento anterior). Informado em alguma, a soma da condição precisa fechar 100.';
COMMENT ON COLUMN payment_condition_installments.base_event IS
  'A partir de quando os dias contam: EMISSAO (padrão), ENTRADA (no ato, dias sempre 0), ENTREGA ou FATURAMENTO.';

-- Percentual fora de (0,100] não é parcela, é engano de digitação.
ALTER TABLE payment_condition_installments
    ADD CONSTRAINT payment_condition_installments_percentage_check
    CHECK (percentage IS NULL OR (percentage > 0 AND percentage <= 100));

-- A entrada é paga no ato: contar dias a partir dela não significa nada.
ALTER TABLE payment_condition_installments
    ADD CONSTRAINT payment_condition_installments_entrada_sem_prazo_check
    CHECK (base_event <> 'ENTRADA' OR due_days = 0);
