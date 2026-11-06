// Package estoque_uc reúne as operações de estoque que vão além do lançamento:
// transferência entre almoxarifados, estorno, extrato (kardex), posição em
// data, ATP com situação do estoque, reposição, análise e requisição interna.
package estoque_uc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/aprovacao"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

// RequisicoesDeCompra cria a requisição de compra (VSUP0300) a partir da reposição.
type RequisicoesDeCompra interface {
	Create(ctx context.Context, dto request.CreatePurchaseRequisitionDTO) (*response.PurchaseRequisitionResponse, error)
}

// OrdensPlanejadas cria a ordem planejada do item reposto por produção.
type OrdensPlanejadas interface {
	Execute(ctx context.Context, dto request.CreatePlannedOrderDTO) (*response.PlannedOrderResponse, error)
}

// AprovacoesDeRequisicao é o motor de aprovação visto pela requisição.
type AprovacoesDeRequisicao interface {
	TemPoliticaAtiva(ctx context.Context, t aprovacao.Tipo) (bool, error)
	Solicitar(ctx context.Context, t aprovacao.Tipo, documento int64, valor decimal.Decimal, descricao string) (*aprovacao.Solicitacao, error)
}

type UseCase struct {
	Ops        repository.OperacoesEstoque
	Auth       ports.AuthService
	Compras    RequisicoesDeCompra
	Planejadas OrdensPlanejadas
	Aprovacoes AprovacoesDeRequisicao
}

func (uc *UseCase) podeMovimentar(ctx context.Context) error {
	if !uc.Auth.CanCreateStockMovement(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return nil
}

func (uc *UseCase) podeConsultar(ctx context.Context) error {
	if !uc.Auth.CanGetStockBalance(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return nil
}

func quantidade(campo, v string) (decimal.Decimal, error) {
	q, err := decimal.NewFromString(strings.ReplaceAll(strings.TrimSpace(v), ",", "."))
	if err != nil || !q.IsPositive() {
		return decimal.Zero, errorsuc.NewValidationError(fmt.Sprintf("%s inválida: informe um número maior que zero", campo))
	}
	return q, nil
}

// Transferir move o material entre almoxarifados numa operação só.
func (uc *UseCase) Transferir(ctx context.Context, dto request.TransferenciaAlmoxarifadoDTO) (*entity.ResultadoTransferencia, error) {
	if err := uc.podeMovimentar(ctx); err != nil {
		return nil, err
	}
	if dto.ItemCode <= 0 || dto.FromWarehouseID <= 0 || dto.ToWarehouseID <= 0 {
		return nil, errorsuc.NewValidationError("informe o item, o almoxarifado de origem e o de destino")
	}
	q, err := quantidade("quantidade", dto.Quantity)
	if err != nil {
		return nil, err
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	return uc.Ops.TransferirEntreAlmoxarifados(ctx, entity.TransferenciaEstoque{
		ItemCode: dto.ItemCode, Mask: dto.Mask, DeAlmox: dto.FromWarehouseID, ParaAlmox: dto.ToWarehouseID,
		Quantidade: q, Lote: vazioNil(dto.Lot), EnderecoDe: vazioNil(dto.FromAddress), EnderecoPara: vazioNil(dto.ToAddress),
		Notas: vazioNil(dto.Notes), Ator: ator,
	})
}

// Estornar lança o movimento contrário ao informado.
func (uc *UseCase) Estornar(ctx context.Context, id int64, dto request.EstornoMovimentoDTO) ([]*entity.StockMovement, error) {
	if err := uc.podeMovimentar(ctx); err != nil {
		return nil, err
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	return uc.Ops.EstornarMovimento(ctx, id, ator, dto.Reason)
}

// Kardex: período padrão = mês corrente. Data em branco segue como zero: o
// repositório resolve "hoje" e "início do mês" no fuso da empresa, não no do
// processo (que em produção roda em UTC).
func (uc *UseCase) Kardex(ctx context.Context, itemCode int64, almox *int64, de, ate string) (*entity.Kardex, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	var inicio, fim time.Time
	if strings.TrimSpace(de) != "" {
		t, msg := datetime.Obrigatoria("data inicial", de)
		if msg != "" {
			return nil, errorsuc.NewValidationError(msg)
		}
		inicio = t
	}
	if strings.TrimSpace(ate) != "" {
		t, msg := datetime.Obrigatoria("data final", ate)
		if msg != "" {
			return nil, errorsuc.NewValidationError(msg)
		}
		fim = t
	}
	if !inicio.IsZero() && !fim.IsZero() && fim.Before(inicio) {
		return nil, errorsuc.NewValidationError("a data final é anterior à inicial")
	}
	return uc.Ops.Kardex(ctx, itemCode, almox, inicio, fim)
}

// Posicao: saldo ao fim do dia pedido (padrão hoje, no fuso da empresa).
func (uc *UseCase) Posicao(ctx context.Context, data string, almox *int64) ([]entity.PosicaoItem, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	var d time.Time
	if strings.TrimSpace(data) != "" {
		t, msg := datetime.Obrigatoria("data da posição", data)
		if msg != "" {
			return nil, errorsuc.NewValidationError(msg)
		}
		d = t
	}
	return uc.Ops.PosicaoEmData(ctx, d, almox)
}

// ATP com situação: só o almoxarifado disponível, que não seja de inspeção nem
// rejeição, entra no disponível para promessa.
func (uc *UseCase) ATP(ctx context.Context, itemCode int64, mask string) (*response.ATPResponse, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	saldos, err := uc.Ops.SaldosComSituacao(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	out := &response.ATPResponse{ItemCode: itemCode, Mask: mask, Warehouses: []*response.StockBalanceResponse{}}
	var noAlmox, reservado, disponivel, indisponivel decimal.Decimal
	for _, s := range saldos {
		if mask != "" && s.Mask != mask {
			continue
		}
		noAlmox = noAlmox.Add(s.Quantity)
		if s.Prometivel {
			reservado = reservado.Add(s.ReservedQty)
			disponivel = disponivel.Add(s.AvailableQty)
		} else {
			indisponivel = indisponivel.Add(s.Quantity)
		}
		out.Warehouses = append(out.Warehouses, &response.StockBalanceResponse{
			ID: s.ID, ItemCode: s.ItemCode, Mask: s.Mask, WarehouseID: s.WarehouseID, Quantity: s.Quantity.InexactFloat64(),
			ReservedQty: s.ReservedQty.InexactFloat64(), AvailableQty: s.AvailableQty.InexactFloat64(),
			AvgCost: s.AvgCost.InexactFloat64(), TotalCost: s.TotalCost.InexactFloat64(),
			StockStatus: s.Situacao, AvailableToPromise: s.Prometivel,
		})
	}
	out.TotalOnHand, out.TotalReserved = noAlmox.InexactFloat64(), reservado.InexactFloat64()
	out.TotalAvailable, out.TotalNotAvailable = disponivel.InexactFloat64(), indisponivel.InexactFloat64()
	return out, nil
}

func (uc *UseCase) Reposicao(ctx context.Context) ([]entity.SugestaoReposicao, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	return uc.Ops.SugestoesDeReposicao(ctx)
}

// GerarRequisicaoDeCompra transforma as sugestões escolhidas numa requisição de
// compra (VSUP0300), que segue o fluxo normal de cotação e pedido.
func (uc *UseCase) GerarRequisicaoDeCompra(ctx context.Context, dto request.ReposicaoParaRequisicaoDTO) (*response.PurchaseRequisitionResponse, error) {
	if !uc.Auth.CanCreatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if uc.Compras == nil {
		return nil, errorsuc.NewValidationError("requisição de compra indisponível")
	}
	itens := make([]request.RequisitionItemInput, 0, len(dto.Items))
	for _, it := range dto.Items {
		if it.ItemCode <= 0 || it.Quantity <= 0 {
			continue
		}
		itens = append(itens, request.RequisitionItemInput{ItemCode: it.ItemCode, Mask: strings.TrimSpace(it.Mask), Quantity: decimal.NewFromFloat(it.Quantity)})
	}
	if len(itens) == 0 {
		return nil, errorsuc.NewValidationError("escolha ao menos um item com quantidade para a requisição de compra")
	}
	nota := "Gerada pela reposição de estoque (ponto de pedido / mínimo)"
	if dto.Notes != nil && strings.TrimSpace(*dto.Notes) != "" {
		nota = strings.TrimSpace(*dto.Notes)
	}
	return uc.Compras.Create(ctx, request.CreatePurchaseRequisitionDTO{Notes: &nota, Items: itens})
}

func (uc *UseCase) Analise(ctx context.Context, meses, diasParado int) ([]entity.AnaliseItem, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	return uc.Ops.AnaliseDeEstoque(ctx, min(meses, maxMesesDeAnalise), diasParado)
}

// maxMesesDeAnalise limita a janela: um número digitado errado virava uma data
// fora do que o banco aceita.
const maxMesesDeAnalise = 120

// Acuracidade apura as contagens encerradas nos últimos meses. Sem tolerância
// informada: classe A exata, B 2% e C 5%.
func (uc *UseCase) Acuracidade(ctx context.Context, meses int, almox *int64, tolA, tolB, tolC string) (*entity.ResultadoAcuracidade, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	if meses <= 0 {
		meses = 12
	}
	meses = min(meses, maxMesesDeAnalise)
	param, err := uc.Ops.ParametrosDeEstoque(ctx)
	if err != nil {
		return nil, err
	}
	tol := entity.ToleranciaAcuracidade{A: decimal.Zero, B: decimal.NewFromInt(2), C: decimal.NewFromInt(5)}
	for _, p := range []struct{ de, para *decimal.Decimal }{{param.ToleranceA, &tol.A}, {param.ToleranceB, &tol.B}, {param.ToleranceC, &tol.C}} {
		if p.de != nil {
			*p.para = *p.de
		}
	}
	for _, t := range []struct {
		classe, valor string
		destino       *decimal.Decimal
	}{{"A", tolA, &tol.A}, {"B", tolB, &tol.B}, {"C", tolC, &tol.C}} {
		if strings.TrimSpace(t.valor) == "" {
			continue
		}
		v, err := decimal.NewFromString(strings.ReplaceAll(strings.TrimSpace(t.valor), ",", "."))
		if err != nil || v.IsNegative() {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("tolerância da classe %s: informe um percentual", t.classe))
		}
		*t.destino = v
	}
	contagens, err := uc.Ops.ContagensParaAcuracidade(ctx, meses, almox)
	if err != nil {
		return nil, err
	}
	r := entity.ApurarAcuracidade(contagens, tol)
	return &r, nil
}

// IdadeDoEstoque agrupa o saldo pelo tempo sem movimento real.
func (uc *UseCase) IdadeDoEstoque(ctx context.Context) ([]entity.FaixaDeIdade, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	itens, err := uc.Ops.AnaliseDeEstoque(ctx, 12, 90)
	if err != nil {
		return nil, err
	}
	return entity.FaixasDeIdade(itens), nil
}

func (uc *UseCase) LotesAVencer(ctx context.Context, dias int) ([]entity.LoteAVencer, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	return uc.Ops.LotesAVencer(ctx, dias)
}

// Inconsistencias lista as divergências entre o saldo gravado e a sua origem.
func (uc *UseCase) Inconsistencias(ctx context.Context, almox, item *int64) ([]entity.InconsistenciaEstoque, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	return uc.Ops.ListarInconsistencias(ctx, almox, item)
}

// AcertarInconsistencia refaz o saldo e deixa registrado quem acertou e por quê.
func (uc *UseCase) AcertarInconsistencia(ctx context.Context, dto request.AcertoInconsistenciaDTO) error {
	if err := uc.podeMovimentar(ctx); err != nil {
		return err
	}
	motivo := strings.TrimSpace(dto.Reason)
	if len([]rune(motivo)) < 5 {
		return errorsuc.NewValidationError("informe o motivo do acerto")
	}
	if dto.ItemCode <= 0 || dto.WarehouseID <= 0 {
		return errorsuc.NewValidationError("informe o item e o almoxarifado do acerto")
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}
	return uc.Ops.CorrigirInconsistencia(ctx, dto.Kind, dto.ItemCode, dto.Mask, dto.WarehouseID, motivo, ator)
}

// Fechamentos lista os meses de estoque fechados.
func (uc *UseCase) Fechamentos(ctx context.Context) ([]entity.Fechamento, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	return uc.Ops.Fechamentos(ctx)
}

// FecharMes fecha a competência (AAAA-MM); com simular, só mostra o resultado.
func (uc *UseCase) FecharMes(ctx context.Context, competencia string, simular bool) (*entity.ResultadoFechamento, error) {
	if err := uc.podeMovimentar(ctx); err != nil {
		return nil, err
	}
	t, err := time.Parse("2006-01", strings.TrimSpace(competencia))
	if err != nil {
		return nil, errorsuc.NewValidationError("informe o mês a fechar")
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	return uc.Ops.FecharMes(ctx, t, ator, simular)
}

// ReabrirMes reabre o último mês fechado, com motivo.
func (uc *UseCase) ReabrirMes(ctx context.Context, id int64, motivo string) error {
	if err := uc.podeMovimentar(ctx); err != nil {
		return err
	}
	motivo = strings.TrimSpace(motivo)
	if len([]rune(motivo)) < 5 {
		return errorsuc.NewValidationError("informe o motivo da reabertura")
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}
	return uc.Ops.ReabrirMes(ctx, id, motivo, ator)
}

// ---------- Requisição ao almoxarifado ----------

func (uc *UseCase) CriarRequisicao(ctx context.Context, dto request.RequisicaoMaterialDTO) (*entity.RequisicaoMaterial, error) {
	if err := uc.podeMovimentar(ctx); err != nil {
		return nil, err
	}
	if dto.WarehouseID <= 0 {
		return nil, errorsuc.NewValidationError("informe o almoxarifado que vai atender")
	}
	if len(dto.Items) == 0 {
		return nil, errorsuc.NewValidationError("a requisição precisa de ao menos um item")
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	req := &entity.RequisicaoMaterial{
		Code: dto.Code, WarehouseID: dto.WarehouseID, DestinoID: dto.DestinationWarehouseID, RequesterName: vazioNil(dto.RequesterName),
		RequesterEmployeeCode: dto.RequesterEmployeeCode, CostCenterCode: dto.CostCenterCode,
		Notes: vazioNil(dto.Notes), CreatedBy: ator,
	}
	if dto.NeededDate != nil && strings.TrimSpace(*dto.NeededDate) != "" {
		t, msg := datetime.Obrigatoria("data de necessidade", *dto.NeededDate)
		if msg != "" {
			return nil, errorsuc.NewValidationError(msg)
		}
		req.NeededDate = &t
	}
	for i, it := range dto.Items {
		if it.ItemCode <= 0 {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("linha %d: escolha o item", i+1))
		}
		q, err := quantidade(fmt.Sprintf("linha %d: quantidade", i+1), it.Quantity)
		if err != nil {
			return nil, err
		}
		req.Itens = append(req.Itens, entity.RequisicaoMaterialItem{ItemCode: it.ItemCode, Mask: it.Mask, Quantity: q, Notes: vazioNil(it.Notes)})
	}
	if uc.Aprovacoes != nil {
		if req.AguardaAprovacao, err = uc.Aprovacoes.TemPoliticaAtiva(ctx, aprovacao.RequisicaoMaterial); err != nil {
			return nil, err
		}
	}
	criada, err := uc.Ops.CriarRequisicaoMaterial(ctx, req)
	if err != nil {
		return nil, err
	}
	if criada.Status == entity.RequisicaoPendente {
		descricao := fmt.Sprintf("Requisição ao almoxarifado %d", criada.Code)
		if _, err := uc.Aprovacoes.Solicitar(ctx, aprovacao.RequisicaoMaterial, criada.ID, criada.Valor, descricao); err != nil {
			// Sem a solicitação ninguém aprovaria: a requisição não pode ficar parada.
			return nil, errors.Join(err, uc.Ops.CancelarRequisicaoMaterial(ctx, criada.ID))
		}
	}
	return criada, nil
}

func (uc *UseCase) ObterRequisicao(ctx context.Context, id int64) (*entity.RequisicaoMaterial, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	return uc.Ops.ObterRequisicaoMaterial(ctx, id)
}

func (uc *UseCase) ListarRequisicoes(ctx context.Context, status string) ([]*entity.RequisicaoMaterial, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	var filtro *string
	if s := strings.ToUpper(strings.TrimSpace(status)); s != "" {
		filtro = &s
	}
	return uc.Ops.ListarRequisicoesMaterial(ctx, filtro)
}

func (uc *UseCase) AtenderRequisicao(ctx context.Context, id int64, dto request.AtendimentoRequisicaoDTO) (*entity.RequisicaoMaterial, error) {
	if err := uc.podeMovimentar(ctx); err != nil {
		return nil, err
	}
	if len(dto.Lines) == 0 {
		return nil, errorsuc.NewValidationError("informe o que entregar de cada linha")
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	linhas := make([]entity.AtendimentoLinha, 0, len(dto.Lines))
	for _, l := range dto.Lines {
		if strings.TrimSpace(l.Quantity) == "" || strings.TrimSpace(l.Quantity) == "0" {
			continue
		}
		q, err := quantidade("quantidade entregue", l.Quantity)
		if err != nil {
			return nil, err
		}
		linhas = append(linhas, entity.AtendimentoLinha{ItemID: l.ItemID, Quantity: q, Lot: vazioNil(l.Lot), Address: vazioNil(l.Address), DestinationAddress: vazioNil(l.DestinationAddress), ViaTransito: l.ViaTransit, AlmoxTransito: l.TransitWarehouseID})
	}
	if len(linhas) == 0 {
		return nil, errorsuc.NewValidationError("nenhuma quantidade a entregar")
	}
	return uc.Ops.AtenderRequisicaoMaterial(ctx, id, linhas, ator)
}

func (uc *UseCase) CancelarRequisicao(ctx context.Context, id int64) error {
	if err := uc.podeMovimentar(ctx); err != nil {
		return err
	}
	return uc.Ops.CancelarRequisicaoMaterial(ctx, id)
}

func vazioNil(s *string) *string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	v := strings.TrimSpace(*s)
	return &v
}
