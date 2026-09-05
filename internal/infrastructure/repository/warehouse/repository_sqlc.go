package warehouse

import (
	"context"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"

	"github.com/FelipePn10/panossoerp/internal/domain/warehouse/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/warehouse"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
)

const warehouseColumns = `id,code,description,location::text,type::text,disposition,reservations_allowed,created_by,created_at`

func scanWarehouse(row pgx.Row) (*entity.Warehouse, error) {
	var out entity.Warehouse
	var location, warehouseType string
	if err := row.Scan(&out.ID, &out.Code, &out.Description, &location, &warehouseType,
		&out.Disposition, &out.ReservationsAllowed, &out.CreatedBy, &out.CreatedAt); err != nil {
		return nil, err
	}
	out.Location = mapper.WarehouseLocationToDomain(location)
	out.Type = mapper.WarehouseTypeToDomain(warehouseType)
	return &out, nil
}

func (r *repositoryWarehouseSQLC) Create(ctx context.Context, value *entity.Warehouse) (*entity.Warehouse, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, `INSERT INTO warehouse
		(code,description,location,type,disposition,reservations_allowed,created_by,enterprise_id)
		VALUES ($1,$2,$3::warehouse_location,$4::warehouse_type,$5,$6,$7,$8) RETURNING `+warehouseColumns,
		value.Code, value.Description, mapper.WarehouseLocationToDB(value.Location), mapper.WarehouseTypeToDB(value.Type),
		value.Disposition, value.ReservationsAllowed, value.CreatedBy, enterpriseID)
	created, err := scanWarehouse(row)
	if err != nil {
		return nil, fmt.Errorf("criando almoxarifado: %w", err)
	}
	return created, nil
}

func (r *repositoryWarehouseSQLC) List(ctx context.Context) ([]*entity.Warehouse, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+warehouseColumns+` FROM warehouse WHERE enterprise_id=$1 ORDER BY code,id`, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listando almoxarifados: %w", err)
	}
	defer rows.Close()
	out := make([]*entity.Warehouse, 0)
	for rows.Next() {
		value, scanErr := scanWarehouse(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("lendo almoxarifado: %w", scanErr)
		}
		out = append(out, value)
	}
	return out, rows.Err()
}

func (r *repositoryWarehouseSQLC) GetByCode(ctx context.Context, code string) (*entity.Warehouse, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	value, err := scanWarehouse(r.pool.QueryRow(ctx, `SELECT `+warehouseColumns+` FROM warehouse WHERE code=$1 AND enterprise_id=$2`, code, enterpriseID))
	if err == pgx.ErrNoRows {
		return nil, errorsuc.NewNotFoundError("almoxarifado não encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("consultando almoxarifado: %w", err)
	}
	return value, nil
}
