package security

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	enums "github.com/FelipePn10/panossoerp/internal/domain/enums/types"
)

// DecodeBody lê o corpo JSON da requisição e, quando ele não serve, responde
// explicando o motivo em português.
//
// Existe porque os handlers respondiam "invalid request body" para qualquer
// falha de leitura. Com um formulário grande — o cadastro de item, por exemplo —
// o usuário preenchia tudo, recebia essa frase e não tinha como saber qual dos
// campos o servidor recusou. Agora um valor fora de uma lista fechada devolve
// 422 dizendo o campo, o valor recebido e o que é aceito.
//
// Devolve false quando já respondeu; nesse caso o handler deve apenas retornar.
func DecodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	err := json.NewDecoder(r.Body).Decode(dst)
	if err == nil {
		return true
	}

	var invalid *enums.InvalidValueError
	if errors.As(err, &invalid) {
		RespondErrorCode(w, http.StatusUnprocessableEntity, "VALOR_NAO_ACEITO", invalid.Error())
		return false
	}

	var unmarshalType *json.UnmarshalTypeError
	if errors.As(err, &unmarshalType) {
		campo := unmarshalType.Field
		if campo == "" {
			campo = "um dos campos"
		}
		RespondErrorCode(w, http.StatusUnprocessableEntity, "TIPO_INVALIDO",
			"o campo "+campo+" recebeu um valor de tipo incompatível")
		return false
	}

	if errors.Is(err, io.EOF) {
		RespondErrorCode(w, http.StatusBadRequest, "CORPO_VAZIO",
			"a requisição chegou sem dados")
		return false
	}

	RespondErrorCode(w, http.StatusBadRequest, "CORPO_INVALIDO",
		"não foi possível ler os dados enviados: "+err.Error())
	return false
}
