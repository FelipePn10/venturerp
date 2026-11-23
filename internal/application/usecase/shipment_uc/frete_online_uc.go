package shipment_uc

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

// DadosDoFrete lê o que a cotação e o rastreio do romaneio precisam (repositório PG).
type DadosDoFrete interface {
	DadosDoFrete(ctx context.Context, code int64) (*ports.PedidoDeCotacao, string, error)
}

// FreteOnlineUseCase cota o romaneio no hub de frete e rastreia a carga pela
// chave da NF-e (melhoria 13; provedor em docs/dev/integracoes-provedores.md).
type FreteOnlineUseCase struct {
	Repo     DadosDoFrete
	Cotacao  ports.CotacaoDeFrete
	Rastreio ports.RastreioDeCarga
}

func desligada(err error) error {
	if errors.Is(err, ports.ErrIntegracaoDesligada) {
		return errorsuc.NewValidationError("a integração com o hub de frete não está configurada (FRETERAPIDO_TOKEN)")
	}
	return err
}

func (uc *FreteOnlineUseCase) Cotar(ctx context.Context, code int64) ([]ports.OfertaDeFrete, error) {
	p, _, err := uc.Repo.DadosDoFrete(ctx, code)
	if err != nil {
		return nil, err
	}
	if len(p.Volumes) == 0 {
		return nil, errorsuc.NewValidationError("monte os volumes do romaneio (medidas e peso) antes de cotar o frete")
	}
	if p.CEPOrigem == "" || p.CNPJEmbarcador == "" {
		return nil, errorsuc.NewValidationError("informe CNPJ e CEP da empresa na Configuração Fiscal (VFIS0100)")
	}
	if p.CEPDestino == "" {
		return nil, errorsuc.NewValidationError("o cliente do pedido não tem CEP de entrega cadastrado")
	}
	of, err := uc.Cotacao.Cotar(ctx, *p)
	if err != nil {
		return nil, desligada(err)
	}
	if of == nil {
		of = []ports.OfertaDeFrete{}
	}
	return of, nil
}

func (uc *FreteOnlineUseCase) Rastrear(ctx context.Context, code int64) ([]ports.EventoDeRastreio, error) {
	_, chave, err := uc.Repo.DadosDoFrete(ctx, code)
	if err != nil {
		return nil, err
	}
	if chave == "" {
		return nil, errorsuc.NewValidationError("o romaneio ainda não tem NF-e autorizada: o rastreio usa a chave da nota")
	}
	ev, err := uc.Rastreio.Rastrear(ctx, "", chave)
	if err != nil {
		return nil, desligada(err)
	}
	if ev == nil {
		ev = []ports.EventoDeRastreio{}
	}
	return ev, nil
}
