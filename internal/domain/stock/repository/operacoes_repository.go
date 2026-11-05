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
	ToleranciasPorPerfil(ctx context.Context) ([]entity.ToleranciaDoPerfil, error)
	SalvarToleranciaDoPerfil(ctx context.Context, t entity.ToleranciaDoPerfil, ator uuid.UUID) error
	ExcluirToleranciaDoPerfil(ctx context.Context, perfil int64) error
	// Cadernos de contagem do inventário.
	Cadernos(ctx context.Context, inventario int64) ([]entity.CadernoInventario, error)
	GerarCadernos(ctx context.Context, inventario int64, linhasPorCaderno int) ([]entity.CadernoInventario, error)
	MarcarCadernoImpresso(ctx context.Context, inventario int64, numero int, ator uuid.UUID) error
	TransferirEntreAlmoxarifados(ctx context.Context, t entity.TransferenciaEstoque) (*entity.ResultadoTransferencia, error)
	EstornarMovimento(ctx context.Context, id int64, ator uuid.UUID, motivo string) ([]*entity.StockMovement, error)
	// Kardex e PosicaoEmData recebem dias do calendário (vale o ano/mês/dia de
	// cada data) e os contam no fuso da empresa; data zero = padrão (hoje e, no
	// kardex, o início do mês corrente).
	Kardex(ctx context.Context, itemCode int64, almox *int64, de, ate time.Time) (*entity.Kardex, error)
	PosicaoEmData(ctx context.Context, data time.Time, almox *int64) ([]entity.PosicaoItem, error)
	SaldosComSituacao(ctx context.Context, itemCode int64) ([]entity.SaldoComSituacao, error)
	SugestoesDeReposicao(ctx context.Context) ([]entity.SugestaoReposicao, error)
	ContagensParaAcuracidade(ctx context.Context, meses int, almox *int64) ([]entity.ContagemAcuracidade, error)
	ResolverLeitura(ctx context.Context, codigo string, almox *int64) (*entity.LeituraResolvida, error)
	EnviarParaTransito(ctx context.Context, e entity.EnvioTransito) (*entity.Transito, error)
	BaixarTransito(ctx context.Context, b entity.BaixaTransito) (*entity.Transito, error)
	ListarTransitos(ctx context.Context, soAbertos bool) ([]entity.Transito, error)
	// HojeDaEmpresa é a data de hoje no fuso da empresa.
	HojeDaEmpresa(ctx context.Context) (time.Time, error)
	ListarPoliticasDeReposicao(ctx context.Context) ([]entity.PoliticaReposicao, error)
	SalvarPoliticaDeReposicao(ctx context.Context, p entity.PoliticaReposicao) (*entity.PoliticaReposicao, error)
	ExcluirPoliticaDeReposicao(ctx context.Context, id int64) error
	AnaliseDeEstoque(ctx context.Context, meses, diasParado int) ([]entity.AnaliseItem, error)
	LotesAVencer(ctx context.Context, dias int) ([]entity.LoteAVencer, error)
	TipoDeMovimento(ctx context.Context, sigla string) (*entity.TipoMovimentoConfig, error)

	ListarSituacoesLote(ctx context.Context) ([]entity.SituacaoLote, error)
	SalvarSituacaoLote(ctx context.Context, s entity.SituacaoLote) error
	ParametrosDeEstoque(ctx context.Context) (entity.ParametrosEstoque, error)
	SeparadoParaFaturar(ctx context.Context, pedido, item int64, mask string, almox int64) ([]entity.SeparadoDaOnda, error)
	LinhasDeCargaParaOnda(ctx context.Context, carga, almox int64) ([]entity.LinhaPedidoOnda, error)
	ListarCartoesKanban(ctx context.Context) ([]entity.CartaoKanban, error)
	ObterCartaoKanban(ctx context.Context, id int64) (*entity.CartaoKanban, error)
	SalvarCartaoKanban(ctx context.Context, c entity.CartaoKanban, ator uuid.UUID) (*entity.CartaoKanban, error)
	IdadeDosLotes(ctx context.Context, almox *int64) ([]entity.IdadeDoLote, error)
	NiveisPeloConsumo(ctx context.Context, item int64, almox *int64, meses, prazo, diasSeguranca int) (*entity.NiveisSugeridos, error)
	ListarMotivosDivergencia(ctx context.Context) ([]entity.MotivoDivergencia, error)
	SalvarMotivoDivergencia(ctx context.Context, m entity.MotivoDivergencia) error
	SalvarParametrosDeEstoque(ctx context.Context, p entity.ParametrosEstoque, ator uuid.UUID) error
	ControleDoItem(ctx context.Context, itemCode int64) (entity.ControleDoItem, error)
	SalvarControleDoItem(ctx context.Context, c entity.ControleDoItem) error
	MudarSituacaoDoLote(ctx context.Context, m entity.MudancaSituacaoLote, ator uuid.UUID) error
	ImplantarLote(ctx context.Context, i entity.ImplantacaoLote, ator uuid.UUID) error

	Fechamentos(ctx context.Context) ([]entity.Fechamento, error)
	// FecharMes fecha a competência; com simular, faz tudo e desfaz, devolvendo
	// o mesmo resultado.
	FecharMes(ctx context.Context, competencia time.Time, ator uuid.UUID, simular bool) (*entity.ResultadoFechamento, error)
	ReabrirMes(ctx context.Context, id int64, motivo string, ator uuid.UUID) error

	ListarInconsistencias(ctx context.Context, almox, item *int64) ([]entity.InconsistenciaEstoque, error)
	CorrigirInconsistencia(ctx context.Context, tipo string, item int64, mask string, almox int64, motivo string, ator uuid.UUID) error

	CriarRequisicaoMaterial(ctx context.Context, req *entity.RequisicaoMaterial) (*entity.RequisicaoMaterial, error)
	// LiberarRequisicaoAprovada abre a requisição pendente e reserva o material.
	LiberarRequisicaoAprovada(ctx context.Context, id int64) error
	ReprovarRequisicaoMaterial(ctx context.Context, id int64) error
	ObterRequisicaoMaterial(ctx context.Context, id int64) (*entity.RequisicaoMaterial, error)
	ListarRequisicoesMaterial(ctx context.Context, status *string) ([]*entity.RequisicaoMaterial, error)
	AtenderRequisicaoMaterial(ctx context.Context, id int64, linhas []entity.AtendimentoLinha, ator uuid.UUID) (*entity.RequisicaoMaterial, error)
	CancelarRequisicaoMaterial(ctx context.Context, id int64) error
}
