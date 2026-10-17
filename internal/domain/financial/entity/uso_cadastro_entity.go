package entity

// UsoCadastro diz onde um cadastro (plano de contas, centro de custo) está em
// uso e quantos registros dependem dele. Com uso, o cadastro não é excluído e só
// a descrição e a situação mudam: trocar código, tipo ou hierarquia deixaria
// títulos, notas e rateios apontando para outra coisa.
type UsoCadastro struct {
	Onde       string
	Quantidade int64
}
