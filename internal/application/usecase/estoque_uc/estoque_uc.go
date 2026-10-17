// Package estoque_uc reúne as operações de estoque que vão além do lançamento:
// transferência entre almoxarifados, estorno, extrato (kardex), posição em
// data, ATP com situação do estoque, reposição, análise e requisição interna.
package estoque_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

// RequisicoesDeCompra cria a requisição de compra (VSUP0300) a partir da reposição.
type RequisicoesDeCompra interface {
	Create(ctx context.Context, dto request.CreatePurchaseRequisitionDTO) (*response.PurchaseRequisitionResponse, error)
}

type UseCase struct {
	Ops     repository.OperacoesEstoque
	Auth    ports.AuthService
	Compras RequisicoesDeCompra
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

// Kardex: período padrão = mês corrente.
func (uc *UseCase) Kardex(ctx context.Context, itemCode int64, almox *int64, de, ate string) (*entity.Kardex, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	agora := time.Now()
	inicio := time.Date(agora.Year(), agora.Month(), 1, 0, 0, 0, 0, time.Local)
	fim := agora
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
	if fim.Before(inicio) {
		return nil, errorsuc.NewValidationError("a data final é anterior à inicial")
	}
	return uc.Ops.Kardex(ctx, itemCode, almox, truncDia(inicio), truncDia(fim))
}

// Posicao: saldo ao fim do dia pedido (padrão hoje).
func (uc *UseCase) Posicao(ctx context.Context, data string, almox *int64) ([]entity.PosicaoItem, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	d := time.Now()
	if strings.TrimSpace(data) != "" {
		t, msg := datetime.Obrigatoria("data da posição", data)
		if msg != "" {
			return nil, errorsuc.NewValidationError(msg)
		}
		d = t
	}
	return uc.Ops.PosicaoEmData(ctx, truncDia(d), almox)
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
	for _, s := range saldos {
		if mask != "" && s.Mask != mask {
			continue
		}
		out.TotalOnHand += s.Quantity
		if s.Prometivel {
			out.TotalReserved += s.ReservedQty
			out.TotalAvailable += s.AvailableQty
		} else {
			out.TotalNotAvailable += s.Quantity
		}
		out.Warehouses = append(out.Warehouses, &response.StockBalanceResponse{
			ID: s.ID, ItemCode: s.ItemCode, Mask: s.Mask, WarehouseID: s.WarehouseID, Quantity: s.Quantity,
			ReservedQty: s.ReservedQty, AvailableQty: s.AvailableQty, AvgCost: s.AvgCost, TotalCost: s.TotalCost,
			StockStatus: s.Situacao, AvailableToPromise: s.Prometivel,
		})
	}
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
		itens = append(itens, request.RequisitionItemInput{ItemCode: it.ItemCode, Quantity: it.Quantity})
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
	return uc.Ops.AnaliseDeEstoque(ctx, meses, diasParado)
}

func (uc *UseCase) LotesAVencer(ctx context.Context, dias int) ([]entity.LoteAVencer, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	return uc.Ops.LotesAVencer(ctx, dias)
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
		Code: dto.Code, WarehouseID: dto.WarehouseID, RequesterName: vazioNil(dto.RequesterName),
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
	return uc.Ops.CriarRequisicaoMaterial(ctx, req)
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
		linhas = append(linhas, entity.AtendimentoLinha{ItemID: l.ItemID, Quantity: q, Lot: vazioNil(l.Lot), Address: vazioNil(l.Address)})
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

func truncDia(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
