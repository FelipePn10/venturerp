// Package certificado_uc emite o certificado de qualidade ao cliente a partir
// dos lotes de inspeção aprovados do item/lote vendido.
package certificado_uc

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/certificado"
	"github.com/FelipePn10/panossoerp/internal/domain/quality/lote"
)

type UseCase struct {
	Repo   dom.Repository
	Fontes dom.Fontes
	Lotes  lote.Repository
	Auth   ports.AuthService
}

type EmissaoDTO struct {
	ItemCode       int64           `json:"item_code"`
	Mask           string          `json:"mask"`
	Lote           *string         `json:"lote,omitempty"`
	Quantidade     decimal.Decimal `json:"quantidade"`
	LotesInspecao  []int64         `json:"lotes_inspecao,omitempty"`
	CustomerCode   *int64          `json:"customer_code,omitempty"`
	FiscalExitID   *int64          `json:"fiscal_exit_id,omitempty"`
	SalesOrderCode *int64          `json:"sales_order_code,omitempty"`
	Observacoes    *string         `json:"observacoes,omitempty"`
}

func limpo(s *string) *string {
	if s == nil {
		return nil
	}
	if v := strings.TrimSpace(*s); v != "" {
		return &v
	}
	return nil
}

// lotesDoCertificado: os informados ou, sem eles, os lotes de inspeção
// decididos do item com o mesmo lote/corrida.
func (uc *UseCase) lotesDoCertificado(ctx context.Context, dto EmissaoDTO) ([]*lote.Lote, error) {
	var out []*lote.Lote
	if len(dto.LotesInspecao) > 0 {
		for _, id := range dto.LotesInspecao {
			l, err := uc.Lotes.Obter(ctx, id)
			if err != nil {
				return nil, err
			}
			out = append(out, l)
		}
		return out, nil
	}
	if dto.Lote == nil {
		return nil, errorsuc.NewValidationError("informe o lote/corrida ou os lotes de inspeção do certificado")
	}
	ls, err := uc.Lotes.Listar(ctx, lote.Filtro{ItemCode: dto.ItemCode, Status: string(lote.Decidido), Limite: 500})
	if err != nil {
		return nil, err
	}
	for i := range ls {
		if ls[i].Lote != nil && *ls[i].Lote == *dto.Lote && ls[i].Mask == dto.Mask && ls[i].Decisao != nil && ls[i].Decisao.Aproveitada().IsPositive() {
			l, err := uc.Lotes.Obter(ctx, ls[i].ID)
			if err != nil {
				return nil, err
			}
			out = append(out, l)
		}
	}
	return out, nil
}

func (uc *UseCase) Emitir(ctx context.Context, dto EmissaoDTO) (*dom.Certificado, error) {
	ator, err := uc.Auth.UserID(ctx)
	if err != nil || ator == uuid.Nil {
		return nil, errorsuc.ErrUnauthorized
	}
	dto.Lote = limpo(dto.Lote)
	lotes, err := uc.lotesDoCertificado(ctx, dto)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(lotes))
	for _, l := range lotes {
		ids = append(ids, l.ID)
	}
	ja, err := uc.Repo.JaCertificado(ctx, ids)
	if err != nil {
		return nil, err
	}
	if err := dom.ConferirLotes(dto.ItemCode, dto.Quantidade, ja, lotes); err != nil {
		return nil, err
	}
	conteudo := dom.Conteudo{ItemCode: dto.ItemCode, Mask: dto.Mask, Quantidade: dto.Quantidade.String(), PedidoVenda: dto.SalesOrderCode, Declaracao: dom.Declaracao}
	if conteudo.Empresa, err = uc.Fontes.Empresa(ctx); err != nil {
		return nil, err
	}
	if conteudo.ItemNome, err = uc.Fontes.Item(ctx, dto.ItemCode); err != nil {
		return nil, err
	}
	cliente := dto.CustomerCode
	if dto.FiscalExitID != nil {
		nf, daNota, err := uc.Fontes.NotaFiscal(ctx, *dto.FiscalExitID)
		if err != nil {
			return nil, err
		}
		conteudo.NF = nf
		if cliente == nil {
			cliente = daNota
		} else if daNota != nil && *daNota != *cliente {
			return nil, errorsuc.NewValidationError("o cliente informado não é o da NF de saída")
		}
	}
	if cliente != nil {
		if conteudo.Cliente, err = uc.Fontes.Cliente(ctx, *cliente); err != nil {
			return nil, err
		}
	}
	c := &dom.Certificado{
		CustomerCode: cliente, FiscalExitID: dto.FiscalExitID, SalesOrderCode: dto.SalesOrderCode, ItemCode: dto.ItemCode, Mask: dto.Mask,
		Lote: dto.Lote, Quantidade: dto.Quantidade.Round(6), Observacoes: limpo(dto.Observacoes), EmitidoPor: &ator,
	}
	if dto.Lote != nil {
		conteudo.Lote = *dto.Lote
		corrida, cert, err := uc.Fontes.Corrida(ctx, dto.ItemCode, *dto.Lote)
		if err != nil {
			return nil, err
		}
		conteudo.Corrida, conteudo.CertUsina = corrida, cert
		c.Corrida, c.CertificadoUsina = limpo(&corrida), limpo(&cert)
	}
	ids = []int64{}
	for _, l := range lotes {
		for _, r := range l.Resultados {
			if r.InstrumentoID != nil {
				ids = append(ids, *r.InstrumentoID)
			}
		}
	}
	instr, err := uc.Fontes.Instrumentos(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, l := range lotes {
		conteudo.Lotes = append(conteudo.Lotes, dom.Resumir(l, instr))
		c.LotesInspecao = append(c.LotesInspecao, l.ID)
	}
	if c.Observacoes != nil {
		conteudo.Observacoes = *c.Observacoes
	}
	c.Conteudo = conteudo
	limite := decimal.Zero
	for _, l := range lotes {
		limite = limite.Add(l.Decisao.Aproveitada())
	}
	if err := uc.Repo.Emitir(ctx, c, limite); err != nil {
		return nil, err
	}
	return c, nil
}

func (uc *UseCase) Obter(ctx context.Context, id int64) (*dom.Certificado, error) {
	return uc.Repo.Obter(ctx, id)
}

func (uc *UseCase) Listar(ctx context.Context, f dom.Filtro) ([]dom.Certificado, error) {
	return uc.Repo.Listar(ctx, f)
}

func (uc *UseCase) Cancelar(ctx context.Context, id int64, motivo string) (*dom.Certificado, error) {
	ator, err := uc.Auth.UserID(ctx)
	if err != nil || ator == uuid.Nil {
		return nil, errorsuc.ErrUnauthorized
	}
	if motivo = strings.TrimSpace(motivo); motivo == "" {
		return nil, errorsuc.NewValidationError("informe o motivo do cancelamento")
	}
	if err := uc.Repo.Cancelar(ctx, id, motivo, ator); err != nil {
		return nil, err
	}
	return uc.Repo.Obter(ctx, id)
}
