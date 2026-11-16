package customer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

var _ domainrepo.CustomerLifecycleRepository = (*CustomerRepositorySQLC)(nil)

// NewLifecycle devolve o mesmo repositório visto pelo contrato de ciclo de vida.
func NewLifecycle(r domainrepo.CustomerRepository) domainrepo.CustomerLifecycleRepository {
	return r.(*CustomerRepositorySQLC)
}

// openReceivableStatuses são os títulos que ainda contam como exposição; o
// mesmo critério da conferência de crédito do pedido.
const openReceivableStatuses = `('PENDENTE','APROVADO','VENCIDO','RENEGOCIADO')`

func (r *CustomerRepositorySQLC) GetCreditPolicy(ctx context.Context) (entity.CreditPolicy, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return entity.CreditPolicy{}, err
	}
	row, err := r.q.GetCreditPolicy(ctx, empresa)
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.DefaultCreditPolicy(), nil
	}
	if err != nil {
		return entity.CreditPolicy{}, fmt.Errorf("lendo a política de crédito: %w", err)
	}
	return entity.CreditPolicy{
		MaxLimit:              pgutil.FromPgNumericToDecimal(row.MaxLimit),
		NewCustomerCapitalPct: pgutil.FromPgNumericToDecimal(row.NewCustomerCapitalPct),
		NewCustomerCap:        pgutil.FromPgNumericToDecimal(row.NewCustomerCap),
		OverrideTolerancePct:  pgutil.FromPgNumericToDecimal(row.OverrideTolerancePct),
		OverdueBlockDays:      int(row.OverdueBlockDays),
		HistoryMonths:         int(row.HistoryMonths),
		ValidityMonths: map[entity.RiskClass]int{
			entity.RiskA: int(row.ValidityMonthsA), entity.RiskB: int(row.ValidityMonthsB), entity.RiskC: int(row.ValidityMonthsC),
			entity.RiskD: int(row.ValidityMonthsD), entity.RiskE: int(row.ValidityMonthsE),
		},
		RegistrationCheckDays: int(row.RegistrationCheckDays),
	}, nil
}

func (r *CustomerRepositorySQLC) SaveCreditPolicy(ctx context.Context, p entity.CreditPolicy, by uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.q.UpsertCreditPolicy(ctx, sqlc.UpsertCreditPolicyParams{
		EnterpriseID:          empresa,
		MaxLimit:              pgutil.ToPgNumericFromDecimal(p.MaxLimit),
		NewCustomerCapitalPct: pgutil.ToPgNumericFromDecimal(p.NewCustomerCapitalPct),
		NewCustomerCap:        pgutil.ToPgNumericFromDecimal(p.NewCustomerCap),
		OverrideTolerancePct:  pgutil.ToPgNumericFromDecimal(p.OverrideTolerancePct),
		OverdueBlockDays:      int32(p.OverdueBlockDays),
		HistoryMonths:         int32(p.HistoryMonths),
		ValidityMonthsA:       int32(p.ValidityMonths[entity.RiskA]),
		ValidityMonthsB:       int32(p.ValidityMonths[entity.RiskB]),
		ValidityMonthsC:       int32(p.ValidityMonths[entity.RiskC]),
		ValidityMonthsD:       int32(p.ValidityMonths[entity.RiskD]),
		ValidityMonthsE:       int32(p.ValidityMonths[entity.RiskE]),
		RegistrationCheckDays: int32(p.RegistrationCheckDays),
		UpdatedBy:             pgutil.ToPgUUID(by),
	})
	if err != nil {
		return fmt.Errorf("gravando a política de crédito: %w", err)
	}
	return nil
}

var naoDigito = regexp.MustCompile(`\D`)

func (r *CustomerRepositorySQLC) GroupCustomerCodes(ctx context.Context, docType entity.DocumentType, document string) ([]int64, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	digitos := naoDigito.ReplaceAllString(document, "")
	if docType != entity.DocumentCNPJ || len(digitos) != 14 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT code FROM customers
		 WHERE enterprise_id = $1 AND document_type = 'CNPJ'
		   AND LEFT(regexp_replace(document_number, '\D', '', 'g'), 8) = $2
		 ORDER BY code`, empresa, digitos[:8])
	if err != nil {
		return nil, fmt.Errorf("lendo o grupo econômico: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowTo[int64])
}

func (r *CustomerRepositorySQLC) GetCreditHistory(ctx context.Context, codes []int64, historyMonths int, today time.Time) (entity.CreditHistory, error) {
	var h entity.CreditHistory
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return h, err
	}
	if len(codes) == 0 {
		return h, nil
	}
	var primeiro *time.Time
	err = r.pool.QueryRow(ctx, `
		WITH t AS (
		    SELECT cr.valor_bruto AS v, cr.data_emissao AS em, cr.data_vencimento AS ve,
		           cr.data_recebimento AS rec, cr.status, cr.em_protesto,
		           cr.valor_bruto - COALESCE(cr.valor_recebido, 0) AS saldo
		      FROM contas_receber cr
		     WHERE cr.enterprise_id = $1 AND cr.cliente_id = ANY($2::bigint[])
		       AND cr.is_active AND cr.status <> 'CANCELADO'
		), janela AS (
		    SELECT * FROM t WHERE em >= $3::date - make_interval(months => $4::int)
		), pagos AS (
		    SELECT * FROM janela WHERE status = 'RECEBIDO' AND rec IS NOT NULL
		), recentes AS (
		    SELECT * FROM pagos WHERE rec >= $3::date - 90
		), abertos AS (
		    SELECT * FROM t WHERE status IN `+openReceivableStatuses+`
		)
		SELECT
		    (SELECT count(*) FROM pagos)::int,
		    (SELECT COALESCE(sum(v), 0) FROM pagos),
		    (SELECT COALESCE(sum(v) FILTER (WHERE rec <= ve), 0) FROM pagos),
		    (SELECT COALESCE(sum(GREATEST(rec - ve, 0) * v) / NULLIF(sum(v), 0), 0) FROM pagos),
		    (SELECT count(*) FROM recentes)::int,
		    (SELECT COALESCE(sum(GREATEST(rec - ve, 0) * v) / NULLIF(sum(v), 0), 0) FROM recentes),
		    (SELECT COALESCE(max(GREATEST(rec - ve, 0)), 0) FROM pagos)::int,
		    (SELECT COALESCE(sum((ve - em) * v) / NULLIF(sum(v), 0), 0) FROM t WHERE em >= $3::date - interval '12 months'),
		    (SELECT COALESCE(sum(v), 0) FROM t WHERE em >= $3::date - interval '12 months'),
		    (SELECT min(em) FROM t),
		    (SELECT count(*) FROM janela WHERE status = 'RENEGOCIADO')::int,
		    (SELECT count(*) FROM janela WHERE status = 'BAIXADO_PERDA')::int,
		    (SELECT COALESCE(sum(saldo), 0) FROM abertos),
		    (SELECT COALESCE(sum(saldo), 0) FROM abertos WHERE ve < $3::date),
		    (SELECT COALESCE(max($3::date - ve), 0) FROM abertos WHERE ve < $3::date)::int,
		    (SELECT COALESCE(bool_or(em_protesto), FALSE) FROM abertos),
		    (SELECT COALESCE(sum(d.valor_total), 0) FROM devolucoes_venda d
		      WHERE d.enterprise_id = $1 AND d.customer_code = ANY($2::bigint[])
		        AND d.situacao IN ('EMITIDA', 'EFETIVADA') AND d.created_at >= $3::date - interval '12 months')`,
		empresa, codes, today, historyMonths,
	).Scan(&h.PaidTitles, &h.PaidValue, &h.PaidOnTimeValue, &h.WeightedDelayDays,
		&h.RecentPaidTitles, &h.RecentWeightedDelayDays, &h.MaxDelayDays, &h.AvgTermDays, &h.Billed12m,
		&primeiro, &h.Renegotiated, &h.LostTitles, &h.OpenBalance, &h.OverdueBalance, &h.MaxOpenOverdueDays,
		&h.OpenProtest, &h.Returns12m)
	if err != nil {
		return h, fmt.Errorf("lendo o histórico de pagamento: %w", err)
	}
	if primeiro != nil {
		h.RelationshipMonths = (today.Year()-primeiro.Year())*12 + int(today.Month()) - int(primeiro.Month())
		h.RelationshipMonths = max(h.RelationshipMonths, 0)
	}
	return h, nil
}

func (r *CustomerRepositorySQLC) ApplyCreditReview(ctx context.Context, rev *entity.CreditReview) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	factors, err := json.Marshal(rev.Factors)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		score := int32(rev.Score)
		n, err := q.SetCustomerCredit(ctx, sqlc.SetCustomerCreditParams{
			CreditLimit:          pgutil.ToPgNumericFromDecimal(rev.ApprovedLimit),
			CreditLimitSuggested: pgutil.ToPgNumericFromDecimal(rev.SuggestedLimit),
			CreditScore:          &score,
			CreditRiskClass:      pgutil.ToPgText(string(rev.RiskClass)),
			CreditLimitExpiresAt: pgutil.ToPgDate(rev.ExpiresAt),
			Code:                 rev.CustomerCode,
			EnterpriseID:         empresa,
		})
		if err != nil {
			return fmt.Errorf("gravando o limite do cliente: %w", err)
		}
		if n == 0 {
			return errorsuc.NewNotFoundError(fmt.Sprintf("cliente %d não encontrado", rev.CustomerCode))
		}
		row, err := q.CreateCreditReview(ctx, sqlc.CreateCreditReviewParams{
			EnterpriseID:   empresa,
			CustomerCode:   rev.CustomerCode,
			Origin:         rev.Origin,
			SuggestedLimit: pgutil.ToPgNumericFromDecimal(rev.SuggestedLimit),
			ApprovedLimit:  pgutil.ToPgNumericFromDecimal(rev.ApprovedLimit),
			PreviousLimit:  pgutil.ToPgNumericFromDecimalPtr(rev.PreviousLimit),
			Score:          score,
			RiskClass:      string(rev.RiskClass),
			ExpiresAt:      pgutil.ToPgDate(rev.ExpiresAt),
			Deviates:       rev.Deviates,
			Reason:         pgutil.ToPgTextFromPtr(rev.Reason),
			Factors:        factors,
			CreatedBy:      pgutil.ToPgUUID(rev.CreatedBy),
		})
		if err != nil {
			return fmt.Errorf("gravando a análise de crédito: %w", err)
		}
		rev.ID, rev.CreatedAt = row.ID, row.CreatedAt.Time
		return nil
	})
}

func (r *CustomerRepositorySQLC) ListCreditReviews(ctx context.Context, code int64) ([]*entity.CreditReview, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListCreditReviews(ctx, sqlc.ListCreditReviewsParams{EnterpriseID: empresa, CustomerCode: code})
	if err != nil {
		return nil, fmt.Errorf("lendo a trilha de crédito: %w", err)
	}
	out := make([]*entity.CreditReview, 0, len(rows))
	for _, row := range rows {
		rev := &entity.CreditReview{
			ID: row.ID, CustomerCode: row.CustomerCode, Origin: row.Origin,
			SuggestedLimit: pgutil.FromPgNumericToDecimal(row.SuggestedLimit),
			ApprovedLimit:  pgutil.FromPgNumericToDecimal(row.ApprovedLimit),
			PreviousLimit:  pgutil.FromPgNumericToDecimalPtr(row.PreviousLimit),
			Score:          int(row.Score), RiskClass: entity.RiskClass(row.RiskClass),
			ExpiresAt: pgutil.FromPgDate(row.ExpiresAt), Deviates: row.Deviates,
			Reason: pgutil.FromPgTextPtr(row.Reason), CreatedBy: pgutil.FromPgUUID(row.CreatedBy), CreatedByName: row.CreatedByName,
			CreatedAt: row.CreatedAt.Time,
		}
		if len(row.Factors) > 0 {
			if err := json.Unmarshal(row.Factors, &rev.Factors); err != nil {
				return nil, fmt.Errorf("lendo os fatores da análise %d: %w", row.ID, err)
			}
		}
		out = append(out, rev)
	}
	return out, nil
}

func (r *CustomerRepositorySQLC) SetCustomerRegistration(ctx context.Context, code int64, s entity.RegistrationSnapshot) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.q.SetCustomerRegistration(ctx, sqlc.SetCustomerRegistrationParams{
		RegistrationStatus: pgutil.ToPgText(s.Status),
		OpeningDate:        pgutil.ToPgDateFromPtr(s.OpeningDate),
		ShareCapital:       pgutil.ToPgNumericFromDecimalPtr(s.ShareCapital),
		CompanySize:        s.CompanySize,
		LegalNature:        s.LegalNature,
		MainCnae:           s.MainCNAE,
		Code:               code,
		EnterpriseID:       empresa,
	})
	if err != nil {
		return fmt.Errorf("gravando a situação cadastral: %w", err)
	}
	return nil
}

func (r *CustomerRepositorySQLC) MarkRegistrationAttempt(ctx context.Context, code int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	if _, err := r.q.MarkRegistrationAttempt(ctx, sqlc.MarkRegistrationAttemptParams{Code: code, EnterpriseID: empresa}); err != nil {
		return fmt.Errorf("registrando a tentativa de consulta à Receita: %w", err)
	}
	return nil
}

func (r *CustomerRepositorySQLC) ListCustomersForRegistrationCheck(ctx context.Context, max int) ([]entity.RegistrationCheckTarget, error) {
	rows, err := r.q.ListCustomersForRegistrationCheck(ctx, int32(max))
	if err != nil {
		return nil, fmt.Errorf("listando clientes para reconsulta: %w", err)
	}
	out := make([]entity.RegistrationCheckTarget, 0, len(rows))
	for _, row := range rows {
		out = append(out, entity.RegistrationCheckTarget{EnterpriseID: row.EnterpriseID, EnterpriseCode: int64(row.EnterpriseCode), Actor: pgutil.FromPgUUID(row.CreatedBy), Code: row.Code, DocumentNumber: row.DocumentNumber, Blocked: row.Blocked})
	}
	return out, nil
}

// CustomerUsage conta, tabela a tabela, onde o cliente aparece. Cada tabela
// filtra pela convenção de empresa dela ($1 = enterprise_id, $2 =
// enterprise_code). customer_id em icms_reduction_substitutions e
// shipment_delivery_instructions é conferido como código ou id: na dúvida,
// a exclusão fica impedida.
func (r *CustomerRepositorySQLC) FindCustomerByDocumentDigits(ctx context.Context, digits string) (int64, string, bool, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return 0, "", false, err
	}
	var code int64
	var name string
	err = r.pool.QueryRow(ctx, `
		SELECT code, name FROM customers
		 WHERE enterprise_id = $1 AND regexp_replace(document_number, '\D', '', 'g') = $2
		 ORDER BY code LIMIT 1`, empresa, digits).Scan(&code, &name)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", false, nil
	}
	if err != nil {
		return 0, "", false, fmt.Errorf("procurando o documento: %w", err)
	}
	return code, name, true, nil
}

func (r *CustomerRepositorySQLC) CustomerUsage(ctx context.Context, code int64) ([]entity.CustomerUsage, error) {
	return customerUsage(ctx, r.pool, code, "")
}

// querier é o que pool e transação têm em comum.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// customerUsage lê o uso do cliente; lock = " FOR UPDATE" trava a linha do
// cliente até o fim da transação (exclusão).
func customerUsage(ctx context.Context, q querier, code int64, lock string) ([]entity.CustomerUsage, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	empresaCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	var id int64
	if err := q.QueryRow(ctx, `SELECT id FROM customers WHERE code = $1 AND enterprise_id = $2`+lock, code, empresa).Scan(&id); err != nil {
		return nil, naoAchou(err, fmt.Sprintf("cliente %d não encontrado", code))
	}
	rows, err := q.Query(ctx, `
		SELECT 'PEDIDOS', 'pedido(s) de venda', count(*), count(*) FILTER (WHERE status IN ('R','P','A','OA','OF'))
		  FROM sales_orders WHERE enterprise_code = $2 AND customer_code = $3
		UNION ALL
		SELECT 'ORDENS_PRODUCAO', 'ordem(ns) de produção de pedidos do cliente', count(*), count(*) FILTER (WHERE po.status IN ('OPEN','IN_PROGRESS'))
		  FROM production_orders po
		  JOIN sales_orders so ON so.code = po.sales_order_code AND so.enterprise_code = $2
		 WHERE po.enterprise_id = $1 AND so.customer_code = $3
		UNION ALL
		SELECT 'ORCAMENTOS', 'orçamento(s)', count(*), count(*) FILTER (WHERE status IN ('R','P','A','OA','OF','V','OV'))
		  FROM sales_quotations WHERE enterprise_code = $2 AND customer_code = $3
		UNION ALL
		SELECT 'TITULOS', 'título(s) a receber', count(*), count(*) FILTER (WHERE status IN `+openReceivableStatuses+`)
		  FROM contas_receber WHERE enterprise_id = $1 AND cliente_id = $3 AND is_active
		UNION ALL
		SELECT 'NOTAS_FISCAIS', 'nota(s) fiscal(is) de saída', count(*), count(*) FILTER (WHERE status IN ('DRAFT','AGUARDANDO_AUTORIZACAO'))
		  FROM fiscal_exits WHERE enterprise_id = $1 AND customer_code = $3
		UNION ALL
		SELECT 'NFSE', 'nota(s) de serviço', count(*), 0 FROM nfse WHERE enterprise_id = $1 AND customer_code = $3
		UNION ALL
		SELECT 'DEVOLUCOES', 'devolução(ões) de venda', count(*), count(*) FILTER (WHERE situacao IN ('RASCUNHO','EMITIDA'))
		  FROM devolucoes_venda WHERE enterprise_id = $1 AND customer_code = $3
		UNION ALL
		SELECT 'MATERIAL_CLIENTE', 'remessa(s) de material do cliente', count(*), count(*) FILTER (WHERE status::text IN ('ABERTA','PARCIAL'))
		  FROM customer_material_remittances WHERE enterprise_id = $1 AND customer_code = $3
		UNION ALL
		SELECT 'CONTRATOS', 'contrato(s) de venda', count(*), count(*) FILTER (WHERE situacao IN ('RASCUNHO','ATIVO'))
		  FROM contratos_venda WHERE enterprise_id = $1 AND customer_code = $3
		UNION ALL
		SELECT 'ORDENS_SERVICO', 'ordem(ns) de serviço', count(*), count(*) FILTER (WHERE situacao IN ('ABERTA','AGENDADA','EM_EXECUCAO'))
		  FROM ordens_servico WHERE enterprise_id = $1 AND customer_code = $3
		UNION ALL
		SELECT 'VENDAS_RECORRENTES', 'venda(s) recorrente(s)', count(*), count(*) FILTER (WHERE is_active)
		  FROM recurring_sales WHERE enterprise_code = $2 AND customer_code = $3
		UNION ALL
		SELECT 'ASSISTENCIA', 'chamado(s) de assistência técnica', count(*), 0
		  FROM technical_assistance_calls WHERE enterprise_code = $2 AND customer_code = $3
		UNION ALL
		SELECT 'SAC', 'atendimento(s) de SAC', count(*), 0
		  FROM consumer_service_calls WHERE enterprise_code = $2 AND customer_code = $3
		UNION ALL
		SELECT 'QUALIDADE', 'certificado(s)/capa(s) de qualidade', (
		    (SELECT count(*) FROM quality_certificates WHERE enterprise_id = $1 AND customer_code = $3) +
		    (SELECT count(*) FROM quality_capas WHERE enterprise_id = $1 AND customer_code = $3)), 0
		UNION ALL
		SELECT 'NUMEROS_SERIE', 'número(s) de série', count(*), 0 FROM numeros_serie WHERE enterprise_id = $1 AND customer_code = $3
		UNION ALL
		SELECT 'DEMANDAS', 'demanda(s) independente(s)', count(*), 0 FROM independent_demands WHERE enterprise_id = $1 AND customer_code = $3
		UNION ALL
		SELECT 'COMPRAS', 'pedido(s) de compra vinculados', count(*), 0 FROM purchase_orders WHERE enterprise_code = $2 AND customer_code = $3
		UNION ALL
		SELECT 'REGRAS', 'regra(s) comercial(is)/fiscal(is) específicas do cliente', (
		    (SELECT count(*) FROM commercial_policies WHERE enterprise_id = $1 AND customer_code = $3) +
		    (SELECT count(*) FROM icms_ipi_tax_params WHERE enterprise_id = $1 AND customer_code = $3) +
		    (SELECT count(*) FROM icms_reduction_substitutions WHERE enterprise_id = $1 AND customer_id IN ($3, $4)) +
		    (SELECT count(*) FROM restrictions WHERE enterprise_id = $1 AND customer_code = $3) +
		    (SELECT count(*) FROM lot_masks WHERE enterprise_id = $1 AND customer_code = $3) +
		    (SELECT count(*) FROM sla_assistencia WHERE enterprise_id = $1 AND customer_code = $3)), 0
		UNION ALL
		SELECT 'OUTROS', 'outro(s) vínculo(s) (metas, representante, expedição, margem, item exclusivo, parâmetros de orçamento)', (
		    (SELECT count(*) FROM sales_goal_balances WHERE enterprise_code = $2 AND customer_code = $3) +
		    (SELECT count(*) FROM sales_goal_group_customers g JOIN sales_goal_group_targets t ON t.id = g.group_goal_id
		      WHERE t.enterprise_code = $2 AND g.customer_code = $3) +
		    (SELECT count(*) FROM recurring_sales_adjustment_dates WHERE enterprise_code = $2 AND customer_code = $3) +
		    (SELECT count(*) FROM technical_assistance_return_notes n JOIN technical_assistance_calls tc ON tc.code = n.call_code
		      WHERE tc.enterprise_code = $2 AND n.customer_code = $3) +
		    (SELECT count(*) FROM items WHERE enterprise_id = $1 AND commercial_exclusive_customer_code = $3) +
		    (SELECT count(*) FROM sales_quotation_parameters WHERE enterprise_code = $2 AND final_consumer_customer_code = $3) +
		    (SELECT count(*) FROM representatives WHERE enterprise_code = $2 AND customer_code = $3) +
		    (SELECT count(*) FROM shipment_delivery_instructions WHERE enterprise_id = $1 AND customer_id IN ($3, $4)) +
		    (SELECT count(*) FROM contribution_margin WHERE enterprise_id = $1 AND customer_code = $3) +
		    (SELECT count(*) FROM delivery_tank_reservations WHERE enterprise_id = $1 AND customer_code = $3) +
		    (SELECT count(*) FROM consumer_service_customer_contacts WHERE enterprise_id = $1 AND customer_code = $3) +
		    (SELECT count(*) FROM technical_assistance_warranty_responsibles WHERE enterprise_code = $2 AND customer_code = $3)), 0`,
		empresa, empresaCode, code, id)
	if err != nil {
		return nil, fmt.Errorf("conferindo onde o cliente é usado: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (entity.CustomerUsage, error) {
		var u entity.CustomerUsage
		err := row.Scan(&u.Code, &u.Label, &u.Total, &u.Open)
		return u, err
	})
}

func (r *CustomerRepositorySQLC) SetCustomerActive(ctx context.Context, code int64, active bool) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	n, err := r.q.SetCustomerActive(ctx, sqlc.SetCustomerActiveParams{IsActive: active, Code: code, EnterpriseID: empresa})
	if err != nil {
		return fmt.Errorf("alterando a situação do cliente: %w", err)
	}
	if n == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("cliente %d não encontrado", code))
	}
	return nil
}

// DeleteCustomer trava o cliente, confere o uso com `check` (a regra é do
// domínio) e exclui, tudo na mesma transação: um pedido gravado no meio não
// fica apontando para cliente apagado.
func (r *CustomerRepositorySQLC) DeleteCustomer(ctx context.Context, code int64, check func([]entity.CustomerUsage) error) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		usages, err := customerUsage(ctx, tx, code, " FOR UPDATE")
		if err != nil {
			return err
		}
		if err := check(usages); err != nil {
			return err
		}
		n, err := r.q.WithTx(tx).DeleteCustomer(ctx, sqlc.DeleteCustomerParams{Code: code, EnterpriseID: empresa})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				return errorsuc.NewConflictError("o cliente ainda é referenciado por outro cadastro (" + pgErr.TableName + "); use Inativar")
			}
			return fmt.Errorf("excluindo o cliente: %w", err)
		}
		if n == 0 {
			return errorsuc.NewNotFoundError(fmt.Sprintf("cliente %d não encontrado", code))
		}
		return nil
	})
}

func (r *CustomerRepositorySQLC) CustomerOverview(ctx context.Context, code int64, group []int64, today time.Time) (*entity.CustomerOverview, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	empresaCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	if len(group) == 0 {
		group = []int64{code}
	}
	o := &entity.CustomerOverview{GroupMembers: len(group)}
	var totalEmpresa, abertoGrupo, pedidosGrupo decimal.Decimal
	err = r.pool.QueryRow(ctx, `
		WITH pedidos AS (
		    -- Mesmo valor da análise na confirmação (sales_order_uc.valorEmAberto):
		    -- só o saldo das linhas; o faturado já está nos títulos.
		    SELECT so.customer_code,
		           CASE WHEN EXISTS (SELECT 1 FROM sales_order_items i WHERE i.sales_order_code = so.code)
		                THEN COALESCE((SELECT sum(CASE WHEN i.total_net_with_ipi + i.total_st > 0 THEN i.total_net_with_ipi + i.total_st ELSE i.total_gross END
		                                          * GREATEST(i.requested_qty - i.attended_qty - i.cancelled_qty, 0) / i.requested_qty)
		                                 FROM sales_order_items i
		                                WHERE i.sales_order_code = so.code AND i.is_active AND i.requested_qty > 0), 0)
		                ELSE COALESCE(NULLIF(so.total_with_ipi_with_st, 0), NULLIF(so.total_net, 0), so.total_gross, 0) END AS valor
		      FROM sales_orders so WHERE so.enterprise_code = $2 AND so.customer_code = ANY($4::bigint[]) AND so.status IN ('P','A')
		), titulos AS (
		    SELECT cliente_id, data_vencimento, valor_bruto - COALESCE(valor_recebido, 0) AS saldo
		      FROM contas_receber
		     WHERE enterprise_id = $1 AND is_active AND status IN `+openReceivableStatuses+`
		)
		SELECT
		    (SELECT count(*) FROM pedidos WHERE customer_code = $3),
		    (SELECT COALESCE(sum(valor), 0) FROM pedidos WHERE customer_code = $3),
		    (SELECT COALESCE(sum(valor), 0) FROM pedidos),
		    (SELECT count(*) FROM production_orders po JOIN sales_orders so ON so.code = po.sales_order_code AND so.enterprise_code = $2
		      WHERE po.enterprise_id = $1 AND so.customer_code = $3 AND po.status IN ('OPEN','IN_PROGRESS')),
		    (SELECT count(*) FROM sales_quotations WHERE enterprise_code = $2 AND customer_code = $3 AND status IN ('R','P','A','OA','OF','V','OV')),
		    (SELECT COALESCE(sum(saldo), 0) FROM titulos WHERE cliente_id = $3),
		    (SELECT COALESCE(sum(saldo), 0) FROM titulos WHERE cliente_id = $3 AND data_vencimento < $5::date),
		    (SELECT COALESCE(max($5::date - data_vencimento), 0) FROM titulos WHERE cliente_id = $3 AND data_vencimento < $5::date)::int,
		    (SELECT COALESCE(sum(saldo), 0) FROM titulos WHERE cliente_id = ANY($4::bigint[])),
		    (SELECT COALESCE(sum(saldo), 0) FROM titulos),
		    (SELECT max(created_at)::date FROM sales_orders WHERE enterprise_code = $2 AND customer_code = $3 AND status <> 'CANCELLED'),
		    (SELECT max(data_emissao) FROM fiscal_exits WHERE enterprise_id = $1 AND customer_code = $3 AND status = 'AUTHORIZED'),
		    (SELECT count(*) FROM ordens_servico WHERE enterprise_id = $1 AND customer_code = $3 AND situacao IN ('ABERTA','AGENDADA','EM_EXECUCAO'))`,
		empresa, empresaCode, code, group, today,
	).Scan(&o.OpenOrders, &o.OpenOrdersValue, &pedidosGrupo, &o.ProductionOrders, &o.OpenQuotations,
		&o.OpenReceivables, &o.OverdueReceivables, &o.MaxOpenOverdueDays, &abertoGrupo, &totalEmpresa,
		&o.LastOrderDate, &o.LastInvoiceDate, &o.OpenServiceOrders)
	if err != nil {
		return nil, fmt.Errorf("lendo o painel do cliente: %w", err)
	}
	o.GroupExposure = abertoGrupo.Add(pedidosGrupo)
	if totalEmpresa.IsPositive() {
		o.ConcentrationPct = o.OpenReceivables.Div(totalEmpresa).Mul(decimal.NewFromInt(100)).Round(2)
	}
	o.CreditUsed = o.OpenReceivables.Add(o.OpenOrdersValue)

	// Faturamento autorizado dos últimos 12 meses, mês a mês (meses sem nota
	// aparecem zerados para o gráfico não pular).
	rows, err := r.pool.Query(ctx, `
		SELECT to_char(m, 'YYYY-MM'),
		       COALESCE((SELECT sum(valor_total) FROM fiscal_exits
		                  WHERE enterprise_id = $1 AND customer_code = $2 AND status = 'AUTHORIZED'
		                    AND COALESCE(tipo_documento, 1) = 1
		                    AND date_trunc('month', data_emissao) = m), 0)
		  FROM generate_series(date_trunc('month', $3::date) - interval '11 months', date_trunc('month', $3::date), interval '1 month') m
		 ORDER BY m`, empresa, code, today)
	if err != nil {
		return nil, fmt.Errorf("lendo o faturamento mensal: %w", err)
	}
	o.BillingByMonth, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (entity.MonthValue, error) {
		var m entity.MonthValue
		err := row.Scan(&m.Month, &m.Value)
		return m, err
	})
	if err != nil {
		return nil, err
	}
	for _, m := range o.BillingByMonth {
		o.Billed12m = o.Billed12m.Add(m.Value)
	}
	return o, nil
}

// supportTables: recurso da API → tabela, e as referências por CÓDIGO (sem
// FK) que impedem excluir. As referências por id têm FK e o próprio banco
// recusa (23503). Em cada referência, $1 é o código e $2 a empresa na
// convenção da tabela: enterprise_code quando byEnterpriseCode, senão
// enterprise_id.
type supportRef struct {
	sql              string
	byEnterpriseCode bool
}

var supportTables = map[string]struct {
	table, label string
	byCode       []supportRef
}{
	"regions": {"regions", "região", []supportRef{
		{`SELECT EXISTS (SELECT 1 FROM representative_regions WHERE region_code = $1 AND enterprise_code = $2)`, true}}},
	"market-segments": {"market_segments", "segmento", []supportRef{
		{`SELECT EXISTS (SELECT 1 FROM representative_segments WHERE market_segment_code = $1 AND enterprise_code = $2)`, true}}},
	"contact-types":  {"customer_contact_types", "tipo de contato", nil},
	"customer-types": {"customer_types", "tipo de cliente", nil},
	"carriers": {"carriers", "portador", []supportRef{
		{`SELECT EXISTS (SELECT 1 FROM sales_orders WHERE carrier_code = $1 AND enterprise_code = $2)`, true},
		{`SELECT EXISTS (SELECT 1 FROM sales_quotations WHERE carrier_code = $1 AND enterprise_code = $2)`, true}}},
	"carrier-groups": {"carrier_groups", "grupo de portadores", nil},
	"invoice-types": {"invoice_types", "tipo de nota", []supportRef{
		{`SELECT EXISTS (SELECT 1 FROM entry_operation_types WHERE invoice_type_code = $1 AND enterprise_id = $2)`, false},
		{`SELECT EXISTS (SELECT 1 FROM purchase_orders WHERE invoice_type_code = $1 AND enterprise_code = $2)`, true},
		{`SELECT EXISTS (SELECT 1 FROM purchase_order_items i JOIN purchase_orders o ON o.code = i.purchase_order_code
		   WHERE i.invoice_type_code = $1 AND o.enterprise_code = $2)`, true}}},
	"tax-types": {"tax_types", "tipo de imposto", []supportRef{
		{`SELECT EXISTS (SELECT 1 FROM sales_orders WHERE tax_type_code = $1 AND enterprise_code = $2)`, true}}},
}

// supportComSituacao: cadastros de apoio com coluna is_active.
var supportComSituacao = map[string]bool{
	"regions": true, "market-segments": true, "customer-types": true, "carriers": true, "invoice-types": true, "tax-types": true,
}

func (r *CustomerRepositorySQLC) SetSupportActive(ctx context.Context, resource string, code int64, active bool) error {
	spec, ok := supportTables[resource]
	if !ok || !supportComSituacao[resource] {
		return errorsuc.NewValidationError("este cadastro de apoio não tem inativação: " + resource)
	}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE `+spec.table+` SET is_active = $3 WHERE code = $1 AND enterprise_id = $2`, code, empresa, active)
	if err != nil {
		return fmt.Errorf("mudando a situação do(a) %s %d: %w", spec.label, code, err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("%s %d não encontrado(a)", spec.label, code))
	}
	return nil
}

func (r *CustomerRepositorySQLC) DeleteSupport(ctx context.Context, resource string, code int64) error {
	spec, ok := supportTables[resource]
	if !ok {
		return errorsuc.NewValidationError("cadastro de apoio desconhecido: " + resource)
	}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	empresaCode, err := tenant.Code(ctx)
	if err != nil {
		return err
	}
	for _, ref := range spec.byCode {
		tenantArg := empresa
		if ref.byEnterpriseCode {
			tenantArg = empresaCode
		}
		var usado bool
		if err := r.pool.QueryRow(ctx, ref.sql, code, tenantArg).Scan(&usado); err != nil {
			return fmt.Errorf("conferindo o uso do(a) %s %d: %w", spec.label, code, err)
		}
		if usado {
			return errorsuc.NewConflictError(fmt.Sprintf("o(a) %s %d já foi usado(a) em documentos; inative em vez de excluir", spec.label, code))
		}
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM `+spec.table+` WHERE code = $1 AND enterprise_id = $2`, code, empresa)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return errorsuc.NewConflictError(fmt.Sprintf("o(a) %s %d está em uso (%s); inative em vez de excluir", spec.label, code, pgErr.TableName))
		}
		return fmt.Errorf("excluindo o(a) %s %d: %w", spec.label, code, err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("%s %d não encontrado(a)", spec.label, code))
	}
	return nil
}
