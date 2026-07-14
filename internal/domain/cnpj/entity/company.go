// Package entity holds the domain model for a federal-registry (Receita
// Federal) company lookup by CNPJ. Any registry source maps onto this same
// shape so the rest of the system never learns which API answered.
package entity

// StateRegistration is an Inscrição Estadual tied to a UF. A company may hold
// several (one per state where it operates).
type StateRegistration struct {
	UF      string // federative unit, e.g. "PR"
	Number  string // the IE number
	Enabled bool   // whether the registration is currently active
}

// Address is the company's registered address.
type Address struct {
	ZipCode      string
	Street       string
	Number       string
	Complement   string
	Neighborhood string
	City         string
	UF           string
}

// Activity is a CNAE economic activity (code + description).
type Activity struct {
	Code        string
	Description string
}

// Company is the consolidated result of a CNPJ lookup. Fields a given provider
// cannot supply are simply left zero — callers should treat everything except
// CNPJ as best-effort enrichment to pre-fill cadastro forms.
type Company struct {
	CNPJ               string
	LegalName          string // razão social
	TradeName          string // nome fantasia
	RegistrationStatus string // situação cadastral, e.g. "ATIVA"
	LegalNature        string // natureza jurídica
	Size               string // porte (ME, EPP, DEMAIS)
	OpeningDate        string // data de abertura (ISO yyyy-mm-dd)
	Email              string
	Phone              string
	SimplesOptant      bool // optante pelo Simples Nacional
	MEI                bool // microempreendedor individual

	Address             Address
	MainActivity        Activity
	StateRegistrations  []StateRegistration
	SecondaryActivities []Activity

	// Source identifies which registry answered, useful for diagnostics.
	Source string
}

// PrimaryStateRegistration returns the IE for the company's own UF when known,
// otherwise the first enabled registration, otherwise empty.
func (c *Company) PrimaryStateRegistration() string {
	for _, r := range c.StateRegistrations {
		if r.UF == c.Address.UF && r.Enabled {
			return r.Number
		}
	}
	for _, r := range c.StateRegistrations {
		if r.Enabled {
			return r.Number
		}
	}
	if len(c.StateRegistrations) > 0 {
		return c.StateRegistrations[0].Number
	}
	return ""
}
