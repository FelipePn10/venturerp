-- O cancelamento da nota de entrada aprovada marcava os títulos como
-- CANCELADO mas os deixava ativos, e a carteira (que lista is_active) seguia
-- mostrando-os como se fossem dívida. O cancelamento manual e o da NF de saída
-- já desativavam; aqui os títulos já cancelados pela entrada saem da carteira.
UPDATE contas_pagar cp
   SET is_active = FALSE, updated_at = NOW()
  FROM fiscal_entries fe
 WHERE cp.fiscal_entry_id = fe.id
   AND cp.enterprise_id = fe.enterprise_id
   AND fe.status = 'CANCELLED'
   AND UPPER(cp.status) = 'CANCELADO'
   AND cp.is_active;
