package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	fiscalEntity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type BaixaParams struct {
	ContaBancariaID int64     `json:"conta_bancaria_id"`
	ValorPago       float64   `json:"valor_pago"`
	Juros           float64   `json:"juros"`
	Multa           float64   `json:"multa"`
	Desconto        float64   `json:"desconto"`
	DataPagamento   time.Time `json:"data_pagamento"`
	Observacao      *string   `json:"observacao,omitempty"`
	BaixadoPor      uuid.UUID `json:"baixado_por"`
}

type CPFilter struct {
	Status       *string    `json:"status,omitempty"`
	FornecedorID *int64     `json:"fornecedor_id,omitempty"`
	StartDate    *time.Time `json:"start_date,omitempty"`
	EndDate      *time.Time `json:"end_date,omitempty"`
}

type CRFilter struct {
	Status    *string    `json:"status,omitempty"`
	ClienteID *int64     `json:"cliente_id,omitempty"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
}

type AgingResult struct {
	Period string  `json:"period"`
	Total  float64 `json:"total"`
}

type ProjectedFlow struct {
	Data          time.Time `json:"data"`
	ValorEntradas float64   `json:"valor_entradas"`
	ValorSaidas   float64   `json:"valor_saidas"`
	Saldo         float64   `json:"saldo"`
}

type FinancialRepository interface {
	// Contas Bancarias
	CreateContaBancaria(ctx context.Context, c *entity.ContaBancaria) (*entity.ContaBancaria, error)
	ListContasBancarias(ctx context.Context) ([]*entity.ContaBancaria, error)
	GetContaBancaria(ctx context.Context, id int64) (*entity.ContaBancaria, error)
	UpdateSaldo(ctx context.Context, id int64, novoSaldo float64) error

	// Condicoes Pagamento
	CreateCondicaoPagamento(ctx context.Context, c *entity.CondicaoPagamento) (*entity.CondicaoPagamento, error)
	ListCondicoesPagamento(ctx context.Context) ([]*entity.CondicaoPagamento, error)

	// Plano de Contas
	CreatePlanoContas(ctx context.Context, p *entity.PlanoContas) (*entity.PlanoContas, error)
	ListPlanoContas(ctx context.Context) ([]*entity.PlanoContas, error)

	// Centros de Custo
	CreateCentroCusto(ctx context.Context, c *entity.CentroCusto) (*entity.CentroCusto, error)
	ListCentrosCusto(ctx context.Context) ([]*entity.CentroCusto, error)

	// Contas a Pagar
	CreateContaPagar(ctx context.Context, c *entity.ContaPagar) (*entity.ContaPagar, error)
	GetContaPagar(ctx context.Context, id int64) (*entity.ContaPagar, error)
	ListContasPagar(ctx context.Context, filters CPFilter) ([]*entity.ContaPagar, error)
	UpdateContaPagar(ctx context.Context, c *entity.ContaPagar) (*entity.ContaPagar, error)
	ApproveContaPagar(ctx context.Context, id int64, approvedBy uuid.UUID) error
	BaixarContaPagar(ctx context.Context, id int64, params BaixaParams) error
	BaixarContaPagarAtomico(ctx context.Context, id int64, params BaixaParams, fc entity.FluxoCaixa, valorOriginal decimal.Decimal, contaBancariaID int64) error
	CancelContaPagar(ctx context.Context, id int64) error
	GetAgingContasPagar(ctx context.Context) ([]*AgingResult, error)

	// Contas a Receber
	CreateContaReceber(ctx context.Context, c *entity.ContaReceber) (*entity.ContaReceber, error)
	GetContaReceber(ctx context.Context, id int64) (*entity.ContaReceber, error)
	ListContasReceber(ctx context.Context, filters CRFilter) ([]*entity.ContaReceber, error)
	BaixarContaReceber(ctx context.Context, id int64, params BaixaParams) error
	BaixarContaReceberAtomico(ctx context.Context, id int64, params BaixaParams, fc entity.FluxoCaixa, valorOriginal decimal.Decimal, contaBancariaID int64) error
	CancelContaReceber(ctx context.Context, id int64) error
	GetAgingContasReceber(ctx context.Context) ([]*AgingResult, error)

	// Cash Flow
	CreateFluxoCaixa(ctx context.Context, f *entity.FluxoCaixa) (*entity.FluxoCaixa, error)
	GetFluxoCaixa(ctx context.Context, startDate, endDate time.Time) ([]*entity.FluxoCaixa, error)
	GetFluxoProjetado(ctx context.Context, startDate time.Time) ([]*ProjectedFlow, error)
	GetSaldoConta(ctx context.Context, contaID int64) (float64, error)
	GetSaldoConsolidado(ctx context.Context) (float64, error)
	MarcarConciliado(ctx context.Context, id int64) error

	// Tax Assessment
	CreateTaxAssessment(ctx context.Context, t *entity.TaxAssessment) (*entity.TaxAssessment, error)
	UpsertTaxAssessmentCredito(ctx context.Context, t *entity.TaxAssessment) error
	GetTaxAssessment(ctx context.Context, imposto, competencia string) (*entity.TaxAssessment, error)
	ListTaxAssessments(ctx context.Context, competencia string) ([]*entity.TaxAssessment, error)

	// Fiscal Data for Tax Assessment
	GetFiscalDebits(ctx context.Context, competencia string) (map[string]float64, error)
	GetFiscalCredits(ctx context.Context, competencia string) (map[string]float64, error)
	GetFiscalConfig(ctx context.Context) (*fiscalEntity.FiscalConfig, error)

	// Contas Receber helpers
	CancelContasReceberByFiscalExit(ctx context.Context, fiscalExitID int64) error

	// Reports R01-R12
	GetLivroEntradas(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error)
	GetLivroSaidas(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error)
	GetImpostosSaidas(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error)
	GetImpostosEntradas(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error)
	GetDRE(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error)
	GetDREComCMV(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error)
	GetAgingReceberDetalhado(ctx context.Context) ([]map[string]interface{}, error)
	GetAgingPagarDetalhado(ctx context.Context) ([]map[string]interface{}, error)
	GetExtratoPorFornecedor(ctx context.Context, fornecedorID int64) ([]map[string]interface{}, error)
	GetExtratoPorCliente(ctx context.Context, clienteID int64) ([]map[string]interface{}, error)

	// Reports R13-R19
	GetProdutosVendidos(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error)
	GetProdutosProduzidos(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error)
	GetHistoricoCustos(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error)
	GetFichaTecnicaCusto(ctx context.Context, itemCode int64) ([]map[string]interface{}, error)
	GetCurvaABCClientes(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error)
	GetCurvaABCProdutos(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error)
	GetComprasPeriodo(ctx context.Context, startDate, endDate time.Time) ([]map[string]interface{}, error)

	// Adiantamentos (advance payments)
	CreateAdiantamentoAtomico(ctx context.Context, a *entity.Adiantamento, fc entity.FluxoCaixa) (*entity.Adiantamento, error)
	GetAdiantamento(ctx context.Context, id int64) (*entity.Adiantamento, error)
	ListAdiantamentos(ctx context.Context, tipo *string, parceiroID *int64) ([]*entity.Adiantamento, error)
	AplicarAdiantamentoAtomico(ctx context.Context, advID int64, contaTipo string, contaID int64, valor decimal.Decimal, userID uuid.UUID, dataAplicacao time.Time) (*entity.AdiantamentoAplicacao, error)
	ListAplicacoesByAdiantamento(ctx context.Context, advID int64) ([]*entity.AdiantamentoAplicacao, error)

	// Conciliação Bancária
	SaveExtratoItem(ctx context.Context, contaID int64, data time.Time, valor float64, tipo, descricao, fitid, hash string) error
	GetExtratoPendente(ctx context.Context, contaID int64) ([]map[string]interface{}, error)
	ConciliarExtrato(ctx context.Context, extratoID, fluxoID int64) error
	AutoMatchExtrato(ctx context.Context, contaID int64) (int, error)
}
