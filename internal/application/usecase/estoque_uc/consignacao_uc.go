package estoque_uc

import (
	"context"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

// Consignacoes lista as consignações; direção vazia traz as duas.
func (uc *UseCase) Consignacoes(ctx context.Context, direcao string, soAbertas bool) ([]entity.Consignacao, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	return uc.Ops.ListarConsignacoes(ctx, strings.ToUpper(strings.TrimSpace(direcao)), soAbertas)
}

// ReceberConsignacao registra o material do fornecedor, que não entra no estoque.
func (uc *UseCase) ReceberConsignacao(ctx context.Context, dto request.ReceberConsignacaoDTO) (*entity.Consignacao, error) {
	if err := uc.podeMovimentar(ctx); err != nil {
		return nil, err
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if dto.SupplierCode <= 0 || dto.ItemCode <= 0 {
		return nil, errorsuc.NewValidationError("escolha o fornecedor e o item")
	}
	q, err := quantidade("quantidade", dto.Quantity)
	if err != nil {
		return nil, err
	}
	v, err := quantidadeOpcional("valor unitário", &dto.UnitValue)
	if err != nil {
		return nil, err
	}
	if v == nil || !v.IsPositive() {
		return nil, errorsuc.NewValidationError("informe o valor unitário combinado com o fornecedor")
	}
	if dto.Lot != nil && len([]rune(*dto.Lot)) > 50 {
		return nil, errorsuc.NewValidationError("o lote passa de 50 caracteres")
	}
	if documentoLongo(dto.Document) {
		return nil, errorsuc.NewValidationError("o documento passa de 60 caracteres")
	}
	emissao, err := time.Parse("2006-01-02", dto.IssueDate)
	if err != nil {
		return nil, errorsuc.NewValidationError("informe a data da nota")
	}
	c := entity.Consignacao{Parceiro: dto.SupplierCode, ItemCode: dto.ItemCode, Mask: strings.TrimSpace(dto.Mask), Lote: vazioNil(dto.Lot), Quantidade: q, Valor: *v,
		Documento: vazioNil(dto.Document), Emissao: emissao, Notas: vazioNil(dto.Notes), CreatedBy: ator}
	if dto.DueDate != nil && *dto.DueDate != "" {
		prazo, err := time.Parse("2006-01-02", *dto.DueDate)
		if err != nil {
			return nil, errorsuc.NewValidationError("data de vencimento inválida")
		}
		c.Prazo = &prazo
	}
	return uc.Ops.ReceberConsignacao(ctx, c)
}

func documentoLongo(d *string) bool {
	return d != nil && len([]rune(*d)) > 60
}

func (uc *UseCase) baixaDeConsignacao(ctx context.Context, id int64, dto request.BaixaConsignacaoDTO) (entity.BaixaConsignacao, error) {
	var b entity.BaixaConsignacao
	if err := uc.podeMovimentar(ctx); err != nil {
		return b, err
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return b, err
	}
	q, err := quantidade("quantidade", dto.Quantity)
	if err != nil {
		return b, err
	}
	if documentoLongo(dto.Document) {
		return b, errorsuc.NewValidationError("o documento passa de 60 caracteres")
	}
	b = entity.BaixaConsignacao{ID: id, Quantidade: q, Documento: vazioNil(dto.Document), Ator: ator}
	if dto.WarehouseID != nil {
		b.Almox = *dto.WarehouseID
	}
	return b, nil
}

// DevolverConsignacao baixa pela devolução física. A nota de devolução é emitida à
// parte, e o número dela fica no documento.
func (uc *UseCase) DevolverConsignacao(ctx context.Context, id int64, dto request.BaixaConsignacaoDTO) (*entity.Consignacao, error) {
	b, err := uc.baixaDeConsignacao(ctx, id, dto)
	if err != nil {
		return nil, err
	}
	return uc.Ops.DevolverConsignacao(ctx, b)
}

// ConsumirConsignacao passa ao estoque próprio o material do fornecedor.
func (uc *UseCase) ConsumirConsignacao(ctx context.Context, id int64, dto request.BaixaConsignacaoDTO) (*entity.Consignacao, error) {
	b, err := uc.baixaDeConsignacao(ctx, id, dto)
	if err != nil {
		return nil, err
	}
	if b.Almox <= 0 {
		return nil, errorsuc.NewValidationError("escolha o almoxarifado que recebe o material")
	}
	return uc.Ops.ConsumirConsignacao(ctx, b)
}

// TransferenciasEntreCNPJs lista as transferências enviadas e a receber.
func (uc *UseCase) TransferenciasEntreCNPJs(ctx context.Context) ([]entity.TransferenciaEntreCNPJs, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	return uc.Ops.TransferenciasEntreCNPJs(ctx)
}
