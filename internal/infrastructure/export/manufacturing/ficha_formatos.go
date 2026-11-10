package manufacturing

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/shopspring/decimal"
)

// Formatos e rótulos da ordem impressa (ficha_da_ordem.go desenha).

// codigoDoItem: o código de negócio (PORTA-01) é o que a fábrica conhece; o
// número interno fica entre parênteses para busca no sistema.
func codigoDoItem(code int64, negocio string) string {
	if strings.TrimSpace(negocio) == "" {
		return fmt.Sprintf("%d", code)
	}
	return fmt.Sprintf("%s (%d)", negocio, code)
}

// pecas: em unidade de peça a etapa recebe peça inteira. A quantidade com a
// perda das etapas seguintes (ex.: 122,449) vira 123 — é o que se corta.
func pecas(d decimal.Decimal, un string) string {
	switch strings.ToUpper(strings.TrimSpace(un)) {
	case "UN", "UND", "PC", "PÇ", "PCS", "PECA", "PEÇA", "CJ", "KIT":
		return qtd(d.Ceil())
	}
	return qtd(d)
}

// notasLegiveis traduz as marcas técnicas que o MRP grava na observação
// ("parent_item=30016") e descarta as vazias.
var marcaDoPai = regexp.MustCompile(`parent_item=(\d+)`)

func notasLegiveis(texto string) string {
	texto = marcaDoPai.ReplaceAllString(texto, "Abastece a ordem do item $1 (gerada pelo MRP)")
	return strings.TrimSpace(texto)
}

func horasReais(e entity.EtapaDaFicha) string {
	if e.HorasReais.IsZero() && e.Status != "DONE" {
		return ""
	}
	return horas(e.HorasReais)
}

func rotuloStatusOrdem(s string) string {
	switch s {
	case "OPEN":
		return "Aberta"
	case "RELEASED":
		return "Liberada"
	case "IN_PROGRESS":
		return "Em produção"
	case "COMPLETED":
		return "Concluída"
	case "CLOSED":
		return "Encerrada"
	case "CANCELLED", "CANCELED":
		return "Cancelada"
	}
	return s
}

func rotuloStatusEtapa(s string) string {
	switch s {
	case "PENDING":
		return "Aguardando"
	case "IN_PROGRESS":
		return "Em execução"
	case "PAUSED":
		return "Pausada"
	case "DONE":
		return "Concluída"
	case "SKIPPED":
		return "Fantasma (pula)"
	}
	return s
}

func rotuloStatusOS(s string) string {
	switch s {
	case "PLANNED":
		return "Planejada"
	case "FIRM":
		return "Firme"
	case "RELEASED_WITH_PO":
		return "Liberada com pedido"
	case "RELEASED_WITHOUT_PO":
		return "Liberada sem pedido"
	case "COMPLETED":
		return "Concluída"
	case "CANCELLED", "CANCELED":
		return "Cancelada"
	}
	return s
}

func rotuloOrigem(s string) string {
	switch strings.ToUpper(s) {
	case "MRP", "PLANNED_ORDER":
		return "MRP"
	case "MANUAL", "":
		return "Manual"
	case "SALES_ORDER":
		return "Pedido de venda"
	}
	return s
}

func qtd(d decimal.Decimal) string {
	s := d.Round(4).StringFixed(4)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	return strings.Replace(s, ".", ",", 1)
}

func horas(d decimal.Decimal) string {
	return strings.Replace(d.Round(2).StringFixed(2), ".", ",", 1)
}

func data(t *time.Time) string {
	if t == nil || t.IsZero() {
		return "—"
	}
	return t.Format("02/01/2006")
}

func vazioVira(s, padrao string) string {
	if strings.TrimSpace(s) == "" {
		return padrao
	}
	return s
}

func quebrarTexto(texto string, largura int) [][]string {
	var linhas [][]string
	for _, paragrafo := range strings.Split(texto, "\n") {
		atual := ""
		for _, palavra := range strings.Fields(paragrafo) {
			if atual != "" && len([]rune(atual))+1+len([]rune(palavra)) > largura {
				linhas = append(linhas, []string{atual})
				atual = palavra
				continue
			}
			if atual != "" {
				atual += " "
			}
			atual += palavra
		}
		if atual != "" {
			linhas = append(linhas, []string{atual})
		}
	}
	return linhas
}
