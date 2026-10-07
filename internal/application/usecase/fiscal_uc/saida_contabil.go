package fiscal_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/accounting/contabilizacao"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

// SaidaContabil é o que a NF-e de saída precisa para contabilizar a venda
// (receita, impostos sobre vendas e CMV) e para desfazer tudo no cancelamento.
type SaidaContabil interface {
	AccountingParams(ctx context.Context) (*repository.AccountingPostingParams, error)
	ValidAccountingAccounts(ctx context.Context, planID int64, ids []int64) (map[int64]bool, error)
	CustoDaSaida(ctx context.Context, exitID int64) (decimal.Decimal, error)
	GravarLoteContabil(ctx context.Context, lote contabilizacao.Lote) error
	EstornarLoteContabil(ctx context.Context, origem string, id int64, origemEstorno, prefixo string, data time.Time) (int, error)
	EstornarEstoqueDaSaida(ctx context.Context, exitID int64, userID uuid.UUID) (int, error)
}

func moedaF(v float64) decimal.Decimal { return decimal.NewFromFloat(v).Round(2) }

// contabilizarSaida lança a venda depois da autorização. A NF-e já existe na
// SEFAZ e não se desfaz: falta de conta ou erro viram aviso na resposta, para
// alguém completar a configuração e lançar.
func contabilizarSaida(ctx context.Context, c SaidaContabil, exit *entity.FiscalExit, geraReceita bool) []string {
	if c == nil {
		return nil
	}
	p, err := c.AccountingParams(ctx)
	if err != nil {
		return []string{"contabilização da venda não feita: " + err.Error()}
	}
	if p == nil || !p.ContabilizarSaidas {
		return nil
	}
	cmv, err := c.CustoDaSaida(ctx, exit.ID)
	if err != nil {
		return []string{"contabilização da venda não feita: " + err.Error()}
	}
	ls, faltando := contabilizacao.NotaSaida(contabilizacao.Saida{
		Total: moedaF(exit.ValorTotal), IPI: moedaF(exit.ValorIPI), ICMSST: moedaF(exit.ValorICMSST),
		ICMS: moedaF(exit.ValorICMS), PIS: moedaF(exit.ValorPIS), COFINS: moedaF(exit.ValorCOFINS), CMV: cmv.Round(2),
		GeraReceita: geraReceita, Historico: fmt.Sprintf("NF-e %d/%s", exit.NumeroNF, exit.Serie),
	}, contabilizacao.ContasSaida{
		Clientes: p.ClientesAccountID, Receita: p.ReceitaVendasAccountID, IPIRecolher: p.IPIRecolherAccountID,
		ICMSSTRecolher: p.ICMSSTRecolherAccountID, ICMSVendas: p.ICMSVendasAccountID, ICMSRecolher: p.ICMSRecolherAccountID,
		PISVendas: p.PISVendasAccountID, PISRecolher: p.PISRecolherAccountID, COFINSVendas: p.COFINSVendasAccountID,
		COFINSRec: p.COFINSRecolherAccountID, CMV: p.CMVAccountID, Estoque: p.EstoqueAccountID,
	})
	if len(faltando) > 0 {
		return []string{"a contabilização das saídas está ligada, mas " + strings.Join(faltando, "; ") + ": a venda não foi contabilizada"}
	}
	ids := map[int64]bool{}
	for _, l := range ls {
		ids[l.DebitoID], ids[l.CreditoID] = true, true
	}
	lista := make([]int64, 0, len(ids))
	for id := range ids {
		lista = append(lista, id)
	}
	validas, err := c.ValidAccountingAccounts(ctx, p.PlanID, lista)
	if err != nil {
		return []string{"contabilização da venda não feita: " + err.Error()}
	}
	for _, id := range lista {
		if !validas[id] {
			return []string{fmt.Sprintf("a conta contábil %d não é analítica do plano configurado: a venda não foi contabilizada", id)}
		}
	}
	lote := contabilizacao.Lote{PlanID: p.PlanID, Data: exit.DataEmissao, Prefixo: "NFS", SourceType: contabilizacao.OrigemSaida, SourceID: exit.ID, Lancamentos: ls}
	if err := c.GravarLoteContabil(ctx, lote); err != nil {
		return []string{"contabilização da venda não feita: " + err.Error()}
	}
	return nil
}

// desfazerSaida devolve o estoque e estorna a contabilização da nota cancelada.
func desfazerSaida(ctx context.Context, c SaidaContabil, exit *entity.FiscalExit, userID uuid.UUID) []string {
	if c == nil {
		return nil
	}
	var avisos []string
	if _, err := c.EstornarEstoqueDaSaida(ctx, exit.ID, userID); err != nil {
		avisos = append(avisos, "a NF-e foi cancelada, mas o estoque não voltou: "+err.Error())
	}
	if _, err := c.EstornarLoteContabil(ctx, contabilizacao.OrigemSaida, exit.ID, contabilizacao.OrigemSaidaEstorno, "NFSE", time.Now()); err != nil {
		avisos = append(avisos, "a NF-e foi cancelada, mas a contabilização não foi estornada: "+err.Error())
	}
	return avisos
}
