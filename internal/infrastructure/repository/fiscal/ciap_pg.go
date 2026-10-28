package fiscal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/ciap"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// CIAPPG: bens, parâmetros e apuração do CIAP da empresa da sessão.
type CIAPPG struct {
	pool *pgxpool.Pool
}

func NewCIAPPG(pool *pgxpool.Pool) *CIAPPG { return &CIAPPG{pool: pool} }

var _ ciap.Repository = (*CIAPPG)(nil)

type (
	ParametrosCIAP  = ciap.Parametros
	BemCIAP         = ciap.BemCadastrado
	BemManual       = ciap.BemManual
	ApuracaoGravada = ciap.ApuracaoGravada
)

func (r *CIAPPG) Parametros(ctx context.Context) (ParametrosCIAP, error) {
	p := ParametrosCIAP{NumeroParcelas: 48}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return p, err
	}
	var cod *string
	var n int16
	err = r.pool.QueryRow(ctx, `SELECT codigo_ajuste, conta_ativo_id, numero_parcelas FROM ciap_parametros WHERE enterprise_id = $1`, empresa).Scan(&cod, &p.ContaAtivoID, &n)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	if cod != nil {
		p.CodigoAjuste = *cod
	}
	p.NumeroParcelas = int(n)
	return p, nil
}

func (r *CIAPPG) SalvarParametros(ctx context.Context, p ParametrosCIAP, por uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	if p.ContaAtivoID != nil {
		if err := r.conferirConta(ctx, empresa, *p.ContaAtivoID); err != nil {
			return err
		}
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO ciap_parametros (enterprise_id, codigo_ajuste, conta_ativo_id, numero_parcelas, updated_by)
		VALUES ($1, NULLIF($2,''), $3, $4, $5)
		ON CONFLICT (enterprise_id) DO UPDATE SET codigo_ajuste = EXCLUDED.codigo_ajuste, conta_ativo_id = EXCLUDED.conta_ativo_id,
			numero_parcelas = EXCLUDED.numero_parcelas, updated_by = EXCLUDED.updated_by, updated_at = NOW()`,
		empresa, strings.ToUpper(strings.TrimSpace(p.CodigoAjuste)), p.ContaAtivoID, p.NumeroParcelas, por)
	return err
}

// conferirConta: conta analítica do plano contábil da empresa.
func (r *CIAPPG) conferirConta(ctx context.Context, empresa, id int64) error {
	var ok bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM accounting_accounts a JOIN accounting_plans p ON p.id = a.plan_id
		WHERE a.id = $1 AND p.enterprise_id = $2 AND a.is_analytic)`, id, empresa).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return errorsuc.NewValidationError("a conta do ativo tem de ser conta analítica do plano contábil da empresa (VCTB0200)")
	}
	return nil
}

// Sincronizar põe no CIAP os bens das notas de entrada aprovadas (CFOP de
// ativo x551/x406 com ICMS) e tira os da nota cancelada depois: sem parcela
// apropriada, o bem sai; com parcela, fica baixado (OT). Idempotente. O
// optante do Simples Nacional não tem crédito, nem CIAP.
func (r *CIAPPG) Sincronizar(ctx context.Context) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var regime string
	if err := r.pool.QueryRow(ctx, `SELECT COALESCE(regime_tributario,'') FROM fiscal_configs WHERE enterprise_id = $1`, empresa).Scan(&regime); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if regime == "1" {
		return nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		INSERT INTO ciap_bens (enterprise_id, codigo, descricao, origem, fiscal_entry_item_id, data_entrada, icms_op, icms_st,
			numero_parcelas, conta_ativo_id, cost_center_id)
		SELECT fe.enterprise_id,
		       'NF' || fe.numero_nf || '-' || COALESCE(fe.serie,'') || '-' || fi.sequence || '-' || fe.id,
		       COALESCE(NULLIF(btrim(i.name),''), NULLIF(btrim(fi.description),''), 'Bem da NF ' || fe.numero_nf),
		       'NF_ENTRADA', fi.id, COALESCE(fe.data_entrada, fe.data_emissao),
		       COALESCE(fi.valor_icms,0), COALESCE(fi.valor_icms_st,0),
		       COALESCE(p.numero_parcelas, 48), p.conta_ativo_id, fi.centro_custo_id
		  FROM fiscal_entry_items fi
		  JOIN fiscal_entries fe ON fe.id = fi.fiscal_entry_id
		  LEFT JOIN items i ON i.code = fi.item_code AND i.enterprise_id = fe.enterprise_id
		  LEFT JOIN ciap_parametros p ON p.enterprise_id = fe.enterprise_id
		 WHERE fe.enterprise_id = $1 AND fe.status = 'APPROVED'
		   AND right(COALESCE(fi.cfop_entrada, ''), 3) IN ('551', '406')
		   AND COALESCE(fi.valor_icms,0) + COALESCE(fi.valor_icms_st,0) > 0
		ON CONFLICT (fiscal_entry_item_id) DO NOTHING`, empresa); err != nil {
		return fmt.Errorf("trazendo os bens das notas de entrada para o CIAP: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM ciap_bens b USING fiscal_entry_items fi, fiscal_entries fe
		 WHERE b.enterprise_id = $1 AND b.fiscal_entry_item_id = fi.id AND fe.id = fi.fiscal_entry_id AND fe.status = 'CANCELLED'
		   AND NOT EXISTS (SELECT 1 FROM ciap_apuracao_parcelas p WHERE p.bem_id = b.id)`, empresa); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE ciap_bens b SET situacao = 'BAIXADO', data_baixa = CURRENT_DATE, tipo_baixa = 'OT', motivo_baixa = 'NF de entrada cancelada'
		  FROM fiscal_entry_items fi, fiscal_entries fe
		 WHERE b.enterprise_id = $1 AND b.situacao = 'ATIVO' AND b.fiscal_entry_item_id = fi.id AND fe.id = fi.fiscal_entry_id AND fe.status = 'CANCELLED'`, empresa); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

const colunasBem = `b.id, b.codigo, b.descricao, COALESCE(b.funcao,''), COALESCE(b.vida_util_meses,0),
	COALESCE(a.account_number,''), COALESCE(a.description,''), COALESCE(a.nivel,0),
	COALESCE(g.codigo,''), COALESCE(c.description,''),
	b.data_entrada, b.icms_op, b.icms_st, b.icms_frete, b.icms_difal, b.numero_parcelas,
	b.situacao = 'BAIXADO', b.data_baixa, COALESCE(b.tipo_baixa,''),
	b.origem, b.conta_ativo_id, b.cost_center_id,
	fe.modelo, fe.serie, fe.numero_nf, fe.chave_acesso, fe.data_emissao, fe.cnpj_emitente,
	fi.sequence, fi.item_code, fi.id, fi.uom, fi.quantity`

// O nível da conta (0500) é a profundidade na árvore do plano contábil.
const fromBem = ` FROM ciap_bens b
	LEFT JOIN LATERAL (
	  WITH RECURSIVE sobe AS (
	    SELECT x.id, x.parent_id, 1 AS n FROM accounting_accounts x WHERE x.id = b.conta_ativo_id
	    UNION ALL SELECT y.id, y.parent_id, s.n + 1 FROM accounting_accounts y JOIN sobe s ON y.id = s.parent_id WHERE s.n < 20)
	  SELECT aa.account_number, aa.description, (SELECT MAX(n) FROM sobe) AS nivel
	    FROM accounting_accounts aa WHERE aa.id = b.conta_ativo_id) a ON TRUE
	LEFT JOIN cost_centers c ON c.id = b.cost_center_id
	LEFT JOIN cost_center_gestao g ON g.cost_center_id = c.id
	LEFT JOIN fiscal_entry_items fi ON fi.id = b.fiscal_entry_item_id
	LEFT JOIN fiscal_entries fe ON fe.id = fi.fiscal_entry_id`

func scanBem(row pgx.Row) (BemCIAP, error) {
	var b BemCIAP
	var vida, nivel int
	var modelo, serie, chave, cnpj, uom *string
	var numero *int64
	var emissao *time.Time
	var seq *int
	var itemCode, itemLinha *int64
	var qtd *decimal.Decimal
	var nParc int16
	err := row.Scan(&b.ID, &b.Codigo, &b.Descricao, &b.Funcao, &vida, &b.ContaCodigo, &b.ContaNome, &nivel, &b.CentroCodigo, &b.CentroNome,
		&b.DataEntrada, &b.ICMSOp, &b.ICMSST, &b.ICMSFrete, &b.ICMSDifal, &nParc, &b.Baixado, &b.DataBaixa, &b.TipoBaixa,
		&b.Origem, &b.ContaAtivoID, &b.CostCenterID,
		&modelo, &serie, &numero, &chave, &emissao, &cnpj, &seq, &itemCode, &itemLinha, &uom, &qtd)
	if err != nil {
		return b, err
	}
	b.VidaUtilMeses, b.ContaNivel, b.NumeroParcelas = vida, nivel, int(nParc)
	if itemLinha != nil && numero != nil {
		d := &ciap.Documento{Modelo: deref(modelo), Serie: deref(serie), Numero: fmt.Sprint(*numero), Chave: deref(chave),
			CodPart: soDigitosRepo(deref(cnpj)), Unidade: strings.ToUpper(deref(uom))}
		if emissao != nil {
			d.Emissao = *emissao
		}
		if seq != nil {
			d.NumItem = *seq
		}
		d.CodItem = fmt.Sprintf("NFE%d", *itemLinha) // o mesmo código do item que o SPED usa nas entradas
		if itemCode != nil {
			d.CodItem = fmt.Sprint(*itemCode)
		}
		if qtd != nil {
			d.Quantidade = *qtd
		}
		b.Documento = d
	}
	return b, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (r *CIAPPG) Bens(ctx context.Context) ([]BemCIAP, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+colunasBem+`, (SELECT count(*) FROM ciap_apuracao_parcelas p WHERE p.bem_id = b.id AND p.numero_parcela IS NOT NULL)`+
		fromBem+` WHERE b.enterprise_id = $1 ORDER BY b.data_entrada DESC, b.id DESC`, empresa)
	if err != nil {
		return nil, fmt.Errorf("listando os bens do CIAP: %w", err)
	}
	defer rows.Close()
	var out []BemCIAP
	for rows.Next() {
		var usadas int
		b, err := scanBem(scannerComExtra{rows, &usadas})
		if err != nil {
			return nil, err
		}
		b.ParcelasUsadas = usadas
		out = append(out, b)
	}
	return out, rows.Err()
}

// scannerComExtra: lê as colunas do bem e mais uma no fim.
type scannerComExtra struct {
	rows  pgx.Rows
	extra any
}

func (s scannerComExtra) Scan(dest ...any) error { return s.rows.Scan(append(dest, s.extra)...) }

func (r *CIAPPG) Bem(ctx context.Context, id int64) (*BemCIAP, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var usadas int
	b, err := scanBem(scannerRow{r.pool.QueryRow(ctx, `SELECT `+colunasBem+`, (SELECT count(*) FROM ciap_apuracao_parcelas p WHERE p.bem_id = b.id AND p.numero_parcela IS NOT NULL)`+
		fromBem+` WHERE b.enterprise_id = $1 AND b.id = $2`, empresa, id), &usadas})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError("bem do CIAP não encontrado")
	}
	if err != nil {
		return nil, err
	}
	b.ParcelasUsadas = usadas
	return &b, nil
}

type scannerRow struct {
	row   pgx.Row
	extra any
}

func (s scannerRow) Scan(dest ...any) error { return s.row.Scan(append(dest, s.extra)...) }

func (r *CIAPPG) conferirCentro(ctx context.Context, empresa, id int64) error {
	var ok bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM cost_centers WHERE id = $1 AND enterprise_id = $2)`, id, empresa).Scan(&ok); err != nil {
		return err
	}
	if !ok {
		return errorsuc.NewValidationError("o centro de custo não é desta empresa")
	}
	return nil
}

func traduzirCodigoBem(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return errorsuc.NewConflictError("já existe bem no CIAP com este código")
	}
	return err
}

func (r *CIAPPG) CriarBem(ctx context.Context, b BemManual, por uuid.UUID) (int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	if b.ContaAtivoID != nil {
		if err := r.conferirConta(ctx, empresa, *b.ContaAtivoID); err != nil {
			return 0, err
		}
	}
	if b.CostCenterID != nil {
		if err := r.conferirCentro(ctx, empresa, *b.CostCenterID); err != nil {
			return 0, err
		}
	}
	var id int64
	err = r.pool.QueryRow(ctx, `INSERT INTO ciap_bens (enterprise_id, codigo, descricao, funcao, vida_util_meses, conta_ativo_id, cost_center_id,
			origem, data_entrada, icms_op, icms_st, icms_frete, icms_difal, numero_parcelas, created_by)
		VALUES ($1,$2,$3,NULLIF($4,''),NULLIF($5,0),$6,$7,'MANUAL',$8,$9,$10,$11,$12,$13,$14) RETURNING id`,
		empresa, b.Codigo, b.Descricao, b.Funcao, b.VidaUtilMeses, b.ContaAtivoID, b.CostCenterID, b.DataEntrada,
		b.ICMSOp.Round(2), b.ICMSST.Round(2), b.ICMSFrete.Round(2), b.ICMSDifal.Round(2), b.NumeroParcelas, por).Scan(&id)
	return id, traduzirCodigoBem(err)
}

// AtualizarBem: identificação e uso mudam sempre; os valores e a data de
// entrada só enquanto nenhuma parcela foi apropriada (e nunca no bem da NF).
func (r *CIAPPG) AtualizarBem(ctx context.Context, b BemManual, valores bool) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	if b.ContaAtivoID != nil {
		if err := r.conferirConta(ctx, empresa, *b.ContaAtivoID); err != nil {
			return err
		}
	}
	if b.CostCenterID != nil {
		if err := r.conferirCentro(ctx, empresa, *b.CostCenterID); err != nil {
			return err
		}
	}
	tag, err := r.pool.Exec(ctx, `UPDATE ciap_bens SET codigo = $3, descricao = $4, funcao = NULLIF($5,''), vida_util_meses = NULLIF($6,0),
			conta_ativo_id = $7, cost_center_id = $8,
			data_entrada = CASE WHEN $9 THEN $10 ELSE data_entrada END,
			icms_op = CASE WHEN $9 THEN $11 ELSE icms_op END, icms_st = CASE WHEN $9 THEN $12 ELSE icms_st END,
			icms_frete = CASE WHEN $9 THEN $13 ELSE icms_frete END, icms_difal = CASE WHEN $9 THEN $14 ELSE icms_difal END,
			numero_parcelas = CASE WHEN $9 THEN $15 ELSE numero_parcelas END
		  WHERE id = $1 AND enterprise_id = $2`,
		b.ID, empresa, b.Codigo, b.Descricao, b.Funcao, b.VidaUtilMeses, b.ContaAtivoID, b.CostCenterID, valores, b.DataEntrada,
		b.ICMSOp.Round(2), b.ICMSST.Round(2), b.ICMSFrete.Round(2), b.ICMSDifal.Round(2), b.NumeroParcelas)
	if err != nil {
		return traduzirCodigoBem(err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError("bem do CIAP não encontrado")
	}
	return nil
}

func (r *CIAPPG) Baixar(ctx context.Context, id int64, tipo string, dia time.Time, motivo string) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE ciap_bens SET situacao = 'BAIXADO', tipo_baixa = $3, data_baixa = $4, motivo_baixa = $5
		  WHERE id = $1 AND enterprise_id = $2 AND situacao = 'ATIVO' AND data_entrada <= $4`, id, empresa, tipo, dia, motivo)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewValidationError("o bem não está ativo no CIAP ou a data da baixa é anterior à entrada")
	}
	return nil
}

// ExcluirBem: só o manual, sem parcela apropriada.
func (r *CIAPPG) ExcluirBem(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM ciap_bens b WHERE b.id = $1 AND b.enterprise_id = $2 AND b.origem = 'MANUAL'
		AND NOT EXISTS (SELECT 1 FROM ciap_apuracao_parcelas p WHERE p.bem_id = b.id)`, id, empresa)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewConflictError("só sai o bem lançado à mão e ainda sem parcela apropriada; o da nota sai quando a nota é cancelada, e o apropriado se baixa")
	}
	return nil
}

// Saidas do mês para o índice: total das linhas das NF-e de saída autorizadas;
// tributadas e exportação = linha com ICMS destacado ou CFOP 7xxx. A legislação
// da UF pode excluir operações (remessas, devoluções): a tela deixa ajustar.
func (r *CIAPPG) Saidas(ctx context.Context, ano, mes int) (tribExp, total decimal.Decimal, err error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return
	}
	ini := fmt.Sprintf("%04d-%02d-01", ano, mes)
	err = r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(i.total_price) FILTER (WHERE COALESCE(i.valor_icms,0) > 0 OR i.cfop LIKE '7%'), 0),
		       COALESCE(SUM(i.total_price), 0)
		  FROM fiscal_exit_items i JOIN fiscal_exits e ON e.id = i.fiscal_exit_id
		 WHERE e.enterprise_id = $1 AND e.status = 'AUTHORIZED' AND COALESCE(e.tipo_documento, 1) = 1
		   AND e.data_emissao >= $2::date AND e.data_emissao < ($2::date + INTERVAL '1 month')`, empresa, ini).Scan(&tribExp, &total)
	return
}

func (r *CIAPPG) Apuracao(ctx context.Context, ano, mes int) (*ApuracaoGravada, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var a ApuracaoGravada
	err = r.pool.QueryRow(ctx, `SELECT id, saldo_inicial, soma_parcelas, saidas_trib_exp, saidas_total, indice, icms_apropriado, saidas_informadas, fechada, apurado_em
		  FROM ciap_apuracoes WHERE enterprise_id = $1 AND ano = $2 AND mes = $3`, empresa, ano, mes).
		Scan(&a.ID, &a.SaldoInicial, &a.SomaParcelas, &a.SaidasTribExp, &a.SaidasTotal, &a.Indice, &a.ICMSApropriado, &a.SaidasInformadas, &a.Fechada, &a.ApuradoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.Ano, a.Mes = ano, mes
	a.Inicio = time.Date(ano, time.Month(mes), 1, 0, 0, 0, 0, time.UTC)
	a.Fim = a.Inicio.AddDate(0, 1, -1)
	bens, err := r.Bens(ctx)
	if err != nil {
		return nil, err
	}
	porID := make(map[int64]ciap.Bem, len(bens))
	for _, b := range bens {
		porID[b.ID] = b.Bem
	}
	rows, err := r.pool.Query(ctx, `SELECT bem_id, tipo_mov, numero_parcela, valor_parcela FROM ciap_apuracao_parcelas WHERE apuracao_id = $1
		ORDER BY bem_id, CASE tipo_mov WHEN 'SI' THEN 0 WHEN 'IM' THEN 0 ELSE 1 END`, a.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var tipo string
		var n *int16
		var v *decimal.Decimal
		if err := rows.Scan(&id, &tipo, &n, &v); err != nil {
			return nil, err
		}
		b := porID[id]
		m := ciap.Movimento{Bem: b, TipoMov: tipo, ComValores: tipo == ciap.MovSaldoInicial || tipo == ciap.MovImobilizacao}
		switch tipo {
		case ciap.MovSaldoInicial:
			m.Data = a.Inicio
		case ciap.MovImobilizacao:
			m.Data = b.DataEntrada
		case ciap.MovFimPeriodo:
			m.Data = a.Fim
		default:
			if b.DataBaixa != nil {
				m.Data = *b.DataBaixa
			}
		}
		if n != nil {
			m.NumeroParcela = int(*n)
		}
		if v != nil {
			m.ValorParcela = *v
		}
		a.Movimentos = append(a.Movimentos, m)
	}
	return &a, rows.Err()
}

// GravarApuracao substitui a do mês (se não estiver fechada).
func (r *CIAPPG) GravarApuracao(ctx context.Context, a *ciap.Apuracao, informadas bool, por uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var fechada bool
	err = tx.QueryRow(ctx, `SELECT fechada FROM ciap_apuracoes WHERE enterprise_id = $1 AND ano = $2 AND mes = $3 FOR UPDATE`, empresa, a.Ano, a.Mes).Scan(&fechada)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if fechada {
		return errorsuc.NewConflictError(fmt.Sprintf("a apuração do CIAP de %02d/%d está fechada: reabra antes de apurar de novo", a.Mes, a.Ano))
	}
	if _, err := tx.Exec(ctx, `DELETE FROM ciap_apuracoes WHERE enterprise_id = $1 AND ano = $2 AND mes = $3`, empresa, a.Ano, a.Mes); err != nil {
		return err
	}
	var id int64
	if err := tx.QueryRow(ctx, `INSERT INTO ciap_apuracoes (enterprise_id, ano, mes, saldo_inicial, soma_parcelas, saidas_trib_exp, saidas_total, indice,
			icms_apropriado, saidas_informadas, apurado_por)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`, empresa, a.Ano, a.Mes, a.SaldoInicial, a.SomaParcelas, a.SaidasTribExp,
		a.SaidasTotal, a.Indice, a.ICMSApropriado, informadas, por).Scan(&id); err != nil {
		return err
	}
	for _, m := range a.Movimentos {
		var n *int
		var v *decimal.Decimal
		if m.NumeroParcela > 0 {
			nn, vv := m.NumeroParcela, m.ValorParcela
			n, v = &nn, &vv
		}
		if _, err := tx.Exec(ctx, `INSERT INTO ciap_apuracao_parcelas (apuracao_id, bem_id, tipo_mov, numero_parcela, valor_parcela) VALUES ($1,$2,$3,$4,$5)`,
			id, m.Bem.ID, m.TipoMov, n, v); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *CIAPPG) Fechar(ctx context.Context, ano, mes int, fechar bool) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE ciap_apuracoes SET fechada = $4 WHERE enterprise_id = $1 AND ano = $2 AND mes = $3`, empresa, ano, mes, fechar)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("não há apuração do CIAP de %02d/%d", mes, ano))
	}
	return nil
}

// ApuracoesDoAno: o resumo de cada mês apurado.
func (r *CIAPPG) ApuracoesDoAno(ctx context.Context, ano int) ([]ApuracaoGravada, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT id, mes, saldo_inicial, soma_parcelas, saidas_trib_exp, saidas_total, indice, icms_apropriado, saidas_informadas, fechada, apurado_em
		  FROM ciap_apuracoes WHERE enterprise_id = $1 AND ano = $2 ORDER BY mes`, empresa, ano)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ApuracaoGravada
	for rows.Next() {
		var a ApuracaoGravada
		var mes int16
		if err := rows.Scan(&a.ID, &mes, &a.SaldoInicial, &a.SomaParcelas, &a.SaidasTribExp, &a.SaidasTotal, &a.Indice, &a.ICMSApropriado,
			&a.SaidasInformadas, &a.Fechada, &a.ApuradoEm); err != nil {
			return nil, err
		}
		a.Ano, a.Mes = ano, int(mes)
		out = append(out, a)
	}
	return out, rows.Err()
}
