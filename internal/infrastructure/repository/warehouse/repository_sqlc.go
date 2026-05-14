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
		Location:            mapper.WarehouseLocationToDB(warehouse.Location),
		Type:                mapper.WarehouseTypeToDB(warehouse.Type),
		Disposition:         warehouse.Disposition,
		ReservationsAllowed: warehouse.ReservationsAllowed,
		CreatedBy:           pgutil.ToPgUUID(warehouse.CreatedBy),
	}
	dbWarehouse, err := r.q.CreateWarehouse(ctx, params)
	if err != nil {
		return nil, err
	}

	code, _ := strconv.Atoi(dbWarehouse.Code)
	locStr, _ := dbWarehouse.Location.(string)
	typStr, _ := dbWarehouse.Type.(string)
	return &entity.Warehouse{
		Code:                code,
		Description:         dbWarehouse.Description,
		Location:            mapper.WarehouseLocationToDomain(locStr),
		Type:                mapper.WarehouseTypeToDomain(typStr),
		Disposition:         dbWarehouse.Disposition,
		ReservationsAllowed: dbWarehouse.ReservationsAllowed,
		CreatedBy:           pgutil.FromPgUUID(dbWarehouse.CreatedBy),
		CreatedAt:           pgutil.FromPgTimestamp(dbWarehouse.CreatedAt),
	}, nil
}
