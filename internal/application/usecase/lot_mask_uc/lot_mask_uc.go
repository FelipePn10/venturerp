// Package lot_mask_uc implements the Lot/Serial Mask register (Cadastro de
// Máscara de Lotes/Séries) and the automatic lot-code generation.
package lot_mask_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/lot_mask/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

type LotMaskUseCase struct {
	Q *sqlc.Queries
	// Now is overridable for tests; defaults to time.Now.
	Now func() time.Time
}

func New(q *sqlc.Queries) *LotMaskUseCase { return &LotMaskUseCase{Q: q, Now: time.Now} }

func (uc *LotMaskUseCase) now() time.Time {
	if uc.Now != nil {
		return uc.Now()
	}
	return time.Now()
}

func textOrNull(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

// ─── lot masks ────────────────────────────────────────────────────────────────

func (uc *LotMaskUseCase) Create(ctx context.Context, dto request.LotMaskDTO) (*response.LotMaskResponse, error) {
	m, err := entity.NewLotMask(dto.Application, dto.CreatedBy)
	if err != nil {
		return nil, err
	}
	row, err := uc.Q.CreateLotMask(ctx, lotMaskParams(dto, 0, m.Application))
	if err != nil {
		return nil, fmt.Errorf("criando máscara de lote: %w", err)
	}
	return lotMaskToResponse(row, nil), nil
}

func (uc *LotMaskUseCase) Update(ctx context.Context, dto request.LotMaskDTO) (*response.LotMaskResponse, error) {
	app := dto.Application
	if app == "" {
		app = "GERAL"
	}
	row, err := uc.Q.UpdateLotMask(ctx, lotMaskParams(dto, dto.ID, app))
	if err != nil {
		return nil, fmt.Errorf("atualizando máscara de lote: %w", err)
	}
	parts, _ := uc.Q.ListLotMaskParts(ctx, dto.ID)
	return lotMaskToResponse(row, parts), nil
}

func (uc *LotMaskUseCase) Get(ctx context.Context, id int64) (*response.LotMaskResponse, error) {
	row, err := uc.Q.GetLotMask(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("máscara de lote não encontrada: %w", err)
	}
	parts, _ := uc.Q.ListLotMaskParts(ctx, id)
	return lotMaskToResponse(row, parts), nil
}

func (uc *LotMaskUseCase) List(ctx context.Context, onlyActive bool) ([]*response.LotMaskResponse, error) {
	rows, err := uc.Q.ListLotMasks(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*response.LotMaskResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, lotMaskToResponse(r, nil))
	}
	return out, nil
}

func (uc *LotMaskUseCase) Deactivate(ctx context.Context, id int64) error {
	return uc.Q.DeactivateLotMask(ctx, id)
}

// ─── parts ────────────────────────────────────────────────────────────────────

func (uc *LotMaskUseCase) AddPart(ctx context.Context, lotMaskID int64, dto request.LotMaskPartDTO) (*response.LotMaskPartResponse, error) {
	p := entity.LotMaskPart{Sequence: dto.Sequence, PartType: dto.PartType, DateFormat: dto.DateFormat, Size: dto.Size}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	row, err := uc.Q.AddLotMaskPart(ctx, sqlc.LotMaskPartParams{
		LotMaskID: lotMaskID, Sequence: int32(dto.Sequence), PartType: dto.PartType, Value: dto.Value,
		Size: int32(dto.Size), DateFormat: textOrNull(p.DateFormat), ZeroOnYearChange: dto.ZeroOnYearChange,
	})
	if err != nil {
		return nil, fmt.Errorf("adicionando partição: %w", err)
	}
	return lotPartToResponse(row), nil
}

func (uc *LotMaskUseCase) UpdatePart(ctx context.Context, id int64, dto request.LotMaskPartDTO) (*response.LotMaskPartResponse, error) {
	p := entity.LotMaskPart{Sequence: dto.Sequence, PartType: dto.PartType, DateFormat: dto.DateFormat, Size: dto.Size}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	row, err := uc.Q.UpdateLotMaskPart(ctx, sqlc.LotMaskPartParams{
		ID: id, Sequence: int32(dto.Sequence), PartType: dto.PartType, Value: dto.Value,
		Size: int32(dto.Size), DateFormat: textOrNull(p.DateFormat), ZeroOnYearChange: dto.ZeroOnYearChange,
	})
	if err != nil {
		return nil, fmt.Errorf("atualizando partição: %w", err)
	}
	return lotPartToResponse(row), nil
}

func (uc *LotMaskUseCase) DeletePart(ctx context.Context, id int64) error {
	return uc.Q.DeleteLotMaskPart(ctx, id)
}

// ─── geração ──────────────────────────────────────────────────────────────────

// Generate resolves the mask (explicit id or by context) and produces a lot code,
// advancing and persisting the sequence state of the incremental parts.
func (uc *LotMaskUseCase) Generate(ctx context.Context, dto request.GenerateLotDTO) (*response.GeneratedLotResponse, error) {
	maskID, err := uc.resolveMaskID(ctx, dto)
	if err != nil {
		return nil, err
	}
	parts, err := uc.Q.ListLotMaskParts(ctx, maskID)
	if err != nil {
		return nil, fmt.Errorf("carregando partições: %w", err)
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("máscara %d não possui partições", maskID)
	}
	domainParts := make([]entity.LotMaskPart, 0, len(parts))
	for _, p := range parts {
		domainParts = append(domainParts, toDomainPart(p))
	}
	res, err := entity.Generate(domainParts, uc.now())
	if err != nil {
		return nil, err
	}
	for _, up := range res.Updates {
		if err := uc.Q.UpdateLotMaskPartState(ctx, up.PartID, up.NewCurrent, int32(up.NewYear)); err != nil {
			return nil, fmt.Errorf("atualizando estado da sequência: %w", err)
		}
	}
	return &response.GeneratedLotResponse{LotMaskID: maskID, Code: res.Code}, nil
}

func (uc *LotMaskUseCase) resolveMaskID(ctx context.Context, dto request.GenerateLotDTO) (int64, error) {
	if dto.LotMaskID != nil && *dto.LotMaskID > 0 {
		return *dto.LotMaskID, nil
	}
	app := dto.Application
	if app == "" {
		app = "GERAL"
	}
	id, err := uc.Q.ResolveLotMask(ctx, app, pgutil.ToPgInt8Ptr(dto.CustomerCode),
		pgutil.ToPgInt8Ptr(dto.ItemCode), pgutil.ToPgInt8Ptr(dto.ClassificationCode))
	if err != nil {
		return 0, fmt.Errorf("nenhuma máscara de lote aplicável para o contexto informado")
	}
	return id, nil
}

// ─── mappers ──────────────────────────────────────────────────────────────────

func lotMaskParams(dto request.LotMaskDTO, id int64, app string) sqlc.LotMaskParams {
	return sqlc.LotMaskParams{
		ID:                 id,
		Application:        app,
		CustomerCode:       pgutil.ToPgInt8Ptr(dto.CustomerCode),
		ItemCode:           pgutil.ToPgInt8Ptr(dto.ItemCode),
		ClassificationType: textOrNull(dto.ClassificationType),
		ClassificationCode: pgutil.ToPgInt8Ptr(dto.ClassificationCode),
		ZeroOnYearChange:   dto.ZeroOnYearChange,
		Description:        textOrNull(dto.Description),
		CreatedBy:          pgutil.ToPgUUID(dto.CreatedBy),
	}
}

func toDomainPart(p sqlc.DBLotMaskPart) entity.LotMaskPart {
	dp := entity.LotMaskPart{
		ID:               p.ID,
		LotMaskID:        p.LotMaskID,
		Sequence:         int(p.Sequence),
		PartType:         p.PartType,
		Value:            p.Value,
		Size:             int(p.Size),
		DateFormat:       pgutil.FromPgText(p.DateFormat),
		ZeroOnYearChange: p.ZeroOnYearChange,
		CurrentValue:     p.CurrentValue,
	}
	if p.LastYear.Valid {
		y := int(p.LastYear.Int32)
		dp.LastYear = &y
	}
	return dp
}

func lotMaskToResponse(m sqlc.DBLotMask, parts []sqlc.DBLotMaskPart) *response.LotMaskResponse {
	r := &response.LotMaskResponse{
		ID:                 m.ID,
		Application:        m.Application,
		CustomerCode:       pgutil.FromPgInt8Ptr(m.CustomerCode),
		ItemCode:           pgutil.FromPgInt8Ptr(m.ItemCode),
		ClassificationType: pgutil.FromPgText(m.ClassificationType),
		ClassificationCode: pgutil.FromPgInt8Ptr(m.ClassificationCode),
		ZeroOnYearChange:   m.ZeroOnYearChange,
		IsActive:           m.IsActive,
		Description:        pgutil.FromPgText(m.Description),
	}
	for _, p := range parts {
		r.Parts = append(r.Parts, *lotPartToResponse(p))
	}
	return r
}

func lotPartToResponse(p sqlc.DBLotMaskPart) *response.LotMaskPartResponse {
	out := &response.LotMaskPartResponse{
		ID:               p.ID,
		LotMaskID:        p.LotMaskID,
		Sequence:         int(p.Sequence),
		PartType:         p.PartType,
		Value:            p.Value,
		Size:             int(p.Size),
		DateFormat:       pgutil.FromPgText(p.DateFormat),
		ZeroOnYearChange: p.ZeroOnYearChange,
		CurrentValue:     p.CurrentValue,
	}
	if p.LastYear.Valid {
		y := int(p.LastYear.Int32)
		out.LastYear = &y
	}
	return out
}
