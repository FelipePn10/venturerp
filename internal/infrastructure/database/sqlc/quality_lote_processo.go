package sqlc

// Escrito à mão, como quality.sql.go: abre o lote de inspeção de processo
// dentro da transação que cria a ordem de produção (Queries pode estar ligado
// à transação), com as características copiadas do plano de inspeção.

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type AbrirLoteDeProcessoParams struct {
	EnterpriseID      int64
	ItemCode          int64
	Quantidade        pgtype.Numeric
	ProductionOrderID int64
	RouteOperationID  int64
	InspectionPlanID  int64
	ProcedimentoID    *int64
	Chave             *string
	Plano             []byte
	Pulado            bool
	CreatedBy         pgtype.UUID
}

func (q *Queries) AbrirLoteDeProcesso(ctx context.Context, arg AbrirLoteDeProcessoParams) (int64, error) {
	if _, err := q.db.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('quality_lot:' || $1::bigint, 0))`, arg.EnterpriseID); err != nil {
		return 0, err
	}
	var id int64
	err := q.db.QueryRow(ctx, `
		INSERT INTO quality_inspection_lots (enterprise_id, numero, origem, status, item_code, quantidade, production_order_id,
			route_operation_id, inspection_plan_id, sampling_procedure_id, chave_amostragem, plano, pulado, created_by)
		SELECT $1, COALESCE((SELECT MAX(numero) FROM quality_inspection_lots WHERE enterprise_id = $1), 0) + 1, 'PROCESSO', 'ABERTO',
		       $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		 WHERE EXISTS (SELECT 1 FROM items WHERE code = $2 AND enterprise_id = $1)
		RETURNING id`,
		arg.EnterpriseID, arg.ItemCode, arg.Quantidade, arg.ProductionOrderID, arg.RouteOperationID, arg.InspectionPlanID,
		arg.ProcedimentoID, arg.Chave, arg.Plano, arg.Pulado, arg.CreatedBy).Scan(&id)
	if err != nil {
		return 0, err
	}
	_, err = q.db.Exec(ctx, `
		INSERT INTO quality_lot_characteristics (lot_id, enterprise_id, sequencia, nome, tipo, obrigatoria, critica, nominal, minimo, maximo,
			unidade, origem_characteristic_id)
		SELECT $1, $2, ROW_NUMBER() OVER (ORDER BY c.id), c.name,
		       CASE WHEN c.nominal IS NULL AND c.tolerance_lower IS NULL AND c.tolerance_upper IS NULL THEN 'ATRIBUTO' ELSE 'VARIAVEL' END,
		       TRUE, c.is_critical, c.nominal, c.tolerance_lower, c.tolerance_upper, c.unit, c.id
		  FROM inspection_plan_characteristics c WHERE c.plan_id = $3`, id, arg.EnterpriseID, arg.InspectionPlanID)
	return id, err
}
