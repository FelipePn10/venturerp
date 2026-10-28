package cost_center

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/cost_center/gestao"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// GestaoPG: gestão, orçamento e aprovação de requisição do centro de custo.
type GestaoPG struct {
	pool *pgxpool.Pool
}

func NewGestaoPG(pool *pgxpool.Pool) *GestaoPG { return &GestaoPG{pool: pool} }

var _ gestao.Repository = (*GestaoPG)(nil)

const colunasGestao = `c.id, c.code, g.codigo, c.description, c.parent_code, c.type::text, c.is_ratio, c.start_date, c.end_date, c.is_active,
	g.responsavel_id, g.aprovador_id, g.aprova_requisicoes, g.conta_padrao_id,
	COALESCE(ur.name, ''), COALESCE(ua.name, '')`

const fromGestao = ` FROM cost_centers c
	JOIN cost_center_gestao g ON g.cost_center_id = c.id
	LEFT JOIN users ur ON ur.id = g.responsavel_id
	LEFT JOIN users ua ON ua.id = g.aprovador_id`

func scanCentroGestao(row pgx.Row) (gestao.Centro, error) {
	var c gestao.Centro
	err := row.Scan(&c.ID, &c.Code, &c.Codigo, &c.Descricao, &c.ParentCode, &c.Tipo, &c.IsRatio, &c.StartDate, &c.EndDate, &c.IsActive,
		&c.Gestao.ResponsavelID, &c.Gestao.AprovadorID, &c.Gestao.AprovaRequisicoes, &c.Gestao.ContaPadraoID, &c.Responsavel, &c.Aprovador)
	c.Gestao.CostCenterID, c.Gestao.Codigo = c.ID, c.Codigo
	return c, err
}

func (r *GestaoPG) Centros(ctx context.Context) ([]gestao.Centro, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+colunasGestao+fromGestao+` WHERE c.enterprise_id = $1 ORDER BY g.codigo`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listando os centros de custo: %w", err)
	}
	defer rows.Close()
	var out []gestao.Centro
	for rows.Next() {
		c, err := scanCentroGestao(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *GestaoPG) Centro(ctx context.Context, code int32) (*gestao.Centro, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	c, err := scanCentroGestao(r.pool.QueryRow(ctx, `SELECT `+colunasGestao+fromGestao+` WHERE c.enterprise_id = $1 AND c.code = $2`, empresa, code))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("centro de custo %d não encontrado", code))
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// SalvarGestao: o responsável e o aprovador têm de ser usuários da empresa, e
// a conta padrão, uma conta do plano da empresa.
func (r *GestaoPG) SalvarGestao(ctx context.Context, g gestao.Gestao, por uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	for _, u := range []*uuid.UUID{g.ResponsavelID, g.AprovadorID} {
		if u == nil {
			continue
		}
		var ok bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM user_enterprises WHERE user_id = $1 AND enterprise_id = $2)`, *u, empresa).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return errorsuc.NewValidationError("o responsável e o aprovador têm de ser usuários desta empresa")
		}
	}
	if g.ContaPadraoID != nil {
		var ok bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM plano_contas WHERE id = $1 AND enterprise_id = $2 AND is_active)`, *g.ContaPadraoID, empresa).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return errorsuc.NewValidationError("a conta padrão não existe no plano de contas desta empresa ou está inativa")
		}
	}
	tag, err := r.pool.Exec(ctx, `UPDATE cost_center_gestao g SET codigo = $3, responsavel_id = $4, aprovador_id = $5, aprova_requisicoes = $6,
			conta_padrao_id = $7, updated_by = $8, updated_at = NOW()
		  WHERE g.cost_center_id = $1 AND g.enterprise_id = $2`,
		g.CostCenterID, empresa, g.Codigo, g.ResponsavelID, g.AprovadorID, g.AprovaRequisicoes, g.ContaPadraoID, por)
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return errorsuc.NewConflictError(fmt.Sprintf("já existe centro de custo com o código %s nesta empresa", g.Codigo))
		}
		return fmt.Errorf("gravando a gestão do centro de custo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError("centro de custo não encontrado")
	}
	return nil
}

func (r *GestaoPG) Orcamentos(ctx context.Context, ano int) ([]gestao.Orcamento, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT cost_center_id, ano, mes, valor FROM cost_center_orcamentos WHERE enterprise_id = $1 AND ano = $2 ORDER BY cost_center_id, mes`, empresa, ano)
	if err != nil {
		return nil, fmt.Errorf("lendo o orçamento: %w", err)
	}
	defer rows.Close()
	var out []gestao.Orcamento
	for rows.Next() {
		var o gestao.Orcamento
		var a, m int16
		if err := rows.Scan(&o.CostCenterID, &a, &m, &o.Valor); err != nil {
			return nil, err
		}
		o.Ano, o.Mes = int(a), int(m)
		out = append(out, o)
	}
	return out, rows.Err()
}

// SalvarOrcamentos grava (ou zera) os meses numa transação; o centro tem de ser
// da empresa.
func (r *GestaoPG) SalvarOrcamentos(ctx context.Context, os []gestao.Orcamento, por uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, o := range os {
		tag, err := tx.Exec(ctx, `INSERT INTO cost_center_orcamentos (enterprise_id, cost_center_id, ano, mes, valor, updated_by)
			SELECT $1, c.id, $3, $4, $5, $6 FROM cost_centers c WHERE c.id = $2 AND c.enterprise_id = $1
			ON CONFLICT (cost_center_id, ano, mes) DO UPDATE SET valor = EXCLUDED.valor, updated_by = EXCLUDED.updated_by, updated_at = NOW()`,
			empresa, o.CostCenterID, o.Ano, o.Mes, o.Valor.Round(2), por)
		if err != nil {
			return fmt.Errorf("gravando o orçamento: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return errorsuc.NewValidationError(fmt.Sprintf("o centro de custo %d não é desta empresa", o.CostCenterID))
		}
	}
	return tx.Commit(ctx)
}

func (r *GestaoPG) Realizado(ctx context.Context, ano, mes int) (map[int64]decimal.Decimal, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT x.centro, SUM(x.valor) FROM (
			SELECT r.centro_custo_id AS centro, cp.data_emissao AS dia, r.valor
			  FROM contas_pagar_rateios r JOIN contas_pagar cp ON cp.id = r.conta_pagar_id
			 WHERE cp.enterprise_id = $1 AND cp.is_active AND cp.status <> 'CANCELADO' AND r.centro_custo_id IS NOT NULL
			UNION ALL
			SELECT cp.centro_custo_id, cp.data_emissao, cp.valor_bruto
			  FROM contas_pagar cp
			 WHERE cp.enterprise_id = $1 AND cp.is_active AND cp.status <> 'CANCELADO' AND cp.centro_custo_id IS NOT NULL
			   AND NOT EXISTS (SELECT 1 FROM contas_pagar_rateios r WHERE r.conta_pagar_id = cp.id)
		) x
		WHERE EXTRACT(YEAR FROM x.dia) = $2 AND ($3 = 0 OR EXTRACT(MONTH FROM x.dia) = $3)
		GROUP BY x.centro`, empresa, ano, mes)
	if err != nil {
		return nil, fmt.Errorf("somando o realizado por centro: %w", err)
	}
	defer rows.Close()
	out := map[int64]decimal.Decimal{}
	for rows.Next() {
		var id int64
		var v decimal.Decimal
		if err := rows.Scan(&id, &v); err != nil {
			return nil, err
		}
		out[id] = v
	}
	return out, rows.Err()
}

// AbrirAprovacoes: para cada centro da requisição que aprova requisições, uma
// aprovação pendente. A já decidida (aprovada ou reprovada) volta a pendente
// quando a requisição ganha item novo do centro depois da decisão — é assim
// que a reprovada, corrigida, volta para o aprovador.
func (r *GestaoPG) AbrirAprovacoes(ctx context.Context, requisitionCode int64) error {
	return abrirAprovacoes(ctx, r.pool, requisitionCode)
}

// AprovacoesNaTransacao: o mesmo AbrirAprovacoes dentro de uma transação já
// aberta (a firmação da ordem planejada grava a requisição na transação dela).
func AprovacoesNaTransacao(tx pgx.Tx) interface {
	AbrirAprovacoes(ctx context.Context, requisitionCode int64) error
} {
	return aprovacoesTx{tx: tx}
}

type aprovacoesTx struct{ tx pgx.Tx }

func (a aprovacoesTx) AbrirAprovacoes(ctx context.Context, requisitionCode int64) error {
	return abrirAprovacoes(ctx, a.tx, requisitionCode)
}

type executor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func abrirAprovacoes(ctx context.Context, db executor, requisitionCode int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	codigoEmpresa, err := tenant.Code(ctx)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `
		INSERT INTO requisicao_aprovacoes_centro (enterprise_id, requisition_code, cost_center_id, aprovador_id)
		SELECT DISTINCT $1::bigint, pri.requisition_code, c.id, g.aprovador_id
		  FROM purchase_requisition_items pri
		  JOIN purchase_requisitions pr ON pr.code = pri.requisition_code AND pr.enterprise_code = $2
		  JOIN cost_centers c ON c.code = pri.cost_center_code AND c.enterprise_id = $1
		  JOIN cost_center_gestao g ON g.cost_center_id = c.id AND g.aprova_requisicoes AND g.aprovador_id IS NOT NULL
		 WHERE pri.requisition_code = $3 AND pri.is_active
		ON CONFLICT (requisition_code, cost_center_id) DO UPDATE
		   SET status = 'PENDENTE', aprovador_id = EXCLUDED.aprovador_id, decidido_por = NULL, decidido_em = NULL, justificativa = NULL, criado_em = NOW()
		 WHERE requisicao_aprovacoes_centro.status IN ('APROVADA', 'REPROVADA')
		   AND EXISTS (SELECT 1 FROM purchase_requisition_items i JOIN cost_centers cc ON cc.code = i.cost_center_code
		                WHERE i.requisition_code = EXCLUDED.requisition_code AND cc.id = EXCLUDED.cost_center_id
		                  AND i.is_active AND i.created_at > requisicao_aprovacoes_centro.decidido_em)`,
		empresa, codigoEmpresa, requisitionCode)
	if err != nil {
		return fmt.Errorf("abrindo a aprovação do centro de custo: %w", err)
	}
	return nil
}

const colunasAprovacao = `a.id, a.requisition_code, a.cost_center_id, g.codigo, c.description, a.status, a.aprovador_id, pr.created_by,
	a.decidido_por, a.decidido_em, a.justificativa, a.criado_em`

const fromAprovacao = ` FROM requisicao_aprovacoes_centro a
	JOIN cost_centers c ON c.id = a.cost_center_id
	JOIN cost_center_gestao g ON g.cost_center_id = c.id
	LEFT JOIN purchase_requisitions pr ON pr.code = a.requisition_code`

func scanAprovacao(row pgx.Row) (gestao.AprovacaoRequisicao, error) {
	var a gestao.AprovacaoRequisicao
	err := row.Scan(&a.ID, &a.RequisitionCode, &a.CostCenterID, &a.CentroCodigo, &a.CentroDescricao, &a.Status, &a.AprovadorID, &a.Solicitante,
		&a.DecididoPor, &a.DecididoEm, &a.Justificativa, &a.CriadoEm)
	return a, err
}

func (r *GestaoPG) Aprovacoes(ctx context.Context, status string, aprovador *uuid.UUID, requisitionCode *int64) ([]gestao.AprovacaoRequisicao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+colunasAprovacao+fromAprovacao+`
		 WHERE a.enterprise_id = $1 AND ($2 = '' OR a.status = $2) AND ($3::uuid IS NULL OR a.aprovador_id = $3) AND ($4::bigint IS NULL OR a.requisition_code = $4)
		 ORDER BY a.criado_em DESC LIMIT 500`, empresa, strings.ToUpper(strings.TrimSpace(status)), aprovador, requisitionCode)
	if err != nil {
		return nil, fmt.Errorf("listando as aprovações de requisição: %w", err)
	}
	defer rows.Close()
	var out []gestao.AprovacaoRequisicao
	for rows.Next() {
		a, err := scanAprovacao(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *GestaoPG) Aprovacao(ctx context.Context, id int64) (*gestao.AprovacaoRequisicao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	a, err := scanAprovacao(r.pool.QueryRow(ctx, `SELECT `+colunasAprovacao+fromAprovacao+` WHERE a.enterprise_id = $1 AND a.id = $2`, empresa, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError("aprovação não encontrada")
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// GravarDecisao só grava sobre a pendente: duas decisões simultâneas, uma passa.
func (r *GestaoPG) GravarDecisao(ctx context.Context, a *gestao.AprovacaoRequisicao) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE requisicao_aprovacoes_centro SET status = $3, decidido_por = $4, decidido_em = $5, justificativa = $6
		  WHERE id = $1 AND enterprise_id = $2 AND status = 'PENDENTE'`, a.ID, empresa, a.Status, a.DecididoPor, a.DecididoEm, a.Justificativa)
	if err != nil {
		return fmt.Errorf("gravando a decisão: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewConflictError("esta aprovação já foi decidida por outra pessoa")
	}
	return nil
}

func (r *GestaoPG) RequisicoesBloqueadas(ctx context.Context, codes []int64) (map[int64]string, error) {
	out := map[int64]string{}
	if len(codes) == 0 {
		return out, nil
	}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT a.requisition_code, a.status, g.codigo FROM requisicao_aprovacoes_centro a
		JOIN cost_center_gestao g ON g.cost_center_id = a.cost_center_id
		WHERE a.enterprise_id = $1 AND a.requisition_code = ANY($2) AND a.status <> 'APROVADA'
		ORDER BY a.requisition_code, g.codigo`, empresa, codes)
	if err != nil {
		return nil, fmt.Errorf("conferindo as aprovações das requisições: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var code int64
		var status, centro string
		if err := rows.Scan(&code, &status, &centro); err != nil {
			return nil, err
		}
		motivo := fmt.Sprintf("aguardando o aprovador do centro %s", centro)
		if status == gestao.AprovacaoReprovada {
			motivo = fmt.Sprintf("reprovada pelo aprovador do centro %s", centro)
		}
		if out[code] != "" {
			out[code] += "; " + motivo
		} else {
			out[code] = motivo
		}
	}
	return out, rows.Err()
}

func (r *GestaoPG) Usuarios(ctx context.Context) ([]gestao.Usuario, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT u.id, u.name FROM users u JOIN user_enterprises ue ON ue.user_id = u.id
		WHERE ue.enterprise_id = $1 AND COALESCE(u.is_active, TRUE) ORDER BY u.name`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listando os usuários da empresa: %w", err)
	}
	defer rows.Close()
	var out []gestao.Usuario
	for rows.Next() {
		var x gestao.Usuario
		if err := rows.Scan(&x.ID, &x.Nome); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
