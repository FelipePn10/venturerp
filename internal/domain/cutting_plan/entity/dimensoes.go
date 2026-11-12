package entity

import (
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
)

// Cada tipo de corte lê uma medida: barra (1D) o comprimento, chapa (2D) e
// contorno real a largura e a altura. Peça ou estoque sem a medida do tipo do
// plano não entra no encaixe — o otimizador ignorava a barra sem comprimento e
// devolvia um plano "pronto" com todas as peças sem lugar, em vez de um erro.

func nomeDaPeca(p *CuttingPlanPart) string {
	if s := strings.TrimSpace(p.Label); s != "" {
		return fmt.Sprintf("a peça %q", s)
	}
	if p.ItemCode != nil {
		return fmt.Sprintf("a peça do item %d", *p.ItemCode)
	}
	return "a peça"
}

// ConferirPeca recusa a peça sem a medida que o tipo de corte do plano usa.
func (cp *CuttingPlan) ConferirPeca(p *CuttingPlanPart) error {
	switch cp.CutType {
	case CutTypeLinear1D:
		if p.LengthMM <= 0 {
			return domainerr.NewValidation(fmt.Sprintf("%s não tem comprimento: o plano é de corte em barra (1D) e precisa do comprimento de cada peça", nomeDaPeca(p)))
		}
	case CutTypeGuillotine2D, CutTypeTrueShape2D:
		if p.WidthMM <= 0 || p.HeightMM <= 0 {
			return domainerr.NewValidation(fmt.Sprintf("%s não tem largura e altura: o plano é de corte em chapa (2D) e precisa das duas medidas de cada peça", nomeDaPeca(p)))
		}
	}
	return nil
}

// ConferirEstoque recusa a barra ou chapa sem a medida do tipo de corte do plano.
func (cp *CuttingPlan) ConferirEstoque(s *CuttingStockPiece) error {
	qual := "a barra"
	if s.IsRemnant {
		qual = "o retalho"
	}
	switch cp.CutType {
	case CutTypeLinear1D:
		if s.LengthMM <= 0 {
			return domainerr.NewValidation(fmt.Sprintf("%s do estoque não tem comprimento: o plano é de corte em barra (1D)", qual))
		}
	case CutTypeGuillotine2D, CutTypeTrueShape2D:
		if s.WidthMM <= 0 || s.HeightMM <= 0 {
			if !s.IsRemnant {
				qual = "a chapa"
			}
			return domainerr.NewValidation(fmt.Sprintf("%s do estoque não tem largura e altura: o plano é de corte em chapa (2D)", qual))
		}
	}
	return nil
}

// ConferirDimensoes confere todas as peças e todo o estoque do plano.
func (cp *CuttingPlan) ConferirDimensoes(parts []*CuttingPlanPart, stock []*CuttingStockPiece) error {
	for _, p := range parts {
		if err := cp.ConferirPeca(p); err != nil {
			return err
		}
	}
	for _, s := range stock {
		if err := cp.ConferirEstoque(s); err != nil {
			return err
		}
	}
	return nil
}
