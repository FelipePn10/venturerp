package salespricing

import (
	"context"
	"fmt"
	"strconv"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	customerentity "github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
)

type Repository interface {
	GetSalesTableByCode(context.Context, int64) (*customerentity.SalesTable, error)
	GetSalesTablePrice(context.Context, int64, string) (*customerentity.SalesTablePrice, error)
}

type Result struct {
	BasePrice, AppliedPrice float64
	Source                  string
}

func Resolve(ctx context.Context, repo Repository, tableCode, itemCode int64, quantity float64) (*Result, error) {
	if repo == nil || tableCode <= 0 {
		return nil, errorsuc.NewValidationError("informe uma tabela de preço válida")
	}
	if quantity <= 0 {
		return nil, errorsuc.NewValidationError("a quantidade deve ser maior que zero")
	}
	table, err := repo.GetSalesTableByCode(ctx, tableCode)
	if err != nil {
		return nil, errorsuc.NewValidationError("tabela de preço não encontrada")
	}
	now := time.Now()
	if !table.IsActive {
		return nil, errorsuc.NewValidationError("a tabela de preço está inativa")
	}
	if table.ValidityStart != nil && now.Before(*table.ValidityStart) {
		return nil, errorsuc.NewValidationError("a vigência da tabela de preço ainda não iniciou")
	}
	if table.ValidityEnd != nil && now.After(table.ValidityEnd.Add(24*time.Hour)) {
		return nil, errorsuc.NewValidationError("a tabela de preço está vencida")
	}
	price, err := repo.GetSalesTablePrice(ctx, table.ID, strconv.FormatInt(itemCode, 10))
	if err != nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("não existe preço válido para o item %d na tabela %d", itemCode, tableCode))
	}
	if price.Blocked {
		return nil, errorsuc.NewValidationError("o preço do item está bloqueado na tabela")
	}
	if price.Situation == customerentity.PriceSituationInativo {
		return nil, errorsuc.NewValidationError("o preço do item está inativo na tabela")
	}
	if price.Price <= 0 {
		return nil, errorsuc.NewValidationError("o item não possui preço positivo válido")
	}
	return &Result{BasePrice: price.Price, AppliedPrice: price.Price, Source: "SALES_TABLE"}, nil
}
