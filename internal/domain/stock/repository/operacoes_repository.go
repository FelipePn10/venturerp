package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

// OperacoesEstoque são as operações de estoque que vão além do lançamento
// simples: transferência, estorno, consultas analíticas e requisição interna.
type OperacoesEstoque interface {
	TransferirEntreAlmoxarifados(ctx context.Context, t entity.TransferenciaEstoque) (*entity.ResultadoTransferencia, error)
	EstornarMovimento(ctx context.Context, id int64, ator uuid.UUID, motivo string) ([]*entity.StockMovement, error)
	// Kardex e PosicaoEmData recebem dias do calendário (vale o ano/mês/dia de
	// cada data) e os contam no fuso da empresa; data zero = padrão (hoje e, no
	// kardex, o início do mês corrente).
	Kardex(ctx context.Context, itemCode int64, almox *int64, de, ate time.Time) (*entity.Kardex, error)
	PosicaoEmData(ctx context.Context, data time.Time, almox *int64) ([]entity.PosicaoItem, error)
	SaldosComSituacao(ctx context.Context, itemCode int64) ([]entity.SaldoComSituacao, error)
	SugestoesDeReposicao(ctx context.Context) ([]entity.SugestaoReposicao, error)
	AnaliseDeEstoque(ctx context.Context, meses, diasParado int) ([]entity.AnaliseItem, error)
	LotesAVencer(ctx context.Context, dias int) ([]entity.LoteAVencer, error)
	TipoDeMovimento(ctx context.Context, sigla string) (*entity.TipoMovimentoConfig, error)

	CriarRequisicaoMaterial(ctx context.Context, req *entity.RequisicaoMaterial) (*entity.RequisicaoMaterial, error)
	ObterRequisicaoMaterial(ctx context.Context, id int64) (*entity.RequisicaoMaterial, error)
	ListarRequisicoesMaterial(ctx context.Context, status *string) ([]*entity.RequisicaoMaterial, error)
	AtenderRequisicaoMaterial(ctx context.Context, id int64, linhas []entity.AtendimentoLinha, ator uuid.UUID) (*entity.RequisicaoMaterial, error)
	CancelarRequisicaoMaterial(ctx context.Context, id int64) error
}
