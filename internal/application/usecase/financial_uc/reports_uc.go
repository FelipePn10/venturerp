package financial_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

// ---------- R01 ----------

type GetLivroEntradasUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetLivroEntradasUseCase) Execute(ctx context.Context, start, end time.Time) ([]map[string]interface{}, error) {
	if !uc.Auth.CanExportRelatorios(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetLivroEntradas(ctx, start, end)
}

// ---------- R02 ----------

type GetLivroSaidasUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetLivroSaidasUseCase) Execute(ctx context.Context, start, end time.Time) ([]map[string]interface{}, error) {
	if !uc.Auth.CanExportRelatorios(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetLivroSaidas(ctx, start, end)
}

// ---------- R03 ----------

type GetImpostosSaidasUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetImpostosSaidasUseCase) Execute(ctx context.Context, start, end time.Time) ([]map[string]interface{}, error) {
	if !uc.Auth.CanExportRelatorios(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetImpostosSaidas(ctx, start, end)
}

// ---------- R04 ----------

type GetImpostosEntradasUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetImpostosEntradasUseCase) Execute(ctx context.Context, start, end time.Time) ([]map[string]interface{}, error) {
	if !uc.Auth.CanExportRelatorios(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetImpostosEntradas(ctx, start, end)
}

// ---------- R06 DRE com CMV ----------

type GetDREUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetDREUseCase) Execute(ctx context.Context, start, end time.Time) (map[string]interface{}, error) {
	if !uc.Auth.CanExportRelatorios(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetDREComCMV(ctx, start, end)
}

// ---------- R09 Aging Receber Detalhado ----------

type GetAgingReceberDetalhadoUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetAgingReceberDetalhadoUseCase) Execute(ctx context.Context) ([]map[string]interface{}, error) {
	if !uc.Auth.CanGetAgingReceber(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetAgingReceberDetalhado(ctx)
}

// ---------- R10 Aging Pagar Detalhado ----------

type GetAgingPagarDetalhadoUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetAgingPagarDetalhadoUseCase) Execute(ctx context.Context) ([]map[string]interface{}, error) {
	if !uc.Auth.CanGetAgingPagar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetAgingPagarDetalhado(ctx)
}

// ---------- R11 Extrato por Fornecedor ----------

type GetExtratoPorFornecedorUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetExtratoPorFornecedorUseCase) Execute(ctx context.Context, fornecedorID int64) ([]map[string]interface{}, error) {
	if !uc.Auth.CanExportRelatorios(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetExtratoPorFornecedor(ctx, fornecedorID)
}

// ---------- R12 Extrato por Cliente ----------

type GetExtratoPorClienteUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetExtratoPorClienteUseCase) Execute(ctx context.Context, clienteID int64) ([]map[string]interface{}, error) {
	if !uc.Auth.CanExportRelatorios(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetExtratoPorCliente(ctx, clienteID)
}

// ---------- R13 Produtos Vendidos ----------

type GetProdutosVendidosUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetProdutosVendidosUseCase) Execute(ctx context.Context, start, end time.Time) ([]map[string]interface{}, error) {
	if !uc.Auth.CanExportRelatorios(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetProdutosVendidos(ctx, start, end)
}

// ---------- R14 Produtos Produzidos ----------

type GetProdutosProduzidosUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetProdutosProduzidosUseCase) Execute(ctx context.Context, start, end time.Time) ([]map[string]interface{}, error) {
	if !uc.Auth.CanExportRelatorios(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetProdutosProduzidos(ctx, start, end)
}

// ---------- R15 Histórico de Custos ----------

type GetHistoricoCustosUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetHistoricoCustosUseCase) Execute(ctx context.Context, start, end time.Time) ([]map[string]interface{}, error) {
	if !uc.Auth.CanExportRelatorios(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetHistoricoCustos(ctx, start, end)
}

// ---------- R16 Ficha Técnica com Custo ----------

type GetFichaTecnicaCustoUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetFichaTecnicaCustoUseCase) Execute(ctx context.Context, itemCode int64) ([]map[string]interface{}, error) {
	if !uc.Auth.CanExportRelatorios(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetFichaTecnicaCusto(ctx, itemCode)
}

// ---------- R17 Curva ABC Clientes ----------

type GetCurvaABCClientesUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetCurvaABCClientesUseCase) Execute(ctx context.Context, start, end time.Time) ([]map[string]interface{}, error) {
	if !uc.Auth.CanExportRelatorios(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetCurvaABCClientes(ctx, start, end)
}

// ---------- R18 Curva ABC Produtos ----------

type GetCurvaABCProdutosUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetCurvaABCProdutosUseCase) Execute(ctx context.Context, start, end time.Time) ([]map[string]interface{}, error) {
	if !uc.Auth.CanExportRelatorios(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetCurvaABCProdutos(ctx, start, end)
}

// ---------- R19 Compras no Período ----------

type GetComprasPeriodoUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetComprasPeriodoUseCase) Execute(ctx context.Context, start, end time.Time) ([]map[string]interface{}, error) {
	if !uc.Auth.CanExportRelatorios(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetComprasPeriodo(ctx, start, end)
}
