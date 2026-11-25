package structure_uc

import errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"

func validarIndicadoresECustos(
	isCoproduct, isFixedQty, customerSupplied bool,
	setupLoss, costLoss float64,
	costLossType string,
) error {
	if isCoproduct && isFixedQty {
		return errorsuc.NewValidationError("co-produto e quantidade por ordem não podem ser marcados juntos")
	}
	if isCoproduct && customerSupplied {
		return errorsuc.NewValidationError("co-produto sai da ordem; não pode ser também material fornecido pelo cliente")
	}
	if setupLoss < 0 {
		return errorsuc.NewValidationError("a perda de preparação não pode ser negativa")
	}
	if costLoss < 0 {
		return errorsuc.NewValidationError("a perda de custo não pode ser negativa")
	}
	if (costLossType == "" || costLossType == "PERCENTUAL") && costLoss > 100 {
		return errorsuc.NewValidationError("a perda de custo percentual deve estar entre 0 e 100")
	}
	return nil
}
