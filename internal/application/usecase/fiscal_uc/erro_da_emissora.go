package fiscal_uc

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

const tokenFocusAusente = "o token da Focus NF-e não está configurado: informe-o nos Parâmetros Fiscais (VFIS0100)"

func erroDaEmissora(ctx context.Context, acao string, nota int64, err error) error {
	return ErroDaEmissora(ctx, "a SEFAZ", acao, nota, err)
}

// ErroDaEmissora traduz a falha de falar com a Focus para quem está na tela. O
// status HTTP e o código ficam no log e no registro da comunicação; na resposta
// vai o que dá para fazer. Recusa é dado a corrigir (422), rede é 502. autoridade
// é quem recusa ("a SEFAZ", "a prefeitura"); acao é o objeto da frase: "o
// cancelamento", "a carta de correção".
func ErroDaEmissora(ctx context.Context, autoridade, acao string, nota int64, err error) error {
	slog.ErrorContext(ctx, "falha na comunicação com a Focus", "acao", acao, "documento", nota, "erro", err)

	var rejeicao *focusnfe.RejeicaoError
	if errors.As(err, &rejeicao) {
		motivo := strings.TrimSuffix(strings.TrimSpace(rejeicao.Motivo), ".")
		if motivo == "" {
			motivo = "ela não informou o motivo"
		}
		return errorsuc.NewValidationError(autoridade + " não aceitou " + acao + ": " + motivo)
	}

	var http4xx *focusnfe.HTTPError
	if errors.As(err, &http4xx) && http4xx.StatusCode < http.StatusInternalServerError {
		if http4xx.StatusCode == http.StatusUnauthorized || http4xx.StatusCode == http.StatusForbidden {
			return errorsuc.NewValidationError("a Focus NF-e recusou o token da empresa: confira-o nos Parâmetros Fiscais (VFIS0100)")
		}
		motivo := strings.TrimSuffix(strings.TrimSpace(http4xx.Mensagem), ".")
		if motivo == "" {
			motivo = "ela não informou o motivo"
		}
		return errorsuc.NewValidationError("a Focus NF-e não aceitou " + acao + ": " + motivo)
	}

	return errorsuc.NewExternalServiceError("", "não foi possível falar com o serviço de emissão de notas agora: tente de novo em instantes")
}
