package sales_forecast_uc

import (
	"errors"
	"strings"
	"time"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_forecast/entity"
)

// erroDeRegra converte as violações de regra do domínio em erro de validação
// TIPADO.
//
// As mensagens já eram boas e em português — "a semana deve estar entre 1 e 53",
// "a data inicial deve ser anterior à final" —, mas nasciam de `errors.New`
// puro. `RespondUseCaseError` classifica por tipo, então elas caíam no ramo
// final e o usuário recebia **"erro interno do servidor"** no lugar da
// explicação que já estava escrita. Foi assim que três telas de previsão de
// venda (VPRE0102, VPRE0201 e VPRE0251) devolviam 500 ao salvar.
func erroDeRegra(err error) error {
	if err == nil {
		return nil
	}
	for _, conhecido := range []error{
		entity.ErrInvalidWeek,
		entity.ErrInvalidYear,
		entity.ErrInvalidQuantity,
		entity.ErrInvalidItemCode,
		entity.ErrInvalidBlockDates,
		entity.ErrInvalidDescription,
		entity.ErrPercentageSumTooHigh,
	} {
		if errors.Is(err, conhecido) {
			return errorsuc.NewValidationError(err.Error())
		}
	}
	return err
}

// dataDoBloqueio converte AAAA-MM-DD nomeando o campo no erro. `time.Parse`
// devolvia o erro cru do Go — `parsing time "" as "2006-01-02"` — que virava
// 500 e não dizia ao usuário QUAL data estava errada.
func dataDoBloqueio(valor, campo string) (time.Time, error) {
	texto := strings.TrimSpace(valor)
	if texto == "" {
		return time.Time{}, errorsuc.NewValidationError("informe a " + campo)
	}
	t, err := time.Parse("2006-01-02", texto)
	if err != nil {
		return time.Time{}, errorsuc.NewValidationError(campo + " inválida: use o formato AAAA-MM-DD")
	}
	return t, nil
}
