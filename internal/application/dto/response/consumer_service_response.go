package response

import (
	"encoding/json"
	"time"

	csrepo "github.com/FelipePn10/panossoerp/internal/domain/consumer_service/repository"
	"github.com/google/uuid"
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
	SituationLabel        string                                  `json:"situation_label"`
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
	Code        int64   `json:"id"`
	CallCode    int64   `json:"call_code"`
	FileName    string  `json:"file_name"`
	ContentType *string `json:"content_type,omitempty"`
	FileSize    int64   `json:"file_size"`
	DownloadURL string  `json:"download_url"`
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
	Code                      int64                                 `json:"code"`
	EnterpriseCode            int64                                 `json:"enterprise_code"`
	CustomerCode              int64                                 `json:"customer_code"`
	EstablishmentCode         *int64                                `json:"establishment_code,omitempty"`
	ItemCode                  int64                                 `json:"item_code"`
	ItemMask                  *string                               `json:"item_mask,omitempty"`
	SalesPlanCode             *int64                                `json:"sales_plan_code,omitempty"`
	MovementType              string                                `json:"movement_type"`
	TermType                  string                                `json:"term_type"`
	SaleDate                  time.Time                             `json:"sale_date"`
	NextAdjustmentDate        *time.Time                            `json:"next_adjustment_date,omitempty"`
	MonthsQuantity            *int                                  `json:"months_quantity,omitempty"`
	PaymentsQuantity          *int                                  `json:"payments_quantity,omitempty"`
	GraceMonths               int                                   `json:"grace_months"`
	PaymentValue              *float64                              `json:"payment_value,omitempty"`
	Quantity                  float64                               `json:"quantity"`
	UnitValue                 float64                               `json:"unit_value"`
	MonthlyValue              float64                               `json:"monthly_value"`
	Reason                    *string                               `json:"reason,omitempty"`
	GeneratedOrderCode        *int64                                `json:"generated_order_code,omitempty"`
	GeneratedOrderAt          *time.Time                            `json:"generated_order_at,omitempty"`
	SourceRecurringSaleCode   *int64                                `json:"source_recurring_sale_code,omitempty"`
	OriginalAdjustmentCode    *int64                                `json:"original_adjustment_code,omitempty"`
	AdjustmentPercent         *float64                              `json:"adjustment_percent,omitempty"`
	IsActive                  bool                                  `json:"is_active"`
	LifecycleStatus           string                                `json:"lifecycle_status"`
	EffectiveFrom             *time.Time                            `json:"effective_from,omitempty"`
	EffectiveUntil            *time.Time                            `json:"effective_until,omitempty"`
	CancellationEffectiveDate *time.Time                            `json:"cancellation_effective_date,omitempty"`
	FutureOrdersPolicy        *string                               `json:"future_orders_policy,omitempty"`
	Frequency                 string                                `json:"frequency"`
	PriceTableCode            *int64                                `json:"price_table_code,omitempty"`
	CurrencyCode              string                                `json:"currency_code"`
	AdjustmentIndex           *string                               `json:"adjustment_index,omitempty"`
	AdjustmentPeriodMonths    *int                                  `json:"adjustment_period_months,omitempty"`
	AdjustmentFloorPct        *float64                              `json:"adjustment_floor_pct,omitempty"`
	AdjustmentCapPct          *float64                              `json:"adjustment_cap_pct,omitempty"`
	BillingPolicy             json.RawMessage                       `json:"billing_policy"`
	DeliveryPolicy            json.RawMessage                       `json:"delivery_policy"`
	TaxPolicy                 json.RawMessage                       `json:"tax_policy"`
	CostCenterCode            *int64                                `json:"cost_center_code,omitempty"`
	RenewalPolicy             string                                `json:"renewal_policy"`
	CreatedAt                 time.Time                             `json:"created_at"`
	Representatives           []RecurringSaleRepresentativeResponse `json:"representatives,omitempty"`
	MissingPreconditions      []string                              `json:"missing_preconditions"`
	CanGenerateOrder          bool                                  `json:"can_generate_order"`
	CanCancel                 bool                                  `json:"can_cancel"`
	CanAdjust                 bool                                  `json:"can_adjust"`
	AllowedActions            []string                              `json:"allowed_actions"`
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
	Rows       []RecurringSaleResponse                      `json:"rows"`
	Impacts    []RecurringSalesAdjustmentLineImpactResponse `json:"impacts"`
	TotalRows  int                                          `json:"total_rows"`
	TotalValue float64                                      `json:"total_value"`
	Confirmed  bool                                         `json:"confirmed"`
}

type RecurringSalesAdjustmentLineImpactResponse struct {
	SourceCodes       []int64   `json:"source_codes"`
	ItemCode          int64     `json:"item_code"`
	ItemMask          *string   `json:"item_mask,omitempty"`
	PreviousUnitValue float64   `json:"previous_unit_value"`
	NewUnitValue      float64   `json:"new_unit_value"`
	Quantity          float64   `json:"quantity"`
	PreviousTotal     float64   `json:"previous_total"`
	NewTotal          float64   `json:"new_total"`
	AdjustmentPercent float64   `json:"adjustment_percent"`
	AdjustmentIndex   string    `json:"adjustment_index,omitempty"`
	LegalBasis        string    `json:"legal_basis,omitempty"`
	EffectiveDate     time.Time `json:"effective_date"`
	Reason            string    `json:"reason"`
}

type TechnicalAssistanceRMAResponse struct {
	Code                int64                                    `json:"code"`
	CallCode            int64                                    `json:"call_code"`
	Status              string                                   `json:"status"`
	ReasonCode          string                                   `json:"reason_code"`
	ReasonDescription   *string                                  `json:"reason_description,omitempty"`
	EligibilityStatus   string                                   `json:"eligibility_status"`
	EligibilityReason   string                                   `json:"eligibility_reason"`
	AuthorizationNumber *string                                  `json:"authorization_number,omitempty"`
	AuthorizedAt        *time.Time                               `json:"authorized_at,omitempty"`
	ReverseCarrierCode  *int64                                   `json:"reverse_carrier_code,omitempty"`
	ReverseTrackingCode *string                                  `json:"reverse_tracking_code,omitempty"`
	ReceivedAt          *time.Time                               `json:"received_at,omitempty"`
	InspectionNotes     *string                                  `json:"inspection_notes,omitempty"`
	InspectedAt         *time.Time                               `json:"inspected_at,omitempty"`
	Destination         *string                                  `json:"destination,omitempty"`
	SLADueAt            time.Time                                `json:"sla_due_at"`
	ProductCost         float64                                  `json:"product_cost"`
	FreightCost         float64                                  `json:"freight_cost"`
	ServiceCost         float64                                  `json:"service_cost"`
	TotalCost           float64                                  `json:"total_cost"`
	FiscalDocumentKey   *string                                  `json:"fiscal_document_key,omitempty"`
	StockMovementCode   *int64                                   `json:"stock_movement_code,omitempty"`
	CreatedAt           time.Time                                `json:"created_at"`
	UpdatedAt           time.Time                                `json:"updated_at"`
	AllowedActions      []string                                 `json:"allowed_actions"`
	Items               []TechnicalAssistanceRMAItemResponse     `json:"items"`
	Events              []TechnicalAssistanceRMAEventResponse    `json:"events"`
	Evidences           []TechnicalAssistanceRMAEvidenceResponse `json:"evidences"`
}

type TechnicalAssistanceRMAEvidenceResponse struct {
	ID          uuid.UUID `json:"id"`
	RMACode     int64     `json:"rma_code"`
	FileName    string    `json:"file_name"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	SHA256      string    `json:"sha256"`
	DownloadURL string    `json:"download_url"`
	UploadedBy  uuid.UUID `json:"uploaded_by"`
	CreatedAt   time.Time `json:"created_at"`
}

type TechnicalAssistanceRMAItemResponse struct {
	Code                 int64   `json:"code"`
	CallItemCode         int64   `json:"call_item_code"`
	ItemCode             int64   `json:"item_code"`
	Quantity             float64 `json:"quantity"`
	SerialNumber         *string `json:"serial_number,omitempty"`
	LotNumber            *string `json:"lot_number,omitempty"`
	RequestedDestination *string `json:"requested_destination,omitempty"`
	InspectionResult     *string `json:"inspection_result,omitempty"`
}

type TechnicalAssistanceRMAEventResponse struct {
	Code          int64           `json:"code"`
	EventType     string          `json:"event_type"`
	BeforeState   json.RawMessage `json:"before,omitempty"`
	AfterState    json.RawMessage `json:"after"`
	Reason        *string         `json:"reason,omitempty"`
	CorrelationID *string         `json:"correlation_id,omitempty"`
	OccurredAt    time.Time       `json:"occurred_at"`
	ActorID       uuid.UUID       `json:"actor_id"`
}
