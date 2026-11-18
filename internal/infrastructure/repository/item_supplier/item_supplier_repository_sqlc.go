package item_supplier

import (
	"context"
	"errors"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/item_supplier/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/item_supplier/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	itemrepository "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"strings"
	"time"
)

type ItemSupplierRepositorySQLC struct{ pool *pgxpool.Pool }

func New(_ *sqlc.Queries, pool *pgxpool.Pool) domainrepo.ItemSupplierRepository {
	return &ItemSupplierRepositorySQLC{pool: pool}
}

type scanner interface{ Scan(...any) error }

const cols = `s.id,s.enterprise_id,s.item_code,s.supplier_code,s.mask,s.ranking,s.supplier_item_code,s.supplier_description,s.uom,s.xml_uom,s.conversion_factor,s.package_quantity,s.is_preferred,s.supplier_uf,s.classification_id,s.classification_date,s.classification_grade,s.direct_billing,s.third_party_order,s.ignore_avg_cost_addition,s.ecommerce,s.barcode,s.notes,s.valid_until,s.lead_time_days,s.is_active,s.created_at,s.created_by,s.updated_at,s.valid_from,s.min_order_qty,s.order_multiple,s.quota_percent,s.blocked,s.blocked_reason,s.blocked_at,s.blocked_by`
const returningCols = `id,enterprise_id,item_code,supplier_code,mask,ranking,supplier_item_code,supplier_description,uom,xml_uom,conversion_factor,package_quantity,is_preferred,supplier_uf,classification_id,classification_date,classification_grade,direct_billing,third_party_order,ignore_avg_cost_addition,ecommerce,barcode,notes,valid_until,lead_time_days,is_active,created_at,created_by,updated_at,valid_from,min_order_qty,order_multiple,quota_percent,blocked,blocked_reason,blocked_at,blocked_by`

func scan(s scanner) (*entity.ItemPreferredSupplier, error) {
	x := &entity.ItemPreferredSupplier{}
	err := s.Scan(&x.ID, &x.EnterpriseID, &x.ItemCode, &x.SupplierCode, &x.Mask, &x.Ranking, &x.SupplierItemCode, &x.SupplierDescription, &x.UOM, &x.XMLUOM, &x.ConversionFactor, &x.PackageQuantity, &x.IsPreferred, &x.SupplierUF, &x.ClassificationID, &x.ClassificationDate, &x.ClassificationGrade, &x.DirectBilling, &x.ThirdPartyOrder, &x.IgnoreAvgCostAddition, &x.Ecommerce, &x.Barcode, &x.Notes, &x.ValidUntil, &x.LeadTimeDays, &x.IsActive, &x.CreatedAt, &x.CreatedBy, &x.UpdatedAt, &x.ValidFrom, &x.MinOrderQty, &x.OrderMultiple, &x.QuotaPercent, &x.Blocked, &x.BlockedReason, &x.BlockedAt, &x.BlockedBy)
	return x, err
}
func (r *ItemSupplierRepositorySQLC) Upsert(ctx context.Context, s *entity.ItemPreferredSupplier) (*entity.ItemPreferredSupplier, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	// Código de item e de fornecedor são únicos no banco inteiro: sem esta
	// conferência, o vínculo apontava para o item ou o fornecedor de outra empresa.
	var doEmpresa bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM items WHERE code=$2 AND enterprise_id=$1)
		AND EXISTS(SELECT 1 FROM suppliers WHERE code=$3 AND enterprise_id=$1)`, s.EnterpriseID, s.ItemCode, s.SupplierCode).Scan(&doEmpresa); err != nil {
		return nil, err
	}
	if !doEmpresa {
		return nil, errorsuc.NewNotFoundError("item ou fornecedor não encontrado na empresa")
	}
	if s.SupplierItemCode != nil {
		if err = itemrepository.ConferirCodigoDoFornecedor(ctx, tx, s.EnterpriseID, s.ItemCode, *s.SupplierItemCode); err != nil {
			return nil, err
		}
	}
	if s.SupplierUF == nil {
		// UF do endereço do fornecedor da mesma empresa; sem endereço, fica vazia.
		if err = tx.QueryRow(ctx, `SELECT a.uf FROM suppliers p JOIN supplier_addresses a ON a.supplier_id=p.id WHERE p.code=$1 AND p.enterprise_id=$2 ORDER BY a.is_default DESC,a.id LIMIT 1`, s.SupplierCode, s.EnterpriseID).Scan(&s.SupplierUF); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	if s.ClassificationID == nil && s.ClassificationDate == nil {
		if err = tx.QueryRow(ctx, `SELECT classification_id,classification_date,classification_grade FROM item_preferred_suppliers WHERE enterprise_id=$1 AND item_code=$2 AND supplier_code=$3 AND is_active LIMIT 1`, s.EnterpriseID, s.ItemCode, s.SupplierCode).Scan(&s.ClassificationID, &s.ClassificationDate, &s.ClassificationGrade); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	if s.IsPreferred {
		if _, err = tx.Exec(ctx, `UPDATE item_preferred_suppliers SET is_preferred=FALSE,ranking=GREATEST(ranking,2),updated_at=NOW() WHERE enterprise_id=$1 AND item_code=$2 AND supplier_code<>$3 AND is_active`, s.EnterpriseID, s.ItemCode, s.SupplierCode); err != nil {
			return nil, err
		}
	}
	if s.QuotaPercent != nil {
		// A soma das cotas do item (na mesma máscara) não passa de 100%. O
		// bloqueado conta: a cota dele fica reservada, senão desbloquear
		// passaria de 100%. A trava consultiva do item serializa dois cadastros
		// simultâneos — FOR UPDATE nas linhas não pega um vínculo NOVO.
		if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('cota_fornecedor_item:' || $1::bigint::text || ':' || $2::bigint::text, 0))`, s.EnterpriseID, s.ItemCode); err != nil {
			return nil, err
		}
		// A necessidade de uma máscara reparte entre os fornecedores gerais
		// (máscara vazia) e os daquela máscara: geral + cada máscara ≤ 100%.
		// Cadastro geral confere contra a máscara mais carregada.
		var outras decimal.Decimal
		if err = tx.QueryRow(ctx, `WITH c AS (
			SELECT mask, quota_percent FROM item_preferred_suppliers
			 WHERE enterprise_id=$1 AND item_code=$2 AND is_active AND quota_percent IS NOT NULL
			   AND NOT (supplier_code=$3 AND mask=$4))
			SELECT COALESCE((SELECT SUM(quota_percent) FROM c WHERE mask=''),0)
			     + CASE WHEN $4='' THEN COALESCE((SELECT MAX(t) FROM (SELECT SUM(quota_percent) t FROM c WHERE mask<>'' GROUP BY mask) m),0)
			            ELSE COALESCE((SELECT SUM(quota_percent) FROM c WHERE mask=$4),0) END`,
			s.EnterpriseID, s.ItemCode, s.SupplierCode, s.Mask).Scan(&outras); err != nil {
			return nil, err
		}
		if outras.Add(*s.QuotaPercent).GreaterThan(entity.CotaMaxima) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("a soma das cotas do item passaria de 100%% (os outros fornecedores já têm %s%%)", outras.StringFixed(2)))
		}
	}
	// Código, descrição e UM do fornecedor: a entrada de NF-e aprende sozinha.
	// Regravar o vínculo sem eles (a tela não os edita) não pode apagá-los —
	// limpar é ação própria (ClearLearned).
	row := tx.QueryRow(ctx, `INSERT INTO item_preferred_suppliers(enterprise_id,item_code,supplier_code,mask,ranking,supplier_item_code,supplier_description,uom,xml_uom,conversion_factor,package_quantity,is_preferred,supplier_uf,classification_id,classification_date,classification_grade,direct_billing,third_party_order,ignore_avg_cost_addition,ecommerce,barcode,notes,valid_until,lead_time_days,created_by,valid_from,min_order_qty,order_multiple,quota_percent,quota_since)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,
		       CASE WHEN $29::numeric IS NULL THEN NULL ELSE CURRENT_DATE END)
		ON CONFLICT(enterprise_id,item_code,supplier_code,mask) WHERE enterprise_id IS NOT NULL DO UPDATE SET ranking=EXCLUDED.ranking,
		supplier_item_code=COALESCE(EXCLUDED.supplier_item_code,item_preferred_suppliers.supplier_item_code),
		supplier_description=COALESCE(EXCLUDED.supplier_description,item_preferred_suppliers.supplier_description),
		uom=COALESCE(EXCLUDED.uom,item_preferred_suppliers.uom),
		xml_uom=EXCLUDED.xml_uom,conversion_factor=EXCLUDED.conversion_factor,package_quantity=EXCLUDED.package_quantity,is_preferred=EXCLUDED.is_preferred,supplier_uf=EXCLUDED.supplier_uf,classification_id=EXCLUDED.classification_id,classification_date=EXCLUDED.classification_date,classification_grade=EXCLUDED.classification_grade,direct_billing=EXCLUDED.direct_billing,third_party_order=EXCLUDED.third_party_order,ignore_avg_cost_addition=EXCLUDED.ignore_avg_cost_addition,ecommerce=EXCLUDED.ecommerce,barcode=EXCLUDED.barcode,notes=EXCLUDED.notes,valid_until=EXCLUDED.valid_until,lead_time_days=EXCLUDED.lead_time_days,
		valid_from=EXCLUDED.valid_from,min_order_qty=EXCLUDED.min_order_qty,order_multiple=EXCLUDED.order_multiple,quota_percent=EXCLUDED.quota_percent,
		quota_since=CASE WHEN EXCLUDED.quota_percent IS NOT DISTINCT FROM item_preferred_suppliers.quota_percent THEN item_preferred_suppliers.quota_since ELSE EXCLUDED.quota_since END,
		is_active=TRUE,updated_at=NOW() RETURNING `+returningCols, s.EnterpriseID, s.ItemCode, s.SupplierCode, s.Mask, s.Ranking, s.SupplierItemCode, s.SupplierDescription, s.UOM, s.XMLUOM, s.ConversionFactor, s.PackageQuantity, s.IsPreferred, s.SupplierUF, s.ClassificationID, s.ClassificationDate, s.ClassificationGrade, s.DirectBilling, s.ThirdPartyOrder, s.IgnoreAvgCostAddition, s.Ecommerce, s.Barcode, s.Notes, s.ValidUntil, s.LeadTimeDays, s.CreatedBy, s.ValidFrom, s.MinOrderQty, s.OrderMultiple, s.QuotaPercent)
	saved, err := scan(row)
	if err != nil {
		return nil, fmt.Errorf("upserting item supplier: %w", err)
	}
	if _, err = tx.Exec(ctx, `UPDATE item_preferred_suppliers SET classification_id=$4,classification_date=$5,classification_grade=$6,updated_at=NOW() WHERE enterprise_id=$1 AND item_code=$2 AND supplier_code=$3 AND id<>$7`, s.EnterpriseID, s.ItemCode, s.SupplierCode, s.ClassificationID, s.ClassificationDate, s.ClassificationGrade, saved.ID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return saved, nil
}
func (r *ItemSupplierRepositorySQLC) list(ctx context.Context, q string, args ...any) ([]*entity.ItemPreferredSupplier, error) {
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*entity.ItemPreferredSupplier{}
	for rows.Next() {
		x, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *ItemSupplierRepositorySQLC) ListByItem(ctx context.Context, e, item int64) ([]*entity.ItemPreferredSupplier, error) {
	// Todos os vínculos ativos, inclusive bloqueados e fora da vigência: a tela
	// mostra a situação de cada um (sumir com o vencido escondia por que o item
	// "não tem fornecedor").
	return r.list(ctx, `SELECT `+cols+` FROM item_preferred_suppliers s WHERE s.enterprise_id=$1 AND s.item_code=$2 AND s.is_active ORDER BY s.is_preferred DESC,s.ranking,s.supplier_code,s.mask`, e, item)
}
func (r *ItemSupplierRepositorySQLC) ListBySupplier(ctx context.Context, e, supplier int64) ([]*entity.ItemPreferredSupplier, error) {
	return r.list(ctx, `SELECT `+cols+` FROM item_preferred_suppliers s WHERE s.enterprise_id=$1 AND s.supplier_code=$2 AND s.is_active AND (s.valid_until IS NULL OR s.valid_until>=CURRENT_DATE) ORDER BY s.item_code,s.mask`, e, supplier)
}
func (r *ItemSupplierRepositorySQLC) SearchExternal(ctx context.Context, e, supplier int64, term string) ([]*entity.ItemPreferredSupplier, error) {
	term = strings.TrimSpace(term)
	return r.list(ctx, `SELECT `+cols+` FROM item_preferred_suppliers s WHERE s.enterprise_id=$1 AND s.supplier_code=$2 AND s.is_active AND (s.valid_until IS NULL OR s.valid_until>=CURRENT_DATE) AND (upper(btrim(COALESCE(s.supplier_item_code,'')))=upper(btrim($3)) OR s.supplier_description ILIKE '%'||$3||'%') ORDER BY CASE WHEN upper(btrim(COALESCE(s.supplier_item_code,'')))=upper(btrim($3)) THEN 0 ELSE 1 END,s.is_preferred DESC,s.ranking,s.id LIMIT 20`, e, supplier, term)
}
func (r *ItemSupplierRepositorySQLC) GetPreferred(ctx context.Context, e, item int64) (*entity.ItemPreferredSupplier, error) {
	// O marcado como preferencial; sem ele, o de melhor ranking. Só fonte
	// válida hoje: ativa, desbloqueada e vigente. A data vai como parâmetro.
	hoje := time.Now()
	x, err := scan(r.pool.QueryRow(ctx, `SELECT `+cols+` FROM item_preferred_suppliers s WHERE s.enterprise_id=$1 AND s.item_code=$2 AND s.is_active AND NOT s.blocked
		AND (s.valid_from IS NULL OR s.valid_from<=$3::date) AND (s.valid_until IS NULL OR s.valid_until>=$3::date)
		ORDER BY s.is_preferred DESC,s.ranking,s.supplier_code LIMIT 1`, e, item, time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, time.UTC)))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return x, err
}
func (r *ItemSupplierRepositorySQLC) Delete(ctx context.Context, e, id int64) error {
	tag, err := r.pool.Exec(ctx, `UPDATE item_preferred_suppliers SET is_active=FALSE,updated_at=NOW() WHERE enterprise_id=$1 AND id=$2`, e, id)
	if err == nil && tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
func (r *ItemSupplierRepositorySQLC) ItemAllowsConversionFactor(ctx context.Context, item int64) (bool, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var ok bool
	err = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM items WHERE enterprise_id=$1 AND code=$2 AND nature=0)`, e, item).Scan(&ok)
	return ok, err
}
func (r *ItemSupplierRepositorySQLC) CreateQualityReport(ctx context.Context, q *entity.QualityReport) (*entity.QualityReport, error) {
	x := &entity.QualityReport{}
	err := r.pool.QueryRow(ctx, `INSERT INTO item_supplier_quality_reports(enterprise_id,item_supplier_id,registered_on,status,report_file_name,report_content_type,report_content,notes,created_by) SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9 WHERE EXISTS(SELECT 1 FROM item_preferred_suppliers WHERE enterprise_id=$1 AND id=$2) RETURNING id,enterprise_id,item_supplier_id,registered_on,status,report_file_name,report_content_type,report_content,notes,created_at,created_by`, q.EnterpriseID, q.ItemSupplierID, q.RegisteredOn, q.Status, q.FileName, q.ContentType, q.Content, q.Notes, q.CreatedBy).Scan(&x.ID, &x.EnterpriseID, &x.ItemSupplierID, &x.RegisteredOn, &x.Status, &x.FileName, &x.ContentType, &x.Content, &x.Notes, &x.CreatedAt, &x.CreatedBy)
	if err != nil {
		return nil, err
	}
	return x, nil
}
func (r *ItemSupplierRepositorySQLC) ListQualityReports(ctx context.Context, e, link int64) ([]*entity.QualityReport, error) {
	rows, err := r.pool.Query(ctx, `SELECT q.id,q.enterprise_id,q.item_supplier_id,q.registered_on,q.status,q.report_file_name,q.report_content_type,q.report_content,q.notes,q.created_at,q.created_by FROM item_supplier_quality_reports q JOIN item_preferred_suppliers s ON s.id=q.item_supplier_id WHERE q.enterprise_id=$1 AND q.item_supplier_id=$2 AND s.enterprise_id=$1 ORDER BY q.registered_on DESC,q.id DESC`, e, link)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*entity.QualityReport{}
	for rows.Next() {
		x := &entity.QualityReport{}
		if err = rows.Scan(&x.ID, &x.EnterpriseID, &x.ItemSupplierID, &x.RegisteredOn, &x.Status, &x.FileName, &x.ContentType, &x.Content, &x.Notes, &x.CreatedAt, &x.CreatedBy); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *ItemSupplierRepositorySQLC) GetQualityReport(ctx context.Context, e, id int64) (*entity.QualityReport, error) {
	x := &entity.QualityReport{}
	err := r.pool.QueryRow(ctx, `SELECT q.id,q.enterprise_id,q.item_supplier_id,q.registered_on,q.status,q.report_file_name,q.report_content_type,q.report_content,q.notes,q.created_at,q.created_by FROM item_supplier_quality_reports q JOIN item_preferred_suppliers s ON s.id=q.item_supplier_id AND s.enterprise_id=q.enterprise_id WHERE q.enterprise_id=$1 AND q.id=$2`, e, id).Scan(&x.ID, &x.EnterpriseID, &x.ItemSupplierID, &x.RegisteredOn, &x.Status, &x.FileName, &x.ContentType, &x.Content, &x.Notes, &x.CreatedAt, &x.CreatedBy)
	if err != nil {
		return nil, err
	}
	return x, nil
}

// SetBlocked bloqueia ou desbloqueia o vínculo e grava o histórico, na mesma
// transação. O vínculo tem de ser da empresa; repetir o estado é recusado.
func (r *ItemSupplierRepositorySQLC) SetBlocked(ctx context.Context, ev *entity.BlockEvent) (*entity.ItemPreferredSupplier, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	bloquear := ev.Action == entity.AcaoBloqueio
	var atual bool
	if err = tx.QueryRow(ctx, `SELECT blocked FROM item_preferred_suppliers WHERE enterprise_id=$1 AND id=$2 AND is_active FOR UPDATE`, ev.EnterpriseID, ev.ItemSupplierID).Scan(&atual); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError("vínculo de fornecedor não encontrado na empresa")
		}
		return nil, err
	}
	if atual == bloquear {
		if bloquear {
			return nil, errorsuc.NewValidationError("o vínculo já está bloqueado")
		}
		return nil, errorsuc.NewValidationError("o vínculo não está bloqueado")
	}
	var motivo *string
	var quem any
	var quando any
	if bloquear {
		motivo, quem, quando = &ev.Reason, ev.ActorID, time.Now()
	}
	saved, err := scan(tx.QueryRow(ctx, `UPDATE item_preferred_suppliers SET blocked=$3,blocked_reason=$4,blocked_by=$5,blocked_at=$6,updated_at=NOW()
		WHERE enterprise_id=$1 AND id=$2 RETURNING `+returningCols, ev.EnterpriseID, ev.ItemSupplierID, bloquear, motivo, quem, quando))
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO item_supplier_block_history(enterprise_id,item_supplier_id,action,reason,actor_id)
		SELECT $1,$2,$3,$4,$5 WHERE EXISTS(SELECT 1 FROM item_preferred_suppliers WHERE enterprise_id=$1 AND id=$2)`,
		ev.EnterpriseID, ev.ItemSupplierID, ev.Action, ev.Reason, ev.ActorID); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return saved, nil
}

func (r *ItemSupplierRepositorySQLC) ListBlockHistory(ctx context.Context, e, link int64) ([]*entity.BlockEvent, error) {
	rows, err := r.pool.Query(ctx, `SELECT h.id,h.enterprise_id,h.item_supplier_id,h.action,h.reason,h.actor_id,h.created_at
		FROM item_supplier_block_history h JOIN item_preferred_suppliers s ON s.id=h.item_supplier_id AND s.enterprise_id=h.enterprise_id
		WHERE h.enterprise_id=$1 AND h.item_supplier_id=$2 ORDER BY h.created_at DESC,h.id DESC`, e, link)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*entity.BlockEvent{}
	for rows.Next() {
		x := &entity.BlockEvent{}
		if err := rows.Scan(&x.ID, &x.EnterpriseID, &x.ItemSupplierID, &x.Action, &x.Reason, &x.ActorID, &x.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// ClearLearned apaga o código e a descrição do item no fornecedor — o que a
// entrada de NF-e aprendeu. Serve para desfazer uma ligação errada: a próxima
// nota volta a pedir a escolha do item.
func (r *ItemSupplierRepositorySQLC) ClearLearned(ctx context.Context, e, id int64) (*entity.ItemPreferredSupplier, error) {
	x, err := scan(r.pool.QueryRow(ctx, `UPDATE item_preferred_suppliers SET supplier_item_code=NULL,supplier_description=NULL,updated_at=NOW()
		WHERE enterprise_id=$1 AND id=$2 AND is_active RETURNING `+returningCols, e, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError("vínculo de fornecedor não encontrado na empresa")
	}
	return x, err
}

// FonteDoItem segue as mesmas regras da trava do banco
// (erp_fonte_de_suprimento_valida): ativa, desbloqueada, vigente, máscara vazia
// ou igual. Empresa que desligou a exigência aceita o fornecedor informado.
// A data é a do BANCO (CURRENT_DATE), igual à da trava: prever a trava com o
// relógio do processo divergiria no último dia da vigência se os fusos diferirem.
func (r *ItemSupplierRepositorySQLC) FonteDoItem(ctx context.Context, e, item int64, mask string, informado *int64) (int64, bool, error) {
	var fornecedor int64
	err := r.pool.QueryRow(ctx, `SELECT s.supplier_code FROM item_preferred_suppliers s
		JOIN items i ON i.code = s.item_code AND i.enterprise_id = s.enterprise_id
		WHERE s.enterprise_id=$1 AND s.item_code=$2 AND s.is_active AND NOT s.blocked
		  AND (s.mask='' OR s.mask=$3)
		  AND (s.valid_from IS NULL OR s.valid_from<=CURRENT_DATE) AND (s.valid_until IS NULL OR s.valid_until>=CURRENT_DATE)
		  AND ($4::bigint IS NULL OR s.supplier_code=$4)
		ORDER BY s.is_preferred DESC, (s.mask=$3) DESC, s.ranking, s.supplier_code LIMIT 1`, e, item, mask, informado).Scan(&fornecedor)
	if err == nil {
		return fornecedor, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, false, err
	}
	if informado != nil {
		var exige bool
		if err = r.pool.QueryRow(ctx, `SELECT erp_exige_fonte_de_suprimento($1)`, e).Scan(&exige); err != nil {
			return 0, false, err
		}
		if !exige {
			return *informado, true, nil
		}
	}
	return 0, false, nil
}
