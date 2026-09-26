// Package shipping_carrier grava o cadastro de transportadora.
//
// Empresa por `enterprise_id`, igual ao pai (`suppliers`). Toda consulta filtra
// pela empresa da sessão: o cadastro de frete de uma empresa não aparece na
// outra.
package shipping_carrier

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/shipping_carrier/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/shipping_carrier/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const colunasCapa = `c.id, c.enterprise_id, c.supplier_code, COALESCE(s.name,''), COALESCE(s.document_number,''),
c.antt_rntrc, c.antt_expiry, c.shipper_type::text, c.modal::text, c.issues_cte, c.default_freight_type,
c.freight_min_value, c.freight_kg_rate, c.freight_pct_value, c.gris_pct, c.toll_per_100kg,
c.insurance_company, c.insurance_policy, c.insurance_expiry, c.insurance_coverage,
c.average_lead_days, c.tracking_url, c.contact_name, c.contact_phone, c.contact_email,
c.notes, c.is_active, c.created_at, c.updated_at`

func scanCapa(row pgx.Row) (*entity.Transportadora, error) {
	var t entity.Transportadora
	var tipo *string
	var modal string
	if err := row.Scan(&t.ID, &t.EnterpriseID, &t.SupplierCode, &t.SupplierName, &t.SupplierDocument,
		&t.ANTTRNTRC, &t.ANTTExpiry, &tipo, &modal, &t.IssuesCTe, &t.DefaultFreightType,
		&t.FreightMinValue, &t.FreightKgRate, &t.FreightPctValue, &t.GrisPct, &t.TollPer100Kg,
		&t.InsuranceCompany, &t.InsurancePolicy, &t.InsuranceExpiry, &t.InsuranceCoverage,
		&t.AverageLeadDays, &t.TrackingURL, &t.ContactName, &t.ContactPhone, &t.ContactEmail,
		&t.Notes, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	t.Modal = entity.Modal(modal)
	if tipo != nil && *tipo != "" {
		v := entity.TipoTransportador(*tipo)
		t.ShipperType = &v
	}
	return &t, nil
}

func (r *Repository) Listar(ctx context.Context, f domrepo.Filtro) ([]*entity.Transportadora, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	condicoes := []string{"c.enterprise_id = $1"}
	args := []any{enterpriseID}
	if f.SomenteAtivas {
		condicoes = append(condicoes, "c.is_active = TRUE")
	}
	if busca := strings.TrimSpace(f.Busca); busca != "" {
		args = append(args, "%"+strings.ToLower(busca)+"%")
		condicoes = append(condicoes, fmt.Sprintf("(LOWER(s.name) LIKE $%d OR s.document_number LIKE $%d OR c.antt_rntrc LIKE $%d)", len(args), len(args), len(args)))
	}
	if modal := strings.ToUpper(strings.TrimSpace(f.Modal)); modal != "" {
		args = append(args, modal)
		condicoes = append(condicoes, fmt.Sprintf("c.modal::text = $%d", len(args)))
	}
	if uf := strings.ToUpper(strings.TrimSpace(f.UF)); uf != "" {
		args = append(args, uf)
		condicoes = append(condicoes, fmt.Sprintf(`EXISTS (SELECT 1 FROM public.shipping_carrier_service_areas a
			WHERE a.carrier_id = c.id AND a.is_active = TRUE AND a.state = $%d)`, len(args)))
	}
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
SELECT %s FROM public.shipping_carriers c
LEFT JOIN public.suppliers s ON s.code = c.supplier_code
WHERE %s
ORDER BY s.name`, colunasCapa, strings.Join(condicoes, " AND ")), args...)
	if err != nil {
		return nil, fmt.Errorf("listar transportadoras: %w", err)
	}
	defer rows.Close()
	var out []*entity.Transportadora
	var ids []int64
	for rows.Next() {
		t, err := scanCapa(rows)
		if err != nil {
			return nil, fmt.Errorf("ler transportadora: %w", err)
		}
		out = append(out, t)
		ids = append(ids, t.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// A lista traz frota e regiões numa consulta só: sem isso a tela pediria
	// N+1 requisições para mostrar o alerta de cada linha.
	if err := r.carregarFilhos(ctx, enterpriseID, out, ids); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) carregarFilhos(ctx context.Context, enterpriseID int64, lista []*entity.Transportadora, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	porID := map[int64]*entity.Transportadora{}
	for _, t := range lista {
		porID[t.ID] = t
	}
	rows, err := r.pool.Query(ctx, `
SELECT id, enterprise_id, carrier_id, plate, description, vehicle_type, axles, capacity_kg, capacity_m3,
       antt_owner, driver_name, driver_document, driver_license, is_active, created_at, updated_at
FROM public.shipping_carrier_vehicles
WHERE carrier_id = ANY($1) AND enterprise_id = $2
ORDER BY plate`, ids, enterpriseID)
	if err != nil {
		return fmt.Errorf("listar frota: %w", err)
	}
	for rows.Next() {
		var v entity.Veiculo
		if err := rows.Scan(&v.ID, &v.EnterpriseID, &v.CarrierID, &v.Plate, &v.Description, &v.VehicleType,
			&v.Axles, &v.CapacityKg, &v.CapacityM3, &v.ANTTOwner, &v.DriverName, &v.DriverDocument,
			&v.DriverLicense, &v.IsActive, &v.CreatedAt, &v.UpdatedAt); err != nil {
			rows.Close()
			return fmt.Errorf("ler veículo: %w", err)
		}
		if t := porID[v.CarrierID]; t != nil {
			t.Vehicles = append(t.Vehicles, &v)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	areas, err := r.pool.Query(ctx, `
SELECT id, enterprise_id, carrier_id, state, city, postal_code_from, postal_code_to, lead_days,
       min_value, kg_rate, pct_value, is_active, created_at, updated_at
FROM public.shipping_carrier_service_areas
WHERE carrier_id = ANY($1) AND enterprise_id = $2
ORDER BY state NULLS LAST, postal_code_from NULLS LAST`, ids, enterpriseID)
	if err != nil {
		return fmt.Errorf("listar regiões atendidas: %w", err)
	}
	defer areas.Close()
	for areas.Next() {
		var a entity.RegiaoAtendida
		if err := areas.Scan(&a.ID, &a.EnterpriseID, &a.CarrierID, &a.State, &a.City, &a.PostalCodeFrom,
			&a.PostalCodeTo, &a.LeadDays, &a.MinValue, &a.KgRate, &a.PctValue, &a.IsActive,
			&a.CreatedAt, &a.UpdatedAt); err != nil {
			return fmt.Errorf("ler região atendida: %w", err)
		}
		if t := porID[a.CarrierID]; t != nil {
			t.ServiceAreas = append(t.ServiceAreas, &a)
		}
	}
	return areas.Err()
}

func (r *Repository) Obter(ctx context.Context, id int64) (*entity.Transportadora, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	t, err := scanCapa(r.pool.QueryRow(ctx, fmt.Sprintf(`
SELECT %s FROM public.shipping_carriers c
LEFT JOIN public.suppliers s ON s.code = c.supplier_code
WHERE c.id = $1 AND c.enterprise_id = $2`, colunasCapa), id, enterpriseID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("transportadora %d não encontrada", id))
	}
	if err != nil {
		return nil, fmt.Errorf("obter transportadora: %w", err)
	}
	if err := r.carregarFilhos(ctx, enterpriseID, []*entity.Transportadora{t}, []int64{t.ID}); err != nil {
		return nil, err
	}
	return t, nil
}

func (r *Repository) ObterPorFornecedor(ctx context.Context, supplierCode int64) (*entity.Transportadora, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	t, err := scanCapa(r.pool.QueryRow(ctx, fmt.Sprintf(`
SELECT %s FROM public.shipping_carriers c
LEFT JOIN public.suppliers s ON s.code = c.supplier_code
WHERE c.supplier_code = $1 AND c.enterprise_id = $2`, colunasCapa), supplierCode, enterpriseID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("o fornecedor %d não tem perfil de transportadora", supplierCode))
	}
	if err != nil {
		return nil, fmt.Errorf("obter transportadora por fornecedor: %w", err)
	}
	if err := r.carregarFilhos(ctx, enterpriseID, []*entity.Transportadora{t}, []int64{t.ID}); err != nil {
		return nil, err
	}
	return t, nil
}

func (r *Repository) FornecedorExiste(ctx context.Context, supplierCode int64) (string, string, bool, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return "", "", false, err
	}
	var nome, documento string
	var ativo bool
	err = r.pool.QueryRow(ctx, `
SELECT COALESCE(name,''), COALESCE(document_number,''), is_active
FROM public.suppliers WHERE code = $1 AND enterprise_id = $2`, supplierCode, enterpriseID).
		Scan(&nome, &documento, &ativo)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, fmt.Errorf("conferir fornecedor da transportadora: %w", err)
	}
	return nome, documento, ativo, nil
}

// Salvar grava capa, frota e regiões de uma vez. Frota e regiões são
// substituídas inteiras: é assim que a tela envia, e meia frota gravada é pior
// que nenhuma.
func (r *Repository) Salvar(ctx context.Context, t *entity.Transportadora) (*entity.Transportadora, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var id int64
	if t.ID > 0 {
		err = tx.QueryRow(ctx, `
UPDATE public.shipping_carriers SET
 supplier_code=$3, antt_rntrc=$4, antt_expiry=$5, shipper_type=$6::carrier_shipper_type_enum,
 modal=$7::carrier_modal_enum, issues_cte=$8, default_freight_type=$9,
 freight_min_value=$10, freight_kg_rate=$11, freight_pct_value=$12, gris_pct=$13, toll_per_100kg=$14,
 insurance_company=$15, insurance_policy=$16, insurance_expiry=$17, insurance_coverage=$18,
 average_lead_days=$19, tracking_url=$20, contact_name=$21, contact_phone=$22, contact_email=$23,
 notes=$24, is_active=$25, updated_at=NOW()
WHERE id=$1 AND enterprise_id=$2
RETURNING id`,
			t.ID, enterpriseID, t.SupplierCode, t.ANTTRNTRC, t.ANTTExpiry, tipoTexto(t.ShipperType),
			string(t.Modal), t.IssuesCTe, t.DefaultFreightType,
			t.FreightMinValue, t.FreightKgRate, t.FreightPctValue, t.GrisPct, t.TollPer100Kg,
			t.InsuranceCompany, t.InsurancePolicy, t.InsuranceExpiry, t.InsuranceCoverage,
			t.AverageLeadDays, t.TrackingURL, t.ContactName, t.ContactPhone, t.ContactEmail,
			t.Notes, t.IsActive).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("transportadora %d não encontrada", t.ID))
		}
	} else {
		err = tx.QueryRow(ctx, `
INSERT INTO public.shipping_carriers
 (enterprise_id, supplier_code, antt_rntrc, antt_expiry, shipper_type, modal, issues_cte,
  default_freight_type, freight_min_value, freight_kg_rate, freight_pct_value, gris_pct, toll_per_100kg,
  insurance_company, insurance_policy, insurance_expiry, insurance_coverage, average_lead_days,
  tracking_url, contact_name, contact_phone, contact_email, notes, is_active)
VALUES ($1,$2,$3,$4,$5::carrier_shipper_type_enum,$6::carrier_modal_enum,$7,$8,$9,$10,$11,$12,$13,
        $14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24)
RETURNING id`,
			enterpriseID, t.SupplierCode, t.ANTTRNTRC, t.ANTTExpiry, tipoTexto(t.ShipperType),
			string(t.Modal), t.IssuesCTe, t.DefaultFreightType,
			t.FreightMinValue, t.FreightKgRate, t.FreightPctValue, t.GrisPct, t.TollPer100Kg,
			t.InsuranceCompany, t.InsurancePolicy, t.InsuranceExpiry, t.InsuranceCoverage,
			t.AverageLeadDays, t.TrackingURL, t.ContactName, t.ContactPhone, t.ContactEmail,
			t.Notes, t.IsActive).Scan(&id)
	}
	if err != nil {
		return nil, traduzirErro(err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM public.shipping_carrier_vehicles WHERE carrier_id=$1 AND enterprise_id=$2`, id, enterpriseID); err != nil {
		return nil, fmt.Errorf("limpar frota: %w", err)
	}
	for _, v := range t.Vehicles {
		if _, err := tx.Exec(ctx, `
INSERT INTO public.shipping_carrier_vehicles
 (enterprise_id, carrier_id, plate, description, vehicle_type, axles, capacity_kg, capacity_m3,
  antt_owner, driver_name, driver_document, driver_license, is_active)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
			enterpriseID, id, v.Plate, v.Description, v.VehicleType, v.Axles, v.CapacityKg, v.CapacityM3,
			v.ANTTOwner, v.DriverName, v.DriverDocument, v.DriverLicense, v.IsActive); err != nil {
			return nil, traduzirErro(err)
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM public.shipping_carrier_service_areas WHERE carrier_id=$1 AND enterprise_id=$2`, id, enterpriseID); err != nil {
		return nil, fmt.Errorf("limpar regiões atendidas: %w", err)
	}
	for _, a := range t.ServiceAreas {
		if _, err := tx.Exec(ctx, `
INSERT INTO public.shipping_carrier_service_areas
 (enterprise_id, carrier_id, state, city, postal_code_from, postal_code_to, lead_days,
  min_value, kg_rate, pct_value, is_active)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			enterpriseID, id, a.State, a.City, a.PostalCodeFrom, a.PostalCodeTo, a.LeadDays,
			a.MinValue, a.KgRate, a.PctValue, a.IsActive); err != nil {
			return nil, traduzirErro(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.Obter(ctx, id)
}

func tipoTexto(t *entity.TipoTransportador) *string {
	if t == nil {
		return nil
	}
	v := string(*t)
	return &v
}

func traduzirErro(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "shipping_carriers_unico"):
		return errorsuc.NewConflictError("este fornecedor já tem um perfil de transportadora cadastrado")
	case strings.Contains(msg, "shipping_carrier_vehicles_unico"):
		return errorsuc.NewConflictError("a mesma placa aparece duas vezes na frota")
	case strings.Contains(msg, "shipping_carriers_supplier_code_fkey"):
		return errorsuc.NewValidationError("o fornecedor informado não existe")
	}
	return fmt.Errorf("gravar transportadora: %w", err)
}

func (r *Repository) DefinirSituacao(ctx context.Context, id int64, ativa bool) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `
UPDATE public.shipping_carriers SET is_active=$3, updated_at=NOW()
WHERE id=$1 AND enterprise_id=$2`, id, enterpriseID, ativa)
	if err != nil {
		return fmt.Errorf("mudar situação da transportadora: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("transportadora %d não encontrada", id))
	}
	return nil
}

func (r *Repository) RegistrarOcorrencia(ctx context.Context, o *domrepo.Ocorrencia) (*domrepo.Ocorrencia, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var autor any
	if user, ok := ctx.Value(contextkey.UserKey).(*appsecurity.AuthUser); ok && user != nil && user.ID != "" {
		autor = user.ID
	}
	err = r.pool.QueryRow(ctx, `
INSERT INTO public.shipping_carrier_occurrences
 (enterprise_id, carrier_id, occurrence_date, occurrence_type, sales_order_code, delay_days, cost_impact, description, created_by)
SELECT $1, c.id, $3, $4, $5, $6, $7, $8, $9
FROM public.shipping_carriers c WHERE c.id = $2 AND c.enterprise_id = $1
RETURNING id, occurrence_date, created_at`,
		enterpriseID, o.CarrierID, o.OccurrenceDate, o.OccurrenceType, o.SalesOrderCode,
		o.DelayDays, o.CostImpact, o.Description, autor).
		Scan(&o.ID, &o.OccurrenceDate, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("transportadora %d não encontrada", o.CarrierID))
	}
	if err != nil {
		return nil, fmt.Errorf("registrar ocorrência: %w", err)
	}
	o.EnterpriseID = enterpriseID
	return o, nil
}

func (r *Repository) ListarOcorrencias(ctx context.Context, carrierID int64, limite int) ([]*domrepo.Ocorrencia, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if limite <= 0 || limite > 500 {
		limite = 100
	}
	rows, err := r.pool.Query(ctx, `
SELECT id, enterprise_id, carrier_id, occurrence_date, occurrence_type, sales_order_code,
       delay_days, cost_impact, description, created_at
FROM public.shipping_carrier_occurrences
WHERE carrier_id=$1 AND enterprise_id=$2
ORDER BY occurrence_date DESC, id DESC
LIMIT $3`, carrierID, enterpriseID, limite)
	if err != nil {
		return nil, fmt.Errorf("listar ocorrências: %w", err)
	}
	defer rows.Close()
	var out []*domrepo.Ocorrencia
	for rows.Next() {
		var o domrepo.Ocorrencia
		if err := rows.Scan(&o.ID, &o.EnterpriseID, &o.CarrierID, &o.OccurrenceDate, &o.OccurrenceType,
			&o.SalesOrderCode, &o.DelayDays, &o.CostImpact, &o.Description, &o.CreatedAt); err != nil {
			return nil, fmt.Errorf("ler ocorrência: %w", err)
		}
		out = append(out, &o)
	}
	return out, rows.Err()
}

func (r *Repository) Desempenho(ctx context.Context, carrierID int64, desde time.Time) (domrepo.Desempenho, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return domrepo.Desempenho{}, err
	}
	var d domrepo.Desempenho
	var media, custo decimal.Decimal
	err = r.pool.QueryRow(ctx, `
SELECT COUNT(*), COALESCE(AVG(delay_days),0), COALESCE(SUM(cost_impact),0), MAX(occurrence_date)
FROM public.shipping_carrier_occurrences
WHERE carrier_id=$1 AND enterprise_id=$2 AND occurrence_date >= $3`,
		carrierID, enterpriseID, desde).Scan(&d.Ocorrencias, &media, &custo, &d.UltimaData)
	if err != nil {
		return domrepo.Desempenho{}, fmt.Errorf("desempenho da transportadora: %w", err)
	}
	d.AtrasoMedio = media.Round(2)
	d.CustoTotal = custo
	return d, nil
}

var _ domrepo.Repository = (*Repository)(nil)
