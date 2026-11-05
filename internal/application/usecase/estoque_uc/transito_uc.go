package estoque_uc

import (
	"context"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
)

// EnviarParaTransito é a primeira etapa da transferência: sai da origem e fica
// no almoxarifado de trânsito até ser recebido.
func (uc *UseCase) EnviarParaTransito(ctx context.Context, dto request.EnvioTransitoDTO) (*entity.Transito, error) {
	if err := uc.podeMovimentar(ctx); err != nil {
		return nil, err
	}
	if dto.ItemCode <= 0 || dto.FromWarehouseID <= 0 || dto.ToWarehouseID <= 0 {
		return nil, errorsuc.NewValidationError("informe item, origem e destino")
	}
	q, err := quantidade("quantidade", dto.Quantity)
	if err != nil {
		return nil, err
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	e := entity.EnvioTransito{
		ItemCode: dto.ItemCode, Mask: dto.Mask, Lote: vazioNil(dto.Lot), EnderecoDe: vazioNil(dto.FromAddress),
		DeAlmox: dto.FromWarehouseID, AlmoxTransito: dto.TransitWarehouseID, ParaAlmox: dto.ToWarehouseID,
		Quantidade: q, Notas: vazioNil(dto.Notes), Ator: ator,
	}
	if v := vazioNil(dto.ExpectedDate); v != nil {
		t, msg := datetime.Obrigatoria("previsão de chegada", *v)
		if msg != "" {
			return nil, errorsuc.NewValidationError(msg)
		}
		e.Previsao = &t
	}
	return uc.Ops.EnviarParaTransito(ctx, e)
}

// BaixarTransito dá destino a parte do envio. Devolução e perda precisam de motivo.
func (uc *UseCase) BaixarTransito(ctx context.Context, id int64, dto request.BaixaTransitoDTO) (*entity.Transito, error) {
	if err := uc.podeMovimentar(ctx); err != nil {
		return nil, err
	}
	q, err := quantidade("quantidade", dto.Quantity)
	if err != nil {
		return nil, err
	}
	acao := strings.ToUpper(strings.TrimSpace(dto.Action))
	motivo := strings.TrimSpace(dto.Reason)
	if acao != entity.TransitoReceber && len([]rune(motivo)) < 5 {
		return nil, errorsuc.NewValidationError("informe o motivo da devolução ou da perda")
	}
	ator, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	return uc.Ops.BaixarTransito(ctx, entity.BaixaTransito{ID: id, Acao: acao, Quantidade: q, Endereco: vazioNil(dto.Address), Motivo: motivo, Ator: ator})
}

// Transitos lista os envios; com soAbertos, só o que ainda está na estrada.
func (uc *UseCase) Transitos(ctx context.Context, soAbertos bool) ([]entity.Transito, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	return uc.Ops.ListarTransitos(ctx, soAbertos)
}

// ResolverLeitura diz o que o leitor de código de barras leu.
func (uc *UseCase) ResolverLeitura(ctx context.Context, codigo string, almox *int64) (*entity.LeituraResolvida, error) {
	if err := uc.podeConsultar(ctx); err != nil {
		return nil, err
	}
	return uc.Ops.ResolverLeitura(ctx, codigo, almox)
}
