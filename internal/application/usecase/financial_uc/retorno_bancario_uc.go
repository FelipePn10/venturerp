package financial_uc

import (
	"context"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/retorno"
)

// TitulosPorNossoNumero acha o título do boleto (repositório PG).
type TitulosPorNossoNumero interface {
	ContaReceberPorNossoNumero(ctx context.Context, nossoNumero string, conta int64) (id int64, aberto bool, saldo decimal.Decimal, err error)
}

// RetornoBancarioUseCase dá baixa automática nos boletos pagos que vêm no
// retorno CNAB 240 do banco (melhoria 10). Antes o sistema só gerava a remessa:
// cada boleto pago era baixado à mão.
type RetornoBancarioUseCase struct {
	Titulos TitulosPorNossoNumero
	Baixa   *BaixarContaReceberUseCase
}

// RetornoDTO: a conta bancária do crédito e o arquivo (texto).
type RetornoDTO struct {
	ContaBancariaID int64  `json:"conta_bancaria_id"`
	Conteudo        string `json:"content"`
}

// ResultadoRetorno resume o processamento; repetir o arquivo não baixa de novo.
type ResultadoRetorno struct {
	Baixados          []string `json:"settled"`
	JaBaixados        []string `json:"already_settled"`
	NaoEncontrados    []string `json:"not_found"`
	Falhas            []string `json:"failures"`
	OutrasOcorrencias int      `json:"other_occurrences"`
}

func (uc *RetornoBancarioUseCase) Processar(ctx context.Context, dto RetornoDTO) (*ResultadoRetorno, error) {
	if !uc.Baixa.Auth.CanBaixarContaReceber(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.ContaBancariaID <= 0 {
		return nil, errorsuc.NewValidationError("informe a conta bancária do crédito")
	}
	liq, outras, err := retorno.Ler(dto.Conteudo)
	if err != nil {
		return nil, errorsuc.NewValidationError("arquivo de retorno inválido: " + err.Error())
	}
	out := &ResultadoRetorno{Baixados: []string{}, JaBaixados: []string{}, NaoEncontrados: []string{}, Falhas: []string{}, OutrasOcorrencias: len(outras)}
	for _, l := range liq {
		id, aberto, saldo, err := uc.Titulos.ContaReceberPorNossoNumero(ctx, l.NossoNumero, dto.ContaBancariaID)
		if errors.Is(err, retorno.ErrNossoNumeroAmbiguo) {
			out.Falhas = append(out.Falhas, fmt.Sprintf("%s: %v", l.NossoNumero, err))
			continue
		}
		if err != nil {
			return nil, err
		}
		switch {
		case id == 0:
			out.NaoEncontrados = append(out.NaoEncontrados, l.NossoNumero)
			continue
		case !aberto:
			out.JaBaixados = append(out.JaBaixados, l.NossoNumero)
			continue
		}
		// O valor pago inclui os encargos que o banco cobrou: o principal é o resto.
		principal := decimal.Min(l.ValorPago.Sub(l.Juros), saldo.Sub(l.Desconto)).Round(2)
		juros := l.Juros
		obs := fmt.Sprintf("baixa pelo retorno bancário (nosso número %s, movimento %s)", l.NossoNumero, l.Movimento)
		if err := uc.Baixa.Execute(ctx, id, request.BaixarContaReceberDTO{ContaBancariaID: dto.ContaBancariaID, ValorRecebido: principal,
			Desconto: l.Desconto, DataRecebimento: l.Credito.Format("2006-01-02"), Observacao: &obs, EncargosInformados: &juros}); err != nil {
			out.Falhas = append(out.Falhas, fmt.Sprintf("%s: %v", l.NossoNumero, err))
			continue
		}
		out.Baixados = append(out.Baixados, l.NossoNumero)
	}
	return out, nil
}
