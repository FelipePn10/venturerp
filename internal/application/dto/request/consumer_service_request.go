package request

import "github.com/google/uuid"

type CreateConsumerServiceCallTypeDTO struct {
	Description string    `json:"description"`
	IsComplaint bool      `json:"is_complaint"`
	CreatedBy   uuid.UUID `json:"created_by"`
}

type CreateConsumerServiceKnowledgeSourceDTO struct {
	Description string    `json:"description"`
	CreatedBy   uuid.UUID `json:"created_by"`
}

type CreateConsumerDTO struct {
	Code              int64                      `json:"code"`
	Name              string                     `json:"name"`
	PersonType        string                     `json:"person_type"`
	CPF               *string                    `json:"cpf"`
	RG                *string                    `json:"rg"`
	CNPJ              *string                    `json:"cnpj"`
	StateRegistration *string                    `json:"state_registration"`
	ZipCode           *string                    `json:"zip_code"`
	City              *string                    `json:"city"`
	State             *string                    `json:"state"`
	Address           *string                    `json:"address"`
	AddressNumber     *string                    `json:"address_number"`
	Complement        *string                    `json:"complement"`
	District          *string                    `json:"district"`
	MarketSegmentCode *int64                     `json:"market_segment_code"`
	KnowledgeCode     *int64                     `json:"knowledge_code"`
	Notes             *string                    `json:"notes"`
	CreatedBy         uuid.UUID                  `json:"created_by"`
	Phones            []CreateConsumerPhoneDTO   `json:"phones"`
	Emails            []CreateConsumerEmailDTO   `json:"emails"`
	Contacts          []CreateConsumerContactDTO `json:"contacts"`
}

type UpdateConsumerDTO struct {
	Name              string  `json:"name"`
	IsActive          *bool   `json:"is_active"`
	PersonType        string  `json:"person_type"`
	CPF               *string `json:"cpf"`
	RG                *string `json:"rg"`
	CNPJ              *string `json:"cnpj"`
	StateRegistration *string `json:"state_registration"`
	ZipCode           *string `json:"zip_code"`
	City              *string `json:"city"`
	State             *string `json:"state"`
	Address           *string `json:"address"`
	AddressNumber     *string `json:"address_number"`
	Complement        *string `json:"complement"`
	District          *string `json:"district"`
	MarketSegmentCode *int64  `json:"market_segment_code"`
	KnowledgeCode     *int64  `json:"knowledge_code"`
	Notes             *string `json:"notes"`
}

type CreateConsumerPhoneDTO struct {
	ConsumerCode int64  `json:"consumer_code"`
	ContactCode  *int64 `json:"contact_code"`
	PhoneType    string `json:"phone_type"`
	Number       string `json:"number"`
	IsPrimary    bool   `json:"is_primary"`
}

type CreateConsumerEmailDTO struct {
	ConsumerCode int64  `json:"consumer_code"`
	ContactCode  *int64 `json:"contact_code"`
	Email        string `json:"email"`
	IsPrimary    bool   `json:"is_primary"`
}

type CreateConsumerContactDTO struct {
	ConsumerCode int64   `json:"consumer_code"`
	Name         string  `json:"name"`
	Role         *string `json:"role"`
	ContactType  *string `json:"contact_type"`
	Notes        *string `json:"notes"`
}

type CreateCustomerContactHistoryDTO struct {
	CustomerCode int64     `json:"customer_code"`
	OpenedAt     string    `json:"opened_at"`
	ScheduledAt  string    `json:"scheduled_at"`
	UserCode     *int64    `json:"user_code"`
	ContactType  string    `json:"contact_type"`
	Description  string    `json:"description"`
	CreatedBy    uuid.UUID `json:"created_by"`
}

type CreateConsumerServiceCallDTO struct {
	EnterpriseCode        int64     `json:"enterprise_code"`
	ConsumerCode          int64     `json:"consumer_code"`
	CustomerCode          *int64    `json:"customer_code"`
	CallTypeCode          int64     `json:"call_type_code"`
	Direction             string    `json:"direction"`
	InWarranty            bool      `json:"in_warranty"`
	DefectGroupCode       *int64    `json:"defect_group_code"`
	DefectReasonCode      *int64    `json:"defect_reason_code"`
	ResponsibleUserCode   *int64    `json:"responsible_user_code"`
	Position              string    `json:"position"`
	Situation             string    `json:"situation"`
	OpenedAt              string    `json:"opened_at"`
	ReturnDate            string    `json:"return_date"`
	VisitRequestedDate    string    `json:"visit_requested_date"`
	VisitReturnedDate     string    `json:"visit_returned_date"`
	SaleStoreCode         *int64    `json:"sale_store_code"`
	EstablishmentCode     *int64    `json:"establishment_code"`
	TechnicianDescription *string   `json:"technician_description"`
	Symptoms              *string   `json:"symptoms"`
	ForwardedStoreCode    *int64    `json:"forwarded_store_code"`
	Subject               string    `json:"subject"`
	Description           *string   `json:"description"`
	ChecklistCode         *int64    `json:"checklist_code"`
	CreatedBy             uuid.UUID `json:"created_by"`
}

type UpdateConsumerServiceCallDTO struct {
	CallTypeCode          int64   `json:"call_type_code"`
	Direction             string  `json:"direction"`
	InWarranty            bool    `json:"in_warranty"`
	DefectGroupCode       *int64  `json:"defect_group_code"`
	DefectReasonCode      *int64  `json:"defect_reason_code"`
	ResponsibleUserCode   *int64  `json:"responsible_user_code"`
	Position              string  `json:"position"`
	Situation             string  `json:"situation"`
	ReturnDate            string  `json:"return_date"`
	VisitRequestedDate    string  `json:"visit_requested_date"`
	VisitReturnedDate     string  `json:"visit_returned_date"`
	SaleStoreCode         *int64  `json:"sale_store_code"`
	EstablishmentCode     *int64  `json:"establishment_code"`
	TechnicianDescription *string `json:"technician_description"`
	Symptoms              *string `json:"symptoms"`
	ForwardedStoreCode    *int64  `json:"forwarded_store_code"`
	Subject               string  `json:"subject"`
	Description           *string `json:"description"`
	Solution              *string `json:"solution"`
	ChecklistCode         *int64  `json:"checklist_code"`
}

type AddConsumerServiceCallReturnDTO struct {
	CallCode     int64     `json:"call_code"`
	ContactedAt  string    `json:"contacted_at"`
	ContactType  string    `json:"contact_type"`
	Description  string    `json:"description"`
	NextReturnAt string    `json:"next_return_at"`
	UserCode     *int64    `json:"user_code"`
	CreatedBy    uuid.UUID `json:"created_by"`
}

type AddConsumerServiceCallAttachmentDTO struct {
	CallCode    int64     `json:"call_code"`
	FileName    string    `json:"file_name"`
	FilePath    string    `json:"file_path"`
	ContentType *string   `json:"content_type"`
	Notes       *string   `json:"notes"`
	CreatedBy   uuid.UUID `json:"created_by"`
}

type AddConsumerServiceChecklistItemDTO struct {
	CallCode    int64   `json:"call_code"`
	Sequence    int     `json:"sequence"`
	Description string  `json:"description"`
	Notes       *string `json:"notes"`
}

type SetConsumerServiceChecklistItemDoneDTO struct {
	Done  bool    `json:"done"`
	Notes *string `json:"notes"`
}

type UpsertRecurringSalesParametersDTO struct {
	EnterpriseCode              int64     `json:"enterprise_code"`
	CurrentMonthBillingLimitDay int       `json:"current_month_billing_limit_day"`
	GroupOrderItemTotal         bool      `json:"group_order_item_total"`
	IndefiniteDeliveryDay       int       `json:"indefinite_delivery_day"`
	FixedTermDeliveryDay        int       `json:"fixed_term_delivery_day"`
	ConsiderDiscountsAdditions  bool      `json:"consider_discounts_additions"`
	GenericRepresentativeCode   *int64    `json:"generic_representative_code"`
	GenericSalesPlanCode        *int64    `json:"generic_sales_plan_code"`
	UpdatedBy                   uuid.UUID `json:"updated_by"`
}

type CreateRecurringSalesAdjustmentDateDTO struct {
	EnterpriseCode    int64     `json:"enterprise_code"`
	CustomerCode      int64     `json:"customer_code"`
	EstablishmentCode *int64    `json:"establishment_code"`
	AdjustmentDate    string    `json:"adjustment_date"`
	Notes             *string   `json:"notes"`
	CreatedBy         uuid.UUID `json:"created_by"`
}

type CreateRecurringSaleDTO struct {
	EnterpriseCode     int64                                  `json:"enterprise_code"`
	CustomerCode       int64                                  `json:"customer_code"`
	EstablishmentCode  *int64                                 `json:"establishment_code"`
	ItemCode           int64                                  `json:"item_code"`
	ItemMask           *string                                `json:"item_mask"`
	SalesPlanCode      *int64                                 `json:"sales_plan_code"`
	MovementType       string                                 `json:"movement_type"`
	TermType           string                                 `json:"term_type"`
	SaleDate           string                                 `json:"sale_date"`
	NextAdjustmentDate string                                 `json:"next_adjustment_date"`
	MonthsQuantity     *int                                   `json:"months_quantity"`
	PaymentsQuantity   *int                                   `json:"payments_quantity"`
	GraceMonths        int                                    `json:"grace_months"`
	PaymentValue       *float64                               `json:"payment_value"`
	Quantity           float64                                `json:"quantity"`
	UnitValue          float64                                `json:"unit_value"`
	Reason             *string                                `json:"reason"`
	CreatedBy          uuid.UUID                              `json:"created_by"`
	Representatives    []CreateRecurringSaleRepresentativeDTO `json:"representatives"`
}

type UpdateRecurringSaleDTO struct {
	SalesPlanCode      *int64   `json:"sales_plan_code"`
	SaleDate           string   `json:"sale_date"`
	NextAdjustmentDate string   `json:"next_adjustment_date"`
	MonthsQuantity     *int     `json:"months_quantity"`
	PaymentsQuantity   *int     `json:"payments_quantity"`
	GraceMonths        int      `json:"grace_months"`
	PaymentValue       *float64 `json:"payment_value"`
	Quantity           float64  `json:"quantity"`
	UnitValue          float64  `json:"unit_value"`
	Reason             *string  `json:"reason"`
	IsActive           *bool    `json:"is_active"`
}

type CreateRecurringSaleRepresentativeDTO struct {
	RecurringSaleCode      int64   `json:"recurring_sale_code"`
	RepresentativeCode     int64   `json:"representative_code"`
	IsPrimary              bool    `json:"is_primary"`
	CommissionPercent      float64 `json:"commission_percent"`
	CommissionBase         string  `json:"commission_base"`
	IsLifetime             bool    `json:"is_lifetime"`
	CommissionInstallments *int    `json:"commission_installments"`
}

type MarkRecurringSaleOrderDTO struct {
	OrderCode         int64     `json:"order_code"`
	CreatedBy         uuid.UUID `json:"created_by"`
	EmissionDate      string    `json:"emission_date"`
	Status            string    `json:"status"`
	SalesDivisionCode *int64    `json:"sales_division_code"`
	PaymentTermCode   *int64    `json:"payment_term_code"`
	PriceTableCode    *int64    `json:"price_table_code"`
	WarehouseCode     *int64    `json:"warehouse_code"`
	SalesUOM          *string   `json:"sales_uom"`
	ConfirmOrder      bool      `json:"confirm_order"`
}

type CancelRecurringSaleDTO struct {
	Reason    *string   `json:"reason"`
	CreatedBy uuid.UUID `json:"created_by"`
}

type CalculateRecurringSalesAdjustmentDTO struct {
	EnterpriseCode    *int64    `json:"enterprise_code"`
	CustomerCode      *int64    `json:"customer_code"`
	EstablishmentCode *int64    `json:"establishment_code"`
	ItemCode          *int64    `json:"item_code"`
	AdjustmentDate    string    `json:"adjustment_date"`
	AdjustmentPercent float64   `json:"adjustment_percent"`
	Reason            string    `json:"reason"`
	CreatedBy         uuid.UUID `json:"created_by"`
	Confirm           bool      `json:"confirm"`
}

type RecalculateRecurringSalesAdjustmentDTO struct {
	AdjustmentPercent float64 `json:"adjustment_percent"`
	Reason            string  `json:"reason"`
}
