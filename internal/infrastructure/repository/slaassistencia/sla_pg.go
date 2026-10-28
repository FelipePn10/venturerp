// Package slaassistencia grava as regras de SLA da assistência técnica e os
// prazos dos chamados, sempre da empresa da sessão. O chamado é da empresa pelo
// enterprise_code (convenção do comercial); a regra, pelo enterprise_id.
package slaassistencia

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/posvenda/sla"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type PG struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *PG { return &PG{pool: pool} }

// chamadoDaEmpresa: o chamado pertence à empresa da sessão.
const chamadoDaEmpresa = `c.enterprise_code = (SELECT code FROM enterprise WHERE id = $1)`

func (r *PG) Regras(ctx context.Context) ([]dom.Regra, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT s.id, s.prioridade, s.customer_code, COALESCE(cu.name,''), s.horas_resposta, s.horas_solucao, s.alerta_pct, s.horas_uteis, s.ativo, s.updated_at
		  FROM sla_assistencia s LEFT JOIN customers cu ON cu.code = s.customer_code AND cu.enterprise_id = s.enterprise_id
		 WHERE s.enterprise_id = $1
		 ORDER BY s.customer_code NULLS FIRST, array_position(ARRAY['CRITICAL','HIGH','NORMAL','LOW']::text[], s.prioridade::text)`, empresa)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dom.Regra{}
	for rows.Next() {
		var g dom.Regra
		if err := rows.Scan(&g.ID, &g.Prioridade, &g.CustomerCode, &g.ClienteNome, &g.HorasResposta, &g.HorasSolucao, &g.AlertaPct, &g.HorasUteis, &g.Ativo, &g.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *PG) SalvarRegra(ctx context.Context, g *dom.Regra, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	if g.CustomerCode != nil {
		var ok bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM customers WHERE code=$1 AND enterprise_id=$2)`, *g.CustomerCode, empresa).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return errorsuc.NewNotFoundError(fmt.Sprintf("cliente %d", *g.CustomerCode))
		}
	}
	if g.ID > 0 {
		tag, err := r.pool.Exec(ctx, `UPDATE sla_assistencia SET prioridade=$3, customer_code=$4, horas_resposta=$5, horas_solucao=$6, alerta_pct=$7, ativo=$8,
				updated_by=$9, updated_at=NOW(), horas_uteis=$10
			WHERE enterprise_id=$1 AND id=$2`, empresa, g.ID, g.Prioridade, g.CustomerCode, g.HorasResposta, g.HorasSolucao, g.AlertaPct, g.Ativo, por, g.HorasUteis)
		if err != nil {
			return duplicada(err)
		}
		if tag.RowsAffected() == 0 {
			return errorsuc.NewNotFoundError(fmt.Sprintf("regra de SLA %d", g.ID))
		}
		return nil
	}
	err = r.pool.QueryRow(ctx, `INSERT INTO sla_assistencia (enterprise_id, prioridade, customer_code, horas_resposta, horas_solucao, alerta_pct, ativo, created_by, updated_by, horas_uteis)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8,$9) RETURNING id`, empresa, g.Prioridade, g.CustomerCode, g.HorasResposta, g.HorasSolucao, g.AlertaPct, g.Ativo, por, g.HorasUteis).Scan(&g.ID)
	return duplicada(err)
}

func duplicada(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return errorsuc.NewValidationError("já existe regra de SLA para essa prioridade e esse cliente: altere a existente")
	}
	return err
}

func (r *PG) ExcluirRegra(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM sla_assistencia WHERE enterprise_id=$1 AND id=$2`, empresa, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("regra de SLA %d", id))
	}
	return nil
}

func (r *PG) DadosDoChamado(ctx context.Context, callCode int64) (string, int64, time.Time, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return "", 0, time.Time{}, err
	}
	var prioridade string
	var cliente int64
	var abertura time.Time
	err = r.pool.QueryRow(ctx, `SELECT c.priority, c.customer_code, c.opened_at FROM technical_assistance_calls c WHERE c.code=$2 AND `+chamadoDaEmpresa, empresa, callCode).
		Scan(&prioridade, &cliente, &abertura)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", 0, time.Time{}, errorsuc.NewNotFoundError(fmt.Sprintf("chamado %d", callCode))
	}
	return prioridade, cliente, abertura, err
}

func (r *PG) AplicarPrazo(ctx context.Context, callCode int64, p dom.Prazo) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `UPDATE technical_assistance_calls c SET sla_id=$3, sla_resposta_ate=$4, sla_solucao_ate=$5, sla_alerta_pct=$6,
			sla_resposta_risco_em=$7, sla_solucao_risco_em=$8, sla_horas_uteis=$9, updated_at=NOW()
		WHERE c.code=$2 AND `+chamadoDaEmpresa, empresa, callCode, p.RegraID, p.RespostaAte, p.SolucaoAte, p.AlertaPct, p.RespostaRisco, p.SolucaoRisco, p.HorasUteis)
	return err
}

// fusoDaEmpresa: o mesmo das notificações e do sequenciamento (padrão
// America/Sao_Paulo) — o expediente é no horário da empresa, não do servidor.
func (r *PG) fusoDaEmpresa(ctx context.Context, empresa int64) (*time.Location, error) {
	var tz string
	if err := r.pool.QueryRow(ctx, `SELECT COALESCE((SELECT NULLIF(timezone,'') FROM enterprise_notification_settings WHERE enterprise_id=$1),'America/Sao_Paulo')`, empresa).Scan(&tz); err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("fuso da empresa %q: %w", tz, err)
	}
	return loc, nil
}

func (r *PG) Expediente(ctx context.Context) ([]dom.Intervalo, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT dia_semana, inicio_min, fim_min FROM sla_expediente WHERE enterprise_id=$1 ORDER BY dia_semana, inicio_min`, empresa)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dom.Intervalo{}
	for rows.Next() {
		var x dom.Intervalo
		if err := rows.Scan(&x.DiaSemana, &x.Inicio, &x.Fim); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// SalvarExpediente substitui o expediente inteiro da empresa.
func (r *PG) SalvarExpediente(ctx context.Context, iv []dom.Intervalo, por *uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM sla_expediente WHERE enterprise_id=$1`, empresa); err != nil {
		return err
	}
	for _, x := range iv {
		if _, err := tx.Exec(ctx, `INSERT INTO sla_expediente (enterprise_id, dia_semana, inicio_min, fim_min, updated_by) VALUES ($1,$2,$3,$4,$5)`,
			empresa, x.DiaSemana, x.Inicio, x.Fim, por); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// Calendario: expediente da empresa e folgas do calendário industrial (de um
// ano atrás a dois à frente — o prazo mais longo cabe aí).
func (r *PG) Calendario(ctx context.Context) (*dom.Calendario, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	loc, err := r.fusoDaEmpresa(ctx, empresa)
	if err != nil {
		return nil, err
	}
	iv, err := r.Expediente(ctx)
	if err != nil {
		return nil, err
	}
	ano := time.Now().In(loc).Year()
	rows, err := r.pool.Query(ctx, `SELECT year, month, day FROM industrial_calendar
		WHERE enterprise_id=$1 AND NOT is_workday AND year BETWEEN $2 AND $3`, empresa, ano-1, ano+2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var folgas []time.Time
	for rows.Next() {
		var y, m, d int
		if err := rows.Scan(&y, &m, &d); err != nil {
			return nil, err
		}
		folgas = append(folgas, time.Date(y, time.Month(m), d, 0, 0, 0, 0, loc))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return dom.NovoCalendario(loc, iv, folgas), nil
}

func (r *PG) Pausar(ctx context.Context, callCode int64, em time.Time, motivo string) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE technical_assistance_calls c SET sla_pausado_desde=$3
		WHERE c.code=$2 AND c.sla_solucao_ate IS NOT NULL AND c.sla_pausado_desde IS NULL AND `+chamadoDaEmpresa, empresa, callCode, em)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil // sem SLA ou já parado
	}
	if _, err := tx.Exec(ctx, `INSERT INTO sla_pausas (enterprise_id, call_code, motivo, inicio) VALUES ($1,$2,$3,$4)`, empresa, callCode, motivo, em); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PG) EstadoDoRelogio(ctx context.Context, callCode int64) (*dom.EstadoDoRelogio, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var e dom.EstadoDoRelogio
	err = r.pool.QueryRow(ctx, `SELECT c.sla_pausado_desde, c.sla_horas_uteis, c.respondido_em IS NOT NULL,
			c.sla_resposta_ate, c.sla_resposta_risco_em, c.sla_solucao_ate, c.sla_solucao_risco_em
		  FROM technical_assistance_calls c WHERE c.code=$2 AND `+chamadoDaEmpresa, empresa, callCode).
		Scan(&e.PausadoDesde, &e.HorasUteis, &e.Respondido, &e.RespostaAte, &e.RespostaRisco, &e.SolucaoAte, &e.SolucaoRisco)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("chamado %d", callCode))
	}
	return &e, err
}

func (r *PG) Retomar(ctx context.Context, callCode int64, p dom.Prazo, em time.Time, parado time.Duration) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	horas := decimal.NewFromFloat(parado.Hours()).Round(2)
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE technical_assistance_calls c SET sla_pausado_desde=NULL,
			sla_resposta_ate=$3, sla_resposta_risco_em=$4, sla_solucao_ate=$5, sla_solucao_risco_em=$6,
			sla_horas_pausadas=sla_horas_pausadas+$7, updated_at=NOW()
		WHERE c.code=$2 AND c.sla_pausado_desde IS NOT NULL AND `+chamadoDaEmpresa, empresa, callCode,
		p.RespostaAte, p.RespostaRisco, p.SolucaoAte, p.SolucaoRisco, horas)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, `UPDATE sla_pausas SET fim=$3, horas=$4 WHERE enterprise_id=$1 AND call_code=$2 AND fim IS NULL`, empresa, callCode, em, horas); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PG) MarcarResposta(ctx context.Context, callCode int64, em time.Time) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `UPDATE technical_assistance_calls c SET respondido_em=$3 WHERE c.code=$2 AND c.respondido_em IS NULL AND `+chamadoDaEmpresa, empresa, callCode, em)
	return err
}

func (r *PG) Chamados(ctx context.Context, f dom.Filtro) ([]dom.Chamado, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	abertos := f.Situacao != "TODOS"
	rows, err := r.pool.Query(ctx, `SELECT c.code, c.call_number, c.customer_code, COALESCE(cu.name,''), c.subject, c.priority, c.status, c.opened_at,
			c.sla_resposta_ate, c.respondido_em, c.sla_solucao_ate, COALESCE(c.attended_at, c.closed_at), COALESCE(c.sla_alerta_pct, 80), c.status='CANCELLED',
			c.sla_resposta_risco_em, c.sla_solucao_risco_em, c.sla_horas_uteis, c.sla_pausado_desde, c.sla_horas_pausadas
		  FROM technical_assistance_calls c
		  LEFT JOIN customers cu ON cu.code = c.customer_code AND cu.enterprise_id = $1
		 WHERE `+chamadoDaEmpresa+` AND c.is_active AND c.sla_solucao_ate IS NOT NULL
		   AND (NOT $2 OR c.status NOT IN ('ATTENDED','CLOSED','CANCELLED'))
		   AND ($3::bigint IS NULL OR c.customer_code = $3)
		   AND ($4::timestamptz IS NULL OR c.opened_at >= $4)
		 ORDER BY c.sla_solucao_ate, c.code
		 LIMIT 500`, empresa, abertos, f.Cliente, f.Desde)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dom.Chamado{}
	for rows.Next() {
		var c dom.Chamado
		if err := rows.Scan(&c.CallCode, &c.CallNumber, &c.CustomerCode, &c.ClienteNome, &c.Assunto, &c.Prioridade, &c.Status, &c.AbertoEm,
			&c.RespostaAte, &c.RespondidoEm, &c.SolucaoAte, &c.SolucionadoEm, &c.AlertaPct, &c.Cancelado,
			&c.RespostaRisco, &c.SolucaoRisco, &c.HorasUteis, &c.PausadoDesde, &c.HorasPausadas); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
