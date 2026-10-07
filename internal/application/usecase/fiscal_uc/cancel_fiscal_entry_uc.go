package fiscal_uc

import (
	"context"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	financialrepo "github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

// CancelFiscalEntryUseCase cancela a nota de entrada. Pendente ou conferida,
// só sai de cena (e a chave fica livre para reimportar o XML certo).
// Aprovada, estorna tudo o que a aprovação fez — títulos, estoque, pedido de
// compra, contabilização e créditos de imposto —, desde que nenhum título
// tenha sido pago e o material ainda esteja no estoque. Material já usado não
// se cancela: é devolução ao fornecedor (nota de devolução).
type CancelFiscalEntryUseCase struct {
	Docs          repository.FiscalEntryDocumentRepository
	Fiscal        repository.FiscalRepository
	FinancialRepo financialrepo.FinancialRepository
	Auth          ports.AuthService
}

func (uc *CancelFiscalEntryUseCase) Execute(ctx context.Context, id int64, dto request.CancelFiscalEntryDTO) (*response.FiscalEntryResponse, error) {
	if !uc.Auth.CanApproveFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	motivo := strings.TrimSpace(dto.Motivo)
	if len([]rune(motivo)) < 10 {
		return nil, errorsuc.NewValidationError("informe o motivo do cancelamento (pelo menos 10 caracteres)")
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	antes, err := uc.Docs.GetEntryDocument(ctx, id)
	if err != nil {
		return nil, err
	}
	res, err := uc.Docs.CancelarEntrada(ctx, repository.CancelamentoEntrada{EntryID: id, UserID: userID, Motivo: motivo})
	if err != nil {
		return nil, err
	}
	var avisos []string
	if res.EraAprovada {
		avisos = append(avisos, lancarCreditosDaNota(ctx, uc.FinancialRepo, antes, decimal.NewFromInt(-1))...)
	}
	doc, err := uc.Docs.GetEntryDocument(ctx, id)
	if err != nil {
		return nil, err
	}
	doc.Warnings = avisos
	return (&EntradaServico{Docs: uc.Docs, Fiscal: uc.Fiscal}).Responder(ctx, doc)
}
