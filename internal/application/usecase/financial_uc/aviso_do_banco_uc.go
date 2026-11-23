package financial_uc

import (
	"context"
	"errors"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
)

// TitulosDoBanco acha o título pelo "seu número" mandado ao banco.
type TitulosDoBanco interface {
	TituloPorSeuNumero(ctx context.Context, seuNumero string) (id int64, aberto bool, saldo decimal.Decimal, err error)
}

// AvisoDoBancoUseCase dá baixa no título quando o banco avisa o pagamento
// (webhook). O corpo do aviso não é tomado como verdade: cada cobrança avisada é
// consultada no banco, e só a situação "recebido" da consulta baixa o título.
// Repetir o aviso não baixa de novo (o título já não está em aberto).
type AvisoDoBancoUseCase struct {
	Cobranca        ports.CobrancaBancaria
	Titulos         TitulosDoBanco
	Baixa           *BaixarContaReceberUseCase
	ContaBancariaID int64
}

// ResultadoAviso: o que cada cobrança avisada virou.
type ResultadoAviso struct {
	Baixados, Ignorados []string
}

// Processar consulta e baixa. Erro de consulta volta como erro (o banco reenvia).
func (uc *AvisoDoBancoUseCase) Processar(ctx context.Context, codigos []string) (*ResultadoAviso, error) {
	out := &ResultadoAviso{}
	for _, cod := range codigos {
		p, err := uc.Cobranca.ConsultarPagamento(ctx, cod)
		if err != nil {
			return out, fmt.Errorf("consultando a cobrança %s no banco: %w", cod, err)
		}
		if !p.Pago || !p.Valor.IsPositive() {
			out.Ignorados = append(out.Ignorados, cod+": "+p.Situacao)
			continue
		}
		id, aberto, saldo, err := uc.Titulos.TituloPorSeuNumero(ctx, p.SeuNumero)
		if err != nil {
			return out, err
		}
		if id == 0 || !aberto {
			out.Ignorados = append(out.Ignorados, cod+": título não encontrado ou já baixado")
			continue
		}
		principal := decimal.Min(p.Valor, saldo).Round(2)
		encargos := decimal.Max(p.Valor.Sub(principal), decimal.Zero).Round(2)
		obs := fmt.Sprintf("baixa automática pelo aviso do banco (cobrança %s, %s)", cod, p.Forma)
		if err := uc.Baixa.Execute(ctx, id, request.BaixarContaReceberDTO{ContaBancariaID: uc.ContaBancariaID, ValorRecebido: principal,
			DataRecebimento: p.Data.Format("2006-01-02"), Observacao: &obs, EncargosInformados: &encargos}); err != nil {
			// Dois avisos da mesma cobrança ao mesmo tempo: o título travado já foi
			// baixado pelo outro — não é falha (e o 503 só faria o banco reenviar).
			var conflito *errorsuc.ConflictError
			if errors.As(err, &conflito) {
				out.Ignorados = append(out.Ignorados, cod+": já baixado por outro aviso")
				continue
			}
			return out, fmt.Errorf("baixando o título %d: %w", id, err)
		}
		out.Baixados = append(out.Baixados, cod)
	}
	return out, nil
}
