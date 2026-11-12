package cutting_plan_uc

import (
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/cutting_plan/entity"
)

// MapaCorte é o que o PDF do mapa de corte precisa: o plano, o material e os
// padrões otimizados (com as peças posicionadas).
type MapaCorte struct {
	Empresa        string
	EmpresaCNPJ    string
	Logo           []byte
	Plano          *entity.CuttingPlan
	MaterialCodigo string
	MaterialNome   string
	Padroes        []*entity.CuttingPattern
	Gerado         time.Time
}

// GeradorMapaCorte desenha o mapa de corte em PDF (layout em export/documentos).
// SVG e DXF continuam no serviço de domínio: são arquivos de máquina.
type GeradorMapaCorte interface {
	MapaCorte(d MapaCorte) []byte
}

// WithMapaPDF liga o desenho próprio do mapa de corte em PDF.
func (uc *CuttingPlanUseCase) WithMapaPDF(g GeradorMapaCorte) *CuttingPlanUseCase {
	uc.mapaPDF = g
	return uc
}
