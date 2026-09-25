// Package entity é o cadastro de transportadora: o perfil de transporte do
// fornecedor, a frota, as regiões atendidas e as ocorrências de entrega.
//
// A transportadora continua sendo um fornecedor (o frete é pago pelo contas a
// pagar de sempre). O que existe aqui é o que faltava para operar frete:
// habilitação na ANTT, modal, tabela de frete, seguro, veículo/motorista e
// prazo por região.
package entity

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type Modal string

const (
	ModalRodoviario  Modal = "RODOVIARIO"
	ModalAereo       Modal = "AEREO"
	ModalMaritimo    Modal = "MARITIMO"
	ModalFerroviario Modal = "FERROVIARIO"
	ModalDutoviario  Modal = "DUTOVIARIO"
	ModalMultimodal  Modal = "MULTIMODAL"
)

var modais = map[Modal]string{
	ModalRodoviario:  "Rodoviário",
	ModalAereo:       "Aéreo",
	ModalMaritimo:    "Marítimo",
	ModalFerroviario: "Ferroviário",
	ModalDutoviario:  "Dutoviário",
	ModalMultimodal:  "Multimodal",
}

func (m Modal) Valido() bool { _, ok := modais[m]; return ok }
func (m Modal) Rotulo() string {
	if r, ok := modais[m]; ok {
		return r
	}
	return string(m)
}

// TipoTransportador são as categorias da ANTT: ETC (empresa), CTC (cooperativa)
// e TAC (autônomo). O tipo muda a obrigação fiscal do CT-e.
type TipoTransportador string

const (
	TipoETC TipoTransportador = "ETC"
	TipoCTC TipoTransportador = "CTC"
	TipoTAC TipoTransportador = "TAC"
)

var tipos = map[TipoTransportador]string{
	TipoETC: "ETC — Empresa de Transporte de Carga",
	TipoCTC: "CTC — Cooperativa de Transporte de Carga",
	TipoTAC: "TAC — Transportador Autônomo",
}

func (t TipoTransportador) Valido() bool { _, ok := tipos[t]; return ok }
func (t TipoTransportador) Rotulo() string {
	if r, ok := tipos[t]; ok {
		return r
	}
	return string(t)
}

var (
	somenteDigitos   = regexp.MustCompile(`^[0-9]+$`)
	placaAntiga      = regexp.MustCompile(`^[A-Z]{3}[0-9]{4}$`)
	placaMercosul    = regexp.MustCompile(`^[A-Z]{3}[0-9][A-Z][0-9]{2}$`)
	cepFormatoValido = regexp.MustCompile(`^[0-9]{8}$`)
)

type Transportadora struct {
	ID                 int64
	EnterpriseID       int64
	SupplierCode       int64
	SupplierName       string // leitura, vem da junção com suppliers
	SupplierDocument   string // leitura
	ANTTRNTRC          *string
	ANTTExpiry         *time.Time
	ShipperType        *TipoTransportador
	Modal              Modal
	IssuesCTe          bool
	DefaultFreightType *string
	FreightMinValue    decimal.Decimal
	FreightKgRate      decimal.Decimal
	FreightPctValue    decimal.Decimal
	GrisPct            decimal.Decimal
	TollPer100Kg       decimal.Decimal
	InsuranceCompany   *string
	InsurancePolicy    *string
	InsuranceExpiry    *time.Time
	InsuranceCoverage  decimal.Decimal
	AverageLeadDays    *int16
	TrackingURL        *string
	ContactName        *string
	ContactPhone       *string
	ContactEmail       *string
	Notes              *string
	IsActive           bool
	CreatedAt          time.Time
	UpdatedAt          time.Time

	Vehicles     []*Veiculo
	ServiceAreas []*RegiaoAtendida
}

func (t *Transportadora) Normalizar() {
	if strings.TrimSpace(string(t.Modal)) == "" {
		t.Modal = ModalRodoviario
	}
	t.Modal = Modal(strings.ToUpper(strings.TrimSpace(string(t.Modal))))
	if t.ShipperType != nil {
		v := TipoTransportador(strings.ToUpper(strings.TrimSpace(string(*t.ShipperType))))
		if v == "" {
			t.ShipperType = nil
		} else {
			t.ShipperType = &v
		}
	}
	if t.ANTTRNTRC != nil {
		// O usuário digita "123.456-78"; o que vale é o número.
		limpo := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, *t.ANTTRNTRC)
		if limpo == "" {
			t.ANTTRNTRC = nil
		} else {
			t.ANTTRNTRC = &limpo
		}
	}
	if t.DefaultFreightType != nil {
		v := strings.ToUpper(strings.TrimSpace(*t.DefaultFreightType))
		if v == "" {
			t.DefaultFreightType = nil
		} else {
			t.DefaultFreightType = &v
		}
	}
	t.ContactEmail = limparTexto(t.ContactEmail)
	t.ContactName = limparTexto(t.ContactName)
	t.ContactPhone = limparTexto(t.ContactPhone)
	t.InsuranceCompany = limparTexto(t.InsuranceCompany)
	t.InsurancePolicy = limparTexto(t.InsurancePolicy)
	t.TrackingURL = limparTexto(t.TrackingURL)
	t.Notes = limparTexto(t.Notes)
}

func limparTexto(v *string) *string {
	if v == nil {
		return nil
	}
	limpo := strings.TrimSpace(*v)
	if limpo == "" {
		return nil
	}
	return &limpo
}

func (t *Transportadora) Validar() error {
	if t.SupplierCode <= 0 {
		return errors.New("informe o fornecedor da transportadora")
	}
	if !t.Modal.Valido() {
		return fmt.Errorf("modal de transporte inválido: %s", t.Modal)
	}
	if t.ShipperType != nil && !t.ShipperType.Valido() {
		return fmt.Errorf("tipo de transportador inválido: %s", *t.ShipperType)
	}
	// RNTRC tem 8 dígitos. Com número errado o CT-e é rejeitado na SEFAZ, e o
	// erro aparece na expedição, não no cadastro.
	if t.ANTTRNTRC != nil {
		if !somenteDigitos.MatchString(*t.ANTTRNTRC) || len(*t.ANTTRNTRC) != 8 {
			return errors.New("o RNTRC deve ter 8 dígitos")
		}
	}
	// Modal rodoviário emitindo CT-e sem RNTRC não passa na SEFAZ.
	if t.Modal == ModalRodoviario && t.IssuesCTe && t.ANTTRNTRC == nil {
		return errors.New("transportadora rodoviária que emite CT-e precisa do RNTRC")
	}
	if t.DefaultFreightType != nil {
		switch *t.DefaultFreightType {
		case "CIF", "FOB", "TERCEIROS", "SEM_FRETE":
		default:
			return fmt.Errorf("tipo de frete padrão inválido: %s", *t.DefaultFreightType)
		}
	}
	for _, d := range []struct {
		nome  string
		valor decimal.Decimal
	}{
		{"valor mínimo de frete", t.FreightMinValue},
		{"valor por quilo", t.FreightKgRate},
		{"pedágio por 100 kg", t.TollPer100Kg},
		{"cobertura do seguro", t.InsuranceCoverage},
	} {
		if d.valor.IsNegative() {
			return fmt.Errorf("o %s não pode ser negativo", d.nome)
		}
	}
	cem := decimal.NewFromInt(100)
	if t.FreightPctValue.IsNegative() || t.FreightPctValue.GreaterThan(cem) {
		return errors.New("o percentual de frete sobre o valor deve estar entre 0 e 100")
	}
	if t.GrisPct.IsNegative() || t.GrisPct.GreaterThan(cem) {
		return errors.New("o percentual de GRIS deve estar entre 0 e 100")
	}
	if t.AverageLeadDays != nil && *t.AverageLeadDays < 0 {
		return errors.New("o prazo médio não pode ser negativo")
	}
	if t.InsurancePolicy != nil && t.InsuranceCompany == nil {
		return errors.New("informe a seguradora da apólice")
	}
	for _, v := range t.Vehicles {
		if err := v.Validar(); err != nil {
			return err
		}
	}
	for _, a := range t.ServiceAreas {
		if err := a.Validar(); err != nil {
			return err
		}
	}
	return nil
}

// Alertas são as pendências que não impedem gravar mas impedem transportar:
// habilitação ou seguro vencido, frota sem veículo ativo, tabela zerada. É o
// que a tela mostra em amarelo em vez de descobrir na hora de despachar.
func (t *Transportadora) Alertas(hoje time.Time) []string {
	var out []string
	hoje = hoje.Truncate(24 * time.Hour)
	if t.ANTTRNTRC == nil && t.Modal == ModalRodoviario {
		out = append(out, "transportadora rodoviária sem RNTRC informado")
	}
	if t.ANTTExpiry != nil && t.ANTTExpiry.Before(hoje) {
		out = append(out, fmt.Sprintf("RNTRC vencido em %s", t.ANTTExpiry.Format("02/01/2006")))
	}
	if t.InsuranceExpiry != nil && t.InsuranceExpiry.Before(hoje) {
		out = append(out, fmt.Sprintf("seguro vencido em %s", t.InsuranceExpiry.Format("02/01/2006")))
	}
	if t.InsuranceCompany == nil {
		out = append(out, "sem seguro de carga cadastrado")
	}
	if t.FreightMinValue.IsZero() && t.FreightKgRate.IsZero() && t.FreightPctValue.IsZero() {
		out = append(out, "tabela de frete não preenchida: o frete terá de ser digitado a cada pedido")
	}
	if t.Modal == ModalRodoviario {
		ativos := 0
		for _, v := range t.Vehicles {
			if v.IsActive {
				ativos++
			}
		}
		if ativos == 0 {
			out = append(out, "nenhum veículo ativo cadastrado")
		}
	}
	if len(t.ServiceAreas) == 0 {
		out = append(out, "nenhuma região atendida cadastrada: o prazo de entrega não pode ser calculado")
	}
	return out
}

type Veiculo struct {
	ID             int64
	EnterpriseID   int64
	CarrierID      int64
	Plate          string
	Description    *string
	VehicleType    *string
	Axles          *int16
	CapacityKg     decimal.Decimal
	CapacityM3     decimal.Decimal
	ANTTOwner      *string
	DriverName     *string
	DriverDocument *string
	DriverLicense  *string
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (v *Veiculo) Normalizar() {
	v.Plate = strings.ToUpper(strings.NewReplacer("-", "", " ", "", ".", "").Replace(strings.TrimSpace(v.Plate)))
	v.Description = limparTexto(v.Description)
	v.VehicleType = limparTexto(v.VehicleType)
	v.ANTTOwner = limparTexto(v.ANTTOwner)
	v.DriverName = limparTexto(v.DriverName)
	v.DriverDocument = limparTexto(v.DriverDocument)
	v.DriverLicense = limparTexto(v.DriverLicense)
}

func (v *Veiculo) Validar() error {
	if v.Plate == "" {
		return errors.New("informe a placa do veículo")
	}
	// Aceita o padrão antigo (AAA9999) e o Mercosul (AAA9A99): a frota real tem
	// os dois, e recusar um dos formatos é travar o cadastro sem motivo.
	if !placaAntiga.MatchString(v.Plate) && !placaMercosul.MatchString(v.Plate) {
		return fmt.Errorf("placa inválida: %s (use AAA9999 ou AAA9A99)", v.Plate)
	}
	if v.CapacityKg.IsNegative() || v.CapacityM3.IsNegative() {
		return errors.New("a capacidade do veículo não pode ser negativa")
	}
	if v.Axles != nil && (*v.Axles < 2 || *v.Axles > 12) {
		return errors.New("o número de eixos deve estar entre 2 e 12")
	}
	return nil
}

// Cabe responde se a carga caberia neste veículo. É a pergunta da expedição.
func (v *Veiculo) Cabe(pesoKg, volumeM3 decimal.Decimal) bool {
	if v.CapacityKg.IsPositive() && pesoKg.GreaterThan(v.CapacityKg) {
		return false
	}
	if v.CapacityM3.IsPositive() && volumeM3.GreaterThan(v.CapacityM3) {
		return false
	}
	return true
}

type RegiaoAtendida struct {
	ID             int64
	EnterpriseID   int64
	CarrierID      int64
	State          *string
	City           *string
	PostalCodeFrom *string
	PostalCodeTo   *string
	LeadDays       int16
	MinValue       decimal.Decimal
	KgRate         decimal.Decimal
	PctValue       decimal.Decimal
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (a *RegiaoAtendida) Normalizar() {
	if a.State != nil {
		uf := strings.ToUpper(strings.TrimSpace(*a.State))
		if uf == "" {
			a.State = nil
		} else {
			a.State = &uf
		}
	}
	a.City = limparTexto(a.City)
	a.PostalCodeFrom = limparCEP(a.PostalCodeFrom)
	a.PostalCodeTo = limparCEP(a.PostalCodeTo)
}

func limparCEP(v *string) *string {
	if v == nil {
		return nil
	}
	limpo := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, *v)
	if limpo == "" {
		return nil
	}
	return &limpo
}

func (a *RegiaoAtendida) Validar() error {
	if a.State == nil && a.PostalCodeFrom == nil {
		return errors.New("informe a UF ou a faixa de CEP da região atendida")
	}
	if a.State != nil && len(*a.State) != 2 {
		return fmt.Errorf("UF inválida: %s", *a.State)
	}
	for _, cep := range []*string{a.PostalCodeFrom, a.PostalCodeTo} {
		if cep != nil && !cepFormatoValido.MatchString(*cep) {
			return fmt.Errorf("CEP inválido: %s (use 8 dígitos)", *cep)
		}
	}
	if a.PostalCodeFrom != nil && a.PostalCodeTo != nil && *a.PostalCodeFrom > *a.PostalCodeTo {
		return errors.New("a faixa de CEP começa depois de terminar")
	}
	if a.LeadDays < 0 {
		return errors.New("o prazo da região não pode ser negativo")
	}
	if a.MinValue.IsNegative() || a.KgRate.IsNegative() {
		return errors.New("os valores da região não podem ser negativos")
	}
	if a.PctValue.IsNegative() || a.PctValue.GreaterThan(decimal.NewFromInt(100)) {
		return errors.New("o percentual da região deve estar entre 0 e 100")
	}
	return nil
}

// Atende diz se o destino cai nesta região. UF sozinha atende o estado todo;
// com faixa de CEP, a faixa é quem manda.
func (a *RegiaoAtendida) Atende(uf, cep string) bool {
	if !a.IsActive {
		return false
	}
	cep = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, cep)
	if a.PostalCodeFrom != nil && len(cep) == 8 {
		fim := *a.PostalCodeFrom
		if a.PostalCodeTo != nil {
			fim = *a.PostalCodeTo
		}
		return cep >= *a.PostalCodeFrom && cep <= fim
	}
	if a.State != nil {
		return strings.EqualFold(*a.State, uf)
	}
	return false
}
