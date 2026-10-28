// Package indicadores: indicadores do pós-venda (Fase 3.5) — custo da
// garantia, taxa de falha e MTBF da base instalada, SLA cumprido e tempo de
// atendimento.
package indicadores

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/posvenda/sla"
)

var cem = decimal.NewFromInt(100)

// Percentual de parte sobre total, com 2 casas; nil sem total.
func Percentual(parte, total int64) *decimal.Decimal {
	if total <= 0 {
		return nil
	}
	p := decimal.NewFromInt(parte).Mul(cem).Div(decimal.NewFromInt(total)).Round(2)
	return &p
}

// TaxaFalha: séries que falharam no período sobre as séries em campo (%).
func TaxaFalha(falharam, emCampo int64) *decimal.Decimal { return Percentual(falharam, emCampo) }

// MTBF de campo: dias de operação somados de todas as séries ÷ falhas. Sem
// falha, não há MTBF a mostrar (o equipamento não falhou ainda).
func MTBF(diasEmCampo decimal.Decimal, falhas int64) *decimal.Decimal {
	if falhas <= 0 {
		return nil
	}
	m := diasEmCampo.Div(decimal.NewFromInt(falhas)).Round(1)
	return &m
}

// Periodo [De, Ate) dos indicadores.
type Periodo struct {
	De  time.Time
	Ate time.Time
}

// Custo da garantia agrupado (por item, defeito ou lote).
type Grupo struct {
	Chave         string          `json:"chave"`
	Descricao     string          `json:"descricao"`
	OS            int64           `json:"os"`
	CustoGarantia decimal.Decimal `json:"custo_garantia"`
	Cobrado       decimal.Decimal `json:"cobrado"`
}

// Confiabilidade de um item com série.
type Confiabilidade struct {
	ItemCode     int64            `json:"item_code"`
	ItemNome     string           `json:"item_nome"`
	EmCampo      int64            `json:"em_campo"`
	Falharam     int64            `json:"falharam_no_periodo"`
	FalhasTotais int64            `json:"falhas_totais"`
	DiasEmCampo  decimal.Decimal  `json:"dias_em_campo"`
	TaxaFalha    *decimal.Decimal `json:"taxa_falha,omitempty"`
	MTBFDias     *decimal.Decimal `json:"mtbf_dias,omitempty"`
}

func (c *Confiabilidade) Calcular() {
	c.TaxaFalha = TaxaFalha(c.Falharam, c.EmCampo)
	c.MTBFDias = MTBF(c.DiasEmCampo, c.FalhasTotais)
}

// SLA: chamados do período com prazo e quantos cumpriram.
type SLA struct {
	ComPrazo         int64            `json:"com_prazo"`
	RespostaAvaliada int64            `json:"resposta_avaliada"`
	RespostaCumprida int64            `json:"resposta_cumprida"`
	SolucaoAvaliada  int64            `json:"solucao_avaliada"`
	SolucaoCumprida  int64            `json:"solucao_cumprida"`
	RespostaPct      *decimal.Decimal `json:"resposta_pct,omitempty"`
	SolucaoPct       *decimal.Decimal `json:"solucao_pct,omitempty"`
}

// Avaliar conta os prazos já decididos (cumpridos, estourados ou vencidos) dos
// chamados; prazo ainda correndo não entra na conta.
func (s *SLA) Avaliar(cs []sla.Chamado, agora time.Time) {
	for _, c := range cs {
		if c.Cancelado {
			continue
		}
		s.ComPrazo++
		c.Classificar(agora)
		switch c.SituacaoResp {
		case sla.Cumprido:
			s.RespostaAvaliada++
			s.RespostaCumprida++
		case sla.Estourado, sla.Vencido:
			s.RespostaAvaliada++
		}
		switch c.SituacaoSoluc {
		case sla.Cumprido:
			s.SolucaoAvaliada++
			s.SolucaoCumprida++
		case sla.Estourado, sla.Vencido:
			s.SolucaoAvaliada++
		}
	}
	s.Calcular()
}

func (s *SLA) Calcular() {
	s.RespostaPct = Percentual(s.RespostaCumprida, s.RespostaAvaliada)
	s.SolucaoPct = Percentual(s.SolucaoCumprida, s.SolucaoAvaliada)
}

// Resumo do período.
type Resumo struct {
	Chamados        int64            `json:"chamados"`
	OSConcluidas    int64            `json:"os_concluidas"`
	CustoGarantia   decimal.Decimal  `json:"custo_garantia"`
	Cobrado         decimal.Decimal  `json:"cobrado"`
	Reclamado       decimal.Decimal  `json:"reclamado_fornecedor"`
	Recuperado      decimal.Decimal  `json:"recuperado_fornecedor"`
	CustoLiquido    decimal.Decimal  `json:"custo_liquido"`
	MTTRHoras       *decimal.Decimal `json:"mttr_horas,omitempty"`
	ChamadosComCAPA int64            `json:"chamados_com_capa"`
}

type Painel struct {
	Periodo        Periodo          `json:"periodo"`
	Resumo         Resumo           `json:"resumo"`
	SLA            SLA              `json:"sla"`
	PorItem        []Grupo          `json:"por_item"`
	PorDefeito     []Grupo          `json:"por_defeito"`
	PorLote        []Grupo          `json:"por_lote"`
	Confiabilidade []Confiabilidade `json:"confiabilidade"`
}

// Repository: leitura agregada das tabelas do pós-venda da empresa da sessão.
type Repository interface {
	Resumo(ctx context.Context, p Periodo) (Resumo, error)
	// ChamadosComSLA abertos no período (a classificação é do domínio sla).
	ChamadosComSLA(ctx context.Context, p Periodo) ([]sla.Chamado, error)
	// Custo por dimensão: "item", "defeito" ou "lote".
	CustoPor(ctx context.Context, p Periodo, dimensao string) ([]Grupo, error)
	Confiabilidade(ctx context.Context, p Periodo) ([]Confiabilidade, error)
}
