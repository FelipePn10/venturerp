package types

import "fmt"

// InvalidValueError diz qual campo recebeu um valor fora da lista aceita.
//
// Existe para que a camada HTTP não precise adivinhar pelo texto do erro. Antes,
// o handler de item comparava a mensagem com "invalid TypeUnitOfMeasurementItem"
// para responder algo útil; qualquer outro enum inválido caía num
// "invalid request body" genérico, que não dizia ao usuário qual campo estava
// errado — e a comparação quebrava assim que a mensagem mudava.
type InvalidValueError struct {
	// Field é o nome do campo em português, como aparece na tela.
	Field string
	// Value é o que veio na requisição.
	Value string
	// Allowed são os valores aceitos, quando a lista é curta o bastante.
	Allowed []string
}

func (e *InvalidValueError) Error() string {
	msg := fmt.Sprintf("%s: valor %q não é aceito", e.Field, e.Value)
	if len(e.Allowed) > 0 {
		msg += ". Use um destes: "
		for i, a := range e.Allowed {
			if i > 0 {
				msg += ", "
			}
			msg += a
		}
	}
	return msg
}

// NewInvalidValue monta o erro de valor fora da lista.
func NewInvalidValue(field, value string, allowed ...string) error {
	return &InvalidValueError{Field: field, Value: value, Allowed: allowed}
}
