package product

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/domain/product/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r *repositoryProductSQLC) Save(
	ctx context.Context,
	product *entity.Product,
) (*entity.Product, error) {

	params := sqlc.CreateProductParams{
		ID:   product.ID,
		Code: product.Code,
		GroupCode: pgtype.Text{
			String: product.GroupCode,
			Valid:  product.GroupCode != "",
		},
		Name:      product.Name,
		CreatedBy: pgutil.ToPgUUID(product.CreatedBy),
	}

	dbProduct, err := r.q.CreateProduct(ctx, params)
	if err != nil {
		return nil, err
	}

	return &entity.Product{
		ID:        dbProduct.ID,
		Code:      dbProduct.Code,
		GroupCode: dbProduct.GroupCode.String,
		Name:      dbProduct.Name,
		CreatedBy: pgutil.FromPgUUID(dbProduct.CreatedBy),
		CreatedAt: dbProduct.CreatedAt.Time,
	}, nil

}

func (r *repositoryProductSQLC) Delete(
	ctx context.Context,
	id int64,
) error {
	return r.q.DeleteProduct(ctx, id)
}

func (r *repositoryProductSQLC) ExistsProductByCode(
	ctx context.Context,
	code string,
) (bool, error) {
	_, err := r.q.ExistsProductByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// func (r *repositoryProductSQLC) FindByID(
// 	ctx context.Context,
// 	id uuid.UUID,
// ) (*entity.Product, error) {

// 	dbProduct, err := r.q.GetProductByID(ctx, id)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &entity.Product{
// 		ID:        dbProduct.ID,
// 		Code:      dbProduct.Code,
// 		GroupCode: dbProduct.GroupCode,
// 		Name:      dbProduct.Name,
// 		CreatedBy: dbProduct.CreatedBy,
// 		CreatedAt: dbProduct.CreatedAt,
// 		UpdatedAt: dbProduct.UpdatedAt,
// 	}, nil
// }
