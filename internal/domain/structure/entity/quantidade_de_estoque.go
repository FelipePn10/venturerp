package entity

// A unidade da linha de estrutura e a unidade em que o item é ESTOCADO podem
// ser diferentes — desenhar em m² uma chapa estocada em kg é legítimo e comum.
// O que não pode é o resto do sistema ler o número achando que já está na
// unidade de estoque: o MRP reservaria 2 kg onde a engenharia pediu 2 m².
//
// O fator foi calculado e CONGELADO quando a linha foi gravada (migração 354),
// a partir do cadastro de conversões por item. Congelado de propósito: corrigir
// a conversão amanhã não pode reescrever o consumo de uma ordem já aberta.

// FatorParaEstoque converte da unidade da estrutura para a de estoque.
// Linhas anteriores à migração 354 têm fator ausente — nelas a unidade da
// estrutura era sempre a de estoque, então 1 é a resposta certa. Devolver zero
// aqui zeraria a necessidade de material e a ordem nasceria sem componente.
func (s *ItemStructure) FatorParaEstoque() float64 {
	if s.ConversionFactor > 0 {
		return s.ConversionFactor
	}
	return 1
}

// QuantidadeNaUnidadeDeEstoque é a quantidade do componente por unidade do pai,
// na unidade em que ele é estocado — a única em que faz sentido reservar,
// consumir e custear.
func (s *ItemStructure) QuantidadeNaUnidadeDeEstoque() float64 {
	return s.Quantity * s.FatorParaEstoque()
}
