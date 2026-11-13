package financial_uc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
	"github.com/shopspring/decimal"
)

// ConciliacaoUseCase é o trabalho que vem DEPOIS de importar o OFX: ver o extrato
// da conta, casar cada linha com o movimento do caixa, lançar o que não tem título
// (tarifa, IOF, aplicação) e desfazer o que foi casado errado.
//
// Antes só existia a importação. As linhas entravam no banco e nenhuma tela as
// mostrava; quem importava via "0 conciliados", ia ao fluxo de caixa procurar os
// pendentes que a tela prometia e não achava nada.
type ConciliacaoUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

type ExtratoLinhaResponse struct {
	ID                 int64            `json:"id"`
	Data               string           `json:"data"`
	Valor              decimal.Decimal  `json:"valor"`
	Tipo               string           `json:"tipo"` // CREDIT | DEBIT
	Descricao          string           `json:"descricao"`
	FitID              string           `json:"fitid"`
	Conciliado         bool             `json:"conciliado"`
	FluxoCaixaID       *int64           `json:"fluxo_caixa_id,omitempty"`
	MovimentoData      string           `json:"movimento_data,omitempty"`
	MovimentoDescricao string           `json:"movimento_descricao,omitempty"`
	MovimentoValor     *decimal.Decimal `json:"movimento_valor,omitempty"`
	ContasPagarID      *int64           `json:"contas_pagar_id,omitempty"`
	ContasReceberID    *int64           `json:"contas_receber_id,omitempty"`
	LancadoPeloExtrato bool             `json:"lancado_pelo_extrato"`
}

type ExtratoResumo struct {
	Total             int             `json:"total"`
	Pendentes         int             `json:"pendentes"`
	Conciliados       int             `json:"conciliados"`
	CreditosPendentes decimal.Decimal `json:"creditos_pendentes"`
	DebitosPendentes  decimal.Decimal `json:"debitos_pendentes"`
}

type ExtratoContaResponse struct {
	ContaBancariaID int64                  `json:"conta_bancaria_id"`
	SaldoSistema    decimal.Decimal        `json:"saldo_sistema"`
	Resumo          ExtratoResumo          `json:"resumo"`
	Linhas          []ExtratoLinhaResponse `json:"linhas"`
}

type CandidatoResponse struct {
	FluxoCaixaID    int64           `json:"fluxo_caixa_id"`
	Data            string          `json:"data"`
	Valor           decimal.Decimal `json:"valor"`
	Tipo            string          `json:"tipo"`
	Descricao       string          `json:"descricao"`
	Documento       string          `json:"documento,omitempty"`
	ContasPagarID   *int64          `json:"contas_pagar_id,omitempty"`
	ContasReceberID *int64          `json:"contas_receber_id,omitempty"`
	DiferencaDias   int             `json:"diferenca_dias"`
}

type AutoConciliacaoResponse struct {
	Conciliados int `json:"conciliados"`
	Pendentes   int `json:"pendentes"`
}

type LancamentoExtratoResponse struct {
	FluxoCaixaID int64 `json:"fluxo_caixa_id"`
}

const formatoData = "2006-01-02"

// ListarExtrato devolve as linhas do extrato da conta, o resumo e o saldo do
// sistema. situacao: "pendente", "conciliado" ou vazio (todas).
func (uc *ConciliacaoUseCase) ListarExtrato(ctx context.Context, contaID int64, situacao, inicio, fim string) (*ExtratoContaResponse, error) {
	if !uc.Auth.CanGetFluxoCaixa(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	filtro := repository.ExtratoFiltro{}
	switch strings.ToUpper(strings.TrimSpace(situacao)) {
	case "", "TODOS":
	case "PENDENTE", "PENDENTES":
		filtro.Situacao = repository.ExtratoPendentes
	case "CONCILIADO", "CONCILIADOS":
		filtro.Situacao = repository.ExtratoConciliados
	default:
		return nil, errorsuc.NewValidationError(fmt.Sprintf(
			"situação %q não existe: use pendente, conciliado ou deixe em branco para todas", situacao))
	}
	var err error
	if filtro.Inicio, err = dataOpcional(inicio, "data inicial"); err != nil {
		return nil, err
	}
	if filtro.Fim, err = dataOpcional(fim, "data final"); err != nil {
		return nil, err
	}
	if filtro.Inicio != nil && filtro.Fim != nil && filtro.Fim.Before(*filtro.Inicio) {
		return nil, errorsuc.NewValidationError("a data final é anterior à data inicial")
	}

	linhas, err := uc.Repo.ListExtrato(ctx, contaID, filtro)
	if err != nil {
		return nil, err
	}
	saldo, err := uc.Repo.GetSaldoConta(ctx, contaID)
	if err != nil {
		return nil, err
	}

	out := &ExtratoContaResponse{
		ContaBancariaID: contaID,
		SaldoSistema:    saldo,
		Linhas:          make([]ExtratoLinhaResponse, 0, len(linhas)),
	}
	for _, l := range linhas {
		out.Resumo.Total++
		if l.Conciliado {
			out.Resumo.Conciliados++
		} else {
			out.Resumo.Pendentes++
			if l.Tipo == "DEBIT" {
				out.Resumo.DebitosPendentes = out.Resumo.DebitosPendentes.Add(l.Valor)
			} else {
				out.Resumo.CreditosPendentes = out.Resumo.CreditosPendentes.Add(l.Valor)
			}
		}
		r := ExtratoLinhaResponse{
			ID:                 l.ID,
			Data:               l.Data.Format(formatoData),
			Valor:              l.Valor,
			Tipo:               string(l.Tipo),
			Descricao:          l.Descricao,
			FitID:              l.FitID,
			Conciliado:         l.Conciliado,
			FluxoCaixaID:       l.FluxoCaixaID,
			MovimentoValor:     l.MovimentoValor,
			ContasPagarID:      l.ContasPagarID,
			ContasReceberID:    l.ContasReceberID,
			LancadoPeloExtrato: l.LancadoPeloExtrato,
		}
		if l.MovimentoData != nil {
			r.MovimentoData = l.MovimentoData.Format(formatoData)
		}
		if l.MovimentoDescricao != nil {
			r.MovimentoDescricao = *l.MovimentoDescricao
		}
		out.Linhas = append(out.Linhas, r)
	}
	return out, nil
}

// Candidatos lista os movimentos do caixa que podem casar com a linha.
func (uc *ConciliacaoUseCase) Candidatos(ctx context.Context, extratoID int64) ([]CandidatoResponse, error) {
	if !uc.Auth.CanGetFluxoCaixa(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	cands, err := uc.Repo.CandidatosConciliacao(ctx, extratoID)
	if err != nil {
		return nil, err
	}
	out := make([]CandidatoResponse, 0, len(cands))
	for _, c := range cands {
		r := CandidatoResponse{
			FluxoCaixaID:    c.FluxoCaixaID,
			Data:            c.Data.Format(formatoData),
			Valor:           c.Valor,
			Tipo:            string(c.Tipo),
			ContasPagarID:   c.ContasPagarID,
			ContasReceberID: c.ContasReceberID,
			DiferencaDias:   c.DiferencaDias,
		}
		if c.Descricao != nil {
			r.Descricao = *c.Descricao
		}
		if c.Documento != nil {
			r.Documento = *c.Documento
		}
		out = append(out, r)
	}
	return out, nil
}

// Conciliar casa a linha do extrato com o movimento escolhido.
func (uc *ConciliacaoUseCase) Conciliar(ctx context.Context, extratoID, fluxoID int64) error {
	if !uc.Auth.CanImportarOFX(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if fluxoID <= 0 {
		return errorsuc.NewValidationError("escolha o movimento do caixa que corresponde a este lançamento")
	}
	return uc.Repo.ConciliarExtrato(ctx, extratoID, fluxoID)
}

// Lancar cria no caixa o movimento que a linha do extrato representa e já o
// concilia. É para o que não tem título por trás; pagamento de fornecedor e
// recebimento de cliente se baixam no título, que é o que fecha a carteira.
func (uc *ConciliacaoUseCase) Lancar(ctx context.Context, extratoID int64, descricao string) (*LancamentoExtratoResponse, error) {
	if !uc.Auth.CanImportarOFX(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	id, err := uc.Repo.LancarExtratoNoCaixa(ctx, extratoID, strings.TrimSpace(descricao))
	if err != nil {
		return nil, err
	}
	return &LancamentoExtratoResponse{FluxoCaixaID: id}, nil
}

// Desfazer solta a linha do movimento.
func (uc *ConciliacaoUseCase) Desfazer(ctx context.Context, extratoID int64) error {
	if !uc.Auth.CanImportarOFX(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.DesfazerConciliacao(ctx, extratoID)
}

// AutoConciliar roda a conciliação automática de novo — útil depois de baixar os
// títulos, quando os movimentos que faltavam passaram a existir.
func (uc *ConciliacaoUseCase) AutoConciliar(ctx context.Context, contaID int64) (*AutoConciliacaoResponse, error) {
	if !uc.Auth.CanImportarOFX(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	n, err := uc.Repo.AutoMatchExtrato(ctx, contaID)
	if err != nil {
		return nil, err
	}
	pend, err := uc.Repo.ListExtrato(ctx, contaID, repository.ExtratoFiltro{Situacao: repository.ExtratoPendentes})
	if err != nil {
		return nil, err
	}
	return &AutoConciliacaoResponse{Conciliados: n, Pendentes: len(pend)}, nil
}

func dataOpcional(s, campo string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(formatoData, s)
	if err != nil {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("%s inválida: use o formato AAAA-MM-DD", campo))
	}
	return &t, nil
}
