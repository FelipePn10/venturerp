package security

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

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
	return decode(w, r, dst, false, false)
}

// DecodeBodyStrict é DecodeBody recusando campos que o servidor não conhece —
// para as rotas que já eram estritas (evita gravar algo que a tela achou que
// enviou num campo de outro nome).
func DecodeBodyStrict(w http.ResponseWriter, r *http.Request, dst any) bool {
	return decode(w, r, dst, true, false)
}

// DecodeBodyOptional é DecodeBody para rotas cujo corpo é opcional (um motivo,
// uma opção): corpo vazio vale os padrões, mas JSON malformado é recusado —
// antes era ignorado, e a operação seguia sem o motivo que o usuário digitou.
func DecodeBodyOptional(w http.ResponseWriter, r *http.Request, dst any) bool {
	if r.Body == nil || r.Body == http.NoBody {
		return true
	}
	return decode(w, r, dst, false, true)
}

func decode(w http.ResponseWriter, r *http.Request, dst any, estrito, opcional bool) bool {
	dec := json.NewDecoder(r.Body)
	if estrito {
		dec.DisallowUnknownFields()
	}
	err := dec.Decode(dst)
	if err == nil {
		return true
	}
	if msg := err.Error(); estrito && strings.HasPrefix(msg, "json: unknown field ") {
		RespondErrorCode(w, http.StatusBadRequest, "CAMPO_DESCONHECIDO",
			"o campo "+strings.TrimPrefix(msg, "json: unknown field ")+" não é aceito nesta operação")
		return false
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
		if opcional {
			return true
		}
		RespondErrorCode(w, http.StatusBadRequest, "CORPO_VAZIO",
			"a requisição chegou sem dados")
		return false
	}

	// JSON quebrado ou cortado: o texto do decodificador ("invalid character
	// 'x' looking for…", "unexpected EOF") não diz nada a quem usa o sistema.
	var sintaxe *json.SyntaxError
	if errors.As(err, &sintaxe) {
		RespondErrorCode(w, http.StatusBadRequest, "CORPO_INVALIDO",
			fmt.Sprintf("os dados enviados não estão no formato esperado (erro perto da posição %d)", sintaxe.Offset))
		return false
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		RespondErrorCode(w, http.StatusBadRequest, "CORPO_INCOMPLETO",
			"os dados enviados chegaram incompletos")
		return false
	}

	RespondErrorCode(w, http.StatusBadRequest, "CORPO_INVALIDO",
		"não foi possível ler os dados enviados")
	return false
}
