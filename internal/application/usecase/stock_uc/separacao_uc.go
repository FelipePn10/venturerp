package stock_uc

import (
	"context"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

// SeparacaoUseCase responde "de onde tirar" — qual lote, de qual endereço.
type SeparacaoUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

// SugerirFEFO monta a sugestão de separação. A regra padrão é FEFO; FIFO fica
// disponível para quem controla por recebimento e não por validade.
func (uc *SeparacaoUseCase) SugerirFEFO(
	ctx context.Context, itemCode int64, mask string, warehouseID int64, necessario float64, regra string,
) (*entity.ResultadoFEFO, error) {
	if necessario <= 0 {
		return nil, errorsuc.NewValidationError("informe a quantidade a separar")
	}
	return uc.Repo.SugerirSeparacaoFEFO(ctx, itemCode, mask, warehouseID, necessario, regra)
}

// SaldoPorEndereco lista o estoque quebrado por endereço.
func (uc *SeparacaoUseCase) SaldoPorEndereco(ctx context.Context, warehouseID, itemCode int64) ([]*entity.StockLotBalance, error) {
	return uc.Repo.ListarSaldoPorEndereco(ctx, warehouseID, itemCode)
}

// TransferirEndereco move material de um endereço para outro no mesmo
// almoxarifado. Sem isto o endereçamento seria só leitura: dava para saber onde
// o material está, não para registrar que ele mudou de lugar.
func (uc *SeparacaoUseCase) TransferirEndereco(ctx context.Context, dto request.TransferenciaEnderecoDTO) (*entity.StockMovement, error) {
	if !uc.Auth.CanCreateStockMovement(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	origem := strings.TrimSpace(dto.AddressFrom)
	destino := strings.TrimSpace(dto.AddressTo)
	if origem == "" || destino == "" {
		return nil, errorsuc.NewValidationError("informe o endereço de origem e o de destino")
	}
	if origem == destino {
		return nil, errorsuc.NewValidationError("origem e destino são o mesmo endereço")
	}
	if dto.Quantity <= 0 {
		return nil, errorsuc.NewValidationError("a quantidade da transferência precisa ser maior que zero")
	}
	usuario, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	movimento := &entity.StockMovement{
		ItemCode: dto.ItemCode, Mask: dto.Mask, WarehouseID: dto.WarehouseID,
		MovementType: entity.MovementTypeAddressTransfer, Quantity: dto.Quantity,
		Lot: dto.Lot, Notes: dto.Notes, CreatedBy: usuario,
	}
	return uc.Repo.TransferirEntreEnderecos(ctx, movimento, origem, destino)
}

// ApurarABC recalcula a curva ABC a partir do valor consumido.
func (uc *SeparacaoUseCase) ApurarABC(ctx context.Context, janelaMeses int, corteA, corteB float64) (*entity.ResumoABC, error) {
	if !uc.Auth.CanCreateStockMovement(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if janelaMeses < 0 || janelaMeses > 60 {
		return nil, errorsuc.NewValidationError("a janela da curva ABC deve ficar entre 1 e 60 meses")
	}
	return uc.Repo.ApurarCurvaABC(ctx, janelaMeses, corteA, corteB)
}

// SugerirGuarda recomenda onde guardar o material recebido.
func (uc *SeparacaoUseCase) SugerirGuarda(
	ctx context.Context, itemCode int64, mask string, warehouseID int64, quantidade float64, zona string,
) ([]*entity.SugestaoGuarda, error) {
	if quantidade <= 0 {
		return nil, errorsuc.NewValidationError("informe a quantidade a guardar")
	}
	if warehouseID <= 0 {
		return nil, errorsuc.NewValidationError("informe o almoxarifado")
	}
	return uc.Repo.SugerirEnderecoDeGuarda(ctx, itemCode, mask, warehouseID, quantidade, zona)
}

// CriarOnda agrupa necessidades numa caminhada só, reservando o que alocou.
func (uc *SeparacaoUseCase) CriarOnda(ctx context.Context, dto request.OndaDeSeparacaoDTO) (*entity.OndaDeSeparacao, error) {
	if !uc.Auth.CanCreateStockMovement(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.Code <= 0 {
		return nil, errorsuc.NewValidationError("informe o número da onda")
	}
	if dto.WarehouseID <= 0 {
		return nil, errorsuc.NewValidationError("informe o almoxarifado da onda")
	}
	if len(dto.Lines) == 0 {
		return nil, errorsuc.NewValidationError("a onda precisa de pelo menos uma necessidade")
	}
	necessidades := make([]entity.NecessidadeOnda, 0, len(dto.Lines))
	for _, l := range dto.Lines {
		if l.Quantity <= 0 {
			return nil, errorsuc.NewValidationError("toda necessidade da onda precisa de quantidade maior que zero")
		}
		necessidades = append(necessidades, entity.NecessidadeOnda{
			ItemCode: l.ItemCode, Mask: l.Mask, Quantity: l.Quantity,
			ReferenceType: l.ReferenceType, ReferenceCode: l.ReferenceCode,
		})
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	return uc.Repo.CriarOndaDeSeparacao(ctx, dto.Code, dto.WarehouseID, dto.Rule, necessidades, ator)
}

// ConfirmarOnda baixa o estoque separado e libera as reservas.
func (uc *SeparacaoUseCase) ConfirmarOnda(ctx context.Context, codigo int64) (*entity.OndaDeSeparacao, error) {
	if !uc.Auth.CanCreateStockMovement(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	return uc.Repo.ConfirmarOndaDeSeparacao(ctx, codigo, ator)
}

// CancelarOnda devolve o reservado sem mexer no estoque físico.
func (uc *SeparacaoUseCase) CancelarOnda(ctx context.Context, codigo int64) (*entity.OndaDeSeparacao, error) {
	if !uc.Auth.CanCreateStockMovement(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	return uc.Repo.CancelarOndaDeSeparacao(ctx, codigo, ator)
}
