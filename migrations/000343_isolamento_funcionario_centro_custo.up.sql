-- Quarta leva do isolamento multiempresa, encontrada com
-- scripts/audit-tenant-isolation.mjs (repositório do front) em 12/09/2026.
--
-- Duas rotas entregavam dado de outra empresa:
--   /api/employee/{code} e /api/employee/list  → cadastro de funcionários
--   /api/cost-center/{code}                    → centros de custo
--
-- `employees` já tinha `enterprise_id`, mas NENHUMA das cinco consultas
-- filtrava por ele — e isso não era só leitura: `UpdateEmployee` e
-- `DeactivateEmployee` localizam por `code`, então uma empresa alterava ou
-- inativava o funcionário da outra. `cost_centers` não tinha coluna de empresa
-- nenhuma: era global por construção.
--
-- Em produção havia 1 empresa, 28 funcionários com enterprise_id nulo e nenhum
-- centro de custo. Nada vazou até aqui; a coluna nasce agora justamente porque
-- ainda dá para dizer a quem pertence o que já existe.

ALTER TABLE employees    ALTER COLUMN enterprise_id DROP DEFAULT;
ALTER TABLE cost_centers ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;

-- O que já existe pertence à empresa mais antiga da base — em base nova, a
-- única. Mesmo critério da migração 340.
UPDATE employees    SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;
UPDATE cost_centers SET enterprise_id = (SELECT MIN(id) FROM enterprise) WHERE enterprise_id IS NULL;

-- O NOT NULL vale sempre que for possível aplicá-lo. Numa instalação nova as
-- tabelas estão vazias, então ele entra sem esforço — a versão anterior desta
-- migração condicionava a existir empresa, e o resultado era pior que o
-- problema: base nova ficava PERMANENTEMENTE sem a restrição, porque nada mais
-- adiante a aplica. Duas instalações com esquemas diferentes conforme a ordem
-- em que foram criadas é exatamente o tipo de divergência que só aparece meses
-- depois. Só pulamos no caso patológico (linhas órfãs sem empresa para herdar),
-- onde forçar abortaria a migração inteira.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM employees WHERE enterprise_id IS NULL) THEN
        ALTER TABLE employees ALTER COLUMN enterprise_id SET NOT NULL;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM cost_centers WHERE enterprise_id IS NULL) THEN
        ALTER TABLE cost_centers ALTER COLUMN enterprise_id SET NOT NULL;
    END IF;
END $$;

ALTER TABLE cost_centers ADD CONSTRAINT fk_cost_centers_enterprise
    FOREIGN KEY (enterprise_id) REFERENCES enterprise(id);

CREATE INDEX IF NOT EXISTS idx_employees_enterprise    ON employees (enterprise_id);
CREATE INDEX IF NOT EXISTS idx_cost_centers_enterprise ON cost_centers (enterprise_id);

-- `employees.code` e `cost_centers.code` continuam únicos globalmente, pelo
-- mesmo motivo documentado para `suppliers.code` na migração 340: são alvo de
-- chave estrangeira de várias tabelas (planned_orders.employee_code,
-- machines.cost_center_code, independent_demands.cost_center_code,
-- allocation_base_items, overhead_allocations…). Trocar por chave composta
-- exigiria reescrever todas elas, com risco desproporcional ao ganho. A
-- consequência é cosmética — a numeração é contínua entre empresas. O
-- isolamento que importa, o de leitura e escrita, vem do enterprise_id nas
-- consultas.
