package request

type CreateRepresentativeTypeDTO struct {
	Description          string `json:"description"`
	IsFree               bool   `json:"is_free"`
	IgnoresDirectBilling *bool  `json:"ignores_direct_billing,omitempty"`
}

type UpdateRepresentativeTypeDTO struct {
	Code                 int64  `json:"code"`
	Description          string `json:"description"`
	IsFree               bool   `json:"is_free"`
	IgnoresDirectBilling bool   `json:"ignores_direct_billing"`
	IsActive             bool   `json:"is_active"`
}

type CreateRepresentativeDTO struct {
	IsCustomer     bool    `json:"is_customer"`
	CustomerCode   *int64  `json:"customer_code,omitempty"`
	IsSupplier     bool    `json:"is_supplier"`
	SupplierCode   *int64  `json:"supplier_code,omitempty"`
	Name           string  `json:"name"`
	TradeName      *string `json:"trade_name,omitempty"`
	TypeCode       *int64  `json:"type_code,omitempty"`
	CategoryCode   *int64  `json:"category_code,omitempty"`
	RegisterDate   string  `json:"register_date"`
	CoreNumber     *string `json:"core_number,omitempty"`
	DocumentNumber string  `json:"document_number"`
	PostalCode     *string `json:"postal_code,omitempty"`
	City           *string `json:"city,omitempty"`
	State          *string `json:"state,omitempty"`
	FullAddress    *string `json:"full_address,omitempty"`
	Street         *string `json:"street,omitempty"`
	StreetNumber   *string `json:"street_number,omitempty"`
	Complement     *string `json:"complement,omitempty"`
	District       *string `json:"district,omitempty"`
	DeviceQuantity int     `json:"device_quantity"`
}

type UpdateRepresentativeDTO struct {
	Code           int64   `json:"code"`
	IsCustomer     bool    `json:"is_customer"`
	CustomerCode   *int64  `json:"customer_code,omitempty"`
	IsSupplier     bool    `json:"is_supplier"`
	SupplierCode   *int64  `json:"supplier_code,omitempty"`
	Name           string  `json:"name"`
	TradeName      *string `json:"trade_name,omitempty"`
	TypeCode       *int64  `json:"type_code,omitempty"`
	CategoryCode   *int64  `json:"category_code,omitempty"`
	RegisterDate   string  `json:"register_date"`
	CoreNumber     *string `json:"core_number,omitempty"`
	DocumentNumber string  `json:"document_number"`
	PostalCode     *string `json:"postal_code,omitempty"`
	City           *string `json:"city,omitempty"`
	State          *string `json:"state,omitempty"`
	FullAddress    *string `json:"full_address,omitempty"`
	Street         *string `json:"street,omitempty"`
	StreetNumber   *string `json:"street_number,omitempty"`
	Complement     *string `json:"complement,omitempty"`
	District       *string `json:"district,omitempty"`
	DeviceQuantity int     `json:"device_quantity"`
	IsActive       bool    `json:"is_active"`
}

type BlockRepresentativeDTO struct {
	Reason string `json:"reason"`
}

type RepresentativeEnterpriseDTO struct {
	RepresentativeCode    int64   `json:"representative_code"`
	EnterpriseCode        int64   `json:"enterprise_code"`
	EnterpriseName        *string `json:"enterprise_name,omitempty"`
	CommissionPatternCode *int64  `json:"commission_pattern_code,omitempty"`
	CommissionPct         float64 `json:"commission_pct"`
	IsDefault             bool    `json:"is_default"`
	IsActive              bool    `json:"is_active"`
}

type RepresentativeAccountingDTO struct {
	RepresentativeCode   int64  `json:"representative_code"`
	EnterpriseCode       *int64 `json:"enterprise_code,omitempty"`
	EventType            string `json:"event_type"`
	DebitAccountCode     *int64 `json:"debit_account_code,omitempty"`
	DebitCostCenterCode  *int64 `json:"debit_cost_center_code,omitempty"`
	CreditAccountCode    *int64 `json:"credit_account_code,omitempty"`
	CreditCostCenterCode *int64 `json:"credit_cost_center_code,omitempty"`
	HistoryCode          *int64 `json:"history_code,omitempty"`
}

type RepresentativeRegionDTO struct {
	RepresentativeCode int64  `json:"representative_code"`
	EnterpriseCode     *int64 `json:"enterprise_code,omitempty"`
	RegionCode         int64  `json:"region_code"`
	MicroregionCode    *int64 `json:"microregion_code,omitempty"`
	IsActive           bool   `json:"is_active"`
}

type RepresentativeSegmentDTO struct {
	RepresentativeCode int64  `json:"representative_code"`
	EnterpriseCode     *int64 `json:"enterprise_code,omitempty"`
	MicroregionCode    *int64 `json:"microregion_code,omitempty"`
	MarketSegmentCode  int64  `json:"market_segment_code"`
	IsActive           bool   `json:"is_active"`
}

type RepresentativeSalesPlanDTO struct {
	RepresentativeCode int64  `json:"representative_code"`
	EnterpriseCode     *int64 `json:"enterprise_code,omitempty"`
	MicroregionCode    *int64 `json:"microregion_code,omitempty"`
	SalesPlanCode      int64  `json:"sales_plan_code"`
	IsActive           bool   `json:"is_active"`
}

type RepresentativeInterestDTO struct {
	RepresentativeCode     int64 `json:"representative_code"`
	ItemClassificationCode int64 `json:"item_classification_code"`
	IsActive               bool  `json:"is_active"`
}

type RepresentativePhoneDTO struct {
	RepresentativeCode int64   `json:"representative_code"`
	DDI                *string `json:"ddi,omitempty"`
	DDD                *string `json:"ddd,omitempty"`
	Phone              string  `json:"phone"`
	PhoneType          string  `json:"phone_type"`
	Ranking            int     `json:"ranking"`
}

type RepresentativeEmailDTO struct {
	RepresentativeCode int64  `json:"representative_code"`
	Email              string `json:"email"`
	Ranking            int    `json:"ranking"`
}

type RepresentativeCorrespondenceAddressDTO struct {
	RepresentativeCode int64   `json:"representative_code"`
	PostalCode         *string `json:"postal_code,omitempty"`
	City               *string `json:"city,omitempty"`
	State              *string `json:"state,omitempty"`
	FullAddress        *string `json:"full_address,omitempty"`
	Street             *string `json:"street,omitempty"`
	StreetNumber       *string `json:"street_number,omitempty"`
	Complement         *string `json:"complement,omitempty"`
	District           *string `json:"district,omitempty"`
	IsDefault          bool    `json:"is_default"`
}

type RepresentativeContactDTO struct {
	RepresentativeCode int64   `json:"representative_code"`
	ContactTypeCode    *int64  `json:"contact_type_code,omitempty"`
	Name               string  `json:"name"`
	Role               *string `json:"role,omitempty"`
	Phone              *string `json:"phone,omitempty"`
	Email              *string `json:"email,omitempty"`
	Notes              *string `json:"notes,omitempty"`
	IsActive           bool    `json:"is_active"`
}
