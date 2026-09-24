package restriction

import (
	"context"
	"errors"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"

	"github.com/FelipePn10/panossoerp/internal/domain/restriction/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// empresa devolve o tenant no formato que o sqlc espera para `enterprise_id`.
// Toda consulta de restrição passa por aqui: a restrição diz o que um cliente
// NÃO pode comprar, e herdar a regra da empresa vizinha bloquearia uma venda
// legítima — ou liberaria uma que deveria parar.
func empresa(ctx context.Context) (*int64, error) {
	id, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (r *RestrictionRepositorySQLC) Create(
	ctx context.Context,
	res *entity.Restriction,
) (*entity.Restriction, error) {
	tenantID, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateRestriction(ctx, sqlc.CreateRestrictionParams{
		Situation:            sqlc.RestrictionSituationEnum(res.Situation),
		CustomerCode:         res.CustomerCode,
		ItemCode:             res.ItemCode,
		ReasonCode:           res.ReasonCode,
		ClassificationType:   pgutil.ToPgTextFromPtr(res.ClassificationType),
		ClassificationOrigin: pgutil.ToPgTextFromPtr(res.ClassificationOrigin),
		DivisionID:           res.DivisionID,
		Weight:               int32(res.Weight),
		CreatedBy:            pgutil.ToPgUUID(res.CreatedBy),
		EnterpriseID:         tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("creating restriction: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *RestrictionRepositorySQLC) Update(
	ctx context.Context,
	res *entity.Restriction,
) (*entity.Restriction, error) {
	tenantID, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdateRestriction(ctx, sqlc.UpdateRestrictionParams{
		EnterpriseID:         tenantID,
		Code:                 pgtype.Int8{Int64: res.Code, Valid: true},
		Situation:            sqlc.RestrictionSituationEnum(res.Situation),
		CustomerCode:         res.CustomerCode,
		ItemCode:             res.ItemCode,
		ReasonCode:           res.ReasonCode,
		ClassificationType:   pgutil.ToPgTextFromPtr(res.ClassificationType),
		ClassificationOrigin: pgutil.ToPgTextFromPtr(res.ClassificationOrigin),
		DivisionID:           res.DivisionID,
		Weight:               int32(res.Weight),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("restrição %d não encontrada", res.Code))
		}
		return nil, fmt.Errorf("updating restriction: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *RestrictionRepositorySQLC) GetByCode(
	ctx context.Context,
	code int64,
) (*entity.Restriction, error) {
	tenantID, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetRestrictionByCode(ctx, sqlc.GetRestrictionByCodeParams{Code: pgtype.Int8{Int64: code, Valid: true}, EnterpriseID: tenantID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("restrição %d não encontrada", code))
		}
		return nil, fmt.Errorf("fetching restriction: %w", err)
	}
	return r.comClausulas(ctx, rowToEntity(row))
}

func (r *RestrictionRepositorySQLC) GetByItemCode(
	ctx context.Context,
	itemCode int64,
) ([]*entity.Restriction, error) {
	tenantID, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.GetRestrictionsByItemCode(ctx, sqlc.GetRestrictionsByItemCodeParams{ItemCode: &itemCode, EnterpriseID: tenantID})
	if err != nil {
		return nil, fmt.Errorf("fetching restrictions for item %d: %w", itemCode, err)
	}
	out := make([]*entity.Restriction, 0, len(rows))
	for _, row := range rows {
		completa, cErr := r.comClausulas(ctx, rowToEntity(row))
		if cErr != nil {
			return nil, cErr
		}
		out = append(out, completa)
	}
	return out, nil
}

// comClausulas preenche o "SE" e o "ENTÃO" da regra.
//
// Sem isso a restrição só era legível no instante em que foi criada: a
// consulta devolvia o cabeçalho e a tela mostrava uma regra vazia, sem como
// conferir o que ela realmente bloqueia.
func (r *RestrictionRepositorySQLC) comClausulas(
	ctx context.Context,
	restricao *entity.Restriction,
) (*entity.Restriction, error) {
	dominantes, err := r.GetDominants(ctx, restricao.ID)
	if err != nil {
		return nil, err
	}
	determinantes, err := r.GetDeterminants(ctx, restricao.ID)
	if err != nil {
		return nil, err
	}
	restricao.Dominants = dominantes
	restricao.Determinants = determinantes
	return restricao, nil
}

func (r *RestrictionRepositorySQLC) GetByCustomerCode(
	ctx context.Context,
	customerCode int64,
) ([]*entity.Restriction, error) {
	tenantID, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.GetRestrictionsByCustomerCode(ctx, sqlc.GetRestrictionsByCustomerCodeParams{CustomerCode: &customerCode, EnterpriseID: tenantID})
	if err != nil {
		return nil, fmt.Errorf("fetching restrictions for customer %d: %w", customerCode, err)
	}
	out := make([]*entity.Restriction, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}
	return out, nil
}

func (r *RestrictionRepositorySQLC) List(ctx context.Context) ([]*entity.Restriction, error) {
	tenantID, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListRestrictions(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listing restrictions: %w", err)
	}
	out := make([]*entity.Restriction, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}
	return out, nil
}

func (r *RestrictionRepositorySQLC) ListActive(ctx context.Context) ([]*entity.Restriction, error) {
	tenantID, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListActiveRestrictions(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listing active restrictions: %w", err)
	}
	out := make([]*entity.Restriction, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}
	return out, nil
}

func (r *RestrictionRepositorySQLC) Deactivate(ctx context.Context, code int64) error {
	tenantID, err := empresa(ctx)
	if err != nil {
		return err
	}
	return r.q.DeactivateRestriction(ctx, sqlc.DeactivateRestrictionParams{Code: pgtype.Int8{Int64: code, Valid: true}, EnterpriseID: tenantID})
}

func (r *RestrictionRepositorySQLC) ListRestrictedItemCodes(
	ctx context.Context,
	itemCodes []int64,
) (map[int64]struct{}, error) {
	if len(itemCodes) == 0 {
		return make(map[int64]struct{}), nil
	}
	tenantID, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListActiveRestrictionsByItems(ctx, sqlc.ListActiveRestrictionsByItemsParams{ItemCodes: itemCodes, EnterpriseID: tenantID})
	if err != nil {
		return nil, fmt.Errorf("listing restricted items: %w", err)
	}
	out := make(map[int64]struct{}, len(rows))
	for _, row := range rows {
		if row.ItemCode != nil {
			out[*row.ItemCode] = struct{}{}
		}
	}
	return out, nil
}

func (r *RestrictionRepositorySQLC) AddDominant(
	ctx context.Context,
	d *entity.RestrictionDominant,
) (*entity.RestrictionDominant, error) {
	row, err := r.q.AddRestrictionDominant(ctx, sqlc.AddRestrictionDominantParams{
		RestrictionID: d.RestrictionID,
		QuestionID:    d.QuestionID,
		Operator:      sqlc.RestrictionOperatorEnum(d.Operator),
		ConditionType: sqlc.RestrictionConditionEnum(d.ConditionType),
		AnswerValue:   d.AnswerValue,
		Sequence:      int32(d.Sequence),
	})
	if err != nil {
		return nil, fmt.Errorf("adding restriction dominant: %w", err)
	}
	return &entity.RestrictionDominant{
		ID:            row.ID,
		RestrictionID: row.RestrictionID,
		QuestionID:    row.QuestionID,
		Operator:      entity.RestrictionOperator(row.Operator),
		ConditionType: entity.RestrictionCondition(row.ConditionType),
		AnswerValue:   row.AnswerValue,
		Sequence:      int(row.Sequence),
	}, nil
}

func (r *RestrictionRepositorySQLC) AddDeterminant(
	ctx context.Context,
	d *entity.RestrictionDeterminant,
) (*entity.RestrictionDeterminant, error) {
	row, err := r.q.AddRestrictionDeterminant(ctx, sqlc.AddRestrictionDeterminantParams{
		RestrictionID: d.RestrictionID,
		QuestionID:    d.QuestionID,
		Operator:      sqlc.RestrictionOperatorEnum(d.Operator),
		AnswerValue:   pgutil.ToPgTextFromPtr(d.AnswerValue),
	})
	if err != nil {
		return nil, fmt.Errorf("adding restriction determinant: %w", err)
	}
	return &entity.RestrictionDeterminant{
		ID:            row.ID,
		RestrictionID: row.RestrictionID,
		QuestionID:    row.QuestionID,
		Operator:      entity.RestrictionOperator(row.Operator),
		AnswerValue:   pgutil.FromPgTextPtr(row.AnswerValue),
	}, nil
}

func (r *RestrictionRepositorySQLC) DeleteDominant(ctx context.Context, id int64) error {
	return r.q.DeleteRestrictionDominant(ctx, id)
}

func (r *RestrictionRepositorySQLC) DeleteDeterminant(ctx context.Context, id int64) error {
	return r.q.DeleteRestrictionDeterminant(ctx, id)
}

func (r *RestrictionRepositorySQLC) GetDominants(
	ctx context.Context,
	restrictionID int64,
) ([]*entity.RestrictionDominant, error) {
	tenantID, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.GetRestrictionDominants(ctx, sqlc.GetRestrictionDominantsParams{RestrictionID: restrictionID, EnterpriseID: tenantID})
	if err != nil {
		return nil, fmt.Errorf("fetching dominants: %w", err)
	}
	out := make([]*entity.RestrictionDominant, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.RestrictionDominant{
			ID:            row.ID,
			RestrictionID: row.RestrictionID,
			QuestionID:    row.QuestionID,
			Operator:      entity.RestrictionOperator(row.Operator),
			ConditionType: entity.RestrictionCondition(row.ConditionType),
			AnswerValue:   row.AnswerValue,
			Sequence:      int(row.Sequence),
		})
	}
	return out, nil
}

func (r *RestrictionRepositorySQLC) GetDeterminants(
	ctx context.Context,
	restrictionID int64,
) ([]*entity.RestrictionDeterminant, error) {
	tenantID, err := empresa(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.GetRestrictionDeterminants(ctx, sqlc.GetRestrictionDeterminantsParams{RestrictionID: restrictionID, EnterpriseID: tenantID})
	if err != nil {
		return nil, fmt.Errorf("fetching determinants: %w", err)
	}
	out := make([]*entity.RestrictionDeterminant, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.RestrictionDeterminant{
			ID:            row.ID,
			RestrictionID: row.RestrictionID,
			QuestionID:    row.QuestionID,
			Operator:      entity.RestrictionOperator(row.Operator),
			AnswerValue:   pgutil.FromPgTextPtr(row.AnswerValue),
		})
	}
	return out, nil
}

func rowToEntity(row sqlc.Restriction) *entity.Restriction {
	e := &entity.Restriction{
		ID:           row.ID,
		Situation:    entity.RestrictionSituation(row.Situation),
		CustomerCode: row.CustomerCode,
		ItemCode:     row.ItemCode,
		ReasonCode:   row.ReasonCode,
		DivisionID:   row.DivisionID,
		Weight:       int(row.Weight),
		CreatedAt:    pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:    pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:    pgutil.FromPgUUID(row.CreatedBy),
	}
	if row.Code.Valid {
		e.Code = row.Code.Int64
	}
	e.ClassificationType = pgutil.FromPgTextPtr(row.ClassificationType)
	e.ClassificationOrigin = pgutil.FromPgTextPtr(row.ClassificationOrigin)
	return e
}
