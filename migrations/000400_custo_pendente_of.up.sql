-- Motivo de a OF ter sido encerrada sem o custo real apurado (a apuração
-- falhou). Fica visível na ordem até "Apurar custo" dar certo, que limpa.
ALTER TABLE production_orders ADD COLUMN IF NOT EXISTS custo_pendente TEXT;
