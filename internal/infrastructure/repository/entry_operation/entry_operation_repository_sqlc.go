package entry_operation

import (
	"context"
	"errors"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"

	"github.com/FelipePn10/panossoerp/internal/domain/entry_operation/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/entry_operation/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EntryOperationRepositorySQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func New(q *sqlc.Queries, pool *pgxpool.Pool) domainrepo.EntryOperationRepository {
	return &EntryOperationRepositorySQLC{q: q, pool: pool}
}

// ─── State Groups ─────────────────────────────────────────────────────────────

func (r *EntryOperationRepositorySQLC) CreateStateGroup(ctx context.Context, g *entity.StateGroup) (*entity.StateGroup, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateStateGroup(ctx, sqlc.CreateStateGroupParams{EnterpriseID: empresa,
		Code:        g.Code,
		Description: g.Description,
		CreatedBy:   pgutil.ToPgUUID(g.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating state group: %w", err)
	}
	return stateGroupToEntity(row), nil
}

func (r *EntryOperationRepositorySQLC) GetStateGroupByCode(ctx context.Context, code int64) (*entity.StateGroup, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetStateGroupByCode(ctx, sqlc.GetStateGroupByCodeParams{Code: code, EnterpriseID: empresa})
	if err != nil {
		return nil, fmt.Errorf("state group %d not found: %w", code, err)
	}
	g := stateGroupToEntity(row)
	if g.UFs, err = r.ListStateGroupUFs(ctx, code); err != nil {
		return nil, err
	}
	return g, nil
}

func (r *EntryOperationRepositorySQLC) ListStateGroups(ctx context.Context) ([]*entity.StateGroup, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListStateGroups(ctx, empresa)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.StateGroup, 0, len(rows))
	for _, row := range rows {
		out = append(out, stateGroupToEntity(row))
	}
	return out, nil
}

func (r *EntryOperationRepositorySQLC) NextStateGroupCode(ctx context.Context) (int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	v, err := r.q.NextStateGroupCode(ctx, empresa)
	return int64(v), err
}

func (r *EntryOperationRepositorySQLC) AddStateGroupUF(ctx context.Context, stateGroupCode int64, uf string) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	if _, err := r.q.AddStateGroupUF(ctx, sqlc.AddStateGroupUFParams{StateGroupCode: stateGroupCode, Uf: uf, EnterpriseID: empresa}); err != nil {
		return err
	}
	// Zero linhas: já estava no grupo, ou o grupo não é desta empresa.
	_, err = r.q.GetStateGroupByCode(ctx, sqlc.GetStateGroupByCodeParams{Code: stateGroupCode, EnterpriseID: empresa})
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewNotFoundError(fmt.Sprintf("grupo de UF %d não encontrado", stateGroupCode))
	}
	return err
}

func (r *EntryOperationRepositorySQLC) ListStateGroupUFs(ctx context.Context, stateGroupCode int64) ([]string, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return r.q.ListStateGroupUFs(ctx, sqlc.ListStateGroupUFsParams{StateGroupCode: stateGroupCode, EnterpriseID: empresa})
}

func (r *EntryOperationRepositorySQLC) UFInGroup(ctx context.Context, stateGroupCode int64, uf string) (bool, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	return r.q.UFInStateGroup(ctx, sqlc.UFInStateGroupParams{EnterpriseID: empresa, StateGroupCode: stateGroupCode, Uf: uf})
}

// ─── Entry Operation Types ────────────────────────────────────────────────────

func (r *EntryOperationRepositorySQLC) CreateEntryOperation(ctx context.Context, o *entity.EntryOperationType) (*entity.EntryOperationType, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateEntryOperationType(ctx, sqlc.CreateEntryOperationTypeParams{EnterpriseID: empresa,
		Code:               o.Code,
		Description:        o.Description,
		InvoiceTypeCode:    o.InvoiceTypeCode,
		NatureOperation:    o.NatureOperation,
		ClassificationType: pgutil.ToPgTextFromPtr(o.ClassificationType),
		ClassificationCode: pgutil.ToPgTextFromPtr(o.ClassificationCode),
		StateGroupCode:     o.StateGroupCode,
		SupplierTypeCode:   o.SupplierTypeCode,
		CreatedBy:          pgutil.ToPgUUID(o.CreatedBy),
		MovimentaEstoque:   o.MovimentaEstoque,
		GeraFinanceiro:     o.GeraFinanceiro,
		CreditaIcms:        o.CreditaICMS,
		CreditaIpi:         o.CreditaIPI,
		CreditaPisCofins:   o.CreditaPISCOFINS,
	})
	if err != nil {
		return nil, fmt.Errorf("creating entry operation type: %w", err)
	}
	return entryOpToEntity(row), nil
}

func (r *EntryOperationRepositorySQLC) UpdateEntryOperation(ctx context.Context, o *entity.EntryOperationType) (*entity.EntryOperationType, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.UpdateEntryOperationType(ctx, sqlc.UpdateEntryOperationTypeParams{EnterpriseID: empresa,
		Code:               o.Code,
		Description:        o.Description,
		InvoiceTypeCode:    o.InvoiceTypeCode,
		NatureOperation:    o.NatureOperation,
		ClassificationType: pgutil.ToPgTextFromPtr(o.ClassificationType),
		ClassificationCode: pgutil.ToPgTextFromPtr(o.ClassificationCode),
		StateGroupCode:     o.StateGroupCode,
		SupplierTypeCode:   o.SupplierTypeCode,
		IsActive:           o.IsActive,
		MovimentaEstoque:   o.MovimentaEstoque,
		GeraFinanceiro:     o.GeraFinanceiro,
		CreditaIcms:        o.CreditaICMS,
		CreditaIpi:         o.CreditaIPI,
		CreditaPisCofins:   o.CreditaPISCOFINS,
	})
	if err != nil {
		return nil, fmt.Errorf("updating entry operation type: %w", err)
	}
	return entryOpToEntity(row), nil
}

func (r *EntryOperationRepositorySQLC) GetEntryOperationByCode(ctx context.Context, code int64) (*entity.EntryOperationType, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetEntryOperationTypeByCode(ctx, sqlc.GetEntryOperationTypeByCodeParams{Code: code, EnterpriseID: empresa})
	if err != nil {
		return nil, fmt.Errorf("entry operation type %d not found: %w", code, err)
	}
	return entryOpToEntity(row), nil
}

func (r *EntryOperationRepositorySQLC) ListEntryOperations(ctx context.Context, onlyActive bool) ([]*entity.EntryOperationType, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListEntryOperationTypes(ctx, sqlc.ListEntryOperationTypesParams{Column1: onlyActive, EnterpriseID: empresa})
	if err != nil {
		return nil, err
	}
	out := make([]*entity.EntryOperationType, 0, len(rows))
	for _, row := range rows {
		out = append(out, entryOpToEntity(row))
	}
	return out, nil
}

func (r *EntryOperationRepositorySQLC) NextEntryOperationCode(ctx context.Context) (int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	v, err := r.q.NextEntryOperationTypeCode(ctx, empresa)
	return int64(v), err
}

// ─── Mappers ──────────────────────────────────────────────────────────────────

func stateGroupToEntity(row sqlc.StateGroup) *entity.StateGroup {
	return &entity.StateGroup{
		ID:          row.ID,
		Code:        row.Code,
		Description: row.Description,
		IsActive:    row.IsActive,
		CreatedAt:   pgutil.FromPgTimestamptz(row.CreatedAt),
		CreatedBy:   pgutil.FromPgUUID(row.CreatedBy),
	}
}

func entryOpToEntity(row sqlc.EntryOperationType) *entity.EntryOperationType {
	return &entity.EntryOperationType{
		ID:                 row.ID,
		Code:               row.Code,
		Description:        row.Description,
		InvoiceTypeCode:    row.InvoiceTypeCode,
		NatureOperation:    row.NatureOperation,
		ClassificationType: pgutil.FromPgTextPtr(row.ClassificationType),
		ClassificationCode: pgutil.FromPgTextPtr(row.ClassificationCode),
		StateGroupCode:     row.StateGroupCode,
		SupplierTypeCode:   row.SupplierTypeCode,
		MovimentaEstoque:   row.MovimentaEstoque,
		GeraFinanceiro:     row.GeraFinanceiro,
		CreditaICMS:        row.CreditaIcms,
		CreditaIPI:         row.CreditaIpi,
		CreditaPISCOFINS:   row.CreditaPisCofins,
		IsActive:           row.IsActive,
		CreatedAt:          pgutil.FromPgTimestamptz(row.CreatedAt),
		CreatedBy:          pgutil.FromPgUUID(row.CreatedBy),
	}
}
