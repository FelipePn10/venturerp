package supplier_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/supplier/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/supplier/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/validation"
)

// SupplierUseCase consolidates all supplier-related operations.
type SupplierUseCase struct {
	repo repository.SupplierRepository
}

func NewSupplierUseCase(repo repository.SupplierRepository) *SupplierUseCase {
	return &SupplierUseCase{repo: repo}
}

// ─── Supplier Types ─────────────────────────────────────────────────────────

func (uc *SupplierUseCase) CreateSupplierType(ctx context.Context, dto request.CreateSupplierTypeDTO) (*entity.SupplierType, error) {
	code, err := uc.repo.NextSupplierTypeCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating supplier type code: %w", err)
	}
	t, err := entity.NewSupplierType(code, dto.Description, entity.SupplierKind(dto.Kind))
	if err != nil {
		return nil, err
	}
	return uc.repo.CreateSupplierType(ctx, t)
}

func (uc *SupplierUseCase) UpdateSupplierType(ctx context.Context, dto request.UpdateSupplierTypeDTO) (*entity.SupplierType, error) {
	t, err := uc.repo.GetSupplierTypeByCode(ctx, dto.Code)
	if err != nil {
		return nil, err
	}
	t.Description = dto.Description
	if dto.Kind != "" {
		t.Kind = entity.SupplierKind(dto.Kind)
	}
	t.IsActive = dto.IsActive
	return uc.repo.UpdateSupplierType(ctx, t)
}

func (uc *SupplierUseCase) GetSupplierType(ctx context.Context, code int64) (*entity.SupplierType, error) {
	return uc.repo.GetSupplierTypeByCode(ctx, code)
}

func (uc *SupplierUseCase) ListSupplierTypes(ctx context.Context, onlyActive bool) ([]*entity.SupplierType, error) {
	return uc.repo.ListSupplierTypes(ctx, onlyActive)
}

// ─── Supplier Contact Types ───────────────────────────────────────────────────

func (uc *SupplierUseCase) CreateContactType(ctx context.Context, dto request.CreateSupplierContactTypeDTO) (*entity.SupplierContactType, error) {
	code, err := uc.repo.NextContactTypeCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating contact type code: %w", err)
	}
	ct, err := entity.NewSupplierContactType(code, dto.Description)
	if err != nil {
		return nil, err
	}
	return uc.repo.CreateContactType(ctx, ct)
}

func (uc *SupplierUseCase) ListContactTypes(ctx context.Context, onlyActive bool) ([]*entity.SupplierContactType, error) {
	return uc.repo.ListContactTypes(ctx, onlyActive)
}

// ─── Suppliers ────────────────────────────────────────────────────────────────

func (uc *SupplierUseCase) CreateSupplier(ctx context.Context, dto request.CreateSupplierDTO) (*entity.Supplier, error) {
	// Reject duplicate document (spec: existing register for the same CPF/CNPJ).
	if existing, err := uc.repo.GetSupplierByDocument(ctx, dto.DocumentNumber); err == nil && existing != nil {
		return nil, fmt.Errorf("já existe um fornecedor (código %d) cadastrado para o documento %s", existing.Code, dto.DocumentNumber)
	}

	// Resolve the supplier type to obtain both its ID and kind (IE rule).
	var typeID *int64
	var typeKind entity.SupplierKind
	if dto.SupplierTypeCode != nil {
		st, err := uc.repo.GetSupplierTypeByCode(ctx, *dto.SupplierTypeCode)
		if err != nil {
			return nil, fmt.Errorf("tipo de fornecedor inválido: %w", err)
		}
		typeID = &st.ID
		typeKind = st.Kind
	}

	code, err := uc.repo.NextSupplierCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generating supplier code: %w", err)
	}

	s, err := entity.NewSupplier(code, entity.SupplierInput{
		Name:                            dto.Name,
		TradeName:                       dto.TradeName,
		PersonType:                      entity.PersonType(dto.PersonType),
		DocumentType:                    entity.DocumentType(dto.DocumentType),
		DocumentNumber:                  dto.DocumentNumber,
		TypeKind:                        typeKind,
		StateRegistration:               dto.StateRegistration,
		IsMEI:                           dto.IsMEI,
		AgricultureMinistryRegistration: dto.AgricultureMinistryRegistration,
	}, dto.CreatedBy)
	if err != nil {
		return nil, err
	}

	// Apply optional / classification fields.
	s.CorporateCode = dto.CorporateCode
	s.IsRepresentative = dto.IsRepresentative
	s.IsCustomer = dto.IsCustomer
	s.MunicipalRegistration = dto.MunicipalRegistration
	s.SupplierTypeID = typeID
	s.PaymentConditionID = dto.PaymentConditionID
	s.CarrierID = dto.CarrierID
	s.RegionID = dto.RegionID
	s.Homologated = dto.Homologated
	if dto.FreightType != "" {
		s.FreightType = entity.FreightType(dto.FreightType)
	}
	if dto.ViticolaObligation != "" {
		s.ViticolaObligation = entity.ViticolaObligation(dto.ViticolaObligation)
	}
	if dto.ICMSContributor != "" {
		s.ICMSContributor = entity.ICMSContributor(dto.ICMSContributor)
	}
	if dto.TrackingPlatform != "" {
		s.TrackingPlatform = entity.TrackingPlatform(dto.TrackingPlatform)
	}
	s.GLNCode = dto.GLNCode

	return uc.repo.CreateSupplier(ctx, s)
}

func (uc *SupplierUseCase) UpdateSupplier(ctx context.Context, dto request.UpdateSupplierDTO) (*entity.Supplier, error) {
	s, err := uc.repo.GetSupplierByCode(ctx, dto.Code)
	if err != nil {
		return nil, err
	}

	var typeKind entity.SupplierKind
	if dto.SupplierTypeCode != nil {
		st, err := uc.repo.GetSupplierTypeByCode(ctx, *dto.SupplierTypeCode)
		if err != nil {
			return nil, fmt.Errorf("tipo de fornecedor inválido: %w", err)
		}
		s.SupplierTypeID = &st.ID
		typeKind = st.Kind
	}

	// Re-validate the state registration / MEI rules on update.
	if typeKind.RequiresStateRegistration() {
		if dto.StateRegistration == nil || *dto.StateRegistration == "" {
			return nil, fmt.Errorf("inscrição estadual é obrigatória para este tipo de fornecedor")
		}
	}
	if dto.IsMEI && entity.PersonType(dto.PersonType) == entity.PersonFisica {
		return nil, fmt.Errorf("microempreendedor individual não pode ser marcado para pessoa física")
	}
	switch entity.DocumentType(dto.DocumentType) {
	case entity.DocumentCNPJ:
		if !validation.ValidateCNPJ(dto.DocumentNumber) {
			return nil, fmt.Errorf("CNPJ inválido")
		}
	case entity.DocumentCPF:
		if !validation.ValidateCPF(dto.DocumentNumber) {
			return nil, fmt.Errorf("CPF inválido")
		}
	}

	s.CorporateCode = dto.CorporateCode
	s.IsActive = dto.IsActive
	s.IsRepresentative = dto.IsRepresentative
	s.IsCustomer = dto.IsCustomer
	s.Name = dto.Name
	s.TradeName = dto.TradeName
	s.PersonType = entity.PersonType(dto.PersonType)
	s.DocumentType = entity.DocumentType(dto.DocumentType)
	s.DocumentNumber = dto.DocumentNumber
	s.StateRegistration = dto.StateRegistration
	s.MunicipalRegistration = dto.MunicipalRegistration
	s.PaymentConditionID = dto.PaymentConditionID
	s.CarrierID = dto.CarrierID
	s.RegionID = dto.RegionID
	s.GLNCode = dto.GLNCode
	s.AgricultureMinistryRegistration = dto.AgricultureMinistryRegistration
	s.IsMEI = dto.IsMEI
	s.Homologated = dto.Homologated
	if dto.FreightType != "" {
		s.FreightType = entity.FreightType(dto.FreightType)
	}
	if dto.ViticolaObligation != "" {
		s.ViticolaObligation = entity.ViticolaObligation(dto.ViticolaObligation)
	}
	if dto.ICMSContributor != "" {
		s.ICMSContributor = entity.ICMSContributor(dto.ICMSContributor)
	}
	if dto.TrackingPlatform != "" {
		s.TrackingPlatform = entity.TrackingPlatform(dto.TrackingPlatform)
	}

	updated, err := uc.repo.UpdateSupplier(ctx, s)
	if err != nil {
		return nil, err
	}
	// Spec: propagate the IE to other registrations sharing the same document.
	if updated.StateRegistration != nil && *updated.StateRegistration != "" {
		if perr := uc.repo.PropagateStateRegistration(ctx, updated.DocumentNumber, updated.StateRegistration, updated.Code); perr != nil {
			return nil, fmt.Errorf("propagating state registration: %w", perr)
		}
	}
	return updated, nil
}

// DeleteSupplier hard-deletes a supplier. Blocked (friendly error) when purchase
// orders / import processes still reference it; inactivate instead.
func (uc *SupplierUseCase) DeleteSupplier(ctx context.Context, code int64) error {
	return uc.repo.DeleteSupplier(ctx, code)
}

func (uc *SupplierUseCase) GetSupplier(ctx context.Context, code int64) (*entity.Supplier, error) {
	s, err := uc.repo.GetSupplierByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	// Hydrate folders.
	if s.Addresses, err = uc.repo.ListAddresses(ctx, s.ID); err != nil {
		return nil, err
	}
	if s.Phones, err = uc.repo.ListPhones(ctx, s.ID); err != nil {
		return nil, err
	}
	if s.Emails, err = uc.repo.ListEmails(ctx, s.ID); err != nil {
		return nil, err
	}
	if s.DueDates, err = uc.repo.ListDueDates(ctx, s.ID); err != nil {
		return nil, err
	}
	contacts, err := uc.repo.ListContacts(ctx, s.ID)
	if err != nil {
		return nil, err
	}
	for _, c := range contacts {
		if c.Phones, err = uc.repo.ListContactPhones(ctx, c.ID); err != nil {
			return nil, err
		}
		if c.Emails, err = uc.repo.ListContactEmails(ctx, c.ID); err != nil {
			return nil, err
		}
	}
	s.Contacts = contacts
	return s, nil
}

func (uc *SupplierUseCase) ListSuppliers(ctx context.Context, onlyActive bool) ([]*entity.Supplier, error) {
	return uc.repo.ListSuppliers(ctx, onlyActive)
}

func (uc *SupplierUseCase) ListEstablishments(ctx context.Context, corporateCode int64) ([]*entity.Supplier, error) {
	return uc.repo.ListEstablishments(ctx, corporateCode)
}

func (uc *SupplierUseCase) BlockSupplier(ctx context.Context, dto request.BlockSupplierDTO) error {
	return uc.repo.BlockSupplier(ctx, dto.Code, dto.Reason)
}

func (uc *SupplierUseCase) UnblockSupplier(ctx context.Context, code int64) error {
	return uc.repo.UnblockSupplier(ctx, code)
}

// ─── Folders ──────────────────────────────────────────────────────────────────

func (uc *SupplierUseCase) AddAddress(ctx context.Context, dto request.AddSupplierAddressDTO) (*entity.SupplierAddress, error) {
	s, err := uc.repo.GetSupplierByCode(ctx, dto.SupplierCode)
	if err != nil {
		return nil, err
	}
	country := dto.Country
	if country == "" {
		country = "Brasil"
	}
	addrType := entity.AddressType(dto.AddressType)
	if addrType == "" {
		addrType = entity.AddressComercial
	}
	return uc.repo.AddAddress(ctx, &entity.SupplierAddress{
		SupplierID:   s.ID,
		AddressType:  addrType,
		ZipCode:      dto.ZipCode,
		Street:       dto.Street,
		Number:       dto.Number,
		Complement:   dto.Complement,
		Neighborhood: dto.Neighborhood,
		City:         dto.City,
		UF:           dto.UF,
		Country:      country,
		IsDefault:    dto.IsDefault,
	})
}

func (uc *SupplierUseCase) ListAddresses(ctx context.Context, supplierCode int64) ([]*entity.SupplierAddress, error) {
	s, err := uc.repo.GetSupplierByCode(ctx, supplierCode)
	if err != nil {
		return nil, err
	}
	return uc.repo.ListAddresses(ctx, s.ID)
}

func (uc *SupplierUseCase) AddPhone(ctx context.Context, dto request.AddSupplierPhoneDTO) (*entity.SupplierPhone, error) {
	s, err := uc.repo.GetSupplierByCode(ctx, dto.SupplierCode)
	if err != nil {
		return nil, err
	}
	ranking := dto.Ranking
	if ranking == 0 {
		ranking = 1
	}
	return uc.repo.AddPhone(ctx, &entity.SupplierPhone{SupplierID: s.ID, Number: dto.Number, Ranking: ranking})
}

func (uc *SupplierUseCase) AddEmail(ctx context.Context, dto request.AddSupplierEmailDTO) (*entity.SupplierEmail, error) {
	s, err := uc.repo.GetSupplierByCode(ctx, dto.SupplierCode)
	if err != nil {
		return nil, err
	}
	ranking := dto.Ranking
	if ranking == 0 {
		ranking = 1
	}
	return uc.repo.AddEmail(ctx, &entity.SupplierEmail{SupplierID: s.ID, Email: dto.Email, Ranking: ranking})
}

func (uc *SupplierUseCase) AddDueDate(ctx context.Context, dto request.AddSupplierDueDateDTO) (*entity.SupplierDueDate, error) {
	s, err := uc.repo.GetSupplierByCode(ctx, dto.SupplierCode)
	if err != nil {
		return nil, err
	}
	d := &entity.SupplierDueDate{
		SupplierID:         s.ID,
		Description:        dto.Description,
		Ranking:            dto.Ranking,
		BaseDate:           entity.BaseDate(orDefault(dto.BaseDate, string(entity.BaseDateEmissao))),
		PaymentConditionID: dto.PaymentConditionID,
		PaymentType:        entity.DuePaymentType(orDefault(dto.PaymentType, string(entity.DuePaymentNaoInformado))),
		SubsequentMonth:    dto.SubsequentMonth,
		Rounding:           entity.DueRounding(orDefault(dto.Rounding, string(entity.RoundingFixo))),
		ReceiptStartTime:   dto.ReceiptStartTime,
		ReceiptEndTime:     dto.ReceiptEndTime,
		AvgUnloadMinutes:   dto.AvgUnloadMinutes,
	}
	if d.Ranking == 0 {
		d.Ranking = 1
	}
	return uc.repo.AddDueDate(ctx, d)
}

func (uc *SupplierUseCase) AddContact(ctx context.Context, dto request.AddSupplierContactDTO) (*entity.SupplierContact, error) {
	s, err := uc.repo.GetSupplierByCode(ctx, dto.SupplierCode)
	if err != nil {
		return nil, err
	}
	if dto.Name == "" {
		return nil, fmt.Errorf("contact name is required")
	}
	ranking := dto.Ranking
	if ranking == 0 {
		ranking = 1
	}
	return uc.repo.AddContact(ctx, &entity.SupplierContact{
		SupplierID:       s.ID,
		ContactTypeID:    dto.ContactTypeID,
		Name:             dto.Name,
		Position:         dto.Position,
		Department:       dto.Department,
		Ranking:          ranking,
		Observation:      dto.Observation,
		PurchaseOrderTag: dto.PurchaseOrderTag,
	})
}

func (uc *SupplierUseCase) ListContacts(ctx context.Context, supplierCode int64) ([]*entity.SupplierContact, error) {
	s, err := uc.repo.GetSupplierByCode(ctx, supplierCode)
	if err != nil {
		return nil, err
	}
	return uc.repo.ListContacts(ctx, s.ID)
}

func (uc *SupplierUseCase) AddContactPhone(ctx context.Context, dto request.AddSupplierContactPhoneDTO) (*entity.SupplierContactPhone, error) {
	ranking := dto.Ranking
	if ranking == 0 {
		ranking = 1
	}
	return uc.repo.AddContactPhone(ctx, &entity.SupplierContactPhone{ContactID: dto.ContactID, Value: dto.Value, Ranking: ranking})
}

func (uc *SupplierUseCase) AddContactEmail(ctx context.Context, dto request.AddSupplierContactEmailDTO) (*entity.SupplierContactEmail, error) {
	ranking := dto.Ranking
	if ranking == 0 {
		ranking = 1
	}
	return uc.repo.AddContactEmail(ctx, &entity.SupplierContactEmail{ContactID: dto.ContactID, Value: dto.Value, Ranking: ranking})
}

// ─── Enterprise links ──────────────────────────────────────────────────────

func (uc *SupplierUseCase) AddEnterprise(ctx context.Context, dto request.AddSupplierEnterpriseDTO) (*entity.SupplierEnterprise, error) {
	s, err := uc.repo.GetSupplierByCode(ctx, dto.SupplierCode)
	if err != nil {
		return nil, err
	}
	return uc.repo.AddEnterprise(ctx, &entity.SupplierEnterprise{
		SupplierID:           s.ID,
		EnterpriseCode:       dto.EnterpriseCode,
		FinancialAccount:     dto.FinancialAccount,
		AppliesIPI:           dto.AppliesIPI,
		DefaultInvoiceTypeID: dto.DefaultInvoiceTypeID,
		PurchasePriceTableID: dto.PurchasePriceTableID,
		IsActive:             true,
	})
}

func (uc *SupplierUseCase) UpdateEnterprise(ctx context.Context, dto request.UpdateSupplierEnterpriseDTO) (*entity.SupplierEnterprise, error) {
	return uc.repo.UpdateEnterprise(ctx, &entity.SupplierEnterprise{
		ID:                   dto.ID,
		FinancialAccount:     dto.FinancialAccount,
		AppliesIPI:           dto.AppliesIPI,
		DefaultInvoiceTypeID: dto.DefaultInvoiceTypeID,
		PurchasePriceTableID: dto.PurchasePriceTableID,
		IsActive:             dto.IsActive,
	})
}

func (uc *SupplierUseCase) ListEnterprises(ctx context.Context, supplierCode int64) ([]*entity.SupplierEnterprise, error) {
	s, err := uc.repo.GetSupplierByCode(ctx, supplierCode)
	if err != nil {
		return nil, err
	}
	return uc.repo.ListEnterprises(ctx, s.ID)
}

// ─── Parameters ─────────────────────────────────────────────────────────────

func (uc *SupplierUseCase) GetParameters(ctx context.Context, enterpriseCode int64) (*entity.SupplierParameters, error) {
	return uc.repo.GetParameters(ctx, enterpriseCode)
}

func (uc *SupplierUseCase) UpsertParameters(ctx context.Context, dto request.UpsertSupplierParametersDTO) (*entity.SupplierParameters, error) {
	return uc.repo.UpsertParameters(ctx, &entity.SupplierParameters{
		EnterpriseCode:            dto.EnterpriseCode,
		DefaultFinancialAccount:   dto.DefaultFinancialAccount,
		UniqueItemCodePerSupplier: dto.UniqueItemCodePerSupplier,
		RequiresFinancialAccount:  dto.RequiresFinancialAccount,
		PurchaseSupplierTypeID:    dto.PurchaseSupplierTypeID,
		CopyObsToPurchaseOrder:    dto.CopyObsToPurchaseOrder,
		CopyObsToEntryInvoice:     dto.CopyObsToEntryInvoice,
		HomologationDefault:       dto.HomologationDefault,
		UseStockUOM:               dto.UseStockUOM,
		GenericSupplierCode:       dto.GenericSupplierCode,
		DefaultDueBaseDate:        entity.BaseDate(orDefault(dto.DefaultDueBaseDate, string(entity.BaseDateEmissao))),
	})
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
