package response

// CNPJLookupResponse is the API representation of a CNPJ registry lookup, shaped
// to pre-fill cadastro forms (cliente, fornecedor, empresa).
type CNPJLookupResponse struct {
	CNPJ               string `json:"cnpj"`
	LegalName          string `json:"legal_name"`          // razão social
	TradeName          string `json:"trade_name"`          // nome fantasia
	RegistrationStatus string `json:"registration_status"` // situação cadastral
	LegalNature        string `json:"legal_nature"`        // natureza jurídica
	Size               string `json:"size"`                // porte
	OpeningDate        string `json:"opening_date"`        // data de abertura
	Email              string `json:"email"`
	Phone              string `json:"phone"`
	SimplesOptant      bool   `json:"simples_optant"`
	MEI                bool   `json:"mei"`

	// StateRegistration is the IE chosen for the company's own UF (best match);
	// the full list is in StateRegistrations.
	StateRegistration  string                      `json:"state_registration"`
	StateRegistrations []StateRegistrationResponse `json:"state_registrations"`
	Address            CNPJAddressResponse         `json:"address"`
	MainActivity       CNPJActivityResponse        `json:"main_activity"`
	SecondaryActivity  []CNPJActivityResponse      `json:"secondary_activities"`

	// Source names the registry that answered ("cnpja", "brasilapi"). When it is
	// "brasilapi" the IE is not available and must be typed manually.
	Source string `json:"source"`
}

type StateRegistrationResponse struct {
	UF      string `json:"uf"`
	Number  string `json:"number"`
	Enabled bool   `json:"enabled"`
}

type CNPJAddressResponse struct {
	ZipCode      string `json:"zip_code"`
	Street       string `json:"street"`
	Number       string `json:"number"`
	Complement   string `json:"complement"`
	Neighborhood string `json:"neighborhood"`
	City         string `json:"city"`
	UF           string `json:"uf"`
}

type CNPJActivityResponse struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}
