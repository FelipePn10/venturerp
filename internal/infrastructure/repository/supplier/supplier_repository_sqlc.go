package supplier

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/supplier/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/supplier/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SupplierRepositorySQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func New(q *sqlc.Queries, pool *pgxpool.Pool) domainrepo.SupplierRepository {
	return &SupplierRepositorySQLC{q: q, pool: pool}
}

// ─── Supplier Types ─────────────────────────────────────────────────────────

func (r *SupplierRepositorySQLC) CreateSupplierType(ctx context.Context, t *entity.SupplierType) (*entity.SupplierType, error) {
	row, err := r.q.CreateSupplierType(ctx, sqlc.CreateSupplierTypeParams{
		Code:        t.Code,
		Description: t.Description,
		Kind:        string(t.Kind),
	})
	if err != nil {
		return nil, fmt.Errorf("creating supplier type: %w", err)
	}
	return supplierTypeToEntity(row), nil
}

func (r *SupplierRepositorySQLC) UpdateSupplierType(ctx context.Context, t *entity.SupplierType) (*entity.SupplierType, error) {
	row, err := r.q.UpdateSupplierType(ctx, sqlc.UpdateSupplierTypeParams{
		Code:        t.Code,
		Description: t.Description,
		Kind:        string(t.Kind),
		IsActive:    t.IsActive,
	})
	if err != nil {
		return nil, fmt.Errorf("updating supplier type: %w", err)
	}
	return supplierTypeToEntity(row), nil
}

func (r *SupplierRepositorySQLC) GetSupplierTypeByCode(ctx context.Context, code int64) (*entity.SupplierType, error) {
	row, err := r.q.GetSupplierTypeByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("supplier type %d not found: %w", code, err)
	}
	return supplierTypeToEntity(row), nil
}

func (r *SupplierRepositorySQLC) ListSupplierTypes(ctx context.Context, onlyActive bool) ([]*entity.SupplierType, error) {
	rows, err := r.q.ListSupplierTypes(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.SupplierType, 0, len(rows))
	for _, row := range rows {
		out = append(out, supplierTypeToEntity(row))
	}
	return out, nil
}

func (r *SupplierRepositorySQLC) NextSupplierTypeCode(ctx context.Context) (int64, error) {
	v, err := r.q.NextSupplierTypeCode(ctx)
	return int64(v), err
}

// ─── Supplier Contact Types ───────────────────────────────────────────────────

func (r *SupplierRepositorySQLC) CreateContactType(ctx context.Context, ct *entity.SupplierContactType) (*entity.SupplierContactType, error) {
	row, err := r.q.CreateSupplierContactType(ctx, sqlc.CreateSupplierContactTypeParams{
		Code:        ct.Code,
		Description: ct.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("creating contact type: %w", err)
	}
	return contactTypeToEntity(row), nil
}

func (r *SupplierRepositorySQLC) UpdateContactType(ctx context.Context, ct *entity.SupplierContactType) (*entity.SupplierContactType, error) {
	row, err := r.q.UpdateSupplierContactType(ctx, sqlc.UpdateSupplierContactTypeParams{
		Code:        ct.Code,
		Description: ct.Description,
		IsActive:    ct.IsActive,
	})
	if err != nil {
		return nil, fmt.Errorf("updating contact type: %w", err)
	}
	return contactTypeToEntity(row), nil
}

func (r *SupplierRepositorySQLC) GetContactTypeByCode(ctx context.Context, code int64) (*entity.SupplierContactType, error) {
	row, err := r.q.GetSupplierContactTypeByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("contact type %d not found: %w", code, err)
	}
	return contactTypeToEntity(row), nil
}

func (r *SupplierRepositorySQLC) ListContactTypes(ctx context.Context, onlyActive bool) ([]*entity.SupplierContactType, error) {
	rows, err := r.q.ListSupplierContactTypes(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.SupplierContactType, 0, len(rows))
	for _, row := range rows {
		out = append(out, contactTypeToEntity(row))
	}
	return out, nil
}

func (r *SupplierRepositorySQLC) NextContactTypeCode(ctx context.Context) (int64, error) {
	v, err := r.q.NextSupplierContactTypeCode(ctx)
	return int64(v), err
}

// ─── Suppliers ────────────────────────────────────────────────────────────────

func (r *SupplierRepositorySQLC) CreateSupplier(ctx context.Context, s *entity.Supplier) (*entity.Supplier, error) {
	row, err := r.q.CreateSupplier(ctx, sqlc.CreateSupplierParams{
		Code:                            s.Code,
		CorporateCode:                   s.CorporateCode,
		IsActive:                        s.IsActive,
		IsRepresentative:                s.IsRepresentative,
		IsCustomer:                      s.IsCustomer,
		Name:                            s.Name,
		TradeName:                       pgutil.ToPgTextFromPtr(s.TradeName),
		PersonType:                      string(s.PersonType),
		DocumentType:                    string(s.DocumentType),
		DocumentNumber:                  s.DocumentNumber,
		StateRegistration:               pgutil.ToPgTextFromPtr(s.StateRegistration),
		MunicipalRegistration:           pgutil.ToPgTextFromPtr(s.MunicipalRegistration),
		SupplierTypeID:                  s.SupplierTypeID,
		PaymentConditionID:              s.PaymentConditionID,
		CarrierID:                       s.CarrierID,
		RegionID:                        s.RegionID,
		FreightType:                     string(s.FreightType),
		RegisterDate:                    pgutil.ToPgDate(s.RegisterDate),
		ViticolaObligation:              string(s.ViticolaObligation),
		GlnCode:                         pgutil.ToPgTextFromPtr(s.GLNCode),
		AgricultureMinistryRegistration: pgutil.ToPgTextFromPtr(s.AgricultureMinistryRegistration),
		IcmsContributor:                 string(s.ICMSContributor),
		IsMei:                           s.IsMEI,
		TrackingPlatform:                string(s.TrackingPlatform),
		Homologated:                     s.Homologated,
		CreatedBy:                       pgutil.ToPgUUID(s.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating supplier: %w", err)
	}
	return supplierToEntity(row), nil
}

func (r *SupplierRepositorySQLC) UpdateSupplier(ctx context.Context, s *entity.Supplier) (*entity.Supplier, error) {
	row, err := r.q.UpdateSupplier(ctx, sqlc.UpdateSupplierParams{
		Code:                            s.Code,
		CorporateCode:                   s.CorporateCode,
		IsActive:                        s.IsActive,
		IsRepresentative:                s.IsRepresentative,
		IsCustomer:                      s.IsCustomer,
		Name:                            s.Name,
		TradeName:                       pgutil.ToPgTextFromPtr(s.TradeName),
		PersonType:                      string(s.PersonType),
		DocumentType:                    string(s.DocumentType),
		DocumentNumber:                  s.DocumentNumber,
		StateRegistration:               pgutil.ToPgTextFromPtr(s.StateRegistration),
		MunicipalRegistration:           pgutil.ToPgTextFromPtr(s.MunicipalRegistration),
		SupplierTypeID:                  s.SupplierTypeID,
		PaymentConditionID:              s.PaymentConditionID,
		CarrierID:                       s.CarrierID,
		RegionID:                        s.RegionID,
		FreightType:                     string(s.FreightType),
		ViticolaObligation:              string(s.ViticolaObligation),
		GlnCode:                         pgutil.ToPgTextFromPtr(s.GLNCode),
		AgricultureMinistryRegistration: pgutil.ToPgTextFromPtr(s.AgricultureMinistryRegistration),
		IcmsContributor:                 string(s.ICMSContributor),
		IsMei:                           s.IsMEI,
		TrackingPlatform:                string(s.TrackingPlatform),
		Homologated:                     s.Homologated,
	})
	if err != nil {
		return nil, fmt.Errorf("updating supplier: %w", err)
	}
	return supplierToEntity(row), nil
}

func (r *SupplierRepositorySQLC) GetSupplierByCode(ctx context.Context, code int64) (*entity.Supplier, error) {
	row, err := r.q.GetSupplierByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("supplier %d not found: %w", code, err)
	}
	return supplierToEntity(row), nil
}

func (r *SupplierRepositorySQLC) GetSupplierByDocument(ctx context.Context, document string) (*entity.Supplier, error) {
	row, err := r.q.GetSupplierByDocument(ctx, document)
	if err != nil {
		return nil, err
	}
	return supplierToEntity(row), nil
}

func (r *SupplierRepositorySQLC) ListSuppliers(ctx context.Context, onlyActive bool) ([]*entity.Supplier, error) {
	rows, err := r.q.ListSuppliers(ctx, onlyActive)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.Supplier, 0, len(rows))
	for _, row := range rows {
		out = append(out, supplierToEntity(row))
	}
	return out, nil
}

func (r *SupplierRepositorySQLC) ListEstablishments(ctx context.Context, corporateCode int64) ([]*entity.Supplier, error) {
	rows, err := r.q.ListSupplierEstablishments(ctx, &corporateCode)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.Supplier, 0, len(rows))
	for _, row := range rows {
		out = append(out, supplierToEntity(row))
	}
	return out, nil
}

func (r *SupplierRepositorySQLC) BlockSupplier(ctx context.Context, code int64, reason string) error {
	return r.q.BlockSupplier(ctx, sqlc.BlockSupplierParams{
		Code:        code,
		BlockReason: pgutil.ToPgTextFromString(reason),
	})
}

func (r *SupplierRepositorySQLC) UnblockSupplier(ctx context.Context, code int64) error {
	return r.q.UnblockSupplier(ctx, code)
}

func (r *SupplierRepositorySQLC) NextSupplierCode(ctx context.Context) (int64, error) {
	v, err := r.q.NextSupplierCode(ctx)
	return int64(v), err
}

func (r *SupplierRepositorySQLC) PropagateStateRegistration(ctx context.Context, document string, ie *string, exceptCode int64) error {
	return r.q.PropagateStateRegistration(ctx, sqlc.PropagateStateRegistrationParams{
		DocumentNumber:    document,
		StateRegistration: pgutil.ToPgTextFromPtr(ie),
		Code:              exceptCode,
	})
}

func (r *SupplierRepositorySQLC) UpdateSefazSnapshot(ctx context.Context, code int64, status, user string) error {
	now := pgutil.ToPgDate(time.Now())
	return r.q.UpdateSupplierSefaz(ctx, sqlc.UpdateSupplierSefazParams{
		Code:                 code,
		LastSefazQuery:       now,
		BillingReceiptStatus: pgutil.ToPgTextFromString(status),
		LastSefazUpdate:      now,
		SefazUpdateUser:      pgutil.ToPgTextFromString(user),
	})
}

func (r *SupplierRepositorySQLC) DeleteSupplier(ctx context.Context, code int64) error {
	if err := r.q.DeleteSupplier(ctx, code); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return fmt.Errorf("não é possível excluir o fornecedor: há pedidos de compra ou processos de importação vinculados — inative-o ou desvincule-o primeiro")
		}
		return fmt.Errorf("deleting supplier: %w", err)
	}
	return nil
}

// ─── Addresses ──────────────────────────────────────────────────────────────

func (r *SupplierRepositorySQLC) AddAddress(ctx context.Context, a *entity.SupplierAddress) (*entity.SupplierAddress, error) {
	row, err := r.q.CreateSupplierAddress(ctx, sqlc.CreateSupplierAddressParams{
		SupplierID:   a.SupplierID,
		AddressType:  string(a.AddressType),
		ZipCode:      pgutil.ToPgTextFromPtr(a.ZipCode),
		Street:       pgutil.ToPgTextFromPtr(a.Street),
		Number:       pgutil.ToPgTextFromPtr(a.Number),
		Complement:   pgutil.ToPgTextFromPtr(a.Complement),
		Neighborhood: pgutil.ToPgTextFromPtr(a.Neighborhood),
		City:         pgutil.ToPgTextFromPtr(a.City),
		Uf:           pgutil.ToPgTextFromPtr(a.UF),
		Country:      a.Country,
		IsDefault:    a.IsDefault,
	})
	if err != nil {
		return nil, fmt.Errorf("adding address: %w", err)
	}
	return addressToEntity(row), nil
}

func (r *SupplierRepositorySQLC) UpdateAddress(ctx context.Context, a *entity.SupplierAddress) (*entity.SupplierAddress, error) {
	row, err := r.q.UpdateSupplierAddress(ctx, sqlc.UpdateSupplierAddressParams{
		ID:           a.ID,
		AddressType:  string(a.AddressType),
		ZipCode:      pgutil.ToPgTextFromPtr(a.ZipCode),
		Street:       pgutil.ToPgTextFromPtr(a.Street),
		Number:       pgutil.ToPgTextFromPtr(a.Number),
		Complement:   pgutil.ToPgTextFromPtr(a.Complement),
		Neighborhood: pgutil.ToPgTextFromPtr(a.Neighborhood),
		City:         pgutil.ToPgTextFromPtr(a.City),
		Uf:           pgutil.ToPgTextFromPtr(a.UF),
		Country:      a.Country,
		IsDefault:    a.IsDefault,
	})
	if err != nil {
		return nil, fmt.Errorf("updating address: %w", err)
	}
	return addressToEntity(row), nil
}

func (r *SupplierRepositorySQLC) ListAddresses(ctx context.Context, supplierID int64) ([]*entity.SupplierAddress, error) {
	rows, err := r.q.ListSupplierAddresses(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.SupplierAddress, 0, len(rows))
	for _, row := range rows {
		out = append(out, addressToEntity(row))
	}
	return out, nil
}

func (r *SupplierRepositorySQLC) DeleteAddress(ctx context.Context, id int64) error {
	return r.q.DeleteSupplierAddress(ctx, id)
}

// ─── Phones ───────────────────────────────────────────────────────────────────

func (r *SupplierRepositorySQLC) AddPhone(ctx context.Context, p *entity.SupplierPhone) (*entity.SupplierPhone, error) {
	row, err := r.q.CreateSupplierPhone(ctx, sqlc.CreateSupplierPhoneParams{
		SupplierID: p.SupplierID,
		Number:     p.Number,
		Ranking:    p.Ranking,
	})
	if err != nil {
		return nil, fmt.Errorf("adding phone: %w", err)
	}
	return phoneToEntity(row), nil
}

func (r *SupplierRepositorySQLC) ListPhones(ctx context.Context, supplierID int64) ([]*entity.SupplierPhone, error) {
	rows, err := r.q.ListSupplierPhones(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.SupplierPhone, 0, len(rows))
	for _, row := range rows {
		out = append(out, phoneToEntity(row))
	}
	return out, nil
}

func (r *SupplierRepositorySQLC) DeletePhone(ctx context.Context, id int64) error {
	return r.q.DeleteSupplierPhone(ctx, id)
}

// ─── Emails ─────────────────────────────────────────────────────────────────

func (r *SupplierRepositorySQLC) AddEmail(ctx context.Context, e *entity.SupplierEmail) (*entity.SupplierEmail, error) {
	row, err := r.q.CreateSupplierEmail(ctx, sqlc.CreateSupplierEmailParams{
		SupplierID: e.SupplierID,
		Email:      e.Email,
		Ranking:    e.Ranking,
	})
	if err != nil {
		return nil, fmt.Errorf("adding email: %w", err)
	}
	return emailToEntity(row), nil
}

func (r *SupplierRepositorySQLC) ListEmails(ctx context.Context, supplierID int64) ([]*entity.SupplierEmail, error) {
	rows, err := r.q.ListSupplierEmails(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.SupplierEmail, 0, len(rows))
	for _, row := range rows {
		out = append(out, emailToEntity(row))
	}
	return out, nil
}

func (r *SupplierRepositorySQLC) DeleteEmail(ctx context.Context, id int64) error {
	return r.q.DeleteSupplierEmail(ctx, id)
}

// ─── Due Dates ─────────────────────────────────────────────────────────────────

func (r *SupplierRepositorySQLC) AddDueDate(ctx context.Context, d *entity.SupplierDueDate) (*entity.SupplierDueDate, error) {
	row, err := r.q.CreateSupplierDueDate(ctx, sqlc.CreateSupplierDueDateParams{
		SupplierID:         d.SupplierID,
		Description:        d.Description,
		Ranking:            d.Ranking,
		BaseDate:           string(d.BaseDate),
		PaymentConditionID: d.PaymentConditionID,
		PaymentType:        string(d.PaymentType),
		SubsequentMonth:    d.SubsequentMonth,
		Rounding:           string(d.Rounding),
		ReceiptStartTime:   pgutil.ToPgTextFromPtr(d.ReceiptStartTime),
		ReceiptEndTime:     pgutil.ToPgTextFromPtr(d.ReceiptEndTime),
		AvgUnloadMinutes:   d.AvgUnloadMinutes,
	})
	if err != nil {
		return nil, fmt.Errorf("adding due date: %w", err)
	}
	return dueDateToEntity(row), nil
}

func (r *SupplierRepositorySQLC) ListDueDates(ctx context.Context, supplierID int64) ([]*entity.SupplierDueDate, error) {
	rows, err := r.q.ListSupplierDueDates(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.SupplierDueDate, 0, len(rows))
	for _, row := range rows {
		out = append(out, dueDateToEntity(row))
	}
	return out, nil
}

func (r *SupplierRepositorySQLC) DeleteDueDate(ctx context.Context, id int64) error {
	return r.q.DeleteSupplierDueDate(ctx, id)
}

// ─── Contacts ─────────────────────────────────────────────────────────────────

func (r *SupplierRepositorySQLC) AddContact(ctx context.Context, c *entity.SupplierContact) (*entity.SupplierContact, error) {
	row, err := r.q.CreateSupplierContact(ctx, sqlc.CreateSupplierContactParams{
		SupplierID:       c.SupplierID,
		ContactTypeID:    c.ContactTypeID,
		Name:             c.Name,
		Position:         pgutil.ToPgTextFromPtr(c.Position),
		Department:       pgutil.ToPgTextFromPtr(c.Department),
		Ranking:          c.Ranking,
		Observation:      pgutil.ToPgTextFromPtr(c.Observation),
		PurchaseOrderTag: pgutil.ToPgTextFromPtr(c.PurchaseOrderTag),
	})
	if err != nil {
		return nil, fmt.Errorf("adding contact: %w", err)
	}
	return contactToEntity(row), nil
}

func (r *SupplierRepositorySQLC) ListContacts(ctx context.Context, supplierID int64) ([]*entity.SupplierContact, error) {
	rows, err := r.q.ListSupplierContacts(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.SupplierContact, 0, len(rows))
	for _, row := range rows {
		out = append(out, contactToEntity(row))
	}
	return out, nil
}

func (r *SupplierRepositorySQLC) DeleteContact(ctx context.Context, id int64) error {
	return r.q.DeleteSupplierContact(ctx, id)
}

func (r *SupplierRepositorySQLC) AddContactPhone(ctx context.Context, p *entity.SupplierContactPhone) (*entity.SupplierContactPhone, error) {
	row, err := r.q.CreateSupplierContactPhone(ctx, sqlc.CreateSupplierContactPhoneParams{
		ContactID: p.ContactID,
		Value:     p.Value,
		Ranking:   p.Ranking,
	})
	if err != nil {
		return nil, fmt.Errorf("adding contact phone: %w", err)
	}
	return &entity.SupplierContactPhone{ID: row.ID, ContactID: row.ContactID, Value: row.Value, Ranking: row.Ranking}, nil
}

func (r *SupplierRepositorySQLC) AddContactEmail(ctx context.Context, e *entity.SupplierContactEmail) (*entity.SupplierContactEmail, error) {
	row, err := r.q.CreateSupplierContactEmail(ctx, sqlc.CreateSupplierContactEmailParams{
		ContactID: e.ContactID,
		Value:     e.Value,
		Ranking:   e.Ranking,
	})
	if err != nil {
		return nil, fmt.Errorf("adding contact email: %w", err)
	}
	return &entity.SupplierContactEmail{ID: row.ID, ContactID: row.ContactID, Value: row.Value, Ranking: row.Ranking}, nil
}

func (r *SupplierRepositorySQLC) ListContactPhones(ctx context.Context, contactID int64) ([]*entity.SupplierContactPhone, error) {
	rows, err := r.q.ListSupplierContactPhones(ctx, contactID)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.SupplierContactPhone, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.SupplierContactPhone{ID: row.ID, ContactID: row.ContactID, Value: row.Value, Ranking: row.Ranking})
	}
	return out, nil
}

func (r *SupplierRepositorySQLC) ListContactEmails(ctx context.Context, contactID int64) ([]*entity.SupplierContactEmail, error) {
	rows, err := r.q.ListSupplierContactEmails(ctx, contactID)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.SupplierContactEmail, 0, len(rows))
	for _, row := range rows {
		out = append(out, &entity.SupplierContactEmail{ID: row.ID, ContactID: row.ContactID, Value: row.Value, Ranking: row.Ranking})
	}
	return out, nil
}

// ─── Enterprise links ──────────────────────────────────────────────────────

func (r *SupplierRepositorySQLC) AddEnterprise(ctx context.Context, e *entity.SupplierEnterprise) (*entity.SupplierEnterprise, error) {
	row, err := r.q.CreateSupplierEnterprise(ctx, sqlc.CreateSupplierEnterpriseParams{
		SupplierID:           e.SupplierID,
		EnterpriseCode:       e.EnterpriseCode,
		FinancialAccount:     pgutil.ToPgTextFromPtr(e.FinancialAccount),
		AppliesIpi:           e.AppliesIPI,
		DefaultInvoiceTypeID: e.DefaultInvoiceTypeID,
		PurchasePriceTableID: e.PurchasePriceTableID,
	})
	if err != nil {
		return nil, fmt.Errorf("adding enterprise link: %w", err)
	}
	return enterpriseToEntity(row), nil
}

func (r *SupplierRepositorySQLC) UpdateEnterprise(ctx context.Context, e *entity.SupplierEnterprise) (*entity.SupplierEnterprise, error) {
	row, err := r.q.UpdateSupplierEnterprise(ctx, sqlc.UpdateSupplierEnterpriseParams{
		ID:                   e.ID,
		FinancialAccount:     pgutil.ToPgTextFromPtr(e.FinancialAccount),
		AppliesIpi:           e.AppliesIPI,
		DefaultInvoiceTypeID: e.DefaultInvoiceTypeID,
		PurchasePriceTableID: e.PurchasePriceTableID,
		IsActive:             e.IsActive,
	})
	if err != nil {
		return nil, fmt.Errorf("updating enterprise link: %w", err)
	}
	return enterpriseToEntity(row), nil
}

func (r *SupplierRepositorySQLC) ListEnterprises(ctx context.Context, supplierID int64) ([]*entity.SupplierEnterprise, error) {
	rows, err := r.q.ListSupplierEnterprises(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.SupplierEnterprise, 0, len(rows))
	for _, row := range rows {
		out = append(out, enterpriseToEntity(row))
	}
	return out, nil
}

func (r *SupplierRepositorySQLC) DeleteEnterprise(ctx context.Context, id int64) error {
	return r.q.DeleteSupplierEnterprise(ctx, id)
}

// ─── Parameters ─────────────────────────────────────────────────────────────

func (r *SupplierRepositorySQLC) GetParameters(ctx context.Context, enterpriseCode int64) (*entity.SupplierParameters, error) {
	row, err := r.q.GetSupplierParameters(ctx, enterpriseCode)
	if err != nil {
		return nil, err
	}
	return parametersToEntity(row), nil
}

func (r *SupplierRepositorySQLC) UpsertParameters(ctx context.Context, p *entity.SupplierParameters) (*entity.SupplierParameters, error) {
	row, err := r.q.UpsertSupplierParameters(ctx, sqlc.UpsertSupplierParametersParams{
		EnterpriseCode:            p.EnterpriseCode,
		DefaultFinancialAccount:   pgutil.ToPgTextFromPtr(p.DefaultFinancialAccount),
		UniqueItemCodePerSupplier: p.UniqueItemCodePerSupplier,
		RequiresFinancialAccount:  p.RequiresFinancialAccount,
		PurchaseSupplierTypeID:    p.PurchaseSupplierTypeID,
		CopyObsToPurchaseOrder:    p.CopyObsToPurchaseOrder,
		CopyObsToEntryInvoice:     p.CopyObsToEntryInvoice,
		HomologationDefault:       p.HomologationDefault,
		UseStockUom:               p.UseStockUOM,
		GenericSupplierCode:       p.GenericSupplierCode,
		DefaultDueBaseDate:        string(p.DefaultDueBaseDate),
	})
	if err != nil {
		return nil, fmt.Errorf("upserting parameters: %w", err)
	}
	return parametersToEntity(row), nil
}

// ─── Mappers ──────────────────────────────────────────────────────────────────

func supplierTypeToEntity(row sqlc.SupplierType) *entity.SupplierType {
	return &entity.SupplierType{
		ID:          row.ID,
		Code:        row.Code,
		Description: row.Description,
		Kind:        entity.SupplierKind(row.Kind),
		IsActive:    row.IsActive,
		CreatedAt:   pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

func contactTypeToEntity(row sqlc.SupplierContactType) *entity.SupplierContactType {
	return &entity.SupplierContactType{
		ID:          row.ID,
		Code:        row.Code,
		Description: row.Description,
		IsActive:    row.IsActive,
		CreatedAt:   pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

func supplierToEntity(row sqlc.Supplier) *entity.Supplier {
	return &entity.Supplier{
		ID:                              row.ID,
		Code:                            row.Code,
		CorporateCode:                   row.CorporateCode,
		IsActive:                        row.IsActive,
		IsRepresentative:                row.IsRepresentative,
		IsCustomer:                      row.IsCustomer,
		Name:                            row.Name,
		TradeName:                       pgutil.FromPgTextPtr(row.TradeName),
		PersonType:                      entity.PersonType(row.PersonType),
		DocumentType:                    entity.DocumentType(row.DocumentType),
		DocumentNumber:                  row.DocumentNumber,
		StateRegistration:               pgutil.FromPgTextPtr(row.StateRegistration),
		MunicipalRegistration:           pgutil.FromPgTextPtr(row.MunicipalRegistration),
		SupplierTypeID:                  row.SupplierTypeID,
		PaymentConditionID:              row.PaymentConditionID,
		CarrierID:                       row.CarrierID,
		RegionID:                        row.RegionID,
		FreightType:                     entity.FreightType(row.FreightType),
		RegisterDate:                    pgutil.FromPgDate(row.RegisterDate),
		ViticolaObligation:              entity.ViticolaObligation(row.ViticolaObligation),
		GLNCode:                         pgutil.FromPgTextPtr(row.GlnCode),
		AgricultureMinistryRegistration: pgutil.FromPgTextPtr(row.AgricultureMinistryRegistration),
		ICMSContributor:                 entity.ICMSContributor(row.IcmsContributor),
		IsMEI:                           row.IsMei,
		TrackingPlatform:                entity.TrackingPlatform(row.TrackingPlatform),
		Homologated:                     row.Homologated,
		LastSefazQuery:                  pgutil.FromPgDateToPtr(row.LastSefazQuery),
		BillingReceiptStatus:            pgutil.FromPgTextPtr(row.BillingReceiptStatus),
		LastSefazUpdate:                 pgutil.FromPgDateToPtr(row.LastSefazUpdate),
		SefazUpdateUser:                 pgutil.FromPgTextPtr(row.SefazUpdateUser),
		Blocked:                         row.Blocked,
		BlockReason:                     pgutil.FromPgTextPtr(row.BlockReason),
		CreatedAt:                       pgutil.FromPgTimestamptz(row.CreatedAt),
		CreatedBy:                       pgutil.FromPgUUID(row.CreatedBy),
		UpdatedAt:                       pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
}

func addressToEntity(row sqlc.SupplierAddress) *entity.SupplierAddress {
	return &entity.SupplierAddress{
		ID:           row.ID,
		SupplierID:   row.SupplierID,
		AddressType:  entity.AddressType(row.AddressType),
		ZipCode:      pgutil.FromPgTextPtr(row.ZipCode),
		Street:       pgutil.FromPgTextPtr(row.Street),
		Number:       pgutil.FromPgTextPtr(row.Number),
		Complement:   pgutil.FromPgTextPtr(row.Complement),
		Neighborhood: pgutil.FromPgTextPtr(row.Neighborhood),
		City:         pgutil.FromPgTextPtr(row.City),
		UF:           pgutil.FromPgTextPtr(row.Uf),
		Country:      row.Country,
		IsDefault:    row.IsDefault,
		CreatedAt:    pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

func phoneToEntity(row sqlc.SupplierPhone) *entity.SupplierPhone {
	return &entity.SupplierPhone{
		ID:         row.ID,
		SupplierID: row.SupplierID,
		Number:     row.Number,
		Ranking:    row.Ranking,
		IsActive:   row.IsActive,
		CreatedAt:  pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

func emailToEntity(row sqlc.SupplierEmail) *entity.SupplierEmail {
	return &entity.SupplierEmail{
		ID:         row.ID,
		SupplierID: row.SupplierID,
		Email:      row.Email,
		Ranking:    row.Ranking,
		IsActive:   row.IsActive,
		CreatedAt:  pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

func dueDateToEntity(row sqlc.SupplierDueDate) *entity.SupplierDueDate {
	return &entity.SupplierDueDate{
		ID:                 row.ID,
		SupplierID:         row.SupplierID,
		Description:        row.Description,
		Ranking:            row.Ranking,
		BaseDate:           entity.BaseDate(row.BaseDate),
		PaymentConditionID: row.PaymentConditionID,
		PaymentType:        entity.DuePaymentType(row.PaymentType),
		SubsequentMonth:    row.SubsequentMonth,
		Rounding:           entity.DueRounding(row.Rounding),
		ReceiptStartTime:   pgutil.FromPgTextPtr(row.ReceiptStartTime),
		ReceiptEndTime:     pgutil.FromPgTextPtr(row.ReceiptEndTime),
		AvgUnloadMinutes:   row.AvgUnloadMinutes,
		CreatedAt:          pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

func contactToEntity(row sqlc.SupplierContact) *entity.SupplierContact {
	return &entity.SupplierContact{
		ID:               row.ID,
		SupplierID:       row.SupplierID,
		ContactTypeID:    row.ContactTypeID,
		Name:             row.Name,
		Position:         pgutil.FromPgTextPtr(row.Position),
		Department:       pgutil.FromPgTextPtr(row.Department),
		Ranking:          row.Ranking,
		Observation:      pgutil.FromPgTextPtr(row.Observation),
		PurchaseOrderTag: pgutil.FromPgTextPtr(row.PurchaseOrderTag),
		IsActive:         row.IsActive,
		CreatedAt:        pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

func enterpriseToEntity(row sqlc.SupplierEnterprise) *entity.SupplierEnterprise {
	return &entity.SupplierEnterprise{
		ID:                   row.ID,
		SupplierID:           row.SupplierID,
		EnterpriseCode:       row.EnterpriseCode,
		FinancialAccount:     pgutil.FromPgTextPtr(row.FinancialAccount),
		AppliesIPI:           row.AppliesIpi,
		DefaultInvoiceTypeID: row.DefaultInvoiceTypeID,
		PurchasePriceTableID: row.PurchasePriceTableID,
		IsActive:             row.IsActive,
		CreatedAt:            pgutil.FromPgTimestamptz(row.CreatedAt),
	}
}

func parametersToEntity(row sqlc.SupplierParameter) *entity.SupplierParameters {
	return &entity.SupplierParameters{
		ID:                        row.ID,
		EnterpriseCode:            row.EnterpriseCode,
		DefaultFinancialAccount:   pgutil.FromPgTextPtr(row.DefaultFinancialAccount),
		UniqueItemCodePerSupplier: row.UniqueItemCodePerSupplier,
		RequiresFinancialAccount:  row.RequiresFinancialAccount,
		PurchaseSupplierTypeID:    row.PurchaseSupplierTypeID,
		CopyObsToPurchaseOrder:    row.CopyObsToPurchaseOrder,
		CopyObsToEntryInvoice:     row.CopyObsToEntryInvoice,
		HomologationDefault:       row.HomologationDefault,
		UseStockUOM:               row.UseStockUom,
		GenericSupplierCode:       row.GenericSupplierCode,
		DefaultDueBaseDate:        entity.BaseDate(row.DefaultDueBaseDate),
		CreatedAt:                 pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:                 pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
}
