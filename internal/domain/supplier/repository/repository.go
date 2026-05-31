package repository

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/supplier/entity"
)

type SupplierRepository interface {
	// ── Supplier Types ─────────────────────────────────────────────────────────
	CreateSupplierType(ctx context.Context, t *entity.SupplierType) (*entity.SupplierType, error)
	UpdateSupplierType(ctx context.Context, t *entity.SupplierType) (*entity.SupplierType, error)
	GetSupplierTypeByCode(ctx context.Context, code int64) (*entity.SupplierType, error)
	ListSupplierTypes(ctx context.Context, onlyActive bool) ([]*entity.SupplierType, error)
	NextSupplierTypeCode(ctx context.Context) (int64, error)

	// ── Supplier Contact Types ───────────────────────────────────────────────────
	CreateContactType(ctx context.Context, ct *entity.SupplierContactType) (*entity.SupplierContactType, error)
	UpdateContactType(ctx context.Context, ct *entity.SupplierContactType) (*entity.SupplierContactType, error)
	GetContactTypeByCode(ctx context.Context, code int64) (*entity.SupplierContactType, error)
	ListContactTypes(ctx context.Context, onlyActive bool) ([]*entity.SupplierContactType, error)
	NextContactTypeCode(ctx context.Context) (int64, error)

	// ── Suppliers ────────────────────────────────────────────────────────────────
	CreateSupplier(ctx context.Context, s *entity.Supplier) (*entity.Supplier, error)
	UpdateSupplier(ctx context.Context, s *entity.Supplier) (*entity.Supplier, error)
	GetSupplierByCode(ctx context.Context, code int64) (*entity.Supplier, error)
	GetSupplierByDocument(ctx context.Context, document string) (*entity.Supplier, error)
	ListSuppliers(ctx context.Context, onlyActive bool) ([]*entity.Supplier, error)
	ListEstablishments(ctx context.Context, corporateCode int64) ([]*entity.Supplier, error)
	BlockSupplier(ctx context.Context, code int64, reason string) error
	UnblockSupplier(ctx context.Context, code int64) error
	NextSupplierCode(ctx context.Context) (int64, error)
	// PropagateStateRegistration updates the IE of every other supplier sharing
	// the same document (estabelecimentos/representantes).
	PropagateStateRegistration(ctx context.Context, document string, ie *string, exceptCode int64) error
	// DeleteSupplier hard-deletes; returns a friendly error when purchase orders
	// still reference the supplier.
	DeleteSupplier(ctx context.Context, code int64) error
	// UpdateSefazSnapshot records the result of a SEFAZ cadastral query.
	UpdateSefazSnapshot(ctx context.Context, code int64, status, user string) error

	// ── Addresses ──────────────────────────────────────────────────────────────
	AddAddress(ctx context.Context, a *entity.SupplierAddress) (*entity.SupplierAddress, error)
	UpdateAddress(ctx context.Context, a *entity.SupplierAddress) (*entity.SupplierAddress, error)
	ListAddresses(ctx context.Context, supplierID int64) ([]*entity.SupplierAddress, error)
	DeleteAddress(ctx context.Context, id int64) error

	// ── Phones ───────────────────────────────────────────────────────────────────
	AddPhone(ctx context.Context, p *entity.SupplierPhone) (*entity.SupplierPhone, error)
	ListPhones(ctx context.Context, supplierID int64) ([]*entity.SupplierPhone, error)
	DeletePhone(ctx context.Context, id int64) error

	// ── Emails ───────────────────────────────────────────────────────────────────
	AddEmail(ctx context.Context, e *entity.SupplierEmail) (*entity.SupplierEmail, error)
	ListEmails(ctx context.Context, supplierID int64) ([]*entity.SupplierEmail, error)
	DeleteEmail(ctx context.Context, id int64) error

	// ── Due Dates ─────────────────────────────────────────────────────────────────
	AddDueDate(ctx context.Context, d *entity.SupplierDueDate) (*entity.SupplierDueDate, error)
	ListDueDates(ctx context.Context, supplierID int64) ([]*entity.SupplierDueDate, error)
	DeleteDueDate(ctx context.Context, id int64) error

	// ── Contacts ─────────────────────────────────────────────────────────────────
	AddContact(ctx context.Context, c *entity.SupplierContact) (*entity.SupplierContact, error)
	ListContacts(ctx context.Context, supplierID int64) ([]*entity.SupplierContact, error)
	DeleteContact(ctx context.Context, id int64) error
	AddContactPhone(ctx context.Context, p *entity.SupplierContactPhone) (*entity.SupplierContactPhone, error)
	AddContactEmail(ctx context.Context, e *entity.SupplierContactEmail) (*entity.SupplierContactEmail, error)
	ListContactPhones(ctx context.Context, contactID int64) ([]*entity.SupplierContactPhone, error)
	ListContactEmails(ctx context.Context, contactID int64) ([]*entity.SupplierContactEmail, error)

	// ── Enterprise links ───────────────────────────────────────────────────────
	AddEnterprise(ctx context.Context, e *entity.SupplierEnterprise) (*entity.SupplierEnterprise, error)
	UpdateEnterprise(ctx context.Context, e *entity.SupplierEnterprise) (*entity.SupplierEnterprise, error)
	ListEnterprises(ctx context.Context, supplierID int64) ([]*entity.SupplierEnterprise, error)
	DeleteEnterprise(ctx context.Context, id int64) error

	// ── Parameters ────────────────────────────────────────────────────────────────
	GetParameters(ctx context.Context, enterpriseCode int64) (*entity.SupplierParameters, error)
	UpsertParameters(ctx context.Context, p *entity.SupplierParameters) (*entity.SupplierParameters, error)
}
