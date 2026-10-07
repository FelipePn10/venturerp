package financial_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/accounting/contabilizacao"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	fiscalrepo "github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

// Contabil é o que a baixa precisa para contabilizar: os parâmetros da
// empresa, o vínculo das contas (plano gerencial, conta bancária) e se a nota
// de origem já foi contabilizada (então o título está em Fornecedores/Clientes).
type Contabil interface {
	AccountingParams(ctx context.Context) (*fiscalrepo.AccountingPostingParams, error)
	PlanoContasAccounts(ctx context.Context, planoIDs []int64) (map[int64]int64, error)
	ValidAccountingAccounts(ctx context.Context, planID int64, ids []int64) (map[int64]bool, error)
	ContaContabilDoBanco(ctx context.Context, contaBancariaID int64) (*int64, error)
	Contabilizado(ctx context.Context, origem string, id int64) (bool, error)
	RetencaoTipo(ctx context.Context, contaPagarID int64) (string, error)
	// FreteDoTitulo: o frete sobre compras de origem do título (nil se não é).
	FreteDoTitulo(ctx context.Context, contaPagarID int64) (*int64, error)
}

func ptr(v int64) *int64 {
	if v <= 0 {
		return nil
	}
	return &v
}

// contasValidas confere que toda conta usada é analítica do plano configurado.
func contasValidas(ctx context.Context, c Contabil, planID int64, ls []contabilizacao.Lancamento) error {
	ids := map[int64]bool{}
	for _, l := range ls {
		ids[l.DebitoID], ids[l.CreditoID] = true, true
	}
	lista := make([]int64, 0, len(ids))
	for id := range ids {
		lista = append(lista, id)
	}
	validas, err := c.ValidAccountingAccounts(ctx, planID, lista)
	if err != nil {
		return err
	}
	for _, id := range lista {
		if !validas[id] {
			return errorsuc.NewValidationError(fmt.Sprintf("a conta contábil %d não é analítica do plano contábil configurado", id))
		}
	}
	return nil
}

func faltaConta(oque string, faltando []string) error {
	return errorsuc.NewValidationError(fmt.Sprintf("a contabilização %s está ligada, mas %s. Ajuste os parâmetros contábeis (VCTB0200 › NF de entrada e ciclo) ou desligue a contabilização", oque, strings.Join(faltando, "; ")))
}

func (c baixaContabil) banco(ctx context.Context, p *fiscalrepo.AccountingPostingParams, contaBancaria int64) (*int64, error) {
	conta, err := c.ContaContabilDoBanco(ctx, contaBancaria)
	if err != nil {
		return nil, err
	}
	if conta == nil {
		conta = p.BancoPadraoAccountID
	}
	return conta, nil
}

type baixaContabil struct{ Contabil }

// lotePagamento monta a contabilização do pagamento; nil quando desligada.
func lotePagamento(ctx context.Context, c Contabil, cp *entity.ContaPagar, contaBancaria int64, principal, juros, desconto decimal.Decimal, data time.Time) (*contabilizacao.Lote, error) {
	if c == nil {
		return nil, nil
	}
	p, err := c.AccountingParams(ctx)
	if err != nil {
		return nil, err
	}
	if p == nil || !p.ContabilizarPagamentos {
		return nil, nil
	}
	bc := baixaContabil{c}
	banco, err := bc.banco(ctx, p, contaBancaria)
	if err != nil {
		return nil, err
	}
	partes, err := partesPagamento(ctx, c, p, cp)
	if err != nil {
		return nil, err
	}
	ls, faltando := contabilizacao.Pagamento(contabilizacao.Baixa{
		Banco: banco, Partes: partes, Principal: principal, Juros: juros, Desconto: desconto,
		ContaJuros: p.JurosPagosAccountID, ContaDesconto: p.DescontosObtidosAccountID,
		Historico: fmt.Sprintf("Pagamento %s", cp.NumeroDocumento),
	})
	if len(faltando) > 0 {
		return nil, faltaConta("dos pagamentos", faltando)
	}
	if err := contasValidas(ctx, c, p.PlanID, ls); err != nil {
		return nil, err
	}
	return &contabilizacao.Lote{PlanID: p.PlanID, Data: data, Prefixo: "PAG", SourceType: contabilizacao.OrigemPagamento, SourceID: cp.ID, Lancamentos: ls}, nil
}

// partesPagamento: a contrapartida do pagamento.
//   - título de nota de entrada já contabilizada: Fornecedores (a despesa e os
//     créditos já foram lançados na aprovação da nota);
//   - título de retenção dessa nota: o imposto retido a recolher;
//   - título avulso: a despesa de cada plano do rateio (regime de caixa).
func partesPagamento(ctx context.Context, c Contabil, p *fiscalrepo.AccountingPostingParams, cp *entity.ContaPagar) ([]contabilizacao.Parte, error) {
	provisionado := false
	if cp.FiscalEntryID != nil {
		ok, err := c.Contabilizado(ctx, contabilizacao.OrigemEntrada, *cp.FiscalEntryID)
		if err != nil {
			return nil, err
		}
		provisionado = ok
	}
	if !provisionado {
		// Título da transportadora: o frete lançado e contabilizado já está em
		// Fornecedores.
		freteID, err := c.FreteDoTitulo(ctx, cp.ID)
		if err != nil {
			return nil, err
		}
		if freteID != nil {
			if provisionado, err = c.Contabilizado(ctx, fiscalrepo.OrigemFrete, *freteID); err != nil {
				return nil, err
			}
		}
	}
	if provisionado && strings.EqualFold(cp.TipoDocumento, "RETENCAO") {
		tipo, err := c.RetencaoTipo(ctx, cp.ID)
		if err != nil {
			return nil, err
		}
		conta := contabilizacao.ContaRetencao(tipo, p.IRRFRecolherAccountID, p.PCCRecolherAccountID, p.INSSRecolherAccountID, p.ISSRecolherAccountID)
		return []contabilizacao.Parte{{Conta: conta, Peso: decimal.NewFromInt(1), Nome: fmt.Sprintf("o %s retido a recolher", strings.ToUpper(tipo))}}, nil
	}
	if provisionado {
		return []contabilizacao.Parte{{Conta: ptr(p.FornecedoresAccountID), Peso: decimal.NewFromInt(1), Nome: "Fornecedores"}}, nil
	}
	return partesDoPlano(ctx, c, p, rateiosOuPlano(cp), "despesa")
}

type pesoPlano struct {
	plano int64
	cc    *int64
	peso  decimal.Decimal
}

func rateiosOuPlano(cp *entity.ContaPagar) []pesoPlano {
	var out []pesoPlano
	for _, r := range cp.Rateios {
		out = append(out, pesoPlano{r.PlanoContasID, r.CentroCustoID, r.Valor})
	}
	if len(out) == 0 && cp.PlanoContasID != nil {
		out = append(out, pesoPlano{*cp.PlanoContasID, cp.CentroCustoID, decimal.NewFromInt(1)})
	}
	return out
}

func partesDoPlano(ctx context.Context, c Contabil, p *fiscalrepo.AccountingPostingParams, planos []pesoPlano, natureza string) ([]contabilizacao.Parte, error) {
	if len(planos) == 0 {
		if p.DespesaPadraoAccountID != nil && natureza == "despesa" {
			return []contabilizacao.Parte{{Conta: p.DespesaPadraoAccountID, Peso: decimal.NewFromInt(1)}}, nil
		}
		return nil, nil
	}
	ids := make([]int64, 0, len(planos))
	for _, pl := range planos {
		ids = append(ids, pl.plano)
	}
	contas, err := c.PlanoContasAccounts(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]contabilizacao.Parte, 0, len(planos))
	for _, pl := range planos {
		conta := ptr(contas[pl.plano])
		if conta == nil && natureza == "despesa" {
			conta = p.DespesaPadraoAccountID
		}
		out = append(out, contabilizacao.Parte{Conta: conta, CC: pl.cc, Peso: pl.peso, Nome: fmt.Sprintf("o plano de contas %d", pl.plano)})
	}
	return out, nil
}

// loteRecebimento monta a contabilização do recebimento; nil quando desligada.
func loteRecebimento(ctx context.Context, c Contabil, cr *entity.ContaReceber, contaBancaria int64, principal, juros, desconto decimal.Decimal, data time.Time) (*contabilizacao.Lote, error) {
	if c == nil {
		return nil, nil
	}
	p, err := c.AccountingParams(ctx)
	if err != nil {
		return nil, err
	}
	if p == nil || !p.ContabilizarRecebimentos {
		return nil, nil
	}
	bc := baixaContabil{c}
	banco, err := bc.banco(ctx, p, contaBancaria)
	if err != nil {
		return nil, err
	}
	provisionado := false
	if cr.FiscalExitID != nil {
		if provisionado, err = c.Contabilizado(ctx, contabilizacao.OrigemSaida, *cr.FiscalExitID); err != nil {
			return nil, err
		}
	}
	// Crédito com o fornecedor (devolução de compra contabilizada): a
	// contrapartida é Fornecedores, que a devolução debitou.
	devolucao := false
	if !provisionado && cr.FiscalExitID != nil {
		if devolucao, err = c.Contabilizado(ctx, fiscalrepo.OrigemDevolucao, *cr.FiscalExitID); err != nil {
			return nil, err
		}
	}
	var partes []contabilizacao.Parte
	switch {
	case provisionado:
		partes = []contabilizacao.Parte{{Conta: p.ClientesAccountID, Peso: decimal.NewFromInt(1), Nome: "Clientes"}}
	case devolucao:
		partes = []contabilizacao.Parte{{Conta: ptr(p.FornecedoresAccountID), Peso: decimal.NewFromInt(1), Nome: "Fornecedores"}}
	default:
		var planos []pesoPlano
		if cr.PlanoContasID != nil {
			planos = append(planos, pesoPlano{*cr.PlanoContasID, cr.CentroCustoID, decimal.NewFromInt(1)})
		}
		if partes, err = partesDoPlano(ctx, c, p, planos, "receita"); err != nil {
			return nil, err
		}
	}
	doc := ""
	if cr.NumeroDocumento != nil {
		doc = *cr.NumeroDocumento
	}
	ls, faltando := contabilizacao.Recebimento(contabilizacao.Baixa{
		Banco: banco, Partes: partes, Principal: principal, Juros: juros, Desconto: desconto,
		ContaJuros: p.JurosRecebidosAccountID, ContaDesconto: p.DescontosConcedidosAccountID,
		Historico: fmt.Sprintf("Recebimento %s", doc),
	})
	if len(faltando) > 0 {
		return nil, faltaConta("dos recebimentos", faltando)
	}
	if err := contasValidas(ctx, c, p.PlanID, ls); err != nil {
		return nil, err
	}
	return &contabilizacao.Lote{PlanID: p.PlanID, Data: data, Prefixo: "REC", SourceType: contabilizacao.OrigemRecebimento, SourceID: cr.ID, Lancamentos: ls}, nil
}
