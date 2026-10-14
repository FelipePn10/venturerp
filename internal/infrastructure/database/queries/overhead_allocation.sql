-- O rateio é da empresa (enterprise_id). Centro de custo de origem e de
-- destino e a base de rateio têm de ser da mesma empresa; os destinos são
-- alcançados pelo rateio.

-- name: CreateOverheadAllocation :one
INSERT INTO overhead_allocations (cost_center_code, plan_account_code, account_code, period_start, period_end, allocation_type, base_code, created_by, enterprise_id)
SELECT sqlc.narg(cost_center_code), sqlc.narg(plan_account_code), sqlc.narg(account_code), sqlc.arg(period_start), sqlc.arg(period_end),
       sqlc.arg(allocation_type), sqlc.narg(base_code), sqlc.arg(created_by), sqlc.arg(enterprise_id)
 WHERE (sqlc.narg(cost_center_code)::int IS NULL OR EXISTS (SELECT 1 FROM cost_centers cc WHERE cc.code = sqlc.narg(cost_center_code) AND cc.enterprise_id = sqlc.arg(enterprise_id)))
   AND (sqlc.narg(base_code)::bigint IS NULL OR EXISTS (SELECT 1 FROM allocation_bases b WHERE b.code = sqlc.narg(base_code) AND b.enterprise_id = sqlc.arg(enterprise_id)))
RETURNING *;

-- name: AddAllocationTarget :one
INSERT INTO overhead_allocation_targets (overhead_code, cost_center_code, percentage, amount)
SELECT sqlc.narg(overhead_code), sqlc.narg(cost_center_code), sqlc.arg(percentage), sqlc.arg(amount)
 WHERE EXISTS (SELECT 1 FROM overhead_allocations o WHERE o.code = sqlc.narg(overhead_code) AND o.enterprise_id = sqlc.arg(enterprise_id))
   AND (sqlc.narg(cost_center_code)::int IS NULL OR EXISTS (SELECT 1 FROM cost_centers cc WHERE cc.code = sqlc.narg(cost_center_code) AND cc.enterprise_id = sqlc.arg(enterprise_id)))
RETURNING *;

-- name: GetOverheadAllocationByCode :one
SELECT * FROM overhead_allocations WHERE code = $1 AND enterprise_id = $2;

-- name: GetAllocationTargets :many
SELECT t.* FROM overhead_allocation_targets t
  JOIN overhead_allocations o ON o.code = t.overhead_code AND o.enterprise_id = $2
 WHERE t.overhead_code = $1;

-- name: ListOverheadAllocations :many
SELECT * FROM overhead_allocations WHERE enterprise_id = $1 ORDER BY created_at DESC;

-- name: ListOverheadAllocationsByCostCenter :many
SELECT * FROM overhead_allocations WHERE cost_center_code = $1 AND enterprise_id = $2;

-- name: DeleteOverheadAllocation :execrows
DELETE FROM overhead_allocations WHERE code = $1 AND enterprise_id = $2;

-- name: DeleteAllocationTargets :exec
DELETE FROM overhead_allocation_targets t
 USING overhead_allocations o
 WHERE t.overhead_code = $1 AND o.code = t.overhead_code AND o.enterprise_id = $2;
