// Package indicadorespv lê os indicadores do pós-venda das tabelas de OS,
// chamados, séries e recuperação, sempre da empresa da sessão. O período vem
// do caso de uso como parâmetro (nunca CURRENT_DATE/NOW() comparado com data
// do Go).
package indicadorespv

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	dom "github.com/FelipePn10/panossoerp/internal/domain/posvenda/indicadores"
	"github.com/FelipePn10/panossoerp/internal/domain/posvenda/sla"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type PG struct{ pool *pgxpool.Pool }

// dia: a data do calendário do período como o Go a vê. Colunas DATE comparam
// com ela, nunca com o timestamptz convertido no fuso do banco (que pode ser
// outro: o CI roda o banco em America/Sao_Paulo e o processo em UTC).
func dia(t time.Time) string { return t.Format("2006-01-02") }

func New(pool *pgxpool.Pool) *PG { return &PG{pool: pool} }

// osDoPeriodo: OS concluídas (ou faturadas) no período, com o custo da
// garantia (linhas não cobradas, pelo custo) e o cobrado (linhas cobradas,
// pelo preço). $1 empresa, $2 início, $3 fim.
const osDoPeriodo = `
WITH os AS (
  SELECT o.* FROM ordens_servico o
   WHERE o.enterprise_id = $1 AND o.situacao IN ('CONCLUIDA','FATURADA') AND o.concluida_em >= $2 AND o.concluida_em < $3
), linhas AS (
  SELECT h.os_id, CASE WHEN h.cobrado THEN 0 ELSE h.horas * h.custo_hora END AS garantia, CASE WHEN h.cobrado THEN h.horas * h.preco_hora ELSE 0 END AS cobrado
    FROM os_horas h WHERE h.enterprise_id = $1
  UNION ALL
  SELECT p.os_id, CASE WHEN p.cobrado THEN 0 ELSE p.quantidade * p.custo_unitario END, CASE WHEN p.cobrado THEN p.quantidade * p.preco_unitario ELSE 0 END
    FROM os_pecas p WHERE p.enterprise_id = $1
  UNION ALL
  SELECT d.os_id, CASE WHEN d.cobrado THEN 0 ELSE d.quantidade * d.custo END, CASE WHEN d.cobrado THEN d.quantidade * d.preco ELSE 0 END
    FROM os_despesas d WHERE d.enterprise_id = $1
), custo AS (
  SELECT os.id, COALESCE(SUM(l.garantia),0) AS garantia, COALESCE(SUM(l.cobrado),0) AS cobrado
    FROM os LEFT JOIN linhas l ON l.os_id = os.id GROUP BY os.id
)`

func (r *PG) Resumo(ctx context.Context, p dom.Periodo) (dom.Resumo, error) {
	var s dom.Resumo
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return s, err
	}
	var mttr *decimal.Decimal
	err = r.pool.QueryRow(ctx, osDoPeriodo+`
		SELECT (SELECT COUNT(*) FROM os),
		       (SELECT COALESCE(SUM(garantia),0) FROM custo), (SELECT COALESCE(SUM(cobrado),0) FROM custo),
		       (SELECT ROUND(AVG(EXTRACT(EPOCH FROM os.concluida_em - os.iniciada_em) / 3600)::numeric, 1) FROM os WHERE os.iniciada_em IS NOT NULL)`,
		empresa, p.De, p.Ate).Scan(&s.OSConcluidas, &s.CustoGarantia, &s.Cobrado, &mttr)
	if err != nil {
		return s, fmt.Errorf("custo das OS: %w", err)
	}
	s.MTTRHoras = mttr
	err = r.pool.QueryRow(ctx, `SELECT COUNT(*), COUNT(*) FILTER (WHERE c.capa_id IS NOT NULL)
		  FROM technical_assistance_calls c
		 WHERE c.enterprise_code = (SELECT code FROM enterprise WHERE id = $1) AND c.is_active AND c.opened_at >= $2 AND c.opened_at < $3`,
		empresa, p.De, p.Ate).Scan(&s.Chamados, &s.ChamadosComCAPA)
	if err != nil {
		return s, fmt.Errorf("chamados do período: %w", err)
	}
	err = r.pool.QueryRow(ctx, `SELECT
		  COALESCE(SUM(valor_reclamado) FILTER (WHERE created_at >= $2 AND created_at < $3), 0),
		  COALESCE(SUM(valor_recuperado) FILTER (WHERE situacao = 'RECUPERADA' AND recuperado_em >= $4::date AND recuperado_em < $5::date), 0)
		  FROM recuperacoes_fornecedor WHERE enterprise_id = $1`, empresa, p.De, p.Ate, dia(p.De), dia(p.Ate)).Scan(&s.Reclamado, &s.Recuperado)
	if err != nil {
		return s, fmt.Errorf("recuperação do período: %w", err)
	}
	s.CustoGarantia, s.Cobrado = s.CustoGarantia.Round(2), s.Cobrado.Round(2)
	s.CustoLiquido = s.CustoGarantia.Sub(s.Recuperado).Round(2)
	return s, nil
}

func (r *PG) ChamadosComSLA(ctx context.Context, p dom.Periodo) ([]sla.Chamado, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT c.code, c.call_number, c.customer_code, c.subject, c.priority, c.status, c.opened_at,
			c.sla_resposta_ate, c.respondido_em, c.sla_solucao_ate, COALESCE(c.attended_at, c.closed_at), COALESCE(c.sla_alerta_pct, 80), c.status = 'CANCELLED'
		  FROM technical_assistance_calls c
		 WHERE c.enterprise_code = (SELECT code FROM enterprise WHERE id = $1) AND c.is_active AND c.sla_solucao_ate IS NOT NULL
		   AND c.opened_at >= $2 AND c.opened_at < $3`, empresa, p.De, p.Ate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []sla.Chamado{}
	for rows.Next() {
		var c sla.Chamado
		if err := rows.Scan(&c.CallCode, &c.CallNumber, &c.CustomerCode, &c.Assunto, &c.Prioridade, &c.Status, &c.AbertoEm,
			&c.RespostaAte, &c.RespondidoEm, &c.SolucaoAte, &c.SolucionadoEm, &c.AlertaPct, &c.Cancelado); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

var dimensoes = map[string]string{
	"item": `SELECT COALESCE(i.business_code, o.item_code::text, ''), COALESCE(i.name, 'Sem item'), COUNT(*), SUM(cu.garantia), SUM(cu.cobrado)
	   FROM os o JOIN custo cu ON cu.id = o.id LEFT JOIN items i ON i.code = o.item_code AND i.enterprise_id = o.enterprise_id
	  GROUP BY 1, 2`,
	"defeito": `SELECT COALESCE(dr.code::text, ''), COALESCE(dr.description, 'Sem motivo de defeito informado'), COUNT(*), SUM(cu.garantia), SUM(cu.cobrado)
	   FROM os o JOIN custo cu ON cu.id = o.id
	   LEFT JOIN technical_assistance_call_items ci ON ci.code = o.call_item_code AND ci.call_code = o.call_code
	   LEFT JOIN technical_assistance_defect_reasons dr ON dr.code = ci.defect_reason_code
	  GROUP BY 1, 2`,
	"lote": `SELECT COALESCE(s.lote, ''), COALESCE(NULLIF(s.lote, ''), 'Sem lote'), COUNT(*), SUM(cu.garantia), SUM(cu.cobrado)
	   FROM os o JOIN custo cu ON cu.id = o.id LEFT JOIN numeros_serie s ON s.id = o.serie_id AND s.enterprise_id = o.enterprise_id
	  GROUP BY 1, 2`,
}

func (r *PG) CustoPor(ctx context.Context, p dom.Periodo, dimensao string) ([]dom.Grupo, error) {
	consulta, ok := dimensoes[dimensao]
	if !ok {
		return nil, fmt.Errorf("dimensão de custo desconhecida: %s", dimensao)
	}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, osDoPeriodo+` SELECT * FROM (`+consulta+`) g ORDER BY 4 DESC, 2 LIMIT 50`, empresa, p.De, p.Ate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dom.Grupo{}
	for rows.Next() {
		var g dom.Grupo
		if err := rows.Scan(&g.Chave, &g.Descricao, &g.OS, &g.CustoGarantia, &g.Cobrado); err != nil {
			return nil, err
		}
		g.CustoGarantia, g.Cobrado = g.CustoGarantia.Round(2), g.Cobrado.Round(2)
		out = append(out, g)
	}
	return out, rows.Err()
}

// Confiabilidade por item com série: em campo = no cliente ou em assistência;
// dias em campo contam da instalação (ou da venda) até o último dia do
// período (ou a baixa); falha = item de chamado com a série.
func (r *PG) Confiabilidade(ctx context.Context, p dom.Periodo) ([]dom.Confiabilidade, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		WITH series AS (
		  SELECT s.id, s.item_code, s.situacao,
		         GREATEST(0, LEAST($4::date - 1, COALESCE((SELECT MIN(e.ocorrido_em)::date FROM numeros_serie_eventos e
		                                                   WHERE e.serie_id = s.id AND e.enterprise_id = s.enterprise_id AND e.evento = 'BAIXA'), $4::date - 1))
		                     - COALESCE(s.data_instalacao, s.data_venda)) AS dias
		    FROM numeros_serie s
		   WHERE s.enterprise_id = $1 AND COALESCE(s.data_instalacao, s.data_venda) IS NOT NULL AND COALESCE(s.data_instalacao, s.data_venda) < $4::date
		), falhas AS (
		  SELECT ci.serie_id, COUNT(*) AS total, COUNT(*) FILTER (WHERE c.opened_at >= $2 AND c.opened_at < $3) AS no_periodo
		    FROM technical_assistance_call_items ci
		    JOIN technical_assistance_calls c ON c.code = ci.call_code AND c.is_active AND c.status <> 'CANCELLED'
		   WHERE ci.serie_id IS NOT NULL AND c.enterprise_code = (SELECT code FROM enterprise WHERE id = $1)
		   GROUP BY ci.serie_id
		)
		SELECT s.item_code, COALESCE(i.name, ''),
		       COUNT(*) FILTER (WHERE s.situacao IN ('NO_CLIENTE','EM_ASSISTENCIA')),
		       COUNT(*) FILTER (WHERE COALESCE(f.no_periodo, 0) > 0),
		       COALESCE(SUM(f.total), 0), COALESCE(SUM(s.dias), 0)
		  FROM series s
		  LEFT JOIN falhas f ON f.serie_id = s.id
		  LEFT JOIN items i ON i.code = s.item_code AND i.enterprise_id = $1
		 GROUP BY s.item_code, i.name
		 ORDER BY 4 DESC, 5 DESC, 2
		 LIMIT 50`, empresa, p.De, p.Ate, dia(p.Ate))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dom.Confiabilidade{}
	for rows.Next() {
		var c dom.Confiabilidade
		if err := rows.Scan(&c.ItemCode, &c.ItemNome, &c.EmCampo, &c.Falharam, &c.FalhasTotais, &c.DiasEmCampo); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
