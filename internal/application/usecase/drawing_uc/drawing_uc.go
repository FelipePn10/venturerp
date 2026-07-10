// Package drawing_uc implements the Drawing register (Cadastro de Desenhos) with
// revisions, distributions and configurator-characteristic links.
package drawing_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/drawing/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
	"github.com/jackc/pgx/v5/pgtype"
)

type DrawingUseCase struct {
	Q *sqlc.Queries
}

func New(q *sqlc.Queries) *DrawingUseCase { return &DrawingUseCase{Q: q} }

func textOrNull(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func numPtr(n pgtype.Numeric) *float64 {
	if !n.Valid {
		return nil
	}
	v := pgutil.FromPgNumericToFloat64(n)
	return &v
}

// ─── drawings ─────────────────────────────────────────────────────────────────

func (uc *DrawingUseCase) Create(ctx context.Context, dto request.DrawingDTO) (*response.DrawingResponse, error) {
	d, err := entity.NewDrawing(dto.Code, dto.Digit, dto.Format, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	row, err := uc.Q.CreateDrawing(ctx, drawingParams(dto, 0))
	if err != nil {
		return nil, fmt.Errorf("criando desenho: %w", err)
	}
	_ = d
	return drawingToResponse(row, nil), nil
}

func (uc *DrawingUseCase) Update(ctx context.Context, dto request.DrawingDTO) (*response.DrawingResponse, error) {
	if dto.Code == "" {
		return nil, fmt.Errorf("código do desenho é obrigatório")
	}
	row, err := uc.Q.UpdateDrawing(ctx, drawingParams(dto, dto.ID))
	if err != nil {
		return nil, fmt.Errorf("atualizando desenho: %w", err)
	}
	revs, _ := uc.listRevisions(ctx, row)
	return drawingToResponse(row, revs), nil
}

func (uc *DrawingUseCase) Get(ctx context.Context, id int64) (*response.DrawingResponse, error) {
	row, err := uc.Q.GetDrawing(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("desenho não encontrado: %w", err)
	}
	revs, _ := uc.listRevisions(ctx, row)
	return drawingToResponse(row, revs), nil
}

func (uc *DrawingUseCase) List(ctx context.Context, onlyActive bool, search string) ([]*response.DrawingResponse, error) {
	rows, err := uc.Q.ListDrawings(ctx, onlyActive, search)
	if err != nil {
		return nil, err
	}
	out := make([]*response.DrawingResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, drawingToResponse(r, nil))
	}
	return out, nil
}

func (uc *DrawingUseCase) Deactivate(ctx context.Context, id int64) error {
	return uc.Q.DeactivateDrawing(ctx, id)
}

// ─── revisions ────────────────────────────────────────────────────────────────

func (uc *DrawingUseCase) AddRevision(ctx context.Context, drawingID int64, dto request.DrawingRevisionDTO) (*response.DrawingRevisionResponse, error) {
	drawing, err := uc.Q.GetDrawing(ctx, drawingID)
	if err != nil {
		return nil, fmt.Errorf("desenho não encontrado: %w", err)
	}
	rev := &entity.DrawingRevision{
		Revision:  dto.Revision,
		StartDate: datetime.ParseDatePtr(&dto.StartDate),
		EndDate:   datetime.ParseDatePtr(&dto.EndDate),
	}
	if err := rev.Validate(); err != nil {
		return nil, err
	}
	row, err := uc.Q.AddDrawingRevision(ctx, revisionParams(drawingID, dto, 0))
	if err != nil {
		return nil, fmt.Errorf("adicionando revisão: %w", err)
	}
	if dto.IsCurrent {
		_ = uc.Q.SetCurrentDrawingRevision(ctx, drawingID, row.ID)
		row.IsCurrent = true
	}
	return revisionToResponse(drawing, row, nil), nil
}

func (uc *DrawingUseCase) UpdateRevision(ctx context.Context, id int64, dto request.DrawingRevisionDTO) (*response.DrawingRevisionResponse, error) {
	dto.ID = id
	row, err := uc.Q.UpdateDrawingRevision(ctx, revisionParams(0, dto, id))
	if err != nil {
		return nil, fmt.Errorf("atualizando revisão: %w", err)
	}
	if dto.IsCurrent {
		_ = uc.Q.SetCurrentDrawingRevision(ctx, row.DrawingID, row.ID)
	}
	drawing, _ := uc.Q.GetDrawing(ctx, row.DrawingID)
	dists, _ := uc.Q.ListDrawingDistributions(ctx, row.ID)
	return revisionToResponse(drawing, row, dists), nil
}

func (uc *DrawingUseCase) ListRevisions(ctx context.Context, drawingID int64) ([]*response.DrawingRevisionResponse, error) {
	drawing, err := uc.Q.GetDrawing(ctx, drawingID)
	if err != nil {
		return nil, fmt.Errorf("desenho não encontrado: %w", err)
	}
	rows, err := uc.Q.ListDrawingRevisions(ctx, drawingID)
	if err != nil {
		return nil, err
	}
	out := make([]*response.DrawingRevisionResponse, 0, len(rows))
	for _, r := range rows {
		dists, _ := uc.Q.ListDrawingDistributions(ctx, r.ID)
		out = append(out, revisionToResponse(drawing, r, dists))
	}
	return out, nil
}

func (uc *DrawingUseCase) DeleteRevision(ctx context.Context, id int64) error {
	return uc.Q.DeleteDrawingRevision(ctx, id)
}

// ─── distributions ────────────────────────────────────────────────────────────

func (uc *DrawingUseCase) AddDistribution(ctx context.Context, revisionID int64, dto request.DrawingDistributionDTO) (*response.DrawingDistributionResponse, error) {
	if dto.Recipient == "" {
		return nil, fmt.Errorf("destinatário é obrigatório")
	}
	row, err := uc.Q.AddDrawingDistribution(ctx, revisionID, dto.Recipient,
		pgutil.ToPgDateFromPtr(datetime.ParseDatePtr(&dto.DistributedAt)), textOrNull(dto.Notes))
	if err != nil {
		return nil, fmt.Errorf("adicionando distribuição: %w", err)
	}
	return &response.DrawingDistributionResponse{
		ID: row.ID, RevisionID: row.RevisionID, Recipient: row.Recipient,
		DistributedAt: pgutil.FromPgDateToPtr(row.DistributedAt), Notes: pgutil.FromPgText(row.Notes),
	}, nil
}

func (uc *DrawingUseCase) DeleteDistribution(ctx context.Context, id int64) error {
	return uc.Q.DeleteDrawingDistribution(ctx, id)
}

// ─── characteristics link ─────────────────────────────────────────────────────

func (uc *DrawingUseCase) AddCharacteristic(ctx context.Context, drawingID int64, dto request.DrawingCharacteristicDTO) (*response.DrawingCharacteristicResponse, error) {
	op := dto.Operator
	if op == "" {
		op = "EQUAL"
	}
	row, err := uc.Q.AddDrawingCharacteristic(ctx, drawingID, dto.CharacteristicID, op, pgutil.ToPgInt8Ptr(dto.VariableID))
	if err != nil {
		return nil, fmt.Errorf("vinculando característica ao desenho: %w", err)
	}
	return &response.DrawingCharacteristicResponse{
		ID: row.ID, DrawingID: row.DrawingID, CharacteristicID: row.CharacteristicID,
		Operator: row.Operator, VariableID: pgutil.FromPgInt8Ptr(row.VariableID),
	}, nil
}

func (uc *DrawingUseCase) ListCharacteristics(ctx context.Context, drawingID int64) ([]*response.DrawingCharacteristicResponse, error) {
	rows, err := uc.Q.ListDrawingCharacteristics(ctx, drawingID)
	if err != nil {
		return nil, err
	}
	out := make([]*response.DrawingCharacteristicResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, &response.DrawingCharacteristicResponse{
			ID: r.ID, DrawingID: r.DrawingID, CharacteristicID: r.CharacteristicID,
			Operator: r.Operator, VariableID: pgutil.FromPgInt8Ptr(r.VariableID),
		})
	}
	return out, nil
}

func (uc *DrawingUseCase) DeleteCharacteristic(ctx context.Context, id int64) error {
	return uc.Q.DeleteDrawingCharacteristic(ctx, id)
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func (uc *DrawingUseCase) listRevisions(ctx context.Context, d sqlc.DBDrawing) ([]sqlc.DBDrawingRevision, error) {
	return uc.Q.ListDrawingRevisions(ctx, d.ID)
}

func drawingParams(dto request.DrawingDTO, id int64) sqlc.DrawingParams {
	return sqlc.DrawingParams{
		ID:           id,
		Code:         dto.Code,
		Digit:        dto.Digit,
		Format:       dto.Format,
		Model:        textOrNull(dto.Model),
		ItemCode:     pgutil.ToPgInt8Ptr(dto.ItemCode),
		Description:  textOrNull(dto.Description),
		Uom:          textOrNull(dto.UOM),
		Weight:       pgutil.ToPgNumericFromFloat64Ptr(dto.Weight),
		MaterialSpec: textOrNull(dto.MaterialSpec),
		CreationDate: pgutil.ToPgDateFromPtr(datetime.ParseDatePtr(&dto.CreationDate)),
		CreatedBy:    pgutil.ToPgUUID(dto.CreatedBy),
	}
}

func revisionParams(drawingID int64, dto request.DrawingRevisionDTO, id int64) sqlc.DrawingRevisionParams {
	return sqlc.DrawingRevisionParams{
		ID:           id,
		DrawingID:    drawingID,
		Revision:     dto.Revision,
		StartDate:    pgutil.ToPgDateFromPtr(datetime.ParseDatePtr(&dto.StartDate)),
		EndDate:      pgutil.ToPgDateFromPtr(datetime.ParseDatePtr(&dto.EndDate)),
		MaterialSpec: textOrNull(dto.MaterialSpec),
		Reason:       textOrNull(dto.Reason),
		ApprovedBy:   textOrNull(dto.ApprovedBy),
		ApprovalDate: pgutil.ToPgDateFromPtr(datetime.ParseDatePtr(&dto.ApprovalDate)),
		IsCurrent:    dto.IsCurrent,
	}
}

func drawingToResponse(d sqlc.DBDrawing, revs []sqlc.DBDrawingRevision) *response.DrawingResponse {
	r := &response.DrawingResponse{
		ID:           d.ID,
		Code:         d.Code,
		Digit:        d.Digit,
		Format:       d.Format,
		Model:        pgutil.FromPgText(d.Model),
		ItemCode:     pgutil.FromPgInt8Ptr(d.ItemCode),
		Description:  pgutil.FromPgText(d.Description),
		UOM:          pgutil.FromPgText(d.Uom),
		Weight:       numPtr(d.Weight),
		MaterialSpec: pgutil.FromPgText(d.MaterialSpec),
		CreationDate: pgutil.FromPgDateToPtr(d.CreationDate),
		IsActive:     d.IsActive,
	}
	for _, rev := range revs {
		r.Revisions = append(r.Revisions, *revisionToResponse(d, rev, nil))
	}
	return r
}

func revisionToResponse(d sqlc.DBDrawing, rev sqlc.DBDrawingRevision, dists []sqlc.DBDrawingDistribution) *response.DrawingRevisionResponse {
	ent := entity.Drawing{Code: d.Code, Digit: d.Digit, Format: d.Format}
	out := &response.DrawingRevisionResponse{
		ID:            rev.ID,
		DrawingID:     rev.DrawingID,
		Revision:      rev.Revision,
		CompositeCode: ent.CompositeCode(rev.Revision),
		StartDate:     pgutil.FromPgDateToPtr(rev.StartDate),
		EndDate:       pgutil.FromPgDateToPtr(rev.EndDate),
		MaterialSpec:  pgutil.FromPgText(rev.MaterialSpec),
		Reason:        pgutil.FromPgText(rev.Reason),
		ApprovedBy:    pgutil.FromPgText(rev.ApprovedBy),
		ApprovalDate:  pgutil.FromPgDateToPtr(rev.ApprovalDate),
		IsCurrent:     rev.IsCurrent,
	}
	for _, dist := range dists {
		out.Distributions = append(out.Distributions, response.DrawingDistributionResponse{
			ID: dist.ID, RevisionID: dist.RevisionID, Recipient: dist.Recipient,
			DistributedAt: pgutil.FromPgDateToPtr(dist.DistributedAt), Notes: pgutil.FromPgText(dist.Notes),
		})
	}
	return out
}
