// Package sales_commission grava o rateio de comissão do pedido e do orçamento.
//
// As duas tabelas têm o mesmo desenho, então o mesmo código serve as duas — o
// que muda é o nome da tabela e o da coluna do documento, resolvidos por
// `tabelas` e nunca por texto vindo da requisição.
package sales_commission

import (
	"context"
	"errors"
	"fmt"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_commission/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_commission/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

type tabela struct {
	rateio string
	coluna string
	capa   string
	itens  string
	itemFK string
	rotulo string
}

func tabelaDe(doc entity.Documento) (tabela, error) {
	switch doc {
	case entity.DocumentoPedido:
		return tabela{
			rateio: "public.sales_order_representatives",
			coluna: "sales_order_code",
			capa:   "public.sales_orders",
			itens:  "public.sales_order_items",
			itemFK: "sales_order_code",
			rotulo: "pedido",
		}, nil
	case entity.DocumentoOrcamento:
		return tabela{
			rateio: "public.sales_quotation_representatives",
			coluna: "sales_quotation_code",
			capa:   "public.sales_quotations",
			itens:  "public.sales_quotation_items",
			itemFK: "sales_quotation_code",
			rotulo: "orçamento",
		}, nil
	}
	return tabela{}, errorsuc.NewValidationError(fmt.Sprintf("documento de comissão inválido: %s", doc))
}

func (r *Repository) Listar(ctx context.Context, doc entity.Documento, documentCode int64) ([]*entity.Rateio, error) {
	t, err := tabelaDe(doc)
	if err != nil {
		return nil, err
	}
	tenantCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
SELECT c.id, c.enterprise_code, c.%[2]s, c.representative_code, COALESCE(rep.name, ''),
       c.role::text, c.commission_pct, c.commission_base::text, c.notes, c.created_at, c.updated_at
FROM %[1]s c
LEFT JOIN public.representatives rep ON rep.code = c.representative_code
WHERE c.%[2]s = $1 AND c.enterprise_code = $2
ORDER BY (c.role <> 'PRINCIPAL'), c.representative_code`, t.rateio, t.coluna), documentCode, tenantCode)
	if err != nil {
		return nil, fmt.Errorf("listar rateio de comissão: %w", err)
	}
	defer rows.Close()
	var out []*entity.Rateio
	for rows.Next() {
		var l entity.Rateio
		var papel, base string
		if err := rows.Scan(&l.ID, &l.EnterpriseCode, &l.DocumentCode, &l.RepresentativeCode, &l.RepresentativeName,
			&papel, &l.CommissionPct, &base, &l.Notes, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, fmt.Errorf("ler rateio de comissão: %w", err)
		}
		l.Role = entity.Papel(papel)
		l.CommissionBase = entity.Base(base)
		out = append(out, &l)
	}
	return out, rows.Err()
}

func (r *Repository) Totais(ctx context.Context, doc entity.Documento, documentCode int64) (domrepo.Totais, error) {
	t, err := tabelaDe(doc)
	if err != nil {
		return domrepo.Totais{}, err
	}
	tenantCode, err := tenant.Code(ctx)
	if err != nil {
		return domrepo.Totais{}, err
	}
	var tot domrepo.Totais
	// O total dos produtos é somado dos itens: na capa do orçamento o
	// `total_net` já carrega frete, seguro e desconto de capa, e comissão sobre
	// frete é justamente o que a base explícita evita.
	err = r.pool.QueryRow(ctx, fmt.Sprintf(`
SELECT COALESCE((SELECT SUM(i.total_net) FROM %[2]s i WHERE i.%[3]s = d.code AND i.is_active = TRUE), 0),
       COALESCE(d.total_net, 0), d.representative_code, COALESCE(d.commission_pct, 0)
FROM %[1]s d
WHERE d.code = $1 AND d.enterprise_code = $2`, t.capa, t.itens, t.itemFK), documentCode, tenantCode).
		Scan(&tot.TotalProdutos, &tot.TotalLiquido, &tot.RepresentativeCode, &tot.CommissionPct)
	if errors.Is(err, pgx.ErrNoRows) {
		return domrepo.Totais{}, errorsuc.NewNotFoundError(fmt.Sprintf("%s %d não encontrado", t.rotulo, documentCode))
	}
	if err != nil {
		return domrepo.Totais{}, fmt.Errorf("totais do documento de comissão: %w", err)
	}
	return tot, nil
}

func (r *Repository) RepresentantesAtivos(ctx context.Context, codigos []int64) (map[int64]string, error) {
	out := map[int64]string{}
	if len(codigos) == 0 {
		return out, nil
	}
	tenantCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	// Mesma regra da validação do representante na capa do pedido: só vale o
	// representante vinculado à empresa da sessão. Sem isso, o rateio aceitaria
	// o representante de outra empresa.
	rows, err := r.pool.Query(ctx, `
SELECT r.code, COALESCE(r.name, '') FROM public.representatives r
WHERE r.code = ANY($1) AND r.is_active = TRUE AND r.blocked = FALSE
  AND EXISTS (SELECT 1 FROM public.representative_enterprises re
              WHERE re.representative_code = r.code AND re.enterprise_code = $2)`, codigos, tenantCode)
	if err != nil {
		return nil, fmt.Errorf("conferir representantes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var code int64
		var name string
		if err := rows.Scan(&code, &name); err != nil {
			return nil, err
		}
		out[code] = name
	}
	return out, rows.Err()
}

// Substituir grava o rateio inteiro de uma vez. É tudo ou nada: rateio pela
// metade é comissão paga errado.
func (r *Repository) Substituir(ctx context.Context, doc entity.Documento, documentCode int64, linhas []*entity.Rateio) ([]*entity.Rateio, error) {
	t, err := tabelaDe(doc)
	if err != nil {
		return nil, err
	}
	tenantCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var existe int64
	err = tx.QueryRow(ctx, fmt.Sprintf(`SELECT code FROM %s WHERE code = $1 AND enterprise_code = $2 FOR UPDATE`, t.capa),
		documentCode, tenantCode).Scan(&existe)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("%s %d não encontrado", t.rotulo, documentCode))
	}
	if err != nil {
		return nil, fmt.Errorf("travar documento de comissão: %w", err)
	}

	if _, err := tx.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE %s = $1 AND enterprise_code = $2`, t.rateio, t.coluna),
		documentCode, tenantCode); err != nil {
		return nil, fmt.Errorf("limpar rateio de comissão: %w", err)
	}
	for _, l := range linhas {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`
INSERT INTO %s (enterprise_code, %s, representative_code, role, commission_pct, commission_base, notes)
VALUES ($1, $2, $3, $4::sales_commission_role_enum, $5, $6::sales_commission_base_enum, $7)`, t.rateio, t.coluna),
			tenantCode, documentCode, l.RepresentativeCode, string(l.Role), l.CommissionPct, string(l.CommissionBase), l.Notes); err != nil {
			return nil, fmt.Errorf("gravar rateio de comissão: %w", err)
		}
	}

	// A capa continua espelhando o principal: relatórios e integrações que leem
	// `representative_code`/`commission_pct` seguem respondendo o mesmo.
	principal := entity.Principal(linhas)
	if principal != nil {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`
UPDATE %s SET representative_code = $1, commission_pct = $2, updated_at = NOW()
WHERE code = $3 AND enterprise_code = $4`, t.capa),
			principal.RepresentativeCode, principal.CommissionPct, documentCode, tenantCode); err != nil {
			return nil, fmt.Errorf("espelhar representante principal: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.Listar(ctx, doc, documentCode)
}

var _ domrepo.Repository = (*Repository)(nil)
