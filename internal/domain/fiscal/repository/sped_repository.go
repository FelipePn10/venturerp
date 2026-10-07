package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/sped"
)

// SpedRepository lê, da empresa do contexto, o que a EFD ICMS/IPI do período
// declara: a empresa (configuração fiscal), as NF-e de entrada aprovadas pela
// data de entrada, as NF-e próprias autorizadas e canceladas pela data de
// emissão e os CT-e de frete lançados.
type SpedRepository interface {
	CarregarPeriodoEFD(ctx context.Context, inicio, fim time.Time) (*sped.DadosEFD, error)
	// InventarioEFD: o estoque próprio da empresa no fim do dia `data`, por
	// item, com quantidade e valor somados dos movimentos até ali; a conta
	// contábil de estoque dos parâmetros de contabilização (vazia sem eles).
	InventarioEFD(ctx context.Context, data time.Time) (itens []sped.ItemInventario, contaEstoque string, err error)
}
