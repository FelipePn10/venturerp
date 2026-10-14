package stock_movement

import (
	"context"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"

	"github.com/FelipePn10/panossoerp/internal/domain/stock_movement/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/stock_movement/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StockMovementTypeRepositoryPG struct {
	pool *pgxpool.Pool
}

var _ domainrepo.StockMovementTypeRepository = (*StockMovementTypeRepositoryPG)(nil)

func New(pool *pgxpool.Pool) domainrepo.StockMovementTypeRepository {
	return &StockMovementTypeRepositoryPG{pool: pool}
}

// Tipo de movimento de estoque é catálogo da empresa (sigla única por empresa).

func (r *StockMovementTypeRepositoryPG) Create(ctx context.Context, s *entity.StockMovementType) (*entity.StockMovementType, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx,
		`INSERT INTO stock_movement_types
		 (sigla, description, usage_type, entry_order, exit_order, considers_consumption, updates_avg_cost,
		  is_adjustment, updates_cycle_count, shows_in_summary, entry_exit, generates_fci_movement, is_active, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,TRUE,$13) RETURNING id, created_at`,
		s.Sigla, s.Description, string(s.UsageType), s.EntryOrder, s.ExitOrder,
		s.ConsidersConsumption, s.UpdatesAvgCost, s.IsAdjustment, s.UpdatesCycleCount,
		s.ShowsInSummary, string(s.EntryExit), s.GeneratesFCIMovement, empresa,
	).Scan(&s.ID, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating stock movement type: %w", err)
	}
	return s, nil
}

func (r *StockMovementTypeRepositoryPG) Update(ctx context.Context, s *entity.StockMovementType) (*entity.StockMovementType, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE stock_movement_types
		 SET description=$1, usage_type=$2, entry_order=$3, exit_order=$4, considers_consumption=$5,
		     updates_avg_cost=$6, is_adjustment=$7, updates_cycle_count=$8, shows_in_summary=$9,
		     entry_exit=$10, generates_fci_movement=$11, is_active=$12
		 WHERE id=$13 AND enterprise_id=$14`,
		s.Description, string(s.UsageType), s.EntryOrder, s.ExitOrder, s.ConsidersConsumption,
		s.UpdatesAvgCost, s.IsAdjustment, s.UpdatesCycleCount, s.ShowsInSummary,
		string(s.EntryExit), s.GeneratesFCIMovement, s.IsActive, s.ID, empresa)
	if err != nil {
		return nil, fmt.Errorf("updating stock movement type %d: %w", s.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("tipo de movimento de estoque %d não encontrado", s.ID))
	}
	return s, nil
}

func (r *StockMovementTypeRepositoryPG) GetByID(ctx context.Context, id int64) (*entity.StockMovementType, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx,
		`SELECT id, sigla, description, usage_type, entry_order, exit_order, considers_consumption,
		 updates_avg_cost, is_adjustment, updates_cycle_count, shows_in_summary, entry_exit, generates_fci_movement, is_active, created_at
		 FROM stock_movement_types WHERE id=$1 AND enterprise_id=$2`, id, empresa)
	s, err := scanSMT(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("tipo de movimento de estoque %d não encontrado", id))
		}
		return nil, fmt.Errorf("getting stock movement type: %w", err)
	}
	return s, nil
}

func (r *StockMovementTypeRepositoryPG) GetBySigla(ctx context.Context, sigla string) (*entity.StockMovementType, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx,
		`SELECT id, sigla, description, usage_type, entry_order, exit_order, considers_consumption,
		 updates_avg_cost, is_adjustment, updates_cycle_count, shows_in_summary, entry_exit, generates_fci_movement, is_active, created_at
		 FROM stock_movement_types WHERE sigla=$1 AND enterprise_id=$2`, sigla, empresa)
	s, err := scanSMT(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("tipo de movimento de estoque %s não encontrado", sigla))
		}
		return nil, fmt.Errorf("getting stock movement type by sigla: %w", err)
	}
	return s, nil
}

func (r *StockMovementTypeRepositoryPG) List(ctx context.Context, onlyActive bool) ([]*entity.StockMovementType, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	q := `SELECT id, sigla, description, usage_type, entry_order, exit_order, considers_consumption,
	      updates_avg_cost, is_adjustment, updates_cycle_count, shows_in_summary, entry_exit, generates_fci_movement, is_active, created_at
	      FROM stock_movement_types WHERE enterprise_id = $1`
	if onlyActive {
		q += ` AND is_active = TRUE`
	}
	q += ` ORDER BY sigla`
	rows, err := r.pool.Query(ctx, q, empresa)
	if err != nil {
		return nil, fmt.Errorf("listing stock movement types: %w", err)
	}
	defer rows.Close()
	var out []*entity.StockMovementType
	for rows.Next() {
		s, err := scanSMT(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanSMT(row scannable) (*entity.StockMovementType, error) {
	var s entity.StockMovementType
	var usage, dir string
	err := row.Scan(&s.ID, &s.Sigla, &s.Description, &usage, &s.EntryOrder, &s.ExitOrder,
		&s.ConsidersConsumption, &s.UpdatesAvgCost, &s.IsAdjustment, &s.UpdatesCycleCount,
		&s.ShowsInSummary, &dir, &s.GeneratesFCIMovement, &s.IsActive, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	s.UsageType = entity.UsageType(usage)
	s.EntryExit = entity.Direction(dir)
	return &s, nil
}
