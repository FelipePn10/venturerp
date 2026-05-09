package sqlc

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createMRPExceptionMessage = `
INSERT INTO mrp_exception_messages
    (plan_code, item_code, message_type, source_code, source_type, description)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING code, plan_code, item_code, message_type, source_code, source_type, description, created_at
`

type CreateMRPExceptionMessageParams struct {
	PlanCode    int64
	ItemCode    int64
	MessageType string
	SourceCode  *int64
	SourceType  pgtype.Text
	Description string
}

func (q *Queries) CreateMRPExceptionMessage(ctx context.Context, arg CreateMRPExceptionMessageParams) (MrpExceptionMessage, error) {
	row := q.db.QueryRow(ctx, createMRPExceptionMessage,
		arg.PlanCode, arg.ItemCode, arg.MessageType, arg.SourceCode, arg.SourceType, arg.Description)
	var m MrpExceptionMessage
	err := row.Scan(&m.Code, &m.PlanCode, &m.ItemCode, &m.MessageType, &m.SourceCode, &m.SourceType, &m.Description, &m.CreatedAt)
	return m, err
}

const listMRPExceptionMessages = `
SELECT code, plan_code, item_code, message_type, source_code, source_type, description, created_at
FROM mrp_exception_messages
WHERE plan_code = $1
ORDER BY item_code, code
`

func (q *Queries) ListMRPExceptionMessages(ctx context.Context, planCode int64) ([]MrpExceptionMessage, error) {
	rows, err := q.db.Query(ctx, listMRPExceptionMessages, planCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []MrpExceptionMessage
	for rows.Next() {
		var m MrpExceptionMessage
		if err := rows.Scan(
			&m.Code, &m.PlanCode, &m.ItemCode, &m.MessageType,
			&m.SourceCode, &m.SourceType, &m.Description, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

const deleteMRPExceptionMessages = `DELETE FROM mrp_exception_messages WHERE plan_code = $1`

func (q *Queries) DeleteMRPExceptionMessages(ctx context.Context, planCode int64) error {
	_, err := q.db.Exec(ctx, deleteMRPExceptionMessages, planCode)
	return err
}
