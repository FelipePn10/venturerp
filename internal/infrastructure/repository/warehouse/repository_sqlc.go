package warehouse

import (
	"context"
	"strconv"

	"github.com/FelipePn10/panossoerp/internal/domain/warehouse/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	mapper "github.com/FelipePn10/panossoerp/internal/infrastructure/mapper/warehouse"
)

func (r *repositoryWarehouseSQLC) Create(
	ctx context.Context,
	warehouse *entity.Warehouse,
) (*entity.Warehouse, error) {
	params := sqlc.CreateWarehouseParams{
		Code:                strconv.Itoa(warehouse.Code),
		Description:         warehouse.Description,
		Column3:             mapper.WarehouseLocationToDB(warehouse.Location),
		Column4:             mapper.WarehouseTypeToDB(warehouse.Type),
		Disposition:         warehouse.Disposition,
		ReservationsAllowed: warehouse.ReservationsAllowed,
		CreatedBy:           pgutil.ToPgUUID(warehouse.CreatedBy),
	}
	dbWarehouse, err := r.q.CreateWarehouse(ctx, params)
	if err != nil {
		return nil, err
	}

	code, _ := strconv.Atoi(dbWarehouse.Code)
	return &entity.Warehouse{
		ID:                  int32(dbWarehouse.ID),
		Code:                code,
		Description:         dbWarehouse.Description,
		Location:            mapper.WarehouseLocationToDomain(dbWarehouse.Location),
		Type:                mapper.WarehouseTypeToDomain(dbWarehouse.Type),
		Disposition:         dbWarehouse.Disposition,
		ReservationsAllowed: dbWarehouse.ReservationsAllowed,
		CreatedBy:           pgutil.FromPgUUID(dbWarehouse.CreatedBy),
		CreatedAt:           pgutil.FromPgTimestamp(dbWarehouse.CreatedAt),
	}, nil
}


func rowToWarehouse(row sqlc.CreateWarehouseRow) *entity.Warehouse {
	c, _ := strconv.Atoi(row.Code)
	return &entity.Warehouse{
		ID:                  int32(row.ID),
		Code:                c,
		Description:         row.Description,
		Location:            mapper.WarehouseLocationToDomain(row.Location),
		Type:                mapper.WarehouseTypeToDomain(row.Type),
		Disposition:         row.Disposition,
		ReservationsAllowed: row.ReservationsAllowed,
		CreatedBy:           pgutil.FromPgUUID(row.CreatedBy),
		CreatedAt:           pgutil.FromPgTimestamp(row.CreatedAt),
	}
}

func listRowToWarehouse(row sqlc.ListWarehousesRow) *entity.Warehouse {
	c, _ := strconv.Atoi(row.Code)
	return &entity.Warehouse{
		ID:                  int32(row.ID),
		Code:                c,
		Description:         row.Description,
		Location:            mapper.WarehouseLocationToDomain(row.Location),
		Type:                mapper.WarehouseTypeToDomain(row.Type),
		Disposition:         row.Disposition,
		ReservationsAllowed: row.ReservationsAllowed,
		CreatedBy:           pgutil.FromPgUUID(row.CreatedBy),
		CreatedAt:           pgutil.FromPgTimestamp(row.CreatedAt),
	}
}

func getRowToWarehouse(row sqlc.GetWarehouseByCodeRow) *entity.Warehouse {
	c, _ := strconv.Atoi(row.Code)
	return &entity.Warehouse{
		ID:                  int32(row.ID),
		Code:                c,
		Description:         row.Description,
		Location:            mapper.WarehouseLocationToDomain(row.Location),
		Type:                mapper.WarehouseTypeToDomain(row.Type),
		Disposition:         row.Disposition,
		ReservationsAllowed: row.ReservationsAllowed,
		CreatedBy:           pgutil.FromPgUUID(row.CreatedBy),
		CreatedAt:           pgutil.FromPgTimestamp(row.CreatedAt),
	}
}

func (r *repositoryWarehouseSQLC) List(ctx context.Context) ([]*entity.Warehouse, error) {
	rows, err := r.q.ListWarehouses(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.Warehouse, 0, len(rows))
	for _, row := range rows {
		out = append(out, listRowToWarehouse(row))
	}
	return out, nil
}

func (r *repositoryWarehouseSQLC) GetByCode(ctx context.Context, code string) (*entity.Warehouse, error) {
	row, err := r.q.GetWarehouseByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return getRowToWarehouse(row), nil
}
