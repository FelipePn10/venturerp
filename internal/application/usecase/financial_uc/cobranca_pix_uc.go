package financial_uc

import (
	"context"
	"errors"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

// GravadorDePix grava a cobrança gerada no título e lê o pagador (repositório PG).
type GravadorDePix interface {
	SalvarCobrancaDoTitulo(ctx context.Context, id int64, g *ports.PixGerado) error
	PagadorDoCliente(ctx context.Context, cliente int64) (*ports.Pagador, error)
	CobrancaDoTitulo(ctx context.Context, id int64) (string, error)
}

// CobrancaNoBanco tira do banco a cobrança de um título que deixou de ser
// cobrável (cancelado, ou recebido por outro meio): sem isto o cliente ainda
// conseguia pagar o boleto/PIX e pagava duas vezes.
type CobrancaNoBanco struct {
	Cobranca ports.CobrancaBancaria
	Titulos  interface {
		CobrancaDoTitulo(ctx context.Context, id int64) (string, error)
	}
}

// Cancelar: nada a fazer sem cobrança emitida; integração desligada não impede
// (não há como falar com o banco), mas qualquer outra falha volta ao chamador.
func (b *CobrancaNoBanco) Cancelar(ctx context.Context, id int64, motivo string) error {
	if b == nil || b.Cobranca == nil || b.Titulos == nil {
		return nil
	}
	ext, err := b.Titulos.CobrancaDoTitulo(ctx, id)
	if err != nil || ext == "" {
		return err
	}
	if err := b.Cobranca.CancelarCobranca(ctx, ext, motivo); err != nil && !errors.Is(err, ports.ErrIntegracaoDesligada) {
		return fmt.Errorf("cancelando a cobrança %s no banco: %w", ext, err)
	}
	return nil
}

// SeuNumeroDoTitulo é como o título é identificado no banco (até 15 caracteres).
func SeuNumeroDoTitulo(id int64) string { return fmt.Sprintf("CR%d", id) }

// CobrancaPixUseCase gera o PIX do título a receber pelo banco/PSP configurado.
type CobrancaPixUseCase struct {
	Repo     repository.FinancialRepository
	Auth     ports.AuthService
	Cobranca ports.CobrancaBancaria
	Gravar   GravadorDePix
}

func (uc *CobrancaPixUseCase) Gerar(ctx context.Context, id int64) (*ports.PixGerado, error) {
	if !uc.Auth.CanBaixarContaReceber(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	cr, err := uc.Repo.GetContaReceber(ctx, id)
	if err != nil {
		return nil, err
	}
	if cr.Status != entity.ContaReceberStatusPendente && cr.Status != entity.ContaReceberStatusAprovado && cr.Status != entity.ContaReceberStatusVencido {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("só título em aberto gera cobrança; este está %s", cr.Status))
	}
	saldo := cr.ValorBruto.Sub(cr.ValorRecebido).Sub(cr.Desconto)
	if !saldo.IsPositive() {
		return nil, errorsuc.NewValidationError("o título não tem saldo a cobrar")
	}
	doc := fmt.Sprint(cr.ID)
	if cr.NumeroDocumento != nil {
		doc = *cr.NumeroDocumento
	}
	var pagador ports.Pagador
	if cr.ClienteID != nil {
		p, err := uc.Gravar.PagadorDoCliente(ctx, *cr.ClienteID)
		if err != nil {
			return nil, err
		}
		pagador = *p
	}
	// Já emitida: consulta a mesma (o banco ainda podia estar processando). Só
	// emite outra se a anterior foi cancelada ou expirou no banco.
	if ext, err := uc.Gravar.CobrancaDoTitulo(ctx, cr.ID); err != nil {
		return nil, err
	} else if ext != "" {
		atual, err := uc.Cobranca.ConsultarCobranca(ctx, ext)
		if errors.Is(err, ports.ErrIntegracaoDesligada) {
			return nil, errorsuc.NewValidationError("a cobrança bancária (boleto com PIX) não está configurada: o administrador precisa ligar a integração com o Banco Inter")
		}
		if err != nil {
			return nil, fmt.Errorf("consultando a cobrança %s no banco: %w", ext, err)
		}
		if atual.Situacao != "CANCELADO" && atual.Situacao != "EXPIRADO" && atual.Situacao != "FALHA_EMISSAO" {
			if err := uc.Gravar.SalvarCobrancaDoTitulo(ctx, cr.ID, atual); err != nil {
				return nil, err
			}
			return atual, nil
		}
	}
	pix, err := uc.Cobranca.GerarPix(ctx, ports.CobrancaPix{TituloID: cr.ID, SeuNumero: SeuNumeroDoTitulo(cr.ID), Documento: doc,
		Valor: saldo.Round(2), Vencimento: cr.DataVencimento, Pagador: pagador})
	if errors.Is(err, ports.ErrIntegracaoDesligada) {
		return nil, errorsuc.NewValidationError("a cobrança bancária (boleto com PIX) não está configurada: o administrador precisa ligar a integração com o Banco Inter")
	}
	if err != nil {
		return nil, fmt.Errorf("gerando o PIX no banco: %w", err)
	}
	if err := uc.Gravar.SalvarCobrancaDoTitulo(ctx, cr.ID, pix); err != nil {
		return nil, err
	}
	return pix, nil
}
