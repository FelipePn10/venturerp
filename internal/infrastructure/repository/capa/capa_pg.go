// Package capa grava CAPA/8D da empresa da sessão: cabeçalho, NCs ligadas,
// equipe, ações e histórico imutável.
package capa

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	dom "github.com/FelipePn10/panossoerp/internal/domain/quality/capa"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type PG struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *PG { return &PG{pool: pool} }

var _ dom.Repository = (*PG)(nil)

const colunas = `id, numero, metodo, tipo, origem, titulo, item_code, supplier_code, customer_code, referencia, responsavel, prazo,
	etapa, status, problema, contencao_dispensada, metodo_causa, cinco_porques, ishikawa, causa_raiz, ponto_de_escape, prevencao,
	eficaz, verificacao, licoes, ciclo, created_at, created_by, updated_at, closed_at, closed_by`

func scan(row pgx.Row) (*dom.Capa, error) {
	var c dom.Capa
	var metodo, status string
	var problema, porques, ishikawa []byte
	if err := row.Scan(&c.ID, &c.Numero, &metodo, &c.Tipo, &c.Origem, &c.Titulo, &c.ItemCode, &c.SupplierCode, &c.CustomerCode, &c.Referencia,
		&c.Responsavel, &c.Prazo, &c.Etapa, &status, &problema, &c.ContencaoDispensada, &c.MetodoCausa, &porques, &ishikawa, &c.CausaRaiz,
		&c.PontoDeEscape, &c.Prevencao, &c.Eficaz, &c.Verificacao, &c.Licoes, &c.Ciclo, &c.CreatedAt, &c.CreatedBy, &c.UpdatedAt,
		&c.ClosedAt, &c.ClosedBy); err != nil {
		return nil, err
	}
	c.Metodo, c.Status = dom.Metodo(metodo), dom.Status(status)
	_ = json.Unmarshal(problema, &c.Problema)
	_ = json.Unmarshal(porques, &c.CincoPorques)
	_ = json.Unmarshal(ishikawa, &c.Ishikawa)
	if c.CincoPorques == nil {
		c.CincoPorques = []string{}
	}
	if c.Ishikawa == nil {
		c.Ishikawa = map[string][]string{}
	}
	c.NCs, c.Equipe, c.Acoes = []int64{}, []dom.Membro{}, []dom.Acao{}
	return &c, nil
}

// ncsDaEmpresa confere que as NCs são da empresa (a NC não tem coluna de
// empresa: é a do item).
func ncsDaEmpresa(ctx context.Context, tx pgx.Tx, empresa int64, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	var n int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM non_conformances n JOIN items it ON it.code = n.item_code AND it.enterprise_id = $1
		 WHERE n.id = ANY($2)`, empresa, ids).Scan(&n); err != nil {
		return err
	}
	if n != len(ids) {
		return errorsuc.NewValidationError("não conformidade não encontrada nesta empresa")
	}
	return nil
}

func evento(ctx context.Context, tx pgx.Tx, empresa, capa int64, ev *dom.Evento, ator uuid.UUID) error {
	if ev == nil {
		return nil
	}
	var por *uuid.UUID
	if ator != uuid.Nil {
		por = &ator
	}
	_, err := tx.Exec(ctx, `INSERT INTO quality_capa_eventos (capa_id, enterprise_id, evento, de, para, detalhe, ator) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		capa, empresa, ev.Evento, ev.De, ev.Para, ev.Detalhe, por)
	return err
}

func (r *PG) Abrir(ctx context.Context, c *dom.Capa, ator uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := ncsDaEmpresa(ctx, tx, empresa, c.NCs); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('quality_capa:' || $1::bigint, 0))`, empresa); err != nil {
		return err
	}
	problema, _ := json.Marshal(c.Problema)
	porques, _ := json.Marshal(c.CincoPorques)
	ishikawa, _ := json.Marshal(c.Ishikawa)
	var por *uuid.UUID
	if ator != uuid.Nil {
		por = &ator
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO quality_capas (enterprise_id, numero, metodo, tipo, origem, titulo, item_code, supplier_code, customer_code, referencia,
			responsavel, prazo, etapa, status, problema, cinco_porques, ishikawa, ciclo, created_by)
		VALUES ($1, COALESCE((SELECT MAX(numero) FROM quality_capas WHERE enterprise_id = $1), 0) + 1, $2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		RETURNING id, numero, created_at, updated_at`,
		empresa, string(c.Metodo), c.Tipo, c.Origem, c.Titulo, c.ItemCode, c.SupplierCode, c.CustomerCode, c.Referencia,
		c.Responsavel, c.Prazo, c.Etapa, string(c.Status), problema, porques, ishikawa, c.Ciclo, por).Scan(&c.ID, &c.Numero, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("abrindo a CAPA: %w", err)
	}
	for _, nc := range c.NCs {
		if _, err := tx.Exec(ctx, `INSERT INTO quality_capa_ncs (capa_id, enterprise_id, nc_id) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, c.ID, empresa, nc); err != nil {
			return err
		}
	}
	if err := gravarEquipe(ctx, tx, empresa, c); err != nil {
		return err
	}
	para := c.Etapa
	if err := evento(ctx, tx, empresa, c.ID, &dom.Evento{Evento: "ABERTA", Para: &para}, ator); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func gravarEquipe(ctx context.Context, tx pgx.Tx, empresa int64, c *dom.Capa) error {
	if _, err := tx.Exec(ctx, `DELETE FROM quality_capa_equipe WHERE capa_id = $1 AND enterprise_id = $2`, c.ID, empresa); err != nil {
		return err
	}
	for _, m := range c.Equipe {
		if _, err := tx.Exec(ctx, `INSERT INTO quality_capa_equipe (capa_id, enterprise_id, user_id, papel) VALUES ($1,$2,$3,$4)`, c.ID, empresa, m.UserID, m.Papel); err != nil {
			return fmt.Errorf("gravando a equipe: %w", err)
		}
	}
	return nil
}

func (r *PG) carregar(ctx context.Context, tx pgx.Tx, empresa, id int64) (*dom.Capa, error) {
	var row pgx.Row
	sql := `SELECT ` + colunas + ` FROM quality_capas WHERE enterprise_id = $1 AND id = $2`
	if tx != nil {
		row = tx.QueryRow(ctx, sql+` FOR UPDATE`, empresa, id)
	} else {
		row = r.pool.QueryRow(ctx, sql, empresa, id)
	}
	c, err := scan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("CAPA %d", id))
	}
	if err != nil {
		return nil, fmt.Errorf("lendo a CAPA: %w", err)
	}
	var q interface {
		Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	} = r.pool
	if tx != nil {
		q = tx
	}
	rows, err := q.Query(ctx, `SELECT nc_id FROM quality_capa_ncs WHERE capa_id = $1 AND enterprise_id = $2 ORDER BY nc_id`, id, empresa)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var n int64
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return nil, err
		}
		c.NCs = append(c.NCs, n)
	}
	rows.Close()
	rows, err = q.Query(ctx, `SELECT e.user_id, COALESCE(u.name, ''), e.papel FROM quality_capa_equipe e LEFT JOIN users u ON u.id = e.user_id
		 WHERE e.capa_id = $1 AND e.enterprise_id = $2 ORDER BY e.papel, u.name`, id, empresa)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var m dom.Membro
		if err := rows.Scan(&m.UserID, &m.Nome, &m.Papel); err != nil {
			rows.Close()
			return nil, err
		}
		c.Equipe = append(c.Equipe, m)
	}
	rows.Close()
	rows, err = q.Query(ctx, `SELECT id, tipo, descricao, responsavel, prazo, status, evidencia, concluida_em, concluida_por, ciclo, created_at
		  FROM quality_capa_acoes WHERE capa_id = $1 AND enterprise_id = $2 ORDER BY ciclo, tipo, prazo, id`, id, empresa)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var a dom.Acao
		if err := rows.Scan(&a.ID, &a.Tipo, &a.Descricao, &a.Responsavel, &a.Prazo, &a.Status, &a.Evidencia, &a.ConcluidaEm, &a.ConcluidaPor, &a.Ciclo, &a.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		c.Acoes = append(c.Acoes, a)
	}
	rows.Close()
	if tx == nil {
		rows, err = q.Query(ctx, `SELECT id, evento, de, para, detalhe, ator, created_at FROM quality_capa_eventos
			 WHERE capa_id = $1 AND enterprise_id = $2 ORDER BY created_at, id`, id, empresa)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var ev dom.Evento
			if err := rows.Scan(&ev.ID, &ev.Evento, &ev.De, &ev.Para, &ev.Detalhe, &ev.Ator, &ev.CreatedAt); err != nil {
				rows.Close()
				return nil, err
			}
			c.Eventos = append(c.Eventos, ev)
		}
		rows.Close()
	}
	return c, rows.Err()
}

func (r *PG) Obter(ctx context.Context, id int64) (*dom.Capa, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	return r.carregar(ctx, nil, empresa, id)
}

func (r *PG) Listar(ctx context.Context, f dom.Filtro) ([]dom.Capa, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if f.Hoje.IsZero() {
		f.Hoje = time.Now()
	}
	rows, err := r.pool.Query(ctx, `SELECT `+colunas+` FROM quality_capas c
		 WHERE enterprise_id = $1 AND ($2 = '' OR status = $2) AND ($3::uuid IS NULL OR responsavel = $3
		        OR EXISTS (SELECT 1 FROM quality_capa_equipe e WHERE e.capa_id = c.id AND e.user_id = $3))
		   AND (NOT $4 OR (status = 'ABERTA' AND (prazo < $5::date OR EXISTS (SELECT 1 FROM quality_capa_acoes a WHERE a.capa_id = c.id
		        AND a.status IN ('PENDENTE','EM_ANDAMENTO') AND a.prazo < $5::date))))
		 ORDER BY status = 'ABERTA' DESC, prazo NULLS LAST, numero DESC LIMIT 300`,
		empresa, f.Status, f.Responsavel, f.Atrasadas, f.Hoje)
	if err != nil {
		return nil, fmt.Errorf("listando as CAPAs: %w", err)
	}
	defer rows.Close()
	out := []dom.Capa{}
	for rows.Next() {
		c, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *PG) Atualizar(ctx context.Context, id int64, ator uuid.UUID, fn dom.Mudanca) (*dom.Capa, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	c, err := r.carregar(ctx, tx, empresa, id)
	if err != nil {
		return nil, err
	}
	ev, err := fn(c)
	if err != nil {
		return nil, err
	}
	problema, _ := json.Marshal(c.Problema)
	porques, _ := json.Marshal(c.CincoPorques)
	ishikawa, _ := json.Marshal(c.Ishikawa)
	if _, err := tx.Exec(ctx, `UPDATE quality_capas SET titulo=$3, responsavel=$4, prazo=$5, etapa=$6, status=$7, problema=$8,
			contencao_dispensada=$9, metodo_causa=$10, cinco_porques=$11, ishikawa=$12, causa_raiz=$13, ponto_de_escape=$14, prevencao=$15,
			eficaz=$16, verificacao=$17, licoes=$18, ciclo=$19, closed_at=$20, closed_by=$21, updated_at=NOW()
		 WHERE id = $1 AND enterprise_id = $2`,
		id, empresa, c.Titulo, c.Responsavel, c.Prazo, c.Etapa, string(c.Status), problema, c.ContencaoDispensada, c.MetodoCausa, porques,
		ishikawa, c.CausaRaiz, c.PontoDeEscape, c.Prevencao, c.Eficaz, c.Verificacao, c.Licoes, c.Ciclo, c.ClosedAt, c.ClosedBy); err != nil {
		return nil, fmt.Errorf("gravando a CAPA: %w", err)
	}
	if err := gravarEquipe(ctx, tx, empresa, c); err != nil {
		return nil, err
	}
	if err := evento(ctx, tx, empresa, id, ev, ator); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.carregar(ctx, nil, empresa, id)
}

func (r *PG) AdicionarAcao(ctx context.Context, capaID int64, a *dom.Acao, ator uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	c, err := r.carregar(ctx, tx, empresa, capaID)
	if err != nil {
		return err
	}
	if c.Status != dom.Aberta {
		return errorsuc.NewValidationError("a CAPA não está aberta")
	}
	a.Ciclo, a.Status = c.Ciclo, "PENDENTE"
	var por *uuid.UUID
	if ator != uuid.Nil {
		por = &ator
	}
	if err := tx.QueryRow(ctx, `INSERT INTO quality_capa_acoes (capa_id, enterprise_id, tipo, descricao, responsavel, prazo, ciclo, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id, created_at`, capaID, empresa, a.Tipo, a.Descricao, a.Responsavel, a.Prazo, a.Ciclo, por).
		Scan(&a.ID, &a.CreatedAt); err != nil {
		return fmt.Errorf("gravando a ação: %w", err)
	}
	d := fmt.Sprintf("%s: %s", a.Tipo, a.Descricao)
	if err := evento(ctx, tx, empresa, capaID, &dom.Evento{Evento: "ACAO_INCLUIDA", Detalhe: &d}, ator); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PG) AtualizarAcao(ctx context.Context, capaID, acaoID int64, ator uuid.UUID, fn func(c *dom.Capa, a *dom.Acao) (*dom.Evento, error)) (*dom.Capa, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	c, err := r.carregar(ctx, tx, empresa, capaID)
	if err != nil {
		return nil, err
	}
	var alvo *dom.Acao
	for i := range c.Acoes {
		if c.Acoes[i].ID == acaoID {
			alvo = &c.Acoes[i]
		}
	}
	if alvo == nil {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("ação %d da CAPA %d", acaoID, c.Numero))
	}
	ev, err := fn(c, alvo)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE quality_capa_acoes SET descricao=$3, responsavel=$4, prazo=$5, status=$6, evidencia=$7, concluida_em=$8, concluida_por=$9
		 WHERE id = $1 AND enterprise_id = $2`, acaoID, empresa, alvo.Descricao, alvo.Responsavel, alvo.Prazo, alvo.Status, alvo.Evidencia,
		alvo.ConcluidaEm, alvo.ConcluidaPor); err != nil {
		return nil, fmt.Errorf("gravando a ação: %w", err)
	}
	if err := evento(ctx, tx, empresa, capaID, ev, ator); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.carregar(ctx, nil, empresa, capaID)
}

func (r *PG) VincularNC(ctx context.Context, capaID, ncID int64, ator uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	c, err := r.carregar(ctx, tx, empresa, capaID)
	if err != nil {
		return err
	}
	if c.Status != dom.Aberta {
		return errorsuc.NewValidationError("a CAPA não está aberta")
	}
	if err := ncsDaEmpresa(ctx, tx, empresa, []int64{ncID}); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO quality_capa_ncs (capa_id, enterprise_id, nc_id) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, capaID, empresa, ncID); err != nil {
		return err
	}
	d := fmt.Sprintf("NC %d", ncID)
	if err := evento(ctx, tx, empresa, capaID, &dom.Evento{Evento: "NC_VINCULADA", Detalhe: &d}, ator); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PG) AcoesPendentes(ctx context.Context, responsavel *uuid.UUID) ([]dom.AcaoPendente, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT a.id, a.tipo, a.descricao, a.responsavel, a.prazo, a.status, a.evidencia, a.concluida_em, a.concluida_por,
			a.ciclo, a.created_at, c.id, c.numero, c.titulo, a.prazo < CURRENT_DATE
		  FROM quality_capa_acoes a JOIN quality_capas c ON c.id = a.capa_id AND c.enterprise_id = a.enterprise_id
		 WHERE a.enterprise_id = $1 AND c.status = 'ABERTA' AND a.status IN ('PENDENTE', 'EM_ANDAMENTO') AND ($2::uuid IS NULL OR a.responsavel = $2)
		 ORDER BY a.prazo, c.numero LIMIT 500`, empresa, responsavel)
	if err != nil {
		return nil, fmt.Errorf("listando as ações pendentes: %w", err)
	}
	defer rows.Close()
	out := []dom.AcaoPendente{}
	for rows.Next() {
		var p dom.AcaoPendente
		if err := rows.Scan(&p.ID, &p.Tipo, &p.Descricao, &p.Responsavel, &p.Prazo, &p.Status, &p.Evidencia, &p.ConcluidaEm, &p.ConcluidaPor,
			&p.Ciclo, &p.CreatedAt, &p.CapaID, &p.CapaNumero, &p.CapaTitulo, &p.Atrasada); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PG) NCs(ctx context.Context, f dom.FiltroNC) ([]dom.NC, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if f.Limite <= 0 || f.Limite > 500 {
		f.Limite = 300
	}
	rows, err := r.pool.Query(ctx, `
		SELECT n.id, n.code, n.item_code, n.lot, n.lot_id, l.numero, l.supplier_code, n.production_order_id, n.nonconform_qty::text,
		       n.description, n.severity::text, n.disposition::text, n.is_open, n.created_at,
		       COALESCE(ARRAY_AGG(c.id ORDER BY c.numero) FILTER (WHERE c.id IS NOT NULL), '{}'),
		       COALESCE(ARRAY_AGG(c.numero ORDER BY c.numero) FILTER (WHERE c.id IS NOT NULL), '{}')
		  FROM non_conformances n
		  JOIN items it ON it.code = n.item_code AND it.enterprise_id = $1
		  LEFT JOIN quality_inspection_lots l ON l.id = n.lot_id AND l.enterprise_id = $1
		  LEFT JOIN quality_capa_ncs cn ON cn.nc_id = n.id AND cn.enterprise_id = $1
		  LEFT JOIN quality_capas c ON c.id = cn.capa_id AND c.enterprise_id = $1 AND c.status <> 'CANCELADA'
		 WHERE (NOT $2 OR n.is_open) AND ($3 = 0 OR n.item_code = $3) AND ($4 = 0 OR l.supplier_code = $4)
		 GROUP BY n.id, l.numero, l.supplier_code
		HAVING NOT $5 OR COUNT(c.id) = 0
		 ORDER BY n.is_open DESC, n.created_at DESC LIMIT $6`,
		empresa, f.SoAbertas, f.ItemCode, f.SupplierCode, f.SemCapa, f.Limite)
	if err != nil {
		return nil, fmt.Errorf("listando as não conformidades: %w", err)
	}
	defer rows.Close()
	out := []dom.NC{}
	for rows.Next() {
		var n dom.NC
		if err := rows.Scan(&n.ID, &n.Code, &n.ItemCode, &n.Lot, &n.LotID, &n.LoteNumero, &n.SupplierCode, &n.ProductionOrderID, &n.Quantidade,
			&n.Descricao, &n.Severidade, &n.Disposicao, &n.Aberta, &n.CreatedAt, &n.Capas, &n.CapaNumeros); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *PG) UsuariosDaEmpresa(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT u.id, u.name FROM users u JOIN user_enterprises ue ON ue.user_id = u.id AND ue.enterprise_id = $1
		 WHERE u.id = ANY($2) AND u.is_active`, empresa, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[uuid.UUID]string{}
	for rows.Next() {
		var id uuid.UUID
		var nome string
		if err := rows.Scan(&id, &nome); err != nil {
			return nil, err
		}
		out[id] = nome
	}
	return out, rows.Err()
}

func (r *PG) Usuarios(ctx context.Context) ([]dom.Membro, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT u.id, u.name FROM users u JOIN user_enterprises ue ON ue.user_id = u.id AND ue.enterprise_id = $1
		 WHERE u.is_active ORDER BY u.name`, empresa)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dom.Membro{}
	for rows.Next() {
		var m dom.Membro
		if err := rows.Scan(&m.UserID, &m.Nome); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
