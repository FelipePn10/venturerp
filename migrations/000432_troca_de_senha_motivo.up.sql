-- Motivo informado por quem pede a troca de senha. A tela já pedia e mandava o
-- motivo, mas a API o descartava: o administrador aprovava sem saber por quê.
ALTER TABLE password_change_requests ADD COLUMN IF NOT EXISTS reason VARCHAR(500);

-- Aprovação vencida e nunca usada ficava APPROVED para sempre e, pelo índice de
-- uma solicitação ativa por usuário, impedia o usuário de pedir outra.
UPDATE password_change_requests
   SET status = 'EXPIRED', updated_at = now()
 WHERE status = 'APPROVED' AND expires_at <= now();
