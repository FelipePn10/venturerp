package mrp_calculation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

func optionalPgDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func exceptionFromRow(row sqlc.MrpExceptionMessage) *entity.MRPExceptionMessage {
	m := &entity.MRPExceptionMessage{
		Code:           row.Code,
		PlanCode:       row.PlanCode,
		ItemCode:       row.ItemCode,
		MessageType:    entity.ExceptionMessageType(row.MessageType),
		Description:    row.Description,
		CreatedAt:      pgutil.FromPgTimestamptz(row.CreatedAt),
		HandlingAction: row.HandlingAction.String,
		HandlingNote:   row.HandlingNote.String,
	}
	if row.SourceCode != nil {
		v := *row.SourceCode
		m.SourceCode = &v
	}
	if row.SourceType.Valid {
		v := row.SourceType.String
		m.SourceType = &v
	}
	if row.SuggestedDate.Valid {
		d := row.SuggestedDate.Time
		m.SuggestedDate = &d
	}
	if row.HandledAt.Valid {
		t := row.HandledAt.Time
		m.HandledAt = &t
	}
	if row.HandledBy.Valid {
		v := uuid.UUID(row.HandledBy.Bytes).String()
		m.HandledBy = &v
	}
	return m
}

// GetException devolve uma exceção da empresa da sessão.
func (r *MRPCalculationRepositorySQLC) GetException(ctx context.Context, code int64) (*entity.MRPExceptionMessage, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetMRPExceptionMessage(ctx, sqlc.GetMRPExceptionMessageParams{Code: code, EnterpriseID: enterpriseID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("exceção %d não encontrada", code))
		}
		return nil, err
	}
	return exceptionFromRow(row), nil
}

// MarkExceptionHandled registra como e por quem a exceção foi tratada.
func (r *MRPCalculationRepositorySQLC) MarkExceptionHandled(ctx context.Context, code int64, action string, note *string, by uuid.UUID) (*entity.MRPExceptionMessage, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	nota := pgtype.Text{}
	if note != nil {
		nota = pgtype.Text{String: *note, Valid: true}
	}
	row, err := r.q.MarkMRPExceptionHandled(ctx, sqlc.MarkMRPExceptionHandledParams{
		HandledBy:      pgtype.UUID{Bytes: by, Valid: true},
		HandlingAction: pgtype.Text{String: action, Valid: true},
		HandlingNote:   nota,
		Code:           code,
		EnterpriseID:   enterpriseID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Só grava se ainda não tratada: com dois cliques ao mesmo tempo,
			// o segundo chega aqui e recebe conflito em vez de sobrescrever.
			if _, getErr := r.GetException(ctx, code); getErr != nil {
				return nil, getErr
			}
			return nil, errorsuc.NewConflictError("esta exceção já foi tratada")
		}
		return nil, err
	}
	return exceptionFromRow(row), nil
}
