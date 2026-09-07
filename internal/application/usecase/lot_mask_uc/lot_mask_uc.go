// Package lot_mask_uc implements the Lot/Serial Mask register (Cadastro de
// Máscara de Lotes/Séries) and the automatic lot-code generation.
package lot_mask_uc

import (
	"context"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	"github.com/FelipePn10/panossoerp/internal/domain/lot_mask/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LotMaskUseCase struct {
	Q        *sqlc.Queries
	Pool     *pgxpool.Pool
	ItemRepo any
	// Now is overridable for tests; defaults to time.Now.
	Now func() time.Time
}

func New(q *sqlc.Queries, itemRepo ...any) *LotMaskUseCase {
	uc := &LotMaskUseCase{Q: q, Now: time.Now}
	for _, dependency := range itemRepo {
		if pool, ok := dependency.(*pgxpool.Pool); ok {
			uc.Pool = pool
		} else {
			uc.ItemRepo = dependency
		}
	}
	return uc
}

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
	itemCode, err := uc.resolveItemCode(ctx, dto.ItemCode)
	if err != nil {
		return nil, err
	}
	params := lotMaskParams(dto, itemCode, 0, m.Application)
	var row sqlc.DBLotMask
	if uc.Pool != nil {
		enterpriseID, tenantErr := tenant.ID(ctx)
		if tenantErr != nil {
			return nil, tenantErr
		}
		row, err = scanTenantLotMask(uc.Pool.QueryRow(ctx, `INSERT INTO lot_masks
		 (application,customer_code,item_code,classification_type,classification_code,zero_on_year_change,description,created_by,enterprise_id)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING `+tenantLotMaskCols,
			params.Application, params.CustomerCode, params.ItemCode, params.ClassificationType, params.ClassificationCode,
			params.ZeroOnYearChange, params.Description, params.CreatedBy, enterpriseID))
	} else {
		row, err = uc.Q.CreateLotMask(ctx, params)
	}
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
	if !entity.ValidApplication(app) {
		return nil, errorsuc.NewValidationError("aplicação deve ser SUPRIMENTOS, PRODUCAO, VENDAS, EXPEDICAO ou GERAL")
	}
	itemCode, err := uc.resolveItemCode(ctx, dto.ItemCode)
	if err != nil {
		return nil, err
	}
	params := lotMaskParams(dto, itemCode, dto.ID, app)
	var row sqlc.DBLotMask
	if uc.Pool != nil {
		enterpriseID, tenantErr := tenant.ID(ctx)
		if tenantErr != nil {
			return nil, tenantErr
		}
		row, err = scanTenantLotMask(uc.Pool.QueryRow(ctx, `UPDATE lot_masks SET application=$2,customer_code=$3,item_code=$4,
		 classification_type=$5,classification_code=$6,zero_on_year_change=$7,description=$8,updated_at=NOW()
		 WHERE id=$1 AND enterprise_id=$9 RETURNING `+tenantLotMaskCols, params.ID, params.Application, params.CustomerCode,
			params.ItemCode, params.ClassificationType, params.ClassificationCode, params.ZeroOnYearChange, params.Description, enterpriseID))
	} else {
		row, err = uc.Q.UpdateLotMask(ctx, params)
	}
	if err != nil {
		return nil, fmt.Errorf("atualizando máscara de lote: %w", err)
	}
	parts, _ := uc.Q.ListLotMaskParts(ctx, dto.ID)
	return lotMaskToResponse(row, parts), nil
}

func (uc *LotMaskUseCase) Get(ctx context.Context, id int64) (*response.LotMaskResponse, error) {
	var row sqlc.DBLotMask
	var err error
	if uc.Pool != nil {
		enterpriseID, tenantErr := tenant.ID(ctx)
		if tenantErr != nil {
			return nil, tenantErr
		}
		row, err = scanTenantLotMask(uc.Pool.QueryRow(ctx, `SELECT `+tenantLotMaskCols+` FROM lot_masks WHERE id=$1 AND enterprise_id=$2`, id, enterpriseID))
	} else {
		row, err = uc.Q.GetLotMask(ctx, id)
	}
	if err != nil {
		return nil, fmt.Errorf("máscara de lote não encontrada: %w", err)
	}
	parts, _ := uc.Q.ListLotMaskParts(ctx, id)
	return lotMaskToResponse(row, parts), nil
}

func (uc *LotMaskUseCase) List(ctx context.Context, onlyActive bool) ([]*response.LotMaskResponse, error) {
	rows, err := uc.listTenantMasks(ctx, onlyActive)
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
	if uc.Pool != nil {
		enterpriseID, err := tenant.ID(ctx)
		if err != nil {
			return err
		}
		tag, err := uc.Pool.Exec(ctx, `UPDATE lot_masks SET is_active=FALSE,updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`, id, enterpriseID)
		if err == nil && tag.RowsAffected() == 0 {
			return errorsuc.NewNotFoundError("máscara de lote não encontrada")
		}
		return err
	}
	return uc.Q.DeactivateLotMask(ctx, id)
}

// ─── parts ────────────────────────────────────────────────────────────────────

func (uc *LotMaskUseCase) AddPart(ctx context.Context, lotMaskID int64, dto request.LotMaskPartDTO) (*response.LotMaskPartResponse, error) {
	if err := uc.ensureMaskTenant(ctx, lotMaskID); err != nil {
		return nil, err
	}
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
	if err := uc.ensurePartTenant(ctx, id); err != nil {
		return nil, err
	}
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
	if err := uc.ensurePartTenant(ctx, id); err != nil {
		return err
	}
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

const tenantLotMaskCols = `id,application,customer_code,item_code,classification_type,classification_code,zero_on_year_change,is_active,description,created_at,updated_at,created_by`

type maskScanner interface{ Scan(...any) error }

func scanTenantLotMask(row maskScanner) (sqlc.DBLotMask, error) {
	var value sqlc.DBLotMask
	err := row.Scan(&value.ID, &value.Application, &value.CustomerCode, &value.ItemCode, &value.ClassificationType, &value.ClassificationCode,
		&value.ZeroOnYearChange, &value.IsActive, &value.Description, &value.CreatedAt, &value.UpdatedAt, &value.CreatedBy)
	return value, err
}
func (uc *LotMaskUseCase) listTenantMasks(ctx context.Context, onlyActive bool) ([]sqlc.DBLotMask, error) {
	if uc.Pool == nil {
		return uc.Q.ListLotMasks(ctx, onlyActive)
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := uc.Pool.Query(ctx, `SELECT `+tenantLotMaskCols+` FROM lot_masks WHERE enterprise_id=$1 AND ($2::boolean=FALSE OR is_active) ORDER BY application,id`, enterpriseID, onlyActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]sqlc.DBLotMask, 0)
	for rows.Next() {
		value, scanErr := scanTenantLotMask(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, value)
	}
	return out, rows.Err()
}
func (uc *LotMaskUseCase) ensureMaskTenant(ctx context.Context, id int64) error {
	if uc.Pool == nil {
		return nil
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var valid bool
	err = uc.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM lot_masks WHERE id=$1 AND enterprise_id=$2)`, id, enterpriseID).Scan(&valid)
	if err != nil {
		return err
	}
	if !valid {
		return errorsuc.NewNotFoundError("máscara de lote não encontrada")
	}
	return nil
}

func (uc *LotMaskUseCase) ensurePartTenant(ctx context.Context, id int64) error {
	if uc.Pool == nil {
		return nil
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var valid bool
	err = uc.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM lot_mask_parts p JOIN lot_masks m ON m.id=p.lot_mask_id WHERE p.id=$1 AND m.enterprise_id=$2)`, id, enterpriseID).Scan(&valid)
	if err != nil {
		return err
	}
	if !valid {
		return errorsuc.NewNotFoundError("partição da máscara de lote não encontrada")
	}
	return nil
}

func (uc *LotMaskUseCase) resolveMaskID(ctx context.Context, dto request.GenerateLotDTO) (int64, error) {
	if dto.LotMaskID != nil && *dto.LotMaskID > 0 {
		if err := uc.ensureMaskTenant(ctx, *dto.LotMaskID); err != nil {
			return 0, err
		}
		return *dto.LotMaskID, nil
	}
	app := dto.Application
	if app == "" {
		app = "GERAL"
	}
	itemCode, resolveErr := uc.resolveItemCode(ctx, dto.ItemCode)
	if resolveErr != nil {
		return 0, resolveErr
	}
	var id int64
	var err error
	if uc.Pool != nil {
		enterpriseID, tenantErr := tenant.ID(ctx)
		if tenantErr != nil {
			return 0, tenantErr
		}
		err = uc.Pool.QueryRow(ctx, `SELECT id FROM lot_masks WHERE enterprise_id=$1 AND is_active AND application=$2
		 AND (customer_code IS NULL OR customer_code=$3) AND (item_code IS NULL OR item_code=$4)
		 AND (classification_code IS NULL OR classification_code=$5)
		 ORDER BY (customer_code IS NOT NULL)::int*8+(item_code IS NOT NULL)::int*4+(classification_code IS NOT NULL)::int*2 DESC,id LIMIT 1`,
			enterpriseID, app, pgutil.ToPgInt8Ptr(dto.CustomerCode), pgutil.ToPgInt8Ptr(itemCode), pgutil.ToPgInt8Ptr(dto.ClassificationCode)).Scan(&id)
	} else {
		id, err = uc.Q.ResolveLotMask(ctx, app, pgutil.ToPgInt8Ptr(dto.CustomerCode), pgutil.ToPgInt8Ptr(itemCode), pgutil.ToPgInt8Ptr(dto.ClassificationCode))
	}
	if err != nil {
		return 0, fmt.Errorf("nenhuma máscara de lote aplicável para o contexto informado")
	}
	return id, nil
}

// ─── mappers ──────────────────────────────────────────────────────────────────

func (uc *LotMaskUseCase) resolveItemCode(ctx context.Context, raw *request.TextCode) (*int64, error) {
	if raw == nil {
		return nil, nil
	}
	if uc.ItemRepo == nil {
		return nil, fmt.Errorf("repositório de itens não configurado")
	}
	item, err := itemresolution.Resolve(ctx, uc.ItemRepo, *raw)
	if err != nil {
		return nil, err
	}
	code := int64(item.Code)
	return &code, nil
}

func lotMaskParams(dto request.LotMaskDTO, itemCode *int64, id int64, app string) sqlc.LotMaskParams {
	return sqlc.LotMaskParams{
		ID:                 id,
		Application:        app,
		CustomerCode:       pgutil.ToPgInt8Ptr(dto.CustomerCode),
		ItemCode:           pgutil.ToPgInt8Ptr(itemCode),
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
