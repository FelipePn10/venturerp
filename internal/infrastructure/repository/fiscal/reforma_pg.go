package fiscal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/reforma"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// ReformaPG implementa a parametrização da reforma tributária. A tabela
// oficial de cClassTrib é nacional (sem empresa); alíquotas, CFOP e notas são
// sempre filtrados pela empresa da sessão.
type ReformaPG struct {
	pool *pgxpool.Pool
}

func NewReformaPG(pool *pgxpool.Pool) *ReformaPG { return &ReformaPG{pool: pool} }

var _ repository.ReformaTributariaRepository = (*ReformaPG)(nil)

// ─── alíquotas ────────────────────────────────────────────────────────────────

func (r *ReformaPG) AliquotasDoAno(ctx context.Context, ano int) (*reforma.Aliquotas, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var a reforma.Aliquotas
	err = r.pool.QueryRow(ctx,
		`SELECT ibs_uf, ibs_mun, cbs FROM ibscbs_aliquotas WHERE enterprise_id = $1 AND ano = $2`,
		empresa, ano).Scan(&a.IBSUF, &a.IBSMun, &a.CBS)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lendo as alíquotas de IBS/CBS de %d: %w", ano, err)
	}
	a.Fonte = reforma.FonteEmpresa
	return &a, nil
}

func (r *ReformaPG) ListarAliquotas(ctx context.Context) ([]reforma.AliquotasCadastradas, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT ano, ibs_uf, ibs_mun, cbs, fundamento, updated_at, updated_by
		   FROM ibscbs_aliquotas WHERE enterprise_id = $1 ORDER BY ano`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listando as alíquotas de IBS/CBS: %w", err)
	}
	defer rows.Close()
	out := []reforma.AliquotasCadastradas{}
	for rows.Next() {
		var c reforma.AliquotasCadastradas
		if err := rows.Scan(&c.Ano, &c.Aliquotas.IBSUF, &c.Aliquotas.IBSMun, &c.Aliquotas.CBS, &c.Fundamento, &c.UpdatedAt, &c.UpdatedBy); err != nil {
			return nil, err
		}
		c.Aliquotas.Fonte = reforma.FonteEmpresa
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *ReformaPG) SalvarAliquotas(ctx context.Context, ano int, a reforma.Aliquotas, fundamento string, actor uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO ibscbs_aliquotas (enterprise_id, ano, ibs_uf, ibs_mun, cbs, fundamento, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (enterprise_id, ano) DO UPDATE
		   SET ibs_uf = EXCLUDED.ibs_uf, ibs_mun = EXCLUDED.ibs_mun, cbs = EXCLUDED.cbs,
		       fundamento = EXCLUDED.fundamento, updated_by = EXCLUDED.updated_by, updated_at = NOW()`,
		empresa, ano, a.IBSUF, a.IBSMun, a.CBS, fundamento, actor)
	if err != nil {
		return fmt.Errorf("gravando as alíquotas de IBS/CBS de %d: %w", ano, err)
	}
	return nil
}

func (r *ReformaPG) ExcluirAliquotas(ctx context.Context, ano int) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM ibscbs_aliquotas WHERE enterprise_id = $1 AND ano = $2`, empresa, ano)
	if err != nil {
		return fmt.Errorf("excluindo as alíquotas de IBS/CBS de %d: %w", ano, err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("não há alíquotas cadastradas para %d", ano))
	}
	return nil
}

// ─── tabela oficial ───────────────────────────────────────────────────────────

const colunasClassificacao = `cclass_trib, cst, cst_descricao, descricao, exige_tributacao, reducao_aliquota, diferimento,
	perc_reducao_ibs, perc_reducao_cbs, tipo_aliquota, tributacao_regular, aceita_nfe,
	vigencia_inicio, vigencia_fim, url_legislacao, versao_tabela`

func scanClassificacao(row pgx.Row) (reforma.ClassificacaoOficial, error) {
	var c reforma.ClassificacaoOficial
	err := row.Scan(&c.ClassTrib, &c.CST, &c.CSTDescricao, &c.Descricao, &c.ExigeTributacao, &c.ReducaoAliquota, &c.Diferimento,
		&c.PercReducaoIBS, &c.PercReducaoCBS, &c.TipoAliquota, &c.TributacaoRegular, &c.AceitaNFe,
		&c.VigenciaInicio, &c.VigenciaFim, &c.URLLegislacao, &c.VersaoTabela)
	return c, err
}

func (r *ReformaPG) ListarClassificacoes(ctx context.Context, cst, busca string, somenteNFe bool) ([]reforma.ClassificacaoOficial, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+colunasClassificacao+`
		  FROM ibscbs_classificacoes
		 WHERE ($1 = '' OR cst = $1)
		   AND ($2 = '' OR cclass_trib LIKE $2 || '%' OR descricao ILIKE '%' || $2 || '%' OR cst_descricao ILIKE '%' || $2 || '%')
		   AND (NOT $3 OR aceita_nfe)
		 ORDER BY cclass_trib`, strings.TrimSpace(cst), strings.TrimSpace(busca), somenteNFe)
	if err != nil {
		return nil, fmt.Errorf("listando a tabela de classificação tributária: %w", err)
	}
	defer rows.Close()
	out := []reforma.ClassificacaoOficial{}
	for rows.Next() {
		c, err := scanClassificacao(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *ReformaPG) BuscarClassificacoes(ctx context.Context, codigos []string) (map[string]reforma.ClassificacaoOficial, error) {
	out := map[string]reforma.ClassificacaoOficial{}
	if len(codigos) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT `+colunasClassificacao+` FROM ibscbs_classificacoes WHERE cclass_trib = ANY($1)`, codigos)
	if err != nil {
		return nil, fmt.Errorf("lendo a tabela de classificação tributária: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		c, err := scanClassificacao(rows)
		if err != nil {
			return nil, err
		}
		out[c.ClassTrib] = c
	}
	return out, rows.Err()
}

// ─── CFOP ─────────────────────────────────────────────────────────────────────

func (r *ReformaPG) ListarCFOP(ctx context.Context) ([]reforma.CFOPClassificacao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT m.cfop, m.cclass_trib, c.descricao, m.observacao, m.updated_at, m.updated_by
		  FROM ibscbs_cfop_classificacoes m
		  JOIN ibscbs_classificacoes c ON c.cclass_trib = m.cclass_trib
		 WHERE m.enterprise_id = $1 ORDER BY m.cfop`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listando o cClassTrib por CFOP: %w", err)
	}
	defer rows.Close()
	out := []reforma.CFOPClassificacao{}
	for rows.Next() {
		var m reforma.CFOPClassificacao
		if err := rows.Scan(&m.CFOP, &m.ClassTrib, &m.Descricao, &m.Observacao, &m.UpdatedAt, &m.UpdatedBy); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *ReformaPG) SalvarCFOP(ctx context.Context, cfop, classTrib string, observacao *string, actor uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO ibscbs_cfop_classificacoes (enterprise_id, cfop, cclass_trib, observacao, updated_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (enterprise_id, cfop) DO UPDATE
		   SET cclass_trib = EXCLUDED.cclass_trib, observacao = EXCLUDED.observacao,
		       updated_by = EXCLUDED.updated_by, updated_at = NOW()`,
		empresa, cfop, classTrib, observacao, actor)
	if err != nil {
		return fmt.Errorf("gravando o cClassTrib do CFOP %s: %w", cfop, err)
	}
	return nil
}

func (r *ReformaPG) ExcluirCFOP(ctx context.Context, cfop string) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM ibscbs_cfop_classificacoes WHERE enterprise_id = $1 AND cfop = $2`, empresa, cfop)
	if err != nil {
		return fmt.Errorf("excluindo o cClassTrib do CFOP %s: %w", cfop, err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("o CFOP %s não tem cClassTrib padrão cadastrado", cfop))
	}
	return nil
}

// ─── nota de saída ────────────────────────────────────────────────────────────

// CandidatosDaNota: a classificação fiscal é a de venda do item (código ou
// NCM), vigente na data de emissão; sem item cadastrado, a do NCM da linha. A
// classificação encontrada sem cClassTrib não procura outra: devolve vazio e a
// precedência cai para o padrão.
func (r *ReformaPG) CandidatosDaNota(ctx context.Context, exitID int64) (map[int64]reforma.Candidatos, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT it.id,
		       COALESCE(btrim(it.ibscbs_cclass_trib), ''),
		       COALESCE(btrim(m.cclass_trib), ''),
		       COALESCE((
		           SELECT NULLIF(btrim(f.cod_clas_trib), '')
		             FROM fiscal_classifications f
		             LEFT JOIN items i ON i.code = it.item_code AND i.enterprise_id = e.enterprise_id
		            WHERE f.enterprise_id = e.enterprise_id AND f.is_active
		              AND f.valid_from <= e.data_emissao AND (f.valid_until IS NULL OR f.valid_until >= e.data_emissao)
		              AND ((NULLIF(i.accounting_sale_fiscal_classification_code, '') IS NOT NULL
		                    AND (f.code::text = i.accounting_sale_fiscal_classification_code OR f.ncm = i.accounting_sale_fiscal_classification_code))
		                   OR (COALESCE(it.ncm, '') <> ''
		                    AND regexp_replace(COALESCE(f.ncm, ''), '[^0-9]', '', 'g') = regexp_replace(it.ncm, '[^0-9]', '', 'g')))
		            ORDER BY (f.code::text = COALESCE(i.accounting_sale_fiscal_classification_code, '')) DESC,
		                     (f.ncm = COALESCE(i.accounting_sale_fiscal_classification_code, '')) DESC,
		                     f.valid_from DESC, f.id DESC
		            LIMIT 1), '')
		  FROM fiscal_exit_items it
		  JOIN fiscal_exits e ON e.id = it.fiscal_exit_id
		  LEFT JOIN ibscbs_cfop_classificacoes m ON m.enterprise_id = e.enterprise_id AND m.cfop = it.cfop
		 WHERE it.fiscal_exit_id = $1 AND e.enterprise_id = $2`, exitID, empresa)
	if err != nil {
		return nil, fmt.Errorf("resolvendo o cClassTrib dos itens da nota %d: %w", exitID, err)
	}
	defer rows.Close()
	out := map[int64]reforma.Candidatos{}
	for rows.Next() {
		var id int64
		var c reforma.Candidatos
		if err := rows.Scan(&id, &c.DoItem, &c.DoCFOP, &c.DaClassificacao); err != nil {
			return nil, err
		}
		out[id] = c
	}
	return out, rows.Err()
}

func (r *ReformaPG) DefinirClassTribDoItem(ctx context.Context, exitID, itemID int64, classTrib *string) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var status string
	err = r.pool.QueryRow(ctx, `
		SELECT e.status FROM fiscal_exit_items it JOIN fiscal_exits e ON e.id = it.fiscal_exit_id
		 WHERE it.id = $1 AND it.fiscal_exit_id = $2 AND e.enterprise_id = $3`, itemID, exitID, empresa).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewNotFoundError(fmt.Sprintf("a linha %d não pertence à NF-e %d", itemID, exitID))
	}
	if err != nil {
		return fmt.Errorf("lendo a linha %d da NF-e %d: %w", itemID, exitID, err)
	}
	// Mesmos estados em que a nota ainda pode ser transmitida.
	if status != "DRAFT" && status != "REJECTED" {
		return errorsuc.NewValidationError("a classificação tributária só muda em nota em rascunho ou rejeitada")
	}
	_, err = r.pool.Exec(ctx, `UPDATE fiscal_exit_items SET ibscbs_cclass_trib = $1 WHERE id = $2 AND fiscal_exit_id = $3`, classTrib, itemID, exitID)
	if err != nil {
		return fmt.Errorf("gravando o cClassTrib da linha %d: %w", itemID, err)
	}
	return nil
}

func (r *ReformaPG) GravarRetrato(ctx context.Context, exitID int64, itens []reforma.RetratoItem, totalIBS, totalCBS decimal.Decimal) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE fiscal_exits SET valor_ibs = $1, valor_cbs = $2 WHERE id = $3 AND enterprise_id = $4`,
		totalIBS, totalCBS, exitID, empresa)
	if err != nil {
		return fmt.Errorf("gravando os totais de IBS/CBS da NF-e %d: %w", exitID, err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("NF-e %d não encontrada", exitID))
	}
	for _, it := range itens {
		g := it.Grupo
		var base, ufAliq, ufValor, munAliq, munValor, cbsAliq, cbsValor *decimal.Decimal
		if g.ComValores {
			base, ufAliq, ufValor = &g.Base, &g.IBSUF.AliquotaEfetiva, &g.IBSUF.Valor
			munAliq, munValor, cbsAliq, cbsValor = &g.IBSMun.AliquotaEfetiva, &g.IBSMun.Valor, &g.CBS.AliquotaEfetiva, &g.CBS.Valor
		}
		if _, err := tx.Exec(ctx, `
			UPDATE fiscal_exit_items
			   SET ibscbs_cst_enviado = $1, ibscbs_cclass_enviado = $2, ibscbs_origem = $3, ibscbs_base = $4,
			       ibs_uf_aliquota_efetiva = $5, ibs_uf_valor = $6, ibs_mun_aliquota_efetiva = $7, ibs_mun_valor = $8,
			       cbs_aliquota_efetiva = $9, cbs_valor = $10
			 WHERE id = $11 AND fiscal_exit_id = $12`,
			g.CST, g.ClassTrib, it.Origem, base, ufAliq, ufValor, munAliq, munValor, cbsAliq, cbsValor, it.ItemID, exitID); err != nil {
			return fmt.Errorf("gravando o IBS/CBS da linha %d: %w", it.ItemID, err)
		}
	}
	return tx.Commit(ctx)
}

// ─── modo ─────────────────────────────────────────────────────────────────────

func (r *ReformaPG) Parametros(ctx context.Context) (reforma.Parametros, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return reforma.Parametros{}, err
	}
	p := reforma.Parametros{Modo: reforma.ModoAutomatico}
	var modo string
	err = r.pool.QueryRow(ctx, `SELECT modo, motivo, updated_at, updated_by FROM ibscbs_parametros WHERE enterprise_id = $1`, empresa).
		Scan(&modo, &p.Motivo, &p.UpdatedAt, &p.UpdatedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, nil
	}
	if err != nil {
		return p, fmt.Errorf("lendo o modo da reforma tributária: %w", err)
	}
	p.Modo = reforma.Modo(modo)
	return p, nil
}

func (r *ReformaPG) SalvarParametros(ctx context.Context, p reforma.Parametros, actor uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO ibscbs_parametros (enterprise_id, modo, motivo, updated_by) VALUES ($1, $2, $3, $4)
		ON CONFLICT (enterprise_id) DO UPDATE SET modo = EXCLUDED.modo, motivo = EXCLUDED.motivo,
		       updated_by = EXCLUDED.updated_by, updated_at = NOW()`, empresa, string(p.Modo), p.Motivo, actor)
	if err != nil {
		return fmt.Errorf("gravando o modo da reforma tributária: %w", err)
	}
	return nil
}
