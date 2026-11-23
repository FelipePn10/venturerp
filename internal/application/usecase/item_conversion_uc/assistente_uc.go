package item_conversion_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/service"
)

// AssistenteConversaoDTO: o item, as medidas da peça e o que aplicar.
type AssistenteConversaoDTO struct {
	ItemCode request.TextCode `json:"item_code"`
	service.MedidasDaPeca
	// Aplicar: as sugestões a gravar ("UN>KG", "UN>MM", "UN>M2"); vazio = todas.
	Aplicar []string `json:"apply,omitempty"`
	// AplicarLoteMultiplo grava o lote múltiplo 1 no planejamento do item.
	AplicarLoteMultiplo bool `json:"apply_multiple_lot"`
}

// SugestaoAvaliada é a sugestão com o que o cadastro já tem.
type SugestaoAvaliada struct {
	service.SugestaoDeConversao
	// Atual: o fator que o item já converte entre as unidades (nil = sem conversão).
	Atual    *decimal.Decimal `json:"current_factor,omitempty"`
	Aplicada bool             `json:"applied"`
	Aviso    string           `json:"warning,omitempty"`
}

// AssistenteConversaoResponse é o resultado do assistente.
type AssistenteConversaoResponse struct {
	ItemCode     int64              `json:"item_code"`
	PesoKg       decimal.Decimal    `json:"weight_kg"`
	Formula      string             `json:"formula"`
	Sugestoes    []SugestaoAvaliada `json:"suggestions"`
	LoteMultiplo decimal.Decimal    `json:"multiple_lot"`
	Explicacao   string             `json:"multiple_lot_explanation"`
	LoteAplicado bool               `json:"multiple_lot_applied"`
}

// LoteMultiploDoItem grava o lote múltiplo do planejamento do item (repositório PG).
type LoteMultiploDoItem interface {
	DefinirLoteMultiplo(ctx context.Context, itemCode int64, lote decimal.Decimal) error
}

// Assistente calcula o peso da peça e compara as conversões sugeridas com as do cadastro.
func (uc *ItemConversionUseCase) Assistente(ctx context.Context, dto AssistenteConversaoDTO) (*AssistenteConversaoResponse, error) {
	item, err := uc.ResolveItem(ctx, dto.ItemCode)
	if err != nil {
		return nil, err
	}
	r, err := service.CalcularPeca(dto.MedidasDaPeca)
	if err != nil {
		return nil, errorsuc.NewValidationError(err.Error())
	}
	out := &AssistenteConversaoResponse{ItemCode: item, PesoKg: r.PesoKg, Formula: r.Formula, LoteMultiplo: r.LoteMultiplo, Explicacao: r.Explicacao}
	for _, s := range r.Sugestoes {
		a := SugestaoAvaliada{SugestaoDeConversao: s}
		f, achou, err := uc.FactorConfigured(ctx, item, dto.Mask(), s.De, s.Para)
		if err != nil {
			return nil, err
		}
		if achou {
			a.Atual = &f
			if !f.Round(4).Equal(s.Fator.Round(4)) {
				a.Aviso = fmt.Sprintf("o cadastro diz 1 %s = %s %s; as medidas dão %s", s.De, service.Numero(f), s.Para, service.Numero(s.Fator))
			}
		}
		out.Sugestoes = append(out.Sugestoes, a)
	}
	return out, nil
}

// AplicarAssistente grava as conversões sugeridas que o item ainda não tem
// (a que existe com outro fator não é trocada: a troca pede motivo, na VSUP0110)
// e, pedindo, o lote múltiplo 1.
func (uc *ItemConversionUseCase) AplicarAssistente(ctx context.Context, dto AssistenteConversaoDTO) (*AssistenteConversaoResponse, error) {
	out, err := uc.Assistente(ctx, dto)
	if err != nil {
		return nil, err
	}
	quer := map[string]bool{}
	for _, k := range dto.Aplicar {
		quer[strings.ToUpper(strings.TrimSpace(k))] = true
	}
	nota := "assistente de conversão: " + out.Formula
	for i := range out.Sugestoes {
		s := &out.Sugestoes[i]
		if len(quer) > 0 && !quer[s.De+">"+s.Para] {
			continue
		}
		if s.Atual != nil {
			if s.Aviso == "" {
				s.Aplicada = true // já está certa
			}
			continue
		}
		if _, err := uc.Create(ctx, request.CreateItemConversionDTO{ItemCode: dto.ItemCode, FromUOM: s.De, ToUOM: s.Para, Factor: s.Fator, Notes: &nota}); err != nil {
			return nil, fmt.Errorf("gravando 1 %s = %s %s: %w", s.De, service.Numero(s.Fator), s.Para, err)
		}
		s.Aplicada = true
	}
	if dto.AplicarLoteMultiplo {
		l, ok := uc.repo.(LoteMultiploDoItem)
		if !ok {
			return nil, errorsuc.NewValidationError("lote múltiplo indisponível nesta instalação")
		}
		if err := l.DefinirLoteMultiplo(ctx, out.ItemCode, out.LoteMultiplo); err != nil {
			return nil, err
		}
		out.LoteAplicado = true
	}
	return out, nil
}

// Mask: o assistente vale para o item sem máscara.
func (AssistenteConversaoDTO) Mask() string { return "" }
