package fiscal_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

// AccountingParamsUseCase mantém as contas da contabilização automática da
// nota de entrada (fornecedores, impostos a recuperar, retenções a recolher)
// e o vínculo do plano de contas gerencial com a conta contábil.
type AccountingParamsUseCase struct {
	Docs repository.FiscalEntryDocumentRepository
	Auth ports.AuthService
}

func (uc *AccountingParamsUseCase) Get(ctx context.Context) (*repository.AccountingPostingParams, error) {
	if !uc.Auth.CanGetFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	p, err := uc.Docs.AccountingParams(ctx)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return &repository.AccountingPostingParams{}, nil
	}
	return p, nil
}

func (uc *AccountingParamsUseCase) Save(ctx context.Context, p repository.AccountingPostingParams) (*repository.AccountingPostingParams, error) {
	if !uc.Auth.CanApproveFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if p.PlanID <= 0 {
		return nil, errorsuc.NewValidationError("informe o plano contábil")
	}
	if p.FornecedoresAccountID <= 0 {
		return nil, errorsuc.NewValidationError("informe a conta contábil de fornecedores")
	}
	existe, err := uc.Docs.AccountingPlanExists(ctx, p.PlanID)
	if err != nil {
		return nil, err
	}
	if !existe {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o plano contábil %d não existe", p.PlanID))
	}
	ids := []int64{p.FornecedoresAccountID}
	for _, c := range contasOpcionais(&p) {
		if *c != nil && **c > 0 {
			ids = append(ids, **c)
		} else {
			*c = nil // 0 vindo da tela = conta não configurada
		}
	}
	validas, err := uc.Docs.ValidAccountingAccounts(ctx, p.PlanID, ids)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		if !validas[id] {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("a conta contábil %d não é analítica do plano contábil %d", id, p.PlanID))
		}
	}
	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	p.UpdatedBy = &userID
	if err := uc.Docs.SaveAccountingParams(ctx, &p); err != nil {
		return nil, err
	}
	return uc.Docs.AccountingParams(ctx)
}

// VincularPlano liga o plano de contas gerencial à conta contábil (nil desliga).
func (uc *AccountingParamsUseCase) VincularPlano(ctx context.Context, planoID int64, contaID *int64) error {
	if !uc.Auth.CanApproveFiscalEntry(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if contaID != nil && *contaID > 0 {
		p, err := uc.Docs.AccountingParams(ctx)
		if err != nil {
			return err
		}
		if p != nil {
			validas, err := uc.Docs.ValidAccountingAccounts(ctx, p.PlanID, []int64{*contaID})
			if err != nil {
				return err
			}
			if !validas[*contaID] {
				return errorsuc.NewValidationError(fmt.Sprintf("a conta contábil %d não é analítica do plano contábil configurado", *contaID))
			}
		}
	} else {
		contaID = nil
	}
	return uc.Docs.VincularPlanoContaContabil(ctx, planoID, contaID)
}

// contasOpcionais lista os campos de conta opcionais dos parâmetros.
func contasOpcionais(p *repository.AccountingPostingParams) []**int64 {
	return []**int64{&p.ICMSRecuperarAccountID, &p.IPIRecuperarAccountID, &p.PISRecuperarAccountID, &p.COFINSRecuperarAccountID,
		&p.IBSRecuperarAccountID, &p.CBSRecuperarAccountID, &p.IRRFRecolherAccountID, &p.PCCRecolherAccountID, &p.INSSRecolherAccountID,
		&p.ISSRecolherAccountID, &p.DespesaPadraoAccountID,
		&p.BancoPadraoAccountID, &p.JurosPagosAccountID, &p.DescontosObtidosAccountID, &p.ClientesAccountID,
		&p.JurosRecebidosAccountID, &p.DescontosConcedidosAccountID, &p.ReceitaVendasAccountID, &p.ICMSVendasAccountID,
		&p.ICMSRecolherAccountID, &p.ICMSSTRecolherAccountID, &p.IPIRecolherAccountID, &p.PISVendasAccountID,
		&p.PISRecolherAccountID, &p.COFINSVendasAccountID, &p.COFINSRecolherAccountID, &p.CMVAccountID, &p.EstoqueAccountID}
}

// VincularContaBancaria liga a conta bancária à conta contábil do banco (nil desliga).
func (uc *AccountingParamsUseCase) VincularContaBancaria(ctx context.Context, contaBancariaID int64, contaID *int64) error {
	if !uc.Auth.CanApproveFiscalEntry(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if contaID != nil && *contaID > 0 {
		p, err := uc.Docs.AccountingParams(ctx)
		if err != nil {
			return err
		}
		if p == nil {
			return errorsuc.NewValidationError("configure primeiro o plano contábil nos parâmetros da contabilização")
		}
		validas, err := uc.Docs.ValidAccountingAccounts(ctx, p.PlanID, []int64{*contaID})
		if err != nil {
			return err
		}
		if !validas[*contaID] {
			return errorsuc.NewValidationError(fmt.Sprintf("a conta contábil %d não é analítica do plano contábil configurado", *contaID))
		}
	} else {
		contaID = nil
	}
	return uc.Docs.VincularContaBancariaContabil(ctx, contaBancariaID, contaID)
}
