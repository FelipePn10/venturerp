package response

import (
	"time"

	csrepo "github.com/FelipePn10/panossoerp/internal/domain/consumer_service/repository"
)

type ConsumerServiceCallTypeResponse struct {
	Code        int64     `json:"code"`
	Description string    `json:"description"`
	IsComplaint bool      `json:"is_complaint"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

type ConsumerServiceKnowledgeSourceResponse struct {
	Code        int64     `json:"code"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

type ConsumerResponse struct {
	Code              int64                     `json:"code"`
	Name              string                    `json:"name"`
	IsActive          bool                      `json:"is_active"`
	PersonType        string                    `json:"person_type"`
	CPF               *string                   `json:"cpf,omitempty"`
	RG                *string                   `json:"rg,omitempty"`
	CNPJ              *string                   `json:"cnpj,omitempty"`
	StateRegistration *string                   `json:"state_registration,omitempty"`
	ZipCode           *string                   `json:"zip_code,omitempty"`
	City              *string                   `json:"city,omitempty"`
	State             *string                   `json:"state,omitempty"`
	Address           *string                   `json:"address,omitempty"`
	AddressNumber     *string                   `json:"address_number,omitempty"`
	Complement        *string                   `json:"complement,omitempty"`
	District          *string                   `json:"district,omitempty"`
	MarketSegmentCode *int64                    `json:"market_segment_code,omitempty"`
	KnowledgeCode     *int64                    `json:"knowledge_code,omitempty"`
	Notes             *string                   `json:"notes,omitempty"`
	CreatedAt         time.Time                 `json:"created_at"`
	Phones            []ConsumerPhoneResponse   `json:"phones,omitempty"`
	Emails            []ConsumerEmailResponse   `json:"emails,omitempty"`
	Contacts          []ConsumerContactResponse `json:"contacts,omitempty"`
}

type ConsumerPhoneResponse struct {
	Code        int64  `json:"code"`
	PhoneType   string `json:"phone_type"`
	Number      string `json:"number"`
	IsPrimary   bool   `json:"is_primary"`
	ContactCode *int64 `json:"contact_code,omitempty"`
}

type ConsumerEmailResponse struct {
	Code        int64  `json:"code"`
	Email       string `json:"email"`
	IsPrimary   bool   `json:"is_primary"`
	ContactCode *int64 `json:"contact_code,omitempty"`
}

type ConsumerContactResponse struct {
	Code        int64   `json:"code"`
	Name        string  `json:"name"`
	Role        *string `json:"role,omitempty"`
	ContactType *string `json:"contact_type,omitempty"`
	Notes       *string `json:"notes,omitempty"`
}

type CustomerContactHistoryResponse struct {
	Code         int64     `json:"code"`
	CustomerCode int64     `json:"customer_code"`
	OpenedAt     time.Time `json:"opened_at"`
	ScheduledAt  time.Time `json:"scheduled_at"`
	UserCode     *int64    `json:"user_code,omitempty"`
	ContactType  string    `json:"contact_type"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
}

type ConsumerServiceCallResponse struct {
	Code                  int64                                   `json:"code"`
	CallNumber            int64                                   `json:"call_number"`
	EnterpriseCode        int64                                   `json:"enterprise_code"`
	ConsumerCode          int64                                   `json:"consumer_code"`
	CustomerCode          *int64                                  `json:"customer_code,omitempty"`
	CallTypeCode          int64                                   `json:"call_type_code"`
	Direction             string                                  `json:"direction"`
	InWarranty            bool                                    `json:"in_warranty"`
	DefectGroupCode       *int64                                  `json:"defect_group_code,omitempty"`
	DefectReasonCode      *int64                                  `json:"defect_reason_code,omitempty"`
	ResponsibleUserCode   *int64                                  `json:"responsible_user_code,omitempty"`
	Position              string                                  `json:"position"`
	Situation             string                                  `json:"situation"`
	OpenedAt              time.Time                               `json:"opened_at"`
	ReturnDate            *time.Time                              `json:"return_date,omitempty"`
	VisitRequestedDate    *time.Time                              `json:"visit_requested_date,omitempty"`
	VisitReturnedDate     *time.Time                              `json:"visit_returned_date,omitempty"`
	SaleStoreCode         *int64                                  `json:"sale_store_code,omitempty"`
	EstablishmentCode     *int64                                  `json:"establishment_code,omitempty"`
	TechnicianDescription *string                                 `json:"technician_description,omitempty"`
	Symptoms              *string                                 `json:"symptoms,omitempty"`
	ForwardedStoreCode    *int64                                  `json:"forwarded_store_code,omitempty"`
	Subject               string                                  `json:"subject"`
	Description           *string                                 `json:"description,omitempty"`
	Solution              *string                                 `json:"solution,omitempty"`
	ChecklistCode         *int64                                  `json:"checklist_code,omitempty"`
	IsActive              bool                                    `json:"is_active"`
	CreatedAt             time.Time                               `json:"created_at"`
	Returns               []ConsumerServiceCallReturnResponse     `json:"returns,omitempty"`
	Attachments           []ConsumerServiceCallAttachmentResponse `json:"attachments,omitempty"`
	ChecklistItems        []ConsumerServiceChecklistItemResponse  `json:"checklist_items,omitempty"`
}

type ConsumerServiceCallReturnResponse struct {
	Code         int64      `json:"code"`
	CallCode     int64      `json:"call_code"`
	ContactedAt  time.Time  `json:"contacted_at"`
	ContactType  string     `json:"contact_type"`
	Description  string     `json:"description"`
	NextReturnAt *time.Time `json:"next_return_at,omitempty"`
	UserCode     *int64     `json:"user_code,omitempty"`
}

type ConsumerServiceCallAttachmentResponse struct {
	Code        int64   `json:"code"`
	CallCode    int64   `json:"call_code"`
	FileName    string  `json:"file_name"`
	FilePath    string  `json:"file_path"`
	ContentType *string `json:"content_type,omitempty"`
	Notes       *string `json:"notes,omitempty"`
}

type ConsumerServiceChecklistItemResponse struct {
	Code        int64      `json:"code"`
	CallCode    int64      `json:"call_code"`
	Sequence    int        `json:"sequence"`
	Description string     `json:"description"`
	IsDone      bool       `json:"is_done"`
	DoneAt      *time.Time `json:"done_at,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
}

type ConsumerServiceCallReportResponse = csrepo.CallReport

type RecurringSalesParametersResponse struct {
	EnterpriseCode              int64     `json:"enterprise_code"`
	CurrentMonthBillingLimitDay int       `json:"current_month_billing_limit_day"`
	GroupOrderItemTotal         bool      `json:"group_order_item_total"`
	IndefiniteDeliveryDay       int       `json:"indefinite_delivery_day"`
	FixedTermDeliveryDay        int       `json:"fixed_term_delivery_day"`
	ConsiderDiscountsAdditions  bool      `json:"consider_discounts_additions"`
	GenericRepresentativeCode   *int64    `json:"generic_representative_code,omitempty"`
	GenericSalesPlanCode        *int64    `json:"generic_sales_plan_code,omitempty"`
	UpdatedAt                   time.Time `json:"updated_at"`
}

type RecurringSalesAdjustmentDateResponse struct {
	Code              int64     `json:"code"`
	EnterpriseCode    int64     `json:"enterprise_code"`
	CustomerCode      int64     `json:"customer_code"`
	EstablishmentCode *int64    `json:"establishment_code,omitempty"`
	AdjustmentDate    time.Time `json:"adjustment_date"`
	Notes             *string   `json:"notes,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

type RecurringSaleResponse struct {
	Code                    int64                                 `json:"code"`
	EnterpriseCode          int64                                 `json:"enterprise_code"`
	CustomerCode            int64                                 `json:"customer_code"`
	EstablishmentCode       *int64                                `json:"establishment_code,omitempty"`
	ItemCode                int64                                 `json:"item_code"`
	ItemMask                *string                               `json:"item_mask,omitempty"`
	SalesPlanCode           *int64                                `json:"sales_plan_code,omitempty"`
	MovementType            string                                `json:"movement_type"`
	TermType                string                                `json:"term_type"`
	SaleDate                time.Time                             `json:"sale_date"`
	NextAdjustmentDate      *time.Time                            `json:"next_adjustment_date,omitempty"`
	MonthsQuantity          *int                                  `json:"months_quantity,omitempty"`
	PaymentsQuantity        *int                                  `json:"payments_quantity,omitempty"`
	GraceMonths             int                                   `json:"grace_months"`
	PaymentValue            *float64                              `json:"payment_value,omitempty"`
	Quantity                float64                               `json:"quantity"`
	UnitValue               float64                               `json:"unit_value"`
	MonthlyValue            float64                               `json:"monthly_value"`
	Reason                  *string                               `json:"reason,omitempty"`
	GeneratedOrderCode      *int64                                `json:"generated_order_code,omitempty"`
	GeneratedOrderAt        *time.Time                            `json:"generated_order_at,omitempty"`
	SourceRecurringSaleCode *int64                                `json:"source_recurring_sale_code,omitempty"`
	OriginalAdjustmentCode  *int64                                `json:"original_adjustment_code,omitempty"`
	AdjustmentPercent       *float64                              `json:"adjustment_percent,omitempty"`
	IsActive                bool                                  `json:"is_active"`
	CreatedAt               time.Time                             `json:"created_at"`
	Representatives         []RecurringSaleRepresentativeResponse `json:"representatives,omitempty"`
}

type RecurringSaleRepresentativeResponse struct {
	Code                   int64   `json:"code"`
	RepresentativeCode     int64   `json:"representative_code"`
	IsPrimary              bool    `json:"is_primary"`
	CommissionPercent      float64 `json:"commission_percent"`
	CommissionBase         string  `json:"commission_base"`
	IsLifetime             bool    `json:"is_lifetime"`
	CommissionInstallments *int    `json:"commission_installments,omitempty"`
}

type RecurringSalesAdjustmentImpactResponse struct {
	Rows       []RecurringSaleResponse `json:"rows"`
	TotalRows  int                     `json:"total_rows"`
	TotalValue float64                 `json:"total_value"`
	Confirmed  bool                    `json:"confirmed"`
}
