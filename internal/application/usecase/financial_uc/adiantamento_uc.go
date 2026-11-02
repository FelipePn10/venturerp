package financial_uc

import (
	"context"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/accounting/contabilizacao"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/shopspring/decimal"
)

// CreateAdiantamentoUseCase registers an advance and posts its cash movement
// (and, when the company's accounting is on, its journal entry).
type CreateAdiantamentoUseCase struct {
	Repo     repository.FinancialRepository
	Auth     ports.AuthService
	Contabil Contabil
}

func (uc *CreateAdiantamentoUseCase) Execute(ctx context.Context, dto request.CreateAdiantamentoDTO) (*response.AdiantamentoResponse, error) {
	tipo := entity.AdiantamentoTipo(dto.Tipo)
	if tipo != entity.AdiantamentoTipoPagar && tipo != entity.AdiantamentoTipoReceber {
		return nil, errorsuc.NewValidationError("escolha se o adiantamento foi pago a fornecedor ou recebido de cliente")
	}
	// Quem lança o pagamento antecipado a fornecedor é quem baixa contas a pagar;
	// o recebido de cliente, quem baixa contas a receber.
	pode := uc.Auth.CanBaixarContaPagar(ctx)
	if tipo == entity.AdiantamentoTipoReceber {
		pode = uc.Auth.CanBaixarContaReceber(ctx)
	}
	if !pode {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if dto.ParceiroID == nil || *dto.ParceiroID <= 0 {
		if tipo == entity.AdiantamentoTipoPagar {
			return nil, errorsuc.NewValidationError("informe o fornecedor que recebeu o adiantamento")
		}
		return nil, errorsuc.NewValidationError("informe o cliente que fez o adiantamento")
	}
	if dto.ValorOriginal <= 0 {
		return nil, errorsuc.NewValidationError("o valor do adiantamento precisa ser maior que zero")
	}
	if dto.ContaBancariaID == 0 {
		return nil, errorsuc.NewValidationError("informe a conta bancária")
	}

	data, err := time.Parse("2006-01-02", dto.DataAdiantamento)
	if err != nil {
		return nil, errorsuc.NewValidationError("data do adiantamento inválida: use o formato AAAA-MM-DD")
	}

	adv := &entity.Adiantamento{
		Tipo:             tipo,
		ParceiroID:       dto.ParceiroID,
		ContaBancariaID:  dto.ContaBancariaID,
		NumeroDocumento:  dto.NumeroDocumento,
		DataAdiantamento: data,
		ValorOriginal:    decimal.NewFromFloat(dto.ValorOriginal),
		Descricao:        dto.Descricao,
		CreatedBy:        userID,
	}

	// Advance to supplier = cash OUT; advance from customer = cash IN.
	fcTipo := entity.FluxoCaixaTipoSaida
	if tipo == entity.AdiantamentoTipoReceber {
		fcTipo = entity.FluxoCaixaTipoEntrada
	}
	desc := "Adiantamento"
	if dto.Descricao != nil && *dto.Descricao != "" {
		desc = *dto.Descricao
	}
	contaID := dto.ContaBancariaID
	fc := entity.FluxoCaixa{
		Data:            data,
		Tipo:            fcTipo,
		Valor:           decimal.NewFromFloat(dto.ValorOriginal),
		ContaBancariaID: &contaID,
		Descricao:       &desc,
	}

	lote, err := loteAdiantamento(ctx, uc.Contabil, adv)
	if err != nil {
		return nil, err
	}
	created, err := uc.Repo.CreateAdiantamentoAtomico(ctx, adv, fc, lote)
	if err != nil {
		return nil, err
	}
	return toAdiantamentoResponse(created), nil
}

// ListAdiantamentosUseCase lists advances, optionally filtered by tipo/parceiro.
type ListAdiantamentosUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *ListAdiantamentosUseCase) Execute(ctx context.Context, tipo *string, parceiroID *int64) ([]*response.AdiantamentoResponse, error) {
	if !uc.Auth.CanGetContaPagar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListAdiantamentos(ctx, tipo, parceiroID)
	if err != nil {
		return nil, err
	}
	return toAdiantamentoResponses(list), nil
}

// GetAdiantamentoUseCase returns one advance with its applications.
type GetAdiantamentoUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

type AdiantamentoDetail struct {
	*response.AdiantamentoResponse
	Aplicacoes []*response.AdiantamentoAplicacaoResponse `json:"aplicacoes"`
}

func (uc *GetAdiantamentoUseCase) Execute(ctx context.Context, id int64) (*AdiantamentoDetail, error) {
	if !uc.Auth.CanGetContaPagar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	adv, err := uc.Repo.GetAdiantamento(ctx, id)
	if err != nil {
		return nil, err
	}
	aplicacoes, err := uc.Repo.ListAplicacoesByAdiantamento(ctx, id)
	if err != nil {
		return nil, err
	}
	aps := make([]*response.AdiantamentoAplicacaoResponse, 0, len(aplicacoes))
	for _, a := range aplicacoes {
		aps = append(aps, toAdiantamentoAplicacaoResponse(a))
	}
	return &AdiantamentoDetail{AdiantamentoResponse: toAdiantamentoResponse(adv), Aplicacoes: aps}, nil
}

// AplicarAdiantamentoUseCase applies an advance balance onto a title.
type AplicarAdiantamentoUseCase struct {
	Repo     repository.FinancialRepository
	Auth     ports.AuthService
	Contabil Contabil
}

func (uc *AplicarAdiantamentoUseCase) Execute(ctx context.Context, advID int64, dto request.AplicarAdiantamentoDTO) (*response.AdiantamentoAplicacaoResponse, error) {
	contaTipo := dto.ContaTipo
	if contaTipo == string(entity.AdiantamentoTipoReceber) {
		if !uc.Auth.CanBaixarContaReceber(ctx) {
			return nil, errorsuc.ErrUnauthorized
		}
	} else if contaTipo == string(entity.AdiantamentoTipoPagar) {
		if !uc.Auth.CanBaixarContaPagar(ctx) {
			return nil, errorsuc.ErrUnauthorized
		}
	} else {
		return nil, errorsuc.NewValidationError("escolha se o título é uma conta a pagar ou a receber")
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if dto.Valor <= 0 {
		return nil, errorsuc.NewValidationError("o valor a aplicar precisa ser maior que zero")
	}

	data := time.Now()
	if dto.DataAplicacao != nil && *dto.DataAplicacao != "" {
		data, err = time.Parse("2006-01-02", *dto.DataAplicacao)
		if err != nil {
			return nil, errorsuc.NewValidationError("data de aplicação inválida: use o formato AAAA-MM-DD")
		}
	}

	valor := decimal.NewFromFloat(dto.Valor).Round(2)
	lote, err := uc.loteDaAplicacao(ctx, advID, contaTipo, dto.ContaID, valor, data)
	if err != nil {
		return nil, err
	}
	created, err := uc.Repo.AplicarAdiantamentoAtomico(ctx, advID, contaTipo, dto.ContaID, valor, userID, data, lote)
	if err != nil {
		return nil, err
	}
	return toAdiantamentoAplicacaoResponse(created), nil
}

// loteDaAplicacao lê o adiantamento e o título só quando a contabilização
// está configurada; as travas de negócio (parceiro, lado, saldo) ficam no
// repositório, dentro da transação com o bloqueio das linhas.
func (uc *AplicarAdiantamentoUseCase) loteDaAplicacao(ctx context.Context, advID int64, contaTipo string, contaID int64, valor decimal.Decimal, data time.Time) (*contabilizacao.Lote, error) {
	if uc.Contabil == nil {
		return nil, nil
	}
	adv, err := uc.Repo.GetAdiantamento(ctx, advID)
	if err != nil {
		return nil, err
	}
	if string(adv.Tipo) != contaTipo {
		return nil, nil // o repositório recusa com a mensagem certa
	}
	if contaTipo == string(entity.AdiantamentoTipoPagar) {
		cp, err := uc.Repo.GetContaPagar(ctx, contaID)
		if err != nil {
			return nil, err
		}
		return loteAplicacao(ctx, uc.Contabil, adv, cp, nil, valor, data)
	}
	cr, err := uc.Repo.GetContaReceber(ctx, contaID)
	if err != nil {
		return nil, err
	}
	return loteAplicacao(ctx, uc.Contabil, adv, nil, cr, valor, data)
}

// CancelarAdiantamentoUseCase desfaz um adiantamento ainda não aplicado.
type CancelarAdiantamentoUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *CancelarAdiantamentoUseCase) Execute(ctx context.Context, id int64, motivo string) (*response.AdiantamentoResponse, error) {
	adv, err := uc.Repo.GetAdiantamento(ctx, id)
	if err != nil {
		return nil, err
	}
	pode := uc.Auth.CanBaixarContaPagar(ctx)
	if adv.Tipo == entity.AdiantamentoTipoReceber {
		pode = uc.Auth.CanBaixarContaReceber(ctx)
	}
	if !pode {
		return nil, errorsuc.ErrUnauthorized
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	motivo = strings.TrimSpace(motivo)
	if len([]rune(motivo)) < 5 {
		return nil, errorsuc.NewValidationError("informe o motivo do cancelamento (ao menos 5 letras)")
	}
	cancelado, err := uc.Repo.CancelarAdiantamentoAtomico(ctx, id, motivo, userID)
	if err != nil {
		return nil, err
	}
	return toAdiantamentoResponse(cancelado), nil
}
