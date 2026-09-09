// Package ptrutil guarda ajudantes para campos opcionais de atualização.
package ptrutil

// BoolOr devolve o valor apontado por p; quando p é nil, devolve `atual`.
//
// É o contrato de atualização parcial: um PUT que não menciona o campo não deve
// mudá-lo. Sem isso, um `bool` ausente no corpo chega como `false` e desativa o
// cadastro — foi assim que fornecedor atualizado sumia da listagem.
func BoolOr(p *bool, atual bool) bool {
	if p == nil {
		return atual
	}
	return *p
}

// BoolOrTrue devolve o valor apontado por p; quando p é nil, devolve true.
// Usado onde o registro nasce ativo salvo indicação contrária.
func BoolOrTrue(p *bool) bool {
	return p == nil || *p
}
