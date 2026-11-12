package gantt

import (
	"time"

	uc "github.com/FelipePn10/panossoerp/internal/application/usecase/impressao_uc"
	"github.com/FelipePn10/panossoerp/internal/domain/aps/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/export/documentos"
)

// renderPDF imprime o quadro do APS com o layout da programação da produção
// (export/documentos): dias sem programação somem do eixo, o turno vira a
// escala de cada dia, as barras levam a OF e a etapa, e cada linha do quadro
// (centro de trabalho ou ordem) ganha a sua sequência para o chão de fábrica.
// O quadro antigo desenhava o mês inteiro em escala de dia — operações de
// minutos viravam riscos sem rótulo.
func renderPDF(m *entity.GanttMonth, b Branding) []byte {
	gerado := b.GeneratedAt
	if gerado.IsZero() {
		gerado = time.Now()
	}
	d := uc.Programacao{
		Cabecalho: uc.Cabecalho{Emitente: uc.Emitente{Parte: uc.Parte{Nome: b.CompanyName}, Logo: b.Logo, CorMarca: b.BrandColorHex}, Gerado: gerado},
		De:        m.RangeFrom,
		Ate:       m.RangeTo.AddDate(0, 0, -1),
		Grupo:     monthTitle(m),
	}
	if !d.Ate.After(d.De) {
		d.Ate = d.De
	}
	d.Agrupamento = "centro de trabalho"
	if m.GroupBy == entity.GroupByOrder {
		d.Agrupamento = "ordem de produção"
	}
	for _, row := range m.Rows {
		for _, bar := range row.Bars {
			if !bar.End.After(bar.Start) {
				continue
			}
			centro := bar.WorkCenterName
			if centro == "" {
				centro = row.SubLabel
			}
			op := uc.OperacaoProgramada{
				Maquina: row.Label, Centro: centro, Ordem: bar.OrderNumber, Item: bar.ItemBusinessCode, Descricao: bar.ItemName,
				Etapa: bar.SequencePosition, Operacao: bar.OperationName, Inicio: bar.Start, Fim: bar.End, Situacao: bar.Status,
			}
			if op.Ordem == 0 { // o quadro rotula a OF pelo número; sem ele, pelo id
				op.Ordem = bar.ProductionOrderID
			}
			d.Operacoes = append(d.Operacoes, op)
		}
	}
	return documentos.Gerador{}.Programacao(d)
}
