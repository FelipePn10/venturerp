package ports

import "context"

// RecalculoDeEstrutura regrava, nas linhas de estrutura, a quantidade na
// unidade de estoque atual do componente (structure_uc.RecalcularFatores).
// Quem muda unidade de estoque ou conversão chama depois de gravar. Devolve
// quantas linhas mudaram e um aviso por linha que ficou sem conversão.
type RecalculoDeEstrutura interface {
	RecalcularDoComponente(ctx context.Context, itemCode int64, motivo string) (atualizadas int, avisos []string, err error)
	RecalcularDaEmpresa(ctx context.Context, motivo string) (atualizadas int, avisos []string, err error)
}
